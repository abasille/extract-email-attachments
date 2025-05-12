package internal

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

	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/term"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"crypto/sha256"
	"extract-email-attachments/internal/config"
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
		return NewError("ProcessEmails", err, "failed to initialize Gmail service")
	}

	activityManager := NewActivityManager()
	if err := activityManager.Load(); err != nil {
		return NewError("ProcessEmails", err, "failed to load activity data")
	}

	lastFetchTime, err := activityManager.ReadLastFetchTime()
	if err != nil {
		log.Printf("Warning: Error reading last fetch time: %v", err)
		lastFetchTime = time.Now().AddDate(0, 0, -30).Format(config.DefaultDateFormat)
	}

	messages, err := gmailService.listMessages(lastFetchTime)
	if err != nil {
		return NewError("ProcessEmails", err, "failed to list messages")
	}

	if len(messages) == 0 {
		message := fmt.Sprintf("No messages found since last fetch: %s", lastFetchTime)
		fmt.Println(message)
		if err := displayNotification(message); err != nil {
			log.Printf("Warning: Could not display notification: %v", err)
			// Ne pas retourner l'erreur car ce n'est pas critique
		}
		return nil
	}

	message := fmt.Sprintf("Found %d messages with PDF attachments in the last 30 days.", len(messages))
	fmt.Println(message)
	if err := displayNotification(message); err != nil {
		log.Printf("Warning: Could not display notification: %v", err)
		// Ne pas retourner l'erreur car ce n'est pas critique
	}

	var processingErrors []error
	for _, msg := range messages {
		if err := gmailService.processMessage(activityManager, msg); err != nil {
			err = NewError("ProcessEmails", err, fmt.Sprintf("failed to process message %s", msg.Id))
			log.Printf("Error: %v", err)
			processingErrors = append(processingErrors, err)
			continue
		}
	}

	if err := activityManager.StoreLastFetchTime(); err != nil {
		log.Printf("Warning: Error writing last fetch time: %v", err)
		// Ne pas retourner l'erreur car ce n'est pas critique
	}

	if err := activityManager.Save(); err != nil {
		return NewError("ProcessEmails", err, "failed to save activity data")
	}

	// Si des erreurs de traitement se sont produites, les retourner
	if len(processingErrors) > 0 {
		return NewError("ProcessEmails", ErrEmailProcessing, fmt.Sprintf("encountered %d errors while processing messages", len(processingErrors)))
	}

	return nil
}

// listMessages retrieves messages with PDF attachments after the given time
func (gs *GmailService) listMessages(afterTime string) ([]*gmail.Message, error) {
	query := fmt.Sprintf("after:%s has:attachment filename:pdf", afterTime)
	r, err := gs.service.Users.Messages.List(gs.user).Q(query).Do()
	if err != nil {
		return nil, NewError("listMessages", err, "failed to retrieve messages from Gmail API")
	}

	if len(r.Messages) == 0 {
		return nil, nil
	}

	var messages []*gmail.Message
	var errors []error
	for _, m := range r.Messages {
		msg, err := gs.service.Users.Messages.Get(gs.user, m.Id).Do()
		if err != nil {
			err = NewError("listMessages", err, fmt.Sprintf("failed to get message %s", m.Id))
			log.Printf("Error: %v", err)
			errors = append(errors, err)
			continue
		}
		messages = append(messages, msg)
	}

	if len(errors) > 0 {
		return messages, NewError("listMessages", ErrGmailAPI, fmt.Sprintf("encountered %d errors while retrieving messages", len(errors)))
	}

	return messages, nil
}

// processMessage processes a single email message
func (gs *GmailService) processMessage(am *ActivityManager, msg *gmail.Message) error {
	fmt.Printf("Message ID: %s, Subject: %s\n", msg.Id, getSubject(msg))

	if msg.Id == "" {
		return NewError("processMessage", ErrInvalidEmailID, "message ID is empty")
	}

	if am.HasEmailID(msg.Id) {
		fmt.Printf("Skipping message %s as it was already processed\n", msg.Id)
		return nil
	}

	if err := am.StoreEmailMeta(msg.Id, msg); err != nil {
		return NewError("processMessage", err, "failed to store email metadata")
	}

	if err := gs.downloadAttachments(msg, am); err != nil {
		return NewError("processMessage", err, "failed to download attachments")
	}

	return nil
}

// downloadAttachments downloads PDF attachments from a message
func (gs *GmailService) downloadAttachments(msg *gmail.Message, am *ActivityManager) error {
	var errors []error
	for _, part := range msg.Payload.Parts {
		if part.Filename != "" && part.MimeType == "application/pdf" {
			if err := gs.downloadAttachment(msg.Id, part, am); err != nil {
				err = NewError("downloadAttachments", err, fmt.Sprintf("failed to download attachment %s", part.Filename))
				log.Printf("Error: %v", err)
				errors = append(errors, err)
				continue
			}
		}
	}

	if len(errors) > 0 {
		return NewError("downloadAttachments", ErrAttachmentProcessing, fmt.Sprintf("encountered %d errors while downloading attachments", len(errors)))
	}

	return nil
}

// downloadAttachment downloads a single attachment
func (gs *GmailService) downloadAttachment(messageID string, part *gmail.MessagePart, am *ActivityManager) error {
	if messageID == "" {
		return NewError("downloadAttachment", ErrInvalidEmailID, "message ID is empty")
	}

	if part.Filename == "" {
		return NewError("downloadAttachment", ErrInvalidFilename, "attachment filename is empty")
	}

	attachment, err := gs.service.Users.Messages.Attachments.Get(gs.user, messageID, part.Body.AttachmentId).Do()
	if err != nil {
		return NewError("downloadAttachment", err, "failed to get attachment from Gmail API")
	}

	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return NewError("downloadAttachment", err, "failed to decode attachment data")
	}

	if err := os.MkdirAll(config.AppAttachmentsDir, defaultDirPerm); err != nil {
		return NewError("downloadAttachment", err, "failed to create attachments directory")
	}

	filePath := fmt.Sprintf("%s/%s", config.AppAttachmentsDir, part.Filename)
	if err := os.WriteFile(filePath, data, defaultFilePerm); err != nil {
		return NewError("downloadAttachment", err, "failed to write attachment file")
	}

	sha256Hash := fmt.Sprintf("%x", sha256.Sum256(data))

	if err := am.StoreAttachmentMeta(part.Filename, messageID, sha256Hash); err != nil {
		log.Printf("Warning: Error storing attachment metadata: %v", err)
		// Ne pas retourner l'erreur car ce n'est pas critique
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
