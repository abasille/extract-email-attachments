package main

import (
	"context"
	"encoding/json"
	"extract-email-attachments/config"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

// getOAuth2Client retrieves a token, saves the token, then returns the generated client.
func getOAuth2Client(oauth2Config *oauth2.Config) *http.Client {
	tokenFilePath := config.CacheDir + "/token.json"
	token, err := tokenFromFile(tokenFilePath)
	if err != nil {
		token = getTokenFromWeb(oauth2Config)
		saveToken(tokenFilePath, token)
	}
	return oauth2Config.Client(context.Background(), token)
}

// getTokenFromWeb requests a token from the web using a local server with a custom redirect URI.
func getTokenFromWeb(oauth2Config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())

	// Generate PKCE challenge and verifier
	verifier := oauth2.GenerateVerifier()
	challenge := oauth2.S256ChallengeOption(verifier)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to signal server shutdown
	serverDone := make(chan struct{})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check for OAuth2 errors
		if err := r.FormValue("error"); err != "" {
			http.Error(w, fmt.Sprintf("OAuth2 error: %s - %s", err, r.FormValue("error_description")), http.StatusBadRequest)
			return
		}

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

	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
		close(serverDone)
	}()

	// Ensure server is shut down
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
		<-serverDone
	}()

	// Set the redirect URI to http://localhost:8080
	oauth2Config.RedirectURL = "http://localhost:8080"
	authURL := oauth2Config.AuthCodeURL(randState, oauth2.AccessTypeOffline, challenge)

	// Open the browser with the auth URL
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", authURL).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL).Start()
	case "darwin":
		err = exec.Command("open", authURL).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		fmt.Printf("Warning: Unable to open browser automatically. Please visit this URL manually:\n%v\n", authURL)
	} else {
		fmt.Println("Opening browser for authentication...")
	}

	// Wait for the authorization code with timeout
	var code string
	select {
	case code = <-ch:
	case <-time.After(5 * time.Minute):
		log.Fatal("Timeout waiting for authorization code")
	}

	tok, err := oauth2Config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		log.Fatalf("Unable to retrieve token from web [code: %s]: %v", code, err)
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
