package core

import (
	"fmt"
	. "point-set/base"
	. "point-set/codec"
	msg "point-set/message"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go/v5"
	"google.golang.org/protobuf/proto"
)

const (
	Team0 uint8 = 0
	Team1 uint8 = 1
	Team2 uint8 = 2
	Team3 uint8 = 3
	Team4 uint8 = 4
)

const sendBufSize = 16

type PlayerBasic struct {
	PlayerId string `json:"player_id"`
	Team     uint8  `json:"team"`
}

type PlayerConfig struct {
	PlayerId string `json:"player_id"`
	Team     uint8  `json:"team"`
	Password string `json:"password"`
	Conv     uint32 `json:"conv"`
}

type Player struct {
	// readonly fields
	room    *Room
	config  *PlayerConfig
	session ISession

	// mutable fields
	channel  chan interface{}
	sendBuf  []byte
	recvBuf  []byte
	state    msg.NetPlayerState
	frame    uint32
	deadline time.Time
	cmdHeap  *CommandHeap
	cmdBufs  [][]byte
	players  []*Player
}

func NewPlayer(
	config *PlayerConfig,
	room *Room,
	session ISession,
) (*Player, error) {
	if room == nil || config == nil || session == nil {
		return nil, errors.WithStack(ErrArguments)
	}

	player := &Player{
		room:    room,
		config:  config,
		session: session,

		channel:  make(chan interface{}, KCPWindowSize),
		sendBuf:  make([]byte, 0, MaxPacketSize),
		recvBuf:  make([]byte, MaxPacketSize+1),
		state:    msg.NetPlayerState_Initing,
		frame:    0,
		deadline: TimeZero,
		cmdHeap:  NewCommandHeap(KCPWindowSize),
		cmdBufs:  make([][]byte, 0, sendBufSize),
		players:  make([]*Player, 0, room.MaxPlayers()),
	}

	return player, nil
}

func (p *Player) RoomId() string {
	return p.room.RoomId()
}

func (p *Player) PlayerId() string {
	return p.config.PlayerId
}

func (p *Player) Conv() uint32 {
	return p.config.Conv
}

func (p *Player) Close() {
	p.channel <- msg.NetFinish{
		Frame: p.frame,
		Cause: msg.NetFinishCause_ServerError,
	}
}

func (p *Player) Update() {
	p.logInfo(nil, "start")
	p.updateImpl()
	p.logInfo(nil, "finish")
}

func (p *Player) updateImpl() {
	p.deadline = p.room.CreatedAt().Add(ConnectTimeout)

	updateErr := (func() (err error) {
		for {
			size, message, err := p.session.Recv(p.recvBuf, p.channel, p.deadline)
			if err != nil {
				if errors.Is(err, kcp.ErrTimeout) {
					if p.state == msg.NetPlayerState_Running {
						return errors.WithStack(ErrTimeOutOfSync)
					} else {
						return errors.WithStack(ErrNetworkBroken)
					}
				} else {
					return errors.WithStack(err)
				}
			}

			if size != 0 {
				err = p.handleKCP(p.recvBuf[:size])
				if err != nil {
					return err
				}

			} else if message != nil {
				for message != nil {
					err = p.handleChan(message)
					if err != nil {
						return err
					}
					if len(p.channel) > 0 {
						message = <-p.channel
					} else {
						message = nil
					}
				}

			} else {
				return errors.WithStack(ErrUnexpected)
			}
		}
	})()

	if updateErr != nil {
		p.handleError(updateErr)
		dura := time.Until(p.deadline)
		if dura > 0 {
			time.Sleep(dura)
		}
	}

	if err := p.session.Close(); err != nil {
		p.logError(err)
	}
	p.room.Leave(p.Conv())
}

func (p *Player) handleKCP(buffer []byte) (err error) {
	message, offset, err := DecodeMessage(buffer)
	if err != nil {
		return err
	}

	p.logDebug("Recv", message)

	switch p.state {
	case msg.NetPlayerState_Initing:
		switch x := message.(type) {
		case *msg.NetConnect:
			if err = p.onConnect(x); err == nil {
				p.updateState(msg.NetPlayerState_Waiting)
				p.deadline = p.room.CreatedAt().Add(StartTimeout + SyncLowLimit)
			}
			return err
		case *msg.NetFinish:
			return errors.Wrapf(ErrRemoteFinish, "cause(%d)", x.Cause)
		default:
			return errors.WithStack(ErrPacketBroken)
		}

	case msg.NetPlayerState_Waiting:
		switch x := message.(type) {
		case *msg.NetFinish:
			return errors.Wrapf(ErrRemoteFinish, "cause(%d)", x.Cause)
		default:
			return errors.WithStack(ErrPacketBroken)
		}

	case msg.NetPlayerState_Running:
		switch x := message.(type) {
		case *msg.NetHash:
			return p.onHash(x)
		case *msg.NetCommand:
			if err = p.onKCPCommand(x, offset, buffer); err == nil {
				p.deadline, err = p.nextDealine()
			}
			return err
		case *msg.NetFinish:
			return errors.Wrapf(ErrRemoteFinish, "cause(%d)", x.Cause)
		default:
			return errors.WithStack(ErrPacketBroken)
		}

	case msg.NetPlayerState_Stopped:
		return nil

	default:
		return errors.WithStack(ErrUnexpected)
	}
}

