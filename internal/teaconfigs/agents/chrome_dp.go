package agents

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

var timeOutError = errors.New("获取指定path的的页面元素超时")

var documentReferrerRex *regexp.Regexp
var indexOfRex *regexp.Regexp
var locationHrefRex *regexp.Regexp
var evalRex *regexp.Regexp
var unicodeRex *regexp.Regexp
var baseRex *regexp.Regexp
var displayNoneRex *regexp.Regexp
var positionAbsoluteRex *regexp.Regexp
var positionAbsoluteTopRex *regexp.Regexp
var positionAbsoluteLeftRex *regexp.Regexp
var positionAbsoluteBottomRex *regexp.Regexp
var positionAbsoluteRightRex *regexp.Regexp
var whiteColorRex *regexp.Regexp
var fontSize0Rex *regexp.Regexp
var marqueeHeightRex *regexp.Regexp
var marqueeWidthRex *regexp.Regexp

func init() {
	documentReferrerRex, _ = regexp.Compile(`document\.referrer`)                                                                      //特殊关键词
	indexOfRex, _ = regexp.Compile(`\.indexOf\(`)                                                                                      //特殊关键词
	locationHrefRex, _ = regexp.Compile(`(window\.l|l|self\.l|this\.l)ocation\.href`)                                                  //特殊关键词
	evalRex, _ = regexp.Compile(`eval\(`)                                                                                              //js压缩标识
	unicodeRex, _ = regexp.Compile(`\&\#\d{1,};`)                                                                                      //unicode标识
	baseRex, _ = regexp.Compile(`(\\u|\\x|\|u|\|x)\d{1,}`)                                                                             //十进制 16进制标识
	displayNoneRex, _ = regexp.Compile(`display\:\s{0,}none`)                                                                          //隐藏标识
	positionAbsoluteRex, _ = regexp.Compile(`position\:\s{0,}absolute`)                                                                //position隐藏标识
	positionAbsoluteTopRex, _ = regexp.Compile(`top\:`)                                                                                //position隐藏标识
	positionAbsoluteLeftRex, _ = regexp.Compile(`left\:`)                                                                              //position隐藏标识
	positionAbsoluteBottomRex, _ = regexp.Compile(`bottom\:`)                                                                          //position隐藏标识
	positionAbsoluteRightRex, _ = regexp.Compile(`right\:`)                                                                            //position隐藏标识
	whiteColorRex, _ = regexp.Compile(`color\:\s{0,}(\#ffffff|white|rgb\(255\,\s{0,}255\,\s{0,}255\)|hsl\(0\,\s{0,}0\%\,\s{0,}100\%)`) //白色字体标识
	fontSize0Rex, _ = regexp.Compile(`font-size\:\s{1,}0`)                                                                             //字体大小是0  标识
	marqueeHeightRex, _ = regexp.Compile(`height="\d{1}"`)                                                                             //marquee标签高很小  标识
	marqueeWidthRex, _ = regexp.Compile(`width="\d{1}"`)                                                                               //marquee标签宽很小  标识
}

type (
	ChromeDpEngine struct {
		Context context.Context `json:"context"` //chromedp上下文信息
		Url     string          `json:"url"`     //请求的第一个地址
		Html    *string         `json:"html"`    //拿到的响应内容
		//WithTargetID string          `json:"with_target_id"` //无头浏览器ID
		DoMainTop  string   `json:"do_main_top"` //顶级域名
		DoMain     string   `json:"do_main"`     //域名
		OnLevel    int      `json:"on_level"`    //当前等级 默认0 首页
		Level      int      `json:"level"`       //下探等级 最大3
		UseRequest sync.Map `json:"use_request"` //已经请求过的地址
		Urls       sync.Map `json:"urls"`        //需要请求的地址列表
		CheckType  int      `json:"check_type"`  // 1敏感词 2暗链 3挂马
	}
	CheckRes struct {
		Url   string `json:"url"`   //页面地址
		Value string `json:"value"` //监测到的内容
	}
)

