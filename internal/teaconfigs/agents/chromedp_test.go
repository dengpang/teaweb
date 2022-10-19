package agents

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"regexp"
	"testing"
	"time"
)

func Test_run(t *testing.T) {
	en, html, err := chromeDpRun("http://127.0.0.3", nil)
	fmt.Println(en)
	fmt.Println(html)
	fmt.Println(err)
	time.Sleep(time.Second * 5)
	en, html, err = chromeDpRun("http://www.baidu.com", en.Context)
	fmt.Println(en)
	fmt.Println(html)
	fmt.Println(err)
	time.Sleep(time.Second * 5)
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
	ctx, cancel2 := chromedp.NewRemoteAllocator(ctx, "ws://182.150.0.86:9222") //使用远程调试，可以结合下面的容器使用
	fmt.Println(2)

	ctx, cancel3 := chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf)) // 会打开浏览器并且新建一个标签页进行操作
	fmt.Println(3)

	//err := chromedp.Run(ctx, chromedp.Navigate(`https://blog.csdn.net/`))
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(`http://www.webl.cn`),
	})
	fmt.Println(err)

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
	url := "https://www.baidu.com"
	fmt.Println(GetDomain(url))
}
