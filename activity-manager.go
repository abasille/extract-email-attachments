package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/gmail/v1"
)

const (
	defaultDirPerm  = 0755
	defaultFilePath = "./activity.json"
	defaultFilePerm = 0644
)

// ActivityManagerInterface defines the interface for activity management operations
type ActivityManagerInterface interface {
	Load() error
	Save() error
	HasEmailID(string) bool
	StoreEmailMeta(string, *gmail.Message) error
	StoreAttachmentMeta(string, string) error
	ReadLastFetchTime() (string, error)
	StoreLastFetchTime() error
	UpdateAttachmentStatus(string, string) error
	GetEmailByID(string) (*EmailData, error)
}

// ActivityData represents the structure of the activity.json file.
type ActivityData struct {
	LastFetchTime string           `json:"lastFetchTime"`
	Emails        []EmailData      `json:"emails"`
	Attachments   []AttachmentData `json:"attachments"`
}

// EmailData represents the structure for storing email metadata.
type EmailData struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Subject     string `json:"subject"`
	SenderName  string `json:"senderName"`
	SenderEmail string `json:"senderEmail"`
}

// AttachmentData represents the structure for storing attachment metadata.
type AttachmentData struct {
	Filename string `json:"filename"`
	EmailID  string `json:"emailId"`
	Status   string `json:"status,omitempty"`
}

// ActivityManager manages the activity data operations.
type ActivityManager struct {
	mu       sync.RWMutex
	data     ActivityData
	filePath string
}

// NewActivityManager creates a new ActivityManager instance.
func NewActivityManager() *ActivityManager {
	return &ActivityManager{
		data:     ActivityData{},
		filePath: defaultFilePath,
	}
}

// NewActivityManagerWithPath creates a new ActivityManager instance with a custom file path.
func NewActivityManagerWithPath(filePath string) *ActivityManager {
	return &ActivityManager{
		data:     ActivityData{},
		filePath: filePath,
	}
}

// Load loads the activity data from the file into memory.
func (am *ActivityManager) Load() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	file, err := os.Open(am.filePath)
	if err != nil {
		// If the file does not exist, initialize the data structure
		if os.IsNotExist(err) {
			am.data = ActivityData{
				Emails:      []EmailData{},
				Attachments: []AttachmentData{},
			}
			return nil
		}
		return fmt.Errorf("error opening activity file: %v", err)
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
	am.mu.Lock()
	defer am.mu.Unlock()

	// Marshal the data with indentation
	data, err := json.MarshalIndent(am.data, "", "    ")
	if err != nil {
		return fmt.Errorf("error encoding activity data: %v", err)
	}

	// Write the formatted data to the file
	file, err := os.Create(am.filePath)
	if err != nil {
		return fmt.Errorf("error creating activity file: %v", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("error writing to activity file: %v", err)
	}

	fmt.Println("Saved activity data to", am.filePath)
	return nil
}

// ReadLastFetchTime reads the last fetch time from the in-memory activity data and formats it as '2006/01/02'.
func (am *ActivityManager) ReadLastFetchTime() (string, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.data.LastFetchTime == "" {
		return time.Now().AddDate(0, 0, -30).Format(defaultDateFormat), nil
	}

	t, err := time.Parse(time.RFC3339, am.data.LastFetchTime)
	if err != nil {
		return "", fmt.Errorf("error parsing last fetch time: %v", err)
	}

	return t.Format(defaultDateFormat), nil
}

// StoreLastFetchTime updates the last fetch time in the in-memory activity data.
func (am *ActivityManager) StoreLastFetchTime() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.data.LastFetchTime = time.Now().Format(time.RFC3339)
	fmt.Println("Updated last fetch time in memory.")
	return nil
}

// StoreEmailMeta stores the metadata of a fetched email into the in-memory activity data.
func (am *ActivityManager) StoreEmailMeta(emailID string, msg *gmail.Message) error {
	if emailID == "" {
		return fmt.Errorf("email ID cannot be empty")
	}
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	// Initialize EmailData if it doesn't exist
	if am.data.Emails == nil {
		am.data.Emails = []EmailData{}
	}

	// Extract the time from the email message
	var emailDate string
	var subject string
	var senderName string
	var senderEmail string

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

	// Append the new email metadata
	am.data.Emails = append(am.data.Emails, EmailData{
		ID:          emailID,
		Date:        emailDate,
		Subject:     subject,
		SenderName:  senderName,
		SenderEmail: senderEmail,
	})

	fmt.Printf("Stored email ID %s with date %s, subject: %s, sender: %s <%s> in memory.\n",
		emailID, emailDate, subject, senderName, senderEmail)
	return nil
}

// StoreAttachmentMeta stores the metadata of an attachment into the in-memory activity data.
func (am *ActivityManager) StoreAttachmentMeta(filename string, emailID string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if emailID == "" {
		return fmt.Errorf("email ID cannot be empty")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	// Initialize Attachments if it doesn't exist
	if am.data.Attachments == nil {
		am.data.Attachments = []AttachmentData{}
	}

	// Append the new attachment metadata
	am.data.Attachments = append(am.data.Attachments, AttachmentData{
		Filename: filename,
		EmailID:  emailID,
	})

	fmt.Printf("Stored attachment %s for email ID %s in memory.\n", filename, emailID)
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

// HasEmailID checks if an email ID already exists in the activity data.
func (am *ActivityManager) HasEmailID(emailID string) bool {
	if emailID == "" {
		return false
	}

	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, email := range am.data.Emails {
		if email.ID == emailID {
			return true
		}
	}
	return false
}

// UpdateAttachmentStatus updates the status of an attachment
func (am *ActivityManager) UpdateAttachmentStatus(filename string, status string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	for i, attachment := range am.data.Attachments {
		if attachment.Filename == filename {
			am.data.Attachments[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("attachment not found: %s", filename)
}

// GetEmailByID returns the email data for a given ID
func (am *ActivityManager) GetEmailByID(emailID string) (*EmailData, error) {
	if emailID == "" {
		return nil, fmt.Errorf("email ID cannot be empty")
	}

	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, email := range am.data.Emails {
		if email.ID == emailID {
			return &email, nil
		}
	}
	return nil, fmt.Errorf("email not found: %s", emailID)
}
