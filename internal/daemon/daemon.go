package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// PIDFile returns the path to the daemon PID file
func PIDFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "notebridge", "daemon.pid")
}

// WritePID writes the current process ID to the PID file
func WritePID() error {
	pidFile := PIDFile()
	pidDir := filepath.Dir(pidFile)

	// Ensure directory exists
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Write PID
	pid := os.Getpid()
	content := fmt.Sprintf("%d\n", pid)
	if err := os.WriteFile(pidFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// ReadPID reads the daemon PID from the PID file
func ReadPID() (int, error) {
	pidFile := PIDFile()
	content, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("daemon not running (PID file not found)")
		}
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(content[:len(content)-1])) // Remove trailing newline
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// RemovePID removes the PID file
func RemovePID() error {
	pidFile := PIDFile()
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// IsRunning checks if the daemon is currently running
func IsRunning() (bool, int, time.Time) {
	pid, err := ReadPID()
	if err != nil {
		return false, 0, time.Time{} // Not running if PID file doesn't exist
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0, time.Time{}
	}

	// Send signal 0 to check if process is alive (doesn't actually kill it)
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist, clean up stale PID file
		if cleanupErr := RemovePID(); cleanupErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove stale PID file: %v\n", cleanupErr)
		}
		return false, 0, time.Time{}
	}

	// Get PID file modification time as approximation of start time
	pidFile := PIDFile()
	info, err := os.Stat(pidFile)
	var startTime time.Time
	if err == nil {
		startTime = info.ModTime()
	}

	return true, pid, startTime
}

// Stop stops the daemon by sending SIGTERM
func Stop() error {
	running, pid, _ := IsRunning()

	if !running {
		return fmt.Errorf("daemon is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	return nil
}

// Daemonize starts the process as a background daemon
func Daemonize(args []string) error {
	// Check if already running
	running, pid, _ := IsRunning()
	if running {
		return fmt.Errorf("daemon already running with PID %d", pid)
	}

	// Get the executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare command to run in background
	cmd := exec.Command(executable, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process detached
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Don't wait for the process
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to release daemon process: %w", err)
	}

	return nil
}
