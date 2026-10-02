package filestore

import (
	"context"
	"fmt"
)

type Config struct {
	Driver string
	FS     struct {
		PathPrefix string
		UrlPrefix  string
	}
	S3 struct {
		Bucket          string
		Region          string
		AccessKeyID     string
		SecretAccessKey string
		Endpoint        string
	}
	GCS struct {
		Bucket          string
		CredentialsFile string
	}
	Postgres struct {
		URL string
	}
	SurrealDB struct {
		URL string
	}
	SqliteP2P struct {
		URL       string
		UrlPrefix string
	}
}

func CreateFileStore(ctx context.Context, cfg Config) (IFileStore, error) {
	switch cfg.Driver {
	case "sqlite-p2p", "p2p", "sqlite", "":
		url := cfg.SqliteP2P.URL
		if url == "" && cfg.FS.PathPrefix != "" {
			url = cfg.FS.PathPrefix
		}
		return NewSqliteP2PFileStore(url, cfg.SqliteP2P.UrlPrefix)
	case "fs", "local":
		return NewLocalFileStore(cfg.FS.PathPrefix, cfg.FS.UrlPrefix), nil
	case "s3":
		return NewS3FileStore(ctx, cfg.S3.Bucket, cfg.S3.Region, cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, cfg.S3.Endpoint)
	case "gcs":
		return NewGCSFileStore(ctx, cfg.GCS.Bucket, cfg.GCS.CredentialsFile)
	case "postgres":
		return NewPostgresFileStore(cfg.Postgres.URL)
	case "surrealdb":
		return NewSurrealFileStore(ctx, cfg.SurrealDB.URL)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.Driver)
	}
}
