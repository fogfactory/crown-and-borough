package firestorestore

import (
	"context"
	"strings"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fogfactory/crown-and-borough/internal/models"
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
	doc := s.client.Collection("players").Doc(uid)
	var result store.PlayerProfile
	s.recordTransaction()
	err = s.client.RunTransaction(operationContext, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
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
		if document.DisplayName != normalized {
			document.DisplayName = normalized
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
		return store.PlayerProfile{}, wrapOperation("update profile", err)
	}
	if err := s.refreshActorName(operationContext, uid, normalized); err != nil {
		return store.PlayerProfile{}, wrapOperation("refresh game profiles", err)
	}
	return result, nil
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
		playerID := models.PlayerID("")
		for index := range game.Players {
			if game.Players[index].ActorID == uid {
				playerID = game.Players[index].ID
				game.Players[index].Name = displayName
			}
		}
		if playerID == "" {
			return nil
		}
		canonicalSnapshot, err := transaction.Get(canonicalRef(s.client, id))
		s.recordReads(1)
		if err != nil {
			return err
		}
		canonical, err := decodeCanonicalDocument(canonicalSnapshot)
		if err != nil {
			return err
		}
		state := &models.GameState{}
		if err := decodeJSONMap(canonical.State, state); err != nil {
			return err
		}
		for index := range state.Players {
			if state.Players[index].ID == playerID {
				state.Players[index].Name = displayName
			}
		}
		if err := state.Validate(); err != nil {
			return err
		}
		now := s.now().UTC()
		game.UpdatedAt = now
		canonical.State, err = jsonMap(state)
		if err != nil {
			return err
		}
		canonical.UpdatedAt = now
		if err := transaction.Set(gameRef(s.client, id), game); err != nil {
			return err
		}
		if err := transaction.Set(canonicalRef(s.client, id), canonical); err != nil {
			return err
		}
		for _, player := range game.Players {
			if !isAssignedActor(player.ActorID) {
				continue
			}
			view, err := s.viewDocument(id, player.ActorID, player.ID, store.Revision(game.Revision), state, now)
			if err != nil {
				return err
			}
			if err := transaction.Set(viewRef(s.client, id, player.ActorID), view); err != nil {
				return err
			}
		}
		s.recordWrites(2 + assignedDocumentPlayerCount(game.Players))
		s.recordProjectionWrites(assignedDocumentPlayerCount(game.Players))
		return nil
	})
	return wrapTransactionResult(err)
}

func toPlayerProfile(document profileDocument) store.PlayerProfile {
	return store.PlayerProfile{
		UID:         document.UID,
		Email:       document.Email,
		DisplayName: document.DisplayName,
		CreatedAt:   document.CreatedAt,
		UpdatedAt:   document.UpdatedAt,
	}
}
