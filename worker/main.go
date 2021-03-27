// author: ashing
// time: 2020/7/19 12:07 下午
// mail: axingfly@gmail.com
// Less is more.

package main

import (
	"clock/v3/config"
	"clock/v3/storage"
	"clock/v3/worker/server"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

var (
	filePath  string // 配置文件路径
	help      bool   // 帮助
	version   bool
	gitHash   string
	buildTime string
	goVersion string
)

func init() {
	flag.StringVar(&filePath, "c", "./config.yaml", "配置文件所在")
	flag.BoolVar(&help, "h", false, "帮助")
	flag.BoolVar(&version, "v", false, "版本说明")
	flag.Usage = usage
}

func main() {
	flag.Parse()
	if help {
		flag.PrintDefaults()
		return
	}
	if version {
		buildInfo()
		return
	}

	// 设置配置文件和静态变量
	config.SetConfig(filePath)

	if err := storage.SetDb(); err != nil {
		log.Errorf("[main] set up error: %v", err)
		return
	}
	defer storage.RevokeDb()

	address := config.Config.GetString("server.worker")
	if address == "" {
		log.Errorf("[main] 找不到对应的服务端地址")
		return
	}
	if err := startMetricsServer(address); err != nil {
		log.Errorf("[main] %v", err)
		return
	}
	log.Infof("[main] metrics 服务已经启动: %v", address)

	// 初始化调度器
	if err := storage.InitScheduler(); err != nil {
		log.Fatalf("[main] 创建调度器发生错误")
	}

	c := make(chan os.Signal, 1)
	go storage.SubCronJob(c)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// DONE: 优雅关停
	for {
		s := <-c
		log.Infof("[main] 捕获信号 %s", s.String())
		switch s {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			// 停止调度器 并等待正在 running 的任务执行结束 TODO: 有没有必要设置一个 timeout 假设一直不停止怎么办
			ctx := storage.StopScheduler()
			timer := time.NewTimer(1 * time.Second)
			for {
				select {
				case s = <-c: // 再次接收到中断信号 则直接退出
					if s == syscall.SIGINT {
						log.Debugf("[main] 再次接收到退出信号 %s", s.String())
						goto End
					}
				case <-ctx.Done():
					log.Infof("[main] 调度器所有任务执行完成")
					goto End
				case <-timer.C:
					log.Debugf("[main] 调度器有任务正在执行中")
					timer.Reset(1 * time.Second)
				}
			}
		End:
			log.Debugf("[main] 暂停计时器")
			timer.Stop()
			log.Infof("[main] worker 正常退出")
			// TODO: notify 消息推送到相关负责人
			return // 很重要 不然程序无法退出
		case syscall.SIGHUP:
			log.Debugf("[main] 终端断开信号，忽略")
		default:
			log.Debugf("[main] other signal")
		}
	}

}

//startMetricsServer 暴露服务 metrics
func startMetricsServer(address string) error {
	engine, err := server.CreateEngine()
	if err != nil {
		return errors.Wrap(err, "metrics server 实例化发生错误")
	}

	go func() {
		var err error
		if err = engine.Start(address); err != nil {
			if strings.Contains(err.Error(), "bind: address already in use") {
				log.Errorf("端口被占用, %s", err.Error())
				return
			}
			log.Errorf("[main] 启动 metrics 服务发生错误: %v", err) // 这里不要用 Fatal 不然优雅关停会直接退出
			return
		}
		//正常启动是走不到这里的 因为上面会阻塞
		//log.Infof("[main] metrics 服务已经启动: %v", address)
	}()

	return nil
}

func usage() {
	fmt.Fprintf(os.Stdout, `clock - simlpe scheduler
Usage: clock [-h help] [-c ./config.yaml]
Options:
`)
	flag.PrintDefaults()
}

func buildInfo() {
	fmt.Fprintf(os.Stdout, `git commit hash: %s
build timestamp: %s
golang version: %s
`, gitHash, buildTime, goVersion)
}
