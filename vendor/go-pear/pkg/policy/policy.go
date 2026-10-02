package policy

import (
	"fmt"
	"strings"
	"sync"
)

// Mode represents the replication access control mode.
type Mode string

const (
	// ModeAllAllowed permits replication with all peers without restriction.
	ModeAllAllowed Mode = "all"
	// ModeWhitelist permits replication only with explicitly whitelisted peers.
	ModeWhitelist Mode = "whitelist"
	// ModeBlacklist permits replication with all peers except blacklisted ones.
	ModeBlacklist Mode = "blacklist"
)

// NormalizeMode validates and normalizes a mode string.
func NormalizeMode(m string) (Mode, error) {
	clean := strings.ToLower(strings.TrimSpace(m))
	switch clean {
	case "", string(ModeAllAllowed):
		return ModeAllAllowed, nil
	case string(ModeWhitelist), "allowlist":
		return ModeWhitelist, nil
	case string(ModeBlacklist), "denylist":
		return ModeBlacklist, nil
	default:
		return "", fmt.Errorf("invalid replication mode %q: must be 'all', 'whitelist', or 'blacklist'", m)
	}
}

// ReplicationPolicy defines thread-safe peer replication access rules.
type ReplicationPolicy struct {
	mu        sync.RWMutex
	mode      Mode
	whitelist map[[32]byte]struct{}
	blacklist map[[32]byte]struct{}
}

// New creates an initialized ReplicationPolicy with the given mode.
func New(mode Mode) *ReplicationPolicy {
	if mode == "" {
		mode = ModeAllAllowed
	}
	return &ReplicationPolicy{
		mode:      mode,
		whitelist: make(map[[32]byte]struct{}),
		blacklist: make(map[[32]byte]struct{}),
	}
}

// Mode returns the active replication mode.
func (p *ReplicationPolicy) Mode() Mode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// SetMode updates the active replication mode. Returns an error if the mode is invalid.
func (p *ReplicationPolicy) SetMode(m Mode) error {
	norm, err := NormalizeMode(string(m))
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = norm
	return nil
}

// IsAllowed evaluates whether the peer public key is authorized to replicate.
// Deny-overrides-allow precedence is strictly enforced: if a key is blacklisted,
// it is denied replication even if it appears in the whitelist or in ModeAllAllowed.
func (p *ReplicationPolicy) IsAllowed(nodePK [32]byte) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Enforce deny-overrides-allow precedence:
	if _, blocked := p.blacklist[nodePK]; blocked {
		return false
	}

	switch p.mode {
	case ModeWhitelist:
		_, ok := p.whitelist[nodePK]
		return ok
	case ModeBlacklist:
		return true
	case ModeAllAllowed:
		return true
	default:
		return false
	}
}

// AddWhitelist adds a peer public key to the whitelist.
func (p *ReplicationPolicy) AddWhitelist(key [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.whitelist[key] = struct{}{}
}

// RemoveWhitelist removes a peer public key from the whitelist.
func (p *ReplicationPolicy) RemoveWhitelist(key [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.whitelist, key)
}

// AddBlacklist adds a peer public key to the blacklist.
func (p *ReplicationPolicy) AddBlacklist(key [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blacklist[key] = struct{}{}
}

// RemoveBlacklist removes a peer public key from the blacklist.
func (p *ReplicationPolicy) RemoveBlacklist(key [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.blacklist, key)
}

// Clear clears all whitelist and blacklist entries.
func (p *ReplicationPolicy) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.whitelist = make(map[[32]byte]struct{})
	p.blacklist = make(map[[32]byte]struct{})
}

// WhitelistKeys returns a slice of all whitelisted public keys.
func (p *ReplicationPolicy) WhitelistKeys() [][32]byte {
	p.mu.RLock()
	defer p.mu.RUnlock()
	keys := make([][32]byte, 0, len(p.whitelist))
	for k := range p.whitelist {
		keys = append(keys, k)
	}
	return keys
}

// BlacklistKeys returns a slice of all blacklisted public keys.
func (p *ReplicationPolicy) BlacklistKeys() [][32]byte {
	p.mu.RLock()
	defer p.mu.RUnlock()
	keys := make([][32]byte, 0, len(p.blacklist))
	for k := range p.blacklist {
		keys = append(keys, k)
	}
	return keys
}
