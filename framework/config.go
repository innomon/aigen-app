package framework

import (
	"fmt"
	"os"

	"github.com/innomon/aigen-cms/core/descriptors"
	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	AppsDir           string                     `yaml:"apps_dir" json:"apps_dir"`
	WWWRoot           string                     `yaml:"www_root" json:"www_root"`
	DatabaseType      string                     `yaml:"database_type" json:"database_type"`
	DatabaseDSN       string                     `yaml:"database_dsn" json:"database_dsn"`
	Domain            string                     `yaml:"domain" json:"domain"`
	Port              string                     `yaml:"port" json:"port"`
	AgenticConfigPath string                     `yaml:"agentic_config_path" json:"agentic_config_path"`
	Channels          descriptors.ChannelsConfig `yaml:"channels" json:"channels"`
	MCP               descriptors.MCPConfig      `yaml:"mcp" json:"mcp"`
}

func DefaultConfig() *Config {
	return &Config{
		AppsDir:           "apps",
		WWWRoot:           "wwwroot",
		DatabaseType:      "SQLite",
		DatabaseDSN:       "formcms.db",
		Port:              "5000",
		AgenticConfigPath: "agentic.yaml",
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
	if appsDir := os.Getenv("FORMCMS_APPS_DIR"); appsDir != "" {
		config.AppsDir = appsDir
	}
	if wwwRoot := os.Getenv("FORMCMS_WWW_ROOT"); wwwRoot != "" {
		config.WWWRoot = wwwRoot
	}
	if dbType := os.Getenv("FORMCMS_DB_TYPE"); dbType != "" {
		config.DatabaseType = dbType
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
