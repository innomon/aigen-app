package descriptors

import (
	"time"
)

type DatabaseProvider string

const (
	Postgres  DatabaseProvider = "Postgres"
	MySQL     DatabaseProvider = "MySQL"
	SQLite    DatabaseProvider = "SQLite"
	SQLServer DatabaseProvider = "SQLServer"
)

type ImageCompressionOptions struct {
	MaxWidth int `json:"maxWidth"`
	Quality  int `json:"quality"`
}

type RouteOptions struct {
	ApiBaseUrl  string `json:"apiBaseUrl"`
	PageBaseUrl string `json:"pageBaseUrl"`
}

type LocalFileStoreOptions struct {
	PathPrefix string `json:"pathPrefix"`
	UrlPrefix  string `json:"urlPrefix"`
}

type SystemSettings struct {
	MapCmsHomePage         bool                    `json:"mapCmsHomePage"`
	GraphQlPath            string                  `json:"graphQlPath"`
	EntitySchemaExpiration time.Duration           `json:"entitySchemaExpiration"`
	PageSchemaExpiration   time.Duration           `json:"pageSchemaExpiration"`
	QuerySchemaExpiration  time.Duration           `json:"querySchemaExpiration"`
	ImageCompression       ImageCompressionOptions `json:"imageCompression"`
	RouteOptions           RouteOptions            `json:"routeOptions"`
	DatabaseProvider       DatabaseProvider        `json:"databaseProvider"`
	ReplicaCount           int                     `json:"replicaCount"`
	KnownPaths             []string                `json:"knownPaths"`
	LocalFileStoreOptions  LocalFileStoreOptions   `json:"localFileStoreOptions"`
	FileSignature          map[string][][]byte     `json:"fileSignature"`
}

func DefaultSystemSettings() *SystemSettings {
	return &SystemSettings{
		MapCmsHomePage:         true,
		GraphQlPath:            "/graph",
		EntitySchemaExpiration: time.Minute,
		PageSchemaExpiration:   time.Minute,
		QuerySchemaExpiration:  time.Minute,
		ImageCompression: ImageCompressionOptions{
			MaxWidth: 1200,
			Quality:  75,
		},
		RouteOptions: RouteOptions{
			ApiBaseUrl:  "/api",
			PageBaseUrl: "",
		},
		KnownPaths: []string{"api", "files"},
		LocalFileStoreOptions: LocalFileStoreOptions{
			PathPrefix: "wwwroot/files", // Default value, will be overridden or kept if WWWRoot is "wwwroot"
			UrlPrefix:  "/files",
		},
		FileSignature: map[string][][]byte{
			".gif":  {[]byte("GIF8")},
			".png":  {[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}},
			".jpeg": {[]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte{0xFF, 0xD8, 0xFF, 0xE2}, []byte{0xFF, 0xD8, 0xFF, 0xE3}},
			".jpg":  {[]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte{0xFF, 0xD8, 0xFF, 0xE1}, []byte{0xFF, 0xD8, 0xFF, 0xE8}},
			".zip":  {[]byte{0x50, 0x4B, 0x03, 0x04}, []byte("PKLITE"), []byte("PKSpX"), []byte{0x50, 0x4B, 0x05, 0x06}, []byte{0x50, 0x4B, 0x07, 0x08}, []byte("WinZip")},
			".mp4":  {[]byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, []byte("\x00\x00\x00 ftyp")},
			".mpeg": {[]byte{0x00, 0x00, 0x01, 0xBA}},
			".mpg":  {[]byte{0x00, 0x00, 0x01, 0xBA}},
		},
	}
}
