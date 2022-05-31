package app

import (
	"encoding/json"
	"fmt"
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaconst"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"regexp"
)

type AddItemAction actions.Action

// 提交保存
func (this *AddItemAction) RunPost(params struct {
	AgentId    string
	AppId      string
	Name       string
	SourceCode string
	On         bool

	DataFormat uint8
	Interval   uint

	CondParams         []string
	CondOps            []string
	CondValues         []string
	CondNoticeLevels   []uint
	CondNoticeMessages []string
	CondActions        []string
	CondMaxFails       []int

	RecoverSuccesses int

	KeywordCheckDiy string
	Must            *actions.Must
}) {
	if teaconst.DemoEnabled {
		this.Fail("演示版无法添加监控项")
	}

	agent := agents.NewAgentConfigFromId(params.AgentId)
	if agent == nil {
		this.Fail("找不到Agent")
	}

	app := agent.FindApp(params.AppId)
	if app == nil {
		this.Fail("找不到App")
	}

	params.Must.
		Field("name", params.Name).
		Require("请输入监控项名称").
		Field("sourceCode", params.SourceCode).
		Require("请选择数据源类型")

	item := agents.NewItem()
	item.On = params.On
	item.Name = params.Name

	// 数据源
	item.SourceCode = params.SourceCode
	item.SourceOptions = map[string]interface{}{}

	// 获取参数值
	instance := agents.FindDataSourceInstance(params.SourceCode, map[string]interface{}{})
	form := instance.Form()
	values, errField, err := form.ApplyRequest(this.Request)
	if err != nil {
		this.FailField(errField, err.Error())
	}
	//接口方式添加监控 敏感词检测自定义关键词需要额外追加
	if item.SourceCode == "keywordCheck" {
		values["diyInputKeyword"] = params.KeywordCheckDiy
	}
	values["dataFormat"] = params.DataFormat
	item.SourceOptions = values

	// 测试
	err = item.Validate()
	if err != nil {
		this.Fail("校验失败：" + err.Error())
	}

	// 刷新间隔等其他选项
	item.Interval = fmt.Sprintf("%ds", params.Interval)
	item.RecoverSuccesses = params.RecoverSuccesses
	//fmt.Println("1")
	// 阈值设置
	for index, param := range params.CondParams {
		if index < len(params.CondValues) &&
			index < len(params.CondOps) &&
			index < len(params.CondValues) &&
			index < len(params.CondNoticeLevels) &&
			index < len(params.CondNoticeMessages) &&
			index < len(params.CondActions) &&
			index < len(params.CondMaxFails) {
			// 校验
			op := params.CondOps[index]
			value := params.CondValues[index]
			if op == agents.ThresholdOperatorRegexp || op == agents.ThresholdOperatorNotRegexp {
				_, err := regexp.Compile(value)
				if err != nil {
					this.Fail("阈值" + param + "正则表达式" + value + "校验失败：" + err.Error())
				}
			}

			// 动作
			actionJSON := params.CondActions[index]

			actionList := []map[string]interface{}{}
			err := json.Unmarshal([]byte(actionJSON), &actionList)
			if err != nil {
				logs.Error(err)
			}

			t := agents.NewThreshold()
			t.Param = param
			t.Operator = op
			t.Value = value
			t.NoticeLevel = types.Uint8(params.CondNoticeLevels[index])
			t.NoticeMessage = params.CondNoticeMessages[index]
			t.Actions = actionList
			t.MaxFails = params.CondMaxFails[index]
			item.AddThreshold(t)
		}
	}
	//fmt.Println("2")

	app.AddItem(item)
	err = agent.Save()
	if err != nil {
		this.Fail("保存失败：" + err.Error())
	}
	//fmt.Println("3")

	// 通知更新
	agentutils.PostAgentEvent(agent.Id, agentutils.NewAgentEvent("ADD_ITEM", maps.Map{
		"appId":  app.Id,
		"itemId": item.Id,
	}))

	if app.IsSharedWithGroup {
		err := agentutils.SyncApp(agent.Id, agent.GroupIds, app, agentutils.NewAgentEvent("ADD_ITEM", maps.Map{
			"appId":  app.Id,
			"itemId": item.Id,
		}), nil)
		if err != nil {
			logs.Error(err)
		}
	}

	this.Data["itemId"] = item.Id

	this.Success()
}
