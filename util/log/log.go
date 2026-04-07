package log

import (
	"main/conf"

	"github.com/lvyonghuan/Ubik-Util/ulog"
)

type Log interface {
	Debug(v string)
	Info(v string)
	Warn(v string)
	Error(v error)
	Fatal(v error)
	System(v string)
}

var Logger Log

func InitLoggr(logConf conf.LogConfig) {
	Logger = ulog.NewULog(logConf.Level, logConf.WriteLevel, true, logConf.LogFilePath)
}
