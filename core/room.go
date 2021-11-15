package core

import (
	. "point-set/base"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	RoomIniting uint8 = 1
	RoomRunning uint8 = 2
	RoomStopped uint8 = 3
)

type Room struct {
	roomId    string
	createdAt time.Time
	duration  time.Duration
	maxFrame  uint32
	configs   map[uint32]*PlayerConfig
	chFinish  chan<- string

	// multi-thread fields
	_mutex     sync.RWMutex
	_state     uint8
	_readySet  map[uint32]bool
	_players   map[uint32]*Player
	_startedAt int64
}

func NewRoom(
	roomId string,
	duration time.Duration,
	configs map[uint32]*PlayerConfig,
	chFinish chan<- string,
) *Room {
	room := &Room{
		roomId:    roomId,
		createdAt: time.Now(),
		duration:  duration,
		maxFrame:  uint32(duration.Seconds()) * FPS,
		configs:   configs,
		chFinish:  chFinish,

		_mutex:     sync.RWMutex{},
		_state:     RoomIniting,
		_readySet:  make(map[uint32]bool, len(configs)),
		_players:   make(map[uint32]*Player, len(configs)),
		_startedAt: 0,
	}

	room.logInfo(LogFields{
		"duration": duration,
		"configs":  configs,
	}, "create room")

	return room
}

func (r *Room) RoomId() string {
	return r.roomId
}

func (r *Room) CreatedAt() time.Time {
	return r.createdAt
}

func (r *Room) MaxFrame() uint32 {
	return r.maxFrame
}

func (r *Room) StartedAt() time.Time {
	return time.UnixMilli(atomic.LoadInt64(&r._startedAt))
}

func (r *Room) MaxPlayers() int {
	return len(r.configs)
}

func (r *Room) GetPlayers(players []*Player) []*Player {
	r._mutex.RLock()
	defer r._mutex.RUnlock()

	for _, player := range r._players {
		players = append(players, player)
	}
	return players
}

func (r *Room) Enter(session ISession) error {
	r.logInfo(LogFields{"conv": session.GetConv()}, "enter room")

	player, err := r.enter(session)
	if err != nil {
		return err
	}
	if !InUnitTest {
		go player.Update()
	}
	return nil
}

func (r *Room) enter(session ISession) (*Player, error) {
	r._mutex.Lock()
	defer r._mutex.Unlock()

	if session == nil {
		return nil, ErrArguments
	}
	if r._state != RoomIniting {
		return nil, errors.WithStack(ErrRoomState)
	}

	config := r.configs[session.GetConv()]
	if config == nil {
		return nil, errors.WithStack(ErrPlayerNotFound)
	}
	if _, ok := r._players[session.GetConv()]; ok {
		return nil, errors.WithStack(ErrPlayerExisted)
	}

	player, err := NewPlayer(config, r, session)
	if err != nil {
		return nil, err
	}
	r._players[config.Conv] = player

	return player, nil
}

func (r *Room) Connect(conv uint32) (running bool, err error) {
	r.logInfo(LogFields{"conv": conv}, "connect room")

	ready, err := r.connect(conv)
	if err != nil {
		return false, err
	}
	if ready {
		atomic.StoreInt64(&r._startedAt, time.Now().UnixMilli())
	}
	return ready, nil
}

func (r *Room) connect(conv uint32) (running bool, err error) {
	r._mutex.Lock()
	defer r._mutex.Unlock()

	if r._state != RoomIniting {
		return false, errors.WithStack(ErrRoomState)
	}
	if _, ok := r._players[conv]; !ok {
		return false, errors.WithStack(ErrPlayerNotFound)
	}

	r._readySet[conv] = true
	if len(r._readySet) == len(r.configs) {
		r._state = RoomRunning
		return true, nil
	}
	return false, nil
}

func (r *Room) Leave(conv uint32) error {
	r.logInfo(LogFields{"conv": conv}, "leave room")

	stopped, err := r.leave(conv)
	if err != nil {
		return err
	}
	if stopped {
		r.chFinish <- r.roomId
	}
	return nil
}

func (r *Room) leave(conv uint32) (stopped bool, err error) {
	r._mutex.Lock()
	defer r._mutex.Unlock()

	if _, ok := r._players[conv]; !ok {
		return false, errors.WithStack(ErrPlayerNotFound)
	}
	delete(r._players, conv)
	delete(r._readySet, conv)

	if len(r._players) != 0 {
		return false, nil
	}
	r._state = RoomStopped
	return true, nil
}

func (r *Room) Close() {
	r.logInfo(nil, "close room")

	var players []*Player
	players = r.GetPlayers(players)
	for _, player := range players {
		player.Close()
	}
}

func (r *Room) logInfo(fields LogFields, args ...interface{}) {
	if fields == nil {
		fields = LogFields{}
	}
	fields["source"] = "Room"
	fields["room_id"] = r.roomId
	fields["state"] = r._state
	fields["players"] = len(r._players)
	LogPrint(LevelInfo, fields, args...)
}
