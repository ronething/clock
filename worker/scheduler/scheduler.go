// author: ashing
// time: 2020/7/19 2:03 下午
// mail: axingfly@gmail.com
// Less is more.

package scheduler

import (
	"clock/storage"
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
	"log"
	"os"
	"time"
)

var scheduler *cron.Cron

func NewScheduler() {
	optLogs := cron.WithLogger(
		cron.VerbosePrintfLogger(
			log.New(os.Stdout, "[Cron]: ", log.LstdFlags)))

	optParser := cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	))

	scheduler = cron.New(optLogs, optParser)
	var tasks []storage.Task
	storage.Db.Find(&tasks)

	if len(tasks) > 0 {
		for _, t := range tasks {
			// 默认清空之前的状态
			t.Status = storage.PENDING
			t.EntryId = -1
			t.UpdateAt = time.Now().Unix()
			if err := PutTask(t); err != nil {
				logrus.Fatalf("[scheduler] error to init the task with error %v", err)
			}
		}
	}

	logrus.Info("[scheduler] start the ticker")
	scheduler.Start()
}

func PutTask(t storage.Task) error {
	// 移除并重新启用
	if t.EntryId > 0 { // c.EntryId == -1 || c.EntryId == 0 , -1 表示 disable、0 表示新增
		scheduler.Remove(cron.EntryID(t.EntryId))
	}
	t.UpdateAt = time.Now().Unix()
	if t.Disable {
		t.EntryId = -1
	} else {
		err := AddScheduler(&t)
		if err != nil {
			logrus.Errorf("[put task] error with %v", err)
		}
	}

	return storage.Db.Save(&t).Error

}

//DeleteTask 调度器删除对应任务
func DeleteTask(eid int) error {
	scheduler.Remove(cron.EntryID(eid)) // 删除的是 entry id，不是 task id
	return nil
}

// 添加任务，需要传入指针,方便修改值
func AddScheduler(t *storage.Task) error {
	f := func() {
		if e := storage.RunTask(t.Tid); e != nil {
			logrus.Error(e)
		}
	}

	entryId, e := scheduler.AddFunc(t.Expression, f)
	if e != nil {
		logrus.Error(e)
		return e
	}

	t.EntryId = int(entryId) // 改变了 entryId, return 回去之后再 DB.Save()
	logrus.Infof("[add scheduler] add the job of %s , with entry id %v", t.Name, t.EntryId)

	return nil
}

//SubCronJob 订阅频道用于更新任务
func SubCronJob() {
	ctx := context.Background()
	channelName := "cron"
	pubsub := storage.Rdb.Subscribe(ctx, channelName)

	defer pubsub.Close()

	for {
		// ReceiveTimeout is a low level API. Use ReceiveMessage instead.
		msgi, err := pubsub.Receive(ctx)
		if err != nil {
			goto End
		}
		switch msg := msgi.(type) {
		case *redis.Subscription:
			logrus.Infof("成功订阅 %s", channelName)
		case *redis.Message:
			t := storage.TaskEvent{}
			if err := msgpack.Unmarshal([]byte(msg.Payload), &t); err != nil {
				logrus.Errorf("[scheduler] msg 解析发生错误")
			}
			logrus.Debugf("[scheduler] taskEvent is %v", t)
			if t.Event == storage.PUT {
				if err := PutTask(t.Task); err != nil {
					logrus.Errorf("[scheduler] PUT 事件失败 %s", err.Error())
				}
			} else if t.Event == storage.DEL {
				if err := DeleteTask(t.Task.EntryId); err != nil {
					logrus.Errorf("[scheduler] DEL 事件失败 %s", err.Error())
				}
			} else { // 后续拓展 不过应该不需要

			}
		default:
			logrus.Warnf("[scheduler] 默认是什么鬼 %v", msg)
			goto End
		}
	}
End:
	logrus.Errorf("[scheduler] 订阅消息发生错误")
}
