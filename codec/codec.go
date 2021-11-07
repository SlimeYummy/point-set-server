package codec

import (
	"encoding/binary"
	. "point-set/base"
	msg "point-set/message"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

func DecodeMessage(buffer []byte) (message proto.Message, offser int, err error) {
	if len(buffer) < MinPacketSize {
		return nil, 0, errors.WithStack(ErrPacketSize)
	}
	if len(buffer) > MaxPacketSize {
		return nil, 0, errors.WithStack(ErrPacketSize)
	}

	net_type := msg.NetType(buffer[0])
	switch net_type {
	case msg.NetType_Connect:
		message = &msg.NetConnect{}
	case msg.NetType_Accept:
		message = &msg.NetAccept{}
	case msg.NetType_State:
		message = &msg.NetState{}
	case msg.NetType_Start:
		message = &msg.NetStart{}
	case msg.NetType_Finish:
		message = &msg.NetFinish{}
	case msg.NetType_Command:
		message = &msg.NetCommand{}
	case msg.NetType_Hash:
		message = &msg.NetHash{}
	default:
		return nil, 0, errors.WithStack(ErrPacketBroken)
	}

	msgSize := binary.BigEndian.Uint16(buffer[1:])
	offset := int(MinPacketSize + msgSize)
	if offset > len(buffer) {
		return nil, 0, errors.WithStack(ErrPacketBroken)
	}

	err = proto.Unmarshal(buffer[MinPacketSize:offset], message)
	if err != nil {
		return nil, 0, errors.WithStack(errors.Wrap(ErrPacketBroken, err.Error()))
	}
	return message, offset, nil
}

func EncodeMessage(message proto.Message, buffer []byte) ([]byte, error) {
	switch message.(type) {
	case *msg.NetConnect:
		buffer = append(buffer, byte(msg.NetType_Connect))
	case *msg.NetAccept:
		buffer = append(buffer, byte(msg.NetType_Accept))
	case *msg.NetState:
		buffer = append(buffer, byte(msg.NetType_State))
	case *msg.NetStart:
		buffer = append(buffer, byte(msg.NetType_Start))
	case *msg.NetFinish:
		buffer = append(buffer, byte(msg.NetType_Finish))
	case *msg.NetCommand:
		buffer = append(buffer, byte(msg.NetType_Command))
	case *msg.NetHash:
		buffer = append(buffer, byte(msg.NetType_Hash))
	default:
		return buffer, errors.WithStack(ErrMessageType)
	}

	buffer = append(buffer, 0, 0)

	buffer, err := proto.MarshalOptions{}.MarshalAppend(buffer, message)
	if err != nil {
		return buffer, errors.WithStack(err)
	}

	if len(buffer) > MaxPacketSize {
		buffer = (buffer)[:0]
		return buffer, errors.WithStack(ErrMessageSize)
	}
	msgSize := len(buffer) - MinPacketSize
	binary.BigEndian.PutUint16((buffer)[1:], uint16(msgSize))

	return buffer, nil
}

func TransfromCommand(
	command *msg.NetCommand,
	inOffset int,
	inBuffer []byte,
	conv uint32,
) (outBuffer []byte, err error) {
	command.Conv = conv
	outBuffer = make([]byte, 0, proto.Size(command)+len(inBuffer)-inOffset)
	outBuffer, err = EncodeMessage(command, outBuffer)
	if err != nil {
		return nil, err
	}
	outBuffer = append(outBuffer, inBuffer[inOffset:]...)
	return outBuffer, nil
}
