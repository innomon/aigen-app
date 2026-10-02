package hyperdht

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go-pear/pkg/compactenc"
)

var (
	ErrDHTClosed      = errors.New("hyperdht: node is closed")
	ErrRequestTimeout = errors.New("hyperdht: request timed out")
)

type packetJob struct {
	data       []byte
	remoteAddr *net.UDPAddr
}

// DHT represents a HyperDHT node.
type DHT struct {
	mu         sync.RWMutex
	ID         [32]byte
	PublicKey  [32]byte
	conn       *net.UDPConn
	routableIP net.IP
	table      *RoutingTable

	topics   map[[32]byte]map[[32]byte]*PeerEndpoint
	inflight map[uint64]chan *RPCMessage
	nextTID  uint64

	bootstrapNodes []*net.UDPAddr
	closed         bool
	doneChan       chan struct{}
	packetQueue    chan packetJob
}

// Config configures DHT node parameters.
type Config struct {
	ID             [32]byte
	PublicKey      [32]byte
	Port           int
	Bootstrap      []string
	WorkerPoolSize int
}

// NewDHT creates a new HyperDHT instance.
func NewDHT(cfg Config) (*DHT, error) {
	id := cfg.ID
	if id == [32]byte{} {
		_, _ = rand.Read(id[:])
	}
	pub := cfg.PublicKey
	if pub == [32]byte{} {
		pub = id
	}

	addr := &net.UDPAddr{IP: net.IPv4zero, Port: cfg.Port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP socket: %w", err)
	}

	routableIP, _ := ResolveRoutableIP()

	workerPoolSize := cfg.WorkerPoolSize
	if workerPoolSize <= 0 {
		workerPoolSize = 64
	}

	d := &DHT{
		ID:          id,
		PublicKey:   pub,
		conn:        conn,
		routableIP:  routableIP,
		table:       NewRoutingTable(id),
		topics:      make(map[[32]byte]map[[32]byte]*PeerEndpoint),
		inflight:    make(map[uint64]chan *RPCMessage),
		nextTID:     1,
		doneChan:    make(chan struct{}),
		packetQueue: make(chan packetJob, workerPoolSize*4),
	}

	for _, b := range cfg.Bootstrap {
		uAddr, err := net.ResolveUDPAddr("udp", b)
		if err == nil {
			d.bootstrapNodes = append(d.bootstrapNodes, uAddr)
		}
	}

	// Start worker pool to process incoming packets without unbounded goroutine spawning
	for i := 0; i < workerPoolSize; i++ {
		go func() {
			for {
				select {
				case <-d.doneChan:
					return
				case job, ok := <-d.packetQueue:
					if !ok {
						return
					}
					d.handlePacket(job.data, job.remoteAddr)
				}
			}
		}()
	}

	go d.readLoop()
	return d, nil
}

// Addr returns the local listening UDP address.
func (d *DHT) Addr() *net.UDPAddr {
	return d.conn.LocalAddr().(*net.UDPAddr)
}

// RoutableAddr returns the local listening UDP address with a reachable non-loopback IP if available.
func (d *DHT) RoutableAddr() *net.UDPAddr {
	addr := d.Addr()
	if addr.IP == nil || addr.IP.IsUnspecified() {
		if d.routableIP != nil && !d.routableIP.IsUnspecified() {
			return &net.UDPAddr{IP: d.routableIP, Port: addr.Port}
		}
	}
	return addr
}

// Port returns the local listening UDP port.
func (d *DHT) Port() int {
	return d.Addr().Port
}

// readLoop listens for incoming UDP RPC frames and dispatches them to the capped worker pool.
func (d *DHT) readLoop() {
	buf := make([]byte, 65535)
	for {
		n, remoteAddr, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-d.doneChan:
				return
			default:
				continue
			}
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])
		select {
		case d.packetQueue <- packetJob{data: packet, remoteAddr: remoteAddr}:
		case <-d.doneChan:
			return
		default:
			// Queue full under packet flood: discard to prevent memory exhaustion
		}
	}
}

