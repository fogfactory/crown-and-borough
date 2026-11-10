package firestorestore

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

var _ store.NotificationStateStore = (*FirestoreStore)(nil)

// Backend-only state of the email notifications:
//
//	player_activity/{uid}              last authenticated API request
//	games/{gameId}/notify_state/{uid}  last message email of the player
//
// Clients can read neither; see firestore.rules.
const (
	playerActivityCollection = "player_activity"
	notifyStateCollection    = "notify_state"
)

type playerActivityDocument struct {
	SchemaVersion int       `firestore:"schemaVersion"`
	LastActiveAt  time.Time `firestore:"lastActiveAt"`
}

type notifyStateDocument struct {
	SchemaVersion  int       `firestore:"schemaVersion"`
	MessageEmailAt time.Time `firestore:"messageEmailAt"`
}

func (s *FirestoreStore) TouchActivity(ctx context.Context, uid string, at time.Time) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return store.ErrProfileNotFound
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	_, err := s.client.Collection(playerActivityCollection).Doc(uid).Set(operationContext, playerActivityDocument{
		SchemaVersion: schemaVersion,
		LastActiveAt:  at.UTC(),
	})
	if err != nil {
		return wrapOperation("touch activity", err)
	}
	s.recordWrites(1)
	return nil
}

func (s *FirestoreStore) LastActivity(ctx context.Context, uid string) (time.Time, error) {
	if err := s.requireClient(); err != nil {
		return time.Time{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	snapshot, err := s.client.Collection(playerActivityCollection).Doc(strings.TrimSpace(uid)).Get(operationContext)
	s.recordReads(1)
	if isCode(err, codes.NotFound) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, wrapOperation("read activity", err)
	}
	var document playerActivityDocument
	if err := snapshot.DataTo(&document); err != nil {
		return time.Time{}, wrapOperation("decode activity", err)
	}
	return document.LastActiveAt.UTC(), nil
}

func (s *FirestoreStore) ClaimMessageEmail(ctx context.Context, id store.GameID, uid string, at time.Time) (bool, error) {
	if err := s.requireClient(); err != nil {
		return false, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	game := gameRef(s.client, id)
	stateRef := game.Collection(notifyStateCollection).Doc(uid)
	readsRef := game.Collection(chatReadsCollection).Doc(uid)
	claimed := false
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(_ context.Context, transaction *cloudfirestore.Transaction) error {
		claimed = false
		var last time.Time
		snapshot, err := transaction.Get(stateRef)
		switch {
		case err == nil:
			var state notifyStateDocument
			if err := snapshot.DataTo(&state); err != nil {
				return err
			}
			last = state.MessageEmailAt
		case !isCode(err, codes.NotFound):
			return err
		}
		var cursors []time.Time
		snapshot, err = transaction.Get(readsRef)
		switch {
		case err == nil:
			var reads chatReadsDocument
			if err := snapshot.DataTo(&reads); err != nil {
				return err
			}
			cursors = slices.Collect(maps.Values(reads.Channels))
		case !isCode(err, codes.NotFound):
			return err
		}
		s.recordReads(2)
		if !store.MessageEmailClaimable(last, cursors) {
			return nil
		}
		claimed = true
		return transaction.Set(stateRef, notifyStateDocument{SchemaVersion: schemaVersion, MessageEmailAt: at.UTC()})
	})
	if err != nil {
		return false, wrapOperation("claim message email", err)
	}
	if claimed {
		s.recordWrites(1)
	}
	return claimed, nil
}

func (s *FirestoreStore) ReleaseMessageEmail(ctx context.Context, id store.GameID, uid string, at time.Time) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	stateRef := gameRef(s.client, id).Collection(notifyStateCollection).Doc(uid)
	// Firestore keeps microseconds: compare with what was stored.
	claimedAt := at.UTC().Truncate(time.Microsecond)
	released := false
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(_ context.Context, transaction *cloudfirestore.Transaction) error {
		released = false
		snapshot, err := transaction.Get(stateRef)
		if isCode(err, codes.NotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var state notifyStateDocument
		if err := snapshot.DataTo(&state); err != nil {
			return err
		}
		if !state.MessageEmailAt.Equal(claimedAt) {
			return nil
		}
		released = true
		return transaction.Delete(stateRef)
	})
	s.recordReads(1)
	if err != nil {
		return wrapOperation("release message email", err)
	}
	if released {
		s.recordWrites(1)
	}
	return nil
}
