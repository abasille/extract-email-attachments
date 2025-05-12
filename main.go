package main

import (
	"log"

	"extract-email-attachments/internal"
	"extract-email-attachments/internal/config"
)

func main() {
	// Initialize application paths
	if err := config.InitAppPaths(); err != nil {
		log.Fatalf("Error initializing application paths: %v", err)
	}
	// Process emails and attachments
	if err := internal.ProcessEmails(); err != nil {
		log.Fatalf("Error processing emails: %v", err)
	}

	if err := internal.ProcessAttachments(); err != nil {
		log.Fatalf("Error processing attachments: %v", err)
	}
}
