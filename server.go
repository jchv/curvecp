package curvecp

import (
	"crypto/rand"
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"github.com/johnwchadwick/curvecp/freelist"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/secretbox"
)

var (
	notImplemented = errors.New("not implemented")

	// Magic IDs at the beginning of packets.
	helloMagic = []byte("QvnQ5XlH")
	cookieMagic = []byte("RL3aNMXK")
	initiateMagic = []byte("QvnQ5XlI")
	messageMagic = []byte("RL3aNMXM")

	// The prefixes for various nonces
	helloNoncePrefix = []byte("CurveCP-client-H")
	cookieNoncePrefix = []byte("CurveCPK")
	initiateNoncePrefix = []byte("CurveCP-client-I")
	vouchNoncePrefix = []byte("CurveCPV")
	serverMessageNoncePrefix = []byte("CurveCP-server-M")
	clientMessageNoncePrefix = []byte("CurveCP-client-M")
	minuteNoncePrefix = []byte("minute-k")
)

type packet struct {
	net.Addr
	buf []byte
}

// Implements net.Listener.
type server struct {
	// From readLoop to pump, incoming packets. Minimal filtering
	// applied, but no cryptographic verification.
	packetIn chan packet
	// To pump, telling it to stop processing new
	// connections. Existing connections still get processed.
	stopListen chan struct{}
	// From pump to Accept() callers, to distribute new conns.
	newConn chan *conn
	//  From connStates to pump, telling it that a connection has been
	//  closed.
	endConn chan string

	// The underlying UDP socket.
	sock *net.UDPConn
	// The long-term secret key, used to authenticate Cookie packets.
	longTermSecretKey [32]byte
	// True if new connections should be accepted.
	listen bool
	// Minute keys to construct/verify cookies.
	minuteKey, prevMinuteKey [32]byte

	// Initiated clients. Pump forwards packets to them for
	// processing.
	conns map[string]chan packet
}

func newServer(sock *net.UDPConn, key []byte) *server {
	if len(key) != 32 {
		panic("Wrong key length")
	}
	s := &server{
	packetIn: make(chan packet),
	stopListen: make(chan struct{}),
	newConn: make(chan *conn),
	endConn: make(chan string),

	sock: sock,
	listen: true,

	conns: make(map[string]chan packet),
	}
	copy(s.longTermSecretKey[:], key)
	randBytes(s.minuteKey[:])
	randBytes(s.prevMinuteKey[:])
	go readLoop(s.sock, s.packetIn)
	go s.pump()
	return s
}

// Listen announces on the CurveCP address laddr and returns a CurveCP
// listener.
func Listen(laddr string, key []byte) (net.Listener, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	sock, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return newServer(sock, key), nil
}

// ListenUDPConn is similar to Listen, but takes an already existing
// net.UDPConn and listens for CurveCP on top of that. The CurveCP
// address is the UDPConn's LocalAddr().
//
// The main use of ListenUDPConn is to first execute a NAT-busting
// protocol on the UDPConn, and then use CurveCP to communicate with
// the peer.
func ListenUDPConn(sock *net.UDPConn, key []byte) (net.Listener, error) {
	sock.SetDeadline(time.Time{})
	return newServer(sock, key), nil
}

// Accept waits for and returns the next connection to the listener.
func (s *server) Accept() (net.Conn, error) {
	conn, ok := <-s.newConn
	if !ok {
		return nil, errors.New("Listener closed")
	}
	return conn, nil
}

func (s *server) Close() error {
	s.stopListen <- struct{}{}
	return nil
}

func (s *server) Addr() net.Addr {
	return s.sock.LocalAddr()
}

func readLoop(sock *net.UDPConn, packetIn chan<- packet) {
	pb := freelist.Packets.Get()
	for {
		// CurveCP datagrams are specified to always fit in the
		// smallest IPv6 datagram, 1280 bytes.
		n, addr, err := sock.ReadFrom(pb)
		if err != nil {
			// TODO: possibly be more discerning about when to return.
			return
		}
		if n < 64 {
			// Packet too small to be any CurveCP packet, discard.
			continue
		}

		pb = pb[:n]
		// messageMagic first, since it's the most common.
		packetIn <- packet{addr, pb}
		pb = freelist.Packets.Get()
	}
}

