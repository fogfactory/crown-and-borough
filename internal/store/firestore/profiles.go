package firestorestore

import (
	"context"
	"strings"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

var _ store.ProfileStore = (*FirestoreStore)(nil)

func (s *FirestoreStore) GetProfile(ctx context.Context, uid string) (store.PlayerProfile, error) {
	if err := s.requireClient(); err != nil {
		return store.PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return store.PlayerProfile{}, store.ErrProfileNotFound
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	document, err := s.client.Collection("players").Doc(uid).Get(operationContext)
	s.recordReads(1)
	if err != nil {
		return store.PlayerProfile{}, mapReadError(err, store.ErrProfileNotFound)
	}
	profile, err := decodeProfileDocument(document)
	if err != nil {
		return store.PlayerProfile{}, err
	}
	return toPlayerProfile(profile), nil
}

func (s *FirestoreStore) EnsureProfile(ctx context.Context, actor store.Actor) (store.PlayerProfile, error) {
	// The server uses the Admin/ADC client here. Firestore rules intentionally
	// reject client writes so UID and email always come from the verified token.
	if err := s.requireClient(); err != nil {
		return store.PlayerProfile{}, err
	}
	uid := strings.TrimSpace(actor.ID)
	if uid == "" {
		return store.PlayerProfile{}, store.ErrProfileNotFound
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	doc := s.client.Collection("players").Doc(uid)
	return s.ensureProfile(operationContext, doc, actor)
}

func (s *FirestoreStore) ensureProfile(ctx context.Context, doc *cloudfirestore.DocumentRef, actor store.Actor) (store.PlayerProfile, error) {
	uid := strings.TrimSpace(actor.ID)
	email := strings.TrimSpace(actor.Email)
	var result store.PlayerProfile
	s.recordTransaction()
	err := s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		snapshot, err := transaction.Get(doc)
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			now := s.now().UTC()
			document := profileDocument{
				SchemaVersion: schemaVersion,
				UID:           uid,
				Email:         email,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := transaction.Create(doc, document); err != nil {
				return err
			}
			s.recordWrites(1)
			result = toPlayerProfile(document)
			return nil
		}
		if err != nil {
			return err
		}
		document, err := decodeProfileDocument(snapshot)
		if err != nil {
			return err
		}
		if email != "" && document.Email != email {
			document.Email = email
			document.UpdatedAt = s.now().UTC()
			if err := transaction.Set(doc, document); err != nil {
				return err
			}
			s.recordWrites(1)
		}
		result = toPlayerProfile(document)
		return nil
	})
	if err != nil {
		return store.PlayerProfile{}, wrapOperation("ensure profile", err)
	}
	return result, nil
}

func (s *FirestoreStore) UpdateProfile(ctx context.Context, uid, displayName string) (store.PlayerProfile, error) {
	if err := s.requireClient(); err != nil {
		return store.PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	normalized, err := store.NormalizeDisplayName(displayName)
	if err != nil {
		return store.PlayerProfile{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	result, err := s.mutateProfile(operationContext, uid, func(document *profileDocument) bool {
		if document.DisplayName == normalized {
			return false
		}
		document.DisplayName = normalized
		return true
	})
	if err != nil {
		return store.PlayerProfile{}, wrapOperation("update profile", err)
	}
	if err := s.refreshActorName(operationContext, uid, normalized); err != nil {
		return store.PlayerProfile{}, wrapOperation("refresh game profiles", err)
	}
	return result, nil
}

func (s *FirestoreStore) UpdateReleaseNotes(ctx context.Context, uid string, update store.ReleaseNotesUpdate) (store.PlayerProfile, error) {
	if err := s.requireClient(); err != nil {
		return store.PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	update, err := store.NormalizeReleaseNotesUpdate(update)
	if err != nil {
		return store.PlayerProfile{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	result, err := s.mutateProfile(operationContext, uid, func(document *profileDocument) bool {
		profile := toPlayerProfile(*document)
		if !store.ApplyReleaseNotesUpdate(&profile, update) {
			return false
		}
		document.LastSeenVersion = profile.LastSeenVersion
		document.HideReleaseNotes = profile.HideReleaseNotes
		return true
	})
	if err != nil {
		return store.PlayerProfile{}, wrapOperation("update release notes", err)
	}
	return result, nil
}

func (s *FirestoreStore) UpdateNotifications(ctx context.Context, uid string, update store.NotificationsUpdate) (store.PlayerProfile, error) {
	if err := s.requireClient(); err != nil {
		return store.PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	update, err := store.NormalizeNotificationsUpdate(update)
	if err != nil {
		return store.PlayerProfile{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	result, err := s.mutateProfile(operationContext, uid, func(document *profileDocument) bool {
		profile := toPlayerProfile(*document)
		if !store.ApplyNotificationsUpdate(&profile, update) {
			return false
		}
		document.Notifications = notificationsDocument(profile.Notifications)
		return true
	})
	if err != nil {
		return store.PlayerProfile{}, wrapOperation("update notifications", err)
	}
	return result, nil
}

// mutateProfile applies mutate to the player document in a transaction and
// writes it, with a fresh UpdatedAt, only when mutate reports a change.
func (s *FirestoreStore) mutateProfile(ctx context.Context, uid string, mutate func(*profileDocument) bool) (store.PlayerProfile, error) {
	doc := s.client.Collection("players").Doc(uid)
	var result store.PlayerProfile
	s.recordTransaction()
	err := s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		snapshot, err := transaction.Get(doc)
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrProfileNotFound
		}
		if err != nil {
			return err
		}
		document, err := decodeProfileDocument(snapshot)
		if err != nil {
			return err
		}
		if mutate(&document) {
			document.UpdatedAt = s.now().UTC()
			if err := transaction.Set(doc, document); err != nil {
				return err
			}
			s.recordWrites(1)
		}
		result = toPlayerProfile(document)
		return nil
	})
	return result, err
}

func (s *FirestoreStore) refreshActorName(ctx context.Context, uid, displayName string) error {
	documents := s.client.Collection("games").Where("memberUids", "array-contains", uid).Documents(ctx)
	for {
		document, err := documents.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		s.recordReads(1)
		game, err := decodeGameDocument(document)
		if err != nil {
			return err
		}
		if err := s.refreshGamePlayerName(ctx, game.ID, uid, displayName); err != nil {
			return err
		}
	}
}

// refreshGamePlayerName records the new display name as the account name of
// the member's houses. It never renames a house, and an incognito game, which
// never recorded an account name, has none to refresh.
func (s *FirestoreStore) refreshGamePlayerName(ctx context.Context, id store.GameID, uid, displayName string) error {
	s.recordTransaction()
	err := s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		profileSnapshot, err := transaction.Get(s.client.Collection("players").Doc(uid))
		s.recordReads(1)
		if err != nil {
			return err
		}
		profile, err := decodeProfileDocument(profileSnapshot)
		if err != nil {
			return err
		}
		displayName = profile.DisplayName
		gameSnapshot, err := transaction.Get(gameRef(s.client, id))
		s.recordReads(1)
		if err != nil {
			return err
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		if game.Incognito {
			return nil
		}
		changed := false
		for index := range game.Players {
			if game.Players[index].ActorID == uid && game.Players[index].AccountName != "" && game.Players[index].AccountName != displayName {
				game.Players[index].AccountName = displayName
				changed = true
			}
		}
		if !changed {
			return nil
		}
		game.UpdatedAt = s.now().UTC()
		s.recordWrites(1)
		return transaction.Set(gameRef(s.client, id), game)
	})
	return wrapTransactionResult(err)
}

func toPlayerProfile(document profileDocument) store.PlayerProfile {
	return store.PlayerProfile{
		UID:              document.UID,
		Email:            document.Email,
		DisplayName:      document.DisplayName,
		LastSeenVersion:  document.LastSeenVersion,
		HideReleaseNotes: document.HideReleaseNotes,
		Notifications:    store.NotificationPreferences(document.Notifications),
		CreatedAt:        document.CreatedAt,
		UpdatedAt:        document.UpdatedAt,
	}
}
