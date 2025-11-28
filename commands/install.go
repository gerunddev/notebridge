package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gerunddev/notebridge/styles"
)

// Install generates system service files for daemon auto-start
func Install() {
	titleStyle := styles.TitleStyle
	successStyle := styles.SuccessStyle
	errorStyle := styles.ErrorStyle
	dimStyle := styles.DimStyle

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

// Uninstall removes system service files
func Uninstall() {
	titleStyle := styles.TitleStyle
	successStyle := styles.SuccessStyle
	errorStyle := styles.ErrorStyle
	warningStyle := styles.WarningStyle

	fmt.Println(titleStyle.Render("NoteBridge Uninstall"))
	fmt.Println()

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Failed to get home directory: " + err.Error()))
		os.Exit(1)
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: Remove launchd plist
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.notebridge.plist")

		// Check if file exists
		if _, err := os.Stat(plistPath); os.IsNotExist(err) {
			fmt.Println(warningStyle.Render("⚠ Service file not found: " + plistPath))
			fmt.Println("Nothing to uninstall.")
			return
		}

		// Try to unload the service first (ignore errors if not loaded)
		fmt.Println("Attempting to unload service...")
		if err := exec.Command("launchctl", "unload", plistPath).Run(); err != nil {
			fmt.Println(warningStyle.Render("⚠ Could not unload service (may not be loaded): " + err.Error()))
		}

		// Remove the plist file
		if err := os.Remove(plistPath); err != nil {
			fmt.Println(errorStyle.Render("✗ Failed to remove service file: " + err.Error()))
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("✓ Service file removed: " + plistPath))
		fmt.Println(successStyle.Render("✓ NoteBridge has been uninstalled"))

	case "linux":
		// Linux: Remove systemd user service
		servicePath := filepath.Join(home, ".config", "systemd", "user", "notebridge.service")

		// Check if file exists
		if _, err := os.Stat(servicePath); os.IsNotExist(err) {
			fmt.Println(warningStyle.Render("⚠ Service file not found: " + servicePath))
			fmt.Println("Nothing to uninstall.")
			return
		}

		// Try to stop and disable the service first (ignore errors if not running)
		fmt.Println("Attempting to stop and disable service...")
		if err := exec.Command("systemctl", "--user", "stop", "notebridge.service").Run(); err != nil {
			fmt.Println(warningStyle.Render("⚠ Could not stop service (may not be running): " + err.Error()))
		}
		if err := exec.Command("systemctl", "--user", "disable", "notebridge.service").Run(); err != nil {
			fmt.Println(warningStyle.Render("⚠ Could not disable service (may not be enabled): " + err.Error()))
		}

		// Remove the service file
		if err := os.Remove(servicePath); err != nil {
			fmt.Println(errorStyle.Render("✗ Failed to remove service file: " + err.Error()))
			os.Exit(1)
		}

		// Reload systemd
		if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to reload systemd daemon: %v\n", err)
		}

		fmt.Println(successStyle.Render("✓ Service file removed: " + servicePath))
		fmt.Println(successStyle.Render("✓ NoteBridge has been uninstalled"))

	default:
		fmt.Println(errorStyle.Render("✗ Unsupported operating system: " + runtime.GOOS))
		fmt.Println("Supported platforms: macOS (darwin), Linux")
		os.Exit(1)
	}
}
