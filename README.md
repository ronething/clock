## Clock
基于 go cron 和 redis 实现分布式任务调度(支持多 worker，多 master)

## 地址
https://github.com/ronething/clock

## 环境
* 后端
    * go 1.13+
    * [cron](https://github.com/robfig/cron) - 定时调度器
    * [echo](https://github.com/labstack/echo) - web framework
    * [gorm](https://github.com/jinzhu/gorm) - database orm
    * [go-redis](https://github.com/go-redis/redis)  - redis client
    * [msgpack](https://github.com/vmihailenco/msgpack) 序列化

## 使用
### 直接使用
下载git上的release列表，根据系统下载相应的二进制文件，使用命令
```
# 分别在 master 和 worker 目录下进行构建
cd master && go build
./master -c ../config/dev.yaml

cd worker && go build
./worker -c ../config/dev.yaml
```

### Api

- 获取所有任务

`GET /v1/task`

- 获取单个任务

`GET /v1/task/:tid`

- 更新单个任务

`PUT /v1/task`

- 删除单个任务

`DEL /v1/task/:tid`

- 获取日志

`GET /v1/log`

```go
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
```
