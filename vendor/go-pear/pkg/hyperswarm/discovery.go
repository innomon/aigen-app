package hyperswarm

import (
	"context"
	"sync"
	"time"
)

// DiscoveryOptions specifies client/server modes for joining a topic.
type DiscoveryOptions struct {
	Server bool
	Client bool
}

// PeerDiscovery coordinates background DHT lookups and announcements for a specific 32-byte topic.
type PeerDiscovery struct {
	mu      sync.RWMutex
	swarm   *Swarm
	Topic   [32]byte
	Options DiscoveryOptions

	active   bool
	cancelFn context.CancelFunc
	doneChan chan struct{}
}

// NewPeerDiscovery starts a discovery session for the given topic.
func NewPeerDiscovery(s *Swarm, topic [32]byte, opts DiscoveryOptions) *PeerDiscovery {
	ctx, cancel := context.WithCancel(context.Background())
	pd := &PeerDiscovery{
		swarm:    s,
		Topic:    topic,
		Options:  opts,
		active:   true,
		cancelFn: cancel,
		doneChan: make(chan struct{}),
	}

	go pd.loop(ctx)
	return pd
}

func (pd *PeerDiscovery) loop(ctx context.Context) {
	defer close(pd.doneChan)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Initial run
	pd.refresh(ctx)

	for {
		select {
		case <-ctx.Done():
			// Unannounce if server
			if pd.Options.Server && pd.swarm.dht != nil {
				unCtx, unCancel := context.WithTimeout(context.Background(), 1*time.Second)
				_ = pd.swarm.dht.Unannounce(unCtx, pd.Topic)
				unCancel()
			}
			return
		case <-ticker.C:
			pd.refresh(ctx)
		}
	}
}

func (pd *PeerDiscovery) refresh(ctx context.Context) {
	if pd.swarm.dht == nil {
		return
	}

	if pd.Options.Server {
		serverPort := pd.swarm.Port()
		_ = pd.swarm.dht.Announce(ctx, pd.Topic, serverPort)
	}

	if pd.Options.Client {
		peers, err := pd.swarm.dht.Lookup(ctx, pd.Topic)
		if err == nil {
			for _, peer := range peers {
				if peer.ID != pd.swarm.dht.ID {
					go pd.swarm.maybeConnect(peer, pd.Topic)
				}
			}
		}
	}
}

// Destroy terminates the discovery session.
func (pd *PeerDiscovery) Destroy() {
	pd.mu.Lock()
	if !pd.active {
		pd.mu.Unlock()
		return
	}
	pd.active = false
	pd.cancelFn()
	pd.mu.Unlock()
	<-pd.doneChan
}
