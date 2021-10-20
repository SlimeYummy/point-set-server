package packet

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
	h.heap[n-1] = nil
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
