package chrome_host

import (
	"fmt"
	"github.com/TeaWeb/build/internal/teaconfigs/chrome_host"
	"github.com/chromedp/chromedp"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"

	"context"
)

type IndexAction actions.Action

// 备份列表
func (this *IndexAction) Run(params struct{}) {
	ctx, _ := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
	files := chrome_host.SharedChromeHostConfig()
	if len(files.List) > 0 {
		this.Data["files"] = lists.Map(files.List, func(k int, v interface{}) interface{} {
			cfg, _ := v.(chrome_host.List)
			winCtx, _ := chromedp.NewRemoteAllocator(ctx, fmt.Sprintf("http://%v:%v", cfg.Addr, cfg.Port)) //使用远程调试，可以结合下面的容器使用
			winCtx, _ = chromedp.NewContext(winCtx)
			targets, err := chromedp.Targets(winCtx)
			window := 0
			if err == nil {
				for _, vv := range targets {
					//fmt.Println(*v)
					if vv.Type == "page" {
						window++
					}
				}
			}
			chromedp.Cancel(winCtx)
			return maps.Map{
				"addr":    cfg.Addr,
				"id":      cfg.Id,
				"port":    cfg.Port,
				"cpu_num": cfg.CpuNum,
				"window":  window,
			}
		})
	} else {
		this.Data["files"] = make([]struct{}, 0)
	}

	this.Show()
}
