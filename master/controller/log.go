package controller

import (
	"fmt"
	"net/http"

	"clock/v3/master/param"
	"clock/v3/storage"

	"github.com/labstack/echo/v4"
)

//GetLogs 列表
func GetLogs(c echo.Context) (err error) {
	var query storage.LogQuery

	resp := param.BuildResp()

	if err := c.Bind(&query); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[get logs] error to get the query param with: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	logs, err := storage.GetLogs(&query)
	if err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[get logs] error to get the logs: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	page := param.ListResponse{
		Items:     logs,
		PageQuery: query,
	}

	resp.Data = page

	return c.JSON(http.StatusOK, resp)
}

//DeleteLogs 清除多少天之前的日志
func DeleteLogs(c echo.Context) error {
	var query storage.LogQuery

	resp := param.BuildResp()

	if err := c.Bind(&query); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[delete logs] error to get the query param with: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	// 异步执行
	go storage.DeleteLogs(&query)
	return c.JSON(http.StatusOK, resp)
}
