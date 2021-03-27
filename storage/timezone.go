package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetAllSupportTimezone(ctx context.Context) ([]Timezone, error) {
	var err error

	timezones := make([]Timezone, 0)
	opts := options.Find()
	opts.SetSort(bson.D{{"label", DESC}})
	cursor, err := TimezoneCol.Find(ctx, bson.D{}, opts)
	if err != nil {
		return timezones, errors.Wrap(err, "get timezone err")
	}
	if err = cursor.All(ctx, &timezones); err != nil {
		return timezones, err
	}

	return timezones, nil
}

//timezoneIsValid 验证时区是否有效
func timezoneIsValid(tz string) error {
	ctx := context.Background()
	timezones, err := GetAllSupportTimezone(ctx)
	if err != nil {
		return err
	}
	for _, v := range timezones {
		if v.Value == tz {
			return nil
		}
	}
	return TimezoneNotFoundErr(tz)
}

//timezoneIsExists 数据库是否存在此时区
func timezoneIsExists(ctx context.Context, tz string) bool {
	timezones, err := GetAllSupportTimezone(ctx)
	if err != nil {
		return false
	}
	for _, v := range timezones {
		if v.Value == tz {
			return true
		}
	}
	return false
}

func CreateSupportTimezone(ctx context.Context, t *Timezone) error {
	t.CreateAt = time.Now().Unix()
	t.UpdateAt = t.CreateAt
	t.Id = primitive.NewObjectID()
	t.Tid = t.Id.Hex()
	log.Debugf("[model] tid is %v", t.Tid)
	if t.Value == "" || t.Label == "" {
		return errors.New("value/label 没有输入值")
	}
	// timezone 校验
	_, err := time.LoadLocation(t.Value)
	if err != nil { // unknown timezone
		return err
	}
	// 是否在现有的数据库中
	if timezoneIsExists(ctx, t.Value) {
		return TimezoneIsExistsErr
	}
	res, err := TimezoneCol.InsertOne(ctx, &t)
	if err != nil {
		return errors.Wrap(err, "[model] insert task err")
	}
	log.Debugf("[model] insert id is: %v", res.InsertedID)

	return nil
}

//GetSupportTimezone
func GetSupportTimezone(ctx context.Context, tid string) (Timezone, error) {
	var t Timezone

	oid, err := primitive.ObjectIDFromHex(tid)
	if err != nil {
		return t, errors.Wrap(err, fmt.Sprintf("[model] get timezone %s err", tid))
	}

	if err = TimezoneCol.
		FindOne(ctx, bson.D{{"_id", oid}}).
		Decode(&t); err != nil {
		log.Errorf("[model] decode task %s err:%v", tid, err)
		return t, errors.Wrap(err, fmt.Sprintf("[model] decode task %s err", tid))
	}
	return t, nil
}

//DeleteSupportTimezone
func DeleteSupportTimezone(ctx context.Context, tid string) error {
	timezone, err := GetSupportTimezone(ctx, tid)
	if err != nil {
		return err
	}

	res, err := TimezoneCol.DeleteOne(ctx, bson.D{{"_id", timezone.Id}})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("[model] delete task %s err", tid))
	}
	log.Debugf("[model] delete count: %d", res.DeletedCount)
	return nil
}
