package main

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	levelError = 1
	levelWarn  = 2
	levelInfo  = 3
)

func LogError(roomId string, playerId string, conv uint32, args ...interface{}) {
	logPrint(levelError, roomId, playerId, conv, args...)
}

func LogWran(roomId string, playerId string, conv uint32, args ...interface{}) {
	logPrint(levelWarn, roomId, playerId, conv, args...)
}

func LogInfo(roomId string, playerId string, conv uint32, args ...interface{}) {
	logPrint(levelInfo, roomId, playerId, conv, args...)
}

func LogPlayerError(p *Player, args ...interface{}) {
	logPrint(levelError, p.room.id, p.playerId, p.session.GetConv(), args...)
}

func LogPlayerWarn(p *Player, args ...interface{}) {
	logPrint(levelWarn, p.room.id, p.playerId, p.session.GetConv(), args...)
}

func LogPlayerInfo(p *Player, args ...interface{}) {
	logPrint(levelInfo, p.room.id, p.playerId, p.session.GetConv(), args...)
}

type errorTracer interface {
	Error() string
	StackTrace() errors.StackTrace
}

func logPrint(level int, roomId string, playerId string, conv uint32, args ...interface{}) {
	fields := log.Fields{}
	if roomId != "" {
		fields["room_id"] = roomId
	}
	if playerId != "" {
		fields["player_id"] = playerId
	}
	if conv != 0 {
		fields["conv"] = conv
	}

	if len(args) == 1 {
		if err, ok := args[0].(errorTracer); ok {
			frames := err.StackTrace()
			stack := make([]string, len(frames))
			for _, frame := range frames {
				stack = append(stack, fmt.Sprintf("%+s:%d\n", frame, frame))
			}
			fields["stack"] = stack

			switch level {
			case levelError:
				log.WithFields(fields).Error(err.Error())
			case levelWarn:
				log.WithFields(fields).Warn(err.Error())
			case levelInfo:
				log.WithFields(fields).Info(err.Error())
			}
			return
		}
	}

	switch level {
	case levelError:
		log.WithFields(fields).Error(fmt.Sprint(args...))
	case levelWarn:
		log.WithFields(fields).Warn(fmt.Sprint(args...))
	case levelInfo:
		log.WithFields(fields).Info(fmt.Sprint(args...))
	}
}
