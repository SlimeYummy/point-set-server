package base

import (
	"os"
	"time"

	"github.com/pkg/errors"
)

const (
	KCPWindowSize = 256
	KCPMtx        = 470

	FPS           = 10
	MinPacketSize = 3
	MaxPacketSize = KCPMtx * 4
)

var TimeZero = time.Time{}

const (
	ListenTimeout  = time.Second * 5
	ConnectTimeout = time.Second * 10
	StartTimeout   = time.Second * 20
	SyncLowLimit   = time.Second * 5
	SyncHighLimit  = time.Second * 2
)

var (
	ErrUnexpected = errors.New("unexpected error")
	ErrArguments  = errors.New("invalid arguments")

	ErrMessageType = errors.New("invalid message type") // local message/packet
	ErrMessageSize = errors.New("invalid message size") // local message/packet

	ErrRoomNotFound = errors.New("room not found")
	ErrRoomExisted  = errors.New("room existed")
	ErrRoomState    = errors.New("invalid room state")

	ErrPlayerNotFound = errors.New("player not found")
	ErrPlayerExisted  = errors.New("player existed")

	// network borken
	ErrNetworkBroken = errors.New("network broken")

	// invalid packet
	ErrPacketBroken = errors.New("packet is broken")    // remote message/packet
	ErrPacketSize   = errors.New("invalid packet size") // remote message/packet

	// auth failed
	ErrAuthFailed = errors.New("authorization failed")

	// time out of sync
	ErrTimeOutOfSync = errors.New("time out of sync")

	// data out of sync
	ErrDataOutOfSync = errors.New("data out of sync")

	// other
	ErrRemoteFinish = errors.New("remote finish")
	ErrLocalFinish  = errors.New("local finish")
)

var InUnitTest = false
var InDebug = false

func init() {
	InUnitTest = os.Getenv("UNIT_TEST") != ""
	InDebug = os.Getenv("DEBUG") != ""
}
