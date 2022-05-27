package forms

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type KeywordCheckBox struct {
	Element
	KeywordParams []KeywordParams `yaml:"keyword_params" json:"keyword_params"`
}
type KeywordParams struct {
	Values    string `yaml:"values" json:"values"`
	IsChecked bool   `yaml:"isChecked" json:"isChecked"`
	Label     string `yaml:"label" json:"label"`
}

func NewKeywordCheckBox(title string, subTitle string, par []KeywordParams) *KeywordCheckBox {
	return &KeywordCheckBox{
		Element: Element{
			Title:    title,
			Subtitle: subTitle,
		},
		KeywordParams: par,
	}
}

func (this *KeywordCheckBox) Super() *Element {
	return &this.Element
}

func (this *KeywordCheckBox) Compose() string {
	ids := []string{}
	idsBy, _ := json.Marshal(this.Value)
	json.Unmarshal(idsBy, &ids)

	html := ""
	for k, v := range this.KeywordParams {
		attrs := map[string]string{
			"name": this.Namespace + "_" + this.Code + strconv.Itoa(k),
		}

		if v.IsChecked || this.InKeywordId(v.Values, ids) {
			attrs["checked"] = "checked"
		}

		attrs["value"] = v.Values

		html += `<div class="ui checkbox">
<input type="checkbox"` + this.ComposeAttrs(attrs) + `/> 
<label>` + v.Label + `</label>
</div>&nbsp;`
	}

	return html

}

func (this *KeywordCheckBox) ApplyRequest(req *http.Request) (value interface{}, skip bool, err error) {

	value = []string{
		req.Form.Get(this.Namespace + "_" + this.Code + "0"), //博彩类
		req.Form.Get(this.Namespace + "_" + this.Code + "1"), //暴恐类
		req.Form.Get(this.Namespace + "_" + this.Code + "2"), //反动类
		req.Form.Get(this.Namespace + "_" + this.Code + "3"), //涉黄类
		req.Form.Get(this.Namespace + "_" + this.Code + "4"), //涉黑类
		req.Form.Get(this.Namespace + "_" + this.Code + "5"), //民生类
		req.Form.Get(this.Namespace + "_" + this.Code + "6"), //政治类
		req.Form.Get(this.Namespace + "_" + this.Code + "7"), //其它类
		req.Form.Get(this.Namespace + "_" + this.Code + "8"), //自定义
	}

	return value, false, nil
}

func (this *KeywordCheckBox) InKeywordId(id string, ids []string) bool {

	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
