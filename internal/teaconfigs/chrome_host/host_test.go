package chrome_host

import "testing"

func TestSharedHostConfig(t *testing.T) {
	config := SharedChromeHostConfig()
	t.Log(config)
}

func TestDBConfig_Save(t *testing.T) {
	config := SharedChromeHostConfig()
	config.List = []List{
		{
			Addr:   "127.0.0.2",
			Port:   9222,
			CpuNum: 4,
		},
	}
	err := config.Save()
	if err != nil {
		t.Fatal(err)
	}
}