func (d *DHT) handlePacket(data []byte, remoteAddr *net.UDPAddr) {
	st := compactenc.NewState(data)
	msg, err := DecodeRPCMessage(st)
	if err != nil {
		return
	}

	// Update routing table with sender endpoint
	d.table.Add(&PeerEndpoint{
		ID:       msg.SenderID,
		Addr:     remoteAddr,
		LastSeen: time.Now(),
	})

	if (msg.Flags & RPCFlagResponse) != 0 {
		d.mu.Lock()
		ch, ok := d.inflight[msg.TID]
		if ok {
			delete(d.inflight, msg.TID)
		}
		d.mu.Unlock()
		if ok && ch != nil {
			select {
			case ch <- &msg:
			default:
			}
		}
		return
	}

	// Handle Request
	switch msg.Type {
	case TypePing:
		resp := RPCMessage{
			TID:      msg.TID,
			Type:     TypePing,
			Flags:    RPCFlagResponse,
			SenderID: d.ID,
		}
		_ = d.sendRaw(resp, remoteAddr)

	case TypeFindNode:
		closest := d.table.Closest(msg.Target, 20)
		var peers []PeerInfo
		for _, c := range closest {
			peers = append(peers, PeerInfo{
				ID:        c.ID,
				IP:        c.Addr.IP,
				Port:      c.Addr.Port,
				PublicKey: c.PublicKey,
			})
		}
		resp := RPCMessage{
			TID:      msg.TID,
			Type:     TypeFindNode,
			Flags:    RPCFlagResponse,
			SenderID: d.ID,
			Target:   msg.Target,
			Peers:    peers,
		}
		_ = d.sendRaw(resp, remoteAddr)

	case TypeAnnounce:
		port := int(msg.Port)
		if port == 0 {
			port = remoteAddr.Port
		}
		senderIP := remoteAddr.IP
		if senderIP == nil || senderIP.IsUnspecified() {
			senderIP = net.IPv4(127, 0, 0, 1)
		}
		d.mu.Lock()
		ann, ok := d.topics[msg.Target]
		if !ok {
			ann = make(map[[32]byte]*PeerEndpoint)
			d.topics[msg.Target] = ann
		}
		ann[msg.SenderID] = &PeerEndpoint{
			ID:        msg.SenderID,
			Addr:      &net.UDPAddr{IP: senderIP, Port: port},
			PublicKey: msg.SenderID,
			LastSeen:  time.Now(),
		}
		d.mu.Unlock()

		resp := RPCMessage{
			TID:      msg.TID,
			Type:     TypeAnnounce,
			Flags:    RPCFlagResponse,
			SenderID: d.ID,
			Target:   msg.Target,
		}
		_ = d.sendRaw(resp, remoteAddr)

	case TypeUnannounce:
		d.mu.Lock()
		if ann, ok := d.topics[msg.Target]; ok {
			delete(ann, msg.SenderID)
		}
		d.mu.Unlock()

		resp := RPCMessage{
			TID:      msg.TID,
			Type:     TypeUnannounce,
			Flags:    RPCFlagResponse,
			SenderID: d.ID,
			Target:   msg.Target,
		}
		_ = d.sendRaw(resp, remoteAddr)

	case TypeLookup:
		d.mu.RLock()
		ann := d.topics[msg.Target]
		var peers []PeerInfo
		for _, p := range ann {
			peerIP := p.Addr.IP
			if peerIP == nil || peerIP.IsUnspecified() {
				if d.routableIP != nil && !d.routableIP.IsUnspecified() {
					peerIP = d.routableIP
				} else {
					peerIP = net.IPv4(127, 0, 0, 1)
				}
			}
			peers = append(peers, PeerInfo{
				ID:        p.ID,
				IP:        peerIP,
				Port:      p.Addr.Port,
				PublicKey: p.PublicKey,
			})
		}
		d.mu.RUnlock()

		resp := RPCMessage{
			TID:      msg.TID,
			Type:     TypeLookup,
			Flags:    RPCFlagResponse,
			SenderID: d.ID,
			Target:   msg.Target,
			Peers:    peers,
		}
		_ = d.sendRaw(resp, remoteAddr)
	}
}

func (d *DHT) sendRaw(m RPCMessage, addr *net.UDPAddr) error {
	sz := PreencodeRPCMessage(m)
	st := compactenc.NewAllocatedState(sz)
	if err := EncodeRPCMessage(st, m); err != nil {
		return err
	}
	_, err := d.conn.WriteToUDP(st.Bytes(), addr)
	return err
}

func (d *DHT) request(ctx context.Context, m RPCMessage, addr *net.UDPAddr) (*RPCMessage, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, ErrDHTClosed
	}
	tid := d.nextTID
	d.nextTID++
	m.TID = tid
	m.SenderID = d.ID

	ch := make(chan *RPCMessage, 1)
	d.inflight[tid] = ch
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.inflight, tid)
		d.mu.Unlock()
	}()

	if err := d.sendRaw(m, addr); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(3 * time.Second):
		return nil, ErrRequestTimeout
	case resp := <-ch:
		return resp, nil
	}
}

// Ping sends a ping RPC to the specified UDP address.
func (d *DHT) Ping(ctx context.Context, addr *net.UDPAddr) error {
	msg := RPCMessage{Type: TypePing}
	_, err := d.request(ctx, msg, addr)
	return err
}