func (s *server) pump() {
	rotateMinuteKey := time.NewTicker(30*time.Second)

	for {
		select {
		case packet := <- s.packetIn:
			if s.checkHello(packet.buf) {
				resp := freelist.Packets.Get()
				resp, scratch := resp[:200], resp[200:]

				pkey, skey, err := box.GenerateKey(rand.Reader)
				if err != nil {
					panic("Ran out of randomness")
				}

				// Client short-term public key
				copy(scratch, packet.buf[40:40+32])
				// Server short-term secret key
				copy(scratch[32:], skey[:])

				// minute-key secretbox nonce
				var nonce [24]byte
				copy(nonce[:], minuteNoncePrefix)
				randBytes(nonce[len(minuteNoncePrefix):])

				secretbox.Seal(scratch[:64], scratch[:64], &nonce, &s.minuteKey)

				// Compressed cookie nonce
				copy(scratch[48:64], nonce[len(minuteNoncePrefix):])
				// Server short-term public key
				copy(scratch[16:48], pkey[:])

				var clientKey [32]byte
				copy(clientKey[:], packet.buf[40:40+32])

				// Cookie box nonce
				copy(nonce[:], cookieNoncePrefix)
				randBytes(nonce[len(cookieNoncePrefix):])

				box.Seal(resp[:56], scratch[16:16+128], &nonce, &clientKey, &s.longTermSecretKey)

				// Packet header, with extensions swapped.
				copy(resp, cookieMagic)
				copy(resp[8:], packet.buf[24:24+16])
				copy(resp[24:], packet.buf[8:8+16])
				copy(resp[40:], nonce[8:])

				s.sock.WriteTo(resp, packet.Addr)
				freelist.Packets.Put(resp)

			} else if serverShortTermKey, domain, valid := s.checkInitiate(packet.buf); valid {
				clientShortTermKey := packet.buf[40:40+32]
				clientLongTermKey := packet.buf[176:176+32]
				if ch, ok := s.conns[string(clientShortTermKey)]; ok {
					// Forward the Initiate to the conn. Because
					// checkInitiate replaces the box in the Initiate
					// packet with its plaintext, and because pump has
					// done all the crypto verification, conn can
					// ignore anything not relevant to maintaining
					// correct stream state.
					ch <- packet
				} else if s.listen {
					// This is a new client initiating. Construct a
					// conn and wait for someone to Accept() it.
					c := newConn(s.sock, clientLongTermKey, clientShortTermKey, serverShortTermKey, domain)
					// TODO: accept timeout or something.
					s.newConn <- c
					s.conns[string(clientShortTermKey)] = c.packetIn
				}
			}

		case <-s.stopListen:
			s.listen = false
			close(s.newConn)
			// We hang onto the long term secret key and minute keys
			// for one full minute key rotation, so that we can still
			// decode retransmitted Initiate packets for a while. The
			// rotateMinuteKey case below will take care of final
			// cleanup.

		case <-rotateMinuteKey.C:
			if !s.listen && bytes.Equal(s.minuteKey[:], s.prevMinuteKey[:]) {
				// At least 30 seconds have passed since we stopped
				// listening, we can clear the key material and stop
				// refreshing minute keys.
				for i := 0; i < len(s.longTermSecretKey); i++ {
					s.minuteKey[i] = 0
					s.prevMinuteKey[i] = 0
					s.longTermSecretKey[i] = 0
				}
				rotateMinuteKey.Stop()
			} else {
				copy(s.prevMinuteKey[:], s.minuteKey[:])
				if s.listen {
					randBytes(s.minuteKey[:])
				}
			}
		}
	}
}

func (s *server) checkHello(pb []byte) bool {
	if !s.listen || len(pb) != 224 || !bytes.Equal(pb[:8], helloMagic) {
		return false
	}

	var clientKey [32]byte
	copy(clientKey[:], pb[40:40+32])

	var nonce [24]byte
	copy(nonce[:], helloNoncePrefix)
	copy(nonce[len(helloNoncePrefix):], pb[136:136+8])

	var out [64]byte
	_, ok := box.Open(out[:0], pb[144:], &nonce, &clientKey, &s.longTermSecretKey)
	return ok
}

// If valid == true, pb[176:] is replaced by the plaintext contents of
// the Initiate C'->S' box.
func (s *server) checkInitiate(pb []byte) (serverShortTermKey []byte, domain string, valid bool) {
	valid = false
	if len(pb) < 544 || !bytes.Equal(pb[:8], initiateMagic) {
		return
	}

	// Try to open the cookie.
	var nonce [24]byte
	copy(nonce[:], minuteNoncePrefix)
	copy(nonce[len(minuteNoncePrefix):], pb[72:72+16])

	var cookie [64]byte
	if _, ok := secretbox.Open(cookie[:0], pb[88:168], &nonce, &s.minuteKey); !ok {
		if _, ok = secretbox.Open(cookie[:0], pb[88:168], &nonce, &s.prevMinuteKey); !ok {
			return
		}
	}

	// Check that the cookie and client match
	if !bytes.Equal(cookie[:32], pb[40:40+32]) {
		return
	}

	// Extract server short-term secret key
	var serverKey [32]byte
	serverShortTermKey = serverKey[:]
	copy(serverShortTermKey, cookie[32:])

	// Open the Initiate box using both short-term secret keys.
	copy(nonce[:], initiateNoncePrefix)
	copy(nonce[len(initiateNoncePrefix):], pb[168:168+8])

	var clientShortTermKey [32]byte
	copy(clientShortTermKey[:], pb[40:40+32])

	initiate := make([]byte, len(pb[176:])-box.Overhead)
	if _, ok := box.Open(initiate[:0], pb[176:], &nonce, &clientShortTermKey, &serverKey); !ok {
		return
	}

	if domain = domainToString(initiate[96:96+256]); domain == "" {
		return
	}

	// Extract client long-term public key and check the vouch
	// subpacket.
	var clientLongTermKey [32]byte
	copy(clientLongTermKey[:], initiate[:32])

	copy(nonce[:], vouchNoncePrefix)
	copy(nonce[len(vouchNoncePrefix):], initiate[32:32+16])

	var vouch [32]byte
	if _, ok := box.Open(vouch[:0], initiate[48:48+48], &nonce, &clientLongTermKey, &s.longTermSecretKey); !ok {
		return
	}

	if !bytes.Equal(vouch[:], pb[40:40+32]) {
		return
	}

	// The Initiate packet is valid, replace the encrypted box with
	// the plaintext and return.
	copy(pb[176:], initiate)
	for i := len(pb)-box.Overhead; i < len(pb); i++ {
		pb[i] = 0
	}
	valid = true

	return
}

// Returns empty string if the domain isn't valid.
func domainToString(d []byte) string {
	var ret []string
	for len(d) > 0 {
		l := int(d[0])
		if l == 0 {
			return strings.Join(ret, ".")
		}
		if l > 63 || l > len(d)-1 {
			return ""
		}

		ret = append(ret, string(d[1:l+1]))
		d = d[l+1:]
	}
	return strings.Join(ret, ".")
}

func randBytes(b []byte) {
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("Ran out of randomness")
	}		
}
