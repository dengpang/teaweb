package agents

import (
	"errors"
	"fmt"
	"github.com/TeaWeb/build/internal/teaconfigs/forms"
	"github.com/TeaWeb/build/internal/teaconfigs/keyword"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teaconfigs/shared"
	"github.com/TeaWeb/build/internal/teaconfigs/widgets"
	"github.com/TeaWeb/build/internal/teautils"
	"github.com/iwind/TeaGo/maps"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 敏感词监测
type KeywordCheckSource struct {
	Source `yaml:",inline"`

	Timeout         int                `yaml:"timeout" json:"timeout"` // 连接超时
	URL             string             `yaml:"url" json:"url"`
	Method          string             `yaml:"method" json:"method"`
	Headers         []*shared.Variable `yaml:"headers" json:"headers"`
	Params          []*shared.Variable `yaml:"params" json:"params"`
	TextBody        string             `yaml:"textBody" json:"textBody"`
	Keywords        []string           `yaml:"keywords" json:"keywords"` //需要匹配的 敏感词ID
	Level           string             `yaml:"level" json:"level"`
	KeywordLists    string             `yaml:"keywordLists" json:"keywordLists"`       //需要匹配的敏感词
	KeywordList     []string           `yaml:"keywordList" json:"keywordList"`         //需要匹配的敏感词
	DiyInputKeyword string             `yaml:"diyInputKeyword" json:"diyInputKeyword"` //自定义输入敏感词 ，非选择的自定义类敏感词
}

// 获取新对象
func NewKeywordCheckSource() *KeywordCheckSource {
	return &KeywordCheckSource{}
}

// 名称
func (this *KeywordCheckSource) Name() string {
	return "敏感词监测"
}

// 代号
func (this *KeywordCheckSource) Code() string {
	return "keywordCheck"
}

// 描述
func (this *KeywordCheckSource) Description() string {
	return "获取网页敏感词信息"
}

// 执行
func (this *KeywordCheckSource) Execute(params map[string]string) (value interface{}, err error) {
	//获取需要匹配的敏感词
	this.KeywordList = strings.Split(this.KeywordLists, ",")
	//追加自定义关键词
	if this.DiyInputKeyword != "" {
		diyKeyword := strings.Split(this.DiyInputKeyword, ",")
		this.KeywordList = append(this.KeywordList, diyKeyword...)
	}
	if len(this.URL) == 0 {
		err = errors.New("'url' should not be empty")
		return maps.Map{
			"status":     0,
			"scanList":   "",
			"scanNum":    0,
			"keywords":   "",
			"keywordNum": 0,
		}, err
	}
	level, err := strconv.Atoi(this.Level)
	if err != nil {
		level = 2
	}
	method := this.Method
	if len(method) == 0 {
		method = http.MethodGet
	}

	var body io.Reader = nil

	before := time.Now()
	req, err := http.NewRequest(method, this.URL, body)
	if err != nil {
		value = maps.Map{
			"cost":       time.Since(before).Seconds(),
			"status":     0,
			"scanList":   "",
			"scanNum":    0,
			"keywords":   "",
			"keywordNum": 0,
		}
		return value, err
	}

	client := teautils.SharedHttpClient(time.Duration(5) * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return maps.Map{
			"status":     0,
			"scanList":   "",
			"scanNum":    0,
			"keywords":   "",
			"keywordNum": 0,
		}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return maps.Map{
			"status":     0,
			"scanList":   "",
			"scanNum":    0,
			"keywords":   "",
			"keywordNum": 0,
		}, err
	}
	//获取网页内的url
	hitUrls := this.MatchUrl(data)
	urlMap := map[string]struct{}{}
	//匹配第一级网页敏感词
	hitKeyword := []string{}
	hitKeyword = this.MatchKeyword(data)

	if len(hitUrls) > 0 {
		var (
			urlLock = &sync.Mutex{}
			keyLock = &sync.Mutex{}
			wg      = &sync.WaitGroup{}
			chMax   = make(chan struct{}, 20)
		)
		//下探一级
		for _, v1 := range hitUrls {
			chMax <- struct{}{}
			wg.Add(1)
			go func(v1 string) {
				defer func() {
					wg.Done()
					<-chMax
				}()
				urlLock.Lock()
				if _, ok := urlMap[v1]; ok {
					urlLock.Unlock()
					//已存在
					return
				} else {
					urlMap[v1] = struct{}{}
				}
				urlLock.Unlock()
				resp1, err := this.GetQuery(v1)
				if err != nil {
					return
				}
				data1, err := ioutil.ReadAll(resp1.Body)
				if err != nil {
					return
				}
				//匹配敏感词
				hitKeyword1 := this.MatchKeyword(data1)
				if len(hitKeyword1) > 0 {
					keyLock.Lock()
					hitKeyword = append(hitKeyword, hitKeyword1...)
					keyLock.Unlock()
				}
				//下探2级
				if level >= 2 {
					hitUrls1 := this.MatchUrl(data1)
					if len(hitUrls1) > 0 {
						//下探二级
						for _, v2 := range hitUrls1 {
							urlLock.Lock()
							if _, ok := urlMap[v2]; ok {
								urlLock.Unlock()
								//已存在
								continue
							} else {
								urlMap[v2] = struct{}{}
							}
							urlLock.Unlock()
							resp2, err := this.GetQuery(v2)
							if err != nil {
								continue
							}
							data2, err := ioutil.ReadAll(resp2.Body)
							if err != nil {
								continue
							}
							//匹配敏感词
							hitKeyword2 := this.MatchKeyword(data2)
							if len(hitKeyword2) > 0 {
								keyLock.Lock()
								hitKeyword = append(hitKeyword, hitKeyword2...)
								keyLock.Unlock()
							}
							//下探3级
							if level >= 3 {
								hitUrls2 := this.MatchUrl(data2)
								if len(hitUrls2) > 0 {
									//下探三级
									for _, v3 := range hitUrls2 {
										urlLock.Lock()
										if _, ok := urlMap[v3]; ok {
											urlLock.Unlock()
											//已存在
											continue
										} else {
											urlMap[v3] = struct{}{}
										}
										urlLock.Unlock()
										resp3, err := this.GetQuery(v3)
										if err != nil {
											continue
										}
										data3, err := ioutil.ReadAll(resp3.Body)
										if err != nil {
											continue
										}
										//匹配敏感词
										hitKeyword3 := this.MatchKeyword(data3)
										if len(hitKeyword3) > 0 {
											keyLock.Lock()
											hitKeyword = append(hitKeyword, hitKeyword3...)
											keyLock.Unlock()
										}
									}
								}
							}
						}
					}
				}
			}(v1)
		}
		wg.Wait()
	}
	//
	urlRes := []string{}
	for k, _ := range urlMap {
		//取20个 地址
		if len(urlRes) > 20 {
			break
		}
		urlRes = append(urlRes, k)
	}

	//敏感词去重
	hitKeywordMap := map[string]struct{}{}
	for _, v := range hitKeyword {
		hitKeywordMap[v] = struct{}{}
	}
	hitKeywordList := []string{}
	for k, _ := range hitKeywordMap {
		hitKeywordList = append(hitKeywordList, k)
	}

	value = maps.Map{
		"cost":       time.Since(before).Seconds(),
		"status":     resp.StatusCode,
		"scanList":   strings.Join(urlRes, `, `),
		"scanNum":    len(urlMap),
		"keywords":   strings.Join(hitKeywordList, ","),
		"keywordNum": len(hitKeywordList),
	}

	return
}

// get请求
func (this *KeywordCheckSource) GetQuery(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := teautils.SharedHttpClient(time.Duration(5) * time.Second)
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return resp, err
}

//查找网页源码内的URL地址
func (this *KeywordCheckSource) MatchUrl(s []byte) (urls []string) {
	// href=('|").*?('|")   [a-zA-z]+://[^\s]*("|')
	urls = []string{}
	urlMap := map[string]struct{}{}
	//先匹配 :// 这类地址
	re, err := regexp.Compile(`[a-zA-z]+://[^\s]+\.[^\s]+('|")`)
	if err != nil {
		return urls
	}
	list := re.FindAll(s, -1)
	if len(list) > 0 {
		for _, v := range list {
			url := string(v)
			urlMap[url] = struct{}{}
		}
	}
	//在匹配 href="" 这类地址
	re, err = regexp.Compile(`href=('|").*?('|")`)
	if err != nil {
		return urls
	}
	list = re.FindAll(s, -1)
	if len(list) > 0 {
		for _, v := range list {
			url := string(v)
			urlMap[url] = struct{}{}
		}
	}
	re1, _ := regexp.Compile(`^href=("|')|('|"|\))[^\s]*?$`)
	re2, _ := regexp.Compile(`^http`)
	re3, _ := regexp.Compile(`(http|https)://(www.)?(\w+(\.)?)+`) //获取域名正则
	re4, _ := regexp.Compile(`\\u0026amp\;`)                      // &符号
	re5, _ := regexp.Compile(`\\u0026`)                           // &符号
	domain := re3.Find(s)
	for v := range urlMap {
		//正则替换
		url := re1.ReplaceAllString(v, "")
		url = re4.ReplaceAllString(url, "&")
		url = re5.ReplaceAllString(url, "&")
		if url == "" {
			continue
		}
		if url != "/" && url != "#" && url != "?" {
			if re2.MatchString(url) { //是包含http的地址
				urls = append(urls, url)
			} else {
				urls = append(urls, string(domain)+url)
			}

		}

	}
	return urls
}

//匹配敏感词
func (this *KeywordCheckSource) MatchKeyword(s []byte) (keyword []string) {
	keyword = []string{}
	if len(this.KeywordList) > 0 {
		//regexp.Compile(`\\\^\$\*\+\?\{\}\.\[\]\(\)\-\|`)
		for _, reg_rule := range this.KeywordList {
			reg := regexp.MustCompile(reg_rule)
			if reg.Match(s) {
				keyword = append(keyword, reg_rule)
				continue
			}
		}
	}
	return keyword
}

// 表单信息
func (this *KeywordCheckSource) Form() *forms.Form {
	form := forms.NewForm(this.Code())
	{
		group := form.NewGroup()
		{
			field := forms.NewTextField("URL ", "url")
			field.Comment = "http://"
			field.IsRequired = true
			field.Code = "url"
			field.Placeholder = "http://"
			field.ValidateCode = `
if (value.length == 0) {
	throw new Error("请输入url地址");
}`
			group.Add(field)
		}

		{
			keywords := keyword.ActionListFiles()
			par := []forms.KeywordParams{}
			defaultValue := []string{}
			for _, v := range keywords {
				par = append(par, forms.KeywordParams{
					Values: v.Id, Label: v.Name,
				})
				defaultValue = append(defaultValue, v.Id)
			}
			field := forms.NewKeywordCheckBox("敏感词类型", "多选", par)
			field.Value = defaultValue
			field.Comment = "默认全部选中"
			field.Code = "keywords"

			group.Add(field)
		}
		{
			field := forms.NewOptions("下探等级", "")
			field.Code = "level"
			field.AddOption("一级", "1")
			field.AddOption("二级", "2")
			field.AddOption("三级", "3")
			field.Value = "3"
			field.Attr("style", "width:10em")
			group.Add(field)
		}
	}

	return form
}

// 变量
func (this *KeywordCheckSource) Variables() []*SourceVariable {
	return []*SourceVariable{
		{
			Code:        "cost",
			Description: "请求耗时（秒）",
		},
		{
			Code:        "status",
			Description: "HTTP状态码",
		},
		{
			Code:        "scanNum",
			Description: "扫描地址总数",
		},
		{
			Code:        "scanList",
			Description: "已扫描地址(前20个)",
		},
		//{
		//	Code:        "length",
		//	Description: "响应的内容长度",
		//},
		{
			Code:        "keywordNum",
			Description: "敏感词数量",
		},
		{
			Code:        "keywords",
			Description: "敏感词",
		},
	}
}

// 阈值
func (this *KeywordCheckSource) Thresholds() []*Threshold {
	result := []*Threshold{}

	// 阈值
	{
		t := NewThreshold()
		t.Param = "${keywordNum}"
		t.Operator = ThresholdOperatorGte
		t.Value = "1"
		t.NoticeLevel = notices.NoticeLevelWarning
		t.NoticeMessage = "匹配到敏感词：${keywords}"
		result = append(result, t)
	}

	return result
}

// 图表
func (this *KeywordCheckSource) Charts() []*widgets.Chart {
	charts := []*widgets.Chart{}

	{
		// chart
		chart := widgets.NewChart()
		chart.Name = "敏感词监测"
		chart.Columns = 2
		chart.Type = "javascript"
		chart.SupportsTimeRange = true
		chart.Options = maps.Map{
			"code": `var chart = new charts.LineChart();

var ones = NewQuery().past(60, time.MINUTE).avg("keywordNum");

var line = new charts.Line();
line.isFilled = true;

ones.$each(function (k, v) {
	if (v.value == "") {
		return;
	}
	line.addValue(v.value.keywordNum );
	chart.addLabel(v.label);
});

chart.addLine(line);
chart.render();`,
		}

		charts = append(charts, chart)
	}

	return charts
}

// 显示信息
func (this *KeywordCheckSource) Presentation() *forms.Presentation {
	keyName := ""
	if len(this.Keywords) > 0 {
		for _, v := range this.Keywords {
			if v != "" {
				keyInfo := keyword.NewKeywordConfigFromId(v)
				if keyInfo != nil {
					keyName = fmt.Sprintf("%s,%s", keyName, keyInfo.Name)
				}
			}

		}
		keyName = strings.Trim(keyName, ",")
	}
	return &forms.Presentation{
		HTML: `
			<tr>
				<td class="color-border">URL</td>
				<td>{{source.url}}</td>
			</tr>
			
			<tr>
				<td class="color-border">敏感词</td>
				<td>` + keyName + `</td>
			</tr>
			`,
		CSS: `.keywordCheck-block-body {
			border: 1px #eee solid;
			padding: 0.4em;
			background: rgba(0, 0, 0, 0.01);
			font-size: 0.9em;
			max-height: 10em;
			overflow-y: auto;
			margin: 0;
		}
		
		.keywordCheck-block-body::-webkit-scrollbar {
			width: 4px;
		}
		`,
	}
}

func (this *KeywordCheckSource) lookupHeader(name string) (value string, ok bool) {
	for _, h := range this.Headers {
		if h.Name == name {
			return h.Value, true
		}
	}
	return "", false
}
