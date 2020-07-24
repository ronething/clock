package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

func RunBashTask(t Task) error {
	logrus.Debugf("[ostool] execute job %s", t.Name)
	// DONE: 分布式锁
	jobDone, err := TryLock(t)
	if err != nil {
		logrus.Errorf("[ostool] 加锁失败: %v", err)
		return err // 在 defer 之前 return，并不会执行 defer 的内容
	}

	var stdOutBuf bytes.Buffer
	var stdErrBuf bytes.Buffer
	start := time.Now().Unix()

	// 保存最后状态
	t.Status = START
	defer func() {
		jobDone <- 1 // 完成 job
		end := time.Now().Unix()
		t.UpdateAt = time.Now().Unix()
		logrus.Debugf("[%v] - now finish task [%s]", t.Tid, t.Name)
		res, err := TaskCol.UpdateOne(context.Background(), bson.D{{"_id", t.Id}}, bson.D{
			{
				"$set",
				&t,
			},
		})
		if err != nil {
			logrus.Errorf("[ostool] update task %s err: %v", t.Tid, err)
			return
		}
		logrus.Debugf("[ostool] update task %s, match: %v, modify: %v",
			t.Tid, res.MatchedCount, res.ModifiedCount)
		saveLog(t, &stdOutBuf, &stdErrBuf, start, end)
	}()

	command := t.Payload["command"].(string)

	if command == "" {
		t.Status = FAILURE
		return errors.New("please do not input the empty command")
	}

	logrus.Debugf("[%v] - now will run the task [%s]", t.Tid, t.Name)

	c := exec.Command("/bin/bash", "-c", command)
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
			err := errors.New(fmt.Sprintf("cmd %s reach to timeout limit", command))
			logrus.Errorln(err.Error())
			stdErrBuf.WriteString(err.Error())
			return err
		case <-done:
			return nil
		}
	}

	e := c.Run()
	if e != nil {
		logrus.Errorf("[ostool] run task err: %v", e)
		// 写入错误信息
		stdErrBuf.WriteString(e.Error())
		t.Status = FAILURE
		return e
	}

	t.Status = SUCCESS
	return nil

}

//RunHTTPTask 请求 http 执行任务
func RunHTTPTask(t Task) error {
	logrus.Debugf("run task %s", t.Tid)
	jobDone, err := TryLock(t)
	if err != nil {
		return err
	}
	defer func() {
		jobDone <- 1
	}()

	endpoint := t.Payload["endpoint"].(string)
	prefix := t.Payload["prefix"].(string)
	url := endpoint + prefix
	logrus.Debugf("[executor] url is %s", url)
	data, ok := t.Payload["data"].(map[string]interface{})
	if !ok {
		return errors.New("[executor] data 不是 map[string]interface{} 类型")
	}
	logrus.Debugf("[executor] data is %v", data)
	b, err := json.Marshal(data)
	if err != nil {
		logrus.Errorf("[executor] data 序列化失败", err)
		return err
	}
	method := t.Payload["method"].(string)
	start := time.Now().Unix()
	var resp ResponseWrapper
	switch method {
	case "GET":
		resp = Get(url, t.TimeOut)
	case "POST":
		resp = PostJson(url, string(b), t.TimeOut)
	}

	end := time.Now().Unix()
	t.Status = SUCCESS
	t.UpdateAt = time.Now().Unix()
	logrus.Debugf("[%v] - now finish task [%s]", t.Tid, t.Name)
	res, err := TaskCol.UpdateOne(context.Background(), bson.D{{"_id", t.Id}}, bson.D{
		{
			"$set",
			&t,
		},
	})
	if err != nil {
		logrus.Errorf("[ostool] update task %s err: %v", t.Tid, err)
		return err
	}
	logrus.Debugf("[ostool] update task %s, match: %v, modify: %v",
		t.Tid, res.MatchedCount, res.ModifiedCount)
	saveLog(t, bytes.NewBuffer([]byte(resp.Body)), bytes.NewBuffer([]byte("")), start, end)

	return nil
}
