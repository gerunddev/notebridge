package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.OrgDir == "" {
		t.Error("Expected OrgDir to be set")
	}
	if cfg.ObsidianDir == "" {
		t.Error("Expected ObsidianDir to be set")
	}
	if cfg.LogFile == "" {
		t.Error("Expected LogFile to be set")
	}
	if cfg.StateFile == "" {
		t.Error("Expected StateFile to be set")
	}
	if cfg.Interval != 30*time.Second {
		t.Errorf("Expected Interval to be 30s, got %v", cfg.Interval)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty org_dir",
			config: &Config{
				OrgDir:      "",
				ObsidianDir: "/path/to/obsidian",
				LogFile:     "/tmp/test.log",
				StateFile:   "/tmp/state.json",
				Interval:    30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "empty obsidian_dir",
			config: &Config{
				OrgDir:      "/path/to/org",
				ObsidianDir: "",
				LogFile:     "/tmp/test.log",
				StateFile:   "/tmp/state.json",
				Interval:    30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero interval",
			config: &Config{
				OrgDir:      "/path/to/org",
				ObsidianDir: "/path/to/obsidian",
				LogFile:     "/tmp/test.log",
				StateFile:   "/tmp/state.json",
				Interval:    0,
			},
			wantErr: true,
		},
		{
			name: "negative interval",
			config: &Config{
				OrgDir:      "/path/to/org",
				ObsidianDir: "/path/to/obsidian",
				LogFile:     "/tmp/test.log",
				StateFile:   "/tmp/state.json",
				Interval:    -5 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "config.json")

	// Override ConfigPath for testing
	originalConfigPath := ConfigPath
	ConfigPath = func() string {
		return testConfigPath
	}
	defer func() {
		ConfigPath = originalConfigPath
	}()

	// Create test config
	testCfg := &Config{
		OrgDir:      "/test/org-roam",
		ObsidianDir: "/test/obsidian",
		LogFile:     "/tmp/notebridge-test.log",
		StateFile:   "/tmp/state-test.json",
		Interval:    45 * time.Second,
	}

	// Save config
	if err := testCfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Compare (paths will be expanded, so just check they're set)
	if loadedCfg.Interval != testCfg.Interval {
		t.Errorf("Interval mismatch: got %v, want %v", loadedCfg.Interval, testCfg.Interval)
	}
	if loadedCfg.LogFile == "" {
		t.Error("LogFile should not be empty")
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "nonexistent.json")

	// Override ConfigPath for testing
	originalConfigPath := ConfigPath
	ConfigPath = func() string {
		return testConfigPath
	}
	defer func() {
		ConfigPath = originalConfigPath
	}()

	// Load should return default config when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error on missing file: %v", err)
	}

	// Should return default config
	if cfg.Interval != 30*time.Second {
		t.Errorf("Expected default interval 30s, got %v", cfg.Interval)
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		contains string // The output should contain this
	}{
		{
			name:     "tilde expansion",
			input:    "~/test",
			contains: homeDir,
		},
		{
			name:     "tilde only",
			input:    "~",
			contains: homeDir,
		},
		{
			name:     "absolute path",
			input:    "/tmp/test",
			contains: "/tmp/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.input)
			if err != nil {
				t.Fatalf("expandPath() error = %v", err)
			}
			if result == "" {
				t.Error("expandPath() returned empty string")
			}
			// Just verify it's not the original unexpanded path
			if tt.input[0] == '~' && result == tt.input {
				t.Errorf("Path was not expanded: %s", result)
			}
		})
	}
}

func TestConfigPathsExpanded(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "config.json")

	// Override ConfigPath for testing
	originalConfigPath := ConfigPath
	ConfigPath = func() string {
		return testConfigPath
	}
	defer func() {
		ConfigPath = originalConfigPath
	}()

	// Create test config with tilde paths
	testCfg := &Config{
		OrgDir:      "~/org-roam",
		ObsidianDir: "~/Documents/obsidian",
		LogFile:     "~/notebridge.log",
		StateFile:   "~/.notebridge/state.json",
		Interval:    30 * time.Second,
	}

	// Save and load
	if err := testCfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify paths are expanded (no longer contain ~)
	if loadedCfg.OrgDir[0] == '~' {
		t.Error("OrgDir was not expanded")
	}
	if loadedCfg.ObsidianDir[0] == '~' {
		t.Error("ObsidianDir was not expanded")
	}
	if loadedCfg.LogFile[0] == '~' {
		t.Error("LogFile was not expanded")
	}
	if loadedCfg.StateFile[0] == '~' {
		t.Error("StateFile was not expanded")
	}
}
