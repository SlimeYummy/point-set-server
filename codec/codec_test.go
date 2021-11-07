package codec

import (
	"encoding/binary"
	. "point-set/base"
	msg "point-set/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestDecodeMessage(t *testing.T) {
	m, _, _ := DecodeMessage([]byte{byte(msg.NetType_Connect), 0, 0})
	assert.IsType(t, &msg.NetConnect{}, m)

	m, _, _ = DecodeMessage([]byte{byte(msg.NetType_Accept), 0, 0})
	assert.IsType(t, &msg.NetAccept{}, m)

	m, _, _ = DecodeMessage([]byte{byte(msg.NetType_State), 0, 0})
	assert.IsType(t, &msg.NetState{}, m)

	m, _, _ = DecodeMessage([]byte{byte(msg.NetType_Start), 0, 0})
	assert.IsType(t, &msg.NetStart{}, m)

	m, _, _ = DecodeMessage([]byte{byte(msg.NetType_Hash), 0, 0})
	assert.IsType(t, &msg.NetHash{}, m)

	buffer := []byte{byte(msg.NetType_Command), 0, 5}
	buffer, _ = proto.MarshalOptions{}.MarshalAppend(buffer, &msg.NetCommand{
		Frame: 123,
		Conv:  999,
	})
	m, offset, err := DecodeMessage(buffer)
	assert.Equal(t, nil, err)
	assert.Equal(t, 8, offset)
	assert.Equal(t, uint32(123), m.(*msg.NetCommand).Frame)
	assert.Equal(t, uint32(999), m.(*msg.NetCommand).Conv)

	// error case

	m, offset, err = DecodeMessage([]byte{})
	assert.Equal(t, nil, m)
	assert.Equal(t, offset, 0)
	assert.ErrorIs(t, err, ErrPacketSize)

	m, offset, err = DecodeMessage(make([]byte, MaxPacketSize+1))
	assert.Equal(t, nil, m)
	assert.Equal(t, offset, 0)
	assert.ErrorIs(t, err, ErrPacketSize)

	buffer = []byte{byte(msg.NetType_Command), 0, 100}
	_, _, err = DecodeMessage(buffer)
	assert.ErrorIs(t, err, ErrPacketBroken)
}

func TestEncodeMessage(t *testing.T) {
	buffer, _ := EncodeMessage(&msg.NetConnect{}, []byte{})
	assert.Equal(t, msg.NetType_Connect, msg.NetType(buffer[0]))

	buffer, _ = EncodeMessage(&msg.NetAccept{}, []byte{})
	assert.Equal(t, msg.NetType_Accept, msg.NetType(buffer[0]))

	buffer, _ = EncodeMessage(&msg.NetState{}, []byte{})
	assert.Equal(t, msg.NetType_State, msg.NetType(buffer[0]))

	buffer, _ = EncodeMessage(&msg.NetStart{}, []byte{})
	assert.Equal(t, msg.NetType_Start, msg.NetType(buffer[0]))

	buffer, _ = EncodeMessage(&msg.NetFinish{}, []byte{})
	assert.Equal(t, msg.NetType_Finish, msg.NetType(buffer[0]))

	buffer, _ = EncodeMessage(&msg.NetCommand{}, []byte{})
	assert.Equal(t, msg.NetType_Command, msg.NetType(buffer[0]))

	buffer, _ = EncodeMessage(&msg.NetHash{}, []byte{})
	assert.Equal(t, msg.NetType_Hash, msg.NetType(buffer[0]))

	sh := &msg.NetHash{
		Frame: 123,
		Hash:  []byte("Mock-Hash"),
	}
	buffer, err := EncodeMessage(sh, []byte{})
	assert.Equal(t, nil, err)
	assert.Equal(t, proto.Size(sh)+3, len(buffer))
	assert.Equal(t, msg.NetType_Hash, msg.NetType(buffer[0]))
	assert.Equal(t, binary.BigEndian.Uint16(buffer[1:]), uint16(proto.Size(sh)))

	// error case

	buffer, err = EncodeMessage(nil, []byte{})
	assert.Equal(t, 0, len(buffer))
	assert.ErrorIs(t, err, ErrMessageType)

	buffer, err = EncodeMessage(&msg.NetConnect{
		Password: string(make([]byte, 10240)),
	}, []byte{})
	assert.Equal(t, 0, len(buffer))
	assert.ErrorIs(t, err, ErrMessageSize)
}

func TestTransfromCommand(t *testing.T) {
	inCmd := &msg.NetCommand{Frame: 101, Conv: 54321}
	inBuffer, _ := EncodeMessage(inCmd, []byte{})
	inOffset := len(inBuffer)
	inBuffer = append(inBuffer, []byte{9, 8, 7, 6, 5}...)
	outBuffer, err := TransfromCommand(inCmd, inOffset, inBuffer, 55555)
	assert.Equal(t, nil, err)
	outCmd, outOffset, err := DecodeMessage(outBuffer)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(101), outCmd.(*msg.NetCommand).Frame)
	assert.Equal(t, uint32(55555), outCmd.(*msg.NetCommand).Conv)
	assert.Equal(t, inBuffer[inOffset:], outBuffer[outOffset:])
}
