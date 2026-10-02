package udx

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// MagicByte is the canonical UDX magic byte identifier (255 / 0xFF).
	MagicByte byte = 255
	// Version is the canonical UDX protocol version (1).
	Version byte = 1

	// HeaderSize is the fixed size in bytes for a canonical libudx packet header (20 bytes):
	// Magic (1B) + Version (1B) + Type (1B) + DataOffset (1B) +
	// StreamID (4B LE) + Window (4B LE) + Seq (4B LE) + Ack (4B LE) = 20 Bytes.
	HeaderSize = 20
	// MaxPacketSize is the maximum MTU datagram size (1400 bytes to avoid IP fragmentation).
	MaxPacketSize = 1400
	// MaxPayloadSize is the maximum allowable payload per packet.
	MaxPayloadSize = MaxPacketSize - HeaderSize
)

// Canonical libudx header flags (UDX_HEADER_*):
const (
	// FlagDATA indicates payload datagram data (UDX_HEADER_DATA).
	FlagDATA byte = 1 << 0 // 0x01
	// FlagEND indicates graceful connection termination (UDX_HEADER_END / FIN).
	FlagEND byte = 1 << 1 // 0x02
	// FlagSACK indicates selective acknowledgment blocks (UDX_HEADER_SACK).
	FlagSACK byte = 1 << 2 // 0x04
	// FlagMESSAGE indicates ephemeral datagram message (UDX_HEADER_MESSAGE).
	FlagMESSAGE byte = 1 << 3 // 0x08
	// FlagDESTROY indicates immediate reset / stream destroy (UDX_HEADER_DESTROY).
	FlagDESTROY byte = 1 << 4 // 0x10
	// FlagHEARTBEAT / FlagPUNCH indicates probe / beacon (UDX_HEADER_HEARTBEAT).
	FlagHEARTBEAT byte = 1 << 5 // 0x20
	// FlagSYN indicates connection establishment request.
	FlagSYN byte = 1 << 6 // 0x40
	// FlagACK indicates acknowledgment.
	FlagACK byte = 1 << 7 // 0x80

	// Backward-compatible aliases
	FlagFIN   = FlagEND
	FlagPUNCH = FlagHEARTBEAT
)

var (
	ErrPacketTooShort     = errors.New("udx: packet shorter than header size")
	ErrPayloadTooLarge    = errors.New("udx: payload exceeds max datagram size")
	ErrInvalidHeaderMagic = errors.New("udx: invalid magic byte or version")
	ErrInvalidDataOffset  = errors.New("udx: data offset out of bounds")
)

// Packet represents a framed UDX reliable datagram matching canonical libudx layout.
type Packet struct {
	Magic      byte
	Version    byte
	Flags      byte
	DataOffset byte
	StreamID   uint32
	Window     uint32
	Seq        uint32
	Ack        uint32
	Payload    []byte
}

// Encode serializes a Packet into a byte buffer adhering to the 20-byte Little-Endian wire specification.
func (p *Packet) Encode() ([]byte, error) {
	if len(p.Payload) > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	buf := make([]byte, HeaderSize+len(p.Payload))
	magic := p.Magic
	if magic == 0 {
		magic = MagicByte
	}
	ver := p.Version
	if ver == 0 {
		ver = Version
	}

	buf[0] = magic
	buf[1] = ver
	buf[2] = p.Flags
	buf[3] = p.DataOffset

	binary.LittleEndian.PutUint32(buf[4:8], p.StreamID)
	binary.LittleEndian.PutUint32(buf[8:12], p.Window)
	binary.LittleEndian.PutUint32(buf[12:16], p.Seq)
	binary.LittleEndian.PutUint32(buf[16:20], p.Ack)

	if len(p.Payload) > 0 {
		copy(buf[HeaderSize:], p.Payload)
	}
	return buf, nil
}

// DecodePacket parses a raw UDP payload into a Packet adhering to canonical libudx.
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, ErrPacketTooShort
	}

	p := &Packet{
		Magic:      data[0],
		Version:    data[1],
		Flags:      data[2],
		DataOffset: data[3],
		StreamID:   binary.LittleEndian.Uint32(data[4:8]),
		Window:     binary.LittleEndian.Uint32(data[8:12]),
		Seq:        binary.LittleEndian.Uint32(data[12:16]),
		Ack:        binary.LittleEndian.Uint32(data[16:20]),
	}

	if p.Magic != MagicByte || p.Version != Version {
		return nil, ErrInvalidHeaderMagic
	}

	payloadStart := HeaderSize + int(p.DataOffset)
	if payloadStart > len(data) {
		return nil, ErrInvalidDataOffset
	}

	if len(data) > payloadStart {
		p.Payload = make([]byte, len(data)-payloadStart)
		copy(p.Payload, data[payloadStart:])
	}

	return p, nil
}

func (p *Packet) String() string {
	flagsStr := ""
	if p.Flags&FlagSYN != 0 {
		flagsStr += "SYN|"
	}
	if p.Flags&FlagACK != 0 {
		flagsStr += "ACK|"
	}
	if p.Flags&FlagEND != 0 {
		flagsStr += "END|"
	}
	if p.Flags&FlagDATA != 0 {
		flagsStr += "DATA|"
	}
	if p.Flags&FlagHEARTBEAT != 0 {
		flagsStr += "HEARTBEAT|"
	}
	if p.Flags&FlagSACK != 0 {
		flagsStr += "SACK|"
	}
	if p.Flags&FlagMESSAGE != 0 {
		flagsStr += "MESSAGE|"
	}
	if p.Flags&FlagDESTROY != 0 {
		flagsStr += "DESTROY|"
	}
	return fmt.Sprintf("UDXPacket{Stream:%d, Win:%d, Seq:%d, Ack:%d, Flags:%s, Len:%d}",
		p.StreamID, p.Window, p.Seq, p.Ack, flagsStr, len(p.Payload))
}
