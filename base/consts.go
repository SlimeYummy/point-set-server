package base

import (
	"time"

	"github.com/pkg/errors"
)

const (
	KCPWindowSize = 160 // 16 seconds
	KCPMtx        = 470

	FPS = 10

	JoinTimeout   = time.Second * 30
	MinPacketSize = 5
	MaxPacketSize = KCPMtx * 3
	SyncLowLimit  = time.Second * 5 // 5 seconds
	SyncHighLimit = time.Second * 2 // 2 seconds
)

var (
	ErrUnexpected  = errors.New("Unexpected error")
	ErrOverflow    = errors.New("Overflow error")
	ErrShortPacket = errors.New("Packet too short")
	ErrLogPacket   = errors.New("Packet too long")
	ErrBadType     = errors.New("Bad type")

	ErrRoomNotFound = errors.New("Room not found")
	ErrRoomExisted  = errors.New("Room existed")
	ErrRoomStarted  = errors.New("Room started")

	ErrPlayerNotFound  = errors.New("Player not found")
	ErrPlayerExisted   = errors.New("Player existed")
	ErrPlayerStart     = errors.New("Player start failed")
	ErrPlayerStop      = errors.New("Player stop failed")
	ErrInvalidPassword = errors.New("Invalid password")
	ErrInvalidConv     = errors.New("Invalid conv")
)
