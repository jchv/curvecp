package curvecp

import (
	"errors"
	"log"
)

var (
	ringbufFull = errors.New("ringbuf full")
	ringbufEmpty = errors.New("ringbuf empty")
)

type ringbuf struct {
	buf []byte
	start, size int
}

func newRingbuf(size int) *ringbuf {
	return &ringbuf{make([]byte, size), 0, 0}
}

func (r *ringbuf) Write(b []byte) int {
	written := 0
	for len(b) > 0 && r.size < len(r.buf) {
		start := (r.start+r.size)%len(r.buf)
		end := start+len(r.buf)-r.size
		if end > len(r.buf) {
			end = len(r.buf)
		}
		log.Printf("r.buf[%d:%d]", start, end)
		n := copy(r.buf[start:end], b)
		b = b[n:]
		r.size += n
		written += n
	}
	return written
}

func (r *ringbuf) Read(b []byte) int {
	read := 0
	for len(b) > 0 && r.size > 0 {
		end := r.start+r.size
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

func (r *ringbuf) Size() int {
	return r.size
}