func (p *Player) handleChan(message interface{}) (err error) {
	fmt.Printf("%T\n", message)

	switch p.state {
	case msg.NetPlayerState_Initing:
		switch x := message.(type) {
		case *msg.NetState:
			return nil
		case *msg.NetFinish:
			if err = p.sendToClient(x); err != nil {
				return err
			}
			return errors.Wrapf(ErrLocalFinish, "cause(%d)", x.Cause)
		default:
			return errors.WithStack(ErrMessageType)
		}

	case msg.NetPlayerState_Waiting:
		switch x := message.(type) {
		case *msg.NetState:
			return p.sendToClient(x)
		case *msg.NetStart:
			if err = p.sendToClient(x); err == nil {
				p.updateState(msg.NetPlayerState_Running)
				p.deadline = p.room.StartedAt().Add(SyncLowLimit)
			}
			return err
		case *msg.NetFinish:
			if err = p.sendToClient(x); err != nil {
				return err
			}
			return errors.Wrapf(ErrLocalFinish, "cause(%d)", x.Cause)
		default:
			return errors.WithStack(ErrMessageType)
		}

	case msg.NetPlayerState_Running:
		switch x := message.(type) {
		case *msg.NetState:
			return p.sendToClient(x)
		case *CommandBuffer:
			return p.onChanCommand(x)
		case *msg.NetFinish:
			if err = p.sendToClient(x); err != nil {
				return err
			}
			return errors.Wrapf(ErrLocalFinish, "cause(%d)", x.Cause)
		default:
			return errors.WithStack(ErrMessageType)
		}

	case msg.NetPlayerState_Stopped:
		return nil

	default:
		return errors.WithStack(ErrUnexpected)
	}
}

func (p *Player) updateState(state msg.NetPlayerState) msg.NetPlayerState {
	p.logInfo(LogFields{"new_state": state}, "state change")

	oldState := p.state
	atomic.StoreInt32((*int32)(unsafe.Pointer(&p.state)), int32(state))

	p.publishInRoom(false, &msg.NetState{
		Conv:  p.Conv(),
		State: p.state,
	})

	return oldState
}

func (p *Player) onConnect(connect *msg.NetConnect) (err error) {
	if p.RoomId() != connect.RoomId {
		return errors.WithStack(ErrAuthFailed)
	}
	if p.PlayerId() != connect.PlayerId {
		return errors.WithStack(ErrAuthFailed)
	}
	if p.config.Password != connect.Password {
		return errors.WithStack(ErrAuthFailed)
	}

	err = p.sendToClient(&msg.NetAccept{})
	if err != nil {
		return err
	}

	players := p.room.GetPlayers([]*Player{})
	for _, player := range players {
		if p != player {
			state := msg.NetPlayerState(atomic.LoadInt32((*int32)(unsafe.Pointer(&player.state))))
			if state != msg.NetPlayerState_Initing {
				err = player.sendToClient(&msg.NetState{Conv: player.Conv(), State: state})
				if err != nil {
					return err
				}
			}
		}
	}

	running, err := p.room.Connect(p.Conv())
	if err != nil {
		return err
	}
	if running {
		p.publishInRoom(true, &msg.NetStart{})
	}

	return nil
}

func (p *Player) nextDealine() (time.Time, error) {
	remote := p.room.StartedAt().Add(time.Second / FPS * time.Duration(p.frame))
	now := time.Now()
	low := now.Add(-SyncLowLimit)
	high := now.Add(SyncHighLimit)
	if remote.Before(low) || remote.After(high) {
		return TimeZero, errors.WithStack(ErrTimeOutOfSync)
	}
	deadline := now.Add(now.Sub(low))
	return deadline, nil
}

