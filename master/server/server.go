package server

import (
	"clock/master/controller"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func addApi(e *echo.Echo) {
	// 增加cors 中间件
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// 使用jwt token验证
	v1 := e.Group("/v1")
	{
		//v1.Use(middleware.JWTWithConfig(createJWTConfig())) 暂时取消登录中间件
		t := v1.Group("/task")
		{
			t.GET("", controller.GetTasks)
			t.GET("/:tid", controller.GetTask)
			t.PUT("", controller.PutTask)
			t.GET("/run", controller.RunTask)
			t.DELETE("/:tid", controller.DeleteTask)
			t.GET("/status", controller.GetTaskStatus)
		}

		l := v1.Group("/log")
		{
			l.GET("", controller.GetLogs)
			l.DELETE("", controller.DeleteLogs)
		}

		// 消息中心
		m := v1.Group("/message")
		{
			m.GET("", controller.GetMessages)
		}

		s := v1.Group("/system")
		{
			s.GET("/load", controller.GetLoadAverage)
			s.GET("/mem", controller.GetMemoryUsage)
			s.GET("/cpu", controller.GetCpuUsage)
		}
	}

}

func CreateEngine() (*echo.Echo, error) {
	e := echo.New()

	addApi(e)

	return e, nil
}
