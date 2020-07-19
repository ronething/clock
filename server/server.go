package server

import (
	"clock/param"
	"strings"

	"clock/controller"
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

		u := v1.Group("/login")
		{
			u.POST("", controller.Login)
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

func createJWTConfig() middleware.JWTConfig {
	d := middleware.DefaultJWTConfig

	d.SigningKey = []byte(param.WebJwt)
	d.TokenLookup = "header:token"
	d.AuthScheme = "duckduckgo"

	filterUri := []string{"webapp", "js", "css"}

	d.Skipper = func(c echo.Context) bool {
		uri := c.Request().RequestURI
		if strings.Contains(uri, "/v1/login") {
			return true
		}

		for _, v := range filterUri {
			if strings.Contains(uri, v) {
				return true
			}
		}

		if strings.Contains(uri, "/v1/task/status") {
			return true
		}

		return false
	}

	return d
}

func CreateEngine() (*echo.Echo, error) {
	e := echo.New()

	addApi(e)

	return e, nil
}