// Announce announces the local peer endpoint for a 32-byte topic key.
func (d *DHT) Announce(ctx context.Context, topic [32]byte, port int) error {
	localIP := d.routableIP
	if localIP == nil || localIP.IsUnspecified() {
		localIP = d.Addr().IP
	}
	if localIP == nil || localIP.IsUnspecified() {
		localIP = net.IPv4(127, 0, 0, 1)
	}

	d.mu.Lock()
	ann, ok := d.topics[topic]
	if !ok {
		ann = make(map[[32]byte]*PeerEndpoint)
		d.topics[topic] = ann
	}
	ann[d.ID] = &PeerEndpoint{
		ID:        d.ID,
		Addr:      &net.UDPAddr{IP: localIP, Port: port},
		PublicKey: d.PublicKey,
		LastSeen:  time.Now(),
	}
	d.mu.Unlock()

	// Query closest nodes in routing table and announce
	targets := d.table.Closest(topic, 20)
	for _, node := range targets {
		msg := RPCMessage{
			Type:   TypeAnnounce,
			Target: topic,
			Port:   uint64(port),
		}
		_, _ = d.request(ctx, msg, node.Addr)
	}
	for _, b := range d.bootstrapNodes {
		msg := RPCMessage{
			Type:   TypeAnnounce,
			Target: topic,
			Port:   uint64(port),
		}
		_, _ = d.request(ctx, msg, b)
	}
	return nil
}

// Unannounce removes the local peer announcement for a 32-byte topic key.
func (d *DHT) Unannounce(ctx context.Context, topic [32]byte) error {
	d.mu.Lock()
	if ann, ok := d.topics[topic]; ok {
		delete(ann, d.ID)
	}
	d.mu.Unlock()

	targets := d.table.Closest(topic, 20)
	for _, node := range targets {
		msg := RPCMessage{
			Type:   TypeUnannounce,
			Target: topic,
		}
		_, _ = d.request(ctx, msg, node.Addr)
	}
	return nil
}

// Lookup finds announcing peers for a 32-byte topic key.
func (d *DHT) Lookup(ctx context.Context, topic [32]byte) ([]*PeerEndpoint, error) {
	d.mu.RLock()
	localPeers := d.topics[topic]
	resMap := make(map[[32]byte]*PeerEndpoint)
	for id, p := range localPeers {
		resMap[id] = p
	}
	d.mu.RUnlock()

	targets := d.table.Closest(topic, 20)
	for _, node := range targets {
		if node.Addr == nil {
			continue
		}
		msg := RPCMessage{
			Type:   TypeLookup,
			Target: topic,
		}
		resp, err := d.request(ctx, msg, node.Addr)
		if err == nil && resp != nil {
			for _, p := range resp.Peers {
				peerIP := p.IP
				if peerIP == nil || peerIP.IsUnspecified() {
					peerIP = node.Addr.IP
				}
				if existing, ok := resMap[p.ID]; ok && existing.Addr != nil && !existing.Addr.IP.IsUnspecified() && !existing.Addr.IP.IsLoopback() {
					if peerIP.IsUnspecified() || peerIP.IsLoopback() {
						peerIP = existing.Addr.IP
					}
				}
				resMap[p.ID] = &PeerEndpoint{
					ID:        p.ID,
					Addr:      &net.UDPAddr{IP: peerIP, Port: p.Port},
					PublicKey: p.PublicKey,
					LastSeen:  time.Now(),
				}
			}
		}
	}

	for _, b := range d.bootstrapNodes {
		if b == nil {
			continue
		}
		msg := RPCMessage{
			Type:   TypeLookup,
			Target: topic,
		}
		resp, err := d.request(ctx, msg, b)
		if err == nil && resp != nil {
			for _, p := range resp.Peers {
				peerIP := p.IP
				if peerIP == nil || peerIP.IsUnspecified() {
					peerIP = b.IP
				}
				if existing, ok := resMap[p.ID]; ok && existing.Addr != nil && !existing.Addr.IP.IsUnspecified() && !existing.Addr.IP.IsLoopback() {
					if peerIP.IsUnspecified() || peerIP.IsLoopback() {
						peerIP = existing.Addr.IP
					}
				}
				resMap[p.ID] = &PeerEndpoint{
					ID:        p.ID,
					Addr:      &net.UDPAddr{IP: peerIP, Port: p.Port},
					PublicKey: p.PublicKey,
					LastSeen:  time.Now(),
				}
			}
		}
	}

	var results []*PeerEndpoint
	for _, p := range resMap {
		results = append(results, p)
	}
	return results, nil
}

// Close terminates the DHT node and socket.
func (d *DHT) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	close(d.doneChan)
	d.mu.Unlock()
	return d.conn.Close()
}
