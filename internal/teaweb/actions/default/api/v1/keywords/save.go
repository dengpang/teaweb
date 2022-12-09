package keywords

import (
	"github.com/TeaWeb/build/internal/teaconfigs/keyword"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/api/apiutils"
	"github.com/iwind/TeaGo/actions"
	"strings"
)

type KeywordsSaveAction actions.Action

func (this *KeywordsSaveAction) RunPost(params struct {
	Id      string
	Keyword string
}) {
	//fmt.Println(params)
	//return
	concent := keyword.NewKeywordConfigFromId(params.Id)
	if concent == nil {
		apiutils.Fail(this, "分类不存在")
		return
	}
	if params.Keyword != "" {
		concent.Keyword = params.Keyword
		concent.Keyword = strings.ReplaceAll(concent.Keyword, "，", ",")
		concent.Number = len(strings.Split(concent.Keyword, ","))
	} else {
		concent.Keyword = ""
		concent.Number = 0
	}
	err := concent.Save()
	if err != nil {
		apiutils.Fail(this, "保存失败")
		return
	}
	this.Success()
}
