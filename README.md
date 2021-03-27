<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Clock](#clock)
  - [env](#env)
  - [usage](#usage)
    - [direct](#direct)
    - [build binary](#build-binary)
    - [build docker image](#build-docker-image)
  - [api](#api)
    - [任务相关](#%E4%BB%BB%E5%8A%A1%E7%9B%B8%E5%85%B3)
    - [日志相关](#%E6%97%A5%E5%BF%97%E7%9B%B8%E5%85%B3)
    - [监控相关](#%E7%9B%91%E6%8E%A7%E7%9B%B8%E5%85%B3)
  - [curl sample](#curl-sample)
  - [acknowledgement](#acknowledgement)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Clock

基于 go cron 和 redis 实现分布式任务调度(支持多 worker，多 master)

### env

base on go1.13+

### usage

#### direct

下载 release 中的文件, 需要启动一个 redis 服务器,数据库使用 mongo

```
# ronething @ ashings-macbook-pro in /tmp/clock [18:17:21] C:130
$ ll
total 99264
-rw-r--r--  1 ronething  wheel    32K  7 19 18:17 clock.db
-rw-r--r--  1 ronething  wheel   521B  7 19 18:05 dev.yaml
-rwxr-xr-x  1 ronething  wheel    26M  7 19 18:12 master
-rwxr-xr-x  1 ronething  wheel    23M  7 19 18:05 worker

# ronething @ ashings-macbook-pro in /tmp/clock [18:17:21] C:130
$ ./master -c ./dev.yaml

# ronething @ ashings-macbook-pro in /tmp/clock [18:17:21] C:130
$ ./worker -c ./dev.yaml

```

#### build binary

- for dev

```sh
# 进入项目目录
# 分别在 master 和 worker 目录下进行构建
cd master && go build
./master -c ../config/dev.yaml

cd worker && go build
./worker -c ../config/dev.yaml
```

- for prod

```sh
cd clock
make && cd bin/
# 可以看到 master 和 worker 二进制文件
# 注：会调用 upx 进行二进制文件压缩
```
 

#### build docker image 

注： `go mod vendor` 很重要，如果更新了 go.mod 请执行此命令对 vendor 文件夹进行更新

```sh
make docker-build-master version=0.0.1
make docker-build-worker version=0.0.1
```

### api

#### 任务相关

- 获取所有任务

`GET /v1/task`

- 获取单个任务

`GET /v1/task/:tid`

- 启停任务

`PUT /v1/task/disable`

- 变更任务定时表达式和时区

`PUT /v1/task/newspec`

- 删除单个任务

`DEL /v1/task/:tid`

#### 日志相关

`GET /v1/log`

#### 监控相关

- prometheus exporter

`GET /metrics`

- 健康检查

`GET /ping`

### curl sample

- 添加时区

```sh
# ronething @ ashings-macbook-pro in ~/Documents/clock/bin on git:v3 x [19:30:29] 
$ curl --location --request POST 'http://127.0.0.1:9528/v1/timezone' \
--header 'Content-Type: application/json' \
--data-raw '{
    "label": "上海",
    "value": "Asia/Shanghai"
}'
{"code":0,"msg":"success","data":"605f18d2e0f10eacb6139a9f"}
```

- 添加任务

```sh
# ronething @ ashings-macbook-pro in ~/Documents/clock/bin on git:v3 x [19:36:51] 
$ curl --location --request POST 'http://127.0.0.1:9528/v1/task' \    
--header 'Content-Type: application/json' \
--data-raw '{
    "name":"任务1",
    "timeout": 5,
    "log_enable": true,
    "expression": "* * * * *",
    "timezone": "Asia/Shanghai",
    "payload": {
        "command": "date"
    },
    "type": "bash"
}'
{"code":0,"msg":"success","data":"605f18e6e0f10eacb6139aa0"}
```

添加任务之后 master 和 worker 的变化

```sh
# master 发布消息到 worker
DEBU[1898] [model] tid is 605f18e6e0f10eacb6139aa0      
DEBU[1898] [model] insert id is: ObjectID("605f18e6e0f10eacb6139aa0") 
DEBU[1898] [model] 发布消息到 redis {1 605f18e6e0f10eacb6139aa0} 
INFO[1898] 127.0.0.1 POST /v1/task 200 61 136.26896ms  curl/7.54.0  prefix=echo

# worker 接收消息并在对应的时间点执行任务
DEBU[1120] [scheduler] taskEvent is {1 605f18e6e0f10eacb6139aa0} 
INFO[1120] [scheduler] 添加定时任务 605f18e6e0f10eacb6139aa0-任务1, 表达式: CRON_TZ=Asia/Shanghai * * * * *, with entryID: 1 
INFO[1120] [scheduler] CREATE 1 事件处理成功                  
[Cron]: 2021/03/27 19:37:10 added, now=2021-03-27T19:37:10+08:00, entry=1, next=2021-03-27T19:38:00+08:00
[Cron]: 2021/03/27 19:38:00 wake, now=2021-03-27T19:38:00+08:00
[Cron]: 2021/03/27 19:38:00 run, now=2021-03-27T19:38:00+08:00, entry=1, next=2021-03-27T19:39:00+08:00
DEBU[1169] [ostool] execute job name: 任务1, tid: 605f18e6e0f10eacb6139aa0 
DEBU[1169] [605f18e6e0f10eacb6139aa0] - now will run the task [任务1] 
DEBU[1169] [605f18e6e0f10eacb6139aa0] - now finish task [任务1] 
DEBU[1169] [ostool] update task 605f18e6e0f10eacb6139aa0, match: 1, modify: 1 
DEBU[1169] [ostool] insert id: ObjectID("605f1918357868e6166358fd") 
DEBU[1170] [ostool] task cron:job:605f18e6e0f10eacb6139aa0 任务完成 
DEBU[1170] [ostool] 删除 key cron:job:605f18e6e0f10eacb6139aa0 成功 
```

- 禁用任务

```sh
# ronething @ ashings-macbook-pro in ~/Documents/clock/bin on git:v3 x [19:37:11] 
$ curl --location --request PUT 'http://127.0.0.1:9528/v1/task/605f18e6e0f10eacb6139aa0/disable' \
--header 'Content-Type: application/json' \
--data-raw '{
    "disable": true
}'
{"code":0,"msg":"success","data":null}
```

```sh
# master 发布消息到 worker
DEBU[2264] [model] 发布消息到 redis {3 605f18e6e0f10eacb6139aa0} 
INFO[2264] 127.0.0.1 PUT /v1/task/605f18e6e0f10eacb6139aa0/disable 200 39 4.94317ms  curl/7.54.0  prefix=echo

# worker 接收消息并从调度器中移除任务
DEBU[1403] [scheduler] taskEvent is {3 605f18e6e0f10eacb6139aa0} 
INFO[1403] job 605f18e6e0f10eacb6139aa0-任务1, entryId is 1 
INFO[1403] [scheduler] 成功禁用任务 605f18e6e0f10eacb6139aa0-任务1 
INFO[1403] [scheduler] DISABLE 3 事件处理成功                 
[Cron]: 2021/03/27 19:41:54 removed, entry=1
DEBU[1429] [scheduler] taskEvent is {3 605f18e6e0f10eacb6139aa0} 
INFO[1429] job 605f18e6e0f10eacb6139aa0-任务1, entryId is 1 
[Cron]: 2021/03/27 19:42:20 removed, entry=1
```

- 更多 api 请查看 [server.go](./master/server/server.go)

### acknowledgement

- [cron](https://github.com/robfig/cron) - cron scheduler
- [echo](https://github.com/labstack/echo) - web framework
- [mongo-go-driver](https://github.com/mongodb/mongo-go-driver) - mongo driver
- [go-redis](https://github.com/go-redis/redis)  - redis client
- [msgpack](https://github.com/vmihailenco/msgpack) - binary serialization
