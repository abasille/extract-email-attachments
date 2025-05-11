package main

import (
	"log"
)

const (
	defaultDateFormat = "2006/01/02"
)

func main() {
	if err := ProcessEmails(); err != nil {
		log.Fatalf("Error processing emails: %v", err)
	}

	if err := ProcessAttachments(); err != nil {
		log.Fatalf("Error processing attachments: %v", err)
	}
}
