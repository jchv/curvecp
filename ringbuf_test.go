package curvecp

import (
	"bytes"
	"testing"
)

const (
	doRead = iota
	doWrite
)

type rbtest struct {
	op        int
	data      string
	bufSize   int
	ret       int
	totalSize int
}

func TestRingbuf(t *testing.T) {
	r := newRingbuf(5)

	tab := []rbtest{
		{doWrite, "abc", 0, 3, 3},
		{doWrite, "d", 0, 1, 4},
		{doRead, "ab", 2, 2, 2},
		{doWrite, "ef", 0, 2, 4},
		{doRead, "cdef", 5, 4, 0},
		{doRead, "", 5, 0, 0},
		{doWrite, "abcdefg", 0, 5, 5},
		{doWrite, "fg", 0, 0, 5},
		{doRead, "abcde", 10, 5, 0},
		{doWrite, "a", 0, 1, 1},
		{doWrite, "b", 0, 1, 2},
		{doWrite, "c", 0, 1, 3},
		{doWrite, "d", 0, 1, 4},
		{doWrite, "e", 0, 1, 5},
		{doWrite, "f", 0, 0, 5},
		{doRead, "a", 1, 1, 4},
		{doRead, "b", 1, 1, 3},
		{doRead, "c", 1, 1, 2},
		{doRead, "d", 1, 1, 1},
		{doRead, "e", 1, 1, 0},
		{doRead, "", 1, 0, 0},
	}

	for _, step := range tab {
		switch step.op {
		case doRead:
			b := make([]byte, step.bufSize)
			n := r.Read(b)
			t.Logf("r.Read(%d) = %d, %#v", step.bufSize, step.ret, string(b[:n]))
			if n != step.ret {
				t.Errorf("r.Read(%#v) = %d, want %d", b, n, step.ret)
			}
			if !bytes.Equal(b[:n], []byte(step.data)) {
				t.Errorf("b = %#v, want %#v", string(b[:n]), step.data)
			}

		case doWrite:
			n := r.Write([]byte(step.data))
			t.Logf("r.Write(%#v) = %d", string(step.data), step.ret)
			if n != step.ret {
				t.Errorf("r.Write(%#v) = %d, want %d", step.data, n, step.ret)
			}
		}
		if r.Size() != step.totalSize {
			t.Errorf("r.Size() = %d, want %d", r.Size(), step.totalSize)
		}
		t.Logf("%#v", r)
	}
}
