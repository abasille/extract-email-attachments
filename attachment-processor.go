package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"extract-email-attachments/config"
)

// ProcessAttachments processes each attachment in the attachments directory
// and performs specific actions based on the sender and filename
func ProcessAttachments() error {
	activityManager := NewActivityManager()
	if err := activityManager.Load(); err != nil {
		return fmt.Errorf("error loading activity data: %v", err)
	}

	// Walk through all files in the attachments directory
	err := filepath.Walk(config.AttachmentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip non-PDF files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".pdf") {
			return nil
		}

		// Get the filename
		filename := info.Name()

		// Find the attachment in the activity manager
		for _, attachment := range activityManager.data.Attachments {
			if attachment.Filename == filename {
				// Get the associated email
				email, err := activityManager.GetEmailByID(attachment.EmailID)
				if err != nil {
					log.Printf("Error finding email for attachment %s: %v", filename, err)
					continue
				}

				// Check if the sender is IKUTO and the filename contains "facture"
				if strings.EqualFold(email.SenderName, "IKUTO") &&
					strings.Contains(strings.ToLower(email.Subject), "facture") {

					// Parse the email date
					emailDate, err := time.Parse(time.RFC3339, email.Date)
					if err != nil {
						log.Printf("Error parsing email date for %s: %v", filename, err)
						continue
					}

					// Create new filename
					newFilename := fmt.Sprintf("%s-facture-IKUTO.pdf",
						emailDate.Format("2006-01"))

					// Rename the file
					oldPath := filepath.Join(config.AttachmentsDir, filename)
					newPath := filepath.Join(config.AttachmentsDir, newFilename)

					if err := os.Rename(oldPath, newPath); err != nil {
						log.Printf("Error renaming file %s to %s: %v", filename, newFilename, err)
						continue
					}

					// Update attachment status
					if err := activityManager.UpdateAttachmentStatus(filename, "processed"); err != nil {
						log.Printf("Error updating attachment status for %s: %v", filename, err)
					}

					fmt.Printf("Renamed %s to %s\n", filename, newFilename)
				}
				break
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error processing attachments: %v", err)
	}

	// Save the updated activity data
	return activityManager.Save()
}
