package agents

import (
	"fmt"
	"github.com/TeaWeb/build/internal/teaconfigs/shared"
	"github.com/TeaWeb/build/internal/teatesting"
	"github.com/iwind/TeaGo/logs"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestKeywordCheckSource_Execute(t *testing.T) {
	source := NewKeywordCheckSource()
	source.URL = "https://baidu.com/"
	value, err := source.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(value, t)
}

func TestKeywordCheckSource_ExecutePost(t *testing.T) {
	if !teatesting.RequireHTTPServer() {
		return
	}

	source := NewKeywordCheckSource()
	source.Method = http.MethodPost
	source.URL = "http://127.0.0.1:9991/webhook?hell=world"
	source.DataFormat = SourceDataFormatSingeLine
	source.Headers = []*shared.Variable{
		/**{
			Name:  "Content-Type",
			Value: "application/json",
		},**/
		{
			Name:  "Hello",
			Value: "World",
		},
	}
	source.Params = []*shared.Variable{
		{
			Name:  "name",
			Value: "lu",
		},
		{
			Name:  "age",
			Value: "20",
		},
	}
	source.TextBody = "Hello, World" // will be ignored because params is not empty
	err := source.Validate()
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(result, t)
}

func TestKeywordCheckSource_ExecutePut(t *testing.T) {
	if !teatesting.RequireHTTPServer() {
		return
	}

	source := NewKeywordCheckSource()
	source.URL = "http://127.0.0.1:9991/webhook"
	source.Method = http.MethodPut
	source.DataFormat = SourceDataFormatSingeLine
	source.Headers = []*shared.Variable{
		{
			Name:  "Content-Type",
			Value: "application/json",
		},
	}
	source.TextBody = "HELLO, WORLD"
	result, err := source.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}

	logs.PrintAsJSON(result, t)
}

func TestReg(t *testing.T) {

	allKeyword := []string{"下贱", "专业代理", "中国猪", "代孕妈妈", "代开发票", "代生孩子", "位的qq", "低价出售", "你他妈", "你吗b", "你妈的", "你麻痹", "信用卡提现", "借腹生子", "傻b", "傻比", "傻逼", "全家不得好死", "全家死光", "全家死绝", "刹笔", "刻章办", "卧槽", "卧艹", "台湾猪", "大sb", "大麻", "妈了个逼", "妈逼", "娘西皮", "婊子", "婊子养的", "婴儿汤", "干你妈", "干你娘", "广告代理", "我操", "我日你", "我草", "找个妈妈", "找个爸爸", "操他妈", "操你全家", "操你大爷", "操你妈", "操你娘", "操你祖宗", "擦你妈", "改卷内幕", "无抵押贷款", "无耻", "日你妈", "替考试", "杀b", "欠干", "款到发货", "死全家", "沙比", "海luo因", "海洛因", "煞笔", "煞逼", "爆你菊", "狗娘养", "狗操", "狗日的", "狗杂种", "狗草", "电脑传讯", "白痴", "真他妈", "私家侦探", "艹你", "草你丫", "草你吗", "蚁力神", "装b", "调查婚外情", "贱b", "贱人", "贱比", "贱货", "送qb", "透视仪", "透视功能", "透视器", "透视扑", "透视眼睛", "透视眼镜", "透视药", "透视镜", "隐形耳机", "马勒", "麻痹的", "麻黄草", "黑手党", "泼尼松", "雄烯二醇", "西布曲明", "莫达非尼", "登录智安云监控平台"}
	bodyStr := `<!doctype html>\n<html lang=\"zh\">\n<head>\n    <meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\"/>\n    <link rel=\"shortcut icon\" href=\"/images/favicon.ico?v=bXlZmk\"/><!--logo.png-->\n    <title>登录智安云监控平台</title>\n\t<meta name=\"viewport\" content=\"width=device-width, initial-scale=1, user-scalable=0\">\n    <script type=\"text/javascript\">\nwindow.TEA = {\n\t\"ACTION\": {\"data\":{\"teaDemoEnabled\":false,\"token\":\"58a03af5bb7d937583ccce454dcea8ed1648446421\"},\"base\":\"\",\"module\":\"\",\"parent\":\"/login\",\"actionParam\":false}\n}\n</script>\n<script type=\"text/javascript\" src=\"/js/vue.js?v=bWlZE2\"></script>\n<script type=\"text/javascript\" src=\"/js/vue.tea.js?v=bXGHSf\"></script>\n<script type=\"text/javascript\" src=\"/_/@default/login/index.js?v=bWlZE2\"></script>\n<link rel=\"stylesheet\" type=\"text/css\" href=\"/_/@default/login/index.css?v=bWlZE2\" media=\"all\"/>\n    <link rel=\"stylesheet\" type=\"text/css\" href=\"/css/semantic.min.css?v=bWlZE2\" media=\"all\"/>\n\t<script type=\"text/javascript\" src=\"/js/md5.min.js?v=bWlZE`
	start := time.Now() // 获取当前时间
	for _, reg_rule := range allKeyword {
		reg := regexp.MustCompile(reg_rule)
		if reg.MatchString(bodyStr) {
			fmt.Println(reg_rule)
			continue
		}
	}
	defer func() {
		elapsed := time.Since(start)
		fmt.Println("搜索执行耗时 ----", elapsed)
	}()
	reg := regexp.MustCompile("莫达非尼|登录智安云监控平台")
	fmt.Println(reg.MatchString(bodyStr))

}

