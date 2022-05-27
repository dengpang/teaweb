package keyword

import (
	"fmt"
	"testing"
)

func TestKeyConfig(t *testing.T) {

	{
		key := NewKeywordConfig()
		fmt.Println(key.Id)
		key.Name = "自定义"
		key.Keyword = ""
		key.Number = 1
		fmt.Println(key.Save())
	}
}

func TestKeyList(t *testing.T) {
	l := ActionListFiles()
	for k, v := range l {
		fmt.Println(k)
		fmt.Println(v)
	}
}
