package keyword

import (
	"github.com/TeaWeb/build/internal/teaconfigs/shared"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"gopkg.in/yaml.v3"
	"regexp"
	"sort"
)

type KeywordConfig struct {
	Id      string `yaml:"id" json:"id"`           // ID
	Default bool   `yaml:"default" json:"default"` // 是否内置 内置不支持修改
	Name    string `yaml:"name" json:"name"`       // 名称
	Keyword string `yaml:"keyword" json:"keyword"` // 关键词
	Number  int    `yaml:"number" json:"number"`   // 关键词数量
	Sort    int    `yaml:"sort" json:"sort"`       // 从小到大排序
}

// 获取新对象
func NewKeywordConfig() *KeywordConfig {
	return &KeywordConfig{
		Default: true,
		Id:      rands.HexString(16),
	}
}

// 从文件中获取对象
func NewKeywordConfigFromFile(filename string) *KeywordConfig {
	reader, err := files.NewReader(Tea.ConfigFile("keyword/" + filename))
	if err != nil {
		return nil
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			logs.Error(err)
		}
	}()
	keywd := &KeywordConfig{}
	err = reader.ReadYAML(keywd)
	if err != nil {
		return nil
	}
	return keywd
}

// 根据ID获取对象
func NewKeywordConfigFromId(agentId string) *KeywordConfig {
	if len(agentId) == 0 {
		return nil
	}
	return NewKeywordConfigFromFile("keyword." + agentId + ".conf")

}

// 判断是否为内置词库
func (this *KeywordConfig) IsDefault() bool {
	return this.Default
}

// 文件名
func (this *KeywordConfig) Filename() string {
	return "keyword." + this.Id + ".conf"
}

// 列出所有文件
func ActionListFiles() []*KeywordConfig {
	// 已备份
	//result := []*KeywordConfig{}
	result := make(personSlice, 0)
	reg := regexp.MustCompile("^keyword.[0-9a-zA-Z]{16}.conf$")
	dir := files.NewFile(Tea.ConfigFile("keyword/"))
	if dir.Exists() {
		for _, f := range dir.List() {
			if !reg.MatchString(f.Name()) {
				continue
			}
			cont, err := f.Reader()
			if err != nil {
				continue
			}
			keywd := &KeywordConfig{}
			err = cont.ReadYAML(keywd)
			if err != nil {
				continue
			}
			result = append(result, keywd)
		}
	}
	sort.Stable(result)
	return result
}

// 保存
func (this *KeywordConfig) Save() error {
	shared.Locker.Lock()
	defer shared.Locker.WriteUnlockNotify()

	dirFile := files.NewFile(Tea.ConfigFile("keyword"))
	if !dirFile.Exists() {
		err := dirFile.Mkdir()
		if err != nil {
			logs.Error(err)
		}
	}

	writer, err := files.NewWriter(Tea.ConfigFile("keyword/" + this.Filename()))
	if err != nil {
		return err
	}
	defer func() {
		err := writer.Close()
		if err != nil {
			logs.Error(err)
		}
	}()
	_, err = writer.WriteYAML(this)
	return err
}

// 删除
func (this *KeywordConfig) Delete() error {

	// 删除board
	{
		f := files.NewFile(Tea.ConfigFile("keyword/keyword." + this.Id + ".conf"))
		if f.Exists() {
			err := f.Delete()
			if err != nil {
				return err
			}
		}
	}

	f := files.NewFile(Tea.ConfigFile("keyword/" + this.Filename()))
	return f.Delete()
}

// YAML编码
func (this *KeywordConfig) EncodeYAML() ([]byte, error) {
	return yaml.Marshal(this)
}

type personSlice []*KeywordConfig

func (s personSlice) Len() int           { return len(s) }
func (s personSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s personSlice) Less(i, j int) bool { return s[i].Sort < s[j].Sort } //升叙
