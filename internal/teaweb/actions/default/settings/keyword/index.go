package keyword

import (
	"github.com/TeaWeb/build/internal/teaconfigs/keyword"
	"github.com/iwind/TeaGo/actions"
)

type IndexAction actions.Action

// 备份列表
func (this *IndexAction) Run(params struct{}) {
	this.Data["files"] = keyword.ActionListFiles()

	this.Show()
}
