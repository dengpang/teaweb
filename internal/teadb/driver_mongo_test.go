package teadb

import (
	"github.com/TeaWeb/build/internal/teatesting"
	"github.com/go-resty/resty/v2"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestMongoDriver_buildFilter(t *testing.T) {
	q := new(Query)
	q.Init()
	q.Attr("name", "lu")
	q.Op("age", OperandGt, 1024)
	q.Op("age", OperandLt, 2048)
	q.Op("count", OperandEq, 3)

	driver := new(MongoDriver)
	filter, err := driver.buildFilter(q)
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(filter, t)
}

func TestMongoDriver_buildFilter_Or(t *testing.T) {
	q := new(Query)
	q.Init()
	q.Attr("a", 1)
	q.Or([]*OperandList{
		NewOperandList().Add("timestamp", NewOperand(OperandEq, "123")),
		NewOperandList().Add("timestamp",
			NewOperand(OperandGt, "456"),
			NewOperand(OperandNotIn, []int{1, 2, 3}),
		),
		NewOperandList().Add("timestamp", NewOperand(OperandLt, 1024)),
	})
	driver := new(MongoDriver)
	filter, err := driver.buildFilter(q)
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(filter, t)
}

func TestMongoDriver_setMapValue(t *testing.T) {
	m := maps.Map{}

	driver := new(MongoDriver)
	driver.setMapValue(m, []string{"a", "b", "c", "d", "e"}, 123)
	logs.PrintAsJSON(m, t)
}

func TestMongoDriver_connect(t *testing.T) {
	if !teatesting.RequireDBAvailable() {
		return
	}

	driver := new(MongoDriver)
	client, err := driver.connect()
	if err != nil {
		t.Log("ERROR:", err.Error())
		return
	}
	t.Log("client:", client)
}

func TestMongoDriver_Test(t *testing.T) {
	if !teatesting.RequireDBAvailable() {
		return
	}

	driver := new(MongoDriver)
	err := driver.Test()
	if err != nil {
		t.Log("ERROR:", err.Error())
		return
	}
	t.Log("client:", driver)
}

func TestMongoDriver_convertArrayElement(t *testing.T) {
	driver := new(MongoDriver)
	t.Log(driver.convertArrayElement("value.usage.avg"))
	t.Log(driver.convertArrayElement("value.usage.all.0"))
	t.Log(driver.convertArrayElement("value.0"))
}

func TestMongoDriver_ListTables(t *testing.T) {
	if !teatesting.RequireDBAvailable() || !teatesting.RequireMongoDB() {
		return
	}

	driver := new(MongoDriver)
	driver.isAvailable = true
	names, err := driver.ListTables()
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log(names)
}

func Test_Push(t *testing.T) {
	wg := &sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)

		go func(i int) {
			cli := resty.New().SetDebug(true).SetTimeout(time.Second * 600).SetHeaders(map[string]string{
				"User-Agent":        "TeaWeb Agent",
				"Tea-Agent-Id":      "b20081671825bee6",
				"Tea-Agent-Key":     "37daec6ae3176f7de5dc84583496c871",
				"Tea-Agent-Version": "1.5.0",
				"Tea-Agent-Os":      "macOs",
				"Tea-Agent-Arch":    "macOs",
			}).R()
			s := strconv.Itoa(i)
			cli.SetBody(`{"event":"ItemEvent","agentId":"b20081671825bee6","appId":"system","itemId":"b137570ed9d96a64","value":{"status":` + s + `},"error":"Get \"https://zq.zj96596.com:689/netbank/login.html\": tls: server selected unsupported protocol version 302","beginAt":1686710335,"timestamp":1686710335,"costMs":187.999852}`).Post("http://127.0.0.1:7777/api/agent/push")
			wg.Done()
		}(i)

	}
	wg.Wait()
	//res, err :=
	//fmt.Println(res, err)
}
