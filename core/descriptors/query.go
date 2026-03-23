package descriptors

import (
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type Variable struct {
	Name       string `json:"name"`
	IsRequired bool   `json:"isRequired"`
}

type Query struct {
	Name       string                `json:"name"`
	EntityName string                `json:"entityName"`
	Source     string                `json:"source"`
	Filters    []datamodels.Filter   `json:"filters"`
	Sorts      []datamodels.Sort     `json:"sorts"`
	Variables  []Variable            `json:"variables"`
	Distinct   bool                  `json:"distinct"`
	IdeUrl     string                `json:"ideUrl"`
	Pagination *datamodels.Pagination `json:"pagination"`
}
