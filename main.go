package main

import (
	"fmt"
	"os"

	"github.com/gerunddev/notebridge/internal/commands"
	"github.com/gerunddev/notebridge/internal/config"
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
		commands.Start(os.Args[2:])
	case "daemon":
		commands.Daemon(os.Args[2:])
	case "stop":
		commands.Stop()
	case "sync":
		commands.Sync(os.Args[2:])
	case "status":
		commands.Status()
	case "browse", "files":
		commands.Browse()
	case "dashboard", "watch":
		commands.Dashboard()
	case "install":
		commands.Install()
	case "uninstall":
		commands.Uninstall()
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
	usage := fmt.Sprintf(`notebridge - Bidirectional sync between org-roam and Obsidian

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
  uninstall   Remove system service files
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
  notebridge uninstall

Configuration:
  Config file: %s
  State file:  %s

For more information, visit: https://github.com/gerunddev/notebridge
`, config.ConfigPath(), config.StateFilePath())
	fmt.Print(usage)
}
