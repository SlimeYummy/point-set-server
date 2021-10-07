package main

import (
	"encoding/binary"
	"point-set/msg"

	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go/v5"
	"google.golang.org/protobuf/proto"
)

type Packet struct {
	Type  msg.Type
	Frame uint32
	Data  []byte
}

func NewPacket(data []byte) (*Packet, error) {
	if len(data) < MinPacketSize {
		return nil, errors.WithStack(ErrShortPacket)
	}
	if len(data) > MaxPacketSize {
		return nil, errors.WithStack(ErrLogPacket)
	}
	return &Packet{
		Type:  msg.Type(data[len(data)-1]),
		Frame: binary.LittleEndian.Uint32(data[len(data)-5 : len(data)-1]),
		Data:  data,
	}, nil
}

func ReadPacket(session *kcp.UDPSession, buf []byte) (*Packet, error) {
	size, err := session.Read(buf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if size < MinPacketSize {
		return nil, errors.WithStack(ErrShortPacket)
	}
	if size > MaxPacketSize {
		return nil, errors.WithStack(ErrLogPacket)
	}

	data := make([]byte, size)
	copy(data, buf[0:size])
	return &Packet{
		Type:  msg.Type(data[len(data)-1]),
		Frame: binary.LittleEndian.Uint32(data[len(data)-5 : len(data)-1]),
		Data:  data,
	}, nil
}

type PacketHeap struct {
	heap []*Packet
}

func NewPacketHeap(cap int) *PacketHeap {
	return &PacketHeap{
		heap: make([]*Packet, 0, cap),
	}
}

func (h *PacketHeap) Len() int {
	return len(h.heap)
}

func (h *PacketHeap) Push(p *Packet) {
	h.heap = append(h.heap, p)
	h.up(h.Len() - 1)
}

func (h *PacketHeap) Peek() *Packet {
	if h.Len() <= 0 {
		return nil
	}
	return h.heap[0]
}

func (h *PacketHeap) Pop() *Packet {
	if h.Len() <= 0 {
		return nil
	}

	n := h.Len() - 1
	h.heap[0], h.heap[n] = h.heap[n], h.heap[0]
	h.down(0, n)

	p := h.heap[n-1]
	h.heap = h.heap[0 : n-1]
	return p
}

func (h *PacketHeap) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || h.heap[j].Frame >= h.heap[i].Frame {
			break
		}
		h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
		j = i
	}
}

func (h *PacketHeap) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && h.heap[j2].Frame < h.heap[j1].Frame {
			j = j2 // = 2*i + 2  // right child
		}
		if h.heap[j].Frame >= h.heap[i].Frame {
			break
		}
		h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
		i = j
	}
	return i > i0
}

func DecodeMsgInit(packet *Packet) (*msg.MsgInit, error) {
	if packet.Type != msg.Type_Init {
		return nil, ErrBadType
	}
	var m msg.MsgInit
	err := proto.Unmarshal(packet.Data, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func EncodeMsgStart(start *msg.MsgStart) (*Packet, error) {
	data, err := proto.Marshal(start)
	if err != nil {
		return nil, err
	}

	data = append(data, byte(msg.Type_Start))
	binary.LittleEndian.PutUint32(data, 0)
	if len(data) > MaxPacketSize {
		return nil, errors.WithStack(ErrLogPacket)
	}

	return &Packet{
		Type:  msg.Type_Start,
		Frame: 0,
		Data:  data,
	}, nil
}

func DecodeMsgFinish(packet *Packet) (*msg.MsgFinish, error) {
	if packet.Type != msg.Type_Finish {
		return nil, ErrBadType
	}
	var m msg.MsgFinish
	err := proto.Unmarshal(packet.Data, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

const (
	CauseStartRoomFailed   = "start room failed"
	CauseStartRoomTimeout  = "start room timeout"
	CausePacketBeforeStart = "Packet before start"
	CausePacketOutOfOrder  = "Packet out of order"
	CauseClientUnsync      = "Client unsync"
)

func EncodeMsgError(frame uint32, error *msg.MsgError) (*Packet, error) {
	data, err := proto.Marshal(error)
	if err != nil {
		return nil, err
	}

	data = append(data, byte(msg.Type_Error))
	binary.LittleEndian.PutUint32(data, frame)
	if len(data) > MaxPacketSize {
		return nil, errors.WithStack(ErrLogPacket)
	}

	return &Packet{
		Type:  msg.Type_Error,
		Frame: frame,
		Data:  data,
	}, nil
}
