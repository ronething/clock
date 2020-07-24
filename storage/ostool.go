package storage

import (
	"bytes"
	"clock/config"
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/sirupsen/logrus"
)

func GenGuid(length int) (string, error) {
	u, e := uuid.NewV4()

	if e != nil {
		logrus.Error(e)
		return "", e
	}

	guid := u.String()
	guid = strings.Replace(guid, "-", "", -1)

	return guid[0:length], nil
}

//TryLock redis 分布式锁
func TryLock(t Task) (chan int, error) {
	ctx := context.Background()
	rid, _ := GenGuid(8)         // 生成 uuid 和 task name 组合作为 key 的 value
	jobDone := make(chan int, 1) // 任务完成的时候 done <- 1
	key := fmt.Sprintf("%s:%s", config.Config.GetString("lease.prefix"), t.Name)
	res, err := Rdb.Exists(ctx, key).Result()
	if err != nil {
		logrus.Errorf("[ostool] redis exists key err: %v", err)
		return nil, err
	}
	if res == 1 { // key 存在 等待下次调度
		logrus.Debugf("[ostool] redis key %s 存在，等待下次调度", key)
		return nil, WaitForNextScheduleErr
	} else {
		res, err := Rdb.SetNX(ctx, key, t.Name+rid, config.Config.GetDuration("lease.expire")*time.Second).Result()
		if err != nil {
			logrus.Errorf("[ostool] redis setnx key err: %v", err)
			return nil, err
		}
		if res == true { // 设置成功
			// DONE: 续租
			go leaseJob(jobDone, t.Name, rid)
		} else {
			return nil, WaitForNextScheduleErr // 等待下次调度
		}
	}
	return jobDone, nil
}

// leaseJob 续租 redis job key
func leaseJob(done chan int, name, rid string) {
	key := fmt.Sprintf("%s:%s", config.Config.GetString("lease.prefix"), name)
	leaseTime := config.Config.GetDuration("lease.per") * time.Second
	timer := time.NewTimer(leaseTime) // 默认 20 秒续租一次
	ctx := context.Background()
	for {
		select {
		case <-done:
			logrus.Debugf("[ostool] task %s 任务完成", name)
			res, _ := Rdb.Get(ctx, key).Result()
			logrus.Debugf("[ostool] %s res is %s", key, res)
			if res == name+rid { // 结果校验 防止误删
				logrus.Debugf("[ostool] 删除 key %s", key)
				Rdb.Del(ctx, key)
			}
			goto End
		case <-timer.C:
			logrus.Debugf("[ostool] 续租 %s", name)
			Rdb.Expire(context.Background(), key, config.Config.GetDuration("lease.expire")*time.Second)
			timer.Reset(leaseTime)
		}
	}
End:
	timer.Stop() // 暂停计时器
}

func saveLog(t Task, stdOut, stdErr *bytes.Buffer, start, end int64) {
	sErr := fmt.Sprintf("%s stderr is : %s", t.Name, stdErr.String())
	sOut := fmt.Sprintf("%s stdout is : %s", t.Name, stdOut.String())

	// 回写日志状态
	if t.LogEnable {
		id := primitive.NewObjectID()
		l := TaskLog{
			Id:       id,
			Lid:      id.Hex(),
			Tid:      t.Tid,
			StdOut:   stdOut.String(),
			StdErr:   stdErr.String(),
			StartAt:  start,
			EndAt:    end,
			CreateAt: time.Now().Unix(),
		}
		// TODO: MDB Batch
		res, err := TaskLogCol.InsertOne(context.Background(), &l)
		if err != nil {
			logrus.Errorf("[ostool] insert log to db err: %v", err)
			return
		}
		logrus.Debugf("[ostool] insert id: %v", res.InsertedID)

	}

	sendMessage(sErr)
	sendMessage(sOut)
}

func sendMessage(msg string) {
	select {
	case Messenger.Channel <- msg:
	default:
		logrus.Warnf("the Messenger is full now ")
	}
}
