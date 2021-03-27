// author: ashing
// time: 2020/7/19 12:07 下午
// mail: axingfly@gmail.com
// Less is more.

package main

import (
	"clock/v3/config"
	"clock/v3/master/server"
	"clock/v3/storage"
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	filePath string // 配置文件路径
	help     bool   // 帮助
)

func usage() {
	fmt.Fprintf(os.Stdout, `clock - simlpe scheduler
Usage: clock [-h help] [-c ./config.yaml]
Options:
`)
	flag.PrintDefaults()
}

func main() {
	flag.StringVar(&filePath, "c", "./config.yaml", "配置文件所在")
	flag.BoolVar(&help, "h", false, "帮助")
	flag.Usage = usage
	flag.Parse()
	if help {
		flag.PrintDefaults()
		return
	}

	// 设置配置文件和静态变量
	config.SetConfig(filePath)
	if err := storage.SetDb(); err != nil {
		logrus.Fatalf("[main] set up error: %v", err)
	}

	defer storage.RevokeDb()
	address := config.Config.GetString("server.host")
	if address == "" {
		logrus.Fatal("can not find any server host config")
	}

	engine, err := server.CreateEngine()
	if err != nil {
		logrus.Fatal(err)
	}

	if e := engine.Start(address); e != nil {
		logrus.Fatal(e)
	}
}
