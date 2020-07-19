package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-redis/redis/v8"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"

	"clock/config"
)

var (
	Db          *gorm.DB
	Rdb         *redis.Client
	Messenger   Message
	MessageSize = 1000
)

const (
	PENDING = iota + 1
	START
	SUCCESS
	FAILURE
)

type (

	// 当前任务
	Task struct {
		Tid        int    `json:"tid" gorm:"PRIMARY_KEY"`            // task id
		Command    string `json:"command"`                           // 当前只支持bash command
		Name       string `json:"name" gorm:"unique_index;not null"` // task 名字
		Disable    bool   `json:"disable"`                           // 是否禁用当前任务
		Status     int    `json:"status" gorm:"default:1"`           // 当前状态
		TimeOut    int    `json:"timeout"`                           // 超时时间
		CreateAt   int64  `json:"create_at"`                         // 创建时间
		UpdateAt   int64  `json:"update_at"`                         // 修改时间
		LogEnable  bool   `json:"log_enable"`                        // 是否启用日志
		Expression string `json:"expression"`                        // 表达式 支持@every [1s | 1m | 1h ] 参考 cron
		EntryId    int    `json:"entry_id"`                          // 调度器生成的 id
	}

	// 任务日志
	TaskLog struct {
		Lid      string `json:"lid"  gorm:"PRIMARY_KEY"`  // 主键Key
		Tid      int    `json:"tid" gorm:"index:idx_tid"` // task id
		StdOut   string `json:"std_out"`                  // 正常输出流
		StdErr   string `json:"std_err"`                  // 异常输出流
		StartAt  int64  `json:"start_at"`                 // 任务开始时间
		EndAt    int64  `json:"end_at"`                   // 任务结束时间
		CreateAt int64  `json:"create_at" gorm:"index"`   // 创建时间
	}
)

type (
	Page struct {
		Count   int    `json:"count"`
		Index   int    `json:"index"`
		Total   int    `json:"total"`
		Order   string `json:"order"`
		LeftTs  int64  `json:"left_ts" query:"left_ts"`
		RightTs int64  `json:"right_ts" query:"right_ts"`
	}

	// task 查询参数
	TaskQuery struct {
		Page
		Task
	}

	LogQuery struct {
		Page
		TaskLog
	}
)

// 应用所需实体
type (

	// websocket 消息
	Message struct {
		Size    int         //容量
		Channel chan string //信息通道
	}

	// 统计数据
	TaskCounter struct {
		Title string `json:"title"`
		Icon  string `json:"icon"`
		Count int    `json:"count"`
		Color string `json:"color"`
	}

	// redis 传输对象
	TaskEvent struct {
		Event string `json:"event"` // PUT/DEL
		Task  *Task  `json:"task"`
	}
)

// 初使化
func SetDb() {
	conn := config.Config.GetString("storage.conn")
	if conn == "" {
		logrus.Fatal("empty conn string")
	}

	backend := config.Config.GetString("storage.backend")
	if backend == "" {
		logrus.Fatal("not find the backend type")
	}

	var err error
	Db, err = gorm.Open(backend, conn)

	if err != nil {
		logrus.Fatal(err)
	}

	Db.AutoMigrate(&Task{}, &TaskLog{})

	Rdb = redis.NewClient(&redis.Options{
		Addr:     config.Config.GetString("cache.redis.addr"),
		Password: config.Config.GetString("cache.redis.auth"),
		DB:       config.Config.GetInt("cache.redis.db"),
	})
	pong, err := Rdb.Ping(context.Background()).Result()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Debugf("[storage] rdb ping res is %s", pong)

	tmp := config.Config.GetInt("message.size")
	if tmp > 0 {
		MessageSize = tmp
	}
	Messenger = NewMessenger(MessageSize)

}

// 初使化信息通道
func NewMessenger(size int) Message {
	return Message{
		Size:    size,
		Channel: make(chan string, size),
	}
}

