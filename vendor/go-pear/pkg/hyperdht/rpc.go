package hyperdht

import (
	"errors"
	"net"

	"go-pear/pkg/compactenc"
)

var (
	// ErrExcessivePeers is returned when RPC message contains an invalid or excessive peer count.
	ErrExcessivePeers = errors.New("hyperdht: excessive or invalid peer count in RPC response")
)

const (
	// MaxPeersPerRPC is the maximum number of peer records permitted in an RPC response packet.
	MaxPeersPerRPC = 128
)

// RPC Message Types
const (
	TypePing       = 0
	TypeFindNode   = 1
	TypeAnnounce   = 2
	TypeUnannounce = 3
	TypeLookup     = 4
)

// RPCFlags
const (
	RPCFlagResponse = 0b00000001
	RPCFlagError    = 0b00000010
)

// RPCMessage represents a framed HyperDHT UDP packet.
type RPCMessage struct {
	TID      uint64   // Transaction ID
	Type     uint64   // Message type
	Flags    uint64   // Flags (Request vs Response, Error)
	SenderID [32]byte // Sender Node ID
	Target   [32]byte // Target Topic or Node ID
	Port     uint64   // Optional port (for Announce)
	Peers    []PeerInfo
	ErrorMsg string
}

// PeerInfo contains peer contact details in RPC messages.
type PeerInfo struct {
	ID        [32]byte
	IP        net.IP
	Port      int
	PublicKey [32]byte
}

func PreencodeRPCMessage(m RPCMessage) int {
	sz := compactenc.PreencodeUint(m.TID) +
		compactenc.PreencodeUint(m.Type) +
		compactenc.PreencodeUint(m.Flags) +
		32 + 32 +
		compactenc.PreencodeUint(m.Port) +
		compactenc.PreencodeString(m.ErrorMsg) +
		compactenc.PreencodeUint(uint64(len(m.Peers)))

	for _, p := range m.Peers {
		sz += 32 + compactenc.PreencodeBuffer(p.IP) + compactenc.PreencodeUint(uint64(p.Port)) + 32
	}
	return sz
}

func EncodeRPCMessage(s *compactenc.State, m RPCMessage) error {
	if err := compactenc.EncodeUint(s, m.TID); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(s, m.Type); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(s, m.Flags); err != nil {
		return err
	}
	if err := compactenc.EncodeFixed32(s, m.SenderID); err != nil {
		return err
	}
	if err := compactenc.EncodeFixed32(s, m.Target); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(s, m.Port); err != nil {
		return err
	}
	if err := compactenc.EncodeString(s, m.ErrorMsg); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(s, uint64(len(m.Peers))); err != nil {
		return err
	}
	for _, p := range m.Peers {
		if err := compactenc.EncodeFixed32(s, p.ID); err != nil {
			return err
		}
		if err := compactenc.EncodeBuffer(s, p.IP); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(s, uint64(p.Port)); err != nil {
			return err
		}
		if err := compactenc.EncodeFixed32(s, p.PublicKey); err != nil {
			return err
		}
	}
	return nil
}

func DecodeRPCMessage(s *compactenc.State) (RPCMessage, error) {
	var m RPCMessage
	var err error
	m.TID, err = compactenc.DecodeUint(s)
	if err != nil {
		return m, err
	}
	m.Type, err = compactenc.DecodeUint(s)
	if err != nil {
		return m, err
	}
	m.Flags, err = compactenc.DecodeUint(s)
	if err != nil {
		return m, err
	}
	m.SenderID, err = compactenc.DecodeFixed32(s)
	if err != nil {
		return m, err
	}
	m.Target, err = compactenc.DecodeFixed32(s)
	if err != nil {
		return m, err
	}
	m.Port, err = compactenc.DecodeUint(s)
	if err != nil {
		return m, err
	}
	m.ErrorMsg, err = compactenc.DecodeString(s)
	if err != nil {
		return m, err
	}

	numPeers, err := compactenc.DecodeUint(s)
	if err != nil {
		return m, err
	}
	remaining := len(s.Buffer) - s.Start
	if numPeers > MaxPeersPerRPC || (remaining >= 0 && numPeers > uint64(remaining/66+1)) {
		return m, ErrExcessivePeers
	}
	m.Peers = make([]PeerInfo, numPeers)
	for i := uint64(0); i < numPeers; i++ {
		pID, err := compactenc.DecodeFixed32(s)
		if err != nil {
			return m, err
		}
		ipBuf, err := compactenc.DecodeBuffer(s)
		if err != nil {
			return m, err
		}
		port, err := compactenc.DecodeUint(s)
		if err != nil {
			return m, err
		}
		pubKey, err := compactenc.DecodeFixed32(s)
		if err != nil {
			return m, err
		}
		m.Peers[i] = PeerInfo{
			ID:        pID,
			IP:        net.IP(ipBuf),
			Port:      int(port),
			PublicKey: pubKey,
		}
	}
	return m, nil
}
