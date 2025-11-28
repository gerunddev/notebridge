package sync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gerunddev/notebridge/internal/config"
	"github.com/gerunddev/notebridge/internal/convert"
	"github.com/gerunddev/notebridge/internal/logger"
	"github.com/gerunddev/notebridge/internal/state"
)

// Error types for categorization
var (
	ErrFileAccess   = errors.New("file access error")
	ErrConversion   = errors.New("conversion error")
	ErrState        = errors.New("state error")
	ErrPermission   = errors.New("permission denied")
)

// isRetryable returns true if the error is transient and worth retrying
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Retry on temporary filesystem errors
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		// Resource temporarily unavailable, etc.
		return true
	}
	return false
}

// withRetry executes a function with retry logic for transient errors
func withRetry(maxRetries int, delay time.Duration, fn func() error) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) {
			return lastErr
		}
		if i < maxRetries {
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("failed after %d retries: %w", maxRetries+1, lastErr)
}

// atomicWriteFile writes content to a file atomically by writing to a temp file first
// If dry-run mode is enabled, skips the actual write but logs what would have been done
func (s *Syncer) atomicWriteFile(path string, content []byte, perm os.FileMode) error {
	// In dry-run mode, skip the actual write
	if s.DryRun {
		s.logger.Info("dry-run: would write file", "path", path, "size", len(content))
		return nil
	}
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temp file in same directory (for atomic rename)
	tmpFile, err := os.CreateTemp(dir, ".notebridge-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Write content
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// Syncer handles bidirectional sync between org-roam and Obsidian
type Syncer struct {
	config *config.Config
	state  *state.State
	logger *logger.Logger
	DryRun bool // If true, skip actual file writes
}

// NewSyncer creates a new syncer instance
func NewSyncer(cfg *config.Config, st *state.State) *Syncer {
	return &Syncer{
		config: cfg,
		state:  st,
		logger: logger.Discard(),
	}
}

// SetLogger sets a custom logger for the syncer
func (s *Syncer) SetLogger(l *logger.Logger) {
	s.logger = l
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

	s.logger.SyncStarted(s.config.OrgDir, s.config.ObsidianDir)

	// 1. Scan org_dir for .org files
	orgFiles, err := ScanDirectory(s.config.OrgDir, ".org", s.config.ExcludePatterns)
	if err != nil {
		s.logger.Error("failed to scan org directory", "error", err)
		return nil, fmt.Errorf("failed to scan org directory: %w", err)
	}

	// 2. Scan obsidian_dir for .md files
	mdFiles, err := ScanDirectory(s.config.ObsidianDir, ".md", s.config.ExcludePatterns)
	if err != nil {
		s.logger.Error("failed to scan obsidian directory", "error", err)
		return nil, fmt.Errorf("failed to scan obsidian directory: %w", err)
	}

	s.logger.Debug("directories scanned",
		"org_files", len(orgFiles),
		"md_files", len(mdFiles))

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
			s.logger.FileError(orgPath, err)
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
			s.logger.FileError(relPath, err)
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
			s.logger.FileError(mdPath, err)
			result.Errors = append(result.Errors, fmt.Errorf("failed to get relative path for %s: %w", mdPath, err))
			continue
		}

		// Replace .md with .org
		baseName := relPath[:len(relPath)-3]
		orgPath := filepath.Join(s.config.OrgDir, baseName+".org")

		// Sync the file pair (org doesn't exist, so md will win)
		synced, err := s.SyncFilePair(orgPath, mdPath)
		if err != nil {
			s.logger.FileError(relPath, err)
			result.Errors = append(result.Errors, fmt.Errorf("sync failed for %s: %w", relPath, err))
			continue
		}
		if synced {
			result.FilesProcessed++
		}
	}

	result.EndTime = time.Now()
	duration := result.EndTime.Sub(result.StartTime)
	s.logger.SyncCompleted(result.FilesProcessed, len(result.Errors), duration)

	return result, nil
}

// ConflictDecision represents the result of conflict resolution
type ConflictDecision struct {
	Winner     string // "org", "obsidian", or "none"
	Reason     string
	OrgChanged bool
	MdChanged  bool
}

// ResolveConflict resolves conflicts using the configured resolution strategy
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

	// Case 7: Both changed - apply configured resolution strategy
	baseName := filepath.Base(orgPath)
	baseName = baseName[:len(baseName)-4] // Remove .org

	switch s.config.ResolutionStrategy {
	case "use-org":
		decision.Winner = "org"
		decision.Reason = "both changed, using org (configured strategy)"
		s.logger.Conflict(baseName, "org", "using org per resolution strategy")

	case "use-markdown":
		decision.Winner = "obsidian"
		decision.Reason = "both changed, using markdown (configured strategy)"
		s.logger.Conflict(baseName, "obsidian", "using markdown per resolution strategy")

	case "last-write-wins":
		fallthrough
	default:
		// Last-write-wins: check modification times
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
			s.logger.Conflict(baseName, "org", "org has newer modification time")
		} else {
			decision.Winner = "obsidian"
			decision.Reason = "both changed, obsidian is newer (last-write-wins)"
			s.logger.Conflict(baseName, "obsidian", "obsidian has newer modification time")
		}
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
			s.logger.ConversionError(orgPath, mdPath, err)
			return false, fmt.Errorf("failed to convert org to md: %w", err)
		}
		s.logger.FileSynced(filepath.Base(orgPath), filepath.Base(mdPath), decision.Reason)

	case "obsidian":
		// Convert md -> org
		if err := s.convertMdToOrg(mdPath, orgPath); err != nil {
			s.logger.ConversionError(mdPath, orgPath, err)
			return false, fmt.Errorf("failed to convert md to org: %w", err)
		}
		s.logger.FileSynced(filepath.Base(mdPath), filepath.Base(orgPath), decision.Reason)
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

