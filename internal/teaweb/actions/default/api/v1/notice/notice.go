package notice

import (
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teadb"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/agents/agentutils"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"time"
)

type IndexAction actions.Action

// 通知首页
func (this *IndexAction) Run(params struct {
	AgentId   string
	AppId     string
	ItemId    string
	StartTime int
	EndTime   int
}) {
	this.Data["agentId"] = params.AgentId

	// 读取数据
	ones, err := teadb.NoticeDAO().ListAgentNoticesByItem(params.AgentId, false, 0, 1, teadb.Item{
		AppId:     params.AppId,
		ItemId:    params.ItemId,
		Level:     2, //告警
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
	})
	if err != nil {
		logs.Error(err)
		this.Data["notices"] = []maps.Map{}
	} else {
		this.Data["notices"] = lists.Map(ones, func(k int, v interface{}) interface{} {
			notice := v.(*notices.Notice)
			isAgent := len(notice.Agent.AgentId) > 0
			if len(notice.Agent.Threshold) > 0 {
				notice.Message += " [触发阈值：" + notice.Agent.Threshold + "]"
			}
			m := maps.Map{
				"id":       notice.Id,
				"isAgent":  isAgent,
				"isRead":   notice.IsRead,
				"message":  notice.Message,
				"datetime": timeutil.Format("Y-m-d H:i:s", time.Unix(notice.Timestamp, 0)),
			}

			// Agent
			if isAgent {
				m["level"] = notices.FindNoticeLevel(notice.Agent.Level)
				m["links"] = agentutils.FindNoticeLinks(notice)
			}

			return m
		})
	}

	this.Success()
}
