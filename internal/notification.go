package internal

import (
	"extract-email-attachments/internal/config"
	"fmt"
	"os/exec"
	"runtime"
)

// displayNotification shows a macOS system notification with the given message
func displayNotification(message string) error {
	// Check if we're running on macOS
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("system notifications are only supported on macOS")
	}

	// Check if terminal-notifier is installed
	if _, err := exec.LookPath("terminal-notifier"); err != nil {
		// Install terminal-notifier using Homebrew
		installCmd := exec.Command("brew", "install", "terminal-notifier")
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install terminal-notifier: %v", err)
		}
	}

	// Create the notification command
	cmd := exec.Command("terminal-notifier",
		"-title", "Extract Email Attachments",
		"-message", message,
		"-open", "file://"+config.AppAttachmentsDir,
		"-sound", "default")

	// Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to display notification: %v", err)
	}

	return nil
}

// HandleNotificationClick handles the click on a notification
func HandleNotificationClick(url string) error {
	if url == "" {
		return nil
	}

	// Check if the URL is our custom protocol
	if len(url) > 12 && url[:12] == "openfolder://" {
		// Extract the path from the URL
		path := url[12:]

		// Execute the command to open the folder
		cmd := exec.Command("open", path)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to open attachments directory: %v", err)
		}
	}

	return nil
}
