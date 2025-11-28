package sync

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gerunddev/notebridge/internal/config"
	"github.com/gerunddev/notebridge/internal/logger"
	"github.com/gerunddev/notebridge/internal/state"
)

func TestResolveConflict(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test config
	cfg := &config.Config{
		OrgDir:      filepath.Join(tmpDir, "org"),
		ObsidianDir: filepath.Join(tmpDir, "obsidian"),
	}

	// Create directories
	if err := os.MkdirAll(cfg.OrgDir, 0755); err != nil {
		t.Fatalf("Failed to create org directory: %v", err)
	}
	if err := os.MkdirAll(cfg.ObsidianDir, 0755); err != nil {
		t.Fatalf("Failed to create obsidian directory: %v", err)
	}

	tests := []struct {
		name           string
		setupOrg       bool
		setupMd        bool
		modifyOrg      bool
		modifyMd       bool
		orgContent     string
		mdContent      string
		orgNewer       bool
		expectedWinner string
		expectedReason string
	}{
		{
			name:           "only org exists",
			setupOrg:       true,
			setupMd:        false,
			orgContent:     "* Test",
			expectedWinner: "org",
			expectedReason: "md file doesn't exist",
		},
		{
			name:           "only md exists",
			setupOrg:       false,
			setupMd:        true,
			mdContent:      "# Test",
			expectedWinner: "obsidian",
			expectedReason: "org file doesn't exist",
		},
		{
			name:           "neither changed",
			setupOrg:       true,
			setupMd:        true,
			modifyOrg:      false,
			modifyMd:       false,
			expectedWinner: "none",
			expectedReason: "no changes detected",
		},
		{
			name:           "only org changed",
			setupOrg:       true,
			setupMd:        true,
			modifyOrg:      true,
			modifyMd:       false,
			orgContent:     "* Modified",
			expectedWinner: "org",
			expectedReason: "only org file changed",
		},
		{
			name:           "only md changed",
			setupOrg:       true,
			setupMd:        true,
			modifyOrg:      false,
			modifyMd:       true,
			mdContent:      "# Modified",
			expectedWinner: "obsidian",
			expectedReason: "only md file changed",
		},
		{
			name:           "both changed - org newer",
			setupOrg:       true,
			setupMd:        true,
			modifyOrg:      true,
			modifyMd:       true,
			orgContent:     "* Modified",
			mdContent:      "# Modified",
			orgNewer:       true,
			expectedWinner: "org",
			expectedReason: "both changed, org is newer (last-write-wins)",
		},
		{
			name:           "both changed - md newer",
			setupOrg:       true,
			setupMd:        true,
			modifyOrg:      true,
			modifyMd:       true,
			orgContent:     "* Modified",
			mdContent:      "# Modified",
			orgNewer:       false,
			expectedWinner: "obsidian",
			expectedReason: "both changed, obsidian is newer (last-write-wins)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh state for each test
			st := state.NewState()
			syncer := NewSyncer(cfg, st)

			orgPath := filepath.Join(cfg.OrgDir, "test.org")
			mdPath := filepath.Join(cfg.ObsidianDir, "test.md")

			// Clean up from previous test
			_ = os.Remove(orgPath) //nolint:errcheck // cleanup - file may not exist
			_ = os.Remove(mdPath)  //nolint:errcheck // cleanup - file may not exist

			// Setup files based on test case
			if tt.setupOrg {
				if err := os.WriteFile(orgPath, []byte("* Initial"), 0644); err != nil {
					t.Fatalf("Failed to create org file: %v", err)
				}
				// Wait for mtime to settle
				time.Sleep(100 * time.Millisecond)
				// Update state to mark as tracked
				if err := st.Update(orgPath, mdPath); err != nil {
					t.Fatalf("Failed to update state for org: %v", err)
				}
			}

			if tt.setupMd {
				if err := os.WriteFile(mdPath, []byte("# Initial"), 0644); err != nil {
					t.Fatalf("Failed to create md file: %v", err)
				}
				// Wait for mtime to settle
				time.Sleep(100 * time.Millisecond)
				// Update state to mark as tracked
				if err := st.Update(mdPath, orgPath); err != nil {
					t.Fatalf("Failed to update state for md: %v", err)
				}
			}

			// Wait for filesystem timestamp resolution before modifications
			if tt.modifyOrg || tt.modifyMd {
				time.Sleep(1100 * time.Millisecond)
			}

			// Modify files if needed (this will trigger "changed" status)
			if tt.modifyOrg {
				if err := os.WriteFile(orgPath, []byte(tt.orgContent), 0644); err != nil {
					t.Fatalf("Failed to modify org file: %v", err)
				}
			}

			if tt.modifyMd {
				if err := os.WriteFile(mdPath, []byte(tt.mdContent), 0644); err != nil {
					t.Fatalf("Failed to modify md file: %v", err)
				}
			}

			// For "both changed" tests, adjust mtime to control which is newer
			if tt.modifyOrg && tt.modifyMd {
				time.Sleep(1100 * time.Millisecond)
				if tt.orgNewer {
					// Touch org file to make it newer
					now := time.Now()
					if err := os.Chtimes(orgPath, now, now); err != nil {
						t.Fatalf("Failed to change org file time: %v", err)
					}
				} else {
					// Touch md file to make it newer
					now := time.Now()
					if err := os.Chtimes(mdPath, now, now); err != nil {
						t.Fatalf("Failed to change md file time: %v", err)
					}
				}
			}

			// Resolve conflict
			decision, err := syncer.ResolveConflict(orgPath, mdPath)
			if err != nil {
				t.Fatalf("ResolveConflict failed: %v", err)
			}

			// Verify decision
			if decision.Winner != tt.expectedWinner {
				t.Errorf("Expected winner %q, got %q (OrgChanged=%v, MdChanged=%v)",
					tt.expectedWinner, decision.Winner, decision.OrgChanged, decision.MdChanged)
			}
			if decision.Reason != tt.expectedReason {
				t.Errorf("Expected reason %q, got %q", tt.expectedReason, decision.Reason)
			}
		})
	}
}

