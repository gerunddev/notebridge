package sync

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gerunddev/notebridge/internal/config"
	"github.com/gerunddev/notebridge/internal/convert"
	"github.com/gerunddev/notebridge/internal/state"
)

// Syncer handles bidirectional sync between org-roam and Obsidian
type Syncer struct {
	config *config.Config
	state  *state.State
	logger *log.Logger
}

// NewSyncer creates a new syncer instance
func NewSyncer(cfg *config.Config, st *state.State) *Syncer {
	return &Syncer{
		config: cfg,
		state:  st,
		logger: log.Default(),
	}
}

// SetLogger sets a custom logger for the syncer
func (s *Syncer) SetLogger(logger *log.Logger) {
	s.logger = logger
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

	// 1. Scan org_dir for .org files
	orgFiles, err := ScanDirectory(s.config.OrgDir, ".org")
	if err != nil {
		return nil, fmt.Errorf("failed to scan org directory: %w", err)
	}

	// 2. Scan obsidian_dir for .md files
	mdFiles, err := ScanDirectory(s.config.ObsidianDir, ".md")
	if err != nil {
		return nil, fmt.Errorf("failed to scan obsidian directory: %w", err)
	}

	// Build a set of md files for quick lookup
	mdFileSet := make(map[string]bool)
	for _, mdPath := range mdFiles {
		mdFileSet[mdPath] = true
	}

	// Track which md files have been processed (to find orphan md files)
	processedMd := make(map[string]bool)

	// 3. Process each org file
	for _, orgPath := range orgFiles {
		// Calculate corresponding md path
		relPath, err := filepath.Rel(s.config.OrgDir, orgPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to get relative path for %s: %w", orgPath, err))
			continue
		}

		// Replace .org with .md
		baseName := relPath[:len(relPath)-4]
		mdPath := filepath.Join(s.config.ObsidianDir, baseName+".md")

		// Mark as processed
		processedMd[mdPath] = true

		// Sync the file pair
		synced, err := s.SyncFilePair(orgPath, mdPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("sync failed for %s: %w", relPath, err))
			continue
		}
		if synced {
			result.FilesProcessed++
		}
	}

	// 4. Handle orphan md files (md files without corresponding org)
	for _, mdPath := range mdFiles {
		if processedMd[mdPath] {
			continue
		}

		// Calculate corresponding org path
		relPath, err := filepath.Rel(s.config.ObsidianDir, mdPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to get relative path for %s: %w", mdPath, err))
			continue
		}

		// Replace .md with .org
		baseName := relPath[:len(relPath)-3]
		orgPath := filepath.Join(s.config.OrgDir, baseName+".org")

		// Sync the file pair (org doesn't exist, so md will win)
		synced, err := s.SyncFilePair(orgPath, mdPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("sync failed for %s: %w", relPath, err))
			continue
		}
		if synced {
			result.FilesProcessed++
		}
	}

	result.EndTime = time.Now()
	return result, nil
}

// ConflictDecision represents the result of conflict resolution
type ConflictDecision struct {
	Winner     string // "org", "obsidian", or "none"
	Reason     string
	OrgChanged bool
	MdChanged  bool
}

