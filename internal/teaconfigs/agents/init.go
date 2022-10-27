package agents

import "time"

func init() {
	RegisterAllDataSources()
}

/**
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
