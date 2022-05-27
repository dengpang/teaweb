package teadb

import (
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/iwind/TeaGo/logs"
)

type SQLKeywordValueDAO struct {
	BaseDAO
}

// 初始化
func (this *SQLKeywordValueDAO) Init() {

}

// 表名
func (this *SQLKeywordValueDAO) TableName() string {
	table := "teaweb_keyword_value"
	this.initTable(table)
	return table
}

// 写入一个通知
func (this *SQLKeywordValueDAO) InsertOne(notice *notices.Notice) error {
	return NewQuery(this.TableName()).
		InsertOne(notice)
}

func (this *SQLKeywordValueDAO) initTable(table string) {
	if isInitializedTable(table) {
		return
	}

	logs.Println("[db]check table '" + table + "'")

	switch sharedDBType {
	case "mysql":
		err := this.driver.(SQLDriverInterface).CreateTable(table, "CREATE TABLE `"+table+"` ("+
			"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"+
			"`_id` varchar(24) DEFAULT NULL,"+
			"`timestamp` int(11) unsigned DEFAULT '0',"+
			"`keyword` varchar(1024) DEFAULT NULL,"+
			"`messageHash` varchar(64) DEFAULT NULL,"+
			"`isRead` tinyint(1) unsigned DEFAULT '0',"+
			"`isNotified` tinyint(1) unsigned DEFAULT '0',"+
			"`receivers` varchar(1024) DEFAULT NULL,"+
			"`proxyServerId` varchar(64) DEFAULT NULL,"+
			"`proxyWebsocket` tinyint(1) unsigned DEFAULT '0',"+
			"`proxyLocationId` varchar(64) DEFAULT NULL,"+
			"`proxyRewriteId` varchar(64) DEFAULT NULL,"+
			"`proxyBackendId` varchar(64) DEFAULT NULL,"+
			"`proxyFastcgiId` varchar(64) DEFAULT NULL,"+
			"`level` tinyint(1) unsigned DEFAULT '0',"+
			"`agentId` varchar(64) DEFAULT NULL,"+
			"`agentAppId` varchar(64) DEFAULT NULL,"+
			"`agentTaskId` varchar(64) DEFAULT NULL,"+
			"`agentItemId` varchar(64) DEFAULT NULL,"+
			"`agentThreshold` varchar(1024) DEFAULT NULL,"+
			"PRIMARY KEY (`id`),"+
			"UNIQUE KEY `_id` (`_id`),"+
			"KEY `messageHash` (`messageHash`),"+
			"KEY `agentId` (`agentId`),"+
			"KEY `isRead` (`isRead`)"+
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")
		if err != nil {
			logs.Error(err)
			removeInitializedTable(table)
		}

	case "postgres":
		err := this.driver.(SQLDriverInterface).CreateTable(table, `CREATE TABLE "public"."`+table+`" (
		"id" serial8 primary key,
		"_id" varchar(24),
		"timestamp" int4 default 0,
		"message" varchar(1024),
		"messageHash" varchar(64),
		"isRead" int2 default 0,
		"isNotified" int2 default 0,
		"receivers" varchar(1024),
		"proxyServerId" varchar(64),
		"proxyWebsocket" int2 default 0,
		"proxyLocationId" varchar(64),
		"proxyRewriteId" varchar(64),
		"proxyBackendId" varchar(64),
		"proxyFastcgiId" varchar(64),
		"level" int2 default 0,
		"agentId" varchar(64),
		"agentAppId" varchar(64),
		"agentTaskId" varchar(64),
		"agentItemId" varchar(64),
		"agentThreshold" varchar(1024)
		)
		;

		CREATE UNIQUE INDEX "`+table+`_id" ON "public"."`+table+`" ("_id");
		CREATE INDEX "`+table+`_messageHash" ON "public"."`+table+`" ("messageHash");
		CREATE INDEX "`+table+`_agentId" ON "public"."`+table+`" ("agentId");
		CREATE INDEX "`+table+`_isRead" ON "public"."`+table+`" ("isRead");
		`)
		if err != nil {
			logs.Error(err)
			removeInitializedTable(table)
		}
	}
}

// 字段映射
func (this *SQLKeywordValueDAO) mapField(k string) string {
	switch k {
	case "agent.agentId":
		k = "agentId"
	case "agent.appId":
		k = "agentAppId"
	case "agent.itemId":
		k = "agentItemId"
	case "agent.level":
		k = "level"
	case "agent.threshold":
		k = "agentThreshold"
	case "proxy.serverId":
		k = "proxyServerId"
	case "proxy.websocket":
		k = "proxyWebsocket"
	case "proxy.locationId":
		k = "proxyLocationId"
	case "proxy.rewriteId":
		k = "proxyRewriteId"
	case "proxy.fastcgiId":
		k = "proxyFastcgiId"
	case "proxy.backendId":
		k = "proxyBackendId"
	case "proxy.level":
		k = "level"
	}
	return k
}
