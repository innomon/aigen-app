package framework

import (
	"fmt"
	"os"

	"github.com/innomon/aigen-app/core/descriptors"
	yaml "gopkg.in/yaml.v3"
)

type S3Config struct {
	Bucket          string `yaml:"bucket" json:"bucket"`
	Region          string `yaml:"region" json:"region"`
	AccessKeyID     string `yaml:"access_key_id,omitempty" json:"access_key_id,omitempty"`
	SecretAccessKey string `yaml:"secret_access_key,omitempty" json:"secret_access_key,omitempty"`
	Endpoint        string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
}

type FSConfig struct {
	Root      string `yaml:"root" json:"root"`
	UrlPrefix string `yaml:"url_prefix" json:"url_prefix"`
}

type GCSConfig struct {
	Bucket          string `yaml:"bucket" json:"bucket"`
	CredentialsFile string `yaml:"credentials_file,omitempty" json:"credentials_file,omitempty"`
}

type PostgresStorageConfig struct {
	URL string `yaml:"url" json:"url"`
}

type StorageConfig struct {
	Driver   string                `yaml:"driver" json:"driver"`
	FS       FSConfig              `yaml:"fs" json:"fs"`
	S3       S3Config              `yaml:"s3" json:"s3"`
	GCS      GCSConfig             `yaml:"gcs" json:"gcs"`
	Postgres PostgresStorageConfig `yaml:"postgres" json:"postgres"`
}

type Config struct {
	BizDefsDir        string                     `yaml:"bizdefs_dir" json:"bizdefs_dir"`
	WWWRoot           string                     `yaml:"www_root" json:"www_root"`
	DatabaseDSN       string                     `yaml:"database_dsn" json:"database_dsn"`
	Domain            string                     `yaml:"domain" json:"domain"`
	Port              string                     `yaml:"port" json:"port"`
	AgenticConfigPath string                     `yaml:"agentic_config_path" json:"agentic_config_path"`
	Channels          descriptors.ChannelsConfig `yaml:"channels" json:"channels"`
	MCP               descriptors.MCPConfig      `yaml:"mcp" json:"mcp"`
	PluginsDir        string                     `yaml:"plugins_dir" json:"plugins_dir"`
	Storage           StorageConfig              `yaml:"storage" json:"storage"`
	TemporaryAccess   []descriptors.TemporaryAccessConfig `yaml:"temporary_access" json:"temporary_access"`
}

func DefaultConfig() *Config {
	return &Config{
		BizDefsDir:        "bizdefs",
		PluginsDir:        "plugins",
		WWWRoot:           "wwwroot",
		DatabaseDSN:       "memory://",
		Port:              "5000",
		AgenticConfigPath: "agentic.yaml",
		Storage: StorageConfig{
			Driver: "fs",
			FS: FSConfig{
				Root:      "wwwroot/files",
				UrlPrefix: "/files",
			},
		},
		TemporaryAccess: []descriptors.TemporaryAccessConfig{
			{
				Path: "tmp",
				TTL:  300,
				Role: "admin",
			},
		},
	}
}


func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	if path == "" {
		if envPath := os.Getenv("FORMCMS_CONFIG_PATH"); envPath != "" {
			path = envPath
		}
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Environment variable overrides
	if pluginsDir := os.Getenv("FORMCMS_PLUGINS_DIR"); pluginsDir != "" {
		config.PluginsDir = pluginsDir
	}
	if bizdefsDir := os.Getenv("FORMCMS_BIZDEFS_DIR"); bizdefsDir != "" {
		config.BizDefsDir = bizdefsDir
	}
	if wwwRoot := os.Getenv("FORMCMS_WWW_ROOT"); wwwRoot != "" {
		config.WWWRoot = wwwRoot
	}
	if dbDSN := os.Getenv("FORMCMS_DB_DSN"); dbDSN != "" {
		config.DatabaseDSN = dbDSN
	}
	if domain := os.Getenv("DOMAIN"); domain != "" {
		config.Domain = domain
	}
	if port := os.Getenv("PORT"); port != "" {
		config.Port = port
	}
	if agenticPath := os.Getenv("FORMCMS_AGENTIC_CONFIG_PATH"); agenticPath != "" {
		config.AgenticConfigPath = agenticPath
	}

	return config, nil
}
