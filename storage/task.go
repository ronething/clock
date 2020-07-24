package storage

import (
	"clock/config"
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
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
		logrus.Errorf("[model] failed to get the page total of tasks : %v", err.Error())
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
		logrus.Errorf("[model] get tasks err: %v", err)
		return tasks, err
	}
	if err = cursor.All(context.Background(), &tasks); err != nil {
		logrus.Errorf("[model] decode all tasks err: %v", err)
		return tasks, err
	}

	return tasks, nil
}

//GetTask 根据 ObjectId 进行查询
func GetTask(tid string) (Task, error) {
	var t Task

	oid, err := primitive.ObjectIDFromHex(tid)
	if err != nil {
		logrus.Errorf("[model] get task %s err:%v", tid, err)
	}

	if err := TaskCol.
		FindOne(context.Background(), bson.D{{"_id", oid}}).
		Decode(&t); err != nil {
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
		logrus.Errorf(msg)
		return errors.New(msg)
	}
	t.CreateAt = time.Now().Unix()
	t.UpdateAt = t.CreateAt
	t.Id = primitive.NewObjectID()
	t.Tid = t.Id.Hex()
	t.EntryId = -1
	logrus.Debugf("[model] tid is %v", t.Tid)
	res, err := TaskCol.InsertOne(context.Background(), &t)
	if err != nil {
		logrus.Errorf("[model] insert task err: %v", err)
		return err
	}
	logrus.Debugf("[model] insert id is: %v", res.InsertedID)
	return nil
}

// 更新任务或者新增任务
func PutTask(t *Task) error {
	if t.Tid == "" { // 新增
		if err := PostTask(t); err != nil {
			logrus.Errorf("[model] post task err: %v", err)
			return err
		}
	} else { // 存在,修改
		t.UpdateAt = time.Now().Unix()
		oldTask, err := GetTask(t.Tid)
		if err != nil {
			logrus.Errorf("[model] 没有对应的 task %s", err.Error())
			return err
		}

		t.Id = oldTask.Id // 赋上 ObjectID
		t.CreateAt = oldTask.CreateAt

		// 数据融合 TODO: 使用 map 获取 json 数据 if ok, v = map[key]
		if t.Payload == nil {
			t.Payload = oldTask.Payload
		}

		if t.Name == "" {
			t.Name = oldTask.Name
		}
		if t.Status == 0 {
			t.Status = oldTask.Status
		}

		if t.Expression == "" {
			t.Expression = oldTask.Expression
		}

		if t.EntryId == 0 {
			t.EntryId = oldTask.EntryId
		}

		// 注意要先进行存储 task 的 tid 会被赋值，然后再带过去 redis
		res, err := TaskCol.UpdateOne(context.Background(),
			bson.D{{
				"_id",
				oldTask.Id,
			}},
			bson.D{{
				"$set",
				t,
			}})
		if err != nil {
			logrus.Errorf("[model] update err: %v", err)
			return err
		}
		logrus.Debugf("[model] match count: %v, modifty count: %v", res.MatchedCount, res.ModifiedCount)
	}

	// DONE: pub to redis
	channelName := config.Config.GetString("pubsub.channel")
	e := TaskEvent{
		Event: PUT,
		Task:  *t,
	}
	m, err := msgpack.Marshal(e)
	logrus.Debugf("[model] 发布消息到 redis %v", e)
	if err != nil {
		logrus.Errorf("[model] 序列化错误 %s", err.Error())
		return err
	}
	if err := Rdb.Publish(context.Background(), channelName, m).Err(); err != nil {
		logrus.Errorf("[model] 发布消息到 redis 失败 %s", err.Error())
		return err
	}
	return nil
}

//RunTask 执行任务
func RunTask(tid string) error {
	task, err := GetTask(tid)
	if err != nil {
		logrus.Errorf("error to find the task with: %v", err)
		return err
	}

	// 这里要阻塞 不然调度器会以为任务已经完成，所以直接 stop
	if task.Type == BashTask {
		err = RunBashTask(task)
	} else if task.Type == HTTPTask {
		err = RunHTTPTask(task)
	} else { // 暂时不支持其他类型作业
		err = errors.New("[task] 暂时不支持除 http,bash 之外的作业类型")
	}
	return err
}

//DeleteTask 删除任务
func DeleteTask(tid string) error {
	// 1、查询出来
	task, err := GetTask(tid)
	if err != nil {
		logrus.Errorf("[model] get task: %v", err)
		return err
	}
	// DONE: 2、pub 到 redis
	channelName := config.Config.GetString("pubsub.channel")
	e := TaskEvent{
		Event: DEL,
		Task:  task,
	}
	m, err := msgpack.Marshal(e)
	logrus.Debugf("[model] 发布消息到 redis %v", e)
	if err != nil {
		logrus.Errorf("[model] 序列化错误 %s", err.Error())
		return err
	}
	if err := Rdb.Publish(context.Background(), channelName, m).Err(); err != nil {
		logrus.Errorf("[model] 发布消息到 redis 失败 %s", err.Error())
		return err
	}
	// 即使 redis publish 有问题，数据库中的任务删除后，
	// 因为每次 worker RunTask 都会从数据库中取出数据，所以如果数据库的任务被删除了, worker 中也无法跑这个任务
	// 2、删除数据库中的 task
	res, err := TaskCol.DeleteOne(context.Background(), bson.D{{"_id", task.Id}})
	if err != nil {
		logrus.Errorf("[model] delete task %s err: %v", task.Tid, err)
		return err
	}
	logrus.Debugf("[model] delete count: %d", res.DeletedCount)
	return nil
}