func TestMatchUrl(t *testing.T) {
	//s := `<div class="xe-widget xe-conversations box2 label-info" onclick="window.open('https://dribbble.com/', '_blank')" data-toggle="tooltip" data-placement="bottom" title="" data-original-title="https://dribbble.com/">
	//                    <div class="xe-comment-entry">
	//                        <a class="xe-user-img">
	//                            <img data-src="../assets/images/logos/dribbble.png" class="lozad img-circle" width="40" src="../assets/images/logos/dribbble.png" data-loaded="true">
	//                        </a>
	//                        <div class="xe-comment">
	//                            <a href="#" class="xe-user-name overflowClip_1">
	//                                <strong>Dribbble</strong>
	//                            </a>
	//                            <p class="overflowClip_2">全球UI设计师作品分享平台。</p>
	//                        </div>
	//                    </div>
	//                </div>`
	//key := &KeywordCheckSource{}
	//list := key.MatchUrl([]byte(s))
	//fmt.Println(list)
}

func Test_re(t *testing.T) {

	keywordsStr := "戴秉国,黄镇,刘延东,刘瑞龙,俞正声,黄敬,薄熙,薄一波,周小川,周建南,温云松,徐明,江泽慧,江绵恒,江绵康,李小鹏,李鹏,李小琳,朱云来,朱容基,让国人愤怒的第二代身份证,第二代身份证,文化大革命***,胡海峰,六四,反共,共产党,陈良宇,老丁,莱仕德事件***,fuck,地下的先烈们纷纷打来电话询问*,李洪志,大纪元,真善忍,新唐人,肉棍,淫靡,淫水,六四事件,迷昏药,迷魂药,窃听器,六合彩,买卖枪支,三唑仑,麻醉药,麻醉乙醚,短信群发器"
	reg, _ := regexp.Compile(`[\\\^\$\*\+\?\{\}\.\[\]\(\)\-\|]`)
	keywordsStr = reg.ReplaceAllStringFunc(keywordsStr, func(b string) string {
		//正则 元字符需要转义
		return `\` + b
	})
	fmt.Println(keywordsStr)
	//regx := regexp.MustCompile("文化大革命\\*\\*\\*s")
	//if regx.MatchString("fkdsjfWE那壶文化大革命***s返回黄金时代") {
	//	fmt.Println("有")
	//} else {
	//	fmt.Println("没有")
	//}
	key := `文化大革命\*\*\*\\`
	keys := reg.ReplaceAllStringFunc(key, func(b string) string {
		//正则 元字符需要转义
		fmt.Println(b)
		return strings.TrimPrefix(b, `\`)
	})
	fmt.Println(keys)
}

func Test_syncMap(t *testing.T) {
	fmt.Println(runtime.NumCPU() / 2)
	m := &sync.Map{}
	m.Store("1", true)
	m.Store("2", false)
	if value, ok := m.Load("1"); ok {
		if value == false {
			fmt.Println("ok")
		}
	}
}
func Test_Run(t *testing.T) {
	eng, html, err := chromeDpRun("http://www.baidu.com", nil)
	fmt.Println(eng)
	fmt.Println(html)
	fmt.Println(err)
	time.Sleep(time.Second * 10)
	eng, html, err = chromeDpRun("http://www.sougou.com", eng.Context)
	fmt.Println(eng)
	fmt.Println(html)
	fmt.Println(err)
	time.Sleep(time.Second * 10)
	eng.Close()
}
