// Package notify sends the opt-in email notifications (new turn, chat
// messages). It knows nothing of the persistence: the stores it reads are
// small interfaces and the transport hides behind Mailer, so either can be
// swapped without touching the rest.
package notify

import (
	"context"
	"log"
)

// Email is one plain text message to one recipient.
type Email struct {
	To      string
	Subject string
	Body    string
	// Reference identifies the email in the logs (kind, game, uid). It never
	// holds an address or message content.
	Reference string
}

// Mailer delivers an email. Implementations must honour the context deadline.
type Mailer interface {
	Send(ctx context.Context, email Email) error
}

// LogMailer is the mailer used when SMTP is not configured: it logs one line
// per skipped email, with no address and no content.
type LogMailer struct{}

var _ Mailer = LogMailer{}

func (LogMailer) Send(_ context.Context, email Email) error {
	log.Printf("notify: email skipped, SMTP is not configured (%s)", email.Reference)
	return nil
}
