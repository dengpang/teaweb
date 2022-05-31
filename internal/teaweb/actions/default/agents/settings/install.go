package settings

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/maps"
)

type InstallAction actions.Action

// 安装部署
func (this *InstallAction) Run(params struct {
	AgentId string
}) {
	this.Data["selectedTab"] = "install"

	agent := agents.NewAgentConfigFromId(params.AgentId)
	if agent == nil {
		this.Fail("找不到Agent")
	}

	//this.Data["agent"] = agent
	this.Data["agent"] = maps.Map{
		"id":  agent.Id,
		"key": agent.Key,
	}
	this.Data["isLocal"] = agent.IsLocal()

	this.Show()
}
