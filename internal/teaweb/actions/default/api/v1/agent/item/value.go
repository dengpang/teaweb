package item

import (
	"errors"
	"fmt"
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teadb"
	"github.com/TeaWeb/build/internal/teautils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	timeutil "github.com/iwind/TeaGo/utils/time"
	json "github.com/json-iterator/go"
	"strings"
	"time"
)

type ValueAction actions.Action

// 监控项信息
func (this *ValueAction) RunGet(params struct {
	AgentId   string
	AppId     string
	ItemId    string
	LastId    string
	Level     notices.NoticeLevel
	StartTime int64
	EndTime   int64
}) {
	teautils.SetCache("abc", "abc", time.Minute)
	agent := agents.NewAgentConfigFromId(params.AgentId)
	if agent == nil {
		this.Fail("找不到Agent")
	}

	app := agent.FindApp(params.AppId)
	if app == nil {
		this.Fail("找不到App")
	}

	item := app.FindItem(params.ItemId)
	if item == nil {
		this.Fail("找不到Item")
	}
	var ones []*agents.Value
	key := fmt.Sprint("%s-%s-%s", params.AgentId, params.AppId, params.ItemId)
	var data interface{}
	var err error
	if teautils.RedisCliPing {
		data, err = teautils.GetCache(key)
	} else {
		var ok bool
		data, ok = teautils.CacheCli.Get(key)
		if !ok {
			err = errors.New("fail")
		}
	}

	if err == nil {
		dataByte, err := json.Marshal(data)
		if err != nil {
			this.Fail("查询失败：" + err.Error())
		}
		if err = json.Unmarshal(dataByte, &ones); err != nil {
			this.Fail("查询失败：" + err.Error())
		}
	} else {
		ones, err = teadb.AgentValueDAO().ListItemValuesByTime(params.AgentId, params.AppId, params.ItemId, params.Level, params.LastId, 0, 1, params.StartTime, params.EndTime)
		if err != nil {
			this.Fail("查询失败：" + err.Error())
		}
		if teautils.RedisCliPing {
			teautils.SetCache(key, ones, time.Minute*5)
		} else {
			teautils.CacheCli.Set(key, ones, time.Minute*5)
		}
	}

	source := item.Source()
	this.Data["values"] = lists.Map(ones, func(k int, v interface{}) interface{} {
		value := v.(*agents.Value)

		vars := []maps.Map{}
		if types.IsMap(value.Value) || types.IsSlice(value.Value) {
			if source != nil {
				for _, variable := range source.Variables() {
					if len(variable.Code) == 0 || strings.Index(variable.Code, "$") > -1 {
						continue
					}
					result := teautils.Get(value.Value, strings.Split(variable.Code, "."))
					vars = append(vars, maps.Map{
						"code":        variable.Code,
						"description": variable.Description,
						"value":       stringutil.JSONEncodePretty(result),
					})
				}
			}
		}

		return maps.Map{
			"id":          value.Id.Hex(),
			"costMs":      value.CostMs,
			"value":       value.Value,
			"error":       value.Error,
			"noticeLevel": notices.FindNoticeLevel(value.NoticeLevel),
			"threshold":   value.Threshold,
			"vars":        vars,
			"beginTime":   timeutil.Format("Y-m-d H:i:s", time.Unix(value.CreatedAt, 0)),
			"endTime":     timeutil.Format("Y-m-d H:i:s", time.Unix(value.Timestamp, 0)),
		}
	})
	this.Success()
}
