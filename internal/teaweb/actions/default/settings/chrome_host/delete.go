package chrome_host

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaconfigs/chrome_host"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/iwind/TeaGo/maps"

	"github.com/iwind/TeaGo/actions"
)

type DeleteAction actions.Action

func (this *DeleteAction) Run(params struct {
	Id int
}) {
	//if params.Id == 0 {
	//	this.FailField("id", "参数错误")
	//	return
	//}
	files := chrome_host.SharedChromeHostConfig()
	if files == nil {
		this.FailField("id", "操作失败")
		return
	}
	//if concent.Default {
	//	this.FailField("keyword","内置敏感词暂不支持修改")
	//}
	this.Data["id"] = params.Id
	newList := make([]chrome_host.List, 0)
	for _, v := range files.List {
		if v.Id == params.Id {
			continue
		}
		newList = append(newList, v)
	}
	files.List = newList
	files.Save()

	{
		for _, agent := range agents.AllSharedAgents() {
			//重新保存浏览器
			agent.UpdateChrome()
			agent.Save()
			// 通知更新
			agentutils.PostAgentEvent(agent.Id, agentutils.NewAgentEvent("UPDATE_AGENT", maps.Map{}))
		}
	}
	this.Success()
}
