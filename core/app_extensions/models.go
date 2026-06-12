package app_extensions

import (
	"encoding/json"
	"strings"
	"time"
)

// AppExtensionManifest represents the content of an app extension's metadata.
// It is typically derived from the JAR's MANIFEST.MF and a metadata file.
type AppExtensionManifest struct {
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

func (p *PermissionRequirement) UnmarshalJSON(data []byte) error {
	type Alias PermissionRequirement
	var obj Alias
	if err := json.Unmarshal(data, &obj); err == nil {
		*p = PermissionRequirement(obj)
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	parts := strings.SplitN(str, ":", 2)
	if len(parts) == 2 {
		p.Type = parts[0]
		p.Value = parts[1]
	} else {
		p.Type = "unknown"
		p.Value = str
	}
	return nil
}

type PermissionGrant struct {
	ExtensionID string    `json:"extension_id"`
	Type        string    `json:"type"`
	Value       string    `json:"value"`
	GrantedBy   string    `json:"granted_by"`
	GrantedAt   time.Time `json:"granted_at"`
}

type VaultEntry struct {
	ExtensionID string    `json:"extension_id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"` // Should be encrypted in a real vault
	UpdatedAt   time.Time `json:"updated_at"`
}

// AppExtensionStatus represents the current status of an extension in the system.
type AppExtensionStatus string

const (
	StatusPending   AppExtensionStatus = "pending"   // Discovered but not verified/trusted
	StatusActive    AppExtensionStatus = "active"    // Verified, trusted, and mounted
	StatusInactive  AppExtensionStatus = "inactive"  // Explicitly disabled by admin
	StatusError     AppExtensionStatus = "error"     // Failed verification or loading
	StatusUntrusted AppExtensionStatus = "untrusted" // Failed signature check
)

// AppExtensionInfo holds the runtime information about a discovered app extension.
type AppExtensionInfo struct {
	Manifest   AppExtensionManifest `json:"manifest"`
	Path       string               `json:"path"`
	Status     AppExtensionStatus   `json:"status"`
	Error      string               `json:"error,omitempty"`
	Signer     string               `json:"signer,omitempty"` // Common Name or fingerprint of the signer
	IsVerified bool                 `json:"is_verified"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}
