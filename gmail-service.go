package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gosxnotifier "github.com/deckarep/gosx-notifier"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/term"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"extract-email-attachments/config"
)

// GmailService represents a Gmail service client
type GmailService struct {
	service *gmail.Service
	user    string
}

type Credentials struct {
	Installed struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"installed"`
}

func getCredentials() (string, string, error) {
	// 1. Variables d'environnement
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientID != "" && clientSecret != "" {
		return clientID, clientSecret, nil
	}

	// 2. Fichier de configuration
	usr, err := user.Current()
	if err == nil {
		configPath := filepath.Join(usr.HomeDir, ".config", "extract-email-attachments", "credentials.json")
		if f, err := os.Open(configPath); err == nil {
			defer f.Close()
			var creds Credentials
			if err := json.NewDecoder(f).Decode(&creds); err == nil {
				if creds.Installed.ClientID != "" && creds.Installed.ClientSecret != "" {
					return creds.Installed.ClientID, creds.Installed.ClientSecret, nil
				}
			} else {
				fmt.Println("Error decoding JSON: ", err)
			}
		} else {
			fmt.Println("Error opening credentials file: ", err)
		}
	} else {
		fmt.Println("Error getting current user: ", err)
	}

	// 3. Prompt interactif
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Entrez votre GOOGLE_CLIENT_ID : ")
	clientID, _ = reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Entrez votre GOOGLE_CLIENT_SECRET : ")
	secretBytes, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	clientSecret = strings.TrimSpace(string(secretBytes))

	if clientID == "" || clientSecret == "" {
		return "", "", fmt.Errorf("Client ID et/ou Client Secret manquant")
	}
	return clientID, clientSecret, nil
}

// NewGmailService creates a new Gmail service client
func NewGmailService() (*GmailService, error) {
	ctx := context.Background()

	clientID, clientSecret, err := getCredentials()
	if err != nil {
		log.Fatal("Impossible d'obtenir les identifiants OAuth2 :", err)
	}

	// Configure OAuth2 for desktop application
	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{gmail.GmailReadonlyScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080",
	}

	httpClient := getOAuth2Client(oauthConfig)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	return &GmailService{
		service: srv,
		user:    "me",
	}, nil
}

// displayNotification shows a macOS system notification with the given message
func displayNotification(message string) error {
	// Check if we're running on macOS
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("system notifications are only supported on macOS")
	}

	// Create a new notification
	note := gosxnotifier.NewNotification(message)

	// Set notification properties
	note.Title = "Extract Email Attachments"
	note.Sound = gosxnotifier.Default
	note.AppIcon = "mail.icns" // Optional: you can set a custom icon

	// Push the notification
	if err := note.Push(); err != nil {
		return fmt.Errorf("failed to display notification: %v", err)
	}

	return nil
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
		message := fmt.Sprintf("No messages found since last fetch: %s", lastFetchTime)
		fmt.Println(message)
		if err := displayNotification(message); err != nil {
			log.Printf("Warning: Could not display notification: %v", err)
		}
		return nil
	}

	message := fmt.Sprintf("Found %d messages with PDF attachments in the last 30 days.", len(messages))
	fmt.Println(message)
	if err := displayNotification(message); err != nil {
		log.Printf("Warning: Could not display notification: %v", err)
	}

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

	if err := am.StoreEmailMeta(msg.Id, msg); err != nil {
		return fmt.Errorf("error storing email metadata: %v", err)
	}

	return gs.downloadAttachments(msg, am)
}

// downloadAttachments downloads PDF attachments from a message
func (gs *GmailService) downloadAttachments(msg *gmail.Message, am *ActivityManager) error {
	for _, part := range msg.Payload.Parts {
		if part.Filename != "" && part.MimeType == "application/pdf" {
			if err := gs.downloadAttachment(msg.Id, part, am); err != nil {
				log.Printf("Error downloading attachment: %v", err)
				continue
			}
		}
	}
	return nil
}

// downloadAttachment downloads a single attachment
func (gs *GmailService) downloadAttachment(messageID string, part *gmail.MessagePart, am *ActivityManager) error {
	attachment, err := gs.service.Users.Messages.Attachments.Get(gs.user, messageID, part.Body.AttachmentId).Do()
	if err != nil {
		return fmt.Errorf("error getting attachment: %v", err)
	}

	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return fmt.Errorf("error decoding attachment: %v", err)
	}

	if err := os.MkdirAll(config.AttachmentsDir, defaultDirPerm); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	filePath := fmt.Sprintf("%s/%s", config.AttachmentsDir, part.Filename)
	if err := os.WriteFile(filePath, data, defaultFilePerm); err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	// Store attachment metadata
	if err := am.StoreAttachmentMeta(part.Filename, messageID); err != nil {
		log.Printf("Error storing attachment metadata: %v", err)
	}

	fmt.Printf("Downloaded attachment: %s\n", filePath)
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
