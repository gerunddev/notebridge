package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	s := NewState()

	if s.Files == nil {
		t.Error("Files map should be initialized")
	}
	if s.IDMap == nil {
		t.Error("IDMap should be initialized")
	}
	if len(s.Files) != 0 {
		t.Error("Files map should be empty")
	}
	if len(s.IDMap) != 0 {
		t.Error("IDMap should be empty")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create test state
	state := NewState()
	state.Files["test.org"] = &FileState{
		MTime:      123456789,
		Hash:       "sha256:abc123",
		PairedWith: "test.md",
	}
	state.IDMap["uuid-123"] = "test-file"

	// Save
	if err := state.Save(statePath); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load
	loaded, err := Load(statePath)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify
	if len(loaded.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(loaded.Files))
	}
	if len(loaded.IDMap) != 1 {
		t.Errorf("Expected 1 ID mapping, got %d", len(loaded.IDMap))
	}

	fileState := loaded.Files["test.org"]
	if fileState == nil {
		t.Fatal("File state not found")
	}
	if fileState.MTime != 123456789 {
		t.Errorf("MTime mismatch: got %d, want 123456789", fileState.MTime)
	}
	if fileState.Hash != "sha256:abc123" {
		t.Errorf("Hash mismatch: got %s, want sha256:abc123", fileState.Hash)
	}
	if fileState.PairedWith != "test.md" {
		t.Errorf("PairedWith mismatch: got %s, want test.md", fileState.PairedWith)
	}

	if loaded.IDMap["uuid-123"] != "test-file" {
		t.Errorf("IDMap mismatch: got %s, want test-file", loaded.IDMap["uuid-123"])
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "nonexistent.json")

	// Should return empty state, not error
	state, err := Load(statePath)
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}

	if state == nil {
		t.Fatal("State should not be nil")
	}
	if len(state.Files) != 0 {
		t.Error("State should be empty")
	}
}

func TestComputeHash(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write test content
	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute hash
	hash, err := ComputeHash(testFile)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	// Verify format
	if len(hash) == 0 {
		t.Error("Hash should not be empty")
	}
	if hash[:7] != "sha256:" {
		t.Errorf("Hash should start with 'sha256:', got: %s", hash)
	}

	// Compute again - should be same
	hash2, err := ComputeHash(testFile)
	if err != nil {
		t.Fatalf("Second ComputeHash failed: %v", err)
	}
	if hash != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Change content - hash should change
	if err := os.WriteFile(testFile, []byte("Different content"), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}
	hash3, err := ComputeHash(testFile)
	if err != nil {
		t.Fatalf("Third ComputeHash failed: %v", err)
	}
	if hash == hash3 {
		t.Error("Hash should change when content changes")
	}
}

func TestHasChanged(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	content := []byte("Initial content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	state := NewState()

	// New file - should be changed
	changed, err := state.HasChanged(testFile)
	if err != nil {
		t.Fatalf("HasChanged failed: %v", err)
	}
	if !changed {
		t.Error("New file should be marked as changed")
	}

	// Update state
	if err := state.Update(testFile, ""); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Unchanged file - should not be changed
	changed, err = state.HasChanged(testFile)
	if err != nil {
		t.Fatalf("HasChanged failed: %v", err)
	}
	if changed {
		t.Error("Unchanged file should not be marked as changed")
	}

	// Touch file (change mtime but not content)
	time.Sleep(1100 * time.Millisecond) // Ensure mtime changes (1 second resolution on some filesystems)
	newTime := time.Now()
	if err := os.Chtimes(testFile, newTime, newTime); err != nil {
		t.Fatalf("Failed to touch file: %v", err)
	}

	// Should check hash and find no real changes
	changed, err = state.HasChanged(testFile)
	if err != nil {
		t.Fatalf("HasChanged failed after touch: %v", err)
	}
	if changed {
		t.Error("File with only mtime change should not be marked as changed")
	}

	// Actually change content
	time.Sleep(1100 * time.Millisecond) // Ensure mtime changes
	if err := os.WriteFile(testFile, []byte("New content"), 0644); err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	// Compute hash of new content for debugging
	newHash, _ := ComputeHash(testFile)
	oldHash := state.Files[testFile].Hash
	info, _ := os.Stat(testFile)
	newMTime := info.ModTime().Unix()
	oldMTime := state.Files[testFile].MTime

	// Should detect change
	changed, err = state.HasChanged(testFile)
	if err != nil {
		t.Fatalf("HasChanged failed after content change: %v", err)
	}
	if !changed {
		t.Errorf("File with content change should be marked as changed\nOld hash: %s\nNew hash: %s\nOld mtime: %d\nNew mtime: %d",
			oldHash, newHash, oldMTime, newMTime)
	}
}

func TestUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("Test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	state := NewState()

	// Update state
	pairedWith := "test.md"
	if err := state.Update(testFile, pairedWith); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify state was updated
	fileState := state.Files[testFile]
	if fileState == nil {
		t.Fatal("File state not found after update")
	}
	if fileState.MTime == 0 {
		t.Error("MTime should be set")
	}
	if fileState.Hash == "" {
		t.Error("Hash should be set")
	}
	if fileState.PairedWith != pairedWith {
		t.Errorf("PairedWith mismatch: got %s, want %s", fileState.PairedWith, pairedWith)
	}
}

func TestGetMTime(t *testing.T) {
	state := NewState()

	// Non-existent file - should return zero time
	mtime := state.GetMTime("nonexistent.txt")
	if !mtime.IsZero() {
		t.Error("MTime for non-existent file should be zero")
	}

	// Add file state
	state.Files["test.txt"] = &FileState{
		MTime: 1234567890,
		Hash:  "sha256:test",
	}

	// Should return correct time
	mtime = state.GetMTime("test.txt")
	if mtime.Unix() != 1234567890 {
		t.Errorf("MTime mismatch: got %d, want 1234567890", mtime.Unix())
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use nested path to test directory creation
	statePath := filepath.Join(tmpDir, "nested", "dir", "state.json")

	state := NewState()
	state.Files["test.org"] = &FileState{
		MTime: 123,
		Hash:  "sha256:test",
	}

	// Should create all parent directories
	if err := state.Save(statePath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file was not created")
	}

	// Verify directory was created
	dir := filepath.Dir(statePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Parent directory was not created")
	}
}
