package internal

import (
	"errors"
	"fmt"
)

// Erreurs de base
var (
	ErrCritical          = errors.New("critical error")
	ErrInvalidFilename   = errors.New("invalid filename")
	ErrInvalidEmailID    = errors.New("invalid email ID")
	ErrInvalidAttachment = errors.New("invalid attachment")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidPath       = errors.New("invalid path")
)

// Erreurs spécifiques
var (
	ErrNotificationFailed   = errors.New("failed to display notification")
	ErrEmailProcessing      = errors.New("failed to process email")
	ErrAttachmentProcessing = errors.New("failed to process attachment")
	ErrOAuth2Failed         = errors.New("OAuth2 authentication failed")
	ErrGmailAPI             = errors.New("Gmail API error")
)

// Erreur enrichie avec contexte
type Error struct {
	Op  string // Opération qui a échoué
	Err error  // Erreur sous-jacente
	Msg string // Message d'erreur supplémentaire
}

func (e *Error) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewError crée une nouvelle erreur avec contexte
func NewError(op string, err error, msg string) error {
	return &Error{
		Op:  op,
		Err: err,
		Msg: msg,
	}
}

// IsCriticalError vérifie si l'erreur est critique
func IsCriticalError(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return errors.Is(e.Err, ErrCritical)
	}
	return errors.Is(err, ErrCritical)
}

// IsRetryableError vérifie si l'erreur peut être réessayée
func IsRetryableError(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		switch {
		case errors.Is(e.Err, ErrGmailAPI):
			return true
		case errors.Is(e.Err, ErrNotificationFailed):
			return true
		default:
			return false
		}
	}
	return false
}
