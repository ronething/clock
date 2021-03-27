package storage

import (
	"clock/v3/config"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Mdb         *mdb
	Rdb         *redis.Client
	Messenger   Message
	MessageSize = 1000
)

// mongo 对应集合
var (
	TaskCol    *mongo.Collection // 任务集合
	TaskLogCol *mongo.Collection // 任务日志集合
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
		logrus.Errorf("[db] mongo connect err: %v", err)
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := client.Ping(ctx, nil); err != nil {
		logrus.Errorf("[db] mongo ping err: %v", err)
		return err
	}
	// insert for ping
	cron := client.Database(config.Config.GetString("database.mongo.db"))
	//res, err := cron.Collection("health").InsertOne(
	//	context.Background(),
	//	bson.D{{"ping", time.Now().Unix()}},
	//)
	//if err != nil {
	//	logrus.Errorf("[db] mongo ping err: %v", err)
	//}
	//logrus.Debugf("[db] mongo ping insert id: %v", res.InsertedID)

	Mdb = &mdb{
		Client: client,
		Cron:   cron,
	}

	// init col
	TaskCol = Mdb.Cron.Collection("task")
	TaskLogCol = Mdb.Cron.Collection("task_log")

	return nil
}

// initRedis
func initRedis() error {
	// 连接 redis
	Rdb = redis.NewClient(&redis.Options{
		Addr:     config.Config.GetString("cache.redis.addr"),
		Password: config.Config.GetString("cache.redis.auth"),
		DB:       config.Config.GetInt("cache.redis.db"),
	})
	pong, err := Rdb.Ping(context.Background()).Result()
	if err != nil {
		logrus.Errorf("[db] redis ping err: %v", err)
		return err
	}
	logrus.Debugf("[db] rdb ping res is %s", pong)

	return nil
}

// SetDb 初始化
func SetDb() error {

	if err := initMongo(); err != nil {
		logrus.Errorf("[db] init mongo err: %v", err)
		return err
	}

	if err := initRedis(); err != nil {
		logrus.Errorf("[db] init redis err: %v", err)
		return err
	}

	tmp := config.Config.GetInt("message.size")
	if tmp > 0 {
		MessageSize = tmp
	}
	Messenger = NewMessenger(MessageSize)

	return nil
}

// 初使化信息通道
func NewMessenger(size int) Message {
	return Message{
		Size:    size,
		Channel: make(chan string, size),
	}
}

//RevokeDb 释放连接
func RevokeDb() {
	var err error
	if err = Mdb.Client.Disconnect(context.Background()); err != nil {
		logrus.Errorf("[db] 释放连接 Mongo db 错误, %v", err)
	}
	if err = Rdb.Close(); err != nil {
		logrus.Errorf("[db] 释放连接 Redis db 错误, %v", err)
	}

}
