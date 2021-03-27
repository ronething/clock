package controller

import (
	"clock/v3/master/param"
	"clock/v3/storage"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

// 列表
func GetTasks(c echo.Context) (err error) {
	var query storage.TaskQuery

	resp := param.BuildResp()

	if err := c.Bind(&query); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[get tasks] error to get the task param with: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	tasks, err := storage.GetTasks(&query)
	if err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[get tasks] error to get the task from db with: %v", err)
		return c.JSON(http.StatusOK, resp)
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

	t, err := storage.GetTask(taskId)
	if err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[get task] error to query the task with: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	resp.Data = t

	return c.JSON(http.StatusOK, resp)
}

//PostTask 新增一个 task
func PostTask(c echo.Context) error {

	resp := param.BuildResp()

	t := storage.Task{}
	if err := c.Bind(&t); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[post task] invalidate param found: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	if err := storage.PostTask(&t); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[post task] err: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	resp.Data = t.Tid
	return c.JSON(http.StatusOK, resp)
}

func RunTask(c echo.Context) error {
	taskId := c.Param("tid")
	resp := param.BuildResp()

	if err := storage.RunTask(taskId); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[run task] error run task with: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(http.StatusOK, resp)
}

func DeleteTask(c echo.Context) error {
	taskId := c.Param("tid")

	resp := param.BuildResp()

	if err := storage.DeleteTask(taskId); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[delete task] error to delete task with:%v", err)
		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(http.StatusOK, resp)
}

//DisableTask 禁用/启动任务
func DisableTask(c echo.Context) error {
	taskId := c.Param("tid")
	resp := param.BuildResp()

	t := param.DisableTask{}

	if err := c.Bind(&t); err != nil {
		resp.Msg = fmt.Sprintf("[disable task] error to query task from db with: %v", err)
		resp.Code = param.Failed
		return c.JSON(http.StatusOK, resp)
	}

	if err := storage.DisableTask(taskId, storage.Struct2bsonD(t, "json").Map()); err != nil {
		resp.Msg = fmt.Sprintf("[disable task] error to query task from db with: %v", err)
		log.Error(resp.Msg)

		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(http.StatusOK, resp)
}

//NewSpecTask 修改任务表达式以及时区
func NewSpecTask(c echo.Context) error {
	taskId := c.Param("tid")
	resp := param.BuildResp()

	t := param.NewSpecTask{}

	if err := c.Bind(&t); err != nil {
		resp.Msg = fmt.Sprintf("[spec task] error to query task from db with: %v", err)
		resp.Code = param.Failed
		return c.JSON(http.StatusOK, resp)
	}

	if err := storage.ModifyTask(taskId, storage.Struct2bsonD(t, "json").Map()); err != nil {
		resp.Msg = fmt.Sprintf("[spec task] error to query task from db with: %v", err)
		log.Error(resp.Msg)
		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(http.StatusOK, resp)
}
