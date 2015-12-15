package henchman

import (
	logrus "gopkg.in/Sirupsen/logrus.v0"
	"os"
)

var jsonLog = logrus.New()

func InitLog() {
	jsonLog.Level = logrus.DebugLevel
	jsonLog.Formatter = new(logrus.JSONFormatter)

	// NOTE: hardcoded for now
	f, _ := os.OpenFile(LogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	jsonLog.Out = f
}

// wrapper for debug
func Debug(fields map[string]interface{}, msg string) {
	if DebugFlag {
		if fields == nil {
			jsonLog.Debug(msg)
		} else {
			jsonLog.WithFields(fields).Debug(msg)
		}
	}
}

// wrapper for Info
func Info(fields map[string]interface{}, msg string) {
	if fields == nil {
		jsonLog.Info(msg)
	} else {
		jsonLog.WithFields(fields).Info(msg)
	}
}

// wrapper for Fatal
func Fatal(fields map[string]interface{}, msg string) {
	if fields == nil {
		logrus.Error(msg)
		jsonLog.Fatal(msg)
	} else {
		logrus.WithFields(fields).Error(msg)
		jsonLog.WithFields(fields).Fatal(msg)
	}
}

// wrapper for Error
func Error(fields map[string]interface{}, msg string) {
	if fields == nil {
		jsonLog.Error(msg)
		logrus.Error(msg)
	} else {
		jsonLog.WithFields(fields).Error(msg)
		logrus.WithFields(fields).Error(msg)
	}
}

func Warn(fields map[string]interface{}, msg string) {
	if fields == nil {
		jsonLog.Warn(msg)
		logrus.Warn(msg)
	} else {
		jsonLog.WithFields(fields).Warn(msg)
		logrus.WithFields(fields).Warn(msg)
	}
}
