package commands

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ParseLogFile reads the last N lines from the log file and extracts sync info
func ParseLogFile(logPath string, maxLines int) ([]string, time.Time, int) {
	content, err := os.ReadFile(logPath)
	if err != nil {
		return []string{"Unable to read log file"}, time.Time{}, 0
	}

	lines := strings.Split(string(content), "\n")

	// Get last N lines
	startIdx := 0
	if len(lines) > maxLines {
		startIdx = len(lines) - maxLines
	}
	recentLines := lines[startIdx:]

	// Parse for last sync time and files synced
	var lastSync time.Time
	filesSynced := 0

	// Look for most recent "sync completed" line
	for i := len(recentLines) - 1; i >= 0; i-- {
		line := recentLines[i]
		if strings.Contains(line, "sync completed") {
			// Try to parse timestamp from start of line
			// Format: 2025-11-27 14:11:57 INFO sync completed
			if len(line) > 19 {
				timeStr := line[:19]
				if t, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
					lastSync = t
				}
			}

			// Try to extract files_synced count (best effort - ignore errors)
			if idx := strings.Index(line, "files_synced="); idx != -1 {
				_, _ = fmt.Sscanf(line[idx:], "files_synced=%d", &filesSynced) //nolint:errcheck // best effort parsing
			}
			break
		}
	}

	return recentLines, lastSync, filesSynced
}
