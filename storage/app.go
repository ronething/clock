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

//GetAllApp 获取所有应用
func GetAllApp(ctx context.Context) ([]App, error) {
	var err error

	apps := make([]App, 0)
	opts := options.Find()
	opts.SetSort(bson.D{{"create_at", DESC}})
	// filter appKey and secretKey
	opts.SetProjection(bson.D{
		{"app_key", 0},
		{"secret_key", 0},
	})
	cursor, err := AppCol.Find(ctx, bson.D{}, opts)
	if err != nil {
		return apps, errors.Wrap(err, "get app err")
	}
	if err = cursor.All(ctx, &apps); err != nil {
		return apps, err
	}

	return apps, nil

}

//GetAppKeyAndSecretKey
func GetAppKeyAndSecretKey(ctx context.Context, aid string) (appKey, secretKey string, err error) {
	app, err := GetOneApp(ctx, aid)
	if err != nil {
		return "", "", err
	}
	return app.AppKey, app.SecretKey, nil
}

//CreateApp 创建一个应用
func CreateApp(ctx context.Context, a *App) error {
	a.CreateAt = time.Now().Unix()
	a.UpdateAt = a.CreateAt
	a.Id = primitive.NewObjectID()
	a.Aid = a.Id.Hex()
	// TODO: 生成 appKey, secretKey
	a.AppKey = ""
	a.SecretKey = ""
	res, err := AppCol.InsertOne(ctx, &a)
	if err != nil {
		return errors.Wrap(err, "新增 app 失败")
	}
	log.Debugf("[model] insert id is: %v", res.InsertedID)

	return nil
}

//GetOneApp 获取一个应用
func GetOneApp(ctx context.Context, aid string) (App, error) {
	var a App

	oid, err := primitive.ObjectIDFromHex(aid)
	if err != nil {
		return a, errors.Wrap(err, fmt.Sprintf("获取 app %s 失败", oid))
	}

	if err = AppCol.FindOne(ctx, bson.D{{"_id", oid}}).Decode(&a); err != nil {
		return a, errors.Wrap(err, fmt.Sprintf("decode app err: %v", err))
	}

	return a, nil

}

//ModifyAppName
func ModifyAppName(ctx context.Context, aid string, name string) error {
	app, err := GetOneApp(ctx, aid)
	if err != nil {
		return err
	}

	res, err := AppCol.UpdateOne(ctx, bson.D{{"name", app.AppName}}, bson.M{
		"$set": bson.M{
			"update_at": time.Now().Unix(),
			"name":      name,
		},
	})
	if err != nil {
		return errors.Wrap(err, "更新 AppName 失败")
	}

	log.Debugf("[model] match count: %v, modify count: %v", res.MatchedCount, res.ModifiedCount)

	return nil
}

//DeleteApp 删除一个应用
func DeleteApp(ctx context.Context, aid string) error {
	app, err := GetOneApp(ctx, aid)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	res, err := AppCol.UpdateOne(ctx, bson.D{{"_id", app.Id}}, bson.M{
		"$set": bson.M{
			"update_at":  now,
			"is_deleted": now,
		},
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("删除 app %s 失败", aid))
	}

	log.Debugf("[model] delete count: %d", res.ModifiedCount)

	return nil
}
