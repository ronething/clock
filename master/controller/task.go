package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"clock/v3/master/param"
	"clock/v3/storage"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
)

func getPathInt(c echo.Context, key string) (int, error) {
	value := c.Param(key)
	if value == "" {
		return 0, errors.New("can not find any param from query ")
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return intValue, nil
}

func getQueryInt(c echo.Context, key string) (int, error) {
	value := c.QueryParam(key)
	if value == "" {
		return 0, errors.New("can not find any param from query ")
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return intValue, nil
}

// 列表
func GetTasks(c echo.Context) (err error) {
	var query storage.TaskQuery

	resp := param.BuildResp()

	if err := c.Bind(&query); err != nil {
		resp.Msg = fmt.Sprintf("[get tasks] error to get the task param with: %v", err)
		logrus.Error(resp.Msg)

		return c.JSON(http.StatusBadRequest, resp)
	}

	tasks, err := storage.GetTasks(&query)
	if err != nil {
		resp.Msg = fmt.Sprintf("[get tasks] error to get the task from db with: %v", err)
		logrus.Error(resp.Msg)

		return c.JSON(http.StatusBadRequest, resp)
	}

	page := param.ListResponse{
		Items:     tasks,
		PageQuery: query,
	}

	resp.Data = page
	return c.JSON(http.StatusOK, resp)
}

// 得到某一个
func GetTask(c echo.Context) (err error) {
	resp := param.BuildResp()

	taskId := c.Param("tid") // ObjectID

	t, e := storage.GetTask(taskId)
	if e != nil {
		resp.Msg = fmt.Sprintf("[get task] error to query the task with: %v", err)
		logrus.Error(resp.Msg)

		return c.JSON(http.StatusNotFound, resp)
	}

	resp.Data = t

	return c.JSON(http.StatusOK, resp)
}

// 更新或新增一个task
func PutTask(c echo.Context) error {
	resp := param.BuildResp()

	t := storage.Task{}

	if err := c.Bind(&t); err != nil {
		resp.Msg = fmt.Sprintf("[put task] invalidate param found: %v", err)
		logrus.Error(resp.Msg)

		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := storage.PutTask(&t); err != nil {
		resp.Msg = fmt.Sprintf("[put task] error to query task from db with: %v", err)
		logrus.Error(resp.Msg)

		return c.JSON(http.StatusBadRequest, resp)
	}

	resp.Data = t.Tid
	return c.JSON(http.StatusOK, resp)
}

// GetTaskStatus 得到当前任务状态 websocket
func GetTaskStatus(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			time.Sleep(50 * time.Millisecond)
			select {
			case msg := <-storage.Messenger.Channel:
				websocket.Message.Send(ws, msg)
			default:
				continue
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func RunTask(c echo.Context) error {
	taskId := c.QueryParam("tid")
	resp := param.BuildResp()

	if err := storage.RunTask(taskId); err != nil {
		resp.Msg = fmt.Sprintf("[run task] error run task with: %v", err)
		logrus.Error(resp.Msg)

		return c.JSON(http.StatusBadRequest, resp)
	}

	return c.JSON(http.StatusOK, resp)
}

func DeleteTask(c echo.Context) error {
	taskId := c.Param("tid")

	resp := param.BuildResp()

	if err := storage.DeleteTask(taskId); err != nil {
		resp.Msg = fmt.Sprintf("[delete task] error to delete task with:%v", err)
		logrus.Error(err)

		return c.JSON(http.StatusBadRequest, resp)
	}

	return c.JSON(http.StatusOK, resp)
}
