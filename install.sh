#!/bin/bash

# Application name
APP_NAME="extract-email-attachments"

# Compile the Go binary
echo "Compiling Go binary..."
go build -o ~/.bin/$APP_NAME

# Make the binary executable
chmod +x ~/.bin/$APP_NAME

echo "Binary installed at: ~/.bin/$APP_NAME"

# Add cron job if it doesn't exist
CRON_JOB="*/10 * * * * ~/.bin/$APP_NAME"
if ! (crontab -l 2>/dev/null | grep "$APP_NAME"); then
    (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
    echo "Cron job added successfully"
else
    echo "Cron job already exists"
fi

echo "Installation complete!"
