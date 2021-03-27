package server

import (
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/labstack/echo/v4"
)

//LogrusLogger 自定义 logrus 日志 中间件
func LogrusLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		res := c.Response()
		start := time.Now()

		var err error
		// 先 next，然后再回来进行计算时间
		if err = next(c); err != nil {
			c.Error(err)
		}
		stop := time.Now()

		logrus.WithField("prefix", "echo").Infof("%s %s %s %3d %s %s %s %s",
			c.RealIP(),
			req.Method,
			req.RequestURI,
			res.Status,
			strconv.FormatInt(res.Size, 10),
			stop.Sub(start).String(),
			req.Referer(),
			req.UserAgent(),
		)
		return err
	}
}
