package controller

import (
	"clock/v3/master/param"
	"clock/v3/storage"
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

//GetAllSupportTimezone
func GetAllSupportTimezone(c echo.Context) (err error) {
	resp := param.BuildResp()
	ctx := context.Background()
	allSupportTimezone, err := storage.GetAllSupportTimezone(ctx)
	if err != nil {
		resp.Msg = fmt.Sprintf("[get tasks] error to get the task from db with: %v", err)
		resp.Code = param.Failed
		return c.JSON(http.StatusOK, resp)
	}

	resp.Data = allSupportTimezone
	return c.JSON(http.StatusOK, resp)
}

//CreateSupportTimezone
func CreateSupportTimezone(c echo.Context) (err error) {
	resp := param.BuildResp()

	t := storage.Timezone{}
	if err := c.Bind(&t); err != nil {
		resp.Msg = fmt.Sprintf("[create timezone] invalidate param found: %v", err)
		resp.Code = param.Failed
		return c.JSON(http.StatusOK, resp)
	}

	ctx := context.Background()
	if err := storage.CreateSupportTimezone(ctx, &t); err != nil {
		resp.Msg = fmt.Sprintf("[create timezone] err: %v", err)
		resp.Code = param.Failed
		return c.JSON(http.StatusOK, resp)
	}

	resp.Data = t.Tid
	return c.JSON(http.StatusOK, resp)
}

//DeleteSupportTimezone
func DeleteSupportTimezone(c echo.Context) (err error) {
	timezoneId := c.Param("tid")

	resp := param.BuildResp()
	ctx := context.Background()
	if err := storage.DeleteSupportTimezone(ctx, timezoneId); err != nil {
		resp.Msg = fmt.Sprintf("[delete timezone] error to delete timezone with:%v", err)
		log.Error(err)
		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(http.StatusOK, resp)
}

//GetOneSupportTimezone
func GetOneSupportTimezone(c echo.Context) (err error) {
	resp := param.BuildResp()

	timezoneId := c.Param("tid")

	ctx := context.Background()
	t, err := storage.GetSupportTimezone(ctx, timezoneId)
	if err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[get timezone] error to query the timezone with: %v", err)
		return c.JSON(http.StatusOK, resp)
	}

	resp.Data = t

	return c.JSON(http.StatusOK, resp)
}
