package datamodels

import "time"

// RecJSON JSON record
type RecJSON struct {
	Namespace string       `json:"namespace"`
	Key       string       `json:"key"`
	Rec       interface{}  `json:"rec"`
	MetaData  interface{}  `json:"metadata"`
	Tmstamp   time.Time    `json:"tmstamp"`
}
