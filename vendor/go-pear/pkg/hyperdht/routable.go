package hyperdht

import (
	"net"
)

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
