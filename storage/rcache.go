package storage

import (
	"clock/v3/config"
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	Client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{Client: client}
}

//TryLock redis 分布式锁
func (r *RedisCache) TryLock(t Task) (chan int, error) {
	// 随机睡眠(0-1) 考虑到各个 worker 之间的时钟有相差 us，所以通过牺牲最多 1s 进行公平抢锁
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	ctx := context.Background()
	rid, _ := GenGuid(8)         // 生成 uuid 和 task name 组合作为 key 的 value
	jobDone := make(chan int, 1) // 任务完成的时候 done <- 1
	key := fmt.Sprintf("%s:%s", config.Config.GetString("lease.prefix"), t.Tid)
	res, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		log.Errorf("[ostool] redis exists key err: %v", err)
		return nil, err
	}
	if res == 1 { // key 存在 等待下次调度
		log.Debugf("[ostool] redis key %s 存在，等待下次调度", key)
		return nil, WaitForNextScheduleErr
	} else {
		val := fmt.Sprintf("%s:%s:%s", t.Tid, t.Name, rid)
		res, err := r.Client.SetNX(ctx, key, val, config.Config.GetDuration("lease.expire")*time.Second).Result()
		if err != nil {
			log.Errorf("[ostool] redis setnx key err: %v", err)
			return nil, err
		}
		if res == true { // 设置成功
			// DONE: 续租
			go r.LeaseJob(jobDone, key, val)
		} else {
			return nil, WaitForNextScheduleErr // 等待下次调度
		}
	}
	return jobDone, nil
}

// leaseJob 续租 redis job key
func (r *RedisCache) LeaseJob(done chan int, key string, val string) {
	leaseTime := config.Config.GetDuration("lease.per") * time.Second
	timer := time.NewTimer(leaseTime) // 默认 20 秒续租一次
	for {
		select {
		case <-done:
			time.Sleep(time.Second) // 延迟一秒完成任务 防止其他 worker 在极限时间内抢到锁再执行一次任务
			log.Debugf("[ostool] task %s 任务完成", key)
			// DONE: 这里如果发生阻塞，锁过期，然后其他 worker 获取到锁，则会有问题，执行到下面会把 key 给删除了
			script := redis.NewScript(`
				if redis.call("get",KEYS[1]) == ARGV[1] then
					return redis.call("del",KEYS[1])
				else
					return 0
				end
			`)
			_, err := script.Run(context.Background(), r.Client, []string{key}, val).Result()
			if err != nil {
				log.Errorf("[ostool] remove key %s err: %v", key, err)
			}
			log.Debugf("[ostool] 删除 key %s 成功", key)
			goto End
		case <-timer.C:
			log.Debugf("[ostool] 续租 %s", key)
			r.Client.Expire(context.Background(), key, config.Config.GetDuration("lease.expire")*time.Second)
			timer.Reset(leaseTime)
		}
	}
End:
	timer.Stop() // 暂停计时器
}

func (r *RedisCache) Publish(event []byte) error {
	ctx := context.Background()
	channelName := config.Config.GetString("pubsub.channel")
	return r.Client.Publish(ctx, channelName, event).Err()
}

func (r *RedisCache) Subscribe(c chan os.Signal) {
	ctx := context.Background()
	channelName := config.Config.GetString("pubsub.channel")
	pubsub := r.Client.Subscribe(ctx, channelName)

	defer pubsub.Close()

	for {
		// ReceiveTimeout is a low level API. Use ReceiveMessage instead.
		msgi, err := pubsub.Receive(ctx)
		if err != nil {
			goto End
		}
		switch msg := msgi.(type) {
		case *redis.Subscription:
			log.Infof("[scheduler] 成功订阅 %s", channelName)
		case *redis.Message:
			t := TaskEvent{}
			if err := msgpack.Unmarshal([]byte(msg.Payload), &t); err != nil {
				log.Errorf("[scheduler] msg 解析发生错误")
			}
			log.Debugf("[scheduler] taskEvent is %v", t)
			task, err := GetTask(t.Tid)
			if err != nil {
				log.Errorf("[scheduler] task %s not found", t.Tid)
				if t.Event == DELETE { // 删除事件特殊处理 数据可能已经被 master 删除了
					if err := NewSchedulerDeleteTask(t.Tid); err != nil {
						log.Errorf("[scheduler] DELETE 事件失败 %s", err.Error())
					}
					log.Infof("[scheduler] DELETE %d 事件处理成功", DELETE)
				}
				continue // 肯定是要 continue 的 后面的不用走
			}
			switch t.Event {
			case CREATE:
				if err := NewInitPutTask(task); err != nil {
					log.Errorf("[scheduler] CREATE 事件失败 %s", err.Error())
				}
				log.Infof("[scheduler] CREATE %d 事件处理成功", CREATE)
			case MODIFY:
				if err := NewSchedulerModifyTask(task); err != nil {
					log.Errorf("[scheduler] MODIFY 事件失败 %s", err.Error())
				}
				log.Infof("[scheduler] MODIFY %d 事件处理成功", MODIFY)
			case DISABLE:
				if err := NewSchedulerDisableTask(task); err != nil {
					log.Errorf("[scheduler] DISABLE 事件失败 %s", err.Error())
				}
				log.Infof("[scheduler] DISABLE %d 事件处理成功", DISABLE)
			case DELETE:
				if err := NewSchedulerDeleteTask(task); err != nil {
					log.Errorf("[scheduler] DELETE 事件失败 %s", err.Error())
				}
				log.Infof("[scheduler] DELETE %d 事件处理成功", DELETE)
			default:
				log.Errorf("[scheduler] 没有对应的类型")
			}
		default:
			log.Warnf("[scheduler] 默认是什么鬼 %v", msg)
			goto End
		}
	}
End:
	c <- os.Interrupt //手动发送信号给 c
	log.Errorf("[scheduler] 消息订阅发生错误")
}

func (r *RedisCache) Ping() (string, error) {
	return r.Client.Ping(context.Background()).Result()
}

func (r *RedisCache) Close() error {
	return r.Client.Close()
}
