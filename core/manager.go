package core

import (
	"fmt"
	. "point-set/base"
	"sync"
	"sync/atomic"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go/v5"
)

type RoomManager struct {
	listener  *kcp.Listener
	chFinish  chan string
	finishSet []string

	// multi-thread fields
	_mutex sync.Mutex
	_rooms map[string]*Room
	_convs map[uint32]*Room
}

func NewRoomManager(addr string) (*RoomManager, error) {
	listener, err := kcp.ListenWithOptions(addr, nil, 0, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &RoomManager{
		listener:  listener,
		chFinish:  make(chan string, 1024),
		finishSet: make([]string, 0, 128),

		_mutex: sync.Mutex{},
		_rooms: make(map[string]*Room, 256),
		_convs: make(map[uint32]*Room, 256),
	}, nil
}

func (m *RoomManager) CreateRoom(
	roomId string,
	duration time.Duration,
	players []PlayerBasic,
) ([]*PlayerConfig, error) {
	cfgsMap := make(map[uint32]*PlayerConfig, len(players))
	cfgsList := make([]*PlayerConfig, 0, len(players))
	for _, player := range players {
		cfg := &PlayerConfig{
			PlayerId: player.PlayerId,
			Team:     player.Team,
			Password: genPassword(),
			Conv:     genConv(),
		}
		cfgsMap[cfg.Conv] = cfg
		cfgsList = append(cfgsList, cfg)
	}

	m._mutex.Lock()
	defer m._mutex.Unlock()

	room := NewRoom(roomId, duration, cfgsMap, m.chFinish)
	if _, ok := m._rooms[roomId]; ok {
		return nil, errors.WithStack(ErrRoomExisted)
	}
	m._rooms[roomId] = room

	for _, config := range cfgsMap {
		m._convs[config.Conv] = room
	}

	return cfgsList, nil
}

func (m *RoomManager) DeleteRoom(roomId string) error {
	m._mutex.Lock()
	room, ok := m._rooms[roomId]
	if !ok {
		m._mutex.Unlock()
		return errors.WithStack(ErrRoomNotFound)
	}
	m._mutex.Unlock()

	room.Close()
	return nil
}

func (m *RoomManager) Listen() error {
	for {
		m.listener.SetReadDeadline(time.Now().Add(ListenTimeout))
		session, err := m.listener.AcceptKCP()
		if err != nil {
			if errors.Is(err, kcp.ErrTimeout) {
				m.handleTimeout()
			} else {
				return errors.WithStack(err)
			}
		} else {
			m.handleSession(session)
		}
	}
}

func (m *RoomManager) handleSession(session *kcp.UDPSession) {
	m._mutex.Lock()
	room, ok := m._convs[session.GetConv()]
	m._mutex.Unlock()
	if !ok {
		fmt.Printf("!!!!!!!!!!!!!!! %v %v", session.GetConv(), m._convs)
		m.logWarn(LogFields{
			"conv": session.GetConv(),
		}, errors.WithStack(ErrRoomNotFound))
		return
	}

	err := room.Enter(session)
	if err != nil {
		m.logWarn(LogFields{"conv": session.GetConv()}, err)
	}
}

func (m *RoomManager) handleTimeout() {
	now := time.Now()

	defer func() { m.finishSet = m.finishSet[:0] }()
	for len(m.chFinish) > 0 {
		m.finishSet = append(m.finishSet, <-m.chFinish)
	}

	m._mutex.Lock()
	defer m._mutex.Unlock()

	for _, roomId := range m.finishSet {
		delete(m._rooms, roomId)
	}
	for conv, room := range m._convs {
		if now.Sub(room.CreatedAt()) > ConnectTimeout {
			delete(m._convs, conv)
		}
	}
}

func (m *RoomManager) CreateTestRoom() {
	cfgsMap := map[uint32]*PlayerConfig{
		100: {
			PlayerId: "p1",
			Team:     Team1,
			Password: "",
			Conv:     100,
		},
		200: {
			PlayerId: "p2",
			Team:     Team2,
			Password: "",
			Conv:     200,
		},
	}

	m._mutex.Lock()
	defer m._mutex.Unlock()

	roomId := "r1"
	room := NewRoom(roomId, time.Minute*40, cfgsMap, m.chFinish)
	if _, ok := m._rooms[roomId]; ok {
		panic(ErrRoomExisted)
	}
	m._rooms[roomId] = room

	for _, config := range cfgsMap {
		m._convs[config.Conv] = room
	}
}

func (r *RoomManager) logWarn(fields LogFields, args ...interface{}) {
	if fields == nil {
		fields = LogFields{}
	}
	fields["source"] = "RoomManager"
	LogPrint(LevelWarn, fields, args...)
}

func genPassword() string {
	password, err := gonanoid.Nanoid()
	if err != nil {
		panic(err)
	}
	return password
}

var counter uint32

func genConv() uint32 {
	t := uint32(time.Now().Second())
	n := atomic.AddUint32(&counter, 7)
	return (n&0x7FFF)<<17 | t&0x1FFFF
}
