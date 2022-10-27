package agents

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"runtime"
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

var MaxWind = runtime.NumCPU() * 2
var WindNum = int(0) //已经打开的窗口数
var TargeLock = &sync.Mutex{}
var CTX context.Context

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

	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true), // debug使用false  正式使用用true
		chromedp.WindowSize(1280, 1024), // 调整浏览器大小
	}
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)
	options = append(options, chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.82 Safari/537.36"))
	options = append(options, chromedp.DisableGPU)
	options = append(options, chromedp.Flag("ignore-certificate-errors", true))       //忽略错误
	options = append(options, chromedp.Flag("blink-settings", "imagesEnabled=false")) //不加载图片
	options = append(options, chromedp.Flag("disable-web-security", true))            //禁用安全标识
	//var cancel context.CancelFunc
	CTX, _ = chromedp.NewExecAllocator(context.Background(), options...)
	CTX, _ = chromedp.NewRemoteAllocator(CTX, "ws://127.0.0.1:9222") //使用远程调试，可以结合下面的容器使用

	//窗口自检 定时关闭因为打开网页失败而导致无法关闭的网页
	if CTX != nil {
		go func() {
			winMap := map[string]string{}
			n := 0
			for {
				<-time.Tick(time.Second * 100)
				winCtx := CTX
				winCtx, _ = chromedp.NewContext(winCtx)
				targets, err := chromedp.Targets(winCtx)
				if err != nil {
					CTX, _ = chromedp.NewRemoteAllocator(CTX, "ws://127.0.0.1:9222") //使用远程调试，可以结合下面的容器使用
					//fmt.Println("获取打开的窗口失败", err)
					continue
				}
				for _, v := range targets {
					//fmt.Println(*v)
					if v.Type == "page" { //不是iframe标签
						if value, ok := winMap[string(v.TargetID)]; ok && value == Md5Str(v.URL+v.Title) {
							//页面tital和url地址没有变化，关闭此窗口
							free, _ := chromedp.NewContext(winCtx, chromedp.WithTargetID(v.TargetID))
							chromedp.Run(free, chromedp.Navigate(`chrome://newtab/`))
							chromedp.Cancel(free)
							time.Sleep(time.Second * 1)
							delete(winMap, string(v.TargetID))
						} else {
							winMap[string(v.TargetID)] = Md5Str(v.URL + v.Title)
						}

					}

				}
				chromedp.Cancel(winCtx)
				if n > 0 && n/10 == 0 { //10次循环清空一次map
					winMap = map[string]string{}
					n = 0
				}
			}
		}()

	}
}

type (
	ChromeDpEngine struct {
		Context  context.Context `json:"context"`  //chromedp上下文信息
		Url      string          `json:"url"`      //请求的第一个地址
		Html     []*string       `json:"html"`     //拿到的响应内容
		Iframe   bool            `json:"iframe"`   //专门获取iframe标签的内容的窗口
		Location string          `json:"location"` //请求地址后，响应页面的地址（js跳转后的地址）
	}
	CheckRes struct {
		Url    string `json:"url"`              //页面地址
		Value  string `json:"value"`            //监测到的内容
		Cate   string `json:"cate,omitempty"`   //敏感词分类
		Number int    `json:"number,omitempty"` //敏感词数
	}
	Cates struct {
		Name  string `json:"name"`  //分类
		Value int    `json:"value"` //数量
	}
)

