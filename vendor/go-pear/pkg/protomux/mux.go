package protomux

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"go-pear/pkg/compactenc"
)

const (
	// MaxFrameSize is the maximum permitted size for an individual multiplexed frame (4 MiB).
	MaxFrameSize = 4 * 1024 * 1024

	// ControlChannelID is the reserved channel ID for protocol negotiation (Channel 0).
	ControlChannelID uint64 = 0

	// TypeOpen is the control message type for opening a channel.
	TypeOpen uint64 = 1
	// TypeClose is the control message type for closing a channel.
	TypeClose uint64 = 3
)

var (
	// ErrMultiplexerClosed is returned when operating on a closed multiplexer.
	ErrMultiplexerClosed = errors.New("protomux: multiplexer closed")
	// ErrChannelNotFound is returned when receiving data for an unknown channel.
	ErrChannelNotFound = errors.New("protomux: channel not found")
	// ErrFrameTooLarge is returned when an incoming wire frame exceeds MaxFrameSize.
	ErrFrameTooLarge = errors.New("protomux: frame exceeds maximum permitted size")
)

// Multiplexer coordinates bidirectional multi-channel protocol streams over a single connection.
type Multiplexer struct {
	conn          net.Conn
	channels      map[uint64]*Channel
	remoteToLocal map[uint64]uint64
	pendingRemote map[string]uint64
	nextID        uint64

	writeMu sync.Mutex
	chanMu  sync.RWMutex
	closeMu sync.Mutex
	closed  bool

	doneChan chan struct{}
}

// NewMultiplexer creates and starts a protocol stream multiplexer over the given connection.
func NewMultiplexer(conn net.Conn) *Multiplexer {
	m := &Multiplexer{
		conn:          conn,
		channels:      make(map[uint64]*Channel),
		remoteToLocal: make(map[uint64]uint64),
		pendingRemote: make(map[string]uint64),
		nextID:        1,
		doneChan:      make(chan struct{}),
	}
	go m.readLoop()
	return m
}

// Conn returns the underlying network connection.
func (m *Multiplexer) Conn() net.Conn {
	return m.conn
}

// OpenChannel registers a new channel for the specified protocol name and negotiates it over the wire.
func (m *Multiplexer) OpenChannel(protocol string, handler MessageHandler) (*Channel, error) {
	m.chanMu.Lock()
	defer m.chanMu.Unlock()

	if m.closed {
		return nil, ErrMultiplexerClosed
	}

	id := m.nextID
	m.nextID++

	ch := &Channel{
		id:        id,
		protocol:  protocol,
		mux:       m,
		onMessage: handler,
	}

	if err := m.registerChannelLocked(ch); err != nil {
		return nil, err
	}

	return ch, nil
}

// OpenChannelWithID registers a channel with an explicit channel ID (useful for peer pairing).
func (m *Multiplexer) OpenChannelWithID(id uint64, protocol string, handler MessageHandler) (*Channel, error) {
	m.chanMu.Lock()
	defer m.chanMu.Unlock()

	if m.closed {
		return nil, ErrMultiplexerClosed
	}

	if _, exists := m.channels[id]; exists {
		return nil, fmt.Errorf("channel id %d already registered", id)
	}

	if id >= m.nextID {
		m.nextID = id + 1
	}

	ch := &Channel{
		id:        id,
		protocol:  protocol,
		mux:       m,
		onMessage: handler,
	}

	if err := m.registerChannelLocked(ch); err != nil {
		return nil, err
	}

	return ch, nil
}

func (m *Multiplexer) registerChannelLocked(ch *Channel) error {
	m.channels[ch.id] = ch
	if remID, ok := m.pendingRemote[ch.protocol]; ok {
		ch.remoteID = remID
		m.remoteToLocal[remID] = ch.id
		delete(m.pendingRemote, ch.protocol)
	}

	// Send wire-level channel-open control frame on Channel 0
	// Open frame payload: localID (uint) + protocol (string) + id (optional buffer)
	sz := compactenc.PreencodeUint(ch.id) +
		compactenc.PreencodeString(ch.protocol) +
		compactenc.PreencodeBuffer(nil)

	st := compactenc.NewAllocatedState(sz)
	_ = compactenc.EncodeUint(st, ch.id)
	_ = compactenc.EncodeString(st, ch.protocol)
	_ = compactenc.EncodeBuffer(st, nil)

	payload := st.Bytes()
	go func() {
		_ = m.sendFrame(ControlChannelID, TypeOpen, payload)
	}()

	return nil
}

// unregisterChannel removes a channel from the active registry and sends a close frame.
func (m *Multiplexer) unregisterChannel(id uint64) {
	m.chanMu.Lock()
	ch, exists := m.channels[id]
	if exists {
		delete(m.channels, id)
		if ch.remoteID != 0 {
			delete(m.remoteToLocal, ch.remoteID)
		}
	}
	m.chanMu.Unlock()

	// Send wire-level channel-close control frame on Channel 0
	sz := compactenc.PreencodeUint(id)
	st := compactenc.NewAllocatedState(sz)
	_ = compactenc.EncodeUint(st, id)
	payload := st.Bytes()
	go func() {
		_ = m.sendFrame(ControlChannelID, TypeClose, payload)
	}()
}

// sendFrame formats and writes a multiplexed message frame without redundant double length-prefixing.
// Wire format: compactenc(frameLen) + compactenc(channelID) + compactenc(typeID) + raw(payload)
func (m *Multiplexer) sendFrame(channelID, typeID uint64, payload []byte) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	return m.sendFrameLocked(channelID, typeID, payload)
}

