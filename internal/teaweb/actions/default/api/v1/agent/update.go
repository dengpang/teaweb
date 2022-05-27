package agent

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
)

type UpdateAction actions.Action

func (this *UpdateAction) RunPost(params struct {
	AgentId string
	Name    string
	Host    string
	On      bool
	Must    *actions.Must
}) {
	params.Must.
		Field("name", params.Name).
		Require("请输入主机名").
		Field("host", params.Host).
		Require("请输入主机地址")

	agent := agents.NewAgentConfigFromId(params.AgentId)
	if agent == nil {
		this.Fail("找不到要修改的Agent")
	}
	agent.On = params.On
	agent.Name = params.Name
	agent.Host = params.Host

	err := agent.Save()
	if err != nil {
		this.Fail("保存失败：" + err.Error())
	}

	// 重建索引
	err = agents.SharedGroupList().BuildIndexes()
	if err != nil {
		logs.Error(err)
	}

	// 通知更新
	agentutils.PostAgentEvent(agent.Id, agentutils.NewAgentEvent("UPDATE_AGENT", maps.Map{}))

	this.Success()
}
