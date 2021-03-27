// author: ashing
// time: 2020/7/19 2:03 下午
// mail: axingfly@gmail.com
// Less is more.

package storage

import (
	"context"
	"fmt"
	standlog "log"
	"os"
	"sync"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

var cronScheduler *CronScheduler

type CronScheduler struct {
	mu        sync.Mutex
	scheduler *cron.Cron
	tasks     map[string]cron.EntryID
}

//NewCronScheduler 构造函数
func NewCronScheduler(scheduler *cron.Cron) *CronScheduler {
	return &CronScheduler{
		mu:        sync.Mutex{},
		scheduler: scheduler,
		tasks:     make(map[string]cron.EntryID),
	}
}

//GetTaskEntryId
func (c *CronScheduler) GetTaskEntryId(tid string) cron.EntryID {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tasks[tid] // if key not exists, value is zero
}

//PutTaskEntryId
func (c *CronScheduler) PutTaskEntryId(tid string, id cron.EntryID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tasks[tid] = id
}

//RemoveTask 删除定时器中的定时任务 根据 tid
func (c *CronScheduler) RemoveTaskByTid(tid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entryId := c.tasks[tid] // if key not exists, value is zero
	log.Infof("job %s, entryId is %v, now remove\n", tid, entryId)
	if entryId != 0 {
		c.scheduler.Remove(entryId)
	} else {
		log.Infof("job %s, entryId is %v, 不需要进行调度器任务删除\n", tid, entryId)
	}
	return
}

//RemoveTask 删除定时器中的定时任务
func (c *CronScheduler) RemoveTask(t *Task) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entryId := c.tasks[t.Tid] // if key not exists, value is zero
	log.Infof("job %s-%s, entryId is %v\n", t.Tid, t.Name, entryId)
	if entryId != 0 {
		c.scheduler.Remove(entryId)
	} else {
		log.Infof("job %s-%s, entryId is %v, 不需要进行调度器任务删除\n", t.Tid, t.Name, entryId)
	}
	return
}

func (c *CronScheduler) AddTask(t *Task) error {
	f := func() {
		if e := RunTask(t.Tid); e != nil {
			log.Errorf("[scheduler] exec task %s err: %v", t.Tid, e)
			// DONE: 如果 err 是没有找到 doc，则从调度器中 remove
			if e == RunTaskNotFoundTaskErr {
				log.Infof("[scheduler] 任务 %s-%s 没有找到"+
					"now remove from cron scheduler", t.Tid, t.Name)
				c.RemoveTask(t)
			}
		}
	}

	//加上时区的选择
	expression := fmt.Sprintf("CRON_TZ=%s %s", t.Timezone, t.Expression)
	entryId, err := c.scheduler.AddFunc(expression, f)
	if err != nil {
		log.Errorf("[scheduler] add func err: %v", err)
		return err
	}

	// 记录 entryId
	c.PutTaskEntryId(t.Tid, entryId)
	log.Infof("[scheduler] 添加定时任务 %s-%s, 表达式: %v, with entryID: %v", t.Tid, t.Name, expression, entryId)

	return nil
}

func addScheduler() *cron.Cron {
	// 创建对应的时区定时器
	optLogs := cron.WithLogger(
		cron.VerbosePrintfLogger(
			standlog.New(os.Stdout, "[Cron]: ", standlog.LstdFlags)))

	optParser := cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	))

	scheduler := cron.New(optLogs, optParser)
	scheduler.Start() // 重复 start 也没事 幂等
	return scheduler
}

//InitScheduler 初始化定时器以及任务
func InitScheduler() error {
	// 初始化 cronScheduler
	cronScheduler = NewCronScheduler(addScheduler())
	// 将任务加入时区定时器
	tasks := make([]Task, 0)
	cursor, err := TaskCol.Find(context.Background(), bson.M{})
	if err != nil {
		log.Errorf("[scheduler] get all tasks err: %v", err)
		return err
	}
	if err = cursor.All(context.Background(), &tasks); err != nil {
		log.Errorf("[message] 加载数据失败: %v", err)
		return err
	}

	for i := 0; i < len(tasks); i++ {
		// 默认清空之前的状态
		t := tasks[i]
		if err := NewInitPutTask(t); err != nil {
			log.Fatalf("[scheduler] error to init the task with error %v", err)
		}
	}
	return nil
}

func StopScheduler() context.Context {
	return cronScheduler.scheduler.Stop()
}

//SubCronJob 订阅频道用于更新任务
func SubCronJob(c chan os.Signal) {
	RCache.Subscribe(c)
}