func (m *Multiplexer) sendFrameLocked(channelID, typeID uint64, payload []byte) error {
	if m.closed {
		return ErrMultiplexerClosed
	}

	bodySize := compactenc.PreencodeUint(channelID) +
		compactenc.PreencodeUint(typeID) +
		len(payload)

	frameLen := uint64(bodySize)
	totalSize := compactenc.PreencodeUint(frameLen) + bodySize

	state := compactenc.NewAllocatedState(totalSize)
	_ = compactenc.EncodeUint(state, frameLen)
	_ = compactenc.EncodeUint(state, channelID)
	_ = compactenc.EncodeUint(state, typeID)
	copy(state.Buffer[state.Start:], payload)
	state.Start += len(payload)

	_, err := m.conn.Write(state.Bytes())
	return err
}

func (m *Multiplexer) handleControlFrame(typeID uint64, payload []byte) {
	st := compactenc.NewState(payload)
	switch typeID {
	case TypeOpen:
		remoteID, err := compactenc.DecodeUint(st)
		if err != nil {
			return
		}
		protocol, err := compactenc.DecodeString(st)
		if err != nil {
			return
		}
		_, _ = compactenc.DecodeBuffer(st) // Optional buffer / handshake

		m.chanMu.Lock()
		defer m.chanMu.Unlock()

		var matched *Channel
		for _, ch := range m.channels {
			if ch.protocol == protocol && (ch.remoteID == 0 || ch.remoteID == remoteID) {
				matched = ch
				break
			}
		}

		if matched != nil {
			matched.remoteID = remoteID
			m.remoteToLocal[remoteID] = matched.id
		} else {
			m.pendingRemote[protocol] = remoteID
		}

	case TypeClose:
		remoteID, err := compactenc.DecodeUint(st)
		if err != nil {
			return
		}

		m.chanMu.Lock()
		localID, ok := m.remoteToLocal[remoteID]
		var ch *Channel
		if ok {
			ch = m.channels[localID]
			delete(m.remoteToLocal, remoteID)
		} else {
			ch = m.channels[remoteID]
		}
		m.chanMu.Unlock()

		if ch != nil {
			_ = ch.Close()
		}
	}
}

// readLoop continuously parses frames from the underlying stream and dispatches them.
func (m *Multiplexer) readLoop() {
	defer m.Close()

	for {
		// Read frame length compact int
		state := compactenc.NewAllocatedState(1)
		if _, err := io.ReadFull(m.conn, state.Buffer[:1]); err != nil {
			return
		}

		tag := state.Buffer[0]
		var frameLen uint64

		if tag <= 0xfc {
			frameLen = uint64(tag)
		} else if tag == 0xfd {
			extra := make([]byte, 2)
			if _, err := io.ReadFull(m.conn, extra); err != nil {
				return
			}
			fullTag := append([]byte{tag}, extra...)
			st := compactenc.NewState(fullTag)
			frameLen, _ = compactenc.DecodeUint(st)
		} else if tag == 0xfe {
			extra := make([]byte, 4)
			if _, err := io.ReadFull(m.conn, extra); err != nil {
				return
			}
			fullTag := append([]byte{tag}, extra...)
			st := compactenc.NewState(fullTag)
			frameLen, _ = compactenc.DecodeUint(st)
		} else {
			extra := make([]byte, 8)
			if _, err := io.ReadFull(m.conn, extra); err != nil {
				return
			}
			fullTag := append([]byte{tag}, extra...)
			st := compactenc.NewState(fullTag)
			frameLen, _ = compactenc.DecodeUint(st)
		}

		if frameLen == 0 {
			continue
		}
		if frameLen > MaxFrameSize {
			_ = m.Close()
			return
		}

		frameBuf := make([]byte, frameLen)
		if _, err := io.ReadFull(m.conn, frameBuf); err != nil {
			return
		}

		decodeState := compactenc.NewState(frameBuf)
		channelID, err := compactenc.DecodeUint(decodeState)
		if err != nil {
			continue
		}
		typeID, err := compactenc.DecodeUint(decodeState)
		if err != nil {
			continue
		}

		// Remaining frame buffer is the raw un-encoded message payload
		payload := frameBuf[decodeState.Start:]

		if channelID == ControlChannelID {
			m.handleControlFrame(typeID, payload)
			continue
		}

		m.chanMu.RLock()
		ch, exists := m.channels[channelID]
		if !exists {
			if localID, ok := m.remoteToLocal[channelID]; ok {
				ch, exists = m.channels[localID]
			}
		}
		m.chanMu.RUnlock()

		if exists && ch != nil {
			ch.mu.RLock()
			handler := ch.onMessage
			ch.mu.RUnlock()
			if handler != nil {
				go func(h MessageHandler, t uint64, p []byte) {
					_ = h(t, p)
				}(handler, typeID, payload)
			}
		}
	}
}

// Close gracefully terminates the multiplexer and closes all active channels and connection.
func (m *Multiplexer) Close() error {
	m.closeMu.Lock()
	if m.closed {
		m.closeMu.Unlock()
		return nil
	}
	m.closed = true
	m.closeMu.Unlock()

	close(m.doneChan)

	m.chanMu.Lock()
	for _, ch := range m.channels {
		ch.mu.Lock()
		ch.closed = true
		handler := ch.onClose
		ch.mu.Unlock()
		if handler != nil {
			_ = handler()
		}
	}
	m.channels = make(map[uint64]*Channel)
	m.chanMu.Unlock()

	return m.conn.Close()
}
