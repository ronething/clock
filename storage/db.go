package storage

import (
	"clock/v3/config"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Mdb    *mdb
	RCache Cache
)

// mongo 对应集合
var (
	TaskCol     *mongo.Collection // 任务集合
	TaskLogCol  *mongo.Collection // 任务日志集合
	TimezoneCol *mongo.Collection // 支持时区集合
	AppCol      *mongo.Collection // 应用集合
)

type mdb struct {
	Client *mongo.Client
	Cron   *mongo.Database
}

// initMongo
func initMongo() error {
	// 连接 mongo db
	opts := options.Client().ApplyURI(config.Config.GetString("database.mongo.conn"))
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		log.Errorf("[db] mongo connect err: %v", err)
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := client.Ping(ctx, nil); err != nil {
		log.Errorf("[db] mongo ping err: %v", err)
		return err
	}
	log.Debugf("[db] mongo ping success")
	// insert for ping
	cron := client.Database(config.Config.GetString("database.mongo.db"))

	Mdb = &mdb{
		Client: client,
		Cron:   cron,
	}

	// init col
	TaskCol = Mdb.Cron.Collection("task")
	TaskLogCol = Mdb.Cron.Collection("task_log")
	TimezoneCol = Mdb.Cron.Collection("timezone")
	AppCol = Mdb.Cron.Collection("app")

	return nil
}

// initRedis
func initRedis() error {
	// 连接 redis
	RCache = NewRedisCache(
		redis.NewClient(&redis.Options{
			Addr:     config.Config.GetString("cache.redis.addr"),
			Password: config.Config.GetString("cache.redis.auth"),
			DB:       config.Config.GetInt("cache.redis.db"),
		}))
	pong, err := RCache.Ping()
	if err != nil {
		log.Errorf("[db] redis ping err: %v", err)
		return err
	}
	log.Debugf("[db] rdb ping res is %s", pong)

	return nil
}

// SetDb 初始化
func SetDb() error {

	if err := initMongo(); err != nil {
		log.Errorf("[db] init mongo err: %v", err)
		return err
	}

	if err := initRedis(); err != nil {
		log.Errorf("[db] init redis err: %v", err)
		return err
	}

	return nil
}

//RevokeDb 释放连接
func RevokeDb() {
	var err error
	if err = Mdb.Client.Disconnect(context.Background()); err != nil {
		log.Errorf("[db] 释放连接 Mongo db 错误, %v", err)
	}
	if err = RCache.Close(); err != nil {
		log.Errorf("[db] 释放连接 Redis db 错误, %v", err)
	}

}
