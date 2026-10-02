// Package compactenc implements the Holepunch compact-encoding binary serialization standard.
package compactenc

import (
	"errors"
)

var (
	// ErrUnexpectedEOF is returned when decoding reaches the end of buffer prematurely.
	ErrUnexpectedEOF = errors.New("compactenc: unexpected end of buffer")
	// ErrBufferTooSmall is returned when encoding into a buffer with insufficient capacity.
	ErrBufferTooSmall = errors.New("compactenc: destination buffer too small")
	// ErrInvalidEncoding is returned when encoded data contains invalid headers or markers.
	ErrInvalidEncoding = errors.New("compactenc: invalid encoding header")
)

// State tracks the buffer, current cursor position, and bounds for encoding/decoding.
type State struct {
	Buffer []byte
	Start  int
	End    int
}

// NewState creates a State initialized with the given buffer.
func NewState(b []byte) *State {
	return &State{
		Buffer: b,
		Start:  0,
		End:    len(b),
	}
}

// NewAllocatedState creates a State with pre-allocated buffer of the specified capacity.
func NewAllocatedState(size int) *State {
	return &State{
		Buffer: make([]byte, size),
		Start:  0,
		End:    size,
	}
}

// EnsureCapacity ensures the state buffer has at least additional bytes available from Start.
func (s *State) EnsureCapacity(additional int) {
	required := s.Start + additional
	if required > len(s.Buffer) {
		newBuf := make([]byte, required*2)
		copy(newBuf, s.Buffer[:s.Start])
		s.Buffer = newBuf
		s.End = len(s.Buffer)
	}
}

// Bytes returns the written slice from beginning up to current Start position.
func (s *State) Bytes() []byte {
	return s.Buffer[:s.Start]
}

// Remaining returns the number of unread bytes between Start and End.
func (s *State) Remaining() int {
	if s.Start > s.End {
		return 0
	}
	return s.End - s.Start
}

// Reset resets cursor to 0 with existing buffer.
func (s *State) Reset() {
	s.Start = 0
	s.End = len(s.Buffer)
}
