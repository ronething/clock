package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
)

func RunBashTask(t Task) error {
	log.Debugf("[ostool] execute job name: %s, tid: %s", t.Name, t.Tid)
	// DONE: 分布式锁
	jobDone, err := RCache.TryLock(t)
	if err != nil {
		log.Errorf("[ostool] 加锁失败: %v", err)
		return err // 在 defer 之前 return，并不会执行 defer 的内容
	}

	var stdOutBuf bytes.Buffer
	var stdErrBuf bytes.Buffer
	start := time.Now().Unix()

	defer func() {
		jobDone <- 1 // 完成 job
		end := time.Now().Unix()
		t.UpdateAt = time.Now().Unix()
		log.Debugf("[%v] - now finish task [%s]", t.Tid, t.Name)
		res, err := TaskCol.UpdateOne(context.Background(), bson.D{{"_id", t.Id}}, bson.D{
			{
				"$set",
				&t,
			},
		})
		if err != nil {
			log.Errorf("[ostool] update task %s err: %v", t.Tid, err)
			return
		}
		log.Debugf("[ostool] update task %s, match: %v, modify: %v",
			t.Tid, res.MatchedCount, res.ModifiedCount)
		go saveLog(t, &stdOutBuf, &stdErrBuf, start, end)
	}()

	command := t.Payload["command"].(string)

	if command == "" {
		return errors.New("please do not input the empty command")
	}

	log.Debugf("[%v] - now will run the task [%s]", t.Tid, t.Name)

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
			log.Error(err.Error())
			stdErrBuf.WriteString(err.Error())
			return err
		case <-done:
			return nil
		}
	}

	e := c.Run()
	if e != nil {
		log.Errorf("[ostool] run task err: %v", e)
		// 写入错误信息
		stdErrBuf.WriteString(e.Error())
		return e
	}

	return nil

}

//RunHTTPTask 请求 http 执行任务
func RunHTTPTask(t Task, now string) error {
	log.Debugf("[ostool] execute job name: %s, tid: %s", t.Name, t.Tid)
	jobDone, err := RCache.TryLock(t)
	if err != nil {
		return err
	}
	defer func() {
		jobDone <- 1
	}()

	// 拼接 url
	endpoint := t.Payload["endpoint"].(string)
	prefix := t.Payload["prefix"].(string)
	url := endpoint + prefix
	log.Debugf("[executor] url is %s", url)
	data, ok := t.Payload["data"].(map[string]interface{})
	if !ok {
		data = make(map[string]interface{})
		//return errors.New("[executor] data 不是 map[string]interface{} 类型")
	}
	log.Debugf("[executor] data is %v", data)
	// 加入 delay 时间到 data 中
	data["delay"] = now
	b, err := json.Marshal(data)
	if err != nil {
		log.Errorf("[executor] data 序列化失败 %v", err)
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
	t.UpdateAt = time.Now().Unix()
	log.Debugf("[%v] - now finish task [%s]", t.Tid, t.Name)
	res, err := TaskCol.UpdateOne(context.Background(), bson.D{{"_id", t.Id}}, bson.D{
		{
			"$set",
			&t,
		},
	})
	if err != nil {
		log.Errorf("[ostool] update task %s err: %v", t.Tid, err)
		return err
	}
	log.Debugf("[ostool] update task %s, match: %v, modify: %v",
		t.Tid, res.MatchedCount, res.ModifiedCount)
	go saveLog(t, bytes.NewBuffer([]byte(resp.Body)), bytes.NewBuffer([]byte("")), start, end)

	return nil
}
