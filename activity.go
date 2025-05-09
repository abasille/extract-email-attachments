package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// ActivityData represents the structure of the activity.json file.
type ActivityData struct {
	LastFetchTime string      `json:"lastFetchTime"`
	Emails        []EmailData `json:"emails"`
}

// EmailData represents the structure for storing email fetch times.
type EmailData struct {
	ID             string `json:"id"`
	Date           string `json:"date"`
	Subject        string `json:"subject"`
	SenderName     string `json:"senderName"`
	SenderEmail    string `json:"senderEmail"`
	AttachmentName string `json:"attachmentName"`
}

// ActivityManager manages the activity data operations.
type ActivityManager struct {
	data ActivityData
}

// NewActivityManager creates a new ActivityManager instance.
func NewActivityManager() *ActivityManager {
	return &ActivityManager{
		data: ActivityData{},
	}
}

// Load loads the activity data from the file into memory.
func (am *ActivityManager) Load() error {
	file, err := os.Open("./activity.json")
	if err != nil {
		// If the file does not exist, create it with an initial structure
		if os.IsNotExist(err) {
			am.data = ActivityData{
				Emails: []EmailData{},
			}
			return am.Save() // Save the initial data to the file
		}
		return fmt.Errorf("error opening activity.json: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&am.data); err != nil {
		return fmt.Errorf("error decoding activity data: %v", err)
	}

	return nil
}

// Save writes the in-memory activity data back to the file.
func (am *ActivityManager) Save() error {
	// Marshal the data with indentation
	data, err := json.MarshalIndent(am.data, "", "    ")
	if err != nil {
		return fmt.Errorf("error encoding activity data: %v", err)
	}

	// Write the formatted data to the file
	file, err := os.Create("./activity.json")
	if err != nil {
		return fmt.Errorf("error creating activity.json: %v", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("error writing to activity.json: %v", err)
	}

	fmt.Println("Saved activity data to activity.json.")
	return nil
}

// ReadLastFetchTime reads the last fetch time from the in-memory activity data and formats it as '2006/01/02'.
func (am *ActivityManager) ReadLastFetchTime() (string, error) {
	t, err := time.Parse(time.RFC3339, am.data.LastFetchTime)
	if err != nil {
		return "", fmt.Errorf("error parsing last fetch time: %v", err)
	}

	return t.Format("2006/01/02"), nil
}

// StoreLastFetchTime updates the last fetch time in the in-memory activity data.
func (am *ActivityManager) StoreLastFetchTime() error {
	am.data.LastFetchTime = time.Now().Format(time.RFC3339)
	fmt.Println("Updated last fetch time in memory.")
	return nil
}

// StoreEmailFetchTime stores the datetime of a fetched email into the in-memory activity data.
func (am *ActivityManager) StoreEmailFetchTime(emailID string, msg *gmail.Message) error {
	// Initialize EmailData if it doesn't exist
	if am.data.Emails == nil {
		am.data.Emails = []EmailData{}
	}

	// Extract the time from the email message
	var emailDate string
	var subject string
	var senderName string
	var senderEmail string
	var attachmentName string

	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "Date":
			// Parse the email date and format it to RFC3339
			t, err := time.Parse(time.RFC1123Z, header.Value)
			if err != nil {
				log.Printf("Error parsing email date: %v", err)
				emailDate = time.Now().Format(time.RFC3339)
			} else {
				emailDate = t.Format(time.RFC3339)
			}
		case "Subject":
			subject = header.Value
		case "From":
			// Extract sender name and email
			senderName, senderEmail = extractSenderInfo(header.Value)
		}
	}

	// Extract attachment name from the message
	for _, part := range msg.Payload.Parts {
		if part.Filename != "" {
			attachmentName = part.Filename
			break
		}
	}

	// Append the new email ID, date, subject, sender name, sender email, and attachment name
	am.data.Emails = append(am.data.Emails, EmailData{
		ID:             emailID,
		Date:           emailDate,
		Subject:        subject,
		SenderName:     senderName,
		SenderEmail:    senderEmail,
		AttachmentName: attachmentName,
	})

	fmt.Printf("Stored email ID %s with date %s, subject: %s, sender: %s <%s>, attachment: %s in memory.\n", emailID, emailDate, subject, senderName, senderEmail, attachmentName)
	return nil
}

// extractSenderInfo extracts the sender's name and email from the "From" header.
func extractSenderInfo(fromHeader string) (name, email string) {
	// Example format: "John Doe <john.doe@example.com>"
	parts := strings.Split(fromHeader, "<")
	if len(parts) == 2 {
		name = strings.TrimSpace(parts[0])
		name = strings.Trim(name, "\"") // Remove quotes from the sender's name
		email = strings.Trim(parts[1], ">")
	} else {
		email = fromHeader
	}
	return
}