// SyncFileWithResolution syncs a file pair with a forced resolution direction
// direction can be "org" (use org version), "obsidian" (use md version), "last-write-wins", or "skip"
func (s *Syncer) SyncFileWithResolution(orgPath, mdPath, direction string) error {
	if direction == "skip" {
		s.logger.Info("file skipped by user", "org", orgPath, "md", mdPath)
		return nil
	}

	// Handle last-write-wins by checking modification times
	if direction == "last-write-wins" {
		orgInfo, err := os.Stat(orgPath)
		if err != nil {
			return fmt.Errorf("failed to stat org file: %w", err)
		}
		mdInfo, err := os.Stat(mdPath)
		if err != nil {
			return fmt.Errorf("failed to stat md file: %w", err)
		}

		if orgInfo.ModTime().After(mdInfo.ModTime()) {
			direction = "org"
		} else {
			direction = "markdown"
		}
		s.logger.Info("last-write-wins resolution",
			"org_mtime", orgInfo.ModTime(),
			"md_mtime", mdInfo.ModTime(),
			"winner", direction)
	}

	// Sync based on direction
	switch direction {
	case "org":
		// Convert org -> md
		if err := s.convertOrgToMd(orgPath, mdPath); err != nil {
			s.logger.ConversionError(orgPath, mdPath, err)
			return fmt.Errorf("failed to convert org to md: %w", err)
		}
		s.logger.FileSynced(filepath.Base(orgPath), filepath.Base(mdPath), "user chose org version")

	case "obsidian", "markdown":
		// Convert md -> org
		if err := s.convertMdToOrg(mdPath, orgPath); err != nil {
			s.logger.ConversionError(mdPath, orgPath, err)
			return fmt.Errorf("failed to convert md to org: %w", err)
		}
		s.logger.FileSynced(filepath.Base(mdPath), filepath.Base(orgPath), "user chose markdown version")

	default:
		return fmt.Errorf("invalid resolution direction: %s", direction)
	}

	// Update state for both files
	if err := s.state.Update(orgPath, mdPath); err != nil {
		return fmt.Errorf("failed to update org state: %w", err)
	}
	if err := s.state.Update(mdPath, orgPath); err != nil {
		return fmt.Errorf("failed to update md state: %w", err)
	}

	return nil
}

// convertOrgToMd converts an org file to markdown with retry and atomic write
func (s *Syncer) convertOrgToMd(orgPath, mdPath string) error {
	var content []byte
	var md string

	// Read with retry
	err := withRetry(2, 100*time.Millisecond, func() error {
		var err error
		content, err = os.ReadFile(orgPath)
		return err
	})
	if err != nil {
		return fmt.Errorf("%w: reading %s: %v", ErrFileAccess, orgPath, err)
	}

	// Convert using id map from state
	md, err = convert.OrgToMarkdown(string(content), s.state.IDMap)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConversion, err)
	}

	// Write atomically with retry
	err = withRetry(2, 100*time.Millisecond, func() error {
		return s.atomicWriteFile(mdPath, []byte(md), 0644)
	})
	if err != nil {
		return fmt.Errorf("%w: writing %s: %v", ErrFileAccess, mdPath, err)
	}

	return nil
}

// convertMdToOrg converts a markdown file to org with retry and atomic write
func (s *Syncer) convertMdToOrg(mdPath, orgPath string) error {
	var content []byte
	var org string

	// Read with retry
	err := withRetry(2, 100*time.Millisecond, func() error {
		var err error
		content, err = os.ReadFile(mdPath)
		return err
	})
	if err != nil {
		return fmt.Errorf("%w: reading %s: %v", ErrFileAccess, mdPath, err)
	}

	// Convert using id map from state
	org, err = convert.MarkdownToOrg(string(content), s.state.IDMap)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConversion, err)
	}

	// Write atomically with retry
	err = withRetry(2, 100*time.Millisecond, func() error {
		return s.atomicWriteFile(orgPath, []byte(org), 0644)
	})
	if err != nil {
		return fmt.Errorf("%w: writing %s: %v", ErrFileAccess, orgPath, err)
	}

	return nil
}

// ScanDirectory scans a directory for files with given extension
// Files matching any of the excludePatterns are skipped
func ScanDirectory(dir string, ext string, excludePatterns []string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ext {
			// Check if file matches any exclude pattern
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				relPath = filepath.Base(path)
			}

			excluded := false
			for _, pattern := range excludePatterns {
				matched, err := filepath.Match(pattern, relPath)
				if err == nil && matched {
					excluded = true
					break
				}
				// Also try matching against the basename
				matched, err = filepath.Match(pattern, filepath.Base(path))
				if err == nil && matched {
					excluded = true
					break
				}
			}

			if !excluded {
				files = append(files, path)
			}
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
