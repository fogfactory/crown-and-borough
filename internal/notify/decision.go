package notify

import (
	"time"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

// ActiveWindow is how long after its last API request a player still counts
// as active, for the "reduce notifications" option.
const ActiveWindow = 5 * time.Minute

// Kind is the event an email announces.
type Kind string

const (
	KindNewTurn        Kind = "new_turn"
	KindPrivateMessage Kind = "private_message"
	KindPublicMessage  Kind = "public_message"
)

// IsMessage reports whether the kind is a chat message, which the anti-spam
// rule applies to.
func (k Kind) IsMessage() bool { return k == KindPrivateMessage || k == KindPublicMessage }

// WasActive reports whether a player whose last API request was at lastActive
// counts as active at eventAt. A player never seen is not active.
func WasActive(lastActive, eventAt time.Time) bool {
	return !lastActive.IsZero() && eventAt.Sub(lastActive) <= ActiveWindow
}

// ShouldSend decides whether a player gets the email of an event, ignoring the
// message anti-spam claim: the matching opt-in is on, the profile has an
// address and, with the reduce option on, the player was not active.
// Preferences left at their zero value never send.
func ShouldSend(preferences store.NotificationPreferences, email string, kind Kind, lastActive, eventAt time.Time) bool {
	if email == "" {
		return false
	}
	var optedIn bool
	switch kind {
	case KindNewTurn:
		optedIn = preferences.NewTurn
	case KindPrivateMessage:
		optedIn = preferences.PrivateMessage
	case KindPublicMessage:
		optedIn = preferences.PublicMessage
	}
	if !optedIn {
		return false
	}
	return !(preferences.ReduceWhenActive && WasActive(lastActive, eventAt))
}
