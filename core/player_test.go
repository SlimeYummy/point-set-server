package core

import (
	. "point-set/base"
	. "point-set/codec"
	msg "point-set/message"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewPlayer(t *testing.T) {
	room := NewRoom(tRid, tDura, tCfgs, tChan)
	sess := &MockSession{}
	sess.On("GetConv").Return(uint32(123))

	_, err := NewPlayer(nil, room, sess)
	assert.ErrorIs(t, err, ErrArguments)
	_, err = NewPlayer(tCfg1, nil, sess)
	assert.ErrorIs(t, err, ErrArguments)
	_, err = NewPlayer(tCfg1, room, nil)
	assert.ErrorIs(t, err, ErrArguments)

	player, err := NewPlayer(tCfg1, room, sess)
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, tCfg1.Conv, player.Conv())
	assert.Equal(t, tCfg1.PlayerId, player.PlayerId())
	assert.Equal(t, msg.NetPlayerState_Initing, player.state)
}

func prepare() (*MockSession, *Room, *Player, *Player) {
	sess := &MockSession{}
	sess.On("GetConv").Return(uint32(123))

	room := NewRoom(tRid, tDura, tCfgs, tChan)
	room._startedAt = time.Now().UnixMilli()

	player1, err := NewPlayer(tCfg1, room, sess)
	if err != nil {
		panic(err)
	}
	room._players[tCfg1.Conv] = player1

	player2, err := NewPlayer(tCfg1, room, sess)
	if err != nil {
		panic(err)
	}
	room._players[tCfg2.Conv] = player2

	return sess, room, player1, player2
}

func TestPlayerKCPIniting(t *testing.T) {
	var buffer []byte
	sess, _, player1, player2 := prepare()
	player2.state = msg.NetPlayerState_Waiting

	buffer, _ = EncodeMessage(&msg.NetConnect{}, []byte{})
	err := player1.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrAuthFailed)
	assert.Equal(t, msg.NetPlayerState_Initing, player1.state)

	{
		buffer, _ = EncodeMessage(&msg.NetState{
			Conv:  player2.Conv(),
			State: msg.NetPlayerState_Waiting,
		}, []byte{})
		sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)

		buffer, _ = EncodeMessage(&msg.NetConnect{
			RoomId:   tRid,
			PlayerId: tCfg1.PlayerId,
			Password: tCfg1.Password,
		}, []byte{})
		err = player1.handleKCP(buffer)
		assert.Equal(t, nil, err)
		assert.Equal(t, msg.NetPlayerState_Waiting, player1.state)
		assert.True(t, time.Since(player1.deadline) < time.Millisecond)

		assert.Equal(t, 1, len(player2.channel))
		state := (<-player2.channel).(*msg.NetState)
		assert.Equal(t, player1.Conv(), state.Conv)
		assert.Equal(t, msg.NetPlayerState_Waiting, state.State)
	}

	buffer, _ = EncodeMessage(&msg.NetFinish{}, []byte{})
	err = player1.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrRemoteFinish)

	buffer, _ = EncodeMessage(&msg.NetCommand{}, []byte{})
	err = player1.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrPacketBroken)
}

func TestPlayerKCPWaiting(t *testing.T) {
	var buffer []byte
	_, _, player, _ := prepare()
	player.state = msg.NetPlayerState_Waiting

	buffer, _ = EncodeMessage(&msg.NetFinish{}, []byte{})
	err := player.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrRemoteFinish)

	buffer, _ = EncodeMessage(&msg.NetCommand{}, []byte{})
	err = player.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrPacketBroken)
}

func TestPlayerKCPRunning(t *testing.T) {
	var buffer []byte
	sess, _, player1, player2 := prepare()
	player1.state = msg.NetPlayerState_Running
	player2.state = msg.NetPlayerState_Running

	buffer, _ = EncodeMessage(&msg.NetHash{}, []byte{})
	err := player1.handleKCP(buffer)
	assert.Equal(t, nil, err)

	buffer, _ = EncodeMessage(&msg.NetCommand{Frame: 0}, []byte{})
	err = player1.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrTimeOutOfSync)

	{
		player1.cmdHeap.Push(&CommandBuffer{Frame: 0, Buffer: []byte{5, 5}})
		player1.cmdHeap.Push(&CommandBuffer{Frame: 0, Buffer: []byte{6, 6}})
		sess.On("SendBatch", [][]byte{{5, 5}, {6, 6}}, mock.Anything).Return(0, nil)

		buffer, _ = EncodeMessage(&msg.NetCommand{Frame: 1, Conv: 7878}, []byte{})
		buffer = append(buffer, 9, 8, 7, 6, 5)
		err = player1.handleKCP(buffer)
		assert.Equal(t, nil, err)

		assert.Equal(t, 1, len(player2.channel))
		cb := (<-player2.channel).(*CommandBuffer)
		assert.Equal(t, uint32(1), cb.Frame)
		assert.Equal(t, tCfg1.Team, cb.PlayerTeam)

		cmd, offset, _ := DecodeMessage(cb.Buffer)
		assert.Equal(t, uint32(1), cmd.(*msg.NetCommand).Frame)
		assert.Equal(t, tCfg1.Conv, cmd.(*msg.NetCommand).Conv)
		assert.Equal(t, []byte{9, 8, 7, 6, 5}, cb.Buffer[offset:])
	}

	buffer, _ = EncodeMessage(&msg.NetFinish{}, []byte{})
	err = player2.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrRemoteFinish)

	buffer, _ = EncodeMessage(&msg.NetAccept{}, []byte{})
	err = player2.handleKCP(buffer)
	assert.ErrorIs(t, err, ErrPacketBroken)
}

