package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"extract-email-attachments/internal/config"
)

// ProcessAttachments processes each attachment in the attachments directory
// and performs specific actions based on the sender and filename
func ProcessAttachments() error {
	activityManager := NewActivityManager()
	if err := activityManager.Load(); err != nil {
		return NewError("ProcessAttachments", err, "failed to load activity data")
	}

	var processingErrors []error

	// Walk through all files in the attachments directory
	err := filepath.Walk(config.AppAttachmentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return NewError("ProcessAttachments", err, fmt.Sprintf("failed to access path %s", path))
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
					err = NewError("ProcessAttachments", err, fmt.Sprintf("failed to find email for attachment %s", filename))
					log.Printf("Error: %v", err)
					processingErrors = append(processingErrors, err)
					continue
				}

				// Check if the sender is IKUTO and the filename contains "facture"
				if strings.EqualFold(email.SenderName, "IKUTO") &&
					strings.Contains(strings.ToLower(email.Subject), "facture") {

					// Parse the email date
					emailDate, err := time.Parse(time.RFC3339, email.Date)
					if err != nil {
						err = NewError("ProcessAttachments", err, fmt.Sprintf("failed to parse email date for %s", filename))
						log.Printf("Error: %v", err)
						processingErrors = append(processingErrors, err)
						continue
					}

					// Create new filename
					newFilename := fmt.Sprintf("%s-facture-IKUTO.pdf",
						emailDate.Format("2006-01"))

					// Rename the file
					oldPath := filepath.Join(config.AppAttachmentsDir, filename)
					newPath := filepath.Join(config.AppAttachmentsDir, newFilename)

					if err := os.Rename(oldPath, newPath); err != nil {
						err = NewError("ProcessAttachments", err, fmt.Sprintf("failed to rename file %s to %s", filename, newFilename))
						log.Printf("Error: %v", err)
						processingErrors = append(processingErrors, err)
						continue
					}

					// Update attachment status
					if err := activityManager.UpdateAttachmentStatus(filename, "processed"); err != nil {
						log.Printf("Warning: Error updating attachment status for %s: %v", filename, err)
						// Ne pas retourner l'erreur car ce n'est pas critique
					}

					fmt.Printf("Renamed %s to %s\n", filename, newFilename)
				}
				break
			}
		}
		return nil
	})

	if err != nil {
		return NewError("ProcessAttachments", err, "failed to walk through attachments directory")
	}

	// Save the updated activity data
	if err := activityManager.Save(); err != nil {
		return NewError("ProcessAttachments", err, "failed to save activity data")
	}

	// Si des erreurs de traitement se sont produites, les retourner
	if len(processingErrors) > 0 {
		return NewError("ProcessAttachments", ErrAttachmentProcessing, fmt.Sprintf("encountered %d errors while processing attachments", len(processingErrors)))
	}

	return nil
}
