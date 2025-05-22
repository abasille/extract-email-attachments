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
		return NewError("displayNotification", ErrCritical, "system notifications are only supported on macOS")
	}

	// Create the notification command
	cmd := exec.Command(config.TerminalNotifierPath,
		"-title", "Extract Email Attachments",
		"-message", message,
		"-open", "file://"+config.AppAttachmentsDir,
		"-sound", "default")

	// Execute the command
	if err := cmd.Run(); err != nil {
		return NewError("displayNotification", ErrNotificationFailed, fmt.Sprintf("failed to display notification: %v", err))
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
			return NewError("HandleNotificationClick", err, fmt.Sprintf("failed to open attachments directory: %s", path))
		}
	}

	return nil
}
