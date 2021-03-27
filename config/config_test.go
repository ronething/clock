package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_initConfig(t *testing.T) {
	tmpConfig := `
grpc:
  addr: "0.0.0.0:8899"

redis:
  addr: "127.0.0.1:6379"
  auth: ""
  db: 15
`
	tmpFile, err := ioutil.TempFile(os.TempDir(), "config*.yaml") // * 会被替换为随机字符串
	if err != nil {
		t.Fatalf("create tempFile err: %v\n", err)
	}
	//t.Logf("tmp file name is %v\n", tmpFile.Name())
	_, err = tmpFile.Write([]byte(tmpConfig))
	if err != nil {
		t.Fatalf("write err: %v\n", err)
	}
	defer os.Remove(tmpFile.Name())
	SetConfig(tmpFile.Name())
	assert.Equal(t, "0.0.0.0:8899", Config.GetString("grpc.addr"))
	assert.Equal(t, 15, Config.GetInt("redis.db"))
}
