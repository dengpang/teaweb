package agent

import (
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/api/apiutils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/maps"
)

type AgentInfoAction actions.Action

// Agent列表
func (this *AgentInfoAction) RunGet(params struct {
	AgentId string
}) {
	state := agentutils.FindAgentState(params.AgentId)
	apiutils.Success(this, maps.Map{
		"version":   state.Version,
		"os_name":   state.OsName,
		"speed":     state.Speed,
		"ip":        state.IP,
		"is_active": state.IsActive,
	})
}