func run() {
	url := "http://127.0.0.1"
	domainTop, domain := GetDomain(url)
	fmt.Println(domainTop, domain)
	engine := &ChromeDpEngine{
		Context:   newChromeDpCtx(),
		Url:       url,
		DoMainTop: domainTop,
		DoMain:    domain,
		Html:      new(string),
		OnLevel:   0, Level: 1,
		UseRequest: sync.Map{},
		Urls:       sync.Map{},
	}
	//defer func() {
	//	if err := chromedp.Cancel(engine.Context); err != nil {
	//		log.Println(err)
	//	}
	//}()

	/** 调试时可以加上，避免主动关闭进程但是浏览器还在执行
	go func() {
		quit := make(chan os.Signal)
		signal.Notify(quit, os.Interrupt, os.Kill)
		<-quit
		_ = chromedp.Cancel(engine.Context)
		os.Exit(1)
	}()
	*/

	if err := chromedp.Run(engine.Context, engine.newTask()); err != nil {
		log.Println("执行失败：", err)
		return
	}

	//fmt.Println(engine.Urls)
	//fmt.Println(*engine.Html)
}

//获得一个chromdp的context
func newChromeDpCtx() context.Context {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false), // debug使用false  正式使用用true
		chromedp.WindowSize(1280, 1024),  // 调整浏览器大小
	}
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)
	//options = append(options, chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.82 Safari/537.36"))
	options = append(options, chromedp.DisableGPU)
	//options = append(options, chromedp.Flag("ignore-certificate-errors", true))//忽略错误
	options = append(options, chromedp.Flag("blink-settings", "imagesEnabled=false")) //不加载图片

	ctx, _ := chromedp.NewExecAllocator(context.Background(), options...)
	ctx, _ = chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf)) // 会打开浏览器并且新建一个标签页进行操作

	ctx, _ = chromedp.NewRemoteAllocator(ctx, "ws://127.0.0.1:9221") //使用远程调试，可以结合下面的容器使用
	//ctx, _ = chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf), chromedp.WithTargetID("EA3271486ADC09ED0504F3C9FCEE698B")) // WithTargetID可以指定一个标签页进行操作
	ctx, _ = chromedp.NewContext(ctx) // 新开WithTargetID

	return ctx
}

//返回一个任务的列队来执行
func (this *ChromeDpEngine) newTask() chromedp.Tasks {
	return chromedp.Tasks{
		this.toUrl("打开首页", this.Url),
		this.getHtml("获取页面元素", this.Html),

		//this.setValue("填写用户名", "//*[@id=\"app\"]/div/div/div[2]/form/div[1]/input", "******"),
		//this.setValue("填写密码", "//*[@id=\"app\"]/div/div/div[2]/form/div[2]/input", "******"),
		//this.click("点击登录按钮", "//*[@id=\"app\"]/div/div/div[2]/form/button"),
		//this.toUrl("跳转至页面", "http://***.***.***.***:*****/#/project/1/dashboard/579"),
		//this.chromedp.Sleep(2 * time.Second),
		//this.screenShot("指定div截图","//*[@id=\"app\"]/div/div/div/div[2]/div/div/div/div/div[2]/div[1]/div"),
	}
}

func (this *ChromeDpEngine) setValue(name, path string, value string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		defer this.handleActionError(name, &err)
		timeout, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if chromedp.WaitVisible(path).Do(timeout) != nil {
			return timeOutError
		}
		return chromedp.SetValue(path, value).Do(timeout)
	}
}

func (this *ChromeDpEngine) click(name, path string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		defer this.handleActionError(name, &err)
		timeout, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if chromedp.WaitVisible(path).Do(timeout) != nil {
			return timeOutError
		}
		return chromedp.Click(path).Do(timeout)
	}
}

func (this *ChromeDpEngine) toUrl(name, url string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		defer this.handleActionError(name, &err)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		//chromedp.Sleep(1 * time.Second).Do(ctx)
		return chromedp.Navigate(url).Do(ctx)
	}
}

