package controller

import (
	"clock/v3/master/param"
	"clock/v3/storage"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

//Ping 健康检查
func Ping(c echo.Context) (err error) {
	resp := param.BuildResp()
	// 1、ping mongo
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := storage.Mdb.Client.Ping(ctx, nil); err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[health] mongo ping err: %v", err)
		return c.JSON(http.StatusOK, resp)
	}
	log.Debugf("[health] mongo ping success")
	// 2、ping redis
	pong, err := storage.RCache.Ping()
	if err != nil {
		resp.Code = param.Failed
		resp.Msg = fmt.Sprintf("[db] redis ping err: %v", err)
		return c.JSON(http.StatusOK, resp)
	}
	log.Debugf("[db] rdb ping res is %s", pong)
	// 3、echo return
	return c.JSON(http.StatusOK, resp)

}
