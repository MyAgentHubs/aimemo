package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds all aimemo configuration.
type Config struct {
	Storage StorageConfig `toml:"storage"`
	Search  SearchConfig  `toml:"search"`
	Scoring ScoringConfig `toml:"scoring"`
	Server  ServerConfig  `toml:"server"`
	MCP     MCPConfig     `toml:"mcp"`
}

type StorageConfig struct {
	GlobalPath  string `toml:"global_path"`
	ProjectFile string `toml:"project_file"`
}

type SearchConfig struct {
	DefaultLimit int  `toml:"default_limit"`
	MaxLimit     int  `toml:"max_limit"`
	FuzzyEnabled bool `toml:"fuzzy_enabled"`
	Highlight    bool `toml:"highlight"`
}

type ScoringConfig struct {
	RecencyWeight float64 `toml:"recency_weight"`
	AccessWeight  float64 `toml:"access_weight"`
}

type ServerConfig struct {
	DefaultTransport string `toml:"default_transport"`
	HTTPPort         int    `toml:"http_port"`
	HTTPHost         string `toml:"http_host"`
}

type MCPConfig struct {
	ServerName    string `toml:"server_name"`
	ServerVersion string `toml:"server_version"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{
		Storage: StorageConfig{
			GlobalPath:  "~/.aimemo",
			ProjectFile: ".aimemo/memory.db",
		},
		Search: SearchConfig{
			DefaultLimit: 10,
			MaxLimit:     50,
			FuzzyEnabled: true,
			Highlight:    true,
		},
		Scoring: ScoringConfig{
			RecencyWeight: 0.6,
			AccessWeight:  0.4,
		},
		Server: ServerConfig{
			DefaultTransport: "stdio",
			HTTPPort:         8080,
			HTTPHost:         "127.0.0.1",
		},
		MCP: MCPConfig{
			ServerName:    "aimemo-memory",
			ServerVersion: "1.0.0",
		},
	}
}

// Load reads config from the given path, falling back to defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	_, err = toml.Decode(string(data), &cfg)
	return cfg, err
}
