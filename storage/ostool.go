package storage

import (
	"bytes"
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
	var stdOutBuf bytes.Buffer
	var stdErrBuf bytes.Buffer

	// 保存最后状态
	t.Status = START
	defer func() {
		t.UpdateAt = time.Now().Unix()
		logrus.Debugf("[%v] - now finish task [%s]", t.Tid, t.Name)
		Db.Save(&t)
		saveLog(t, stdOutBuf, stdErrBuf)
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
			_ = c.Process.Kill()
			logrus.Errorf("cmd %s reach to timeout limit", t.Command)
		case <-done:
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

	t.Status = SUCCESS
	return nil

}


func saveLog(t Task, stdOut, stdErr bytes.Buffer) {
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
