package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// DefaultBeaconPort is the standard UDP broadcast port for local cluster discovery.
const DefaultBeaconPort = 49736

// BeaconMessage carries discovery rendezvous details over LAN broadcast.
type BeaconMessage struct {
	Cluster string `json:"cluster"`
	Topic   string `json:"topic"`
	DHTAddr string `json:"dht_addr"`
	NodeID  string `json:"node_id"`
}

// ResolveRoutableIP detects the primary outbound IPv4 address of the local machine.
func ResolveRoutableIP() (net.IP, error) {
	// Attempt outbound route discovery using UDP dial (no actual packet is transmitted)
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		if localAddr.IP != nil && !localAddr.IP.IsLoopback() && localAddr.IP.To4() != nil {
			return localAddr.IP.To4(), nil
		}
	}

	// Fallback: enumerate network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return net.IPv4(127, 0, 0, 1), err
	}

	for _, iface := range ifaces {
		// Skip down and loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				ip4 := ipNet.IP.To4()
				if ip4 != nil && !ip4.IsLoopback() {
					return ip4, nil
				}
			}
		}
	}

	return net.IPv4(127, 0, 0, 1), nil
}

// NormalizeDHTAddr converts an unspecified IP (0.0.0.0 or [::]) to a reachable LAN IP.
func NormalizeDHTAddr(addrStr string) string {
	host, portStr, err := net.SplitHostPort(addrStr)
	if err != nil {
		return addrStr
	}

	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		routableIP, err := ResolveRoutableIP()
		if err == nil {
			return net.JoinHostPort(routableIP.String(), portStr)
		}
	}
	return addrStr
}

// StartBeaconBroadcaster periodically sends UDP broadcast beacons on the LAN.
func StartBeaconBroadcaster(ctx context.Context, msg BeaconMessage, port int, interval time.Duration) error {
	if port <= 0 {
		port = DefaultBeaconPort
	}
	if interval <= 0 {
		interval = 2500 * time.Millisecond
	}

	// Ensure DHTAddr is routable across LAN
	msg.DHTAddr = NormalizeDHTAddr(msg.DHTAddr)

	targetAddr := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: port,
	}

	conn, err := net.DialUDP("udp4", nil, targetAddr)
	if err != nil {
		return fmt.Errorf("failed to open UDP broadcast socket: %w", err)
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("marshal beacon message: %w", err)
	}

	go func() {
		defer conn.Close()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Initial broadcast
		_, _ = conn.Write(payload)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = conn.Write(payload)
			}
		}
	}()

	return nil
}

// DiscoverLANBootstrap listens for a matching beacon on the LAN for up to timeout duration.
func DiscoverLANBootstrap(ctx context.Context, cluster, topic string, port int, timeout time.Duration) (string, error) {
	if port <= 0 {
		port = DefaultBeaconPort
	}
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}

	listenAddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: port,
	}

	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		return "", fmt.Errorf("failed to listen on beacon port %d: %w", port, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)

	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return "", err // e.g. timeout
		}

		var msg BeaconMessage
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		// Filter by cluster and topic if specified
		if cluster != "" && !strings.EqualFold(msg.Cluster, cluster) {
			continue
		}
		if topic != "" && msg.Topic != "" && !strings.EqualFold(msg.Topic, topic) {
			continue
		}

		if msg.DHTAddr != "" {
			return msg.DHTAddr, nil
		}
	}
}

// FormatDHTEndpoint formats an IP and port into a valid DHT endpoint string.
func FormatDHTEndpoint(ip net.IP, port int) string {
	return net.JoinHostPort(ip.String(), strconv.Itoa(port))
}
