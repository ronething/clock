package server

import (
	echoprometheus "github.com/globocom/echo-prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func addMiddleware(e *echo.Echo) {
	// 增加 cors 中间件
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
}

//CreateEngine echo
func CreateEngine() (*echo.Echo, error) {
	e := echo.New()

	addMiddleware(e)
	addMetrics(e)

	// TODO: debug 模式输出路由

	return e, nil
}

//addMetrics 增加监控
func addMetrics(e *echo.Echo) {
	// 这里 subsystem 要下划线 _, 不能是 - 否则 metrics 会看不到对应的
	e.Use(echoprometheus.MetricsMiddleware()) // TODO:不使用自定义的指标
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}
