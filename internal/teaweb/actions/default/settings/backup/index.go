package backup

import (
	"github.com/TeaWeb/build/internal/teaweb/actions/default/actionutils"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/settings/backup/backuputils"
)

type IndexAction struct {
	actionutils.ParentAction
}

// 备份列表
func (this *IndexAction) Run(params struct{}) {
	//this.Data["files"] = backuputils.ActionListFiles()
	files := backuputils.ActionListFiles()

	page := this.NewPage(int64(len(files)))
	end := page.Offset + page.Size
	if page.Offset > int64(len(files)) {
		page.Offset = 0
	}
	if end > int64(len(files)) {
		end = int64(len(files))
	}
	this.Data["page"] = page.AsHTML()
	this.Data["files"] = files[page.Offset:end]
	this.Data["shouldRestart"] = backuputils.ShouldRestart()

	this.Show()
}
