package storage

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/fatih/structs"
	"go.mongodb.org/mongo-driver/bson"
)

// 任务增删事件
const (
	CREATE  = iota + 1
	MODIFY  // timezone or expression
	DISABLE // on or off
	DELETE
)

// mongo 排序
const (
	DESC = -1
	ASC  = 1
)

// 作业类型
const (
	BashTask = "bash"
	HTTPTask = "http"
)

var (
	WaitForNextScheduleErr = errors.New("key 存在或者 value 设置不成功，等待下次调度")
	RunTaskNotFoundTaskErr = errors.New("没有在数据库中找到对应 task")
	TimezoneNotFoundErr    = func(name string) error {
		return errors.New(fmt.Sprintf(" %s 此时区不在支持的时区范围内", name))
	}
	TimezoneIsExistsErr = errors.New("此时区已存在数据库中")
)

type (

	// 当前任务
	// payload 请求体
	// 如果 type 是 bash，则 payload 需要有 command  TODO: 需要进行高危命令检测
	// 如果是 http，则需要有 url，method，以及 data, appKey, secretKey
	Task struct {
		Id         primitive.ObjectID     `json:"-" bson:"_id,omitempty"`       // mongo object id  omitempty ,之后不能有空格
		Tid        string                 `json:"tid" bson:"tid"`               // task id -> Id.Hex()
		Name       string                 `json:"name" bson:"name"`             // task 名字 TODO: 唯一索引
		Disable    bool                   `json:"disable" bson:"disable"`       // 是否禁用当前任务
		TimeOut    int                    `json:"timeout" bson:"timeout"`       // 超时时间
		CreateAt   int64                  `json:"create_at" bson:"create_at"`   // 创建时间
		UpdateAt   int64                  `json:"update_at" bson:"update_at"`   // 修改时间
		LogEnable  bool                   `json:"log_enable" bson:"log_enable"` // 是否启用日志
		Expression string                 `json:"expression" bson:"expression"` // 表达式 支持@every [1s | 1m | 1h ] 参考 cron
		Delay      bool                   `json:"delay" bson:"delay"`           // 是否是延迟作业
		Timezone   string                 `json:"timezone" bson:"timezone"`     // 新增时区配置
		Payload    map[string]interface{} `json:"payload" bson:"payload"`
		Type       string                 `json:"type" bson:"type"` // 目前支持两种类型 bash 和 http
	}

	// 任务日志
	TaskLog struct {
		Id       primitive.ObjectID `json:"-" bson:"_id,omitempty"`     // mongo object id
		Lid      string             `json:"lid"  bson:"lid"`            // 主键Key
		Tid      string             `json:"tid" bson:"tid"`             // task id TODO: 索引 加速查询
		StdOut   string             `json:"std_out" bson:"std_out"`     // 正常输出流
		StdErr   string             `json:"std_err" bson:"std_err"`     // 异常输出流
		StartAt  int64              `json:"start_at" bson:"start_at"`   // 任务开始时间
		EndAt    int64              `json:"end_at" bson:"end_at"`       // 任务结束时间
		CreateAt int64              `json:"create_at" bson:"create_at"` // 创建时间
	}

	// 支持的时区列表选项
	Timezone struct {
		Id       primitive.ObjectID `json:"-" bson:"_id,omitempty"`     // mongo object id
		Tid      string             `json:"tid" bson:"tid"`             // timezone id
		Value    string             `json:"value" bson:"value"`         // 对应 select 选择框的 value
		Label    string             `json:"label" bson:"label"`         // 对应 select 选择框的 label
		CreateAt int64              `json:"create_at" bson:"create_at"` // 创建时间
		UpdateAt int64              `json:"update_at" bson:"update_at"`
	}

	App struct {
		Id          primitive.ObjectID `json:"-" bson:"_id,omitempty"`           // mongo object id
		Aid         string             `json:"aid" bson:"aid"`                   // app id
		IsDeleted   int64              `json:"is_deleted" bson:"is_deleted"`     // 如果删除，则打上时间戳， 否则为 0
		ValidPeriod int64              `json:"valid_period" bson:"valid_period"` // 有效时长 TODO: 暂不启用
		SecretKey   string             `json:"secret_key" bson:"secret_key"`
		AppKey      string             `json:"app_key" bson:"app_key"`
		AppName     string             `json:"app_name" bson:"app_name"`   // unique
		LastUsed    int64              `json:"last_used" bson:"last_used"` // 最近一次使用时间 TODO: echo middleware
		Counter     int64              `json:"counter" bson:"counter"`     // 调用的次数
		CreateAt    int64              `json:"create_at" bson:"create_at"`
		UpdateAt    int64              `json:"update_at" bson:"update_at"`
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
	// 原先是一整个 Task 后续应该是考虑减少传输数据大小以及
	// task 不好构建(例如修改一个 task，需要传给 redis 的则是更新后的 task, 需要在 master 进行重新查找)
	// 不过这样删除 task 的时候，worker 可能会找不到数据，因为在 master 中这条数据已经被操作删除了 所以需要进行一下处理
	TaskEvent struct {
		Event int    `json:"event"`
		Tid   string `json:"tid"`
	}

	// BashTask
	BashTaskPayload struct {
		Command string `json:"command"`
	}

	// HTTPTask
	HTTPTaskPayload struct {
		EndPoint string      `json:"endpoint"`
		Prefix   string      `json:"prefix"`
		Method   string      `json:"method"`
		Data     interface{} `json:"data"`
	}
)

// inCondition 判断 s 中的某个元素是否和 e 相同
func inCondition(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

//GetWhereDb 进行条件预筛选
func GetWhereDb(object interface{}, filter []string) bson.D {
	var doc bson.D
	// 过滤 bool 和子类型为 struct 内容
	filterKind := []string{"bool", "struct"}
	// 过滤 Page 参数体
	filterStruct := []string{"Page"}

	s := structs.New(object)
	for _, key := range s.Names() {
		tmp := s.Field(key)

		if inCondition(filterStruct, key) {
			continue
		}

		fields := tmp.Fields()
		for _, f := range fields {
			field := f.Tag("bson")
			field = strings.Split(field, ",")[0]

			// 过滤的字段
			if inCondition(filter, field) {
				continue
			}

			kind := fmt.Sprintf("%v", f.Kind())
			// 过滤 bool 类型和 struct 类型
			if inCondition(filterKind, kind) {
				continue
			}

			value := fmt.Sprintf("%v", f.Value())
			if kind == "string" && value != "" {
				// TODO: like
				doc = append(doc, bson.E{Key: field, Value: value})
			}

			if kind == "int" && value != "0" {
				doc = append(doc, bson.E{Key: field, Value: value})
			}

		}
	}

	return doc
}

//Struct2bsonD 结构体转为 bson
func Struct2bsonD(i interface{}, tagName string) bson.D {
	doc := bson.D{}
	t := reflect.TypeOf(i)
	v := reflect.ValueOf(i)

	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get(tagName) // json/bson/custom
		doc = append(doc, bson.E{
			Key:   strings.Split(tag, ",")[0], // 取出第一个 _id,omitempty
			Value: v.Field(i).Interface(),
		})
	}
	log.Debugf("[model] bson is %v", doc)

	return doc
}
