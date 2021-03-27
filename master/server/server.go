package server

import (
	"clock/v3/master/controller"

	echoprometheus "github.com/globocom/echo-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func addMiddleware(e *echo.Echo) {
	// 增加 cors 中间件
	e.Use(middleware.CORS())
	//e.Use(middleware.Logger())
	e.Use(LogrusLogger)
	e.Use(middleware.Recover())
}

func addApi(e *echo.Echo) {

	v1 := e.Group("/v1")
	{
		//v1.Use(middleware.JWTWithConfig(createJWTConfig())) 暂时取消登录中间件
		t := v1.Group("/task")
		{
			t.GET("", controller.GetTasks)
			t.GET("/:tid", controller.GetTask)
			t.POST("", controller.PostTask)
			t.POST("/run/:tid", controller.RunTask)
			t.DELETE("/:tid", controller.DeleteTask)
			t.PUT("/:tid/disable", controller.DisableTask)
			t.PUT("/:tid/new_spec", controller.NewSpecTask)
		}

		tz := v1.Group("/timezone")
		{
			tz.GET("", controller.GetAllSupportTimezone)
			tz.GET("/:tid", controller.GetOneSupportTimezone)
			tz.POST("", controller.CreateSupportTimezone)
			tz.DELETE("/:tid", controller.DeleteSupportTimezone)
		}

		l := v1.Group("/log")
		{
			l.GET("", controller.GetLogs)
			l.DELETE("", controller.DeleteLogs)
		}

	}

}

//addHealthCheck
func addHealthCheck(e *echo.Echo) {
	e.GET("/ping", controller.Ping)
}

//addMetrics 增加监控
func addMetrics(e *echo.Echo) {
	// 这里 subsystem 要下划线 _, 不能是 - 否则 metrics 会看不到对应的
	e.Use(echoprometheus.MetricsMiddleware())
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}

//CreateEngine echo 实例
func CreateEngine() (*echo.Echo, error) {
	e := echo.New()

	addMiddleware(e)
	addMetrics(e)
	addHealthCheck(e)
	addApi(e)

	return e, nil
}