func GetWhereDb(object interface{}, filter []string) *gorm.DB {
	db := Db
	// 过滤bool 和子类型为struct内容
	filterKind := []string{"bool", "struct"}
	// 过滤Page参数体
	filterStruct := []string{"Page"}

	s := structs.New(object)
	for _, key := range s.Names() {
		tmp := s.Field(key)

		if inCondition(filterStruct, key) {
			continue
		}

		fields := tmp.Fields()
		for _, f := range fields {
			field := f.Tag("json")

			// 过滤的字段
			if inCondition(filter, field) {
				continue
			}

			kind := fmt.Sprintf("%v", f.Kind())
			// 过滤bool类型和struct类型
			if inCondition(filterKind, kind) {
				continue
			}

			value := fmt.Sprintf("%v", f.Value())
			if kind == "string" && value != "" {
				db = db.Where(fmt.Sprintf("%v like ?", field), "%"+value+"%")
			}

			if kind == "int" && value != "0" {
				db = db.Where(fmt.Sprintf("%v = ?", field), value)
			}

		}
	}

	return db
}

// 默认取出根任务
func GetTasks(query *TaskQuery) ([]Task, error) {
	var tasks []Task

	if query.Count < 1 {
		query.Count = 10
	}

	if query.Index < 1 {
		query.Index = 1
	}

	queryDB := GetWhereDb(query, nil)
	if e := queryDB.Model(tasks).Count(&query.Total).Error; e != nil {
		logrus.Error("failed to get the page total of tasks :" + e.Error())
		return nil, e
	}

	queryDB = queryDB.Offset((query.Index - 1) * query.Count).Limit(query.Count)

	if query.Order != "" {
		queryDB = queryDB.Order(query.Order)
	}

	if err := queryDB.Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

// 更新query 多页的情况
func GetLogs(query *LogQuery) ([]TaskLog, error) {
	var logs []TaskLog

	if query.Count < 1 {
		query.Count = 10
	}

	if query.Index < 1 {
		query.Index = 1
	}

	queryDB := GetWhereDb(query, []string{"lid"})
	if query.LeftTs > 0 {
		queryDB = queryDB.Where("update_at > ?", query.LeftTs)
	}

	if query.RightTs > 0 {
		queryDB = queryDB.Where("update_at < ?", query.RightTs)
	}

	if e := queryDB.Model(logs).Count(&query.Total).Error; e != nil {
		logrus.Error("failed to get the page total of logs :" + e.Error())
		return nil, e
	}

	queryDB = queryDB.Offset((query.Index - 1) * query.Count).Limit(query.Count).Order("update_at desc")
	if err := queryDB.Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// 根据ts 删除多久以前的数据
func DeleteLogs(query *LogQuery) error {
	queryDB := GetWhereDb(query, []string{"lid"})

	if query.LeftTs > 0 {
		queryDB = queryDB.Where("update_at > ?", query.LeftTs)
	}

	if query.RightTs > 0 {
		queryDB = queryDB.Where("update_at < ?", query.RightTs)
	}

	queryDB.LogMode(true)
	return queryDB.Delete(TaskLog{}).Error
}

func inCondition(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetTask(tid int) (Task, error) {
	var t Task

	if err := Db.Where("tid = ?", tid).First(&t).Error; err != nil {
		return t, err
	}
	return t, nil
}

// 直接执行任务
func RunTask(tid int) error {
	task, err := GetTask(tid)
	if err != nil {
		logrus.Errorf("error to find the task with: %v", err)
		return err
	}

	go RunSingleTask(task) // 改为协程中运行
	return nil
}

// 删除任务
func DeleteTask(tid int) error {
	// 1、查询出来
	var task Task
	if err := Db.Where("tid = ?", tid).Find(&task).Error; err != nil {
		return err
	}
	// TODO: 2、pub 到 redis

	// 2、删除数据库中的 task
	// 使用事物进行原子操作
	if err := Db.Delete(&task).Error; err != nil {
		return err
	}
	return nil
}

// 更新任务或者新增任务
func PutTask(t *Task) error {
	if t.Tid == 0 { // 新增
		t.CreateAt = time.Now().Unix()
	}
	t.UpdateAt = time.Now().Unix()
	// TODO: pub to redis

	if err := Db.Save(&t).Error; err != nil {
		return err
	}
	return nil
}
