package agents

import (
	"context"
	"fmt"
	"github.com/TeaWeb/build/internal/teautils"
	"github.com/chromedp/chromedp"
	"log"
	"regexp"
	"testing"
	"time"
)

func Test_run(t *testing.T) {
	fmt.Println(time.Now())
	url := "http://www.pzsy888.com/abouts/intros.html"
	en, html, err := chromeDpRun(url, nil)
	fmt.Println(en.Url, en.Location)
	fmt.Println(*html[0])
	fmt.Println(err)
	domainTop, domain := GetDomain(url)
	Urls, _, err := GetUrlsAndCheck(html, domainTop, domain, url, 3)
	//监测结果
	if ok, res := checkIframeHangingHorse(html, url, domainTop); ok && len(res) > 0 {
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
	location := ""
	//err := chromedp.Run(ctx, chromedp.Navigate(`https://blog.csdn.net/`))
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(`http://www.fanyi.com`),
		chromedp.EvaluateAsDevTools("window.location.href;", &location),
	})
	fmt.Println(err)

	fmt.Println("location", location)
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

	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// navigate to a page, wait for an element, click
	var example string
	sel := `//*[@id="username"]`
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://github.com/awake1t`),
		chromedp.WaitVisible("body"),
		//缓一缓
		chromedp.Sleep(2*time.Second),

		chromedp.SendKeys(sel, "username", chromedp.BySearch), //匹配xpath

	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Go's time.After example:\n%s", example)
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

func Test_getDomain(t *testing.T) {
	url := "http://47.107.165.133:36111/yeyue.html?k=3Xrn9m913e6ISYyRHelJCLiAzN152L6lHeuUGeulGbv8iOzBHd0hmI6ICbyVnIsUWdyRnOis2YhJmIsUWdyRnOigXZzJCLlVnc0pjIz9WaiwiI1cDO4YDMxgjMxIiOiQWS0NWZylGZlJnIsIiN3ETMiojI0NWZylGZlJnIsATN6ISbvRmbhJnIsISN3ETMiojIklEbl5mbhh2YiwiI0cDO4YDMxgjMxIiOiQWSlRXazJyeOyYPJsi9&_=1666535114338#1666535255540\n["
	fmt.Println(GetDomain(url))
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
