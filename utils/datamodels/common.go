package datamodels

type StrArgs map[string][]string

type ChunkStatus struct {
	Path        string `json:"path"`
	ChunkCount int    `json:"chunkCount"`
}

type UploadSession struct {
	UserId   string `json:"userId"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	Path     string `json:"path"`
}
