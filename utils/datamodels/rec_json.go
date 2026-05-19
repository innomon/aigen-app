package datamodels

import "time"

// MetaData contains record-level information including security and evolution state
type MetaData struct {
	Roles             []string               `json:"roles,omitempty"`
	Owner             interface{}            `json:"owner,omitempty"`
	SchemaVersion     string                 `json:"schema_version,omitempty"`
	SchemaVersionDate string                 `json:"schema_version_date,omitempty"`
	Revision          int64                  `json:"revision,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

// RecJSON JSON record
type RecJSON struct {
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	Rec       interface{} `json:"rec"`
	MetaData  MetaData  `json:"metadata"`
	Tmstamp   time.Time `json:"tmstamp"`
}
