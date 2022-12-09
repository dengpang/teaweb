package chrome_host

import (
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

const (
	ChromeHostConfigFile = "chrome_host.conf"
)

// 变量
var sharedChromeHostConfig *ChromeHostConfig = nil

// 远程浏览器配置
type ChromeHostConfig struct {
	List []List `json:"list"`
}
type List struct {
	Id     int    `json:"id"`
	Addr   string `json:"addr"` //主机地址
	Port   int    `json:"port"` //
	CpuNum int    `json:"cpu_num"`
}

// 取得共享的配置
func SharedChromeHostConfig() *ChromeHostConfig {
	if sharedChromeHostConfig != nil {
		return sharedChromeHostConfig
	}
	config := &ChromeHostConfig{}
	data, err := ioutil.ReadFile(Tea.ConfigFile(ChromeHostConfigFile))
	if err != nil {
		return config
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		logs.Error(err)
		return config
	}
	sharedChromeHostConfig = config
	return config
}

// 保存
func (this *ChromeHostConfig) Save() error {
	data, err := yaml.Marshal(this)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(Tea.ConfigFile(ChromeHostConfigFile), data, 0777)
}