// ResolveConflict implements last-write-wins conflict resolution
// Returns which file should be the source of truth
func (s *Syncer) ResolveConflict(orgPath, mdPath string) (*ConflictDecision, error) {
	// Check if org file exists and has changed
	orgChanged, err := s.state.HasChanged(orgPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check org file: %w", err)
	}
	orgExists := !os.IsNotExist(err)

	// Check if md file exists and has changed
	mdChanged, err := s.state.HasChanged(mdPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check md file: %w", err)
	}
	mdExists := !os.IsNotExist(err)

	decision := &ConflictDecision{
		OrgChanged: orgChanged,
		MdChanged:  mdChanged,
	}

	// Case 1: Neither file exists (shouldn't happen)
	if !orgExists && !mdExists {
		decision.Winner = "none"
		decision.Reason = "neither file exists"
		return decision, nil
	}

	// Case 2: Only org file exists
	if orgExists && !mdExists {
		decision.Winner = "org"
		decision.Reason = "md file doesn't exist"
		return decision, nil
	}

	// Case 3: Only md file exists
	if !orgExists && mdExists {
		decision.Winner = "obsidian"
		decision.Reason = "org file doesn't exist"
		return decision, nil
	}

	// Case 4: Neither changed
	if !orgChanged && !mdChanged {
		decision.Winner = "none"
		decision.Reason = "no changes detected"
		return decision, nil
	}

	// Case 5: Only org changed
	if orgChanged && !mdChanged {
		decision.Winner = "org"
		decision.Reason = "only org file changed"
		return decision, nil
	}

	// Case 6: Only md changed
	if !orgChanged && mdChanged {
		decision.Winner = "obsidian"
		decision.Reason = "only md file changed"
		return decision, nil
	}

	// Case 7: Both changed - last-write-wins
	orgInfo, err := os.Stat(orgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat org file: %w", err)
	}

	mdInfo, err := os.Stat(mdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat md file: %w", err)
	}

	if orgInfo.ModTime().After(mdInfo.ModTime()) {
		decision.Winner = "org"
		decision.Reason = "both changed, org is newer (last-write-wins)"
		s.logger.Printf("[CONFLICT] %s: both modified, org wins (newer mtime)", orgPath)
	} else {
		decision.Winner = "obsidian"
		decision.Reason = "both changed, obsidian is newer (last-write-wins)"
		s.logger.Printf("[CONFLICT] %s: both modified, obsidian wins (newer mtime)", mdPath)
	}

	return decision, nil
}

// SyncFilePair syncs a pair of org and md files based on conflict resolution
// Returns (synced, error) where synced indicates if a sync actually occurred
func (s *Syncer) SyncFilePair(orgPath, mdPath string) (bool, error) {
	decision, err := s.ResolveConflict(orgPath, mdPath)
	if err != nil {
		return false, fmt.Errorf("conflict resolution failed: %w", err)
	}

	// No sync needed
	if decision.Winner == "none" {
		return false, nil
	}

	// Sync based on winner
	switch decision.Winner {
	case "org":
		// Convert org -> md
		if err := s.convertOrgToMd(orgPath, mdPath); err != nil {
			return false, fmt.Errorf("failed to convert org to md: %w", err)
		}
		s.logger.Printf("[SYNC] %s → %s (%s)", filepath.Base(orgPath), filepath.Base(mdPath), decision.Reason)

	case "obsidian":
		// Convert md -> org
		if err := s.convertMdToOrg(mdPath, orgPath); err != nil {
			return false, fmt.Errorf("failed to convert md to org: %w", err)
		}
		s.logger.Printf("[SYNC] %s → %s (%s)", filepath.Base(mdPath), filepath.Base(orgPath), decision.Reason)
	}

	// Update state for both files
	if err := s.state.Update(orgPath, mdPath); err != nil {
		return false, fmt.Errorf("failed to update org state: %w", err)
	}
	if err := s.state.Update(mdPath, orgPath); err != nil {
		return false, fmt.Errorf("failed to update md state: %w", err)
	}

	return true, nil
}

// convertOrgToMd converts an org file to markdown
func (s *Syncer) convertOrgToMd(orgPath, mdPath string) error {
	// Read org file
	content, err := os.ReadFile(orgPath)
	if err != nil {
		return err
	}

	// Convert using id map from state
	md, err := convert.OrgToMarkdown(string(content), s.state.IDMap)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(mdPath), 0755); err != nil {
		return err
	}

	// Write md file
	return os.WriteFile(mdPath, []byte(md), 0644)
}

// convertMdToOrg converts a markdown file to org
func (s *Syncer) convertMdToOrg(mdPath, orgPath string) error {
	// Read md file
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}

	// Convert using id map from state
	org, err := convert.MarkdownToOrg(string(content), s.state.IDMap)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(orgPath), 0755); err != nil {
		return err
	}

	// Write org file
	return os.WriteFile(orgPath, []byte(org), 0644)
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
