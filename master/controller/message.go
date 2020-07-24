package controller

import (
	"clock/master/param"
	"context"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"clock/storage"
)

//GetMessages
func GetMessages(c echo.Context) error {
	resp := param.BuildResp()

	counters, err := getMessages()
	if err != nil {
		resp.Msg = fmt.Sprintf("[get messages] error to get the counts: %v", err)
		return c.JSON(http.StatusBadRequest, resp)
	}
	resp.Data = counters

	return c.JSON(http.StatusOK, resp)
}

func getMessages() ([]storage.TaskCounter, error) {
	tasks := make([]storage.Task, 0)
	counters := make([]storage.TaskCounter, 0)

	cursor, err := storage.TaskCol.Find(context.Background(), bson.M{})
	if err != nil {
		logrus.Errorf("[message] get all tasks err: %v", err)
		return counters, err // 返回 counters 而不返回 nil，这样可以判断 length 是 0
	}
	if err = cursor.All(context.Background(), &tasks); err != nil {
		logrus.Errorf("[message] 加载数据失败: %v", err)
		return counters, err
	}

	pending := storage.TaskCounter{
		Title: "当前等待",
		Icon:  "md-clock",
		Count: 0,
		Color: "#ff9900",
	}

	start := storage.TaskCounter{
		Title: "正在运行",
		Icon:  "md-play",
		Count: 0,
		Color: "#19be6b",
	}

	success := storage.TaskCounter{
		Title: "运行成功",
		Icon:  "md-done-all",
		Count: 0,
		Color: "#2d8cf0",
	}

	failure := storage.TaskCounter{
		Title: "运行失败",
		Icon:  "md-close",
		Count: 0,
		Color: "#ed3f14",
	}

	for _, t := range tasks {
		switch t.Status {
		case storage.PENDING:
			pending.Count += 1
		case storage.START:
			start.Count += 1
		case storage.SUCCESS:
			success.Count += 1
		case storage.FAILURE:
			failure.Count += 1
		default:
			logrus.Warnf("find the unknown tasks %v : status %v ", t.Tid, t.Status)
		}
	}

	counters = append(counters, pending, start, success, failure)
	return counters, nil
}
