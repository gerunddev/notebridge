package logger

import (
	"io"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

// Logger wraps charm/log for structured logging
type Logger struct {
	*log.Logger
}

// New creates a new logger with the given output
func New(w io.Writer) *Logger {
	l := log.NewWithOptions(w, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateTime,
	})
	return &Logger{Logger: l}
}

// NewWithLevel creates a logger with a specific level
func NewWithLevel(w io.Writer, level log.Level) *Logger {
	l := log.NewWithOptions(w, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateTime,
		Level:           level,
	})
	return &Logger{Logger: l}
}

// NewFileLogger creates a logger that writes to a file
func NewFileLogger(path string) (*Logger, func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	l := log.NewWithOptions(f, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateTime,
	})

	cleanup := func() {
		f.Close()
	}

	return &Logger{Logger: l}, cleanup, nil
}

// NewMultiLogger creates a logger that writes to multiple outputs
func NewMultiLogger(writers ...io.Writer) *Logger {
	w := io.MultiWriter(writers...)
	return New(w)
}

// Discard returns a logger that discards all output
func Discard() *Logger {
	return New(io.Discard)
}

// SyncStarted logs the start of a sync operation
func (l *Logger) SyncStarted(orgDir, obsidianDir string) {
	l.Info("sync started",
		"org_dir", orgDir,
		"obsidian_dir", obsidianDir)
}

// SyncCompleted logs the completion of a sync operation
func (l *Logger) SyncCompleted(filesProcessed int, errors int, duration time.Duration) {
	l.Info("sync completed",
		"files_synced", filesProcessed,
		"errors", errors,
		"duration", duration.Round(time.Millisecond))
}

// FileSynced logs a successful file sync
func (l *Logger) FileSynced(source, dest, reason string) {
	l.Info("file synced",
		"source", source,
		"dest", dest,
		"reason", reason)
}

// Conflict logs a conflict resolution
func (l *Logger) Conflict(file, winner, reason string) {
	l.Warn("conflict resolved",
		"file", file,
		"winner", winner,
		"reason", reason)
}

// FileError logs an error for a specific file
func (l *Logger) FileError(file string, err error) {
	l.Error("file error",
		"file", file,
		"error", err)
}

// ConversionError logs a conversion error
func (l *Logger) ConversionError(source, dest string, err error) {
	l.Error("conversion failed",
		"source", source,
		"dest", dest,
		"error", err)
}

// StateError logs a state-related error
func (l *Logger) StateError(operation string, err error) {
	l.Error("state error",
		"operation", operation,
		"error", err)
}

// ConfigLoaded logs successful config loading
func (l *Logger) ConfigLoaded(orgDir, obsidianDir string, interval time.Duration) {
	l.Debug("config loaded",
		"org_dir", orgDir,
		"obsidian_dir", obsidianDir,
		"interval", interval)
}

// Skipped logs when a file is skipped
func (l *Logger) Skipped(file, reason string) {
	l.Debug("file skipped",
		"file", file,
		"reason", reason)
}