//检查是否有9222端口，来判断是否运行在linux上
func checkChromePort() bool {
	addr := net.JoinHostPort("127.0.0.1", "9222")
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

//ctx 可复用
func chromeDpRun(url string, ctx context.Context) (engine *ChromeDpEngine, html []*string, err error) {
	//url := "http://127.0.0.1"
	engine = &ChromeDpEngine{
		//Context: newChromeDpCtx(),
		Url:  url,
		Html: make([]*string, 0),
	}
	if ctx != nil {
		engine.Context = ctx
	} else {
		engine.Context, err = newChromeDpCtx()
		if err != nil {
			for i := 0; i < 60; i++ { //重试60次，每次等待一分钟
				//fmt.Println("等待的url", url, err, time.Now())
				<-time.Tick(time.Second * 10)
				engine.Context, err = newChromeDpCtx()
				if err == nil && engine.Context != nil {
					break
				}
			}
		}
		if engine.Context == nil {
			return engine, make([]*string, 0), errors.New("暂无空闲窗口")
		}
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
	//fmt.Println("run")
	if err := chromedp.Run(engine.Context, engine.newTask()); err != nil {
		log.Println("执行失败：", err)
		return engine, make([]*string, 0), err
	}

	return engine, engine.Html, nil
	//fmt.Println(engine.Urls)
	//fmt.Println(*engine.Html)
}
func (this *ChromeDpEngine) Close() {
	chromedp.Cancel(this.Context)
}
func (this *ChromeDpEngine) UnLockTargetId() {
	//fmt.Println(this.Url,this.Context,"解除占用")
	if this.Context != nil {
		TargeLock.Lock()
		defer TargeLock.Unlock()
		WindNum -= 1
		chromedp.Cancel(this.Context)
	}

}

//获得一个chromdp的context
func newChromeDpCtx() (ctx context.Context, err error) {
	TargeLock.Lock()
	defer TargeLock.Unlock()

	//options := []chromedp.ExecAllocatorOption{
	//	chromedp.Flag("headless", false), // debug使用false  正式使用用true
	//	chromedp.WindowSize(1280, 1024),  // 调整浏览器大小
	//}
	//options = append(chromedp.DefaultExecAllocatorOptions[:], options...)
	//options = append(options, chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.82 Safari/537.36"))
	//options = append(options, chromedp.DisableGPU)
	//options = append(options, chromedp.Flag("ignore-certificate-errors", true))       //忽略错误
	//options = append(options, chromedp.Flag("blink-settings", "imagesEnabled=false")) //不加载图片
	////var cancel context.CancelFunc
	//ctx, _ = chromedp.NewExecAllocator(context.Background(), options...)
	//ctx, _ = chromedp.NewRemoteAllocator(ctx, "ws://127.0.0.1:9222") //使用远程调试，可以结合下面的容器使用

	if WindNum >= MaxWind {
		return nil, errors.New("暂无空闲窗口")
	}
	WindNum += 1
	ctx, _ = chromedp.NewContext(CTX, chromedp.WithLogf(log.Printf)) // 会打开浏览器并且新建一个标签页进行操作
	//defer cancel()

	//ctx, _ = chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf), chromedp.WithTargetID("EA3271486ADC09ED0504F3C9FCEE698B")) // WithTargetID可以指定一个标签页进行操作
	//ctx, _ = chromedp.NewContext(ctx) // 新开WithTargetID

	return ctx, nil

}

//返回一个任务的列队来执行
func (this *ChromeDpEngine) newTask() chromedp.Tasks {
	return chromedp.Tasks{
		this.toUrl("打开首页", this.Url),
		//chromedp.Sleep(time.Second * 10),
		//chromedp.WaitReady(`iframe`, chromedp.ByQuery),
		this.getHtml("获取页面元素"),

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
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err = chromedp.Navigate(url).Do(ctx)
		if err != nil {
			fmt.Println(name, " err=========", err)
			return nil
		}

		chromedp.Sleep(1 * time.Second).Do(ctx)
		return nil
	}
}

//获取网页内的a标签的 url地址
func (this *ChromeDpEngine) getHtml(name string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		defer this.handleActionError(name, &err)
		timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		//if !this.Iframe {
		//	iframeHtml := ""
		//	//chromedp.EvaluateAsDevTools(`document.getElementById('iframe').contentWindow.document.body.outerHTML;`, &iframeHtml).Do(timeout)
		//	chromedp.OuterHTML(`document.querySelectorAll("iframe")[0]`, &iframeHtml, chromedp.ByJSPath).Do(timeout)
		//
		//	fmt.Println("iframeHtml===================================================================")
		//	fmt.Println(string(iframeHtml))
		//}
		html := ""
		if !this.Iframe {
			_, domain := GetDomain(this.Url)
			var iframes []*cdp.Node
			timeout1, _ := context.WithTimeout(ctx, 100*time.Millisecond)
			iframeErr := chromedp.Nodes("iframe", &iframes, chromedp.ByQuery).Do(timeout1)
			if iframeErr == nil && len(iframes) > 0 {
				ifHtmls := make([]*string, 0)
				iframeWin, _ := chromedp.NewContext(CTX, chromedp.WithLogf(log.Printf))
				defer chromedp.Cancel(iframeWin)
				for _, v := range iframes {
					if src, ok := v.Attribute("src"); ok {
						src = strings.TrimPrefix(src, " ")
						ifUrl := GetIframeUrl(src, domain)
						//fmt.Println(src)
						//fmt.Println(ifUrl)

						ifEngine := &ChromeDpEngine{
							Context: iframeWin,
							Url:     ifUrl,
							Html:    ifHtmls,
							Iframe:  true,
						}

						if err := chromedp.Run(ifEngine.Context, ifEngine.newTask()); err != nil {
							//log.Println("获取iframe，执行失败：", err)
							continue
						} else {
							//fmt.Println("ifhtml====")
							//fmt.Println(*ifEngine.Html[0])
							this.Html = append(this.Html, ifEngine.Html...)
						}

					}
				}
			}
			//chromedp.Sleep(time.Second)
			location := ""
			err = chromedp.EvaluateAsDevTools("document.location.href;", &location).Do(timeout)
			//fmt.Println("location err=", err)
			this.Location = location
		}
		//fmt.Println("outerhtml")
		err = chromedp.OuterHTML("html", &html, chromedp.ByQuery).Do(timeout)
		this.Html = append(this.Html, &html)
		//fmt.Println("html====")
		//fmt.Println(html)

		return err
	}
}
func (this *ChromeDpEngine) handleActionError(name string, err *error) {
	if *err != nil {
		*err = fmt.Errorf("【%s】失败=>%w", name, *err)
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

//获取iframe地址
func GetIframeUrl(url, domain string) string {
	//检测字符串是否以指定的前缀开头。
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return url
	}
	//删除左边的前缀 .. 或者 .
	url = strings.TrimPrefix(url, "..")
	url = strings.TrimPrefix(url, ".")
	if strings.HasPrefix(url, "/") {
		return domain + url
	}
	return domain + "/" + url
}

//同域名的子url 如果是暗链，则需要判断url是否是暗链  checkType 1敏感词 2暗链  3挂马
func GetUrlsAndCheck(html []*string, doMainTop, doMain, pageUrl string, checkType int) (urls []string, dark_chain map[string]CheckRes, err error) {
	urls, dark_chain = []string{}, make(map[string]CheckRes, 0)
	if len(html) > 0 {
		for _, v := range html {
			dom, err := goquery.NewDocumentFromReader(strings.NewReader(*v))
			if err != nil {
				return urls, dark_chain, err
			}

			dom.Find("a").Each(func(i int, selection *goquery.Selection) {
				if url, ok := selection.Attr("href"); ok {
					url = strings.TrimPrefix(url, " ")

					//fmt.Println(url)
					//检测字符串是否以指定的前缀开头。
					if strings.HasPrefix(url, "//") {
						url = "http:" + url
					}
					//删除左边的前缀 .. 或者 .
					url = strings.TrimPrefix(url, "..")
					url = strings.TrimPrefix(url, ".")
					if strings.HasPrefix(url, "/") {
						url = doMain + url
					}
					//判断当前地址是否来着当前域名
					if checkUrlDomain(url, doMainTop) {
						urls = append(urls, url)

					} else {
						//暗链监测a标签
						if checkType == 2 && checkUrlDarkChain(selection) {
							//pageUrl页面地址  url=a标签的地址
							dark_chain[Md5Str(pageUrl+url)] = CheckRes{
								Url:   pageUrl,
								Value: url,
							}
						}
					}

				}
			})
		}
	}

	return urls, dark_chain, nil
}

//url去重 并转换成map
func duplicateRemovalUrl(urls []string, urlMap map[string]struct{}) map[string]struct{} {
	if len(urlMap) == 0 {
		urlMap = make(map[string]struct{}, 0)
	}
	for _, v := range urls {
		urlMap[v] = struct{}{}
	}
	return urlMap
}

//检查地址的域名是否是同域名  非相同域名不处理
func checkUrlDomain(url, doMainTop string) (ok bool) {

	return strings.Contains(strings.Split(url, "?")[0], doMainTop)
}

//检查url地址是否有暗链特征
func checkUrlDarkChain(selection *goquery.Selection) (ok bool) {

	//非当前域名url  检测是否是暗链  ，通过当前元素或父级元素的样式 判断是否有可以属性
	content, styleExists := selection.Attr("style")
	parentContent, parentStyleExists := selection.Parent().Attr("style")

	if styleExists || parentStyleExists {

		if displayNoneRex.MatchString(content) || displayNoneRex.MatchString(parentContent) {
			return true
			//todo 暗链=url
			fmt.Println("displayNoneRex true")
		}
		if (positionAbsoluteRex.MatchString(content) || (positionAbsoluteTopRex.MatchString(content) || positionAbsoluteBottomRex.MatchString(content) || positionAbsoluteRightRex.MatchString(content) || positionAbsoluteLeftRex.MatchString(content))) ||
			(positionAbsoluteRex.MatchString(parentContent) || (positionAbsoluteTopRex.MatchString(parentContent) || positionAbsoluteBottomRex.MatchString(parentContent) || positionAbsoluteRightRex.MatchString(parentContent) || positionAbsoluteLeftRex.MatchString(parentContent))) {
			return true
			//todo 暗链=url
			fmt.Println("positionAbsoluteRex true")
		}
		//if whiteColorRex.MatchString(content) || whiteColorRex.MatchString(parentContent) {
		//	return true
		//	//todo 暗链=url
		//	fmt.Println("whiteColorRex true")
		//}
		if fontSize0Rex.MatchString(content) || fontSize0Rex.MatchString(parentContent) {
			return true
			//todo 暗链=url
			fmt.Println("fontSize0Rex true")
		}
	}
	return false
}

//检查script内容是否有暗链特征
func checkScriptDarkChain(html []*string, pageUrl, doMainTop string) (ok bool, dark_chain map[string]CheckRes) {
	dark_chain = make(map[string]CheckRes, 0)
	if len(html) > 0 {
		for _, v := range html {
			dom, err := goquery.NewDocumentFromReader(strings.NewReader(*v))
			if err != nil {
				return false, dark_chain
			}
			//遍历所有script标签 ，通过特征 判断是否是暗链
			dom.Find("script").Each(func(i int, selection *goquery.Selection) {
				content := selection.Text()
				srcUrl := ""
				if url, ok := selection.Attr("src"); ok {
					//fmt.Println("script src==", url)
					url = strings.TrimPrefix(url, " ")
					srcUrl = url
				}
				if documentReferrerRex.MatchString(content) && indexOfRex.MatchString(content) && locationHrefRex.MatchString(content) {
					dark_chain[Md5Str(pageUrl+content)] = CheckRes{
						Url:   pageUrl,
						Value: simplifyContent(content),
					}
					//todo 暗链=url
					//fmt.Println("document.referrer true")
				}
				if evalRex.MatchString(content) {
					dark_chain[Md5Str(pageUrl+content)] = CheckRes{
						Url:   pageUrl,
						Value: simplifyContent(content),
					}
					//todo 暗链=url
					//fmt.Println("eval true")
				}
				if srcUrl != "" && evalRex.MatchString(srcUrl) {
					dark_chain[Md5Str(pageUrl+srcUrl)] = CheckRes{
						Url:   pageUrl,
						Value: srcUrl,
					}
				}
				if unicodeRex.MatchString(content) {
					dark_chain[Md5Str(pageUrl+content)] = CheckRes{
						Url:   pageUrl,
						Value: simplifyContent(content),
					}
					//todo 暗链=url
					//fmt.Println("unicode true")
				}
				if srcUrl != "" && unicodeRex.MatchString(srcUrl) {
					dark_chain[Md5Str(pageUrl+srcUrl)] = CheckRes{
						Url:   pageUrl,
						Value: srcUrl,
					}
				}
				if baseRex.MatchString(content) {
					dark_chain[Md5Str(pageUrl+content)] = CheckRes{
						Url:   pageUrl,
						Value: simplifyContent(content),
					}
					//todo 暗链=url
					fmt.Println("bash true")
				}
				if srcUrl != "" && baseRex.MatchString(srcUrl) {
					dark_chain[Md5Str(pageUrl+srcUrl)] = CheckRes{
						Url:   pageUrl,
						Value: srcUrl,
					}
				}
			})

			//遍历所有marquee标签 ，通过特征 判断是否是暗链
			dom.Find("marquee").Each(func(i int, selection *goquery.Selection) {
				//fmt.Println(selection.Attr("width"))
				aUrl := ""
				selection.Find("a").Each(func(i int, selectionSub *goquery.Selection) {
					ok := false
					aUrl, ok = selectionSub.Attr("href")
					if ok {
						return
					}
				})
				if aUrl != "" && !checkUrlDomain(aUrl, doMainTop) { //marquee标签内有a标签且地址不是当前域名  判断marquee标签宽高是否可疑
					width, widthExists := selection.Attr("width")
					height, heightExists := selection.Attr("height")
					if widthExists && heightExists {
						widthNum, _ := strconv.Atoi(width)
						heightNum, _ := strconv.Atoi(height)
						if widthNum <= 10 && heightNum <= 10 { //宽高都小于10
							//可疑暗链
							dark_chain[Md5Str(pageUrl+aUrl)] = CheckRes{
								Url:   pageUrl,
								Value: aUrl,
							}
						}
					}
				}
			})

			//遍历所有meta标签 ，通过特征 判断是否是暗链
			dom.Find("meta").Each(func(i int, selection *goquery.Selection) {

				aUrl := ""
				ok := false
				aUrl, ok = selection.Attr("href")
				if !ok {
					return
				}
				if aUrl != "" { //mate标签内有url 判断mate是有有 http-equiv属性
					equiv, equivExists := selection.Attr("http-equiv")

					if equivExists && equiv == "refresh" {
						if !checkUrlDomain(aUrl, doMainTop) { //地址不是当前域名
							//可疑暗链
							dark_chain[Md5Str(pageUrl+aUrl)] = CheckRes{
								Url:   pageUrl,
								Value: aUrl,
							}
						}

					}
				}
			})
		}
	}
	return len(dark_chain) > 0, dark_chain
}

//检查iframe内容是否有挂马特征
func checkIframeHangingHorse(html []*string, pageUrl, doMainTop string) (ok bool, hangingHorse map[string]CheckRes) {

	hangingHorse = make(map[string]CheckRes, 0)
	if len(html) > 0 {
		for _, v := range html {

			dom, err := goquery.NewDocumentFromReader(strings.NewReader(*v))
			if err != nil {
				return false, hangingHorse
			}
			//遍历所有iframe标签 ，通过特征 判断是否是挂马
			dom.Find("iframe").Each(func(i int, selection *goquery.Selection) {
				hangingOk := false //   疑是挂马
				srcUrl := ""
				if url, isok := selection.Attr("src"); isok {
					url = strings.TrimPrefix(url, " ")
					srcUrl = url
				}
				//有地址并且不是当前域名的地址
				if srcUrl != "" && !checkUrlDomain(srcUrl, doMainTop) {
					{ //通过标签属性+地址是否同顶级域名 判断是否挂马
						content, styleExists := selection.Attr("style")
						parentContent, parentStyleExists := selection.Parent().Attr("style")
						//fmt.Println("style==", content)
						if styleExists || parentStyleExists {
							//判断元素或父元素属性 带隐藏style
							if displayNoneRex.MatchString(content) || displayNoneRex.MatchString(parentContent) {
								hangingOk = true
							}
							if (positionAbsoluteRex.MatchString(content) || (positionAbsoluteTopRex.MatchString(content) || positionAbsoluteBottomRex.MatchString(content) || positionAbsoluteRightRex.MatchString(content) || positionAbsoluteLeftRex.MatchString(content))) ||
								(positionAbsoluteRex.MatchString(parentContent) || (positionAbsoluteTopRex.MatchString(parentContent) || positionAbsoluteBottomRex.MatchString(parentContent) || positionAbsoluteRightRex.MatchString(parentContent) || positionAbsoluteLeftRex.MatchString(parentContent))) {
								hangingOk = true
							}
							//if whiteColorRex.MatchString(content) || whiteColorRex.MatchString(parentContent) {
							//	none = true
							//}
							if fontSize0Rex.MatchString(content) || fontSize0Rex.MatchString(parentContent) {
								hangingOk = true
							}
						}
						width, widthExists := selection.Attr("width")
						height, heightExists := selection.Attr("height")
						parentWidth, parentWidthExists := selection.Parent().Attr("width")
						parentHeight, parentHeightExists := selection.Parent().Attr("height")
						//fmt.Println("height height==", width, height)
						if !hangingOk && (widthExists && heightExists) {
							//元素宽高 均小于10
							widthNum, _ := strconv.Atoi(width)
							heightNum, _ := strconv.Atoi(height)
							if widthNum <= 10 && heightNum <= 10 {
								hangingOk = true
							}
						}
						if !hangingOk && (parentWidthExists && parentHeightExists) {
							//父元素宽高 均小于10
							widthNum, _ := strconv.Atoi(parentWidth)
							heightNum, _ := strconv.Atoi(parentHeight)
							if widthNum <= 10 && heightNum <= 10 {
								hangingOk = true
							}
						}

					}

					//通过是否备案来判断
					if !hangingOk {
						icp := NewIcpCheckSource()
						icp.GetToken(getIcpTokenKey)
						icp.Domain, _ = GetDomain(srcUrl)
						if _, icpOk, err := icp.Posticp(true); err == nil && !icpOk {
							hangingOk = true
						}
					}
				}

				//fmt.Println("none==", none)
				if hangingOk {
					hangingHorse[Md5Str(pageUrl+srcUrl)] = CheckRes{
						Url:   pageUrl,
						Value: srcUrl,
					}
				}
			})
		}
	}
	//fmt.Println("hangingHorse==", hangingHorse)
	return len(hangingHorse) > 0, hangingHorse
}

//当前请求的url，当前请求地址的顶级域名，得到的响应地址
func checkScriptHangingHorse(domainTop, url, location string) (hangingHorse map[string]CheckRes) {
	hangingHorse = make(map[string]CheckRes, 0)

	//响应地址的顶级域名和请求地址的顶级域名如果不同，可疑挂马
	if top, _ := GetDomain(location); top != domainTop {

		icp := NewIcpCheckSource()
		icp.GetToken(getIcpTokenKey)
		icp.Domain, _ = GetDomain(location)
		if _, icpOk, err := icp.Posticp(true); err == nil && !icpOk {
			hangingHorse[Md5Str(url+location)] = CheckRes{
				Url:   url,
				Value: location,
			}
		}

	}
	return
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

/**
md5
*/
func Md5Str(str string) string {
	data := []byte(str)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	return md5str
}

//简化内容  内容太长中间使用省略号
func simplifyContent(content string) string {
	if len([]byte(content)) < 20 {
		return content
	}
	str := strings.Split(content, "")
	if len(str) > 201 {
		s := str[0:100]
		contentLi := append(s, "........(中间省略)")
		contentLi = append(contentLi, str[len(str)-101:]...)
		content = strings.Join(contentLi, "")
	}
	return content
}
