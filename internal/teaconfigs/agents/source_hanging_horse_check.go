package agents

import (
	"errors"
	"github.com/TeaWeb/build/internal/teaconfigs/forms"
	"github.com/TeaWeb/build/internal/teaconfigs/notices"
	"github.com/TeaWeb/build/internal/teaconfigs/shared"
	"github.com/TeaWeb/build/internal/teaconfigs/widgets"
	"github.com/iwind/TeaGo/maps"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 敏感词监测
type HangingHouseCheckSource struct {
	Source `yaml:",inline"`

	Timeout  int                `yaml:"timeout" json:"timeout"` // 连接超时
	URL      string             `yaml:"url" json:"url"`
	Method   string             `yaml:"method" json:"method"`
	Headers  []*shared.Variable `yaml:"headers" json:"headers"`
	Params   []*shared.Variable `yaml:"params" json:"params"`
	TextBody string             `yaml:"textBody" json:"textBody"`
	Level    string             `yaml:"level" json:"level"`
}

// 获取新对象
func NewHangingHouseCheckSource() *HangingHouseCheckSource {
	return &HangingHouseCheckSource{}
}

// 名称
func (this *HangingHouseCheckSource) Name() string {
	return "挂马监测"
}

// 代号
func (this *HangingHouseCheckSource) Code() string {
	return "hangingHouseCheck"
}

// 描述
func (this *HangingHouseCheckSource) Description() string {
	return "获取网页挂马信息"
}

// 执行
func (this *HangingHouseCheckSource) Execute(params map[string]string) (value interface{}, err error) {

	if len(this.URL) == 0 {
		err = errors.New("'url' should not be empty")
		return maps.Map{
			"status":   0,
			"scanList": "",
			"scanNum":  0,
			"list":     make([]CheckRes, 0),
			"number":   0,
		}, err
	}
	levelOn := int(0)
	level, err := strconv.Atoi(this.Level)
	if err != nil {
		level = 2
	}

	//var body io.Reader = nil

	before := time.Now()
	if !checkChromePort() {
		err = errors.New("chromeDp 未运行")
		return maps.Map{
			"cost":     0,
			"status":   0,
			"list":     make([]CheckRes, 0),
			"scanList": "",
			"scanNum":  0,
			"number":   0,
		}, err
	}
	engine, html, err := chromeDpRun(this.URL, nil)
	defer engine.UnLockTargetId()
	if err != nil {
		value = maps.Map{
			"cost":     time.Since(before).Seconds(),
			"status":   0,
			"scanList": "",
			"scanNum":  0,
			"list":     make([]CheckRes, 0),
			"number":   0,
		}
		return value, err
	}
	domainTop, domain := GetDomain(this.URL)
	Urls, _, err := GetUrlsAndCheck(html, domainTop, domain, this.URL, 3)
	//监测结果
	checkRes := map[string]CheckRes{}
	if ok, res := checkIframeHangingHorse(html, this.URL, domainTop); ok && len(res) > 0 {
		for k, v := range res {
			checkRes[k] = v
		}
	}
	if engine.Location != "" && engine.Location != "chrome-error://chromewebdata/" {
		if scriptHanging := checkScriptHangingHorse(domainTop, this.URL, engine.Location); len(scriptHanging) > 0 {
			for k, v := range scriptHanging {
				checkRes[k] = v
			}
		}
	}
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
		chMax      = make(chan struct{}, 1) //浏览器窗口数
	)
LOOP:
	newUrls, urlMap = []string{}, map[string]struct{}{} //重置
	urlMap = duplicateRemovalUrl(Urls, urlMap)
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

			chMax <- struct{}{}
			wg.Add(1)
			go func(v1 string) {
				defer func() {
					wg.Done()
					<-chMax
				}()

				//fmt.Println("url == ", v1, "level==", levelOn)

				engineSub, subHtml, err := chromeDpRun(v1, engine.Context)
				if err != nil {
					return
				}
				if level > levelOn { //满足继续下探  才收集下级url地址
					new_urls, _, err := GetUrlsAndCheck(subHtml, domainTop, domain, v1, 3)
					//fmt.Println("new_urls==", new_urls)
					if err == nil {
						newUrlLock.Lock()
						newUrls = append(newUrls, new_urls...)
						newUrlLock.Unlock()

					}
				}

				if ok, res := checkIframeHangingHorse(subHtml, v1, domainTop); ok && len(res) > 0 {
					resLock.Lock()
					for k, v := range res {
						checkRes[k] = v
					}
					resLock.Unlock()
				}
				if engineSub.Location != "" && engineSub.Location != "chrome-error://chromewebdata/" {
					if scriptHanging := checkScriptHangingHorse(domainTop, v1, engineSub.Location); len(scriptHanging) > 0 {
						resLock.Lock()
						for k, v := range scriptHanging {
							checkRes[k] = v
						}
						resLock.Unlock()
					}
				}
			}(k1)
		}
		wg.Wait()

		Urls = []string{}
		Urls = append(Urls, newUrls...)

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

	list := []CheckRes{}
	for _, v := range checkRes {
		list = append(list, v)
	}

	value = maps.Map{
		"cost":     time.Since(before).Seconds(),
		"status":   200,
		"scanList": strings.Join(urlRes, `, `),
		"scanNum":  len(urlExistsMap),
		"list":     list,
		"number":   len(list),
	}

	return
}

// 表单信息
func (this *HangingHouseCheckSource) Form() *forms.Form {
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
func (this *HangingHouseCheckSource) Variables() []*SourceVariable {
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
			Code:        "number",
			Description: "挂马数量",
		},
		{
			Code:        "list",
			Description: "挂马内容",
		},
	}
}

// 阈值
func (this *HangingHouseCheckSource) Thresholds() []*Threshold {
	result := []*Threshold{}

	// 阈值
	{
		t := NewThreshold()
		t.Param = "${number}"
		t.Operator = ThresholdOperatorGte
		t.Value = "1"
		t.NoticeLevel = notices.NoticeLevelWarning
		t.NoticeMessage = "匹配到挂马数量：${number}"
		result = append(result, t)
	}

	return result
}

// 图表
func (this *HangingHouseCheckSource) Charts() []*widgets.Chart {
	charts := []*widgets.Chart{}

	{
		// chart
		chart := widgets.NewChart()
		chart.Name = "挂马监测"
		chart.Columns = 2
		chart.Type = "javascript"
		chart.SupportsTimeRange = true
		chart.Options = maps.Map{
			"code": `var chart = new charts.LineChart();

var ones = NewQuery().past(60, time.MINUTE).avg("number");

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
func (this *HangingHouseCheckSource) Presentation() *forms.Presentation {
	return &forms.Presentation{
		HTML: `
			<tr>
				<td class="color-border">URL</td>
				<td>{{source.url}}</td>
			</tr>`,
		CSS: `.darkChainCheck-block-body {
			border: 1px #eee solid;
			padding: 0.4em;
			background: rgba(0, 0, 0, 0.01);
			font-size: 0.9em;
			max-height: 10em;
			overflow-y: auto;
			margin: 0;
		}
		
		.darkChainCheck-block-body::-webkit-scrollbar {
			width: 4px;
		}
		`,
	}
}

func (this *HangingHouseCheckSource) lookupHeader(name string) (value string, ok bool) {
	for _, h := range this.Headers {
		if h.Name == name {
			return h.Value, true
		}
	}
	return "", false
}
