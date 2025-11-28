package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerunddev/notebridge/internal/config"
	"github.com/gerunddev/notebridge/internal/daemon"
	"github.com/gerunddev/notebridge/internal/logger"
	"github.com/gerunddev/notebridge/internal/state"
	"github.com/gerunddev/notebridge/internal/sync"
	"github.com/gerunddev/notebridge/internal/tui"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "start":
		handleStart(os.Args[2:])
	case "daemon":
		handleDaemon(os.Args[2:])
	case "stop":
		handleStop()
	case "sync":
		handleSync(os.Args[2:])
	case "status":
		handleStatus()
	case "browse", "files":
		handleBrowse()
	case "dashboard", "watch":
		handleDashboard()
	case "install":
		handleInstall()
	case "version", "-v", "--version":
		fmt.Printf("notebridge v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	usage := `notebridge - Bidirectional sync between org-roam and Obsidian

Usage:
  notebridge <command> [options]

Commands:
  start       Start daemon in background
  daemon      Run daemon in foreground (for debugging)
  stop        Stop the running daemon
  sync        One-shot manual sync (use --dry-run to preview)
  status      Display sync state
  browse      Browse all tracked files
  dashboard   Live daemon status dashboard
  install     Generate system service files
  version     Show version information
  help        Show this help message

Examples:
  notebridge start --interval 30s
  notebridge stop
  notebridge sync
  notebridge sync --dry-run
  notebridge status
  notebridge browse
  notebridge dashboard
  notebridge install

Configuration:
  Config file: ~/.config/notebridge/config.json
  State file:  ~/.config/notebridge/state.json

For more information, visit: https://github.com/gerunddev/notebridge
`
	fmt.Print(usage)
}

func handleStart(args []string) {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Check if already running
	running, pid, _ := daemon.IsRunning()
	if running {
		fmt.Println(errorStyle.Render(fmt.Sprintf("✗ Daemon already running with PID %d", pid)))
		os.Exit(1)
	}

	// Parse interval from args or use default
	intervalArg := ""
	for i, arg := range args {
		if arg == "--interval" && i+1 < len(args) {
			intervalArg = args[i+1]
			break
		}
	}

	// Build args for daemon process
	daemonArgs := []string{"daemon"}
	if intervalArg != "" {
		daemonArgs = append(daemonArgs, "--interval", intervalArg)
	}

	// Start daemon in background
	if err := daemon.Daemonize(daemonArgs); err != nil {
		fmt.Println(errorStyle.Render("✗ Failed to start daemon: " + err.Error()))
		os.Exit(1)
	}

	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	// Verify it started
	running, pid, _ = daemon.IsRunning()
	if running {
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ Daemon started with PID %d", pid)))
		fmt.Println(dimStyle.Render("  Run 'notebridge dashboard' to monitor the daemon"))
	} else {
		fmt.Println(errorStyle.Render("✗ Daemon failed to start"))
		os.Exit(1)
	}
}

func handleStop() {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Check if running
	running, pid, _ := daemon.IsRunning()
	if !running {
		fmt.Println(dimStyle.Render("Daemon is not running"))
		return
	}

	fmt.Printf("Stopping daemon (PID %d)...\n", pid)

	// Stop the daemon
	if err := daemon.Stop(); err != nil {
		fmt.Println(errorStyle.Render("✗ Failed to stop daemon: " + err.Error()))
		os.Exit(1)
	}

	// Wait for it to stop
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		running, _, _ = daemon.IsRunning()
		if !running {
			break
		}
	}

	if running {
		fmt.Println(errorStyle.Render("✗ Daemon did not stop gracefully"))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("✓ Daemon stopped"))
}

func handleDaemon(args []string) {
	// Parse --interval flag
	interval := 30 * time.Second
	for i, arg := range args {
		if arg == "--interval" && i+1 < len(args) {
			var err error
			interval, err = time.ParseDuration(args[i+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Invalid interval: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override interval if specified
	if interval != 30*time.Second {
		cfg.Interval = interval
	}

	// Load state
	st, err := state.Load(cfg.StateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
		os.Exit(1)
	}

	// Write PID file
	if err := daemon.WritePID(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PID file: %v\n", err)
		os.Exit(1)
	}
	defer daemon.RemovePID()

	// Set up structured logging
	var log *logger.Logger
	if cfg.LogFile != "" {
		l, cleanup, err := logger.NewFileLogger(cfg.LogFile)
		if err == nil {
			defer cleanup()
			log = l
		} else {
			log = logger.Discard()
		}
	} else {
		log = logger.Discard()
	}

	log.Info("daemon started",
		"pid", os.Getpid(),
		"interval", cfg.Interval)

	// Create syncer
	syncer := sync.NewSyncer(cfg, st)
	syncer.SetLogger(log)

	// Channels for sync loop coordination
	stopChan := make(chan bool, 1)
	doneChan := make(chan bool, 1)

	// Run sync loop in background goroutine
	go func() {
		defer func() {
			doneChan <- true
		}()

		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		// Initial sync
		result, err := syncer.Sync()
		if err != nil {
			log.Error("initial sync failed", "error", err)
		} else {
			log.Info("initial sync completed",
				"files_synced", result.FilesProcessed,
				"errors", len(result.Errors))
		}

		// Save state after initial sync
		if err := st.Save(cfg.StateFile); err != nil {
			log.Error("failed to save state", "error", err)
		}

		// Periodic sync loop
		for {
			select {
			case <-ticker.C:
				result, err := syncer.Sync()
				if err != nil {
					log.Error("sync failed", "error", err)
					continue
				}

				log.Debug("sync tick completed",
					"files_synced", result.FilesProcessed,
					"errors", len(result.Errors))

				// Save state after each sync
				if err := st.Save(cfg.StateFile); err != nil {
					log.Error("failed to save state", "error", err)
				}

			case <-stopChan:
				log.Info("sync loop stopping")
				// Save final state
				if err := st.Save(cfg.StateFile); err != nil {
					log.Error("failed to save state on shutdown", "error", err)
				}
				return
			}
		}
	}()

	// Run TUI dashboard in main thread
	m := tui.InitDaemonModel()
	p := tea.NewProgram(m, tea.WithInput(os.Stdin))

	// Function to gather and send daemon data
	sendDaemonData := func() {
		// Check daemon status
		running, pid, startTime := daemon.IsRunning()

		data := &tui.DaemonData{
			Running:   running,
			PID:       pid,
			StartTime: startTime,
		}

		if running {
			// Parse log file for recent activity
			if cfg.LogFile != "" {
				logLines, lastSync, filesSynced := parseLogFile(cfg.LogFile, 20)
				data.LogLines = logLines
				data.LastSyncTime = lastSync
				data.FilesSynced = filesSynced
			}
		}

		p.Send(tui.DaemonMsg{
			Data: data,
			Err:  nil,
		})
	}

	// Set up periodic refresh for TUI
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		// Send initial data
		sendDaemonData()

		// Refresh periodically
		for range ticker.C {
			sendDaemonData()
		}
	}()

	// Run the TUI program
	if _, err := p.Run(); err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		fmt.Println(errorStyle.Render("✗ Error: " + err.Error()))
		stopChan <- true
		<-doneChan // Wait for sync loop to finish
		os.Exit(1)
	}

	// TUI exited normally (user pressed 'q'), stop sync loop gracefully
	stopChan <- true
	<-doneChan // Wait for sync loop to finish
	log.Info("daemon shutdown complete")
}

func handleSync(args []string) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Parse --dry-run flag
	dryRun := false
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
			break
		}
	}

	if dryRun {
		fmt.Println(titleStyle.Render("NoteBridge Sync (DRY RUN)"))
	} else {
		fmt.Println(titleStyle.Render("NoteBridge Sync"))
	}
	fmt.Println()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Error loading config: " + err.Error()))
		os.Exit(1)
	}

	// Load state
	st, err := state.Load(cfg.StateFile)
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Error loading state: " + err.Error()))
		os.Exit(1)
	}

	fmt.Printf("%s ↔ %s\n", dimStyle.Render(cfg.OrgDir), dimStyle.Render(cfg.ObsidianDir))
	if dryRun {
		fmt.Println(dimStyle.Render("(dry run - no files will be modified)"))
	}
	fmt.Println()

	// Create syncer and configure logging
	syncer := sync.NewSyncer(cfg, st)
	syncer.DryRun = dryRun

	// Set up log file if configured
	if cfg.LogFile != "" {
		l, cleanup, err := logger.NewFileLogger(cfg.LogFile)
		if err == nil {
			defer cleanup()
			syncer.SetLogger(l)
		}
	}

	// Initialize Bubble Tea program
	m := tui.InitSyncModel()
	p := tea.NewProgram(m, tea.WithInput(os.Stdin))

	// Run sync in goroutine and send result to program
	go func() {
		startTime := time.Now()
		result, err := syncer.Sync()
		duration := time.Since(startTime)

		var tuiResult *tui.SyncResult
		if result != nil {
			tuiResult = &tui.SyncResult{
				FilesProcessed: result.FilesProcessed,
				Errors:         result.Errors,
				Duration:       duration,
				Success:        err == nil,
			}
		}

		p.Send(tui.SyncMsg{
			Result: tuiResult,
			Err:    err,
		})
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Println(errorStyle.Render("✗ Error: " + err.Error()))
		os.Exit(1)
	}

	// Save state
	if err := st.Save(cfg.StateFile); err != nil {
		fmt.Println(errorStyle.Render("✗ Error saving state: " + err.Error()))
		os.Exit(1)
	}
}

func handleStatus() {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Configuration not found"))
		os.Exit(1)
	}

	// Load state
	st, err := state.Load(cfg.StateFile)
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Error loading state: " + err.Error()))
		os.Exit(1)
	}

	// Create syncer for conflict resolution
	syncer := sync.NewSyncer(cfg, st)

	// Create resolution function
	resolveFunc := func(orgPath, mdPath, direction string) error {
		err := syncer.SyncFileWithResolution(orgPath, mdPath, direction)
		if err == nil {
			// Save state after successful resolution
			if saveErr := st.Save(cfg.StateFile); saveErr != nil {
				return fmt.Errorf("sync succeeded but failed to save state: %w", saveErr)
			}
		}
		return err
	}

	// Bubble Tea program (will be set after creating sendStatusData)
	var p *tea.Program

	// Function to gather and send status data
	sendStatusData := func() {
		// Reload state to get latest changes
		st, err := state.Load(cfg.StateFile)
		if err != nil {
			p.Send(tui.StatusMsg{
				Data: nil,
				Err:  fmt.Errorf("error loading state: %w", err),
			})
			return
		}

		// Scan directories
		orgFiles, err := sync.ScanDirectory(cfg.OrgDir, ".org", cfg.ExcludePatterns)
		if err != nil {
			orgFiles = []string{}
		}

		mdFiles, err := sync.ScanDirectory(cfg.ObsidianDir, ".md", cfg.ExcludePatterns)
		if err != nil {
			mdFiles = []string{}
		}

		// Count tracked files
		trackedCount := len(st.Files)

		// Check for pending changes
		var pendingOrg, pendingMd []string

		for _, orgPath := range orgFiles {
			changed, err := st.HasChanged(orgPath)
			if err == nil && changed {
				relPath, _ := filepath.Rel(cfg.OrgDir, orgPath)
				pendingOrg = append(pendingOrg, relPath)
			}
		}

		for _, mdPath := range mdFiles {
			changed, err := st.HasChanged(mdPath)
			if err == nil && changed {
				relPath, _ := filepath.Rel(cfg.ObsidianDir, mdPath)
				pendingMd = append(pendingMd, relPath)
			}
		}

		// Check for potential conflicts (both sides changed)
		// Build sets for faster lookup
		pendingOrgSet := make(map[string]bool)
		for _, f := range pendingOrg {
			baseName := strings.TrimSuffix(f, ".org")
			pendingOrgSet[baseName] = true
		}

		pendingMdSet := make(map[string]bool)
		for _, f := range pendingMd {
			baseName := strings.TrimSuffix(f, ".md")
			pendingMdSet[baseName] = true
		}

		// Find conflicts (files that appear in both pending sets)
		var conflicts []string
		for baseName := range pendingOrgSet {
			if pendingMdSet[baseName] {
				conflicts = append(conflicts, baseName)
			}
		}

		// Send status data to UI
		p.Send(tui.StatusMsg{
			Data: &tui.StatusData{
				OrgDir:       cfg.OrgDir,
				ObsidianDir:  cfg.ObsidianDir,
				Interval:     cfg.Interval,
				OrgFileCount: len(orgFiles),
				MdFileCount:  len(mdFiles),
				TrackedPairs: trackedCount / 2,
				PendingOrg:   pendingOrg,
				PendingMd:    pendingMd,
				Conflicts:    conflicts,
				IDMapCount:   len(st.IDMap),
				Scanning:     false,
			},
			Err: nil,
		})
	}

	// Initialize Bubble Tea program with all functions
	m := tui.InitStatusModel(cfg.OrgDir, cfg.ObsidianDir, resolveFunc, sendStatusData)
	p = tea.NewProgram(m, tea.WithInput(os.Stdin))

	// Send initial status data
	go sendStatusData()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Println(errorStyle.Render("✗ Error: " + err.Error()))
		os.Exit(1)
	}
}

func handleBrowse() {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Configuration not found"))
		os.Exit(1)
	}

	// Load state
	st, err := state.Load(cfg.StateFile)
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Error loading state: " + err.Error()))
		os.Exit(1)
	}

	// Create syncer for conflict resolution
	syncer := sync.NewSyncer(cfg, st)

	// Create resolution function
	resolveFunc := func(orgPath, mdPath, direction string) error {
		err := syncer.SyncFileWithResolution(orgPath, mdPath, direction)
		if err == nil {
			// Save state after successful resolution
			if saveErr := st.Save(cfg.StateFile); saveErr != nil {
				return fmt.Errorf("sync succeeded but failed to save state: %w", saveErr)
			}
		}
		return err
	}

	// Bubble Tea program (will be set after creating sendBrowseData)
	var p *tea.Program

	// Function to gather and send browse data
	sendBrowseData := func() {
		// Reload state to get latest changes
		st, err := state.Load(cfg.StateFile)
		if err != nil {
			p.Send(tui.BrowseMsg{
				Data: nil,
				Err:  fmt.Errorf("error loading state: %w", err),
			})
			return
		}
		// Get all tracked files from state
		var files []tui.FileInfo

		// Build map of org files
		orgFiles, _ := sync.ScanDirectory(cfg.OrgDir, ".org", cfg.ExcludePatterns)
		orgFileSet := make(map[string]bool)
		for _, orgPath := range orgFiles {
			relPath, _ := filepath.Rel(cfg.OrgDir, orgPath)
			baseName := strings.TrimSuffix(relPath, ".org")
			orgFileSet[baseName] = true
		}

		// Build map of md files
		mdFiles, _ := sync.ScanDirectory(cfg.ObsidianDir, ".md", cfg.ExcludePatterns)
		mdFileSet := make(map[string]bool)
		for _, mdPath := range mdFiles {
			relPath, _ := filepath.Rel(cfg.ObsidianDir, mdPath)
			baseName := strings.TrimSuffix(relPath, ".md")
			mdFileSet[baseName] = true
		}

		// Combine all unique basenames
		allFiles := make(map[string]bool)
		for baseName := range orgFileSet {
			allFiles[baseName] = true
		}
		for baseName := range mdFileSet {
			allFiles[baseName] = true
		}

		// Build file info for each
		for baseName := range allFiles {
			orgPath := filepath.Join(cfg.OrgDir, baseName+".org")
			mdPath := filepath.Join(cfg.ObsidianDir, baseName+".md")

			hasOrg := orgFileSet[baseName]
			hasMd := mdFileSet[baseName]

			// Determine status
			var status, statusIcon string
			if !hasOrg && hasMd {
				status = "md only"
				statusIcon = "←"
			} else if hasOrg && !hasMd {
				status = "org only"
				statusIcon = "→"
			} else {
				// Both exist, check if changed
				orgChanged, _ := st.HasChanged(orgPath)
				mdChanged, _ := st.HasChanged(mdPath)

				if orgChanged && mdChanged {
					status = "conflict"
					statusIcon = "⚠"
				} else if orgChanged {
					status = "org pending"
					statusIcon = "→"
				} else if mdChanged {
					status = "md pending"
					statusIcon = "←"
				} else {
					status = "synced"
					statusIcon = "✓"
				}
			}

			files = append(files, tui.FileInfo{
				BaseName:   baseName,
				OrgPath:    baseName + ".org",
				MdPath:     baseName + ".md",
				Status:     status,
				StatusIcon: statusIcon,
				HasOrgFile: hasOrg,
				HasMdFile:  hasMd,
			})
		}

		// Send browse data to UI
		p.Send(tui.BrowseMsg{
			Data: &tui.BrowseData{
				Files: files,
			},
			Err: nil,
		})
	}

	// Initialize Bubble Tea program with all functions
	m := tui.InitBrowseModel(cfg.OrgDir, cfg.ObsidianDir, resolveFunc, sendBrowseData)
	p = tea.NewProgram(m, tea.WithInput(os.Stdin))

	// Send initial browse data
	go sendBrowseData()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Println(errorStyle.Render("✗ Error: " + err.Error()))
		os.Exit(1)
	}
}

func handleDashboard() {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Configuration not found"))
		os.Exit(1)
	}

	// Initialize Bubble Tea program
	m := tui.InitDaemonModel()
	p := tea.NewProgram(m, tea.WithInput(os.Stdin))

	// Function to gather and send daemon data
	sendDaemonData := func() {
		// Check daemon status
		running, pid, startTime := daemon.IsRunning()

		data := &tui.DaemonData{
			Running:   running,
			PID:       pid,
			StartTime: startTime,
		}

		if running {
			// Parse log file for recent activity
			if cfg.LogFile != "" {
				logLines, lastSync, filesSynced := parseLogFile(cfg.LogFile, 20)
				data.LogLines = logLines
				data.LastSyncTime = lastSync
				data.FilesSynced = filesSynced
			}
		}

		p.Send(tui.DaemonMsg{
			Data: data,
			Err:  nil,
		})
	}

	// Set up periodic refresh
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		// Send initial data
		sendDaemonData()

		// Refresh periodically
		for range ticker.C {
			sendDaemonData()
		}
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Println(errorStyle.Render("✗ Error: " + err.Error()))
		os.Exit(1)
	}
}

// parseLogFile reads the last N lines from the log file and extracts sync info
func parseLogFile(logPath string, maxLines int) ([]string, time.Time, int) {
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

			// Try to extract files_synced count
			if idx := strings.Index(line, "files_synced="); idx != -1 {
				fmt.Sscanf(line[idx:], "files_synced=%d", &filesSynced)
			}
			break
		}
	}

	return recentLines, lastSync, filesSynced
}

// handleInstall generates system service files for daemon auto-start
func handleInstall() {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	fmt.Println(titleStyle.Render("NoteBridge Install"))
	fmt.Println()

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Failed to get home directory: " + err.Error()))
		os.Exit(1)
	}

	// Get the full path to the notebridge binary
	execPath, err := os.Executable()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Failed to get executable path: " + err.Error()))
		os.Exit(1)
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: Generate launchd plist
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.notebridge.plist")
		plistDir := filepath.Dir(plistPath)

		// Create LaunchAgents directory if it doesn't exist
		if err := os.MkdirAll(plistDir, 0755); err != nil {
			fmt.Println(errorStyle.Render("✗ Failed to create LaunchAgents directory: " + err.Error()))
			os.Exit(1)
		}

		plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.notebridge</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>daemon</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/tmp/notebridge.out.log</string>
	<key>StandardErrorPath</key>
	<string>/tmp/notebridge.err.log</string>
</dict>
</plist>`, execPath)

		if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
			fmt.Println(errorStyle.Render("✗ Failed to write plist file: " + err.Error()))
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("✓ Service file created: " + plistPath))
		fmt.Println()
		fmt.Println("To enable the service:")
		fmt.Println(dimStyle.Render("  launchctl load " + plistPath))
		fmt.Println()
		fmt.Println("To disable the service:")
		fmt.Println(dimStyle.Render("  launchctl unload " + plistPath))

	case "linux":
		// Linux: Generate systemd user service
		servicePath := filepath.Join(home, ".config", "systemd", "user", "notebridge.service")
		serviceDir := filepath.Dir(servicePath)

		// Create systemd user directory if it doesn't exist
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			fmt.Println(errorStyle.Render("✗ Failed to create systemd user directory: " + err.Error()))
			os.Exit(1)
		}

		serviceContent := fmt.Sprintf(`[Unit]
Description=NoteBridge - Org-roam and Obsidian bidirectional sync
After=network.target

[Service]
Type=simple
ExecStart=%s daemon
Restart=always
RestartSec=10

[Install]
WantedBy=default.target`, execPath)

		if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
			fmt.Println(errorStyle.Render("✗ Failed to write service file: " + err.Error()))
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("✓ Service file created: " + servicePath))
		fmt.Println()
		fmt.Println("To enable the service:")
		fmt.Println(dimStyle.Render("  systemctl --user daemon-reload"))
		fmt.Println(dimStyle.Render("  systemctl --user enable notebridge.service"))
		fmt.Println(dimStyle.Render("  systemctl --user start notebridge.service"))
		fmt.Println()
		fmt.Println("To disable the service:")
		fmt.Println(dimStyle.Render("  systemctl --user stop notebridge.service"))
		fmt.Println(dimStyle.Render("  systemctl --user disable notebridge.service"))

	default:
		fmt.Println(errorStyle.Render("✗ Unsupported operating system: " + runtime.GOOS))
		fmt.Println("Supported platforms: macOS (darwin), Linux")
		os.Exit(1)
	}
}
