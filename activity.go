package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	ID   string `json:"id"`
	Time string `json:"time"`
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
	file, err := os.Create("./activity.json")
	if err != nil {
		return fmt.Errorf("error creating activity.json: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(am.data); err != nil {
		return fmt.Errorf("error encoding activity data: %v", err)
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

// WriteLastFetchTime updates the last fetch time in the in-memory activity data.
func (am *ActivityManager) WriteLastFetchTime() error {
	am.data.LastFetchTime = time.Now().Format(time.RFC3339)
	fmt.Println("Updated last fetch time in memory.")
	return nil
}

// WriteEmailFetchTime stores the datetime of a fetched email into the in-memory activity data.
func (am *ActivityManager) WriteEmailFetchTime(emailID string, msg *gmail.Message) error {
	// Initialize EmailData if it doesn't exist
	if am.data.Emails == nil {
		am.data.Emails = []EmailData{}
	}

	// Extract the time from the email message
	var emailTime string
	for _, header := range msg.Payload.Headers {
		if header.Name == "Date" {
			// Parse the email date and format it to '2006/01/02'
			t, err := time.Parse(time.RFC1123Z, header.Value)
			if err != nil {
				log.Printf("Error parsing email date: %v", err)
				emailTime = time.Now().Format("2006/01/02")
			} else {
				emailTime = t.Format("2006/01/02")
			}
			break
		}
	}

	// Append the new email ID and formatted time
	am.data.Emails = append(am.data.Emails, EmailData{
		ID:   emailID,
		Time: emailTime,
	})

	fmt.Printf("Stored email ID %s with time %s in memory.\n", emailID, emailTime)
	return nil
}
