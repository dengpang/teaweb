package agents

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestCheck(t *testing.T) {
	s := NewIcpCheckSource()
	s.Domain = "baidu.com"
	v, err := s.Execute(nil)
	fmt.Println(v)
	fmt.Println(err)

	time.Sleep(time.Second * 2)
	v, err = s.Execute(nil)
	fmt.Println(v)
	fmt.Println(err)

}

func TestJson(t *testing.T) {
	str := `{"code":200,"msg":"操作成功","params":{"expire":300000,"refresh":"eyJ0eXBlIjoyLCJ1IjoiMDk4ZjZiY2Q0NjIxZDM3M2NhZGU0ZTgzMjYyN2I0ZjYiLCJzIjoxNjY2MDAxNDI3ODEwLCJlIjoxNjY2MDAyMjA3ODEwfQ.2OJePipIUxMU1id7IE1BH80OKSXm7dX7hRNoC6kgHHI","bussiness":"eyJ0eXBlIjoxLCJ1IjoiMDk4ZjZiY2Q0NjIxZDM3M2NhZGU0ZTgzMjYyN2I0ZjYiLCJzIjoxNjY2MDAxNDI3ODEwLCJlIjoxNjY2MDAxOTA3ODEwfQ._kkc2s0pLn9viEqQRxFf8oORk91flHdrRtIl5wUsELw"},"success":true}
`
	js := &TokenRes{}
	e := json.Unmarshal([]byte(str), &js)
	fmt.Println(e)
	fmt.Println(js)

}
