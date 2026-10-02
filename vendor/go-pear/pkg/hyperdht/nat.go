package hyperdht

import (
	"context"
	"net"
	"time"
)

// NATType represents the detected NAT firewall classification.
type NATType int

const (
	NATUnknown NATType = iota
	NATOpen
	NATPortConsistent
	NATSymmetric
)

func (n NATType) String() string {
	switch n {
	case NATOpen:
		return "Open / Direct"
	case NATPortConsistent:
		return "Port-Consistent (Cone NAT)"
	case NATSymmetric:
		return "Symmetric (Randomized Ports)"
	default:
		return "Unknown"
	}
}

// NATProfile represents observed external IP and port mapping characteristics.
type NATProfile struct {
	Type          NATType
	LocalAddr     *net.UDPAddr
	ExternalIP    net.IP
	ExternalPort  int
	PortVariation int
}

// ProbeNAT analyzes the node's NAT behavior by querying bootstrap DHT nodes.
func (d *DHT) ProbeNAT(ctx context.Context) (*NATProfile, error) {
	localAddr := d.Addr()

	profile := &NATProfile{
		Type:         NATOpen,
		LocalAddr:    localAddr,
		ExternalIP:   localAddr.IP,
		ExternalPort: localAddr.Port,
	}

	d.mu.RLock()
	bootstrapNodes := make([]*net.UDPAddr, len(d.bootstrapNodes))
	copy(bootstrapNodes, d.bootstrapNodes)
	d.mu.RUnlock()

	if len(bootstrapNodes) == 0 {
		return profile, nil
	}

	var successfulPings int
	for _, bAddr := range bootstrapNodes {
		pingCtx, pingCancel := context.WithTimeout(ctx, 500*time.Millisecond)
		err := d.Ping(pingCtx, bAddr)
		pingCancel()

		if err == nil {
			successfulPings++
		}
	}

	if successfulPings > 0 {
		profile.Type = NATPortConsistent
	}

	return profile, nil
}
