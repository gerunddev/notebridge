package commands

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/notebridge/internal/config"
	"github.com/gerunddev/notebridge/internal/daemon"
	"github.com/gerunddev/notebridge/internal/logger"
	"github.com/gerunddev/notebridge/internal/state"
	"github.com/gerunddev/notebridge/internal/styles"
	"github.com/gerunddev/notebridge/internal/sync"
	"github.com/gerunddev/notebridge/internal/tui"
)

// Start starts the daemon in background mode
func Start(args []string) {
	successStyle := styles.SuccessStyle
	errorStyle := styles.ErrorStyle
	dimStyle := styles.DimStyle

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

// Stop stops the running daemon
func Stop() {
	successStyle := styles.SuccessStyle
	errorStyle := styles.ErrorStyle
	dimStyle := styles.DimStyle

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

// Daemon runs the daemon in foreground mode with TUI
func Daemon(args []string) {
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
	st, err := state.Load(config.StateFilePath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
		os.Exit(1)
	}

	// Write PID file
	if err := daemon.WritePID(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PID file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := daemon.RemovePID(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove PID file on shutdown: %v\n", err)
		}
	}()

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
		if err := st.Save(config.StateFilePath()); err != nil {
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
				if err := st.Save(config.StateFilePath()); err != nil {
					log.Error("failed to save state", "error", err)
				}

			case <-stopChan:
				log.Info("sync loop stopping")
				// Save final state
				if err := st.Save(config.StateFilePath()); err != nil {
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
				logLines, lastSync, filesSynced := ParseLogFile(cfg.LogFile, 20)
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
		fmt.Println(styles.ErrorStyle.Render("✗ Error: " + err.Error()))
		stopChan <- true
		<-doneChan // Wait for sync loop to finish
		os.Exit(1)
	}

	// TUI exited normally (user pressed 'q'), stop sync loop gracefully
	stopChan <- true
	<-doneChan // Wait for sync loop to finish
	log.Info("daemon shutdown complete")
}
