package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRoomManager(t *testing.T) {
	_, err := NewRoomManager("127.0.0.1:12345")
	assert.Equal(t, nil, err)
}
