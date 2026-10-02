package hyperswarm

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"go-pear/pkg/hyperdht"
	"go-pear/pkg/secretstream"
	"go-pear/pkg/udx"
)

// SwarmOptions configures a Hyperswarm instance.
type SwarmOptions struct {
	KeyPair          *secretstream.KeyPair
	Port             int
	Bootstrap        []string
	Authorizer       func(remotePK [32]byte) bool
	HandshakeTimeout time.Duration
}

// Swarm coordinates P2P swarming, DHT topic announcements, and encrypted connection pooling.
type Swarm struct {
	mu               sync.RWMutex
	KeyPair          *secretstream.KeyPair
	dht              *hyperdht.DHT
	udxSocket        *udx.Socket
	connections      *ConnectionSet
	discovery        map[[32]byte]*PeerDiscovery
	pendingDials     map[[32]byte]bool
	retryTimer       *RetryTimer
	authorizer       func(remotePK [32]byte) bool
	handshakeTimeout time.Duration

	listener     net.Listener
	connHandlers []func(conn net.Conn, peer *PeerConnection)

	closed   bool
	doneChan chan struct{}
}

// New creates and initializes a Hyperswarm instance.
func New(opts SwarmOptions) (*Swarm, error) {
	kp := opts.KeyPair
	if kp == nil {
		var err error
		kp, err = secretstream.GenerateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate swarm keypair: %w", err)
		}
	}

	// Listen on TCP port for incoming peer connections
	addr := fmt.Sprintf("0.0.0.0:%d", opts.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start swarm TCP listener on %s: %w", addr, err)
	}
	assignedPort := listener.Addr().(*net.TCPAddr).Port

	// Start HyperDHT
	dht, err := hyperdht.NewDHT(hyperdht.Config{
		ID:        kp.Public,
		PublicKey: kp.Public,
		Port:      assignedPort,
		Bootstrap: opts.Bootstrap,
	})
	if err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("failed to initialize HyperDHT: %w", err)
	}

	// Start UDX UDP socket on auto port
	udxSock, _ := udx.Listen("0.0.0.0:0")

	hsTimeout := opts.HandshakeTimeout
	if hsTimeout <= 0 {
		hsTimeout = secretstream.DefaultHandshakeTimeout
	}

	s := &Swarm{
		KeyPair:          kp,
		dht:              dht,
		udxSocket:        udxSock,
		connections:      NewConnectionSet(),
		discovery:        make(map[[32]byte]*PeerDiscovery),
		pendingDials:     make(map[[32]byte]bool),
		retryTimer:       NewRetryTimer(),
		authorizer:       opts.Authorizer,
		handshakeTimeout: hsTimeout,
		listener:         listener,
		connHandlers:     make([]func(conn net.Conn, peer *PeerConnection), 0),
		doneChan:         make(chan struct{}),
	}

	go s.serverAcceptLoop()
	return s, nil
}

// SetAuthorizer updates the peer authorizer hook.
func (s *Swarm) SetAuthorizer(auth func(remotePK [32]byte) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authorizer = auth
}

func (s *Swarm) isPeerAllowed(remotePK [32]byte) bool {
	s.mu.RLock()
	auth := s.authorizer
	s.mu.RUnlock()
	if auth == nil {
		return true
	}
	return auth(remotePK)
}

// Disconnect terminates any active connection to the specified peer public key.
func (s *Swarm) Disconnect(remotePK [32]byte) error {
	conn, ok := s.connections.Get(remotePK)
	if !ok {
		return nil
	}
	s.connections.Remove(remotePK)
	return conn.Conn.Close()
}

// Port returns the active TCP/DHT listening port.
func (s *Swarm) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// DHT returns the underlying HyperDHT instance.
func (s *Swarm) DHT() *hyperdht.DHT {
	return s.dht
}

// UDX returns the underlying UDX socket instance.
func (s *Swarm) UDX() *udx.Socket {
	return s.udxSocket
}

// OnConnection registers a callback for new active encrypted peer connections.
func (s *Swarm) OnConnection(fn func(conn net.Conn, peer *PeerConnection)) {
	s.mu.Lock()
	s.connHandlers = append(s.connHandlers, fn)
	s.mu.Unlock()
}

// Join begins announcing and/or looking up peers for a 32-byte topic.
func (s *Swarm) Join(topic [32]byte, opts DiscoveryOptions) (*PeerDiscovery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("swarm is closed")
	}

	if existing, ok := s.discovery[topic]; ok {
		existing.Options = opts
		return existing, nil
	}

	pd := NewPeerDiscovery(s, topic, opts)
	s.discovery[topic] = pd
	return pd, nil
}

// Leave stops discovery and unannounces a topic.
func (s *Swarm) Leave(topic [32]byte) error {
	s.mu.Lock()
	pd, ok := s.discovery[topic]
	if ok {
		delete(s.discovery, topic)
	}
	s.mu.Unlock()

	if ok && pd != nil {
		pd.Destroy()
	}
	return nil
}

