package plugins

import (
	"time"
)

// PluginManifest represents the content of a plugin's metadata.
// It is typically derived from the JAR's MANIFEST.MF and a plugin-specific metadata file.
type PluginManifest struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	Description string                  `json:"description"`
	Author      string                  `json:"author"`
	EntryPoints map[string]string       `json:"entry_points,omitempty"` // e.g. "agentic": "agentic/agentic.yaml"
	Permissions []PermissionRequirement `json:"permissions"`          // e.g. [{"type": "http", "value": "*.openai.com"}]
	EnvVars     []string                `json:"env_vars"`             // List of required secret keys
}

type PermissionRequirement struct {
	Type  string `json:"type"`  // e.g. "http", "bizdef", "filesystem"
	Value string `json:"value"` // e.g. "*.google.com", "crm", "/tmp/*"
}

type PermissionGrant struct {
	PluginID  string    `json:"plugin_id"`
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	GrantedBy string    `json:"granted_by"`
	GrantedAt time.Time `json:"granted_at"`
}

type VaultEntry struct {
	PluginID  string    `json:"plugin_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"` // Should be encrypted in a real vault
	UpdatedAt time.Time `json:"updated_at"`
}

// PluginState represents the current status of a plugin in the system.
type PluginStatus string

const (
	StatusPending    PluginStatus = "pending"    // Discovered but not verified/trusted
	StatusActive     PluginStatus = "active"     // Verified, trusted, and mounted
	StatusInactive   PluginStatus = "inactive"   // Explicitly disabled by admin
	StatusError      PluginStatus = "error"      // Failed verification or loading
	StatusUntrusted  PluginStatus = "untrusted"  // Failed signature check
)

// PluginInfo holds the runtime information about a discovered plugin.
type PluginInfo struct {
	Manifest    PluginManifest `json:"manifest"`
	Path        string         `json:"path"`
	Status      PluginStatus   `json:"status"`
	Error       string         `json:"error,omitempty"`
	Signer      string         `json:"signer,omitempty"`       // Common Name or fingerprint of the signer
	IsVerified  bool           `json:"is_verified"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
