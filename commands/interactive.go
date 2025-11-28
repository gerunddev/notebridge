package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/notebridge/config"
	"github.com/gerunddev/notebridge/daemon"
	"github.com/gerunddev/notebridge/logger"
	"github.com/gerunddev/notebridge/state"
	"github.com/gerunddev/notebridge/styles"
	"github.com/gerunddev/notebridge/sync"
	"github.com/gerunddev/notebridge/tui"
)

// Sync performs a one-shot sync operation
func Sync(args []string) {
	titleStyle := styles.TitleStyle
	errorStyle := styles.ErrorStyle
	dimStyle := styles.DimStyle

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
	st, err := state.Load(config.StateFilePath())
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
	if err := st.Save(config.StateFilePath()); err != nil {
		fmt.Println(errorStyle.Render("✗ Error saving state: " + err.Error()))
		os.Exit(1)
	}
}

// Status displays the current sync status
func Status() {
	errorStyle := styles.ErrorStyle

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Configuration not found"))
		os.Exit(1)
	}

	// Load state
	st, err := state.Load(config.StateFilePath())
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
			if saveErr := st.Save(config.StateFilePath()); saveErr != nil {
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
		st, err := state.Load(config.StateFilePath())
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

// Browse shows all tracked files in an interactive browser
func Browse() {
	errorStyle := styles.ErrorStyle

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Configuration not found"))
		os.Exit(1)
	}

	// Load state
	st, err := state.Load(config.StateFilePath())
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
			if saveErr := st.Save(config.StateFilePath()); saveErr != nil {
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
		st, err := state.Load(config.StateFilePath())
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
					status = "org → md"
					statusIcon = "✗"
				} else if mdChanged {
					status = "md → org"
					statusIcon = "✗"
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
	m := tui.InitBrowseModel(cfg.OrgDir, cfg.ObsidianDir, st, resolveFunc, sendBrowseData)
	p = tea.NewProgram(m, tea.WithInput(os.Stdin))

	// Send initial browse data
	go sendBrowseData()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Println(errorStyle.Render("✗ Error: " + err.Error()))
		os.Exit(1)
	}
}

// Dashboard displays the live daemon status
func Dashboard() {
	errorStyle := styles.ErrorStyle

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
