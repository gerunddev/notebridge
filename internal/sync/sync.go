package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gerunddev/notebridge/internal/config"
	"github.com/gerunddev/notebridge/internal/state"
)

// Syncer handles bidirectional sync between org-roam and Obsidian
type Syncer struct {
	config *config.Config
	state  *state.State
}

// NewSyncer creates a new syncer instance
func NewSyncer(cfg *config.Config, st *state.State) *Syncer {
	return &Syncer{
		config: cfg,
		state:  st,
	}
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	FilesProcessed int
	Conflicts      []string
	Errors         []error
	StartTime      time.Time
	EndTime        time.Time
}

// Sync performs a one-shot bidirectional sync
func (s *Syncer) Sync() (*SyncResult, error) {
	result := &SyncResult{
		StartTime: time.Now(),
	}

	// TODO: Implement sync logic
	// 1. Scan org_dir for .org files
	// 2. Scan obsidian_dir for .md files
	// 3. For each paired file:
	//    - Check if one or both have changed
	//    - Resolve conflicts (last-write-wins)
	//    - Convert and sync the newer version
	// 4. Handle new files (no pair yet)
	// 5. Update state

	result.EndTime = time.Now()
	return result, nil
}

// ResolveConflict implements last-write-wins conflict resolution
func (s *Syncer) ResolveConflict(orgPath, mdPath string) (string, error) {
	orgInfo, err := os.Stat(orgPath)
	if err != nil {
		return "", err
	}

	mdInfo, err := os.Stat(mdPath)
	if err != nil {
		return "", err
	}

	// Compare modification times
	if orgInfo.ModTime().After(mdInfo.ModTime()) {
		return "org", nil
	}
	return "obsidian", nil
}

// ScanDirectory scans a directory for files with given extension
func ScanDirectory(dir string, ext string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ext {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// String returns a human-readable summary of the sync result
func (r *SyncResult) String() string {
	duration := r.EndTime.Sub(r.StartTime)
	return fmt.Sprintf(
		"Sync complete: %d files synced, %d conflicts, %d errors (took %v)",
		r.FilesProcessed,
		len(r.Conflicts),
		len(r.Errors),
		duration,
	)
}
