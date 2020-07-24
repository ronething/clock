package storage

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//GetLogs 获取任务执行日志 默认前 10
func GetLogs(query *LogQuery) ([]TaskLog, error) {
	logs := make([]TaskLog, 0)

	if query.Count < 1 {
		query.Count = 10
	}

	if query.Index < 1 {
		query.Index = 1
	}

	queryDB := GetWhereDb(query, []string{"lid"})
	if query.LeftTs > 0 {
		queryDB = append(queryDB, bson.E{Key: "create_at", Value: bson.D{{"$gt", query.LeftTs}}})
	}

	if query.RightTs > 0 {
		queryDB = append(queryDB, bson.E{Key: "create_at", Value: bson.D{{"$lt", query.RightTs}}})
	}

	count, err := TaskLogCol.CountDocuments(context.Background(), queryDB.Map())
	if err != nil {
		logrus.Errorf("[model] failed to get the page total of logs : %v", err.Error())
		return logs, err
	}
	query.Total = int(count)

	opts := options.Find()

	opts = opts.
		SetSort(bson.D{{"create_at", DESC}}).
		SetSkip(int64((query.Index - 1) * query.Count)).
		SetLimit(int64(query.Count))

	cursor, err := TaskLogCol.Find(context.Background(), queryDB.Map(), opts)
	if err != nil {
		logrus.Errorf("[model] get logs err: %v", err)
		return logs, err
	}
	if err = cursor.All(context.Background(), &logs); err != nil {
		logrus.Errorf("[model] decode all logs err: %v", err)
		return logs, err
	}

	return logs, nil
}

//DeleteLogs 根据 ts 删除多久以前的数据 NOTICE: 一般不会删除日志
func DeleteLogs(query *LogQuery) error {
	queryDB := GetWhereDb(query, []string{"lid"})

	if query.LeftTs > 0 {
		queryDB = append(queryDB, bson.E{Key: "create_at", Value: bson.D{{"$gt", query.LeftTs}}})
	}

	if query.RightTs > 0 {
		queryDB = append(queryDB, bson.E{Key: "create_at", Value: bson.D{{"$lt", query.RightTs}}})
	}

	res, err := TaskLogCol.DeleteMany(context.Background(), queryDB)
	if err != nil {
		logrus.Errorf("[model] delete logs err: %v", err)
		return err
	}
	logrus.Debugf("[model] delete logs count: %d", res.DeletedCount)
	return nil
}
