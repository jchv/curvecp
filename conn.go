package curvecp

import (
	"container/list"
	"errors"
	"net"
	"time"

	"code.google.com/p/curvecp/ringbuf"
	"code.google.com/p/go.crypto/nacl/box"
)

const (
	numSendBlocks  = 128       // *1024 = 128k of send buffer.
	recvBufferSize = 64 * 1024 // 64k
)

var (
	// TODO: make it an appropriate net.Error
	deadlineExceeded = errors.New("deadline exceeded")
)

type opResult struct {
	n   int
	err error
}

type block struct {
	// The data to be sent.
	buf []byte
	// Position of the first byte of buf in the overall stream.
	pos int64

	// The backing array for buf. Static so that we can preallocate
	// all the memory associated with a connection at the beginning.
	arr [1024]byte
}

// Implements net.Conn. Used by both client and server, with different
// message/packet pumps.
type conn struct {
	// Peer's long-term public key, aka its identity.
	peerIdentity [32]byte
	// The shared key used to seal/open boxes to/from this client.
	sharedKey [32]byte
	// The domain requested during initiation.
	domain string

	// from pump to conn, packets to process. Only Initiate and
	// Message packets come through here.
	packetIn chan packet
	// The socket for sending. Don't read this, use packetIn for
	// reading.
	sock *net.UDPConn

	// From user to pump, request to read/write some data.
	readRequest  chan []byte
	writeRequest chan []byte
	// Deadlines for those ops
	readDeadline time.Time
	writeDeadline time.Time
	// From pump to user, result of a read or write.
	ioResult chan opResult

	// Blocks that needs to be sent.
	toSend *list.List // of *block
	// Freelist of blocks. All allocated on creation of the conn, we
	// never allocate more.
	sendFree *list.List // of *block

	// Received data waiting for a reader.
	received *ringbuf.Ringbuf
}

func newConn(sock *net.UDPConn, peerIdentity, publicKey, privateKey []byte, domain string) *conn {
	if len(peerIdentity) != 32 || len(publicKey) != 32 || len(privateKey) != 32 {
		panic("wrong key size")
	}
	c := &conn{
		domain: domain,

		packetIn: make(chan packet),
		sock:     sock,

		readRequest:  make(chan []byte),
		writeRequest: make(chan []byte),
		ioResult:     make(chan opResult),

		toSend:   list.New(),
		sendFree: list.New(),

		received: ringbuf.New(recvBufferSize),
	}
	// Key setup.
	copy(c.peerIdentity[:], peerIdentity)
	var pub, priv [32]byte
	copy(pub[:], publicKey)
	copy(priv[:], privateKey)
	box.Precompute(&c.sharedKey, &pub, &priv)

	// Send blocks
	for i := 0; i < numSendBlocks; i++ {
		c.sendFree.PushBack(new(block))
	}

	go c.pump()
	return c
}

func (c *conn) Read(b []byte) (int, error) {
	var deadline <-chan time.Time
	if !c.readDeadline.IsZero() {
		deadline = time.After(c.readDeadline.Sub(time.Now()))
	}
	select {
	case c.readRequest <- b:
	case <-deadline:
		return 0, deadlineExceeded
	}
	// Once readRequest has succeeded, this will return promptly, so
	// don't reapply the deadline (plus, it would corrupt the stream
	// to do so - pump is performing an operation on our behalf,
	// ignoring that would cause a gap in the data).
	res := <-c.ioResult
	return res.n, res.err
}

func (c *conn) Write(b []byte) (int, error) {
	var deadline <-chan time.Time
	if !c.writeDeadline.IsZero() {
		deadline = time.After(c.writeDeadline.Sub(time.Now()))
	}
	written := 0
	for len(b) > 0 {
		select {
		case c.writeRequest <- b:
		case <-deadline:
			return written, deadlineExceeded
		}
		// See above, no deadline here.
		res := <-c.ioResult
		written += res.n
		b = b[res.n:]
		if res.err != nil {
			return written, res.err
		}
	}
	return written, nil
}

func (c *conn) Close() error {
	return notImplemented
}

func (c *conn) LocalAddr() net.Addr {
	return c.sock.LocalAddr()
}

func (c *conn) RemoteAddr() net.Addr {
	return c.sock.RemoteAddr()
}

func (c *conn) SetDeadline(t time.Time) error {
	// Not thread-safe. TODO: figure out if it's supposed to be.
	c.readDeadline = t
	c.writeDeadline = t
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return nil
}

func (c *conn) pump() {
	for {
		select {}
	}
}
