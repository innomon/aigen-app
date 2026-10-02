package udx

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrSocketClosed     = errors.New("udx: socket closed")
	ErrConnectionClosed = errors.New("udx: connection closed")
	ErrDialTimeout      = errors.New("udx: dial timeout")
)

// Socket manages a UDP listener and demultiplexes incoming packets to active virtual UDX streams.
type Socket struct {
	mu          sync.RWMutex
	conn        *net.UDPConn
	streams     map[uint32]*Stream
	acceptQueue chan *Stream
	closed      bool
	doneChan    chan struct{}
	nextID      uint32
}

// Listen creates a new UDX socket listening on the specified UDP address (e.g. "0.0.0.0:0").
func Listen(addr string) (*Socket, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	s := &Socket{
		conn:        conn,
		streams:     make(map[uint32]*Stream),
		acceptQueue: make(chan *Stream, 64),
		doneChan:    make(chan struct{}),
		nextID:      uint32(rand.Intn(10000) + 1),
	}

	go s.readLoop()
	return s, nil
}

// Addr returns the local UDP address the socket is bound to.
func (s *Socket) Addr() net.Addr {
	return s.conn.LocalAddr()
}

// Port returns the local UDP port.
func (s *Socket) Port() int {
	return s.conn.LocalAddr().(*net.UDPAddr).Port
}

func (s *Socket) readLoop() {
	buf := make([]byte, 2048)
	for {
		n, remoteAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-s.doneChan:
				return
			default:
				continue
			}
		}

		packet, err := DecodePacket(buf[:n])
		if err != nil {
			continue
		}

		s.handlePacket(packet, remoteAddr)
	}
}

func (s *Socket) handlePacket(p *Packet, addr *net.UDPAddr) {
	// Handle pure SYN (new incoming connection)
	if p.Flags&FlagSYN != 0 && p.Flags&FlagACK == 0 {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return
		}

		localID := atomic.AddUint32(&s.nextID, 1)
		stream := newStream(s, localID, p.StreamID, addr, false)
		s.streams[localID] = stream
		s.mu.Unlock()

		// Send SYN-ACK
		ackPacket := &Packet{
			Flags:    FlagSYN | FlagACK,
			StreamID: p.StreamID,
			Seq:      localID,
			Ack:      p.Seq,
		}
		_ = s.sendPacket(ackPacket, addr)

		select {
		case s.acceptQueue <- stream:
		default:
		}
		return
	}

	s.mu.RLock()
	stream, ok := s.streams[p.StreamID]
	s.mu.RUnlock()

	if ok && stream != nil {
		stream.onPacket(p)
	}
}

func (s *Socket) sendPacket(p *Packet, addr *net.UDPAddr) error {
	bytes, err := p.Encode()
	if err != nil {
		return err
	}
	_, err = s.conn.WriteToUDP(bytes, addr)
	return err
}

// Accept waits for and returns the next incoming UDX stream connection.
func (s *Socket) Accept() (*Stream, error) {
	select {
	case <-s.doneChan:
		return nil, ErrSocketClosed
	case stream, ok := <-s.acceptQueue:
		if !ok {
			return nil, ErrSocketClosed
		}
		return stream, nil
	}
}

// Dial establishes an active UDX virtual connection to the destination UDP address.
func (s *Socket) Dial(ctx context.Context, addr *net.UDPAddr) (*Stream, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrSocketClosed
	}

	localID := atomic.AddUint32(&s.nextID, 1)
	stream := newStream(s, localID, 0, addr, true)
	s.streams[localID] = stream
	s.mu.Unlock()

	// Perform SYN handshake with retry
	synPacket := &Packet{
		Flags:    FlagSYN,
		StreamID: localID,
		Seq:      1,
		Ack:      0,
	}

	ackChan := make(chan uint32, 1)
	stream.handshakeAck = ackChan

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Send initial SYN
	_ = s.sendPacket(synPacket, addr)

	for {
		select {
		case <-ctx.Done():
			stream.Close()
			return nil, ctx.Err()
		case remoteID := <-ackChan:
			stream.remoteID = remoteID
			return stream, nil
		case <-ticker.C:
			_ = s.sendPacket(synPacket, addr)
		}
	}
}

func (s *Socket) removeStream(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.streams, id)
}

// Close gracefully closes the socket and all active virtual streams.
func (s *Socket) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.doneChan)
	close(s.acceptQueue)

	for _, st := range s.streams {
		_ = st.Close()
	}
	s.streams = make(map[uint32]*Stream)
	s.mu.Unlock()

	return s.conn.Close()
}
