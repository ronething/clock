package storage

import (
	log "github.com/sirupsen/logrus"
)

//SchedulerDeleteTask 调度器删除对应任务
func NewSchedulerDeleteTask(t interface{}) error {
	switch t.(type) {
	case Task:
		task := t.(Task)
		cronScheduler.RemoveTask(&task)
	case string:
		tid := t.(string)
		cronScheduler.RemoveTaskByTid(tid)
	default:
		log.Printf("不支持的类型")
	}
	return nil
}

//NewInitPutTask 初始化的时候将 disable 为 false 的任务拉起
func NewInitPutTask(t Task) error {
	if t.Disable {
		log.Debugf("[scheduler] 任务 %s-%s 不需要添加到定时调度器中\n", t.Tid, t.Name)
	} else {
		err := cronScheduler.AddTask(&t)
		if err != nil {
			log.Errorf("[scheduler] 添加任务 %s-%s 失败 %v", t.Tid, t.Name, err)
		}
	}

	return nil
}

//NewSchedulerPutTask 用于程序运行过程中的 put task 支持 多 timezone
//考虑原先时区的问题，需要进行原时区的任务删除,然后将任务新增到新时区
func NewSchedulerModifyTask(t Task) error {
	// 移除并重新启用
	cronScheduler.RemoveTask(&t)
	err := cronScheduler.AddTask(&t)
	if err != nil {
		return err
	}
	return nil
}

func NewSchedulerDisableTask(t Task) error {
	// 先移除再说
	cronScheduler.RemoveTask(&t)
	if t.Disable {
		log.Infof("[scheduler] 成功禁用任务 %s-%s\n", t.Tid, t.Name)
	} else {
		err := cronScheduler.AddTask(&t)
		if err != nil {
			return err
		}
	}
	return nil
}