func (p *Player) onKCPCommand(cmd *msg.NetCommand, inOffset int, inBuffer []byte) error {
	if cmd.Frame != p.frame+1 {
		return errors.WithStack(ErrTimeOutOfSync)
	}
	p.frame = cmd.Frame

	outBuffer, err := TransfromCommand(cmd, inOffset, inBuffer, p.Conv())
	if err != nil {
		return err
	}
	p.publishInRoom(false, &CommandBuffer{
		Frame:      cmd.Frame,
		PlayerTeam: p.config.Team,
		Buffer:     outBuffer,
	})

	p.cmdBufs = p.cmdBufs[:0]
	for p.cmdHeap.Len() > 0 {
		for len(p.cmdBufs) < sendBufSize &&
			p.cmdHeap.Len() > 0 &&
			p.cmdHeap.Peek().Frame <= p.frame {
			buf := p.cmdHeap.Pop().Buffer
			p.cmdBufs = append(p.cmdBufs, buf)

			p.logDebug("Send", buf)
		}

		if len(p.cmdBufs) > 0 {
			_, err = p.session.SendBatch(p.cmdBufs, time.Now().Add(time.Millisecond*10))
			p.cmdBufs = p.cmdBufs[:0]
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Player) onChanCommand(buf *CommandBuffer) error {
	if buf.PlayerTeam == p.config.Team || buf.Frame <= p.frame {
		p.logDebug("Send", buf)

		sent, err := p.session.Send(buf.Buffer, time.Now().Add(time.Millisecond*5))
		if err != nil {
			return err
		}
		if sent != len(buf.Buffer) {
			return errors.WithStack(ErrUnexpected)
		}

	} else {
		p.cmdHeap.Push(buf)
	}

	return nil
}

func (p *Player) onHash(hash *msg.NetHash) error {
	return nil
}

func (p *Player) sendToClient(message proto.Message) (err error) {
	defer func() { p.sendBuf = p.sendBuf[:0] }()

	p.logDebug("Send", message)

	p.sendBuf, err = EncodeMessage(message, p.sendBuf[:0])
	if err != nil {
		return err
	}

	sent, err := p.session.Send(p.sendBuf, time.Now().Add(time.Millisecond*5))
	if err != nil {
		return err
	}
	if sent != len(p.sendBuf) {
		return errors.WithStack(ErrUnexpected)
	}

	return nil
}

func (p *Player) publishInRoom(self bool, message interface{}) {
	defer func() { p.players = p.players[:0] }()
	p.players = p.room.GetPlayers(p.players)
	for _, player := range p.players {
		if self || player != p {
			player.channel <- message
		}
	}
}

func (p *Player) handleError(err error) {
	if err == nil {
		return
	}

	p.logError(err)
	p.logInfo(LogFields{
		"new_state": msg.NetPlayerState_Stopped,
	}, "state change")

	oldState := p.state
	p.state = msg.NetPlayerState_Stopped

	if errors.Is(err, ErrRemoteFinish) || errors.Is(err, ErrLocalFinish) {
		p.deadline = time.Now()
		return
	}

	var cause msg.NetFinishCause
	if errors.Is(err, ErrNetworkBroken) {
		cause = msg.NetFinishCause_NetworkBroken
	} else if errors.Is(err, ErrPacketBroken) || errors.Is(err, ErrPacketSize) {
		cause = msg.NetFinishCause_InvalidPacket
	} else if errors.Is(err, ErrAuthFailed) {
		cause = msg.NetFinishCause_AuthFailed
	} else if errors.Is(err, ErrTimeOutOfSync) {
		cause = msg.NetFinishCause_TimeOutOfSync
	} else if errors.Is(err, ErrDataOutOfSync) {
		cause = msg.NetFinishCause_DataOutOfSync
	} else {
		cause = msg.NetFinishCause_ServerError
	}

	e := p.sendToClient(&msg.NetFinish{
		Frame: p.frame,
		Cause: cause,
	})
	if e == nil {
		p.deadline = time.Now().Add(time.Second * 5)
	} else {
		p.deadline = time.Now()
		p.logError(e)
	}

	if oldState == msg.NetPlayerState_Initing || oldState == msg.NetPlayerState_Waiting {
		p.publishInRoom(false, &msg.NetFinish{
			Frame: 0,
			Cause: msg.NetFinishCause_OtherPlayer,
		})
	} else {
		p.publishInRoom(false, &msg.NetState{
			Conv:  p.Conv(),
			State: p.state,
		})
	}
}

func (p *Player) logError(err error) {
	LogPrint(LevelError, LogFields{
		"source":    "Player",
		"room_id":   p.room.RoomId(),
		"player_id": p.PlayerId(),
		"conv":      p.Conv(),
		"state":     p.state,
		"frame":     p.frame,
	}, err)
}

func (p *Player) logInfo(fields LogFields, args ...interface{}) {
	if fields == nil {
		fields = LogFields{}
	}
	fields["source"] = "Player"
	fields["room_id"] = p.room.RoomId()
	fields["player_id"] = p.PlayerId()
	fields["conv"] = p.Conv()
	fields["state"] = p.state
	fields["frame"] = p.frame
	LogPrint(LevelInfo, fields, args...)
}

func (p *Player) logDebug(act string, msg interface{}) {
	if !InDebug {
		return
	}
	fields := LogFields{}
	fields["source"] = "Player"
	fields["room_id"] = p.room.RoomId()
	fields["player_id"] = p.PlayerId()
	fields["conv"] = p.Conv()
	fields["state"] = p.state
	fields["frame"] = p.frame
	LogPrint(LevelDebug, fields, fmt.Sprintf("#%s# %T{%+v}", act, msg, msg))
}
