package agent

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/rands"
)

type AddAction actions.Action

func (this *AddAction) RunPost(params struct {
	Name string
	Host string
	On   bool
	Must *actions.Must
}) {
	params.Must.
		Field("name", params.Name).
		Require("请输入主机名").
		Field("host", params.Host).
		Require("请输入主机地址")

	agentList, err := agents.SharedAgentList()
	if err != nil {
		this.Fail("保存失败：" + err.Error())
	}

	agent := agents.NewAgentConfig()
	agent.On = params.On
	agent.Name = params.Name
	agent.Host = params.Host
	agent.AddGroup("default")
	agent.AllowAll = true
	agent.Key = rands.HexString(32)
	agent.CheckDisconnections = true
	agent.AutoUpdates = true
	agent.Simple = true //只添加 cpu 内存  磁盘监控
	agent.AddDefaultApps()

	err = agent.Save()
	if err != nil {
		this.Fail("保存失败：" + err.Error())
	}

	agentList.AddAgent(agent.Filename())
	err = agentList.Save()
	if err != nil {
		this.Fail("保存失败：" + err.Error())
	}

	// 重建索引
	err = agents.SharedGroupList().BuildIndexes()
	if err != nil {
		logs.Error(err)
	}

	this.Data["agentId"] = agent.Id
	// 通知更新
	agentutils.PostAgentEvent(agent.Id, agentutils.NewAgentEvent("ADD_AGENT", maps.Map{}))

	this.Success()
}
