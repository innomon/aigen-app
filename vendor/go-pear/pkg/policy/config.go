package policy

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Config represents the serialized configuration format for replication gating.
type Config struct {
	Mode      string   `json:"mode"`
	Whitelist []string `json:"whitelist,omitempty"`
	Blacklist []string `json:"blacklist,omitempty"`
}

// ParseHexKey validates and decodes a 64-character hexadecimal public key string into a [32]byte array.
func ParseHexKey(s string) ([32]byte, error) {
	var key [32]byte
	clean := strings.TrimSpace(s)
	clean = strings.TrimPrefix(clean, "0x")
	if len(clean) != 64 {
		return key, fmt.Errorf("invalid public key length %d: expected 64 hex characters", len(clean))
	}

	decoded, err := hex.DecodeString(clean)
	if err != nil {
		return key, fmt.Errorf("invalid hex encoding for public key %q: %w", s, err)
	}

	copy(key[:], decoded)
	return key, nil
}

// LoadConfig deserializes a JSON configuration stream into a Config struct.
func LoadConfig(r io.Reader) (*Config, error) {
	var cfg Config
	dec := json.NewDecoder(r)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode replication config JSON: %w", err)
	}
	return &cfg, nil
}

// LoadConfigFile reads and parses a JSON configuration file.
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read replication config file %q: %w", path, err)
	}

	var root struct {
		Replication *Config `json:"replication"`
		*Config
	}

	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse replication config file %q: %w", path, err)
	}

	if root.Replication != nil {
		return root.Replication, nil
	}
	if root.Config != nil && (root.Config.Mode != "" || len(root.Config.Whitelist) > 0 || len(root.Config.Blacklist) > 0) {
		return root.Config, nil
	}

	return &Config{Mode: string(ModeAllAllowed)}, nil
}

// NewFromConfig builds a ReplicationPolicy instance from a Config struct.
func NewFromConfig(cfg *Config) (*ReplicationPolicy, error) {
	if cfg == nil {
		return New(ModeAllAllowed), nil
	}

	mode, err := NormalizeMode(cfg.Mode)
	if err != nil {
		return nil, err
	}

	pol := New(mode)
	if err := pol.ApplyConfig(cfg); err != nil {
		return nil, err
	}
	return pol, nil
}

// ApplyConfig updates the policy atomically with rules from Config.
func (p *ReplicationPolicy) ApplyConfig(cfg *Config) error {
	if cfg == nil {
		return nil
	}

	mode, err := NormalizeMode(cfg.Mode)
	if err != nil {
		return err
	}

	whitelists := make(map[[32]byte]struct{})
	for _, raw := range cfg.Whitelist {
		k, err := ParseHexKey(raw)
		if err != nil {
			return fmt.Errorf("whitelist error: %w", err)
		}
		whitelists[k] = struct{}{}
	}

	blacklists := make(map[[32]byte]struct{})
	for _, raw := range cfg.Blacklist {
		k, err := ParseHexKey(raw)
		if err != nil {
			return fmt.Errorf("blacklist error: %w", err)
		}
		// Deny-overrides-allow: blacklist always takes precedence
		delete(whitelists, k)
		blacklists[k] = struct{}{}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode
	p.whitelist = whitelists
	p.blacklist = blacklists
	return nil
}
