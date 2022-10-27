package agents

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TeaWeb/build/internal/teaconfigs/forms"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teautils"
	"github.com/iwind/TeaGo/maps"
	"golang.org/x/sync/singleflight"

	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	lockG          = &singleflight.Group{}
	getIcpTokenKey = "check_getIcpTokenKey"
	Cache          = teautils.New(5*time.Minute, 10*time.Minute)
)

// Ping
type IcpCheckSource struct {
	Source `yaml:",inline"`

	Host   string `yaml:"host" json:"host"`
	Domain string `yaml:"domain" json:"domain"`
}
type IcpCache struct {
	Icp      string `json:"icp"`
	UnitName string `json:"unitName"`
	Ok       bool   `json:"ok"`
}

// 获取新对象
func NewIcpCheckSource() *IcpCheckSource {
	return &IcpCheckSource{}
}

// 名称
func (this *IcpCheckSource) Name() string {
	return "备案检测"
}

// 代号
func (this *IcpCheckSource) Code() string {
	return "icpCheck"
}

// 描述
func (this *IcpCheckSource) Description() string {
	return "备案检测"
}

// 执行
func (this *IcpCheckSource) Execute(params map[string]string) (value interface{}, err error) {
	//host := this.Domain

	// 去除http|https|ftp
	//host = regexp.MustCompile(`^(?i)(http|https|ftp)://`).ReplaceAllLiteralString(host, "")
	this.Domain, _ = GetDomain(this.Domain)
	if len(this.Domain) == 0 {
		err = errors.New("'host' should not be empty")
		return maps.Map{
			"ok":       false,
			"unitName": "",
			"icp":      "",
		}, err
	}
	//var ok bool
	value, _, err = this.Posticp(false)
	//fmt.Println("ok==", ok)
	return value, err
}

// 表单信息
func (this *IcpCheckSource) Form() *forms.Form {
	form := forms.NewForm(this.Code())
	{
		group := form.NewGroup()
		{
			field := forms.NewTextField("域名地址", "domain")
			field.IsRequired = true
			field.Code = "domain"
			field.ValidateCode = `
if (value.length == 0) {
	throw new Error("请输入域名地址");
}
`
			field.Comment = "要检测的域名地址"
			group.Add(field)
		}
	}
	return form
}

// 变量
func (this *IcpCheckSource) Variables() []*SourceVariable {
	return []*SourceVariable{
		{
			Code:        "icp",
			Description: "备案信息",
		},
		{
			Code:        "unitName",
			Description: "单位名称",
		},
		{
			Code:        "ok",
			Description: "是否备案",
		},
	}
}

// 阈值
func (this *IcpCheckSource) Thresholds() []*Threshold {
	result := []*Threshold{}

	{
		t := NewThreshold()
		t.Param = "${ok}"
		t.Operator = ThresholdOperatorEq
		t.Value = "false"
		t.NoticeLevel = notices.NoticeLevelWarning
		t.NoticeMessage = "未备案"
		t.MaxFails = 1
		result = append(result, t)
	}

	return result
}

// 显示信息
func (this *IcpCheckSource) Presentation() *forms.Presentation {
	p := forms.NewPresentation()
	p.HTML = `
<tr>
	<td>域名地址</td>
	<td>{{source.domain}}</td>
</tr>
`
	return p
}

func (this *IcpCheckSource) Posticp(needCache bool) (value interface{}, ok bool, err error) {
	value = maps.Map{
		"icp":      "",
		"unitName": "",
		"ok":       false,
	}
	//fmt.Println(this.Domain)
	icpValue, ok := Cache.Get(this.Domain)
	if needCache && ok { //需要缓存(备案监测任务不需要缓存，挂马监测任务时，备案监测需要缓存)
		icpCache := IcpCache{}
		icpByte, _ := json.Marshal(icpValue)
		//fmt.Println("json err:", err)
		json.Unmarshal(icpByte, &icpCache)
		return icpValue, icpCache.Ok, nil
	} else {
		token, err := this.GetToken(getIcpTokenKey)
		if err != nil {
			fmt.Println("get token err=", err)
			return value, false, err
		}
		body := bytes.NewReader([]byte(`{"pageNum":"","pageSize":"","unitName":"` + this.Domain + `"}`))
		req, err := http.NewRequest("POST", "https://hlwicpfwc.miit.gov.cn/icpproject_query/api/icpAbbreviateInfo/queryByCondition", body)
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		req.Header.Set("token", token)
		req.Header.Set("Origin", "https://beian.miit.gov.cn/")
		req.Header.Set("Referer", "https://beian.miit.gov.cn/")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36")
		//req.Header.Set("CLIENT-IP", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36")
		//req.Header.Set("X-FORWARDED-FOR", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36")

		client := teautils.SharedHttpClient(5 * time.Second)
		resp, err := client.Do(req)
		if err != nil {
			return value, false, err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("get res err=", err)
			return value, false, err
		}
		//fmt.Println(string(data))
		icp := &IcpRes{}
		err = json.Unmarshal(data, &icp)
		if err != nil {
			return value, false, err
		}
		if !icp.Success || icp.Code != 200 {
			return value, false, nil
		}
		if icp.Params != nil && len(icp.Params.List) > 0 {
			value = maps.Map{
				"icp":      icp.Params.List[0].ServiceLicence,
				"unitName": icp.Params.List[0].UnitName,
				"ok":       true,
			}
			ok = true
		}
		Cache.Set(this.Domain, value, time.Duration(3600*2)*time.Second)

	}

	return
}

