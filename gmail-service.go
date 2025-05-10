package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	attachmentsDir   = "./attachments"
	clientSecretPath = "client_secret_736802718299-09qksrnedamuqnub21d2ufm6coa1msuh.apps.googleusercontent.com.json"
)

// GmailService represents a Gmail service client
type GmailService struct {
	service *gmail.Service
	user    string
}

// NewGmailService creates a new Gmail service client
func NewGmailService() (*GmailService, error) {
	ctx := context.Background()
	b, err := os.ReadFile(clientSecretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	httpClient := getOAuth2Client(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	return &GmailService{
		service: srv,
		user:    "me",
	}, nil
}

// ProcessEmails reads emails from the last fetch time and processes them.
// It returns an error if any step of the process fails.
// The function will:
//   - Initialize the Gmail service
//   - Load the activity manager
//   - Fetch messages since the last fetch time
//   - Process each message and download attachments in the attachments directory
//   - Store the new last fetch time for the next run
func ProcessEmails() error {
	gmailService, err := NewGmailService()
	if err != nil {
		return err
	}

	activityManager := NewActivityManager()
	if err := activityManager.Load(); err != nil {
		return fmt.Errorf("error loading activity data: %v", err)
	}

	lastFetchTime, err := activityManager.ReadLastFetchTime()
	if err != nil {
		log.Printf("Error reading last fetch time: %v", err)
		lastFetchTime = time.Now().AddDate(0, 0, -30).Format(defaultDateFormat)
	}

	messages, err := gmailService.listMessages(lastFetchTime)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Printf("No messages found since last fetch: %s\n", lastFetchTime)
		return nil
	}

	fmt.Printf("Found %d messages with PDF attachments in the last 30 days.\n", len(messages))

	for _, msg := range messages {
		if err := gmailService.processMessage(activityManager, msg); err != nil {
			log.Printf("Error processing message %s: %v", msg.Id, err)
			continue
		}
	}

	if err := activityManager.StoreLastFetchTime(); err != nil {
		log.Printf("Error writing last fetch time: %v", err)
	}

	return activityManager.Save()
}

// listMessages retrieves messages with PDF attachments after the given time
func (gs *GmailService) listMessages(afterTime string) ([]*gmail.Message, error) {
	query := fmt.Sprintf("after:%s has:attachment filename:pdf", afterTime)
	r, err := gs.service.Users.Messages.List(gs.user).Q(query).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %v", err)
	}

	if len(r.Messages) == 0 {
		return nil, nil
	}

	var messages []*gmail.Message
	for _, m := range r.Messages {
		msg, err := gs.service.Users.Messages.Get(gs.user, m.Id).Do()
		if err != nil {
			log.Printf("Error getting message %s: %v", m.Id, err)
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// processMessage processes a single email message
func (gs *GmailService) processMessage(am *ActivityManager, msg *gmail.Message) error {
	fmt.Printf("Message ID: %s, Subject: %s\n", msg.Id, getSubject(msg))

	if am.HasEmailID(msg.Id) {
		fmt.Printf("Skipping message %s as it was already processed\n", msg.Id)
		return nil
	}

	if err := am.StoreEmailFetchTime(msg.Id, msg); err != nil {
		return fmt.Errorf("error storing email ID: %v", err)
	}

	return gs.downloadAttachments(msg)
}

// downloadAttachments downloads PDF attachments from a message
func (gs *GmailService) downloadAttachments(msg *gmail.Message) error {
	for _, part := range msg.Payload.Parts {
		if part.Filename != "" && part.MimeType == "application/pdf" {
			if err := gs.downloadAttachment(msg.Id, part); err != nil {
				log.Printf("Error downloading attachment: %v", err)
				continue
			}
		}
	}
	return nil
}

// downloadAttachment downloads a single attachment
func (gs *GmailService) downloadAttachment(messageID string, part *gmail.MessagePart) error {
	attachment, err := gs.service.Users.Messages.Attachments.Get(gs.user, messageID, part.Body.AttachmentId).Do()
	if err != nil {
		return fmt.Errorf("error getting attachment: %v", err)
	}

	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return fmt.Errorf("error decoding attachment: %v", err)
	}

	if err := os.MkdirAll(attachmentsDir, defaultDirPerm); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	filePath := fmt.Sprintf("%s/%s", attachmentsDir, part.Filename)
	if err := os.WriteFile(filePath, data, defaultFilePerm); err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	fmt.Printf("Downloaded attachment: %s\n", part.Filename)
	return nil
}

// getSubject extracts the subject from a Gmail message
func getSubject(msg *gmail.Message) string {
	for _, header := range msg.Payload.Headers {
		if header.Name == "Subject" {
			return header.Value
		}
	}
	return "No Subject"
}
