package core

import (
	. "point-set/base"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	tRid  = "mock-room-id"
	tDura = time.Minute * 15
	tChan = make(chan string, 10)

	tCfg1 = &PlayerConfig{
		PlayerId: "player-1",
		Team:     Team1,
		Password: "password-1",
		Conv:     123,
	}
	tCfg2 = &PlayerConfig{
		PlayerId: "player-2",
		Team:     Team2,
		Password: "password-2",
		Conv:     456,
	}
	tCfgs = map[uint32]*PlayerConfig{
		123: tCfg1,
		456: tCfg2,
	}
)

func TestNewRoom(t *testing.T) {
	room := NewRoom(tRid, tDura, tCfgs, tChan)
	assert.True(t, time.Since(room.CreatedAt()) < time.Millisecond)
	assert.Equal(t, uint32(tDura.Seconds())*FPS, room.MaxFrame())
	assert.Equal(t, time.UnixMilli(0), room.StartedAt())
}

func TestRoomEnter(t *testing.T) {
	room := NewRoom(tRid, tDura, tCfgs, tChan)

	err := room.Enter(nil)
	assert.ErrorIs(t, err, ErrArguments)

	s1 := &MockSession{}
	s1.On("GetConv").Return(uint32(123))
	s2 := &MockSession{}
	s2.On("GetConv").Return(uint32(999))

	err = room.Enter(s1)
	assert.Equal(t, nil, err)
	assert.True(t, room._players[123] != nil)

	err = room.Enter(s1)
	assert.ErrorIs(t, err, ErrPlayerExisted)

	err = room.Enter(s2)
	assert.ErrorIs(t, err, ErrPlayerNotFound)

	room._state = RoomRunning
	err = room.Enter(s2)
	assert.ErrorIs(t, err, ErrRoomState)
}

func TestRoomConnect(t *testing.T) {
	room := NewRoom(tRid, tDura, tCfgs, tChan)
	s1 := &MockSession{}
	s1.On("GetConv").Return(uint32(123))
	s2 := &MockSession{}
	s2.On("GetConv").Return(uint32(456))

	err := room.Enter(s1)
	assert.Equal(t, nil, err)

	_, err = room.Connect(777)
	assert.ErrorIs(t, err, ErrPlayerNotFound)

	running, err := room.Connect(123)
	assert.Equal(t, nil, err)
	assert.Equal(t, false, running)
	assert.Equal(t, time.UnixMilli(0), room.StartedAt())
	assert.Equal(t, RoomIniting, room._state)

	err = room.Enter(s2)
	assert.Equal(t, nil, err)

	running, err = room.Connect(456)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, running)
	assert.True(t, time.Since(room.StartedAt()) < time.Millisecond)
	assert.Equal(t, RoomRunning, room._state)

	_, err = room.Connect(456)
	assert.ErrorIs(t, err, ErrRoomState)
}

func TestRoomLeave(t *testing.T) {
	room := NewRoom(tRid, tDura, tCfgs, tChan)
	s1 := &MockSession{}
	s1.On("GetConv").Return(uint32(123))
	s2 := &MockSession{}
	s2.On("GetConv").Return(uint32(456))

	err := room.Enter(s1)
	assert.Equal(t, nil, err)
	_, err = room.Connect(123)
	assert.Equal(t, nil, err)

	err = room.Enter(s2)
	assert.Equal(t, nil, err)
	_, err = room.Connect(456)
	assert.Equal(t, nil, err)

	err = room.Leave(123)
	assert.Equal(t, nil, err)
	assert.Equal(t, RoomRunning, room._state)
	assert.Equal(t, 1, len(room._players))
	assert.Equal(t, 1, len(room._readySet))

	err = room.Leave(456)
	assert.Equal(t, nil, err)
	assert.Equal(t, RoomStopped, room._state)
	assert.Equal(t, 0, len(room._players))
	assert.Equal(t, 0, len(room._readySet))
	assert.Equal(t, "mock-room-id", <-tChan)
}
