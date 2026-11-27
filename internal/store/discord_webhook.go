package store

import (
	"context"
	"time"
)

// Reasons a Discord webhook was disabled by the server.
const (
	// DiscordDisabledGone means Discord answered 404 or 401: the webhook was
	// deleted on the Discord side or its token is no longer valid.
	DiscordDisabledGone = "webhook_gone"
)

// DiscordTemplates are the announcement templates of the host, one per event.
// An empty template stands for the default one of the language.
type DiscordTemplates struct {
	NewTurn       string `json:"newTurn" firestore:"newTurn"`
	PublicMessage string `json:"publicMessage" firestore:"publicMessage"`
}

// DiscordWebhookConfig is the Discord webhook of one game. URL is a write
// credential on the host's channel: it is stored backend-only and never
// returned to a client or logged.
type DiscordWebhookConfig struct {
	URL       string
	Language  string
	Templates DiscordTemplates
	// Disabled is set by the server when Discord refused the webhook;
	// DisabledReason tells why (DiscordDisabledGone).
	Disabled       bool
	DisabledReason string
	DisabledAt     time.Time
	UpdatedAt      time.Time
	// LastPublicAt is the anti-spam marker of the Discord channel: the last
	// public room announcement of the game.
	LastPublicAt time.Time
}

// Active reports whether announcements are sent.
func (c DiscordWebhookConfig) Active() bool { return c.URL != "" && !c.Disabled }

// DiscordWebhookStore is the backend-only persistence of the per-game Discord
// webhook and of its anti-spam marker. Clients never read it.
type DiscordWebhookStore interface {
	// DiscordWebhook returns the webhook of the game; false when none is set.
	DiscordWebhook(ctx context.Context, id GameID) (DiscordWebhookConfig, bool, error)
	// SaveDiscordWebhook replaces the webhook of the game. The anti-spam
	// marker of the stored configuration is kept.
	SaveDiscordWebhook(ctx context.Context, id GameID, config DiscordWebhookConfig) error
	// DeleteDiscordWebhook removes the webhook; it is not an error when none
	// is set.
	DeleteDiscordWebhook(ctx context.Context, id GameID) error
	// DisableDiscordWebhook marks the webhook disabled, unless it was replaced
	// since url was read (the stored URL differs).
	DisableDiscordWebhook(ctx context.Context, id GameID, url, reason string, at time.Time) error
	// ClaimDiscordPublic atomically decides whether a public room
	// announcement may be sent at at, and records it when so. It reports
	// false when the previous one is less than minInterval old.
	ClaimDiscordPublic(ctx context.Context, id GameID, at time.Time, minInterval time.Duration) (bool, error)
	// ReleaseDiscordPublic gives back the claim made at at when the
	// announcement could not be sent. It does nothing when the recorded claim
	// is another one.
	ReleaseDiscordPublic(ctx context.Context, id GameID, at time.Time) error
}