//获取网页内的a标签的 url地址
func (this *ChromeDpEngine) getHtml(name string, html *string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		defer this.handleActionError(name, &err)
		timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		//if chromedp.WaitVisible("body").Do(timeout) != nil {
		//	return timeOutError
		//}
		err = chromedp.OuterHTML("html", html, chromedp.ByQuery).Do(timeout)
		//解析网页内的地址
		this.GetUrlsAndCheck()

		return err
	}
}
func (this *ChromeDpEngine) screenShot(name string, path string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		defer this.handleActionError(name, &err)
		timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if chromedp.WaitVisible(path).Do(timeout) != nil {
			return timeOutError
		}
		var code []byte
		if err = chromedp.Screenshot(path, &code).Do(timeout); err != nil {
			return
		}
		return ioutil.WriteFile("shot.png", code, 0755)
	}
}

//同域名的子url
func (this *ChromeDpEngine) GetUrlsAndCheck() ([]string, error) {
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(*this.Html))
	if err != nil {

		return []string{}, err
	}

	var str []string
	dom.Find("a").Each(func(i int, selection *goquery.Selection) {
		if url, ok := selection.Attr("href"); ok {
			fmt.Println(url)
			//检测字符串是否以指定的前缀开头。
			if strings.HasPrefix(url, "//") {
				url = "http:" + url
			}
			//删除左边的前缀 .. 或者 .
			url = strings.TrimPrefix(url, "..")
			url = strings.TrimPrefix(url, ".")
			if strings.HasPrefix(url, "/") {
				url = this.DoMain + url
			}
			//判断当前地址是否来着当前域名
			if this.checkUrlDomain(url) {
				str = append(str, url)
				this.Urls.Store(url, struct{}{})
			} else {
				if this.CheckType == 2 { //暗链监测

				}

			}

		}
	})
	//遍历所有script标签 ，通过特征 判断是否是暗链
	dom.Find("script").Each(func(i int, selection *goquery.Selection) {
		fmt.Println(selection.Text())
		if url, ok := selection.Attr("src"); ok {
			fmt.Println("script src==", url)
		}
	})

	//遍历所有marquee标签 ，通过特征 判断是否是暗链
	dom.Find("marquee").Each(func(i int, selection *goquery.Selection) {
		fmt.Println(selection.Attr("width"))
		aUrl := ""
		selection.Find("a").Each(func(i int, selectionSub *goquery.Selection) {
			ok := false
			aUrl, ok = selectionSub.Attr("href")
			if ok {
				return
			}
		})
		if aUrl != "" { //marquee标签内有a标签 判断marquee标签宽高是否可疑
			width, widthExists := selection.Attr("width")
			height, heightExists := selection.Attr("height")
			if widthExists && heightExists {
				widthNum, _ := strconv.Atoi(width)
				heightNum, _ := strconv.Atoi(height)
				if widthNum <= 10 && heightNum <= 10 { //宽高都小于10
					//可疑暗链
					//todo
				}
			}
		}
	})

	//遍历所有meta标签 ，通过特征 判断是否是暗链
	dom.Find("meta").Each(func(i int, selection *goquery.Selection) {

		aUrl := ""
		ok := false
		aUrl, ok = selection.Attr("href")
		if ok {
			return
		}
		if aUrl != "" { //mate标签内有url 判断mate是有有 http-equiv属性
			equiv, equivExists := selection.Attr("http-equiv")

			if equivExists && equiv == "refresh" {
				if !this.checkUrlDomain(aUrl) { //地址不是当前域名
					//可疑暗链
					//todo
				}

			}
		}
	})

	return str, nil
}

//检查地址的域名是否是同域名  非相同域名不处理
func (this *ChromeDpEngine) checkUrlDomain(url string) (ok bool) {

	return strings.Contains(strings.Split(url, "?")[0], this.DoMainTop)
}

