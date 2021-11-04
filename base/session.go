package base

import "time"

type ISession interface {
	GetConv() uint32
	Send(buffer []byte, deadline time.Time) (int, error)
	SendBatch(buffers [][]byte, deadline time.Time) (int, error)
	Recv(buffer []byte, extChan chan interface{}, deadline time.Time) (int, interface{}, error)
	Close() error
}
