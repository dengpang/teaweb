package keyword

import (
	"github.com/TeaWeb/build/internal/teaconfigs/keyword"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
)

type IndexAction actions.Action

// 备份列表
func (this *IndexAction) Run(params struct{}) {
	files := keyword.ActionListFiles()
	this.Data["files"] = lists.Map(files, func(k int, v interface{}) interface{} {
		cfg := v.(*keyword.KeywordConfig)
		return maps.Map{
			"default": cfg.Default,
			"id":      cfg.Id,
			"name":    cfg.Name,
			"number":  cfg.Number,
			"sort":    cfg.Sort,
		}
	})
	this.Show()
}
