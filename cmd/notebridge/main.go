package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gerunddev/notebridge/internal/config"
	"github.com/gerunddev/notebridge/internal/daemon"
	"github.com/gerunddev/notebridge/internal/logger"
	"github.com/gerunddev/notebridge/internal/state"
	"github.com/gerunddev/notebridge/internal/sync"
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
		handleSync()
	case "status":
		handleStatus()
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
  sync        One-shot manual sync
  status      Display sync state
  version     Show version information
  help        Show this help message

Examples:
  notebridge start --interval 30s
  notebridge stop
  notebridge sync
  notebridge status

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

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run sync loop
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

		case sig := <-sigChan:
			log.Info("received signal, shutting down", "signal", sig.String())
			// Save final state
			if err := st.Save(cfg.StateFile); err != nil {
				log.Error("failed to save state on shutdown", "error", err)
			}
			return
		}
	}
}

func handleSync() {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	fmt.Println(titleStyle.Render("NoteBridge Sync"))
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

	fmt.Printf("%s → %s\n", dimStyle.Render(cfg.OrgDir), dimStyle.Render(cfg.ObsidianDir))
	fmt.Println()

	// Create syncer and configure logging
	syncer := sync.NewSyncer(cfg, st)

	// Set up log file if configured
	if cfg.LogFile != "" {
		l, cleanup, err := logger.NewFileLogger(cfg.LogFile)
		if err == nil {
			defer cleanup()
			syncer.SetLogger(l)
		}
	}
	result, err := syncer.Sync()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Sync failed: " + err.Error()))
		os.Exit(1)
	}

	// Report results
	if result.FilesProcessed == 0 && len(result.Errors) == 0 {
		fmt.Println(successStyle.Render("✓ Nothing to sync"))
	} else if len(result.Errors) == 0 {
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ Synced %d file(s)", result.FilesProcessed)))
	} else {
		fmt.Printf("%s, %s\n",
			successStyle.Render(fmt.Sprintf("✓ Synced %d file(s)", result.FilesProcessed)),
			errorStyle.Render(fmt.Sprintf("%d error(s)", len(result.Errors))))
		for _, e := range result.Errors {
			fmt.Printf("  %s\n", errorStyle.Render("• "+e.Error()))
		}
	}

	duration := result.EndTime.Sub(result.StartTime)
	fmt.Println(dimStyle.Render(fmt.Sprintf("\nCompleted in %v", duration.Round(time.Millisecond))))

	// Save state
	if err := st.Save(cfg.StateFile); err != nil {
		fmt.Println(errorStyle.Render("✗ Error saving state: " + err.Error()))
		os.Exit(1)
	}
}

func handleStatus() {
	// Define styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Configuration not found"))
		fmt.Printf("  Run 'notebridge' to create a config at %s\n", config.ConfigPath())
		return
	}

	// Load state
	st, err := state.Load(cfg.StateFile)
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Error loading state: " + err.Error()))
		return
	}

	// Print header
	fmt.Println(titleStyle.Render("NoteBridge Status"))
	fmt.Println()

	// Configuration info
	fmt.Println(labelStyle.Render("Configuration"))
	fmt.Printf("  Org directory:      %s\n", valueStyle.Render(cfg.OrgDir))
	fmt.Printf("  Obsidian directory: %s\n", valueStyle.Render(cfg.ObsidianDir))
	fmt.Printf("  Sync interval:      %s\n", valueStyle.Render(cfg.Interval.String()))
	fmt.Println()

	// Scan directories
	orgFiles, err := sync.ScanDirectory(cfg.OrgDir, ".org")
	if err != nil {
		fmt.Printf("  %s\n", errorStyle.Render("✗ Cannot scan org directory: "+err.Error()))
		orgFiles = []string{}
	}

	mdFiles, err := sync.ScanDirectory(cfg.ObsidianDir, ".md")
	if err != nil {
		fmt.Printf("  %s\n", errorStyle.Render("✗ Cannot scan obsidian directory: "+err.Error()))
		mdFiles = []string{}
	}

	// Count tracked files
	trackedCount := len(st.Files)

	fmt.Println(labelStyle.Render("Files"))
	fmt.Printf("  Org files:      %s\n", valueStyle.Render(fmt.Sprintf("%d", len(orgFiles))))
	fmt.Printf("  Markdown files: %s\n", valueStyle.Render(fmt.Sprintf("%d", len(mdFiles))))
	fmt.Printf("  Tracked pairs:  %s\n", valueStyle.Render(fmt.Sprintf("%d", trackedCount/2)))
	fmt.Println()

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

	fmt.Println(labelStyle.Render("Pending Changes"))
	if len(pendingOrg) == 0 && len(pendingMd) == 0 {
		fmt.Printf("  %s\n", successStyle.Render("✓ No pending changes"))
	} else {
		if len(pendingOrg) > 0 {
			fmt.Printf("  %s\n", warningStyle.Render(fmt.Sprintf("● %d org file(s) changed", len(pendingOrg))))
			for _, f := range pendingOrg {
				if len(pendingOrg) <= 5 {
					fmt.Printf("    - %s\n", f)
				}
			}
			if len(pendingOrg) > 5 {
				fmt.Printf("    ... and %d more\n", len(pendingOrg)-5)
			}
		}
		if len(pendingMd) > 0 {
			fmt.Printf("  %s\n", warningStyle.Render(fmt.Sprintf("● %d markdown file(s) changed", len(pendingMd))))
			for _, f := range pendingMd {
				if len(pendingMd) <= 5 {
					fmt.Printf("    - %s\n", f)
				}
			}
			if len(pendingMd) > 5 {
				fmt.Printf("    ... and %d more\n", len(pendingMd)-5)
			}
		}
	}
	fmt.Println()

	// Check for potential conflicts (both sides changed)
	var conflicts []string
	for _, orgPath := range orgFiles {
		// Get corresponding md path
		relPath, _ := filepath.Rel(cfg.OrgDir, orgPath)
		baseName := relPath[:len(relPath)-4] // Remove .org
		mdPath := filepath.Join(cfg.ObsidianDir, baseName+".md")

		orgChanged, _ := st.HasChanged(orgPath)
		mdChanged, _ := st.HasChanged(mdPath)

		if orgChanged && mdChanged {
			conflicts = append(conflicts, baseName)
		}
	}

	fmt.Println(labelStyle.Render("Conflicts"))
	if len(conflicts) == 0 {
		fmt.Printf("  %s\n", successStyle.Render("✓ No conflicts"))
	} else {
		fmt.Printf("  %s\n", errorStyle.Render(fmt.Sprintf("✗ %d potential conflict(s)", len(conflicts))))
		for _, f := range conflicts {
			fmt.Printf("    - %s\n", f)
		}
	}
	fmt.Println()

	// ID mappings
	fmt.Println(labelStyle.Render("ID Mappings"))
	fmt.Printf("  %s\n", valueStyle.Render(fmt.Sprintf("%d org-roam IDs tracked", len(st.IDMap))))
}

