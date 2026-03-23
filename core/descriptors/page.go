package descriptors

import (
	"encoding/json"
)

type SelectedQuery struct {
	FieldName   string            `json:"fieldName"`
	QueryName   string            `json:"queryName"`
	Type        string            `json:"type"` // list or single
	Args        map[string]string `json:"args"` // Values: 'fromPath' | 'fromQuery'
	Description *string           `json:"description"`
}

type PageArchitecture struct {
	Sections              *json.RawMessage `json:"sections"`
	SelectedQueries       []SelectedQuery  `json:"selectedQueries"`
	ArchitectureHints     *string          `json:"architectureHints"`
	ComponentInstructions *json.RawMessage `json:"componentInstructions"`
}

type PageMetadata struct {
	Architecture     *PageArchitecture `json:"architecture"`
	EnableVisitTrack *bool             `json:"enableVisitTrack"`
	Components       *json.RawMessage  `json:"components"`
	UserInput        *string           `json:"userInput"`
	TemplateId       *string           `json:"templateId"`
	CustomHeader     *string           `json:"customHeader"`
}

type Page struct {
	Name       string        `json:"name"`
	Title      string        `json:"title"`
	Html       string        `json:"html"`
	EntityName *string       `json:"entityName"`
	PageType   *string       `json:"pageType"`
	Metadata   *PageMetadata `json:"metadata"`
}