func TestSyncFilePair(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		OrgDir:      filepath.Join(tmpDir, "org"),
		ObsidianDir: filepath.Join(tmpDir, "obsidian"),
	}

	if err := os.MkdirAll(cfg.OrgDir, 0755); err != nil {
		t.Fatalf("Failed to create org directory: %v", err)
	}
	if err := os.MkdirAll(cfg.ObsidianDir, 0755); err != nil {
		t.Fatalf("Failed to create obsidian directory: %v", err)
	}

	st := state.NewState()
	syncer := NewSyncer(cfg, st)

	// Capture log output
	var logBuf bytes.Buffer
	syncer.SetLogger(logger.New(&logBuf))

	orgPath := filepath.Join(cfg.OrgDir, "test.org")
	mdPath := filepath.Join(cfg.ObsidianDir, "test.md")

	t.Run("sync org to md", func(t *testing.T) {
		// Create org file
		orgContent := `* Test Note

This is a test.`
		if err := os.WriteFile(orgPath, []byte(orgContent), 0644); err != nil {
			t.Fatalf("Failed to create org file: %v", err)
		}

		// Sync
		synced, err := syncer.SyncFilePair(orgPath, mdPath)
		if err != nil {
			t.Fatalf("SyncFilePair failed: %v", err)
		}
		if !synced {
			t.Error("Expected sync to occur")
		}

		// Verify md file was created
		if _, err := os.Stat(mdPath); os.IsNotExist(err) {
			t.Error("MD file was not created")
		}

		// Verify log output (structured logger format)
		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "file synced") {
			t.Errorf("Expected 'file synced' log message, got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "test.org") {
			t.Error("Expected org filename in log")
		}

		// Verify state was updated
		if st.Files[orgPath] == nil {
			t.Error("Org file state was not updated")
		}
		if st.Files[mdPath] == nil {
			t.Error("MD file state was not updated")
		}
	})

	t.Run("no sync when unchanged", func(t *testing.T) {
		logBuf.Reset()

		// Sync again - should be no-op
		synced, err := syncer.SyncFilePair(orgPath, mdPath)
		if err != nil {
			t.Fatalf("Second sync failed: %v", err)
		}
		if synced {
			t.Error("Should not sync unchanged files")
		}

		// Should not log sync (no changes)
		logOutput := logBuf.String()
		if strings.Contains(logOutput, "file synced") {
			t.Error("Should not sync unchanged files")
		}
	})
}

func TestConflictLogging(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		OrgDir:      filepath.Join(tmpDir, "org"),
		ObsidianDir: filepath.Join(tmpDir, "obsidian"),
	}

	if err := os.MkdirAll(cfg.OrgDir, 0755); err != nil {
		t.Fatalf("Failed to create org directory: %v", err)
	}
	if err := os.MkdirAll(cfg.ObsidianDir, 0755); err != nil {
		t.Fatalf("Failed to create obsidian directory: %v", err)
	}

	st := state.NewState()
	syncer := NewSyncer(cfg, st)

	// Capture log output
	var logBuf bytes.Buffer
	syncer.SetLogger(logger.New(&logBuf))

	orgPath := filepath.Join(cfg.OrgDir, "conflict.org")
	mdPath := filepath.Join(cfg.ObsidianDir, "conflict.md")

	// Create both files
	if err := os.WriteFile(orgPath, []byte("* Initial"), 0644); err != nil {
		t.Fatalf("Failed to create org file: %v", err)
	}
	if err := os.WriteFile(mdPath, []byte("# Initial"), 0644); err != nil {
		t.Fatalf("Failed to create md file: %v", err)
	}

	// Update state
	if err := st.Update(orgPath, mdPath); err != nil {
		t.Fatalf("Failed to update state for org: %v", err)
	}

	// Wait for timestamp resolution
	time.Sleep(1100 * time.Millisecond)

	// Modify both files to create conflict
	if err := os.WriteFile(orgPath, []byte("* Modified org"), 0644); err != nil {
		t.Fatalf("Failed to modify org file: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if err := os.WriteFile(mdPath, []byte("# Modified md"), 0644); err != nil {
		t.Fatalf("Failed to modify md file: %v", err)
	}

	// Resolve conflict - md should win (newer)
	decision, err := syncer.ResolveConflict(orgPath, mdPath)
	if err != nil {
		t.Fatalf("ResolveConflict failed: %v", err)
	}

	// Verify conflict was logged (structured logger format)
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "conflict resolved") {
		t.Errorf("Expected 'conflict resolved' log message, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "newer modification time") {
		t.Errorf("Expected 'newer modification time' in log, got: %s", logOutput)
	}

	// Verify correct winner
	if decision.Winner != "obsidian" {
		t.Errorf("Expected obsidian to win, got %s", decision.Winner)
	}
}

func TestScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.org"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.org"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create subdirectory with files
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file3.org"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file in subdir: %v", err)
	}

	// Scan for .org files
	orgFiles, err := ScanDirectory(tmpDir, ".org", []string{})
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// Should find 3 org files
	if len(orgFiles) != 3 {
		t.Errorf("Expected 3 .org files, got %d", len(orgFiles))
	}

	// Scan for .md files
	mdFiles, err := ScanDirectory(tmpDir, ".md", []string{})
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// Should find 1 md file
	if len(mdFiles) != 1 {
		t.Errorf("Expected 1 .md file, got %d", len(mdFiles))
	}
}
