package secretstream

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Conn implements a secure, encrypted net.Conn stream using Noise XX.
type Conn struct {
	raw          net.Conn
	tx           *CipherState
	rx           *CipherState
	localStatic  [KeySize]byte
	remoteStatic [KeySize]byte

	readBuf bytes.Buffer
	readMu  sync.Mutex
	writeMu sync.Mutex
	closeMu sync.Mutex
	closed  bool
}

// DefaultHandshakeTimeout is the standard timeout duration for Noise XX handshakes (10 seconds).
const DefaultHandshakeTimeout = 10 * time.Second

// Upgrade initiates or responds to a Noise XX handshake over an existing net.Conn with default 10s timeout.
func Upgrade(raw net.Conn, localStatic *KeyPair, isInitiator bool) (*Conn, error) {
	return UpgradeWithTimeout(raw, localStatic, isInitiator, DefaultHandshakeTimeout)
}

// UpgradeWithTimeout initiates or responds to a Noise XX handshake with a configurable deadline.
func UpgradeWithTimeout(raw net.Conn, localStatic *KeyPair, isInitiator bool, timeout time.Duration) (*Conn, error) {
	hs, err := NewHandshakeState(localStatic, isInitiator)
	if err != nil {
		return nil, err
	}
	return upgradeWithHandshake(raw, hs, timeout)
}

// UpgradeWithPSK initiates or responds to a Noise XX handshake with a pre-shared key over an existing net.Conn.
func UpgradeWithPSK(raw net.Conn, localStatic *KeyPair, isInitiator bool, psk [32]byte) (*Conn, error) {
	return UpgradeWithPSKTimeout(raw, localStatic, isInitiator, psk, DefaultHandshakeTimeout)
}

// UpgradeWithPSKTimeout initiates or responds to a Noise XX handshake with a pre-shared key and configurable deadline.
func UpgradeWithPSKTimeout(raw net.Conn, localStatic *KeyPair, isInitiator bool, psk [32]byte, timeout time.Duration) (*Conn, error) {
	hs, err := NewHandshakeStateWithPSK(localStatic, isInitiator, &psk)
	if err != nil {
		return nil, err
	}
	return upgradeWithHandshake(raw, hs, timeout)
}

func upgradeWithHandshake(raw net.Conn, hs *HandshakeState, timeout time.Duration) (*Conn, error) {
	if timeout <= 0 {
		timeout = DefaultHandshakeTimeout
	}
	_ = raw.SetDeadline(time.Now().Add(timeout))
	defer func() {
		_ = raw.SetDeadline(time.Time{})
	}()
	if hs.isInitiator {
		// Msg 1 -> Send
		m1, err := hs.WriteMessage(nil)
		if err != nil {
			return nil, err
		}
		if err := writeHandshakeFrame(raw, m1); err != nil {
			return nil, err
		}

		// Msg 2 <- Read
		m2, err := readHandshakeFrame(raw)
		if err != nil {
			return nil, err
		}
		if _, err := hs.ReadMessage(m2); err != nil {
			return nil, err
		}

		// Msg 3 -> Send
		m3, err := hs.WriteMessage(nil)
		if err != nil {
			return nil, err
		}
		if err := writeHandshakeFrame(raw, m3); err != nil {
			return nil, err
		}
	} else {
		// Msg 1 <- Read
		m1, err := readHandshakeFrame(raw)
		if err != nil {
			return nil, err
		}
		if _, err := hs.ReadMessage(m1); err != nil {
			return nil, err
		}

		// Msg 2 -> Send
		m2, err := hs.WriteMessage(nil)
		if err != nil {
			return nil, err
		}
		if err := writeHandshakeFrame(raw, m2); err != nil {
			return nil, err
		}

		// Msg 3 <- Read
		m3, err := readHandshakeFrame(raw)
		if err != nil {
			return nil, err
		}
		if _, err := hs.ReadMessage(m3); err != nil {
			return nil, err
		}
	}

	tx, rx := hs.Finalize()

	c := &Conn{
		raw:          raw,
		tx:           tx,
		rx:           rx,
		localStatic:  hs.localStatic.Public,
		remoteStatic: hs.remoteStatic,
	}
	return c, nil
}

func writeHandshakeFrame(w io.Writer, payload []byte) error {
	var lenBuf [2]byte
	binary.LittleEndian.PutUint16(lenBuf[:], uint16(len(payload)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readHandshakeFrame(r io.Reader) ([]byte, error) {
	var lenBuf [2]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint16(lenBuf[:])
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// RemotePublicKey returns the peer's authenticated Curve25519 static public key.
func (c *Conn) RemotePublicKey() [KeySize]byte {
	return c.remoteStatic
}

// LocalPublicKey returns the local authenticated Curve25519 static public key.
func (c *Conn) LocalPublicKey() [KeySize]byte {
	return c.localStatic
}

// Read reads and decrypts stream frames from connection.
func (c *Conn) Read(b []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	if c.readBuf.Len() > 0 {
		return c.readBuf.Read(b)
	}

	if c.closed {
		return 0, ErrConnectionClosed
	}

	for c.readBuf.Len() == 0 {
		var lenBuf [2]byte
		if _, err := io.ReadFull(c.raw, lenBuf[:]); err != nil {
			return 0, err
		}
		frameLen := binary.LittleEndian.Uint16(lenBuf[:])
		if frameLen < TagSize {
			return 0, fmt.Errorf("invalid frame length: %d", frameLen)
		}

		cipherBuf := make([]byte, frameLen)
		if _, err := io.ReadFull(c.raw, cipherBuf); err != nil {
			return 0, err
		}

		plain, err := c.rx.DecryptWithAd(nil, cipherBuf)
		if err != nil {
			return 0, fmt.Errorf("failed to decrypt stream frame: %w", err)
		}

		c.readBuf.Write(plain)
	}

	return c.readBuf.Read(b)
}

// Write encrypts and writes stream frames to connection.
func (c *Conn) Write(b []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.closed {
		return 0, ErrConnectionClosed
	}

	totalWritten := 0
	for len(b) > 0 {
		chunkSize := len(b)
		if chunkSize > MaxFrameSize {
			chunkSize = MaxFrameSize
		}

		chunk := b[:chunkSize]
		ciphertext := c.tx.EncryptWithAd(nil, chunk)

		var lenBuf [2]byte
		binary.LittleEndian.PutUint16(lenBuf[:], uint16(len(ciphertext)))

		if _, err := c.raw.Write(lenBuf[:]); err != nil {
			return totalWritten, err
		}
		if _, err := c.raw.Write(ciphertext); err != nil {
			return totalWritten, err
		}

		totalWritten += chunkSize
		b = b[chunkSize:]
	}

	return totalWritten, nil
}

// Close closes the underlying transport connection.
func (c *Conn) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	c.closed = true
	return c.raw.Close()
}

func (c *Conn) LocalAddr() net.Addr                { return c.raw.LocalAddr() }
func (c *Conn) RemoteAddr() net.Addr               { return c.raw.RemoteAddr() }
func (c *Conn) SetDeadline(t time.Time) error      { return c.raw.SetDeadline(t) }
func (c *Conn) SetReadDeadline(t time.Time) error  { return c.raw.SetReadDeadline(t) }
func (c *Conn) SetWriteDeadline(t time.Time) error { return c.raw.SetWriteDeadline(t) }
