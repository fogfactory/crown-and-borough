package store

import (
	"context"
	"strings"
	"time"
)

// Languages an email notification can be written in. French is the default
// when the player's language is unknown.
const (
	NotificationLanguageFrench  = "fr"
	NotificationLanguageEnglish = "en"
)

// NotificationPreferences are the opt-in email notification settings of a
// player. Every field defaults to off: a profile saved before the feature
// existed has none of them and never triggers an email.
type NotificationPreferences struct {
	// NewTurn asks for an email when a new turn starts in one of the player's
	// games.
	NewTurn bool `json:"newTurn"`
	// PrivateMessage asks for an email when the player's house receives a
	// private chat message.
	PrivateMessage bool `json:"privateMessage"`
	// PublicMessage asks for an email when another house writes in the public
	// room of one of the player's games.
	PublicMessage bool `json:"publicMessage"`
	// ReduceWhenActive skips an email when the player was active when the
	// event happened. It has no effect while the three notifications above
	// are all off.
	ReduceWhenActive bool `json:"reduceWhenActive"`
	// Language is the language of the emails (fr or en), chosen in the
	// profile; empty means French.
	Language string `json:"language"`
}

// NotificationsUpdate carries the notification settings to change; nil fields
// are left untouched.
type NotificationsUpdate struct {
	NewTurn          *bool
	PrivateMessage   *bool
	PublicMessage    *bool
	ReduceWhenActive *bool
	Language         *string
}

// NormalizeNotificationsUpdate validates an update and returns it with a
// canonical language code.
func NormalizeNotificationsUpdate(update NotificationsUpdate) (NotificationsUpdate, error) {
	if update.NewTurn == nil && update.PrivateMessage == nil && update.PublicMessage == nil &&
		update.ReduceWhenActive == nil && update.Language == nil {
		return NotificationsUpdate{}, ErrInvalidNotifications
	}
	if update.Language == nil {
		return update, nil
	}
	language := strings.ToLower(strings.TrimSpace(*update.Language))
	if language != NotificationLanguageFrench && language != NotificationLanguageEnglish {
		return NotificationsUpdate{}, ErrInvalidNotifications
	}
	update.Language = &language
	return update, nil
}

// ApplyNotificationsUpdate mutates profile with a normalized update and
// reports whether anything changed.
func ApplyNotificationsUpdate(profile *PlayerProfile, update NotificationsUpdate) bool {
	changed := false
	apply := func(target *bool, value *bool) {
		if value != nil && *target != *value {
			*target = *value
			changed = true
		}
	}
	apply(&profile.Notifications.NewTurn, update.NewTurn)
	apply(&profile.Notifications.PrivateMessage, update.PrivateMessage)
	apply(&profile.Notifications.PublicMessage, update.PublicMessage)
	apply(&profile.Notifications.ReduceWhenActive, update.ReduceWhenActive)
	if update.Language != nil && profile.Notifications.Language != *update.Language {
		profile.Notifications.Language = *update.Language
		changed = true
	}
	return changed
}

// NotificationStateStore is the backend-only state the email notifications
// need: the last time each player used the API and the anti-spam marker of the
// message emails. Clients never read it.
type NotificationStateStore interface {
	// TouchActivity records that uid made an authenticated API request at at.
	TouchActivity(ctx context.Context, uid string, at time.Time) error
	// LastActivity returns the last recorded activity of uid; the zero time
	// when there is none.
	LastActivity(ctx context.Context, uid string) (time.Time, error)
	// ClaimMessageEmail atomically decides whether uid may receive a message
	// email for the game at at, and records it when so. It reports false when a
	// message email was already sent and the player has not read their chat
	// since (see MessageEmailClaimable).
	ClaimMessageEmail(ctx context.Context, id GameID, uid string, at time.Time) (bool, error)
	// ReleaseMessageEmail gives back the claim made at at when its email
	// could not be sent, so the next message may claim again. It does nothing
	// when the recorded claim is another one.
	ReleaseMessageEmail(ctx context.Context, id GameID, uid string, at time.Time) error
}

// MessageEmailClaimable is the anti-spam rule of the message emails: a new one
// is allowed when none was sent yet, or when one of the player's chat read
// cursors of the game is later than the last one.
func MessageEmailClaimable(lastEmailAt time.Time, readCursors []time.Time) bool {
	if lastEmailAt.IsZero() {
		return true
	}
	for _, readAt := range readCursors {
		if readAt.After(lastEmailAt) {
			return true
		}
	}
	return false
}

// ChatPostResult is a stored message with what the notifications need: the
// game and the uids of the other houses concerned (the other house of a
// private conversation, or every other house for the public room). The author
// and the observer are never listed.
type ChatPostResult struct {
	Message       ChatMessage
	GameID        GameID
	GameName      string
	RecipientUIDs []string
}
