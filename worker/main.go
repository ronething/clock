// author: ashing
// time: 2020/7/19 12:07 下午
// mail: axingfly@gmail.com
// Less is more.

package main

import (
	"clock/config"
	"clock/master/param"
	"clock/storage"
	"clock/worker/scheduler"
	"flag"
	"fmt"
	"os"
	"time"
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
	param.SetStatic()
	storage.SetDb()

	// 初始化调度器
	scheduler.NewScheduler()

	for {
		time.Sleep(1 * time.Second)
	}

}
