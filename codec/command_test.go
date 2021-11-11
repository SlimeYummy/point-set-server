package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandHeap(t *testing.T) {
	heap := NewCommandHeap(0)
	c0 := &CommandBuffer{Frame: 8}
	c1 := &CommandBuffer{Frame: 10}
	c2 := &CommandBuffer{Frame: 0}
	c3 := &CommandBuffer{Frame: 5}

	assert.Equal(t, (*CommandBuffer)(nil), heap.Peek())
	assert.Equal(t, (*CommandBuffer)(nil), heap.Pop())

	heap.Push(c0)
	assert.Equal(t, c0, heap.Peek())
	assert.Equal(t, c0, heap.Pop())

	heap.Push(c1)
	assert.Equal(t, c1, heap.Peek())
	heap.Push(c2)
	assert.Equal(t, c2, heap.Peek())
	heap.Push(c3)
	assert.Equal(t, c2, heap.Peek())
	assert.Equal(t, 3, heap.Len())

	assert.Equal(t, c2, heap.Pop())
	assert.Equal(t, c3, heap.Pop())
	assert.Equal(t, c1, heap.Pop())
	assert.Equal(t, 0, heap.Len())
}
