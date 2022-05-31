package apps

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teadb"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/actionutils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
)

type IndexAction struct {
	actionutils.ParentAction
}

// 看板首页
func (this *IndexAction) Run(params struct {
	AgentId string
}) {
	this.Data["agentId"] = params.AgentId

	agent := agents.NewAgentConfigFromId(params.AgentId)
	if agent == nil {
		this.Fail("找不到要修改的Agent")
	}

	page := this.NewPage(int64(len(agent.Apps)))
	end := page.Offset + page.Size
	if page.Offset > int64(len(agent.Apps)) {
		page.Offset = 0
	}
	if end > int64(len(agent.Apps)) {
		end = int64(len(agent.Apps))
	}
	this.Data["page"] = page.AsHTML()
	// 用户自定义App
	this.Data["apps"] = lists.Map(agent.Apps[page.Offset:end], func(k int, v interface{}) interface{} {
		app := v.(*agents.AppConfig)

		// 最新一条数据
		level := notices.NoticeLevelNone
		for _, item := range app.Items {
			if !item.On {
				continue
			}
			value, err := teadb.AgentValueDAO().FindLatestItemValue(agent.Id, app.Id, item.Id)
			if err == nil && value != nil {
				if value.NoticeLevel > level && (value.NoticeLevel == notices.NoticeLevelWarning || value.NoticeLevel == notices.NoticeLevelError) {
					level = value.NoticeLevel
				}
			}
		}

		return maps.Map{
			"on":   app.On,
			"id":   app.Id,
			"name": app.Name,
			//"items":  app.Items,
			//"bootingTasks":      app.FindBootingTasks(),
			//"manualTasks":       app.FindManualTasks(),
			//"schedulingTasks":   app.FindSchedulingTasks(),
			"num":                len(app.Items),
			"bootingTasksNum":    len(app.FindBootingTasks()),
			"manualTasksNum":     len(app.FindManualTasks()),
			"schedulingTasksNum": len(app.FindSchedulingTasks()),
			"isSharedWithGroup":  app.IsSharedWithGroup,
			"isWarning":          level == notices.NoticeLevelWarning,
			"isError":            level == notices.NoticeLevelError,
		}
	})

	this.Show()
}
