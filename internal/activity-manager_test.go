package internal

import (
	"extract-email-attachments/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/gmail/v1"
)

func TestActivityManager(t *testing.T) {
	// Créer un dossier temporaire pour les tests
	tempDir, err := os.MkdirTemp("", "activity-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Sauvegarder les valeurs originales
	originalConfigDir := config.AppConfigDir
	config.AppConfigDir = tempDir
	defer func() {
		config.AppConfigDir = originalConfigDir
	}()

	// Créer une nouvelle instance du gestionnaire d'activité
	am := NewActivityManager()

	// Tester le chargement initial
	err = am.Load()
	assert.NoError(t, err)
	assert.NotNil(t, am.data)

	// Tester le stockage des métadonnées d'email
	emailID := "test-email-123"
	msg := &gmail.Message{
		Id: emailID,
		Payload: &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "Subject", Value: "Test Subject"},
				{Name: "From", Value: "test@example.com"},
			},
		},
	}
	err = am.StoreEmailMeta(emailID, msg)
	assert.NoError(t, err)

	// Vérifier que l'email a été stocké
	assert.True(t, am.HasEmailID(emailID))

	// Tester la récupération des métadonnées d'email
	email, err := am.GetEmailByID(emailID)
	assert.NoError(t, err)
	assert.Equal(t, emailID, email.ID)
	assert.Equal(t, "Test Subject", email.Subject)
	assert.Equal(t, "test@example.com", email.SenderEmail)

	// Tester le stockage des métadonnées de pièce jointe
	filename := "test.pdf"
	sha256Hash := "test-sha256-hash"
	err = am.StoreAttachmentMeta(filename, emailID, sha256Hash)
	assert.NoError(t, err)

	// Vérifier que la pièce jointe a été stockée
	attachment, err := am.GetAttachment(sha256Hash)
	assert.NoError(t, err)
	assert.Equal(t, filename, attachment.Filename)
	assert.Equal(t, emailID, attachment.EmailID)

	// Tester la mise à jour du statut de la pièce jointe
	err = am.UpdateAttachmentStatus(filename, "processed")
	assert.NoError(t, err)

	// Vérifier que le statut a été mis à jour
	attachment, err = am.GetAttachment(sha256Hash)
	assert.NoError(t, err)
	assert.Equal(t, "processed", attachment.Status)

	// Tester le stockage de la dernière heure de récupération
	now := time.Now().Format(config.DefaultDateFormat)
	err = am.StoreLastFetchTime()
	assert.NoError(t, err)

	// Vérifier que l'heure a été stockée
	lastFetchTime, err := am.ReadLastFetchTime()
	assert.NoError(t, err)
	assert.Equal(t, now, lastFetchTime)

	// Tester la sauvegarde des données
	err = am.Save()
	assert.NoError(t, err)

	// Vérifier que le fichier de données a été créé
	dataFile := filepath.Join(tempDir, "activity.json")
	_, err = os.Stat(dataFile)
	assert.NoError(t, err)
}

func TestActivityManagerErrors(t *testing.T) {
	// Créer un dossier temporaire pour les tests
	tempDir, err := os.MkdirTemp("", "activity-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Sauvegarder les valeurs originales
	originalConfigDir := config.AppConfigDir
	config.AppConfigDir = tempDir
	defer func() {
		config.AppConfigDir = originalConfigDir
	}()

	// Créer une nouvelle instance du gestionnaire d'activité
	am := NewActivityManager()

	// Tester la récupération d'un email inexistant
	_, err = am.GetEmailByID("non-existent")
	assert.Error(t, err)

	// Tester la récupération d'une pièce jointe inexistante
	_, err = am.GetAttachment("non-existent")
	assert.Error(t, err)

	// Tester la mise à jour du statut d'une pièce jointe inexistante
	err = am.UpdateAttachmentStatus("non-existent.pdf", "processed")
	assert.Error(t, err)
}
