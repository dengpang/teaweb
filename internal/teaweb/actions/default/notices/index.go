package notices

import (
	"github.com/TeaWeb/build/internal/teaconfigs"
	"github.com/TeaWeb/build/internal/teaconfigs/agents"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teadb"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/actionutils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/utils/time"
	"time"
)

type IndexAction struct {
	actionutils.ParentAction
}

// 通知
func (this *IndexAction) Run(params struct {
	Read int
	Page int
}) {
	this.Data["isRead"] = params.Read > 0

	count := 0
	countUnread, err := teadb.NoticeDAO().CountAllUnreadNotices()
	if err != nil {
		logs.Error(err)
	}
	if params.Read == 0 {
		count = countUnread
	} else {
		count, err = teadb.NoticeDAO().CountAllReadNotices()
		if err != nil {
			logs.Error(err)
		}
	}

	this.Data["countUnread"] = countUnread
	this.Data["count"] = count
	this.Data["soundOn"] = notices.SharedNoticeSetting().SoundOn

	// 分页
	page := this.NewPage(int64(count))
	end := page.Offset + page.Size
	if page.Offset > int64(count) {
		page.Offset = 0
	}
	if end > int64(count) {
		end = int64(count)
	}
	this.Data["page"] = page.AsHTML()

	//缓存数据
	levelMaps := map[notices.NoticeLevel]maps.Map{}
	agentCfgMaps := map[string]*agents.AgentConfig{}
	appCfgMaps := map[string]*agents.AppConfig{}
	appItemMaps := map[string]*agents.Item{}
	taskCfgMaps := map[string]*agents.TaskConfig{}
	serverCfgMaps := map[string]*teaconfigs.ServerConfig{}
	locationCfgMaps := map[string]*teaconfigs.LocationConfig{}
	// 读取数据
	ones, err := teadb.NoticeDAO().ListNotices(params.Read == 1, int(page.Offset), int(page.Size))
	if err != nil {
		logs.Error(err)
		this.Data["notices"] = []maps.Map{}
	} else {
		this.Data["notices"] = lists.Map(ones, func(k int, v interface{}) interface{} {
			notice := v.(*notices.Notice)
			isAgent := len(notice.Agent.AgentId) > 0
			isProxy := len(notice.Proxy.ServerId) > 0
			m := maps.Map{
				"id":       notice.Id,
				"isAgent":  isAgent,
				"isProxy":  isProxy,
				"isRead":   notice.IsRead,
				"message":  notice.Message,
				"datetime": timeutil.Format("Y-m-d H:i:s", time.Unix(notice.Timestamp, 0)),
			}

			// Agent
			if isAgent {
				level, ok := levelMaps[notice.Agent.Level]
				if !ok {
					level = notices.FindNoticeLevel(notice.Agent.Level)
					levelMaps[notice.Agent.Level] = level
				}
				m["level"] = level

				links := []maps.Map{}
				agent, ok := agentCfgMaps[notice.Agent.AgentId]
				if !ok {
					agent = agents.NewAgentConfigFromId(notice.Agent.AgentId)
					agentCfgMaps[notice.Agent.AgentId] = agent
				}
				if agent != nil {
					links = append(links, maps.Map{
						"name": agent.Name,
						"url":  "/agents/board?agentId=" + agent.Id,
					})
					app, ok := appCfgMaps[notice.Agent.AppId]
					if !ok {
						app = agent.FindApp(notice.Agent.AppId)
						appCfgMaps[notice.Agent.AppId] = app
					}
					//app := agent.FindApp(notice.Agent.AppId)
					if app != nil {
						links = append(links, maps.Map{
							"name": app.Name,
							"url":  "/agents/apps/detail?agentId=" + agent.Id + "&appId=" + app.Id,
						})

						// item
						if len(notice.Agent.ItemId) > 0 {
							item, ok := appItemMaps[notice.Agent.ItemId]
							if !ok {
								item = app.FindItem(notice.Agent.ItemId)
								appItemMaps[notice.Agent.ItemId] = item
							}
							if item != nil {
								links = append(links, maps.Map{
									"name": item.Name,
									"url":  "/agents/apps/itemDetail?agentId=" + agent.Id + "&appId=" + app.Id + "&itemId=" + item.Id,
								})
							}
						}

						// task
						if len(notice.Agent.TaskId) > 0 {
							task, ok := taskCfgMaps[notice.Agent.TaskId]
							if !ok {
								task = app.FindTask(notice.Agent.TaskId)
								taskCfgMaps[notice.Agent.TaskId] = task
							}
							if task != nil {
								links = append(links, maps.Map{
									"name": task.Name,
									"url":  "/agents/apps/itemDetail?agentId=" + agent.Id + "&appId=" + app.Id + "&taskId=" + task.Id,
								})
							}
						}
					}
				}

				m["links"] = links
			}

			// Proxy
			if isProxy {
				level, ok := levelMaps[notice.Agent.Level]
				if !ok {
					level = notices.FindNoticeLevel(notice.Agent.Level)
					levelMaps[notice.Agent.Level] = level
				}
				m["level"] = level

				links := []maps.Map{}
				server, ok := serverCfgMaps[notice.Proxy.ServerId]
				if !ok {
					server = teaconfigs.NewServerConfigFromId(notice.Proxy.ServerId)
					serverCfgMaps[notice.Proxy.ServerId] = server
				}
				if server != nil {
					links = append(links, maps.Map{
						"name": server.Description,
						"url":  "/proxy/board?serverId=" + server.Id,
					})
				}

				if len(notice.Proxy.BackendId) > 0 {
					if len(notice.Proxy.LocationId) > 0 {
						location, ok := locationCfgMaps[notice.Proxy.LocationId]
						if !ok {
							location = server.FindLocation(notice.Proxy.LocationId)
							locationCfgMaps[notice.Proxy.LocationId] = location
						}
						if location != nil {
							links = append(links, maps.Map{
								"name": location.Pattern,
								"url":  "/proxy/locations/detail?serverId=" + server.Id + "&locationId=" + notice.Proxy.LocationId,
							})
							if notice.Proxy.Websocket {
								links = append(links, maps.Map{
									"name": "Websocket",
									"url":  "/proxy/locations/websocket?serverId=" + server.Id + "&locationId=" + notice.Proxy.LocationId,
								})
								links = append(links, maps.Map{
									"name": "后端服务器",
									"url":  "/proxy/locations/websocket?serverId=" + server.Id + "&locationId=" + notice.Proxy.LocationId,
								})
							} else {
								links = append(links, maps.Map{
									"name": "后端服务器",
									"url":  "/proxy/locations/backends?serverId=" + server.Id + "&locationId=" + notice.Proxy.LocationId,
								})
							}
						}
					} else {
						links = append(links, maps.Map{
							"name": "后端服务器",
							"url":  "/proxy/backend?serverId=" + server.Id,
						})
					}
				}

				m["links"] = links
			}

			return m
		})
	}
	this.Show()
}
