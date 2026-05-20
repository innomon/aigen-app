package plugins

import (
	"time"
)

// PluginManifest represents the content of a plugin's metadata.
// It is typically derived from the JAR's MANIFEST.MF and a plugin-specific metadata file.
type PluginManifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	EntryPoints map[string]string `json:"entry_points,omitempty"` // e.g. "agentic": "agentic/agentic.yaml"
	Permissions []string          `json:"permissions"`          // e.g. ["access:bizdef:crm", "storage:write"]
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
