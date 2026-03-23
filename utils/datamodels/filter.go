package datamodels

type Constraint struct {
	Match  string    `json:"match"`
	Values []*string `json:"values"`
}

const (
	MatchAll = "matchAll"
	MatchAny = "matchAny"
)

type Filter struct {
	FieldName   string       `json:"fieldName"`
	MatchType   string       `json:"matchType"`
	Constraints []Constraint `json:"constraints"`
}
