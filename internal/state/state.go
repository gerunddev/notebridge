package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// FileState represents the state of a single file
type FileState struct {
	MTime      int64  `json:"mtime"`
	Hash       string `json:"hash"`
	PairedWith string `json:"paired_with"`
}

// State represents the sync state
type State struct {
	Files map[string]*FileState `json:"files"`
	IDMap map[string]string     `json:"id_map"` // org-id -> filename
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		Files: make(map[string]*FileState),
		IDMap: make(map[string]string),
	}
}

// Load reads state from the state file
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewState(), nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Files == nil {
		state.Files = make(map[string]*FileState)
	}
	if state.IDMap == nil {
		state.IDMap = make(map[string]string)
	}

	return &state, nil
}

// Save writes state to the state file
func (s *State) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// ComputeHash computes SHA256 hash of a file
func ComputeHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// HasChanged checks if a file has changed since last sync
// Uses hybrid mtime + hash approach
func (s *State) HasChanged(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	mtime := info.ModTime().Unix()

	fileState, exists := s.Files[path]
	if !exists {
		// New file
		return true, nil
	}

	// Fast path: check mtime first
	if mtime == fileState.MTime {
		return false, nil
	}

	// mtime changed, compute hash to check for actual content changes
	hash, err := ComputeHash(path)
	if err != nil {
		return false, err
	}

	return hash != fileState.Hash, nil
}

// Update updates the state for a file
func (s *State) Update(path string, pairedWith string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	hash, err := ComputeHash(path)
	if err != nil {
		return err
	}

	s.Files[path] = &FileState{
		MTime:      info.ModTime().Unix(),
		Hash:       hash,
		PairedWith: pairedWith,
	}

	return nil
}

// GetMTime returns the modification time for a file
func (s *State) GetMTime(path string) time.Time {
	if fileState, exists := s.Files[path]; exists {
		return time.Unix(fileState.MTime, 0)
	}
	return time.Time{}
}
