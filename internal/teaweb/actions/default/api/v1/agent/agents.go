package agent

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/api/apiutils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/maps"
)

type AgentsAction actions.Action

// Agent列表
func (this *AgentsAction) RunGet(params struct{}) {
	result := []maps.Map{}
	for _, agent := range agents.SharedAgents() {
		result = append(result, maps.Map{
			"config": agent,
		})
		//if _, e := ffjson.Marshal(agent); e != nil {
		//	fmt.Println(e)
		//	fmt.Println(agent)
		//	//fmt.Println(agent.Apps[0], agent.Apps[1])
		//	for _, v := range agent.Apps[1].Items {
		//		if _, ee := ffjson.Marshal(v.Charts); ee != nil {
		//			for _, vv := range v.Charts {
		//				if _, eee := ffjson.Marshal(vv.Options); eee != nil {
		//
		//					for kkkk, vvv := range vv.Options {
		//						if _, eeee := ffjson.Marshal(vvv); eeee != nil {
		//							fmt.Println("eeee", kkkk, vvv)
		//							fmt.Println(eeee)
		//						}
		//					}
		//				}
		//			}
		//
		//		}
		//	}
		//}
	}
	//apiutils.Success(this, "ok")
	apiutils.Success(this, result)
}
