package keywords

import (
	"github.com/TeaWeb/build/internal/teaconfigs/keyword"
	"github.com/TeaWeb/build/internal/teaweb/actions/default/api/apiutils"
	"github.com/iwind/TeaGo/actions"
)

type KeywordsAction actions.Action

func (this *KeywordsAction) RunGet(params struct{}) {
	result := keyword.ActionListFiles()
	list := make([]*keyword.KeywordConfig, 0)
	//todo api接口不需要自定义敏感词
	if len(result) > 0 {
		for _, v := range result {
			if v.Id == "a994a9aa58c94a5c" { //自定义类
				continue
			}
			list = append(list, v)
		}
	}
	apiutils.Success(this, list)
}
