package agents

import (
	"errors"
	"github.com/TeaWeb/build/internal/teaconfigs/forms"
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

var CheckReg = []string{`<div\sid=.*?>.*?</div>\s{0,2}<script>`,
	`<div\s*style=.{0,1}position.*?(?:top|left):\s*-[\d]{3,4}px.*?>.*?</div>`,
	`<MARQUEE\s.*?scrollAmount=.?[\d]{4,5}.?.*?(?:width|height)=.?[0-5].?.*?>.*?</marquee>`,
	`<div\s*style=.?text-indent:\s*-[\d]{2,5}px.?>.*?</div>`,
	`<div\s*style=.*?position:\s*absolute\s*;\s*(?:top|left)\s*:\s*expression\(.*?\).*?>.*?</div>`,
	`<MARQUEE[^>]*?width=["\']?[0-9]?\s+height=["\']?[0-9]["\']?[^>]*?>([\s\S]*?)</MARQUEE>`,
	`<marquee\s+height=[0-9]\s+width=[0-9][^>]*?>([\S\s]*?)</marquee>`,
	`<div\s+style\s*=\s*["\']*\s*overflow\s*:\s*hidden\s*;\s*height\s*:\d\d?px\s*;\s*width\s*:\s*\d\d?.*?>([\S\s]*?)</div>`}

// 暗链监测
type DarkChainCheckSource struct {
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
func NewDarkChainCheckSource() *DarkChainCheckSource {
	return &DarkChainCheckSource{}
}

// 名称
func (this *DarkChainCheckSource) Name() string {
	return "暗链监测"
}

// 代号
func (this *DarkChainCheckSource) Code() string {
	return "darkChainCheck"
}

// 描述
func (this *DarkChainCheckSource) Description() string {
	return "获取URL中是否有暗链"
}

// 执行
func (this *DarkChainCheckSource) Execute(params map[string]string) (value interface{}, err error) {
	if len(this.URL) == 0 {
		err = errors.New("'url' should not be empty")
		return maps.Map{
			"cost":     0,
			"status":   0,
			"list":     make([]CheckRes, 0),
			"scanList": "",
			"scanNum":  0,
			"number":   0,
		}, err
	}
	levelOn := int(0)
	level, err := strconv.Atoi(this.Level)
	if err != nil {
		level = 2
	}

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
	html, err := chromeDpRun(this.URL)
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
	Urls, dark_res, err := GetUrlsAndCheck(html, domainTop, domain, this.URL, 2)
	//监测结果
	checkRes := map[string]CheckRes{}
	if len(dark_res) > 0 {
		//url中可疑暗链
		for k, v := range dark_res {
			checkRes[k] = v
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
		chMax      = make(chan struct{}, 2) //浏览器窗口数
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

				subHtml, err := chromeDpRun(v1)
				if err != nil {
					return
				}
				if level > levelOn { //满足继续下探  才收集下级url地址
					new_urls, dark_res2, err := GetUrlsAndCheck(subHtml, domainTop, domain, v1, 2)
					//fmt.Println("new_urls==", new_urls)
					if err == nil {
						newUrlLock.Lock()
						newUrls = append(newUrls, new_urls...)
						newUrlLock.Unlock()
					}
					if len(dark_res2) > 0 {
						//url中可疑暗链
						resLock.Lock()
						for k, v := range dark_res2 {
							checkRes[k] = v
						}
						resLock.Unlock()
					}
				}
				//script标签和其他标签 进行暗链可疑监测
				if ok, res := checkScriptDarkChain(subHtml, v1, domainTop); ok && len(res) > 0 {
					resLock.Lock()
					for k, v := range res {
						checkRes[k] = v
					}
					resLock.Unlock()
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

//func (this *DarkChainCheckSource) ExecuteOld(params map[string]string) (value interface{}, err error) {
//	if len(this.URL) == 0 {
//		err = errors.New("'url' should not be empty")
//		return maps.Map{
//			"status":     0,
//			"list":       "",
//			"scanList":   "",
//			"scanNum":    0,
//			"keywords":   "",
//			"keywordNum": 0,
//		}, err
//	}
//	level, err := strconv.Atoi(this.Level)
//	if err != nil {
//		level = 2
//	}
//	method := this.Method
//	if len(method) == 0 {
//		method = http.MethodGet
//	}
//
//	var body io.Reader = nil
//
//	before := time.Now()
//	req, err := http.NewRequest(method, this.URL, body)
//	if err != nil {
//		value = maps.Map{
//			"cost":     time.Since(before).Seconds(),
//			"status":   0,
//			"list":     "",
//			"scanList": "",
//			"scanNum":  0,
//			"number":   0,
//		}
//		return value, err
//	}
//
//	client := teautils.SharedHttpClient(time.Duration(5) * time.Second)
//	resp, err := client.Do(req)
//	if err != nil {
//		return maps.Map{
//			"cost":     time.Since(before).Seconds(),
//			"status":   0,
//			"list":     "",
//			"scanList": "",
//			"scanNum":  0,
//			"number":   0,
//		}, err
//	}
//	defer func() {
//		_ = resp.Body.Close()
//	}()
//
//	data, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		return maps.Map{
//			"cost":     time.Since(before).Seconds(),
//			"status":   0,
//			"list":     "",
//			"scanList": "",
//			"scanNum":  0,
//			"number":   0,
//		}, err
//	}
//	//获取网页内的url
//	hitUrls := this.MatchUrl(data)
//	urlMap := map[string]struct{}{}
//	//匹配第一级网页 暗链数
//	hitList := []string{}
//	hitList = this.MatchDark(data)
//
//	if len(hitUrls) > 0 {
//		var (
//			urlLock = &sync.Mutex{}
//			keyLock = &sync.Mutex{}
//			wg      = &sync.WaitGroup{}
//			chMax   = make(chan struct{}, 20)
//		)
//		//下探一级
//		for _, v1 := range hitUrls {
//			chMax <- struct{}{}
//			wg.Add(1)
//			go func(v1 string) {
//				defer func() {
//					wg.Done()
//					<-chMax
//				}()
//				urlLock.Lock()
//				if _, ok := urlMap[v1]; ok {
//					urlLock.Unlock()
//					//已存在
//					return
//				} else {
//					urlMap[v1] = struct{}{}
//				}
//				urlLock.Unlock()
//				resp1, err := this.GetQuery(v1)
//				if err != nil {
//					return
//				}
//				data1, err := ioutil.ReadAll(resp1.Body)
//				if err != nil {
//					return
//				}
//				//匹配暗链
//				hitList1 := this.MatchDark(data1)
//				if len(hitList1) > 0 {
//					keyLock.Lock()
//					hitList = append(hitList, hitList1...)
//					keyLock.Unlock()
//				}
//				//下探2级
//				if level >= 2 {
//					hitUrls1 := this.MatchUrl(data1)
//					if len(hitUrls1) > 0 {
//						//下探二级
//						for _, v2 := range hitUrls1 {
//							urlLock.Lock()
//							if _, ok := urlMap[v2]; ok {
//								//已存在
//								urlLock.Unlock()
//								continue
//							} else {
//								urlMap[v2] = struct{}{}
//							}
//							urlLock.Unlock()
//							resp2, err := this.GetQuery(v2)
//							if err != nil {
//								continue
//							}
//							data2, err := ioutil.ReadAll(resp2.Body)
//							if err != nil {
//								continue
//							}
//							//匹配暗链
//							hitList2 := this.MatchDark(data2)
//							if len(hitList2) > 0 {
//								keyLock.Lock()
//								hitList = append(hitList, hitList2...)
//								keyLock.Unlock()
//							}
//							//下探3级
//							if level >= 3 {
//								hitUrls2 := this.MatchUrl(data2)
//								if len(hitUrls2) > 0 {
//									//下探三级
//									urlLock.Lock()
//									for _, v3 := range hitUrls2 {
//										if _, ok := urlMap[v3]; ok {
//											//已存在
//											urlLock.Unlock()
//											continue
//										} else {
//											urlMap[v3] = struct{}{}
//										}
//										urlLock.Unlock()
//										resp3, err := this.GetQuery(v3)
//										if err != nil {
//											continue
//										}
//										data3, err := ioutil.ReadAll(resp3.Body)
//										if err != nil {
//											continue
//										}
//										//匹配暗链
//										hitList3 := this.MatchDark(data3)
//										if len(hitList3) > 0 {
//											keyLock.Lock()
//											hitList = append(hitList, hitList3...)
//											keyLock.Unlock()
//										}
//									}
//								}
//							}
//						}
//					}
//				}
//			}(v1)
//		}
//		wg.Wait()
//	}
//	//
//	urlRes := []string{}
//	for k, _ := range urlMap {
//		//取20个 地址
//		if len(urlRes) > 20 {
//			break
//		}
//		urlRes = append(urlRes, k)
//	}
//
//	hitRes := []string{}
//	for _, v := range hitList {
//		//取20个 地址
//		if len(hitList) > 20 {
//			break
//		}
//		hitRes = append(hitRes, v)
//	}
//
//	value = maps.Map{
//		"cost":     time.Since(before).Seconds(),
//		"status":   resp.StatusCode,
//		"scanList": strings.Join(urlRes, `, `),
//		"scanNum":  len(urlMap),
//		"list":     strings.Join(hitRes, ","),
//		"number":   len(hitList),
//	}
//
//	return
//
//}

// get请求
//func (this *DarkChainCheckSource) GetQuery(url string) (resp *http.Response, err error) {
//	req, err := http.NewRequest(http.MethodGet, url, nil)
//	if err != nil {
//		return nil, err
//	}
//	client := teautils.SharedHttpClient(time.Duration(5) * time.Second)
//	resp, err = client.Do(req)
//	if err != nil {
//		return nil, err
//	}
//	defer func() {
//		_ = resp.Body.Close()
//	}()
//
//	return resp, err
//}

//查找网页源码内的URL地址
//func (this *DarkChainCheckSource) MatchUrl(s []byte) (urls []string) {
//	// href=('|").*?('|")   [a-zA-z]+://[^\s]*("|')
//	urls = []string{}
//	urlMap := map[string]struct{}{}
//	//先匹配 http(s):// 这类地址
//	re, err := regexp.Compile(`[a-zA-z]+://[^\s]+\.[^\s]+('|")`)
//	if err != nil {
//		return urls
//	}
//	list := re.FindAll(s, -1)
//	if len(list) > 0 {
//		for _, v := range list {
//			url := string(v)
//			urlMap[url] = struct{}{}
//		}
//	}
//	//在匹配 href="" 这类地址
//	re, err = regexp.Compile(`href=('|").*?('|")`)
//	if err != nil {
//		return urls
//	}
//	list = re.FindAll(s, -1)
//	if len(list) > 0 {
//		for _, v := range list {
//			url := string(v)
//			urlMap[url] = struct{}{}
//		}
//	}
//	re1, _ := regexp.Compile(`^href=("|')|('|"|\))[^\s]*?$`)
//	re2, _ := regexp.Compile(`^http`)
//	re3, _ := regexp.Compile(`(http|https)://(www.)?(\w+(\.)?)+`) //获取域名正则
//	re4, _ := regexp.Compile(`\\u0026amp\;`)                      // &符号
//	re5, _ := regexp.Compile(`\\u0026`)                           // &符号
//	domain := re3.Find(s)
//	for v := range urlMap {
//		//正则替换
//		url := re1.ReplaceAllString(v, "")
//		url = re4.ReplaceAllString(url, "&")
//		url = re5.ReplaceAllString(url, "&")
//		if url == "" {
//			continue
//		}
//		if url != "/" && url != "#" && url != "?" {
//			if re2.MatchString(url) { //是包含http的地址
//				urls = append(urls, url)
//			} else {
//				urls = append(urls, string(domain)+url)
//			}
//
//		}
//
//	}
//	return urls
//}

func (this *DarkChainCheckSource) MatchDark(s []byte) (list []string) {
	list = []string{}
	for _, reg_rule := range CheckReg {
		reg := regexp.MustCompile(reg_rule)
		res := reg.FindAll(s, -1)
		if len(res) > 0 {
			for _, v := range res {
				list = append(list, string(v))
			}
		}
		//if reg.Match(s) {
		//	continue
		//}
	}
	return list
}

// 表单信息
func (this *DarkChainCheckSource) Form() *forms.Form {
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
func (this *DarkChainCheckSource) Variables() []*SourceVariable {
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
		{
			Code:        "number",
			Description: "暗链数量",
		},
		{
			Code:        "list",
			Description: "暗链列表",
		},
	}
}

// 阈值
func (this *DarkChainCheckSource) Thresholds() []*Threshold {
	result := []*Threshold{}

	// 阈值
	{
		t := NewThreshold()
		t.Param = "${number}"
		t.Operator = ThresholdOperatorGte
		t.Value = "1"
		t.NoticeLevel = notices.NoticeLevelWarning
		t.NoticeMessage = "匹配到暗链数量：${number}"
		result = append(result, t)
	}

	return result
}

// 图表
func (this *DarkChainCheckSource) Charts() []*widgets.Chart {
	charts := []*widgets.Chart{}

	{
		// chart
		chart := widgets.NewChart()
		chart.Name = "暗链监测"
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
	line.addValue(v.value.number * 1000);
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
func (this *DarkChainCheckSource) Presentation() *forms.Presentation {
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

func (this *DarkChainCheckSource) lookupHeader(name string) (value string, ok bool) {
	for _, h := range this.Headers {
		if h.Name == name {
			return h.Value, true
		}
	}
	return "", false
}