//检查url地址是否有暗链特征
func (this *ChromeDpEngine) checkUrlDarkChain(selection *goquery.Selection) (ok bool) {

	//非当前域名url  检测是否是暗链  ，通过当前元素或父级元素的样式 判断是否有可以属性
	content, styleExists := selection.Attr("style")
	parentContent, parentStyleExists := selection.Attr("style")

	if styleExists || parentStyleExists {
		if (documentReferrerRex.MatchString(content) && indexOfRex.MatchString(content) && locationHrefRex.MatchString(content)) ||
			(documentReferrerRex.MatchString(parentContent) && indexOfRex.MatchString(parentContent) && locationHrefRex.MatchString(parentContent)) {
			//todo 暗链=url
			fmt.Println("document.referrer true")
		}
		if evalRex.MatchString(content) || evalRex.MatchString(parentContent) {
			//todo 暗链=url
			fmt.Println("eval true")
		}
		if unicodeRex.MatchString(content) || unicodeRex.MatchString(parentContent) {
			//todo 暗链=url
			fmt.Println("unicode true")
		}
		if baseRex.MatchString(content) || baseRex.MatchString(parentContent) {
			//todo 暗链=url
			fmt.Println("bash true")
		}
		if displayNoneRex.MatchString(content) || displayNoneRex.MatchString(parentContent) {
			//todo 暗链=url
			fmt.Println("displayNoneRex true")
		}
		if (positionAbsoluteRex.MatchString(content) && (positionAbsoluteTopRex.MatchString(content) || positionAbsoluteBottomRex.MatchString(content) || positionAbsoluteRightRex.MatchString(content) || positionAbsoluteLeftRex.MatchString(content))) ||
			(positionAbsoluteRex.MatchString(parentContent) && (positionAbsoluteTopRex.MatchString(parentContent) || positionAbsoluteBottomRex.MatchString(parentContent) || positionAbsoluteRightRex.MatchString(parentContent) || positionAbsoluteLeftRex.MatchString(parentContent))) {
			//todo 暗链=url
			fmt.Println("positionAbsoluteRex true")
		}
		if whiteColorRex.MatchString(content) || whiteColorRex.MatchString(parentContent) {
			//todo 暗链=url
			fmt.Println("whiteColorRex true")
		}
		if fontSize0Rex.MatchString(content) || fontSize0Rex.MatchString(parentContent) {
			//todo 暗链=url
			fmt.Println("fontSize0Rex true")
		}
	}
	return
}

//检查script内容是否有暗链特征
func (this *ChromeDpEngine) checkScriptDarkChain(context string) (ok bool) {

	return
}
func (this *ChromeDpEngine) handleActionError(name string, err *error) {
	if *err != nil {
		*err = fmt.Errorf("【%s】失败=>%w", name, *err)
	}
}

//通过url地址 获取顶级域名和当前域名
func GetDomain(url string) (domainTop, domain string) {
	resoureUrl := url
	url = strings.Split(url, "?")[0]
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "//")
	url = strings.Split(url, "/")[0]
	//是ip
	if matched, err := regexp.MatchString(`^\d{1,}\.\d{1,}\.\d{1,}\.\d{1,}`, url); matched && err == nil {
		return url, strings.Split(resoureUrl, url)[0] + url
	}
	var compoundSuffix, interceptLen = []string{
		".com.cn", ".net.cn", ".org.cn", ".gov.cn", ".edu.cn",
	}, 2 //复合后缀和截取长度
	mainAddr := strings.Split(url, ":")[0] //去掉端口的地址
	for _, v := range compoundSuffix {
		if strings.HasSuffix(mainAddr, v) {
			interceptLen++
			break
		}
	}
	s := strings.Split(url, ".")
	if len(s) <= interceptLen {
		domainTop = strings.Join(s, ".")
		return domainTop, strings.Split(resoureUrl, domainTop)[0] + domainTop
	}
	domainTop = strings.Join(s[len(s)-interceptLen:], ".")
	return domainTop, strings.Split(resoureUrl, domainTop)[0] + domainTop
}
