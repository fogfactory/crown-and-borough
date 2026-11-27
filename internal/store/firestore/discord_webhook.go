package firestorestore

import (
	"context"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

var _ store.DiscordWebhookStore = (*FirestoreStore)(nil)

// The Discord webhook of a game lives in games/{gameId}/discord_webhook/config.
// The URL is a write credential on the host's channel: clients can neither read
// nor write it; see firestore.rules.
const (
	discordWebhookCollection = "discord_webhook"
	discordWebhookDocument   = "config"
)

type discordWebhookData struct {
	SchemaVersion  int                    `firestore:"schemaVersion"`
	URL            string                 `firestore:"url"`
	Language       string                 `firestore:"language"`
	Templates      store.DiscordTemplates `firestore:"templates"`
	Disabled       bool                   `firestore:"disabled"`
	DisabledReason string                 `firestore:"disabledReason"`
	DisabledAt     time.Time              `firestore:"disabledAt"`
	UpdatedAt      time.Time              `firestore:"updatedAt"`
	LastPublicAt   time.Time              `firestore:"lastPublicAt"`
}

func (d discordWebhookData) config() store.DiscordWebhookConfig {
	return store.DiscordWebhookConfig{
		URL:            d.URL,
		Language:       d.Language,
		Templates:      d.Templates,
		Disabled:       d.Disabled,
		DisabledReason: d.DisabledReason,
		DisabledAt:     d.DisabledAt.UTC(),
		UpdatedAt:      d.UpdatedAt.UTC(),
		LastPublicAt:   d.LastPublicAt.UTC(),
	}
}

func discordWebhookRef(client *cloudfirestore.Client, id store.GameID) *cloudfirestore.DocumentRef {
	return gameRef(client, id).Collection(discordWebhookCollection).Doc(discordWebhookDocument)
}

func (s *FirestoreStore) DiscordWebhook(ctx context.Context, id store.GameID) (store.DiscordWebhookConfig, bool, error) {
	if err := s.requireClient(); err != nil {
		return store.DiscordWebhookConfig{}, false, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	snapshot, err := discordWebhookRef(s.client, id).Get(operationContext)
	s.recordReads(1)
	if isCode(err, codes.NotFound) {
		return store.DiscordWebhookConfig{}, false, nil
	}
	if err != nil {
		return store.DiscordWebhookConfig{}, false, wrapOperation("read discord webhook", err)
	}
	var document discordWebhookData
	if err := snapshot.DataTo(&document); err != nil {
		return store.DiscordWebhookConfig{}, false, wrapOperation("decode discord webhook", err)
	}
	return document.config(), true, nil
}

func (s *FirestoreStore) SaveDiscordWebhook(ctx context.Context, id store.GameID, config store.DiscordWebhookConfig) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	ref := discordWebhookRef(s.client, id)
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(_ context.Context, transaction *cloudfirestore.Transaction) error {
		var lastPublic time.Time
		snapshot, err := transaction.Get(ref)
		switch {
		case err == nil:
			var previous discordWebhookData
			if err := snapshot.DataTo(&previous); err != nil {
				return err
			}
			lastPublic = previous.LastPublicAt
		case !isCode(err, codes.NotFound):
			return err
		}
		return transaction.Set(ref, discordWebhookData{
			SchemaVersion:  schemaVersion,
			URL:            config.URL,
			Language:       config.Language,
			Templates:      config.Templates,
			Disabled:       config.Disabled,
			DisabledReason: config.DisabledReason,
			DisabledAt:     config.DisabledAt.UTC(),
			UpdatedAt:      config.UpdatedAt.UTC(),
			LastPublicAt:   lastPublic,
		})
	})
	s.recordReads(1)
	if err != nil {
		return wrapOperation("save discord webhook", err)
	}
	s.recordWrites(1)
	return nil
}

func (s *FirestoreStore) DeleteDiscordWebhook(ctx context.Context, id store.GameID) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	_, err := discordWebhookRef(s.client, id).Delete(operationContext)
	if err != nil {
		return wrapOperation("delete discord webhook", err)
	}
	s.recordWrites(1)
	return nil
}

// updateDiscordWebhook runs one read-modify-write of the webhook document in a
// transaction. mutate edits the document and reports whether it must be
// written; a missing document is left alone. It returns whether a write was
// made. operation names the call in the errors.
func (s *FirestoreStore) updateDiscordWebhook(ctx context.Context, id store.GameID, operation string, mutate func(*discordWebhookData) bool) (bool, error) {
	if err := s.requireClient(); err != nil {
		return false, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	ref := discordWebhookRef(s.client, id)
	written := false
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(_ context.Context, transaction *cloudfirestore.Transaction) error {
		written = false
		snapshot, err := transaction.Get(ref)
		if isCode(err, codes.NotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var document discordWebhookData
		if err := snapshot.DataTo(&document); err != nil {
			return err
		}
		if !mutate(&document) {
			return nil
		}
		written = true
		return transaction.Set(ref, document)
	})
	s.recordReads(1)
	if err != nil {
		return false, wrapOperation(operation, err)
	}
	if written {
		s.recordWrites(1)
	}
	return written, nil
}

func (s *FirestoreStore) DisableDiscordWebhook(ctx context.Context, id store.GameID, url, reason string, at time.Time) error {
	_, err := s.updateDiscordWebhook(ctx, id, "disable discord webhook", func(document *discordWebhookData) bool {
		// The host replaced the webhook meanwhile: the new one is not at fault.
		if document.URL != url {
			return false
		}
		document.Disabled = true
		document.DisabledReason = reason
		document.DisabledAt = at.UTC()
		return true
	})
	return err
}

func (s *FirestoreStore) ClaimDiscordPublic(ctx context.Context, id store.GameID, at time.Time, minInterval time.Duration) (bool, error) {
	return s.updateDiscordWebhook(ctx, id, "claim discord announcement", func(document *discordWebhookData) bool {
		if !document.LastPublicAt.IsZero() && at.Sub(document.LastPublicAt) < minInterval {
			return false
		}
		document.LastPublicAt = at.UTC()
		return true
	})
}

func (s *FirestoreStore) ReleaseDiscordPublic(ctx context.Context, id store.GameID, at time.Time) error {
	// Firestore keeps microseconds: compare with what was stored.
	claimedAt := at.UTC().Truncate(time.Microsecond)
	_, err := s.updateDiscordWebhook(ctx, id, "release discord announcement", func(document *discordWebhookData) bool {
		if !document.LastPublicAt.Equal(claimedAt) {
			return false
		}
		document.LastPublicAt = time.Time{}
		return true
	})
	return err
}