// Flush waits for all active discovery lookups to complete an initial cycle.
func (s *Swarm) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s.mu.RLock()
	var sessions []*PeerDiscovery
	for _, pd := range s.discovery {
		sessions = append(sessions, pd)
	}
	s.mu.RUnlock()

	for _, pd := range sessions {
		pd.refresh(ctx)
	}
	return nil
}

// Connections returns all active peer connections.
func (s *Swarm) Connections() []*PeerConnection {
	return s.connections.List()
}

func (s *Swarm) serverAcceptLoop() {
	for {
		rawConn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.doneChan:
				return
			default:
				continue
			}
		}

		go s.handleInbound(rawConn)
	}
}

func (s *Swarm) handleInbound(raw net.Conn) {
	secConn, err := secretstream.UpgradeWithTimeout(raw, s.KeyPair, false, s.handshakeTimeout)
	if err != nil {
		_ = raw.Close()
		return
	}

	peerPK := secConn.RemotePublicKey()
	if !s.isPeerAllowed(peerPK) {
		_ = secConn.Close()
		return
	}

	peerConn := &PeerConnection{
		RemotePublicKey: peerPK,
		Conn:            secConn,
		IsInitiator:     false,
		CreatedAt:       time.Now(),
	}

	if err := s.connections.Add(peerConn); err != nil {
		_ = secConn.Close()
		return
	}

	s.retryTimer.Reset(peerPK)
	s.dispatchConnection(secConn, peerConn)
}

// ConnectDirect establishes an outgoing encrypted peer connection to a target address.
func (s *Swarm) ConnectDirect(ip string, port int, topic [32]byte) (*PeerConnection, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, fmt.Errorf("swarm is closed")
	}
	s.mu.Unlock()

	targetAddr := net.JoinHostPort(ip, strconv.Itoa(port))
	rawConn, err := net.DialTimeout("tcp", targetAddr, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("direct dial failed: %w", err)
	}

	secConn, err := secretstream.UpgradeWithTimeout(rawConn, s.KeyPair, true, s.handshakeTimeout)
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("secretstream upgrade failed: %w", err)
	}

	peerPK := secConn.RemotePublicKey()
	if !s.isPeerAllowed(peerPK) {
		_ = secConn.Close()
		return nil, fmt.Errorf("connection rejected by peer authorizer")
	}

	peerConn := &PeerConnection{
		RemotePublicKey: peerPK,
		Conn:            secConn,
		IsInitiator:     true,
		Topic:           topic,
		CreatedAt:       time.Now(),
	}

	if err := s.connections.Add(peerConn); err != nil {
		_ = secConn.Close()
		return nil, fmt.Errorf("failed to register connection: %w", err)
	}

	s.retryTimer.Reset(peerPK)
	s.dispatchConnection(secConn, peerConn)
	return peerConn, nil
}

func (s *Swarm) maybeConnect(peer *hyperdht.PeerEndpoint, topic [32]byte) {
	if !s.isPeerAllowed(peer.PublicKey) {
		return
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	if _, exists := s.connections.Get(peer.PublicKey); exists {
		s.mu.Unlock()
		return
	}
	if s.pendingDials[peer.PublicKey] {
		s.mu.Unlock()
		return
	}
	s.pendingDials[peer.PublicKey] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pendingDials, peer.PublicKey)
		s.mu.Unlock()
	}()

	ipStr := peer.Addr.IP.String()
	if peer.Addr.IP == nil || peer.Addr.IP.IsUnspecified() {
		ipStr = "127.0.0.1"
	}
	targetAddr := net.JoinHostPort(ipStr, strconv.Itoa(peer.Addr.Port))
	rawConn, err := net.DialTimeout("tcp", targetAddr, 2*time.Second)
	if err != nil {
		return
	}

	secConn, err := secretstream.UpgradeWithTimeout(rawConn, s.KeyPair, true, s.handshakeTimeout)
	if err != nil {
		_ = rawConn.Close()
		return
	}

	peerPK := secConn.RemotePublicKey()
	if !s.isPeerAllowed(peerPK) {
		_ = secConn.Close()
		return
	}

	peerConn := &PeerConnection{
		RemotePublicKey: peerPK,
		Conn:            secConn,
		IsInitiator:     true,
		Topic:           topic,
		CreatedAt:       time.Now(),
	}

	if err := s.connections.Add(peerConn); err != nil {
		_ = secConn.Close()
		return
	}

	s.retryTimer.Reset(peerPK)
	s.dispatchConnection(secConn, peerConn)
}

func (s *Swarm) dispatchConnection(conn net.Conn, peer *PeerConnection) {
	s.mu.RLock()
	handlers := make([]func(conn net.Conn, peer *PeerConnection), len(s.connHandlers))
	copy(handlers, s.connHandlers)
	s.mu.RUnlock()

	for _, h := range handlers {
		h(conn, peer)
	}
}

// Close gracefully terminates all discovery sessions, active connections, and DHT node.
func (s *Swarm) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.doneChan)
	s.mu.Unlock()

	for _, pd := range s.discovery {
		pd.Destroy()
	}

	s.connections.Close()
	_ = s.listener.Close()
	if s.udxSocket != nil {
		_ = s.udxSocket.Close()
	}
	if s.dht != nil {
		_ = s.dht.Close()
	}
	return nil
}
