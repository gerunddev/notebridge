package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
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
