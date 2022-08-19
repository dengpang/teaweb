package apps

import (
	"encoding/json"
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teadb"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/actionutils"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/utils/time"
	"time"
)

type MonitorAction struct {
	actionutils.ParentAction
}

// 监控
func (this *MonitorAction) Run(params struct {
	AgentId string
	AppId   string
}) {
	app := agentutils.InitAppData(this, params.AgentId, params.AppId, "monitor")
	err := app.Validate()
	if err != nil {
		logs.Error(err)
	}

	m := this.Data["app"].(maps.Map)

	this.Data["noticeLevels"] = notices.AllNoticeLevels()

	page := this.NewPage(int64(len(app.Items)))
	end := page.Offset + page.Size
	if page.Offset > int64(len(app.Items)) {
		page.Offset = 0
	}
	if end > int64(len(app.Items)) {
		end = int64(len(app.Items))
	}
	this.Data["page"] = page.AsHTML()

	m["items"] = app.Items[page.Offset:end]

	this.Show()
}

// 监控数据
func (this *MonitorAction) RunPost(params struct {
	AgentId string
	AppId   string
}) {
	agent := agents.NewAgentConfigFromId(params.AgentId)
	if agent == nil {
		this.Fail("找不到Agent")
	}

	app := agent.FindApp(params.AppId)
	if app == nil {
		this.Fail("找不到App")
	}
	page := this.NewPage(int64(len(app.Items)))
	end := page.Offset + page.Size
	if page.Offset > int64(len(app.Items)) {
		page.Offset = 0
	}
	if end > int64(len(app.Items)) {
		end = int64(len(app.Items))
	}
	this.Data["page"] = page.AsHTML()
	this.Data["items"] = lists.Map(app.Items[page.Offset:end], func(k int, v interface{}) interface{} {
		item := v.(*agents.Item)

		latestValue := ""
		latestTime := ""
		latestLevel := notices.NoticeLevelNone

		value, err := teadb.AgentValueDAO().FindLatestItemValue(params.AgentId, params.AppId, item.Id)
		costMs := float64(0)
		if err != nil {
			if err != teadb.ErrorDBUnavailable {
				logs.Error(err)
			}
		} else if value != nil {
			costMs = value.CostMs
			data, err := json.MarshalIndent(value.Value, "", "  ")
			if err != nil {
				logs.Error(err)
			} else {
				latestValue = string(data)
				latestTime = timeutil.Format("Y-m-d H:i:s", time.Unix(value.Timestamp, 0))
				latestLevel = value.NoticeLevel
			}
		}

		err = item.Validate()
		if err != nil {
			logs.Error(err)
		}

		result := maps.Map{
			"id":          item.Id,
			"name":        item.Name,
			"on":          item.On,
			"interval":    item.Interval,
			"thresholds":  item.Thresholds,
			"latestValue": latestValue,
			"latestTime":  latestTime,
			"isWarning":   latestLevel == notices.NoticeLevelWarning,
			"isError":     latestLevel == notices.NoticeLevelError,
			"costMs":      costMs,
		}

		source := item.Source()
		if source != nil {
			result["sourceName"] = source.Name()
		} else {
			result["sourceName"] = "Agent"
		}

		return result
	})

	this.Success()
}
