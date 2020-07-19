<div align=center>
<img src="https://user-images.githubusercontent.com/12979090/86565300-297abd80-bf9a-11ea-916f-b547f5023ee8.png" /> 
</div>

## Clock
基于go cron的可视化调度轻量级调度框架，支持DAG任务依赖，支持bash命令

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
./clock -c ./config/dev.yaml
```

使用命令`./clock -c config/dev.yaml` 载入你的配置文件

## 特性与功能
* 支持多种数据库: sqlite , mysql ,postgresql
* 跨平台
