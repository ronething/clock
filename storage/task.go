package storage

import (
	"clock/v3/config"
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//GetTasks 取出所有任务 默认前 10
func GetTasks(query *TaskQuery) ([]Task, error) {
	var err error
	tasks := make([]Task, 0)

	if query.Count < 1 {
		query.Count = 10
	}

	if query.Index < 1 {
		query.Index = 1
	}

	queryDB := GetWhereDb(query, nil)
	count, err := TaskCol.CountDocuments(context.Background(), queryDB.Map())
	if err != nil {
		log.Errorf("[model] failed to get the page total of tasks : %v", err.Error())
		return tasks, err
	}
	query.Total = int(count)

	opts := options.Find()

	if query.Order != "" {
		opts = opts.SetSort(bson.D{{query.Order, DESC}}) // TODO: 默认 desc
	}
	opts = opts.SetSkip(int64((query.Index - 1) * query.Count)).SetLimit(int64(query.Count))

	cursor, err := TaskCol.Find(context.Background(), queryDB.Map(), opts)
	if err != nil {
		log.Errorf("[model] get tasks err: %v", err)
		return tasks, err
	}
	if err = cursor.All(context.Background(), &tasks); err != nil {
		log.Errorf("[model] decode all tasks err: %v", err)
		return tasks, err
	}

	return tasks, nil
}

//GetTask 根据 ObjectId 进行查询
func GetTask(tid string) (Task, error) {
	var t Task

	oid, err := primitive.ObjectIDFromHex(tid)
	if err != nil {
		log.Errorf("[model] get task %s err:%v", tid, err)
		return t, err
	}

	if err = TaskCol.
		FindOne(context.Background(), bson.D{{"_id", oid}}).
		Decode(&t); err != nil {
		log.Errorf("[model] decode task %s err:%v", tid, err)
		return t, err
	}
	return t, nil
}

//PostTask 新增任务
func PostTask(t *Task) error {
	if t.Type == "" {
		t.Type = "http"
	}
	if t.Payload == nil {
		msg := "[model] payload is nil"
		log.Errorf(msg)
		return errors.New(msg)
	}
	if t.Expression == "" {
		return errors.New("task expression is nil")
	}
	if t.Timezone == "" {
		t.Timezone = "Asia/Shanghai"
	}
	// 检验 timezone 是否是允许的 timezone 中
	err := timezoneIsValid(t.Timezone)
	if err != nil {
		return err
	}
	t.CreateAt = time.Now().Unix()
	t.UpdateAt = t.CreateAt
	t.Id = primitive.NewObjectID()
	t.Tid = t.Id.Hex()
	log.Debugf("[model] tid is %v", t.Tid)
	res, err := TaskCol.InsertOne(context.Background(), &t)
	if err != nil {
		log.Errorf("[model] insert task err: %v", err)
		return err
	}
	log.Debugf("[model] insert id is: %v", res.InsertedID)

	if err = PubRedis(t.Tid, CREATE); err != nil {
		log.Errorf("[post task] pub event to redis err: %v", err)
		return err
	}

	return nil
}

//DisableTask
func DisableTask(tid string, tMap map[string]interface{}) error {
	return PutTask(tid, tMap, DISABLE)
}

//ModifyTask
func ModifyTask(tid string, tMap map[string]interface{}) error {
	return PutTask(tid, tMap, MODIFY)
}

// 更新任务
func PutTask(tid string, tMap map[string]interface{}, event int) error {
	oldTask, err := GetTask(tid)
	if err != nil {
		log.Errorf("[model] 没有对应的 task %s", err.Error())
		return err
	}

	// 注意要先进行存储 task 的 tid 会被赋值，然后再带过去 redis
	res, err := TaskCol.UpdateOne(context.Background(),
		bson.D{{
			"_id",
			oldTask.Id,
		}},
		bson.D{{
			"$set",
			&tMap,
		}})
	if err != nil {
		log.Errorf("[model] update err: %v", err)
		return err
	}
	log.Debugf("[model] match count: %v, modify count: %v", res.MatchedCount, res.ModifiedCount)

	if err = PubRedis(tid, event); err != nil {
		log.Errorf("[put task] pub event to redis err: %v", err)
		return err
	}

	return nil
}

//PubRedis 发布事件到 redis
func PubRedis(tid string, event int) error {
	if !config.Config.GetBool("pubsub.open") {
		log.Debugf("[pub redis] 通道关闭")
		return nil
	}
	// DONE: pub to redis
	e := TaskEvent{
		Event: event,
		Tid:   tid,
	}
	msg, err := msgpack.Marshal(e)
	log.Debugf("[model] 发布消息到 redis %v", e)
	if err != nil {
		log.Errorf("[model] 序列化错误 %s", err.Error())
		return err
	}
	if err := RCache.Publish(msg); err != nil {
		log.Errorf("[model] 发布消息到 redis 失败 %s", err.Error())
		return err
	}
	return nil
}

//RunTask 执行任务
func RunTask(tid string) error {
	task, err := GetTask(tid)
	if err != nil {
		log.Errorf("error to find the task with: %v", err)
		return RunTaskNotFoundTaskErr
	}
	l, err := time.LoadLocation(task.Timezone)
	if err != nil {
		log.Errorf("[task] load time location err: %v", err)
		return err
	}
	now := time.Now().In(l).Format(time.RFC3339)

	// 这里要阻塞 不然调度器会以为任务已经完成，所以直接 stop
	switch task.Type {
	case BashTask:
		// TODO: shellcheck https://github.com/koalaman/shellcheck
		err = RunBashTask(task)
	case HTTPTask:
		err = RunHTTPTask(task, now)
	default:
		err = errors.New("[task] 暂时不支持除 http,bash 之外的作业类型")
	}
	return err
}

//DeleteTask 删除任务
func DeleteTask(tid string) error {
	// 1、查询出来
	task, err := GetTask(tid)
	if err != nil {
		log.Errorf("[model] get task: %v", err)
		return err
	}
	// DONE: 2、pub 到 redis
	e := TaskEvent{
		Event: DELETE,
		Tid:   tid,
	}
	msg, err := msgpack.Marshal(e)
	log.Debugf("[model] 发布消息到 redis %v", e)
	if err != nil {
		log.Errorf("[model] 序列化错误 %s", err.Error())
		return err
	}
	if err := RCache.Publish(msg); err != nil {
		log.Errorf("[model] 发布消息到 redis 失败 %s", err.Error())
		return err
	}
	// 即使 redis publish 有问题，数据库中的任务删除后，
	// 因为每次 worker RunTask 都会从数据库中取出数据，所以如果数据库的任务被删除了, worker 中也无法跑这个任务
	// 2、删除数据库中的 task
	res, err := TaskCol.DeleteOne(context.Background(), bson.D{{"_id", task.Id}})
	if err != nil {
		log.Errorf("[model] delete task %s err: %v", task.Tid, err)
		return err
	}
	log.Debugf("[model] delete count: %d", res.DeletedCount)
	return nil
}
