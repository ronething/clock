package storage

import "os"

type Cache interface {
	// 获取单例 类型断言

	// 分布式锁
	TryLock(Task) (chan int, error)
	// 续租
	LeaseJob(chan int, string, string)
	// 发布
	Publish([]byte) error
	// 订阅
	Subscribe(chan os.Signal)
	// ping
	Ping() (string, error)
	// close conn
	Close() error
}
