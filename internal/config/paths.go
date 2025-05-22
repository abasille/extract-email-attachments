package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const AppName = "extract-email-attachments"

var (
	// Base directories
	AppConfigDir string
	AppCacheDir  string
	AppLogDir    string

	// Specific paths
	AppAttachmentsDir    string
	TerminalNotifierPath string
	UserDownloadsDir     string
)

// InitAppPaths initializes all necessary paths for the application
func InitAppPaths() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Set up base directories
	AppConfigDir = filepath.Join(homeDir, ".config", AppName)
	AppCacheDir = filepath.Join(AppConfigDir, "caches")
	AppLogDir = filepath.Join(AppConfigDir, "logs")

	// Set up specific paths
	UserDownloadsDir = filepath.Join(homeDir, "Downloads")
	AppAttachmentsDir = filepath.Join(UserDownloadsDir, "attachments")
	TerminalNotifierPath = "/opt/homebrew/bin/terminal-notifier"

	// Create directories if they don't exist
	dirs := []string{AppConfigDir, AppCacheDir, AppLogDir, UserDownloadsDir, AppAttachmentsDir}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
	}

	// Check if terminal-notifier is installed
	if _, err := exec.LookPath(TerminalNotifierPath); err != nil {
		return fmt.Errorf("terminal-notifier is not installed: %w", err)
	}

	return nil
}
