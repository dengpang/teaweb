package agents

import (
	"context"
	"errors"
	"fmt"
	"github.com/TeaWeb/build/internal/teaconfigs/forms"
	"github.com/TeaWeb/build/internal/teaconfigs/keyword"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teaconfigs/shared"
	"github.com/TeaWeb/build/internal/teaconfigs/widgets"
	"github.com/iwind/TeaGo/maps"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var keyCateMap = sync.Map{}

// 敏感词监测
type KeywordCheckSource struct {
	Source `yaml:",inline"`

	Timeout         int                `yaml:"timeout" json:"timeout"` // 连接超时
	URL             string             `yaml:"url" json:"url"`
	Method          string             `yaml:"method" json:"method"`
	Headers         []*shared.Variable `yaml:"headers" json:"headers"`
	Params          []*shared.Variable `yaml:"params" json:"params"`
	TextBody        string             `yaml:"textBody" json:"textBody"`
	Keywords        []string           `yaml:"keywords" json:"keywords"`     //需要匹配的 敏感词ID
	KeywordId       []string           `yaml:"keyword_id" json:"keyword_id"` //需要匹配的 敏感词ID
	Level           string             `yaml:"level" json:"level"`
	KeywordLists    []string           `yaml:"keywordLists" json:"keywordLists"`       //需要匹配的敏感词
	KeywordList     [][]string         `yaml:"keywordList" json:"keywordList"`         //需要匹配的敏感词
	DiyInputKeyword string             `yaml:"diyInputKeyword" json:"diyInputKeyword"` //自定义输入敏感词 ，非选择的自定义类敏感词
}

func init() {
	keyCateMap.Store("2ebfba95c691c02e", "博彩类")
	keyCateMap.Store("8d5abb6c740a965e", "反动类")
	keyCateMap.Store("9e4ad5af6922584a", "涉黑类")
	keyCateMap.Store("98216a07530cc9ef", "政治类")
	keyCateMap.Store("be977ee8ea41b6be", "民生类")
	keyCateMap.Store("df81dd548ed4fb68", "暴恐类")
	keyCateMap.Store("dfc49c00d82454bf", "涉黄类")
	keyCateMap.Store("9d274f7ec0294e71", "其它类")
	keyCateMap.Store("a994a9aa58c94a5c", "自定义")
	keyCateMap.Store("diy", "自定义")
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
	this.KeywordList = make([][]string, 0)
	this.KeywordId = make([]string, 0)
	for _, v := range this.KeywordLists {
		k := strings.Split(v, ",")
		this.KeywordList = append(this.KeywordList, k)
	}
	this.KeywordId = append(this.Keywords)
	//this.KeywordList = strings.Split(this.KeywordLists, ",")
	//追加自定义关键词
	if this.DiyInputKeyword != "" {
		diyKeyword := strings.Split(this.DiyInputKeyword, ",")
		this.KeywordList = append(this.KeywordList, diyKeyword)
		this.KeywordId = append(this.KeywordId, "diy")
	}
	if len(this.URL) == 0 {
		err = errors.New("'url' should not be empty")
		return maps.Map{
			"cost":       0,
			"status":     0,
			"scanList":   "",
			"scanNum":    0,
			"keywords":   make([]CheckRes, 0),
			"keywordNum": 0,
			"cate":       []Cates{},
		}, err
	}
	levelOn := int(0)
	level, err := strconv.Atoi(this.Level)
	if err != nil {
		level = 2
	}

	before := time.Now()
	var ctxs chan context.Context
	for time.Now().Before(before.Add(2 * time.Hour)) {
		//任务并发执行的时候 一定会出现获取窗口达到上限，这里使用两小时内重复获取
		s := GenRandSecond()
		ctxs, err = getWindowCtx()
		if err != nil {
			<-time.Tick(time.Second * time.Duration(s))
		} else {
			break
		}
	}
	if err != nil {
		return maps.Map{
			"cost":       0,
			"status":     0,
			"scanList":   "",
			"scanNum":    0,
			"keywords":   make([]CheckRes, 0),
			"keywordNum": 0,
			"cate":       []Cates{},
		}, err
	}
	//fmt.Println("窗口数==", len(ctxs))
	engine, page, err := chromeDpRun(this.URL, <-ctxs)
	ctxs <- engine.Context
	defer CloseWindow(ctxs)
	//defer engine.UnLockTargetId()
	//fmt.Println(page)
	if err != nil {
		value = maps.Map{
			"cost":       time.Since(before).Seconds(),
			"status":     0,
			"scanList":   "",
			"scanNum":    0,
			"keywords":   make([]CheckRes, 0),
			"keywordNum": 0,
			"cate":       []Cates{},
		}
		return value, err
	}
	//监测结果
	checkRes := map[string]CheckRes{}
	if ok, res := this.MatchKeyword(this.URL, page); ok && len(res) > 0 {
		for k, v := range res {
			checkRes[k] = v
		}
	}
	engine.DomainTop, engine.Domain = engine.GetDomain(this.URL)
	Urls, _, err := engine.GetUrlsAndCheck(page, engine.DomainTop, engine.Domain, this.URL, 1)
	//已经请求过的url
	urlExistsMap := map[string]struct{}{
		this.URL: {},
	}
	//需要请求的url
	urlMap := map[string]struct{}{}
	newUrls := []string{} //新url
	var (
		urlLock    = &sync.Mutex{}
		newUrlLock = &sync.Mutex{}
		resLock    = &sync.Mutex{}
		wg         = &sync.WaitGroup{}
		//chMax      = make(chan struct{}, 1) //浏览器窗口数
	)
	//fmt.Println("=========================loop")
LOOP:
	newUrls, urlMap = []string{}, map[string]struct{}{} //重置
	urlMap = engine.duplicateRemovalUrl(Urls, urlMap)
	//fmt.Println("执行次数  levelOn=", levelOn)
	//下探等级大于等于当前的等级  并且 需要请求的url地址不为空
	if level >= levelOn && len(urlMap) > 0 {

		levelOn++ //当前下探级数
		//下探
		for k1, _ := range urlMap {
			urlLock.Lock()
			if _, ok := urlExistsMap[k1]; ok {
				urlLock.Unlock()
				//已存在
				continue
			} else {
				urlExistsMap[k1] = struct{}{}
			}
			urlLock.Unlock()

			winCtx := <-ctxs
			wg.Add(1)
			go func(ctx context.Context, v1 string) {
				defer func() {
					ctxs <- ctx
					wg.Done()
				}()

				//fmt.Println("url == ", v1, "level==", levelOn)

				_, subHtml, err := chromeDpRun(v1, ctx)
				//defer en.Close()
				if err != nil {
					return
				}
				if level > levelOn { //满足继续下探  才收集下级url地址
					new_urls, _, err := engine.GetUrlsAndCheck(subHtml, engine.DomainTop, engine.Domain, v1, 1)
					//fmt.Println("new_urls==", new_urls)
					if err == nil {
						newUrlLock.Lock()
						newUrls = append(newUrls, new_urls...)
						newUrlLock.Unlock()

					}
				}

				if ok, res := this.MatchKeyword(v1, subHtml); ok && len(res) > 0 {
					resLock.Lock()
					for k, v := range res {
						checkRes[k] = v
					}
					resLock.Unlock()
				}
			}(winCtx, k1)
		}
		wg.Wait()

		Urls = []string{}
		Urls = append(Urls, newUrls...)
		//fmt.Println("Urls==", Urls)
		goto LOOP
	}
	//
	urlRes := []string{}
	for k, _ := range urlExistsMap {
		//取20个 地址
		if len(urlRes) > 20 {
			break
		}
		urlRes = append(urlRes, k)
	}

	list, cateMap, listMap, n, cateSli := []CheckRes{}, map[string]int{}, map[string]CheckRes{}, 0, []Cates{}
	for _, v := range checkRes {
		n += v.Number
		if _, ok := cateMap[v.Cate]; ok {
			cateMap[v.Cate] += v.Number

		} else {
			cateMap[v.Cate] = v.Number

		}

		if value, ok := listMap[v.Url]; ok {
			listMap[v.Url] = CheckRes{
				Url: v.Url, Value: fmt.Sprintf("%s,%s", v.Value, value.Value),
			}
		} else {
			listMap[v.Url] = CheckRes{
				Url: v.Url, Value: v.Value,
			}
		}
	}
	for _, v := range listMap {
		list = append(list, v)
	}

	for k, v := range cateMap {
		cateSli = append(cateSli, Cates{
			Name:  k,
			Value: v,
		})
	}

	value = maps.Map{
		"cost":       time.Since(before).Seconds(),
		"status":     200,
		"scanList":   strings.Join(urlRes, `, `),
		"scanNum":    len(urlExistsMap),
		"keywords":   list,
		"keywordNum": n,
		"cate":       cateSli,
	}

	return
}

// 匹配敏感词
func (this *KeywordCheckSource) MatchKeyword(url string, s []*Page) (ok bool, keyword map[string]CheckRes) {
	keyword = make(map[string]CheckRes, 0)
	if len(this.KeywordList) > 0 {
		//regexp.Compile(`\\\^\$\*\+\?\{\}\.\[\]\(\)\-\|`)
		for i, reg_rule := range this.KeywordList {
			if len(reg_rule) == 0 {
				continue
			}
			cateName := "未知"
			//fmt.Println(len(this.KeywordId), len(this.KeywordList))
			if len(this.KeywordId) == len(this.KeywordList) {
				//fmt.Println("i=", i)
				if name, ok := keyCateMap.Load(this.KeywordId[i]); ok {
					cateName = fmt.Sprintf("%s", name)
				}
			}
			//fmt.Println("cateName==", cateName)
			for _, key := range reg_rule {
				if key == "" {
					continue
				}
				reg := regexp.MustCompile(key)
				if len(s) > 0 {
					for _, html := range s {
						if reg.Match([]byte(html.Html)) {
							if checkRes, ok := keyword[Md5Str(url+cateName)]; ok {
								keyword[Md5Str(url+cateName)] = CheckRes{
									Url:    url,
									Value:  fmt.Sprintf("%s,%s", key, checkRes.Value),
									Cate:   cateName,
									Number: checkRes.Number + 1,
								}
							} else {
								keyword[Md5Str(url+cateName)] = CheckRes{
									Url:    url,
									Value:  key,
									Cate:   cateName,
									Number: 1,
								}
							}

							continue
						}
					}
				}
			}

		}
	}
	return len(keyword) > 0, keyword
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
