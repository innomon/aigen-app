package udx

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	stunMagicCookie = 0x2112A442
	stunBindingReq  = 0x0001
	stunBindingResp = 0x0101
	attrMappedAddr  = 0x0001
	attrXORMapped   = 0x0020
)

// CandidateType classifies a discovered network candidate.
type CandidateType string

const (
	CandidateHost            CandidateType = "host"
	CandidateServerReflexive CandidateType = "srflx"
)

// Candidate represents a candidate network endpoint for NAT holepunching.
type Candidate struct {
	Type     CandidateType
	Addr     *net.UDPAddr
	Priority int
}

// PunchOptions configures the UDP holepunching sequence.
type PunchOptions struct {
	Bursts   int
	Interval time.Duration
}

// DefaultPunchOptions provides reasonable defaults for holepunching bursts.
var DefaultPunchOptions = PunchOptions{
	Bursts:   5,
	Interval: 50 * time.Millisecond,
}

// CreateSTUNBindingRequest constructs a standard 20-byte RFC 5389 Binding Request.
func CreateSTUNBindingRequest() ([]byte, [12]byte, error) {
	var txID [12]byte
	if _, err := rand.Read(txID[:]); err != nil {
		return nil, txID, err
	}
	req := make([]byte, 20)
	binary.BigEndian.PutUint16(req[0:2], stunBindingReq)
	binary.BigEndian.PutUint16(req[2:4], 0) // message length (0 attributes)
	binary.BigEndian.PutUint32(req[4:8], stunMagicCookie)
	copy(req[8:20], txID[:])
	return req, txID, nil
}

// ParseSTUNBindingResponse decodes a STUN binding response and extracts the reflexive address.
func ParseSTUNBindingResponse(data []byte, expectedTxID [12]byte) (*net.UDPAddr, error) {
	if len(data) < 20 {
		return nil, errors.New("stun: response too short")
	}
	msgType := binary.BigEndian.Uint16(data[0:2])
	if msgType != stunBindingResp {
		return nil, fmt.Errorf("stun: unexpected message type 0x%04x", msgType)
	}
	cookie := binary.BigEndian.Uint32(data[4:8])
	if cookie != stunMagicCookie {
		return nil, errors.New("stun: invalid magic cookie")
	}
	if !bytes.Equal(data[8:20], expectedTxID[:]) {
		return nil, errors.New("stun: transaction ID mismatch")
	}
	msgLen := int(binary.BigEndian.Uint16(data[2:4]))
	if len(data) < 20+msgLen {
		return nil, errors.New("stun: truncated attributes")
	}

	offset := 20
	end := 20 + msgLen
	for offset+4 <= end {
		attrType := binary.BigEndian.Uint16(data[offset : offset+2])
		attrLen := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		offset += 4
		if offset+attrLen > end {
			break
		}
		val := data[offset : offset+attrLen]
		offset += (attrLen + 3) & ^3 // 4-byte attribute alignment

		if attrType == attrXORMapped && len(val) >= 8 {
			family := val[1]
			rawPort := binary.BigEndian.Uint16(val[2:4])
			port := int(rawPort ^ uint16(stunMagicCookie>>16))
			if family == 0x01 { // IPv4
				rawIP := binary.BigEndian.Uint32(val[4:8])
				ipBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(ipBytes, rawIP^stunMagicCookie)
				return &net.UDPAddr{IP: net.IP(ipBytes), Port: port}, nil
			}
		} else if attrType == attrMappedAddr && len(val) >= 8 {
			family := val[1]
			port := int(binary.BigEndian.Uint16(val[2:4]))
			if family == 0x01 {
				ip := net.IPv4(val[4], val[5], val[6], val[7])
				return &net.UDPAddr{IP: ip, Port: port}, nil
			}
		}
	}
	return nil, errors.New("stun: no mapped address attribute found")
}

// QuerySTUN contacts a STUN server to discover the local endpoint's external reflexive address.
func QuerySTUN(ctx context.Context, stunServer string) (*net.UDPAddr, error) {
	req, txID, err := CreateSTUNBindingRequest()
	if err != nil {
		return nil, err
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", stunServer)
	if err != nil {
		return nil, fmt.Errorf("failed to dial STUN server: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	}

	if _, err := conn.Write(req); err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	resp := make([]byte, 1024)
	n, err := conn.Read(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read STUN response: %w", err)
	}

	return ParseSTUNBindingResponse(resp[:n], txID)
}

// ResolveLocalCandidates discovers non-loopback active IPv4 addresses from local network interfaces.
func ResolveLocalCandidates(port int) ([]*net.UDPAddr, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var addrs []*net.UDPAddr
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		uaddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range uaddrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				addrs = append(addrs, &net.UDPAddr{IP: ip, Port: port})
			}
		}
	}
	return addrs, nil
}

// ResolveCandidates aggregates local interface candidates and external STUN reflexive candidates.
func ResolveCandidates(ctx context.Context, port int, stunServers ...string) ([]*Candidate, error) {
	var results []*Candidate

	// 1. Host candidates
	locals, _ := ResolveLocalCandidates(port)
	for _, l := range locals {
		results = append(results, &Candidate{
			Type:     CandidateHost,
			Addr:     l,
			Priority: 100,
		})
	}

	// 2. STUN server reflexive candidate
	if len(stunServers) == 0 {
		stunServers = []string{"stun.l.google.com:19302"}
	}
	for _, server := range stunServers {
		stunCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		extAddr, err := QuerySTUN(stunCtx, server)
		cancel()
		if err == nil && extAddr != nil {
			results = append(results, &Candidate{
				Type:     CandidateServerReflexive,
				Addr:     extAddr,
				Priority: 200,
			})
			break
		}
	}

	// 3. Fallback to loopback if no interface addresses found
	if len(results) == 0 {
		results = append(results, &Candidate{
			Type:     CandidateHost,
			Addr:     &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port},
			Priority: 10,
		})
	}

	return results, nil
}

// Holepunch initiates bidirectional UDP packet bursting towards the remote target to open NAT bindings,
// followed by a reliable UDX virtual stream handshake.
func Holepunch(ctx context.Context, socket *Socket, target *net.UDPAddr, opts ...PunchOptions) (*Stream, error) {
	opt := DefaultPunchOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	punchPacket := &Packet{
		Flags:    FlagPUNCH,
		StreamID: 0,
		Seq:      0,
		Ack:      0,
	}

	// Send burst of punch packets to open local outbound firewall state
	for i := 0; i < opt.Bursts; i++ {
		_ = socket.sendPacket(punchPacket, target)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(opt.Interval):
		}
	}

	// Attempt UDX stream dial
	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return socket.Dial(dialCtx, target)
}
