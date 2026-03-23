package datamodels

const (
	SortOrderAsc  = "Asc"
	SortOrderDesc = "Desc"
)

type Sort struct {
	Field string `json:"field"`
	Order string `json:"order"`
}
