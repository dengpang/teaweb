package agents

import (
	"fmt"
	"github.com/TeaWeb/build/internal/teautils"
	"golang.org/x/sync/singleflight"
	"runtime"
	"time"
)

func init() {
	RegisterAllDataSources()
}

var (
	lockG             = &singleflight.Group{}
	getIcpTokenKey    = "check_getIcpTokenKey"
	getIcpResponseKey = "check_getIcpResponseKey:domain:"
	Cache             = teautils.New(5*time.Minute, 10*time.Minute)
	getCssKey         = "check_getCssContentKey"

	chromeHost = []*ChromeHost{}
)

type (
	ChromeHost struct {
		Addr   string `json:"addr"`
		CpuNum int    `json:"cpu_num"`
		Port   int    `json:"port"`
	}
)

func InitChrome(host []*ChromeHost) {
	fmt.Println("chromeHost==", host)
	chromeHost = []*ChromeHost{}
	for _, v := range host {
		chromeHost = append(chromeHost, &ChromeHost{
			Addr:   v.Addr,
			CpuNum: v.CpuNum,
			Port:   v.Port,
		})

	}
	if len(chromeHost) == 0 {
		if checkChromePort("127.0.0.1", "9222") {
			//未配置浏览器主机，默认使用127.0.0.1
			chromeHost = []*ChromeHost{
				{
					Addr:   "127.0.0.1",
					CpuNum: runtime.NumCPU(),
					Port:   9222,
				},
			}
		}

		return
	}
}

/*
*
设置缓存
返回参数,,第一个数据,,第二个数据执行结果
*/
func CheckCache(key string, fn func() (interface{}, error), duration int64, needCache bool) (interface{}, error) {
	s, ok := Cache.Get(key)
	if needCache && ok {
		return s, nil
	} else {
		var re interface{}
		//Num, ok := fn()
		//同一时间只有一个带相同key的函数执行 防击穿
		tokens, err, _ := lockG.Do(key, fn)
		if err == nil {
			Cache.Set(key, tokens, time.Duration(duration)*time.Second)
			re = tokens
		} else {
			re = tokens
		}

		return re, err
	}

}

type WeightNode struct {
	Addr            string //地址
	Weight          int    //初始化权重
	EffectiveWeight int    //有效权重。默认和weight相同
	CurrentWeight   int    //临时权重
	Port            int    //端口
}

type WeightRoundLoadBalance struct {
	list []*WeightNode
}

func (r *WeightRoundLoadBalance) Add(addr string, Weight, CurrentWeight, port int) error {

	newNode := &WeightNode{
		Addr:            addr,
		Weight:          Weight,
		EffectiveWeight: Weight,
		CurrentWeight:   CurrentWeight,
		Port:            port,
	}
	r.list = append(r.list, newNode)
	return nil
}
func (r *WeightRoundLoadBalance) Next() *WeightNode {
	var total int = 0
	var best *WeightNode
	for index, v := range r.list {
		total += v.CurrentWeight
		r.list[index].CurrentWeight += r.list[index].EffectiveWeight
		if v.EffectiveWeight < v.CurrentWeight {
			v.EffectiveWeight++
		}
		if best == nil || best.CurrentWeight < v.CurrentWeight {
			best = v
		}
	}
	if best == nil {
		return nil
	}
	best.CurrentWeight -= total
	return best
}

func (r *WeightRoundLoadBalance) Get() *WeightNode {
	return r.Next()
}
