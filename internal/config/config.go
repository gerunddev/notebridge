package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config represents the notebridge configuration
type Config struct {
	OrgDir      string        `json:"org_dir"`
	ObsidianDir string        `json:"obsidian_dir"`
	LogFile     string        `json:"log_file"`
	StateFile   string        `json:"state_file"`
	Interval    time.Duration `json:"interval"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		LogFile:   "/tmp/notebridge.log",
		StateFile: filepath.Join(home, ".notebridge", "state.json"),
		Interval:  30 * time.Second,
	}
}

// Load reads configuration from ~/.notebridge/config.json
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".notebridge", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return default config if file doesn't exist
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes configuration to ~/.notebridge/config.json
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".notebridge")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
