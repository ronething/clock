package config

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var Config *viper.Viper

func SetConfig(filePath string) {
	log.Infof("[config] run the env with:%s", filePath)

	Config = viper.New()
	Config.SetConfigFile(filePath)
	if err := Config.ReadInConfig(); err != nil {
		log.Fatalf("[config] read config err: %v", err)
	}

	initLog()
	watchFileConfig()
}

func initLog() {

	level := Config.GetString("log.level")
	switch level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	caller := Config.GetBool("log.caller")
	log.SetReportCaller(caller)
	jsonFormat := Config.GetBool("log.jsonformat")
	log.SetFormatter(func() log.Formatter {
		if jsonFormat {
			return &log.JSONFormatter{}
		} else {
			return &log.TextFormatter{}
		}
	}())
}

//watchFileConfig 监听文件变化
func watchFileConfig() {
	Config.WatchConfig()
	Config.OnConfigChange(func(e fsnotify.Event) {
		log.Warnf("config file change: %v %v", e.Name, e.Op)
		if e.Op == fsnotify.Write {
			initLog()
		}
	})
}
