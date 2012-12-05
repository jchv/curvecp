package curvecp

import (
	"net"
	"time"

	"code.google.com/p/go.crypto/nacl/box"
)

// Implements net.Conn. Used by both client and server, with different
// message/packet pumps.
type conn struct {
	// Peer's long-term public key, aka its identity.
	peerIdentity [32]byte
	// The domain requested during initiation.
	domain string
	// The shared key used to seal/open boxes to/from this client.
	sharedKey [32]byte

	// from pump to conn, packets to process. Only Initiate and
	// Message packets come through here.
	packetIn chan packet
	// The socket for sending. Don't read this, use packetIn for
	// reading.
	sock *net.UDPConn

	
}

func newConn(sock *net.UDPConn, peerIdentity, publicKey, privateKey []byte, domain string) *conn {
	if len(peerIdentity) != 32 || len(publicKey) != 32 || len(privateKey) != 32 {
		panic("wrong key size")
	}
	c := &conn{
	domain: domain,
	packetIn: make(chan packet),
	sock: sock,
	}
	copy(c.peerIdentity[:], peerIdentity)
	var pub, priv [32]byte
	copy(pub[:], publicKey)
	copy(priv[:], privateKey)
	box.Precompute(&c.sharedKey, &pub, &priv)
	go c.pump()
	return c
}

func (c *conn) Read(b []byte) (int, error) {
	return 0, notImplemented
}

func (c *conn) Write(b []byte) (int, error) {
	return 0, notImplemented
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
	return notImplemented
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return notImplemented
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return notImplemented
}

func (c *conn) pump() {
	for {
		select {
		}
	}
}
