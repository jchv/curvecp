// Package freelist implements a freelist of byte slices. All slices
// provided by a given freelist have the same size, and can only be
// returned to the freelist if their capacity is unchanged.
package freelist

// Freelist of 1280 byte buffers, big enough for CurveCP packets.
var Packets = New(1280)

type List struct {
	size int
	ch   chan []byte
}

// New returns a new freelist of buffers sized as requested.
func New(size int) *List {
	return &List{size, make(chan []byte, 1024)}
}

// Get returns a buffer, reusing a previously allocated one if
// possible. If not possible, a new one is allocated. In any case, all
// slices are zeroed.
func (b *List) Get() []byte {
	select {
	case buf := <-b.ch:
		return buf
	default:
	}
	return make([]byte, b.size)
}

// Put releases a buffer back to the freelist. The slice is only
// reused if its capacity is appropriate for this freelist.
func (b *List) Put(buf []byte) {
	if cap(buf) == b.size {
		buf = buf[:cap(buf)]
		// In theory, we could reuse the buffers without zeroing, but
		// it's nicer for the API to preserve the expectation that all
		// slices look like freshly allocated slices (not to mention
		// bugs that would accidentally reuse recycled data).
		for i := 0; i < len(buf); i++ {
			buf[i] = 0
		}
		select {
		case b.ch <- buf:
		default:
		}
	}
}
