package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

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

func RunSingleTask(t Task) error {
	logrus.Debugf("[ostool] execute job %s", t.Name)
	// DONE: 分布式锁
	ctx := context.Background()
	rid, _ := GenGuid(8)         // 生成 uuid 和 task name 组合作为 key 的 value
	jobDone := make(chan int, 1) // 任务完成的时候 done <- 1
	key := fmt.Sprintf("cron:job:%s", t.Name)
	res, err := Rdb.Exists(ctx, key).Result()
	if err != nil {
		logrus.Errorf("[ostool] redis exists key err: %v", err)
		return err
	}
	if res == 1 { // key 存在 等待下次调度
		logrus.Debugf("[ostool] redis key %s 存在，等待下次调度", key)
		return nil
	} else {
		res, err := Rdb.SetNX(ctx, key, t.Name+rid, 120*time.Second).Result()
		if err != nil {
			logrus.Errorf("[ostool] redis setnx key err: %v", err)
			return err
		}
		if res == true { // 设置成功
			// DONE: 续租
			go leaseJob(jobDone, t.Name, rid)
		} else {
			return nil // 等待下次调度
		}
	}

	var stdOutBuf bytes.Buffer
	var stdErrBuf bytes.Buffer
	start := time.Now().Unix()

	// 保存最后状态
	t.Status = START
	defer func() {
		end := time.Now().Unix()
		t.UpdateAt = time.Now().Unix()
		logrus.Debugf("[%v] - now finish task [%s]", t.Tid, t.Name)
		Db.Save(&t)
		saveLog(t, stdOutBuf, stdErrBuf, start, end)
	}()

	if t.Command == "" {
		t.Status = FAILURE
		return errors.New("please do not input the empty command")
	}

	logrus.Debugf("[%v] - now will run the task [%s]", t.Tid, t.Name)
	Db.Save(&t)

	c := exec.Command("/bin/bash", "-c", t.Command)
	c.Stdout = &stdOutBuf
	c.Stderr = &stdErrBuf

	if t.TimeOut > 0 {
		timeout := time.After(time.Duration(t.TimeOut) * time.Second)
		done := make(chan error, 1)

		go func() {
			done <- c.Run()
		}()

		select {
		case <-timeout:
			// TODO: 新建进程组 kill
			_ = c.Process.Kill()
			err := errors.New(fmt.Sprintf("cmd %s reach to timeout limit", t.Command))
			logrus.Errorln(err.Error())
			stdErrBuf.WriteString(err.Error())
			jobDone <- 1
			return err
		case <-done:
			jobDone <- 1
			return nil
		}
	}

	e := c.Run()
	if e != nil {
		logrus.Error(e)
		// 写入错误信息
		stdErrBuf.WriteString(e.Error())
		t.Status = FAILURE
		return e
	}
	jobDone <- 1

	t.Status = SUCCESS
	return nil

}

// leaseJob 续租 redis job key
func leaseJob(done chan int, name, rid string) {
	key := fmt.Sprintf("cron:job:%s", name)
	leaseTime := 20 * time.Second
	timer := time.NewTimer(leaseTime) // 20 秒续租一次
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
			Rdb.Expire(context.Background(), key, 120*time.Second)
			timer.Reset(leaseTime)
		}
	}
End:
	timer.Stop() // 暂停计时器
}

func saveLog(t Task, stdOut, stdErr bytes.Buffer, start, end int64) {
	sErr := fmt.Sprintf("%s stderr is : %s", t.Name, stdErr.String())
	sOut := fmt.Sprintf("%s stdout is : %s", t.Name, stdOut.String())

	// 回写日志状态
	if t.LogEnable {
		lid, _ := GenGuid(8)
		l := TaskLog{
			Lid:      lid,
			Tid:      t.Tid,
			StdOut:   stdOut.String(),
			StdErr:   stdErr.String(),
			StartAt:  start,
			EndAt:    end,
			CreateAt: time.Now().Unix(),
		}
		Db.Save(&l)
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
