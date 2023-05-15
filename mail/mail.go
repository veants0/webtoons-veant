package mail

import (
	"errors"
)

var (
	ErrNotFound = errors.New("mail not found")
)

// Mailer interface
type Mailer interface {
	// GetContent should take the email address and return the received content
	// if the mail wasn't be found in time the function MUST return ErrNotFound
	GetContent(address string) (string, error)

	RandomAddress() string
}
