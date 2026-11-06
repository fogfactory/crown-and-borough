package firestorestore

import (
	"context"
	"errors"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

var (
	_ store.ChatBackend = (*FirestoreStore)(nil)
	_ store.RosterStore = (*FirestoreStore)(nil)
)

// Chat layout under games/{gameId}:
//
//	chat_public/{messageId}                    public room
//	chat_private/{conversationKey}/messages/{messageId}
//	                                           one conversation between two
//	                                           houses; the key is the sorted
//	                                           house identifiers (P1_P3)
//	chat_reads/{uid}                           the player's read cursors
//	chat_state/{uid}                           rate-limit state (backend only)
//
// Clients only read; every write goes through the Go API.
const (
	chatPublicCollection   = "chat_public"
	chatPrivateCollection  = "chat_private"
	chatMessagesCollection = "messages"
	chatReadsCollection    = "chat_reads"
	chatStateCollection    = "chat_state"
)

type chatTurnDocument struct {
	Number int           `firestore:"number"`
	Season models.Season `firestore:"season"`
	Year   int           `firestore:"year"`
}

// chatMessageDocument never records an account: the author is a house.
// ParticipantUIDs is set on private messages only; the security rules match
// it against the reader's uid, so clients must query a conversation with
// where("participantUids", "array-contains", uid).
type chatMessageDocument struct {
	SchemaVersion   int              `firestore:"schemaVersion"`
	ID              string           `firestore:"id"`
	Channel         string           `firestore:"channel"`
	AuthorHouseID   string           `firestore:"authorHouseId"`
	Body            string           `firestore:"body"`
	CreatedAt       time.Time        `firestore:"createdAt"`
	Turn            chatTurnDocument `firestore:"turn"`
	BatchID         string           `firestore:"batchId,omitempty"`
	Part            int              `firestore:"part,omitempty"`
	Parts           int              `firestore:"parts,omitempty"`
	ParticipantUIDs []string         `firestore:"participantUids,omitempty"`
}

type chatReadsDocument struct {
	SchemaVersion int                  `firestore:"schemaVersion"`
	UID           string               `firestore:"uid"`
	Channels      map[string]time.Time `firestore:"channels"`
}

type chatStateDocument struct {
	LastSentAt time.Time `firestore:"lastSentAt"`
}

func chatMessagesCollectionRef(client *cloudfirestore.Client, id store.GameID, channel string) *cloudfirestore.CollectionRef {
	if channel == store.ChatPublicChannel {
		return gameRef(client, id).Collection(chatPublicCollection)
	}
	return gameRef(client, id).Collection(chatPrivateCollection).Doc(channel).Collection(chatMessagesCollection)
}

func (s *FirestoreStore) AppendChatMessage(ctx context.Context, id store.GameID, authorUID string, message store.ChatMessage, participantUIDs []string, minInterval time.Duration) (store.ChatMessage, error) {
	if err := s.requireClient(); err != nil {
		return store.ChatMessage{}, err
	}
	messageID, err := store.NewChatMessageID(message.CreatedAt)
	if err != nil {
		return store.ChatMessage{}, err
	}
	message.ID = messageID
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	stateRef := gameRef(s.client, id).Collection(chatStateCollection).Doc(authorUID)
	messageRef := chatMessagesCollectionRef(s.client, id, message.Channel).Doc(messageID)
	document := chatMessageDocument{
		SchemaVersion: schemaVersion,
		ID:            message.ID,
		Channel:       message.Channel,
		AuthorHouseID: string(message.AuthorHouseID),
		Body:          message.Body,
		CreatedAt:     message.CreatedAt.UTC(),
		Turn: chatTurnDocument{
			Number: message.Turn.Number,
			Season: message.Turn.Season,
			Year:   message.Turn.Year,
		},
		BatchID:         message.BatchID,
		Part:            message.Part,
		Parts:           message.Parts,
		ParticipantUIDs: participantUIDs,
	}
	s.recordTransaction()
	err = s.client.RunTransaction(operationContext, func(_ context.Context, transaction *cloudfirestore.Transaction) error {
		snapshot, err := transaction.Get(stateRef)
		switch {
		case err == nil:
			var state chatStateDocument
			if err := snapshot.DataTo(&state); err != nil {
				return err
			}
			if message.CreatedAt.Sub(state.LastSentAt) < minInterval {
				return store.ErrChatRateLimited
			}
		case !isCode(err, codes.NotFound):
			return err
		}
		if err := transaction.Set(stateRef, chatStateDocument{LastSentAt: message.CreatedAt.UTC()}); err != nil {
			return err
		}
		return transaction.Create(messageRef, document)
	})
	if errors.Is(err, store.ErrChatRateLimited) {
		return store.ChatMessage{}, err
	}
	if err != nil {
		return store.ChatMessage{}, wrapOperation("append chat message", err)
	}
	s.recordWrites(2)
	return message, nil
}

func (s *FirestoreStore) SetChatReadCursor(ctx context.Context, id store.GameID, uid string, cursor store.ChatReadCursor) (store.ChatReadCursor, error) {
	if err := s.requireClient(); err != nil {
		return store.ChatReadCursor{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	ref := gameRef(s.client, id).Collection(chatReadsCollection).Doc(uid)
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(_ context.Context, transaction *cloudfirestore.Transaction) error {
		snapshot, err := transaction.Get(ref)
		switch {
		case err == nil:
			var current chatReadsDocument
			if err := snapshot.DataTo(&current); err != nil {
				return err
			}
			if previous, ok := current.Channels[cursor.Channel]; ok && previous.After(cursor.LastReadAt) {
				cursor.LastReadAt = previous
			}
		case !isCode(err, codes.NotFound):
			return err
		}
		// A field path keeps the cursors of the other channels untouched.
		return transaction.Set(ref, map[string]any{
			"schemaVersion": schemaVersion,
			"uid":           uid,
			"channels":      map[string]any{cursor.Channel: cursor.LastReadAt.UTC()},
		}, cloudfirestore.MergeAll)
	})
	if err != nil {
		return store.ChatReadCursor{}, wrapOperation("mark chat channel read", err)
	}
	s.recordWrites(1)
	return cursor, nil
}

func (s *FirestoreStore) ListChatMessages(ctx context.Context, id store.GameID, channel string, before store.ChatCursor, limit int) ([]store.ChatMessage, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	// Ordering by createdAt then by document ID (time-sortable, see
	// store.NewChatMessageID) keeps pages stable; the front end uses the same
	// query with where("participantUids", "array-contains", uid) on private
	// conversations (index in firestore.indexes.json).
	query := chatMessagesCollectionRef(s.client, id, channel).
		OrderBy("createdAt", cloudfirestore.Desc).
		OrderBy(cloudfirestore.DocumentID, cloudfirestore.Desc)
	if !before.IsZero() {
		query = query.StartAfter(before.CreatedAt.UTC(), before.ID)
	}
	iter := query.Limit(limit).Documents(operationContext)
	defer iter.Stop()
	messages := make([]store.ChatMessage, 0, limit)
	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return messages, nil
		}
		if err != nil {
			return nil, wrapOperation("list chat messages", err)
		}
		var document chatMessageDocument
		if err := snapshot.DataTo(&document); err != nil {
			return nil, wrapOperation("decode chat message", err)
		}
		messages = append(messages, store.ChatMessage{
			ID:            document.ID,
			Channel:       document.Channel,
			AuthorHouseID: models.PlayerID(document.AuthorHouseID),
			Body:          document.Body,
			CreatedAt:     document.CreatedAt.UTC(),
			Turn:          store.ChatTurn{Number: document.Turn.Number, Season: document.Turn.Season, Year: document.Turn.Year},
			BatchID:       document.BatchID,
			Part:          document.Part,
			Parts:         document.Parts,
		})
	}
}

// Roster reads the game document alone: players, creator and observer, with
// no canonical state, submissions or reports.
func (s *FirestoreStore) Roster(ctx context.Context, actor store.Actor, id store.GameID) (store.GameSnapshot, error) {
	if err := s.requireClient(); err != nil {
		return store.GameSnapshot{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	snapshot, err := gameRef(s.client, id).Get(operationContext)
	s.recordReads(1)
	if err != nil {
		return store.GameSnapshot{}, mapReadError(err, store.ErrUnknownGame)
	}
	game, err := decodeGameDocument(snapshot)
	if err != nil {
		return store.GameSnapshot{}, err
	}
	if !membershipForActor(game, actor) {
		return store.GameSnapshot{}, store.ErrNotMember
	}
	return store.GameSnapshot{
		ID:           game.ID,
		Players:      playerSlots(game.Players),
		CreatedBy:    game.OwnerUID,
		SpectatorUID: game.SpectatorUID,
	}, nil
}
