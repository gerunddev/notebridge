package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

// Config represents the notebridge configuration
type Config struct {
	OrgDir             string        `json:"org_dir"`
	ObsidianDir        string        `json:"obsidian_dir"`
	LogFile            string        `json:"log_file"`
	Interval           time.Duration `json:"-"` // Custom JSON handling below
	ResolutionStrategy string        `json:"resolution_strategy,omitempty"`
	ExcludePatterns    []string      `json:"exclude_patterns,omitempty"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		OrgDir:             filepath.Join(home, "org-roam"),
		ObsidianDir:        filepath.Join(home, "Documents", "obsidian-vault"),
		LogFile:            "/tmp/notebridge.log",
		Interval:           30 * time.Second,
		ResolutionStrategy: "last-write-wins", // Default strategy
		ExcludePatterns:    []string{},        // No exclusions by default
	}
}

// ConfigPath returns the path to the config file
// Uses ~/.config on all platforms for consistency
// Can be overridden for testing
var ConfigPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to XDG if home dir unavailable
		return filepath.Join(xdg.ConfigHome, "notebridge", "config.json")
	}
	return filepath.Join(home, ".config", "notebridge", "config.json")
}

// StateFilePath returns the path to the state file
// Uses platform-specific XDG data directory
// Can be overridden for testing
var StateFilePath = func() string {
	return filepath.Join(xdg.DataHome, "notebridge", "state.json")
}

// Load reads configuration from the XDG config directory
func Load() (*Config, error) {
	configPath := ConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return default config if file doesn't exist
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	// Use custom struct for JSON parsing to handle duration as string
	var raw struct {
		OrgDir             string   `json:"org_dir"`
		ObsidianDir        string   `json:"obsidian_dir"`
		LogFile            string   `json:"log_file"`
		Interval           string   `json:"interval"`
		ResolutionStrategy string   `json:"resolution_strategy"`
		ExcludePatterns    []string `json:"exclude_patterns"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Parse interval duration
	interval, err := time.ParseDuration(raw.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval format '%s': %w", raw.Interval, err)
	}

	// Set default resolution strategy if not specified
	resolutionStrategy := raw.ResolutionStrategy
	if resolutionStrategy == "" {
		resolutionStrategy = "last-write-wins"
	}

	// Set empty slice for exclude patterns if nil
	excludePatterns := raw.ExcludePatterns
	if excludePatterns == nil {
		excludePatterns = []string{}
	}

	cfg := &Config{
		OrgDir:             raw.OrgDir,
		ObsidianDir:        raw.ObsidianDir,
		LogFile:            raw.LogFile,
		Interval:           interval,
		ResolutionStrategy: resolutionStrategy,
		ExcludePatterns:    excludePatterns,
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Expand paths
	if err := cfg.ExpandPaths(); err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}

	return cfg, nil
}

// Save writes configuration to the XDG config directory
func (c *Config) Save() error {
	configPath := ConfigPath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Use custom struct for JSON to handle duration as string
	raw := struct {
		OrgDir             string   `json:"org_dir"`
		ObsidianDir        string   `json:"obsidian_dir"`
		LogFile            string   `json:"log_file"`
		Interval           string   `json:"interval"`
		ResolutionStrategy string   `json:"resolution_strategy,omitempty"`
		ExcludePatterns    []string `json:"exclude_patterns,omitempty"`
	}{
		OrgDir:             c.OrgDir,
		ObsidianDir:        c.ObsidianDir,
		LogFile:            c.LogFile,
		Interval:           c.Interval.String(),
		ResolutionStrategy: c.ResolutionStrategy,
		ExcludePatterns:    c.ExcludePatterns,
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.OrgDir == "" {
		return fmt.Errorf("org_dir cannot be empty")
	}
	if c.ObsidianDir == "" {
		return fmt.Errorf("obsidian_dir cannot be empty")
	}
	if c.LogFile == "" {
		return fmt.Errorf("log_file cannot be empty")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}

	// Validate resolution strategy
	validStrategies := map[string]bool{
		"last-write-wins": true,
		"use-org":         true,
		"use-markdown":    true,
	}
	if !validStrategies[c.ResolutionStrategy] {
		return fmt.Errorf("invalid resolution_strategy '%s': must be one of: last-write-wins, use-org, use-markdown", c.ResolutionStrategy)
	}

	return nil
}

// ExpandPaths expands any ~ or relative paths to absolute paths
func (c *Config) ExpandPaths() error {
	var err error

	c.OrgDir, err = expandPath(c.OrgDir)
	if err != nil {
		return fmt.Errorf("failed to expand org_dir: %w", err)
	}

	c.ObsidianDir, err = expandPath(c.ObsidianDir)
	if err != nil {
		return fmt.Errorf("failed to expand obsidian_dir: %w", err)
	}

	c.LogFile, err = expandPath(c.LogFile)
	if err != nil {
		return fmt.Errorf("failed to expand log_file: %w", err)
	}

	return nil
}

// expandPath expands ~ to home directory and converts to absolute path
func expandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	// Expand ~ to home directory
	if path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if len(path) == 1 {
			return homeDir, nil
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}
