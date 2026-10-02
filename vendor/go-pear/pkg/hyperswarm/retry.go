package hyperswarm

import (
	"math/rand"
	"sync"
	"time"
)

var defaultBackoffs = []time.Duration{
	250 * time.Millisecond,
	500 * time.Millisecond,
	1000 * time.Millisecond,
	2000 * time.Millisecond,
	4000 * time.Millisecond,
	8000 * time.Millisecond,
}

// RetryTimer coordinates exponential backoff and randomized jitter for reconnecting to peers.
type RetryTimer struct {
	mu       sync.Mutex
	backoffs []time.Duration
	retries  map[[32]byte]int
}

// NewRetryTimer creates a new retry timer with default backoff schedule.
func NewRetryTimer() *RetryTimer {
	return &RetryTimer{
		backoffs: defaultBackoffs,
		retries:  make(map[[32]byte]int),
	}
}

// NextDelay returns the backoff duration for the next retry attempt of a peer.
func (rt *RetryTimer) NextDelay(peerID [32]byte) time.Duration {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	attempt := rt.retries[peerID]
	rt.retries[peerID] = attempt + 1

	idx := attempt
	if idx >= len(rt.backoffs) {
		idx = len(rt.backoffs) - 1
	}

	base := rt.backoffs[idx]
	// Add jitter +/- 20%
	jitterFrac := 0.8 + rand.Float64()*0.4
	return time.Duration(float64(base) * jitterFrac)
}

// Reset resets the retry counter for a peer after a successful connection.
func (rt *RetryTimer) Reset(peerID [32]byte) {
	rt.mu.Lock()
	delete(rt.retries, peerID)
	rt.mu.Unlock()
}
