package keyword

import (
	"github.com/TeaWeb/build/internal/teaconfigs/keyword"
	"github.com/iwind/TeaGo/actions"
	"strings"
)

type UpdateAction actions.Action

func (this *UpdateAction) Run(params struct {
	Id string
}) {
	if params.Id == "" {
		this.FailField("keyword", "请选择类型")
		return
	}
	concent := keyword.NewKeywordConfigFromId(params.Id)
	if concent == nil {
		this.FailField("keyword", "操作失败")
		return
	}
	//if concent.Default {
	//	this.FailField("keyword","内置敏感词暂不支持修改")
	//}
	this.Data["keyword"] = concent.Keyword
	this.Data["keywordName"] = concent.Name
	this.Data["id"] = concent.Id
	this.Data["isDefault"] = concent.Default

	this.Show()
}

func (this *UpdateAction) RunPost(params struct {
	Id      string
	Keyword string
	Must    *actions.Must
}) {
	params.Must.
		Field("Id", params.Id).
		Require("请选择分类")

	concent := keyword.NewKeywordConfigFromId(params.Id)
	if concent == nil {
		this.FailField("keyword", "修改失败")
		return
	}
	if concent.Default {
		this.FailField("keyword", "内置敏感词暂不支持修改")
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
		this.FailField("keyword", "修改失败")
		return
	}
	this.Next("/settings/keyword", nil).Success("保存成功")
}
