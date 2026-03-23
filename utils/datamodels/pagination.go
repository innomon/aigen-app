package datamodels

type Pagination struct {
	Offset *string `json:"offset"`
	Limit  *string `json:"limit"`
}

type ValidPagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
