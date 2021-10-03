package main

import (
	"point-set/msg"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	KCPWindowSize = 160 // 16s
	KCPMtx        = 470

	RoomInittedTimeout = time.Second * 30
)

type TeamNo uint8

const (
	Team0 TeamNo = 0
	Team1 TeamNo = 1
	Team2 TeamNo = 2
	Team3 TeamNo = 3
	Team4 TeamNo = 4
)

type RoomState uint32

const (
	RoomInitted RoomState = 1
	RoomRunning RoomState = 2
	RoomStopped RoomState = 3
)

type PlayerState uint32

const (
	PlayerInitted PlayerState = 1
	PlayerRunning PlayerState = 2
	PlayerStopped PlayerState = 3
)

var (
	ErrRoomExisted = errors.New("Room existed")
	ErrEmptyData   = errors.New("Empty data")
	ErrBadType     = errors.New("Bad type")
	ErrPlayerState = errors.New("Invalid player state")
)

func DecodeMsgInit(data []byte) (*msg.MsgInit, error) {
	if len(data) == 0 {
		return nil, ErrEmptyData
	}
	if data[len(data)-1] != byte(msg.Type_Init) {
		return nil, ErrBadType
	}

	var m msg.MsgInit
	err := proto.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func EncodeMsgStart(start *msg.MsgStart) ([]byte, error) {
	data, err := proto.Marshal(start)
	if err != nil {
		return nil, err
	}
	data = append(data, byte(msg.Type_Start))
	return data, nil
}

func EncodeMsgFinish(start *msg.MsgFinish) ([]byte, error) {
	data, err := proto.Marshal(start)
	if err != nil {
		return nil, err
	}
	data = append(data, byte(msg.Type_Finish))
	return data, nil
}

func DecodeMsgFinish(data []byte) (*msg.MsgFinish, error) {
	if len(data) == 0 {
		return nil, ErrEmptyData
	}
	if data[len(data)-1] != byte(msg.Type_Finish) {
		return nil, ErrBadType
	}

	var m msg.MsgFinish
	err := proto.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// const (
// 	MsgErr
// )

func EncodeMsgError(error *msg.MsgError) ([]byte, error) {
	data, err := proto.Marshal(error)
	if err != nil {
		return nil, err
	}
	data = append(data, byte(msg.Type_Error))
	return data, nil
}
