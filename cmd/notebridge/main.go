package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gerunddev/notebridge/internal/config"
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
	case "daemon":
		handleDaemon(os.Args[2:])
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
  daemon      Run background sync loop
  sync        One-shot manual sync
  status      Display sync state
  version     Show version information
  help        Show this help message

Examples:
  notebridge daemon --interval 30s
  notebridge sync
  notebridge status

Configuration:
  Config file: ~/.notebridge/config.json
  State file:  ~/.notebridge/state.json

For more information, visit: https://github.com/gerunddev/notebridge
`
	fmt.Print(usage)
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

	fmt.Printf("Starting daemon with %v interval...\n", interval)

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

	// Create syncer
	syncer := sync.NewSyncer(cfg, st)

	// Run sync loop
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	fmt.Println("Daemon started. Press Ctrl+C to stop.")

	// Initial sync
	runSync(syncer, st, cfg)

	// Periodic sync
	for range ticker.C {
		runSync(syncer, st, cfg)
	}
}

func handleSync() {
	fmt.Println("Running one-shot sync...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Load state
	st, err := state.Load(cfg.StateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
		os.Exit(1)
	}

	// Create syncer and run
	syncer := sync.NewSyncer(cfg, st)
	runSync(syncer, st, cfg)
}

func handleStatus() {
	fmt.Println("Sync Status:")
	fmt.Println("(status display not yet implemented)")

	// TODO: Implement status display
	// - Last sync time
	// - Pending changes
	// - Recent errors
	// - Files in conflict
}

func runSync(syncer *sync.Syncer, st *state.State, cfg *config.Config) {
	result, err := syncer.Sync()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Sync error: %v\n", err)
		return
	}

	fmt.Println(result.String())

	// Save state
	if err := st.Save(cfg.StateFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving state: %v\n", err)
	}
}
