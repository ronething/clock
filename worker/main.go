// author: ashing
// time: 2020/7/19 12:07 下午
// mail: axingfly@gmail.com
// Less is more.

package main

import (
	"clock/v3/config"
	"clock/v3/storage"
	"clock/v3/worker/scheduler"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// 初始化调度器
	if err := scheduler.NewScheduler(); err != nil {
		logrus.Fatalf("[main] 创建调度器发生错误")
	}

	c := make(chan os.Signal, 1)
	go scheduler.SubCronJob(c)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// DONE: 优雅关停
	for {
		s := <-c
		logrus.Infof("[main] 捕获信号 %s", s.String())
		switch s {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			// 停止调度器 并等待正在 running 的任务执行结束 TODO: 有没有必要设置一个 timeout 假设一直不停止怎么办
			ctx := scheduler.StopScheduler()
			timer := time.NewTimer(1 * time.Second)
			for {
				select {
				case s = <-c: // 再次接收到中断信号 则直接退出
					if s == syscall.SIGINT {
						logrus.Debugf("[main] 再次接收到退出信号 %s", s.String())
						goto End
					}
				case <-ctx.Done():
					logrus.Infof("[main] 调度器所有任务执行完成")
					goto End
				case <-timer.C:
					logrus.Debugf("[main] 调度器有任务正在执行中")
					timer.Reset(1 * time.Second)
				}
			}
		End:
			logrus.Debugf("[main] 暂停计时器")
			timer.Stop()
			logrus.Infof("[main] worker 正常退出")
			return // 很重要 不然程序无法退出
		case syscall.SIGHUP:
			logrus.Debugf("[main] 终端断开信号，忽略")
		default:
			logrus.Debugf("[main] other signal")
		}
	}

}
