package main

import (
	"log"

	"extract-email-attachments/config"
)

const (
	defaultDateFormat = "2006/01/02"
)

func main() {
	// Initialize application paths
	if err := config.InitAppPaths(); err != nil {
		log.Fatalf("Error initializing application paths: %v", err)
	}

	// Process emails and attachments
	if err := ProcessEmails(); err != nil {
		log.Fatalf("Error processing emails: %v", err)
	}

	if err := ProcessAttachments(); err != nil {
		log.Fatalf("Error processing attachments: %v", err)
	}
}
