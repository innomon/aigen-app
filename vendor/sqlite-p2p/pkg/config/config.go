package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go-pear/pkg/policy"
)

// NodeConfig defines the comprehensive node and replication configuration.
type NodeConfig struct {
	NodeID       string         `json:"node_id" yaml:"node_id"`
	SwarmTopic   string         `json:"swarm_topic" yaml:"swarm_topic"`
	SwarmPort    int            `json:"swarm_port" yaml:"swarm_port"`
	Bootstrap    []string       `json:"bootstrap" yaml:"bootstrap"`
	PeerAddrs    []string       `json:"peer_addrs" yaml:"peer_addrs"`
	DBPath       string         `json:"db_path" yaml:"db_path"`
	EnableWAL    bool           `json:"enable_wal" yaml:"enable_wal"`
	EnableCrypto bool           `json:"enable_crypto" yaml:"enable_crypto"`
	AutoSync     bool           `json:"auto_sync" yaml:"auto_sync"`
	Replication  *policy.Config `json:"replication,omitempty" yaml:"replication,omitempty"`
}

// LoadConfig parses a NodeConfig struct from an io.Reader stream containing JSON.
func LoadConfig(r io.Reader) (*NodeConfig, error) {
	var cfg NodeConfig
	dec := json.NewDecoder(r)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode node config JSON: %w", err)
	}
	return &cfg, nil
}

// LoadConfigFile reads and parses a JSON configuration file into a NodeConfig struct.
func LoadConfigFile(path string) (*NodeConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open node config file %q: %w", path, err)
	}
	defer f.Close()

	return LoadConfig(f)
}

// BuildPolicy constructs a go-pear policy.ReplicationPolicy from the NodeConfig.
func (c *NodeConfig) BuildPolicy() (*policy.ReplicationPolicy, error) {
	if c == nil || c.Replication == nil {
		return policy.New(policy.ModeAllAllowed), nil
	}
	return policy.NewFromConfig(c.Replication)
}
