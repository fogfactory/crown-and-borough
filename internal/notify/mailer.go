// Package notify sends the notifications (new turn, chat messages) by email and
// to the Discord webhook of a game. It knows nothing of the persistence: the stores it reads are
// small interfaces and each transport hides behind a Sender, so either can be
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

// Sender delivers one message of a delivery channel (email, Discord webhook,
// later direct messages). Implementations must honour the context deadline and
// must not put secrets (addresses, webhook URLs) in the errors they return.
type Sender[M any] interface {
	Send(ctx context.Context, message M) error
}

// Mailer is the email channel.
type Mailer interface {
	Sender[Email]
}

// LogMailer is the mailer used when SMTP is not configured: it logs one line
// per skipped email, with no address and no content.
type LogMailer struct{}

var _ Mailer = LogMailer{}

func (LogMailer) Send(_ context.Context, email Email) error {
	log.Printf("notify: email skipped, SMTP is not configured (%s)", email.Reference)
	return nil
}
