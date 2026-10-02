// Package protomux implements a lightweight channel multiplexer for peer-to-peer protocols.
package protomux

import (
	"errors"
	"sync"
)

var (
	// ErrChannelClosed is returned when attempting to send on a closed channel.
	ErrChannelClosed = errors.New("protomux: channel is closed")
)

// MessageHandler is the callback function invoked when a message arrives for a channel.
type MessageHandler func(typeID uint64, payload []byte) error

// CloseHandler is the callback invoked when a channel is closed.
type CloseHandler func() error

// Channel represents an isolated protocol sub-stream over a Multiplexer.
type Channel struct {
	id        uint64
	remoteID  uint64
	protocol  string
	mux       *Multiplexer
	onMessage MessageHandler
	onClose   CloseHandler

	mu     sync.RWMutex
	closed bool
}

// ID returns the numeric channel ID.
func (c *Channel) ID() uint64 {
	return c.id
}

// RemoteID returns the remote peer's numeric channel ID.
func (c *Channel) RemoteID() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.remoteID
}

// Protocol returns the protocol name associated with this channel.
func (c *Channel) Protocol() string {
	return c.protocol
}

// SetMessageHandler updates the inbound message callback handler.
func (c *Channel) SetMessageHandler(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = handler
}

// SetCloseHandler updates the close callback handler.
func (c *Channel) SetCloseHandler(handler CloseHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onClose = handler
}

// SendMessage writes a typed protocol payload to this channel.
func (c *Channel) SendMessage(typeID uint64, payload []byte) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrChannelClosed
	}
	targetID := c.id
	if c.remoteID != 0 {
		targetID = c.remoteID
	}
	c.mu.RUnlock()

	return c.mux.sendFrame(targetID, typeID, payload)
}

// Close closes the channel and unregisters it from the multiplexer.
func (c *Channel) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	handler := c.onClose
	c.mu.Unlock()

	c.mux.unregisterChannel(c.id)

	if handler != nil {
		return handler()
	}
	return nil
}