func (this *IcpCheckSource) GetToken(getIcpTokenKey string) (token string, err error) {
	s, e := CheckCache(getIcpTokenKey, this.getToken, 20, true)
	if e != nil {
		return token, e
	}

	//if tokens, ok := Cache.Get(getIcpTokenKey); ok {
	//	token = fmt.Sprintf("%s", tokens)
	//	return token, nil
	//}
	//tokens, ok, _ := lockG.Do(getIcpTokenKey, this.getToken)
	//if ok == nil {
	//	Cache.Set(getIcpTokenKey, tokens, time.Duration(20)*time.Second)
	//
	//}
	token = fmt.Sprintf("%s", s)
	return token, nil

}

func (this *IcpCheckSource) getToken() (token interface{}, err error) {
	token = ""
	now := time.Now().Unix()
	authKey := Md5Str("testtest" + strconv.Itoa(int(now)))
	body := bytes.NewReader([]byte("authKey=" + authKey + "&timeStamp=" + strconv.Itoa(int(now))))
	req, err := http.NewRequest("POST", "https://hlwicpfwc.miit.gov.cn/icpproject_query/api/auth", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("token", "0")
	req.Header.Set("Origin", "https://beian.miit.gov.cn/")
	req.Header.Set("Referer", "https://beian.miit.gov.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36")
	//req.Header.Set("CLIENT-IP", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36")
	//req.Header.Set("X-FORWARDED-FOR", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36")

	client := teautils.SharedHttpClient(5 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return token, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	data, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(data))
	if err != nil {
		return token, err
	}
	tokenres := &TokenRes{}
	err = json.Unmarshal(data, &tokenres)
	if err != nil {
		return token, err
	}
	if !tokenres.Success || tokenres.Code != 200 {
		return token, nil
	}
	if tokenres.Params != nil {
		token = tokenres.Params.Bussiness
	}
	return token, nil
}

type TokenRes struct {
	Code    int          `json:"code"`
	Msg     string       `json:"msg"`
	Params  *TokenParams `json:"params"`
	Success bool         `json:"success"`
}

type TokenParams struct {
	Expire    int    `json:"expire"`
	Refresh   string `json:"refresh"`
	Bussiness string `json:"bussiness"`
}

type IcpRes struct {
	Code    int     `json:"code"`
	Msg     string  `json:"msg"`
	Params  *Params `json:"params"`
	Success bool    `json:"success"`
}

type List struct {
	ContentTypeName  string `json:"contentTypeName"`
	Domain           string `json:"domain"`
	DomainID         int64  `json:"domainId"`
	LeaderName       string `json:"leaderName"`
	LimitAccess      string `json:"limitAccess"`
	MainID           int    `json:"mainId"`
	MainLicence      string `json:"mainLicence"`
	NatureName       string `json:"natureName"`
	ServiceID        int    `json:"serviceId"`
	ServiceLicence   string `json:"serviceLicence"`
	UnitName         string `json:"unitName"`
	UpdateRecordTime string `json:"updateRecordTime"`
}

type Params struct {
	EndRow           int     `json:"endRow"`
	FirstPage        int     `json:"firstPage"`
	HasNextPage      bool    `json:"hasNextPage"`
	HasPreviousPage  bool    `json:"hasPreviousPage"`
	IsFirstPage      bool    `json:"isFirstPage"`
	IsLastPage       bool    `json:"isLastPage"`
	LastPage         int     `json:"lastPage"`
	List             []*List `json:"list"`
	NavigatePages    int     `json:"navigatePages"`
	NavigatepageNums []int   `json:"navigatepageNums"`
	NextPage         int     `json:"nextPage"`
	OrderBy          string  `json:"orderBy"`
	PageNum          int     `json:"pageNum"`
	PageSize         int     `json:"pageSize"`
	Pages            int     `json:"pages"`
	PrePage          int     `json:"prePage"`
	Size             int     `json:"size"`
	StartRow         int     `json:"startRow"`
	Total            int     `json:"total"`
}
