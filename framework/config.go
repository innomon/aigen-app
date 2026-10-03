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

type SurrealDBStorageConfig struct {
	URL string `yaml:"url" json:"url"`
}

type SqliteP2PStorageConfig struct {
	URL       string `yaml:"url" json:"url"`
	UrlPrefix string `yaml:"url_prefix,omitempty" json:"url_prefix,omitempty"`
}

type StorageConfig struct {
	Driver    string                 `yaml:"driver" json:"driver"`
	FS        FSConfig               `yaml:"fs" json:"fs"`
	S3        S3Config               `yaml:"s3" json:"s3"`
	GCS       GCSConfig              `yaml:"gcs" json:"gcs"`
	Postgres  PostgresStorageConfig  `yaml:"postgres" json:"postgres"`
	SurrealDB SurrealDBStorageConfig `yaml:"surrealdb" json:"surrealdb"`
	SqliteP2P SqliteP2PStorageConfig `yaml:"sqlite_p2p" json:"sqlite_p2p"`
}

type AdminConfig struct {
	Email    string `yaml:"email" json:"email"`
	Password string `yaml:"password" json:"password"`
}

type AuthConfig struct {
	JWTSecret string `yaml:"jwt_secret" json:"jwt_secret"`
}

type LogConfig struct {
	Level          string `yaml:"level" json:"level"`
	ConsoleEnabled bool   `yaml:"console_enabled" json:"console_enabled"`
	FileEnabled    bool   `yaml:"file_enabled" json:"file_enabled"`
	Dir            string `yaml:"dir" json:"dir"`
	FileName       string `yaml:"file_name" json:"file_name"`
	MaxSizeMB      int    `yaml:"max_size_mb" json:"max_size_mb"`
	MaxBackups     int    `yaml:"max_backups" json:"max_backups"`
}

type Config struct {
	BizDefsDir        string                     `yaml:"bizdefs_dir" json:"bizdefs_dir"`
	WWWRoot           string                     `yaml:"www_root" json:"www_root"`
	CustomUIPath      string                     `yaml:"custom_ui_path,omitempty" json:"custom_ui_path,omitempty"`
	DatabaseDSN       string                     `yaml:"database_dsn" json:"database_dsn"`
	Domain            string                     `yaml:"domain" json:"domain"`
	Port              string                     `yaml:"port" json:"port"`
	AgenticConfigPath string                     `yaml:"agentic_config_path" json:"agentic_config_path"`
	Channels          descriptors.ChannelsConfig `yaml:"channels" json:"channels"`
	MCP               descriptors.MCPConfig      `yaml:"mcp" json:"mcp"`
	AppExtensionsDir  string                     `yaml:"app_extensions_dir" json:"app_extensions_dir"`
	Storage           StorageConfig              `yaml:"storage" json:"storage"`
	TemporaryAccess   []descriptors.TemporaryAccessConfig `yaml:"temporary_access" json:"temporary_access"`
	Admin             AdminConfig                `yaml:"admin" json:"admin"`
	Auth              AuthConfig                 `yaml:"auth" json:"auth"`
	Log               LogConfig                  `yaml:"log" json:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		BizDefsDir:        "bizdefs",
		AppExtensionsDir:  "app-extensions",
		WWWRoot:           "wwwroot",
		CustomUIPath:      "",
		DatabaseDSN:       "sqlite-p2p://.aigen.db",
		Port:              "5000",
		AgenticConfigPath: "agentic.yaml",
		Storage: StorageConfig{
			Driver: "sqlite-p2p",
			SqliteP2P: SqliteP2PStorageConfig{
				URL:       "sqlite-p2p://.aigen.db",
				UrlPrefix: "/files",
			},
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
		Admin: AdminConfig{
			Email:    "",
			Password: "",
		},
		Auth: AuthConfig{
			JWTSecret: "",
		},
		Log: LogConfig{
			Level:          "INFO",
			ConsoleEnabled: true,
			FileEnabled:    true,
			Dir:            "logs",
			FileName:       "server.log",
			MaxSizeMB:      10,
			MaxBackups:     5,
		},
	}
}


func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	if path == "" {
		if envPath := os.Getenv("FORMCMS_CONFIG_PATH"); envPath != "" {
			path = envPath
		} else if _, err := os.Stat("config.yaml"); err == nil {
			path = "config.yaml"
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
	if appExtensionsDir := os.Getenv("FORMCMS_APP_EXTENSIONS_DIR"); appExtensionsDir != "" {
		config.AppExtensionsDir = appExtensionsDir
	} else if pluginsDir := os.Getenv("FORMCMS_PLUGINS_DIR"); pluginsDir != "" {
		config.AppExtensionsDir = pluginsDir
	}
	if bizdefsDir := os.Getenv("FORMCMS_BIZDEFS_DIR"); bizdefsDir != "" {
		config.BizDefsDir = bizdefsDir
	}
	if wwwRoot := os.Getenv("FORMCMS_WWW_ROOT"); wwwRoot != "" {
		config.WWWRoot = wwwRoot
	}
	if customUIPath := os.Getenv("FORMCMS_CUSTOM_UI_PATH"); customUIPath != "" {
		config.CustomUIPath = customUIPath
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
	if adminEmail := os.Getenv("AIGEN_ADMIN_EMAIL"); adminEmail != "" {
		config.Admin.Email = adminEmail
	}
	if adminPass := os.Getenv("AIGEN_ADMIN_PASSWORD"); adminPass != "" {
		config.Admin.Password = adminPass
	}
	if jwtSecret := os.Getenv("AIGEN_JWT_SECRET"); jwtSecret != "" {
		config.Auth.JWTSecret = jwtSecret
	} else if jwtSecret := os.Getenv("FORMCMS_JWT_SECRET"); jwtSecret != "" {
		config.Auth.JWTSecret = jwtSecret
	}

	return config, nil
}
