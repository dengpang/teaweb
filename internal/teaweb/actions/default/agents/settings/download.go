package settings

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/iwind/TeaGo/actions"
)

type DownloadAction actions.Action

// 设置首页
func (this *DownloadAction) Run(params struct {
	AgentId string
}) {
	this.Data["selectedTab"] = "download"

	agent := agents.NewAgentConfigFromId(params.AgentId)
	if agent == nil {
		this.Fail("找不到Agent")
	}

	this.Data["agent"] = agent
	this.Data["isLocal"] = agent.IsLocal()
	this.Show()
}