func TestPlayerKCPStopped(t *testing.T) {
	var buffer []byte
	_, _, player, _ := prepare()
	player.state = msg.NetPlayerState_Stopped

	buffer, _ = EncodeMessage(&msg.NetFinish{}, []byte{})
	err := player.handleKCP(buffer)
	assert.ErrorIs(t, err, nil)
}

func TestPlayerChanIniting(t *testing.T) {
	var buffer []byte
	sess, _, player, _ := prepare()

	err := player.handleChan(&msg.NetState{})
	assert.Equal(t, nil, err)

	finish := &msg.NetFinish{Cause: 3}
	buffer, _ = EncodeMessage(finish, []byte{})
	sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)
	err = player.handleChan(finish)
	assert.ErrorIs(t, err, ErrLocalFinish)

	err = player.handleChan(&msg.NetConnect{})
	assert.ErrorIs(t, err, ErrMessageType)
}

func TestPlayerChanWaiting(t *testing.T) {
	var buffer []byte
	sess, _, player1, player2 := prepare()
	player1.state = msg.NetPlayerState_Waiting

	state := &msg.NetState{}
	buffer, _ = EncodeMessage(state, []byte{})
	sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)
	err := player1.handleChan(state)
	assert.Equal(t, nil, err)

	{
		start := &msg.NetStart{}
		buffer, _ = EncodeMessage(start, []byte{})
		sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)

		err = player1.handleChan(start)
		assert.Equal(t, nil, err)
		assert.Equal(t, msg.NetPlayerState_Running, player1.state)
		assert.True(t, time.Now().Add(SyncLowLimit).Sub(player1.deadline) < time.Millisecond)

		assert.Equal(t, 1, len(player2.channel))
		state2 := (<-player2.channel).(*msg.NetState)
		assert.Equal(t, player1.Conv(), state2.Conv)
		assert.Equal(t, msg.NetPlayerState_Running, state2.State)
	}

	finish := &msg.NetFinish{Cause: 3}
	buffer, _ = EncodeMessage(finish, []byte{})
	sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)
	err = player1.handleChan(finish)
	assert.ErrorIs(t, err, ErrLocalFinish)

	err = player1.handleChan(&msg.NetConnect{})
	assert.ErrorIs(t, err, ErrMessageType)
}

func TestPlayerChanRunning(t *testing.T) {
	var buffer []byte
	sess, _, player, _ := prepare()
	player.state = msg.NetPlayerState_Running

	state := &msg.NetState{}
	buffer, _ = EncodeMessage(state, []byte{})
	sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)
	err := player.handleChan(state)
	assert.Equal(t, nil, err)

	err = player.handleChan(&CommandBuffer{
		Frame:      1,
		PlayerTeam: Team2,
		Buffer:     []byte{9, 9, 9, 9},
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, player.cmdHeap.Len())

	buffer, _ = EncodeMessage(&msg.NetCommand{Frame: 10, Conv: tCfg1.Conv}, []byte{})
	buffer = append(buffer, 7, 7, 7, 7)
	sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)
	err = player.handleChan(&CommandBuffer{
		Frame:      10,
		PlayerTeam: tCfg1.Team,
		Buffer:     buffer,
	})
	assert.Equal(t, nil, err)

	buffer, _ = EncodeMessage(&msg.NetCommand{Frame: 0, Conv: tCfg2.Conv}, []byte{})
	buffer = append(buffer, 6, 6, 6, 6)
	sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)
	err = player.handleChan(&CommandBuffer{
		Frame:      1,
		PlayerTeam: tCfg2.Team,
		Buffer:     buffer,
	})
	assert.Equal(t, nil, err)

	finish := &msg.NetFinish{Cause: 3}
	buffer, _ = EncodeMessage(finish, []byte{})
	sess.On("Send", buffer, mock.Anything).Return(len(buffer), nil)
	err = player.handleChan(finish)
	assert.ErrorIs(t, err, ErrLocalFinish)

	err = player.handleChan(&msg.NetConnect{})
	assert.ErrorIs(t, err, ErrMessageType)
}

func TestPlayerChanStopped(t *testing.T) {
	var buffer []byte
	_, _, player, _ := prepare()
	player.state = msg.NetPlayerState_Stopped

	buffer, _ = EncodeMessage(&msg.NetFinish{}, []byte{})
	err := player.handleChan(buffer)
	assert.Equal(t, nil, err)
}
