package descriptors

import "time"

// EvolutionAction represents a single transformation action
type EvolutionAction struct {
	Action  string      `json:"action"` // rename, add, drop, transform
	From    string      `json:"from,omitempty"`
	To      string      `json:"to,omitempty"`
	Field   string      `json:"field,omitempty"`
	Default interface{} `json:"default,omitempty"`
}

// EntityVersion represents a specific version in the entity's timeline
type EntityVersion struct {
	Date        time.Time         `json:"date"`
	Description string            `json:"description"`
	Actions     []EvolutionAction `json:"actions"`
}

// EvolutionManifest represents the full timeline for all entities in a BizDef
// Key is the Entity name
type EvolutionManifest map[string]map[string]EntityVersion
