package agent

import (
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/maps"
)

type DownloadAction actions.Action

// Agent下载文件列表
func (this *DownloadAction) RunGet(params struct {
}) {
	this.Data["download"] = maps.Map{
		"1": "/monit-agent-linux-amd64-v1.5.0.zip",
		"2": "/monit-agent-windows-amd64-v1.5.0.zip",
		"3": "/monit-agent-darwin-amd64-v1.5.0.zip",
		"4": "/monit-agent-freebsd-amd64-v1.5.0.zip",
		"5": "/monit-agent-linux-arm64-v1.5.0.zip",
		"6": "/monit-agent-linux-mips64-v1.5.0.zip",
		"7": "/monit-agent-linux-mips64le-v1.5.0.zip",
	}
	this.Success()
}
