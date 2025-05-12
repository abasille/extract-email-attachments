package internal

import (
	"crypto/sha256"
	"extract-email-attachments/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/gmail/v1"
)

func TestProcessAttachments(t *testing.T) {
	// Créer un dossier temporaire pour les tests
	tempDir, err := os.MkdirTemp("", "attachments-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Sauvegarder les valeurs originales
	originalAttachmentsDir := config.AppAttachmentsDir
	originalConfigDir := config.AppConfigDir
	config.AppAttachmentsDir = tempDir
	config.AppConfigDir = tempDir
	defer func() {
		config.AppAttachmentsDir = originalAttachmentsDir
		config.AppConfigDir = originalConfigDir
	}()

	// Créer une instance du gestionnaire d'activité
	am := NewActivityManager()
	err = am.Load()
	assert.NoError(t, err)

	// Créer un email de test
	emailID := "test-email-123"
	msg := &gmail.Message{
		Id: emailID,
		Payload: &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "Subject", Value: "Facture IKUTO"},
				{Name: "From", Value: "IKUTO <test@ikuto.com>"},
			},
		},
	}

	// Stocker les métadonnées de l'email
	err = am.StoreEmailMeta(emailID, msg)
	assert.NoError(t, err)

	// Créer un fichier PDF de test
	filename := "test.pdf"
	filePath := filepath.Join(tempDir, filename)
	fileContent := []byte("test content")
	sha256Hash := fmt.Sprintf("%x", sha256.Sum256(fileContent))
	err = os.WriteFile(filePath, fileContent, 0644)
	assert.NoError(t, err)

	// Stocker les métadonnées de la pièce jointe
	err = am.StoreAttachmentMeta(filename, emailID, sha256Hash)
	assert.NoError(t, err)

	// Tester le traitement des pièces jointes
	err = ProcessAttachments()
	assert.NoError(t, err)

	// Vérifier que le fichier a été renommé
	expectedFilename := time.Now().Format("2006-01") + "-facture-IKUTO.pdf"
	expectedPath := filepath.Join(tempDir, expectedFilename)
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err)
	// Vérifier que le statut a été mis à jour
	attachment, err := am.GetAttachment(sha256Hash)
	assert.NoError(t, err)
	assert.Equal(t, "processed", attachment.Status)
}

func TestProcessAttachmentsErrors(t *testing.T) {
	// Créer un dossier temporaire pour les tests
	tempDir, err := os.MkdirTemp("", "attachments-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Sauvegarder les valeurs originales
	originalAttachmentsDir := config.AppAttachmentsDir
	originalConfigDir := config.AppConfigDir
	config.AppAttachmentsDir = tempDir
	config.AppConfigDir = tempDir
	defer func() {
		config.AppAttachmentsDir = originalAttachmentsDir
		config.AppConfigDir = originalConfigDir
	}()

	// Tester avec un dossier de pièces jointes inexistant
	config.AppAttachmentsDir = filepath.Join(tempDir, "non-existent")
	err = ProcessAttachments()
	assert.Error(t, err)

	// Créer le dossier de pièces jointes
	err = os.MkdirAll(config.AppAttachmentsDir, 0755)
	assert.NoError(t, err)

	// Tester avec un fichier non-PDF
	nonPdfFile := filepath.Join(config.AppAttachmentsDir, "test.txt")
	err = os.WriteFile(nonPdfFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	err = ProcessAttachments()
	assert.NoError(t, err) // Ne devrait pas retourner d'erreur car les fichiers non-PDF sont ignorés

	// Tester avec un fichier PDF sans métadonnées associées
	pdfFile := filepath.Join(config.AppAttachmentsDir, "orphan.pdf")
	err = os.WriteFile(pdfFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	err = ProcessAttachments()
	assert.NoError(t, err) // Ne devrait pas retourner d'erreur car les fichiers sans métadonnées sont ignorés
}
