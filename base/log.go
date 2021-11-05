package base

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	LevelError = 1
	LevelWarn  = 2
	LevelInfo  = 3
)

type LogFields = log.Fields

type errorTracer interface {
	Error() string
	StackTrace() errors.StackTrace
}

func LogPrint(level int, fields log.Fields, args ...interface{}) {
	if len(args) == 1 {
		if err, ok := args[0].(errorTracer); ok {
			frames := err.StackTrace()
			stack := make([]string, len(frames))
			for _, frame := range frames {
				stack = append(stack, fmt.Sprintf("%+s:%d\n", frame, frame))
			}
			fields["stack"] = stack

			switch level {
			case LevelError:
				log.WithFields(fields).Error(err.Error())
			case LevelWarn:
				log.WithFields(fields).Warn(err.Error())
			case LevelInfo:
				log.WithFields(fields).Info(err.Error())
			}
			return
		}
	}

	switch level {
	case LevelError:
		log.WithFields(fields).Error(fmt.Sprint(args...))
	case LevelWarn:
		log.WithFields(fields).Warn(fmt.Sprint(args...))
	case LevelInfo:
		log.WithFields(fields).Info(fmt.Sprint(args...))
	}
}
