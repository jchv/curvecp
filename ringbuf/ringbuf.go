// Package ringbuf implements a byte ring buffer. The interface is
// close to that of an io.ReadWriter, but note that the semantics
// differ in significant ways, because this ringbuf is an
// implementation detail of the curvecp package, and it was more
// convenient like this.
package ringbuf

type Ringbuf struct {
	buf         []byte
	start, size int
}

// New creates a new ring buffer of the given size.
func New(size int) *Ringbuf {
	return &Ringbuf{make([]byte, size), 0, 0}
}

// Write appends as many bytes of b as possible to the ring
// buffer. Returns the number of bytes added to the ring buffer, which
// may be 0 if the buffer is full.
func (r *Ringbuf) Write(b []byte) int {
	written := 0
	for len(b) > 0 && r.size < len(r.buf) {
		start := (r.start + r.size) % len(r.buf)
		end := start + len(r.buf) - r.size
		if end > len(r.buf) {
			end = len(r.buf)
		}
		n := copy(r.buf[start:end], b)
		b = b[n:]
		r.size += n
		written += n
	}
	return written
}

// Read reads as many bytes as possible from the ring buffer into
// b. Returns the number of bytes removed from the ring buffer, which
// may be zero if the buffer is empty.
func (r *Ringbuf) Read(b []byte) int {
	read := 0
	for len(b) > 0 && r.size > 0 {
		end := r.start + r.size
		if end > len(r.buf) {
			end = len(r.buf)
		}
		n := copy(b, r.buf[r.start:end])
		b = b[n:]
		r.start = (r.start + n) % len(r.buf)
		r.size -= n
		read += n
	}
	return read
}

func (r *Ringbuf) Size() int {
	return r.size
}
