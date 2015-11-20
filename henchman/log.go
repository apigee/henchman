package henchman

import (
	log "gopkg.in/Sirupsen/logrus.v0"
)

// wrapper for debug
func Debug(fields map[string]interface{}, msg string) {
	if DebugFlag {
		if fields == nil {
			log.Debug(msg)
		} else {
			log.WithFields(fields).Debug(msg)
		}
	}
}

// wrapper for Info
func Info(fields map[string]interface{}, msg string) {
	if fields == nil {
		log.Info(msg)
	} else {
		log.WithFields(fields).Info(msg)
	}
}

// wrapper for Fatal
func Fatal(fields map[string]interface{}, msg string) {
	if fields == nil {
		log.Fatal(msg)
	} else {
		log.WithFields(fields).Fatal(msg)
	}
}

func Error(fields map[string]interface{}, msg string) {
	if fields == nil {
		log.Error(msg)
	} else {
		log.WithFields(fields).Error(msg)
	}
}

func Warn(fields map[string]interface{}, msg string) {
	if fields == nil {
		log.Warn(msg)
	} else {
		log.WithFields(fields).Warn(msg)
	}
}
