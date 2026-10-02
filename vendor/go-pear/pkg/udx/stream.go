package udx

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"
)

const (
	defaultMaxWindow = 64
	defaultRTO       = 60 * time.Millisecond
	maxRetries       = 15
)

type inflightPkt struct {
	packet  *Packet
	sentAt  time.Time
	retries int
}

// Stream represents a duplex reliable virtual connection over a UDX Socket with in-order reassembly and retransmissions.
type Stream struct {
	mu           sync.Mutex
	cond         *sync.Cond
	sendCond     *sync.Cond
	socket       *Socket
	localID      uint32
	remoteID     uint32
	remoteAddr   *net.UDPAddr
	isInitiator  bool
	handshakeAck chan uint32

	// Reliability - Sender
	sendSeq   uint32
	inflight  map[uint32]*inflightPkt
	maxWindow int
	rto       time.Duration

	// Reliability - Receiver
	nextExpectedSeq uint32
	oooBuf          map[uint32][]byte
	currentBuf      *bytes.Buffer

	closed   bool
	doneChan chan struct{}

	readDeadline  time.Time
	writeDeadline time.Time
}

func newStream(s *Socket, localID, remoteID uint32, remoteAddr *net.UDPAddr, isInitiator bool) *Stream {
	st := &Stream{
		socket:          s,
		localID:         localID,
		remoteID:        remoteID,
		remoteAddr:      remoteAddr,
		isInitiator:     isInitiator,
		sendSeq:         1,
		inflight:        make(map[uint32]*inflightPkt),
		maxWindow:       defaultMaxWindow,
		rto:             defaultRTO,
		nextExpectedSeq: 1,
		oooBuf:          make(map[uint32][]byte),
		currentBuf:      &bytes.Buffer{},
		doneChan:        make(chan struct{}),
	}
	st.cond = sync.NewCond(&st.mu)
	st.sendCond = sync.NewCond(&st.mu)

	go st.retransmitLoop()

	return st
}

func (s *Stream) retransmitLoop() {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.doneChan:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()
				return
			}

			now := time.Now()
			for _, item := range s.inflight {
				if now.Sub(item.sentAt) > s.rto {
					if item.retries > maxRetries {
						_ = s.closeLocked()
						s.mu.Unlock()
						return
					}
					item.sentAt = now
					item.retries++
					_ = s.socket.sendPacket(item.packet, s.remoteAddr)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *Stream) onPacket(p *Packet) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	// Handle SYN-ACK response for initiator handshake
	if p.Flags&FlagACK != 0 && s.handshakeAck != nil {
		select {
		case s.handshakeAck <- p.Seq:
			s.handshakeAck = nil
		default:
		}
	}

	// Handle ACK for sender in-flight window
	if p.Flags&FlagACK != 0 && p.Ack > 0 {
		for seq := range s.inflight {
			if seq <= p.Ack {
				delete(s.inflight, seq)
			}
		}
		s.sendCond.Broadcast()
	}

	// Handle DATA with in-order sequence reassembly
	if p.Flags&FlagDATA != 0 && len(p.Payload) > 0 {
		if p.Seq == s.nextExpectedSeq {
			// In-order packet arrived
			s.currentBuf.Write(p.Payload)
			s.nextExpectedSeq++

			// Drain any contiguous buffered out-of-order packets
			for {
				bufferedPayload, ok := s.oooBuf[s.nextExpectedSeq]
				if !ok {
					break
				}
				s.currentBuf.Write(bufferedPayload)
				delete(s.oooBuf, s.nextExpectedSeq)
				s.nextExpectedSeq++
			}

			s.cond.Broadcast()
		} else if p.Seq > s.nextExpectedSeq {
			// Out-of-order packet arrived: buffer in oooBuf
			if _, exists := s.oooBuf[p.Seq]; !exists {
				payloadCopy := append([]byte(nil), p.Payload...)
				s.oooBuf[p.Seq] = payloadCopy
			}
		}
		// If p.Seq < s.nextExpectedSeq, it is a duplicate packet. We still ACK below.

		// Send cumulative ACK back to sender
		ackPacket := &Packet{
			Flags:    FlagACK,
			StreamID: s.remoteID,
			Window:   uint32(defaultMaxWindow),
			Seq:      0,
			Ack:      s.nextExpectedSeq - 1,
		}
		if len(s.oooBuf) > 0 {
			ackPacket.Flags |= FlagSACK
		}
		_ = s.socket.sendPacket(ackPacket, s.remoteAddr)
	}

	// Handle FIN
	if p.Flags&FlagFIN != 0 {
		s.closeLocked()
		s.cond.Broadcast()
		s.sendCond.Broadcast()
	}
}

// Read reads received payload data from the stream in guaranteed arrival order.
func (s *Stream) Read(b []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if s.currentBuf.Len() > 0 {
			return s.currentBuf.Read(b)
		}

		if s.closed {
			return 0, io.EOF
		}

		if !s.readDeadline.IsZero() && time.Now().After(s.readDeadline) {
			return 0, io.EOF
		}

		s.cond.Wait()
	}
}

// Write transmits data chunked into UDX DATA packets with windowing and retransmissions.
func (s *Stream) Write(b []byte) (n int, err error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return 0, ErrConnectionClosed
	}

	totalWritten := 0
	offset := 0

	for offset < len(b) {
		for len(s.inflight) >= s.maxWindow && !s.closed {
			s.sendCond.Wait()
		}

		if s.closed {
			s.mu.Unlock()
			return totalWritten, ErrConnectionClosed
		}

		end := offset + MaxPayloadSize
		if end > len(b) {
			end = len(b)
		}

		chunk := append([]byte(nil), b[offset:end]...)
		seq := s.sendSeq
		s.sendSeq++

		p := &Packet{
			Flags:    FlagDATA,
			StreamID: s.remoteID,
			Window:   uint32(s.maxWindow),
			Seq:      seq,
			Ack:      s.nextExpectedSeq - 1,
			Payload:  chunk,
		}

		s.inflight[seq] = &inflightPkt{
			packet:  p,
			sentAt:  time.Now(),
			retries: 0,
		}

		_ = s.socket.sendPacket(p, s.remoteAddr)

		totalWritten += len(chunk)
		offset = end
	}

	s.mu.Unlock()
	return totalWritten, nil
}

// Close terminates the virtual connection and sends a FIN packet.
func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeLocked()
}

func (s *Stream) closeLocked() error {
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.doneChan)
	s.cond.Broadcast()
	s.sendCond.Broadcast()

	finPacket := &Packet{
		Flags:    FlagFIN,
		StreamID: s.remoteID,
		Seq:      s.sendSeq,
		Ack:      s.nextExpectedSeq - 1,
	}
	_ = s.socket.sendPacket(finPacket, s.remoteAddr)
	s.socket.removeStream(s.localID)
	return nil
}

// LocalAddr returns the local address.
func (s *Stream) LocalAddr() net.Addr {
	return s.socket.Addr()
}

// RemoteAddr returns the remote address.
func (s *Stream) RemoteAddr() net.Addr {
	return s.remoteAddr
}

// SetDeadline sets both read and write deadlines.
func (s *Stream) SetDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readDeadline = t
	s.writeDeadline = t
	return nil
}

// SetReadDeadline sets the read deadline.
func (s *Stream) SetReadDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readDeadline = t
	return nil
}

// SetWriteDeadline sets the write deadline.
func (s *Stream) SetWriteDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeDeadline = t
	return nil
}
