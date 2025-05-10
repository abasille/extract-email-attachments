package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
)

// getOAuth2Client retrieves a token, saves the token, then returns the generated client.
func getOAuth2Client(config *oauth2.Config) *http.Client {
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
