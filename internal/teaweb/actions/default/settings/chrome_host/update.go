package chrome_host

import (
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaconfigs/chrome_host"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/maps"
	"net"
	"strconv"
	"time"
)

type UpdateAction actions.Action

func (this *UpdateAction) Run(params struct {
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
	this.Data["addr"] = ""
	this.Data["port"] = 9222
	this.Data["cpu_num"] = 4
	for _, v := range files.List {
		if v.Id == params.Id {
			this.Data["id"] = v.Id
			this.Data["addr"] = v.Addr
			this.Data["port"] = v.Port
			this.Data["cpu_num"] = v.CpuNum
			break
		}
	}

	this.Show()
}

func (this *UpdateAction) RunPost(params struct {
	Id     int
	Addr   string
	Port   int
	CpuNum int
	Must   *actions.Must
}) {
	params.Must.
		Field("Id", params.Id).
		Require("请选择主机")

	addr := net.JoinHostPort(params.Addr, strconv.Itoa(params.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		this.FailField("port", "端口连接超时")
		return
	}
	defer conn.Close()

	concent := chrome_host.SharedChromeHostConfig()
	if concent == nil {
		this.FailField("id", "修改失败")
		return
	}
	if len(concent.List) > 0 {
		isUse := false
		for k, v := range concent.List {
			if v.Id == params.Id {
				x := chrome_host.List{
					Id: params.Id, Addr: params.Addr, Port: params.Port, CpuNum: params.CpuNum,
				}
				concent.List[k] = x
				isUse = true
				break
			}
		}
		if !isUse {
			x := chrome_host.List{
				Id: len(concent.List) + 1, Addr: params.Addr, Port: params.Port, CpuNum: params.CpuNum,
			}
			concent.List = append(concent.List, x)
		}
	} else {
		x := chrome_host.List{
			Id: len(concent.List) + 1, Addr: params.Addr, Port: params.Port, CpuNum: params.CpuNum,
		}
		concent.List = append(concent.List, x)

	}

	err = concent.Save()
	if err != nil {
		this.FailField("id", "修改失败")
		return
	}
	{
		for _, agent := range agents.AllSharedAgents() {
			//重新保存浏览器
			agent.UpdateChrome()
			agent.Save()
			// 通知更新
			agentutils.PostAgentEvent(agent.Id, agentutils.NewAgentEvent("UPDATE_AGENT", maps.Map{}))
		}
	}
	this.Next("/settings/chrome_host", nil).Success("保存成功")
}
