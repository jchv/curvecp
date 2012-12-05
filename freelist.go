package curvecp

type buf struct {
	size int
	ch   chan []byte
}

func (b *buf) Get() []byte {
	select {
	case buf := <-b.ch:
		return buf
	default:
	}
	return make([]byte, b.size)
}

func (b *buf) Put(buf []byte) {
	if cap(buf) == b.size {
		buf = buf[:cap(buf)]
		select {
		case b.ch <- buf:
		default:
		}
	}
}

var packetBuf = &buf{1280, make(chan []byte, 1024)}
