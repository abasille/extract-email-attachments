package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"encoding/base64"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// getClient retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// getTokenFromWeb requests a token from the web using a local server with a custom redirect URI.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("state") != randState {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}
		if code := r.FormValue("code"); code != "" {
			fmt.Fprintf(w, "<h1>Success</h1>Authorized.")
			ch <- code
		} else {
			http.Error(w, "code not found", http.StatusBadRequest)
		}
	})
	server := &http.Server{Addr: ":8080", Handler: nil}
	go server.ListenAndServe()
	defer server.Close()

	// Set the redirect URI to http://localhost:8080
	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL(randState, oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	code := <-ch
	tok, err := config.Exchange(context.TODO(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// tokenFromFile retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// ensureActivityFileExists checks if the activity.json file exists and creates it if it doesn't.
func ensureActivityFileExists() {
	if _, err := os.Stat("./activity.json"); os.IsNotExist(err) {
		file, err := os.Create("./activity.json")
		if err != nil {
			log.Fatalf("Error creating activity.json: %v", err)
		}
		file.Close()
		fmt.Println("Created activity.json file.")
	}
}

// readLast30DaysEmails reads emails from Gmail for the last 30 days, filtering only those with PDF attachments.
func readLast30DaysEmails() {
	ensureActivityFileExists()

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
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	user := "me"
	// Calculate the date 30 days ago
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006/01/02")
	query := fmt.Sprintf("after:%s has:attachment filename:pdf", thirtyDaysAgo)
	r, err := srv.Users.Messages.List(user).Q(query).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve messages: %v", err)
	}

	if len(r.Messages) == 0 {
		fmt.Println("No messages found.")
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
