package main

import (
	"time"

	"github.com/pkg/errors"
)

const (
	KCPWindowSize = 160 // 16s
	KCPMtx        = 470

	RoomPrepareTimeout = time.Second * 30

	Team0 TeamNo = 0
	Team1 TeamNo = 1
	Team2 TeamNo = 2
	Team3 TeamNo = 3
	Team4 TeamNo = 4
)

type TeamNo uint8

var (
	ErrRoomExisted = errors.New("Room existed")
)
