package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"encoding/base64"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// readLast30DaysEmails reads emails from Gmail for the last 30 days, filtering only those with PDF attachments.
func readLast30DaysEmails() {
	ctx := context.Background()
	b, err := os.ReadFile("client_secret_736802718299-09qksrnedamuqnub21d2ufm6coa1msuh.apps.googleusercontent.com.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	httpClient := getOAuth2Client(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	user := "me"
	activityManager := NewActivityManager()
	if err := activityManager.Load(); err != nil {
		log.Printf("Error loading activity data: %v", err)
		return
	}
	// Read the last fetch time from activity.json
	lastFetchTime, err := activityManager.ReadLastFetchTime()
	if err != nil {
		log.Printf("Error reading last fetch time: %v", err)
		// Default to last 30 days if there's an error
		lastFetchTime = time.Now().AddDate(0, 0, -30).Format("2006/01/02")
	}

	query := fmt.Sprintf("after:%s has:attachment filename:pdf", lastFetchTime)
	r, err := srv.Users.Messages.List(user).Q(query).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve messages: %v", err)
	}

	if len(r.Messages) == 0 {
		fmt.Printf("No messages found since last fetch: %s\n", lastFetchTime)
		return
	}

	fmt.Printf("Found %d messages with PDF attachments in the last 30 days.\n", len(r.Messages))
	for _, m := range r.Messages {
		msg, err := srv.Users.Messages.Get(user, m.Id).Do()
		if err != nil {
			log.Printf("Error getting message %s: %v", m.Id, err)
			continue
		}
		fmt.Printf("Message ID: %s, Subject: %s\n", m.Id, getSubject(msg))

		// Check if this message ID already exists in activity.json
		if activityManager.HasEmailID(m.Id) {
			fmt.Printf("Skipping message %s as it was already processed\n", m.Id)
			continue
		}

		// Store the email ID and time in activity.json
		if err := activityManager.StoreEmailFetchTime(m.Id, msg); err != nil {
			log.Printf("Error storing email ID: %v", err)
		}

		// Download attachments
		for _, part := range msg.Payload.Parts {
			if part.Filename != "" && part.MimeType == "application/pdf" {
				attachment, err := srv.Users.Messages.Attachments.Get(user, m.Id, part.Body.AttachmentId).Do()
				if err != nil {
					log.Printf("Error getting attachment: %v", err)
					continue
				}
				data, err := base64.URLEncoding.DecodeString(attachment.Data)
				if err != nil {
					log.Printf("Error decoding attachment: %v", err)
					continue
				}
				// Create attachments directory if it doesn't exist
				if err := os.MkdirAll("./attachments", 0755); err != nil {
					log.Printf("Error creating directory: %v", err)
					continue
				}
				filePath := fmt.Sprintf("./attachments/%s", part.Filename)
				if err := os.WriteFile(filePath, data, 0644); err != nil {
					log.Printf("Error writing file: %v", err)
					continue
				}
				fmt.Printf("Downloaded attachment: %s\n", part.Filename)
			}
		}
	}

	// Write the last fetch time to activity.json
	if err := activityManager.StoreLastFetchTime(); err != nil {
		log.Printf("Error writing last fetch time: %v", err)
	}

	if err := activityManager.Save(); err != nil {
		log.Printf("Error saving activity data: %v", err)
	}
}

// getSubject extracts the subject from a Gmail message.
func getSubject(msg *gmail.Message) string {
	for _, header := range msg.Payload.Headers {
		if header.Name == "Subject" {
			return header.Value
		}
	}
	return "No Subject"
}

func main() {
	readLast30DaysEmails()
}
