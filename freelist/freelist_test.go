package freelist

import (
	"testing"
)

func checkLen(t *testing.T, p []byte) {
	if len(p) != 10 {
		t.Errorf("len(p) = %d, want 10", len(p))
	}
	if cap(p) != 10 {
		t.Errorf("cap(p) = %d, want 10", cap(p))
	}
}

func checkChanLen(t *testing.T, l *List, size int) {
	if len(l.ch) != size {
		t.Errorf("len(l.ch) = %d, want %d", len(l.ch), size)
	}
}

func TestFreelist(t *testing.T) {
	l := New(10)
	checkChanLen(t, l, 0)

	// Are the buffers well formed?
	p := l.Get()
	checkLen(t, p)
	checkChanLen(t, l, 0)

	// Do they get recycled?
	p[0] = 42
	l.Put(p)
	checkChanLen(t, l, 1)

	// Do they get zeroed upon recycling?
	p = l.Get()
	checkLen(t, p)
	checkChanLen(t, l, 0)
	if p[0] != 0 {
		t.Errorf("p[0] = %#v, should have been zeroed", p[0])
	}

	// Do truncated slices still get recycled?
	l.Put(p[:5])
	checkChanLen(t, l, 1)

	// Do slices of the wrong capacity get discarded?
	l.Put(make([]byte, 20))
	l.Put(make([]byte, 10)[5:])
	checkChanLen(t, l, 1)
}
