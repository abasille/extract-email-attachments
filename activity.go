package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ActivityData represents the structure of the activity.json file.
type ActivityData struct {
	LastFetchTime string `json:"lastFetchTime"`
}

// WriteLastFetchTime writes the current datetime to the activity.json file.
func WriteLastFetchTime() error {
	data := ActivityData{
		LastFetchTime: time.Now().Format(time.RFC3339),
	}

	file, err := os.Create("./activity.json")
	if err != nil {
		return fmt.Errorf("error creating activity.json: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error encoding activity data: %v", err)
	}

	fmt.Println("Updated activity.json with last fetch time.")
	return nil
}

// ReadLastFetchTime reads the last fetch time from the activity.json file.
func ReadLastFetchTime() (string, error) {
	file, err := os.Open("./activity.json")
	if err != nil {
		return "", fmt.Errorf("error opening activity.json: %v", err)
	}
	defer file.Close()

	var data ActivityData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding activity data: %v", err)
	}

	return data.LastFetchTime, nil
}
