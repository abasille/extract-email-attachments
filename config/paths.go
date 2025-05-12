package config

import (
	"os"
	"path/filepath"
)

const AppName = "extract-email-attachments"

var (
	// Base directories
	AppSupportDir string
	CacheDir      string
	LogDir        string

	// Specific paths
	AttachmentsDir string
	DownloadsDir   string
)

// InitAppPaths initializes all necessary paths for the application
func InitAppPaths() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Set up base directories
	AppSupportDir = filepath.Join(homeDir, "Library", "Application Support", AppName)
	CacheDir = filepath.Join(homeDir, "Library", "Caches", AppName)
	LogDir = filepath.Join(homeDir, "Library", "Logs", AppName)

	// Set up specific paths
	DownloadsDir = filepath.Join(homeDir, "Downloads")
	AttachmentsDir = filepath.Join(DownloadsDir, "attachments")

	if _, err := os.Stat(AttachmentsDir); os.IsNotExist(err) {
		os.MkdirAll(AttachmentsDir, 0755)
	}

	return nil
}
