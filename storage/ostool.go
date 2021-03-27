package storage

import (
	"bytes"
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	uuid "github.com/nu7hatch/gouuid"
	log "github.com/sirupsen/logrus"
)

func GenGuid(length int) (string, error) {
	u, e := uuid.NewV4()

	if e != nil {
		log.Error(e)
		return "", e
	}

	guid := u.String()
	guid = strings.Replace(guid, "-", "", -1)

	return guid[0:length], nil
}

func saveLog(t Task, stdOut, stdErr *bytes.Buffer, start, end int64) {

	// 回写日志状态
	if t.LogEnable {
		id := primitive.NewObjectID()
		l := TaskLog{
			Id:       id,
			Lid:      id.Hex(),
			Tid:      t.Tid,
			StdOut:   stdOut.String(),
			StdErr:   stdErr.String(),
			StartAt:  start,
			EndAt:    end,
			CreateAt: time.Now().Unix(),
		}
		// TODO: MDB Batch
		res, err := TaskLogCol.InsertOne(context.Background(), &l)
		if err != nil {
			log.Errorf("[ostool] insert log to db err: %v", err)
			return
		}
		log.Debugf("[ostool] insert id: %v", res.InsertedID)

	}

}
