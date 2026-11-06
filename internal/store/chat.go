package store

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	// ChatPublicChannel is the channel identifier of a game's public room.
	ChatPublicChannel = "public"
	// ChatMaxBodyRunes is the longest message body, counted in runes. Longer
	// texts are split into parts by the client (see ChatMaxParts).
	ChatMaxBodyRunes = 500
	// ChatMaxParts bounds a batch: 8 parts of 500 runes make the 4 000 runes
	// the client may send in one go.
	ChatMaxParts = 8
	// ChatMinInterval is the minimum delay between two messages of a player.
	ChatMinInterval = time.Second
	// chatDefaultPageSize is the page size of the history queries.
	chatDefaultPageSize = 50
	// chatMaxPageSize caps the page size a caller may request.
	chatMaxPageSize = 100
	// chatMaxBatchIDLength bounds the client-chosen batch identifier.
	chatMaxBatchIDLength = 64
)

// ChatTurn is the game turn a message was written in, read from the game state
// when the message is stored.
type ChatTurn struct {
	Number int           `json:"number"`
	Season models.Season `json:"season"`
	Year   int           `json:"year"`
}

// ChatMessage is a chat message as stored and as returned by the API.
//
// Channel is ChatPublicChannel for the public room, otherwise the key of the
// private conversation (see ChatConversationKey). The author is always a
// house: the account behind it (name, email, uid) is never recorded here.
type ChatMessage struct {
	ID            string          `json:"id"`
	Channel       string          `json:"channel"`
	AuthorHouseID models.PlayerID `json:"authorHouseId"`
	Body          string          `json:"body"`
	CreatedAt     time.Time       `json:"createdAt"`
	Turn          ChatTurn        `json:"turn"`
	BatchID       string          `json:"batchId,omitempty"`
	Part          int             `json:"part,omitempty"`
	Parts         int             `json:"parts,omitempty"`
}

// ChatReadCursor records when a player last read a channel. Channel uses the
// same identifier as ChatMessage.Channel.
type ChatReadCursor struct {
	Channel    string    `json:"channel"`
	LastReadAt time.Time `json:"lastReadAt"`
}

// ChatConversationKey is the deterministic identifier of the private
// conversation between two houses: their identifiers, sorted and joined.
func ChatConversationKey(a, b models.PlayerID) string {
	if a > b {
		a, b = b, a
	}
	return string(a) + "_" + string(b)
}

// ChatPostRequest is the input of ChatService.Post. Channel is
// ChatPublicChannel or the identifier of the recipient house.
type ChatPostRequest struct {
	Channel string `json:"channel"`
	Body    string `json:"body"`
	BatchID string `json:"batchId"`
	Part    int    `json:"part"`
	Parts   int    `json:"parts"`
}

// ChatCursor is the position of a message in a channel, ordered by creation
// time then ID. Pages continue after the oldest message already held, so two
// messages sharing a timestamp are never skipped.
type ChatCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

// IsZero reports whether the cursor starts from the latest message.
func (c ChatCursor) IsZero() bool { return c.CreatedAt.IsZero() }

// Compare orders cursors by creation time then ID, like the stored order.
func (c ChatCursor) Compare(other ChatCursor) int {
	return cmp.Or(c.CreatedAt.Compare(other.CreatedAt), strings.Compare(c.ID, other.ID))
}

// Cursor is the position of the message.
func (m ChatMessage) Cursor() ChatCursor { return ChatCursor{CreatedAt: m.CreatedAt, ID: m.ID} }

// ChatHistoryRequest selects one page of a channel, newest first. Channel is
// ChatPublicChannel, a house identifier (the private conversation with that
// house) or, for the game observer only, any conversation key. Before is the
// cursor of the oldest message already held; the zero value starts from the
// latest message.
type ChatHistoryRequest struct {
	Channel string
	Before  ChatCursor
	Limit   int
}

// ChatBackend is the persistence of chat messages and read cursors.
// Membership, validation and privacy are ChatService's business; the rate
// limit is enforced here only because it must be checked atomically with the
// write, across server instances.
type ChatBackend interface {
	// AppendChatMessage atomically checks the author's rate limit against
	// minInterval and stores the message, assigning its ID. It returns
	// ErrChatRateLimited when the author wrote less than minInterval ago.
	// participantUIDs is set for private messages only.
	AppendChatMessage(ctx context.Context, id GameID, authorUID string, message ChatMessage, participantUIDs []string, minInterval time.Duration) (ChatMessage, error)
	// SetChatReadCursor moves a cursor forward; it never moves it back.
	SetChatReadCursor(ctx context.Context, id GameID, uid string, cursor ChatReadCursor) (ChatReadCursor, error)
	// ListChatMessages returns up to limit messages of a channel older than
	// the cursor (all when zero), newest first.
	ListChatMessages(ctx context.Context, id GameID, channel string, before ChatCursor, limit int) ([]ChatMessage, error)
}

// RosterStore is the optional light read of a game: players, creator and
// observer only, without the canonical state. Chat calls that do not need
// the state (history, read cursors) use it when the game store offers it.
type RosterStore interface {
	Roster(context.Context, Actor, GameID) (GameSnapshot, error)
}

var (
	ErrChatInvalidChannel   = errors.New("store: invalid chat channel")
	ErrChatInvalidRecipient = errors.New("store: chat recipient is not another house of this game")
	ErrChatEmptyBody        = errors.New("store: chat message body is empty")
	ErrChatBodyTooLong      = errors.New("store: chat message body is too long")
	ErrChatInvalidBatch     = errors.New("store: invalid chat batch")
	ErrChatRateLimited      = errors.New("store: chat rate limit exceeded")
	ErrChatNoHouse          = errors.New("store: only a house can take part in the chat")
)

// ChatService applies the chat rules over a game store and a ChatBackend.
// It never writes to the game: sending a message submits no order.
type ChatService struct {
	games   GameStore
	backend ChatBackend
	now     func() time.Time
}

func NewChatService(games GameStore, backend ChatBackend) *ChatService {
	return &ChatService{games: games, backend: backend, now: time.Now}
}

// SetClock replaces the clock; for tests.
func (c *ChatService) SetClock(now func() time.Time) { c.now = now }

// Post validates and stores a message written by actor's house.
func (c *ChatService) Post(ctx context.Context, actor Actor, id GameID, request ChatPostRequest) (ChatMessage, error) {
	snapshot, house, err := c.houseOf(ctx, actor, id, true)
	if err != nil {
		return ChatMessage{}, err
	}
	body := request.Body
	if strings.TrimSpace(body) == "" {
		return ChatMessage{}, ErrChatEmptyBody
	}
	if utf8.RuneCountInString(body) > ChatMaxBodyRunes {
		return ChatMessage{}, ErrChatBodyTooLong
	}
	if err := validateChatBatch(request); err != nil {
		return ChatMessage{}, err
	}
	channel, participants, err := c.resolveRecipient(ctx, snapshot, id, house, request.Channel)
	if err != nil {
		return ChatMessage{}, err
	}
	// Firestore keeps microseconds: stamp what will be stored.
	at := c.now().UTC().Truncate(time.Microsecond)
	message := ChatMessage{
		Channel:       channel,
		AuthorHouseID: house,
		Body:          body,
		CreatedAt:     at,
		Turn: ChatTurn{
			Number: snapshot.State.Turn,
			Season: snapshot.State.Season,
			Year:   snapshot.State.Year(),
		},
		BatchID: request.BatchID,
		Part:    request.Part,
		Parts:   request.Parts,
	}
	return c.backend.AppendChatMessage(ctx, id, actor.ID, message, participants, ChatMinInterval)
}

// MarkRead moves the actor's read cursor of a channel to now.
func (c *ChatService) MarkRead(ctx context.Context, actor Actor, id GameID, channel string) (ChatReadCursor, error) {
	snapshot, house, err := c.houseOf(ctx, actor, id, false)
	if err != nil {
		return ChatReadCursor{}, err
	}
	// Reading never creates a conversation: an eliminated recipient is fine.
	key, _, err := c.channelKey(snapshot, house, channel)
	if err != nil {
		return ChatReadCursor{}, err
	}
	return c.backend.SetChatReadCursor(ctx, id, actor.ID, ChatReadCursor{Channel: key, LastReadAt: c.now().UTC().Truncate(time.Microsecond)})
}

// History returns one page of a channel the actor may read: the public room,
// the actor's own conversations and, for the observer who plays no house,
// every conversation.
func (c *ChatService) History(ctx context.Context, actor Actor, id GameID, request ChatHistoryRequest) ([]ChatMessage, error) {
	snapshot, err := c.load(ctx, actor, id, false)
	if err != nil {
		return nil, err
	}
	house, isPlayer := snapshot.PlayerFor(actor)
	isObserver := IsSpectator(snapshot.SpectatorUID, actor)
	if !isPlayer && !isObserver {
		return nil, ErrNotMember
	}
	// A player reads the public room and their own conversations, whoever
	// created the game. Only the observer, who plays no house, reads any pair.
	var key string
	if isPlayer {
		key, _, err = c.channelKey(snapshot, house, request.Channel)
	} else {
		key, err = observerChannelKey(snapshot, request.Channel)
	}
	if err != nil {
		return nil, err
	}
	return c.backend.ListChatMessages(ctx, id, key, request.Before, ChatPageSize(request.Limit))
}

// ChatPageSize applies the default and the maximum to a requested page size.
func ChatPageSize(limit int) int {
	if limit <= 0 {
		return chatDefaultPageSize
	}
	return min(limit, chatMaxPageSize)
}

// load reads the game; the roster is enough unless the state is needed.
func (c *ChatService) load(ctx context.Context, actor Actor, id GameID, needState bool) (GameSnapshot, error) {
	if roster, ok := c.games.(RosterStore); ok && !needState {
		return roster.Roster(ctx, actor, id)
	}
	return c.games.Get(ctx, actor, id)
}

// houseOf loads the game and returns the house actor plays.
func (c *ChatService) houseOf(ctx context.Context, actor Actor, id GameID, needState bool) (GameSnapshot, models.PlayerID, error) {
	snapshot, err := c.load(ctx, actor, id, needState)
	if err != nil {
		return GameSnapshot{}, "", err
	}
	house, ok := snapshot.PlayerFor(actor)
	if !ok {
		if IsSpectator(snapshot.SpectatorUID, actor) {
			return GameSnapshot{}, "", ErrChatNoHouse
		}
		return GameSnapshot{}, "", ErrNotMember
	}
	if needState && snapshot.State == nil {
		return GameSnapshot{}, "", ErrUnknownGame
	}
	return snapshot, house, nil
}

// channelKey maps a requested channel to its stored identifier for house, and
// returns the other house of a private conversation.
func (c *ChatService) channelKey(snapshot GameSnapshot, house models.PlayerID, channel string) (string, models.PlayerID, error) {
	switch channel {
	case "":
		return "", "", ErrChatInvalidChannel
	case ChatPublicChannel:
		return ChatPublicChannel, "", nil
	}
	other := models.PlayerID(channel)
	if other == house || !hasPlayer(snapshot, other) {
		return "", "", ErrChatInvalidRecipient
	}
	return ChatConversationKey(house, other), other, nil
}

func hasPlayer(snapshot GameSnapshot, id models.PlayerID) bool {
	return slices.ContainsFunc(snapshot.Players, func(p PlayerSlot) bool { return p.ID == id })
}

// observerChannelKey accepts the public room or the key of a conversation between
// two houses of the game.
func observerChannelKey(snapshot GameSnapshot, channel string) (string, error) {
	if channel == ChatPublicChannel {
		return channel, nil
	}
	left, right, ok := strings.Cut(channel, "_")
	l, r := models.PlayerID(left), models.PlayerID(right)
	if ok && l != r && hasPlayer(snapshot, l) && hasPlayer(snapshot, r) && ChatConversationKey(l, r) == channel {
		return channel, nil
	}
	return "", ErrChatInvalidChannel
}

// resolveRecipient returns the stored channel and, for a private message, the
// uids of both participants. An eliminated house cannot be addressed in a new
// conversation, but an existing one stays usable.
func (c *ChatService) resolveRecipient(ctx context.Context, snapshot GameSnapshot, id GameID, house models.PlayerID, channel string) (string, []string, error) {
	key, other, err := c.channelKey(snapshot, house, channel)
	if err != nil {
		return "", nil, err
	}
	if key == ChatPublicChannel {
		return key, nil, nil
	}
	recipient := playerSlotForID(snapshot.Players, other)
	if !isAssignedSlot(recipient) {
		return "", nil, ErrChatInvalidRecipient
	}
	if !engine.PlayerAlive(snapshot.State, other) {
		existing, err := c.backend.ListChatMessages(ctx, id, key, ChatCursor{}, 1)
		if err != nil {
			return "", nil, err
		}
		if len(existing) == 0 {
			return "", nil, ErrChatInvalidRecipient
		}
	}
	author := playerSlotForID(snapshot.Players, house)
	return key, []string{author.ActorID, recipient.ActorID}, nil
}

// validateChatBatch accepts either no batch field or a coherent set of all
// three.
func validateChatBatch(request ChatPostRequest) error {
	if request.BatchID == "" && request.Part == 0 && request.Parts == 0 {
		return nil
	}
	if request.BatchID == "" || len(request.BatchID) > chatMaxBatchIDLength ||
		request.Parts < 1 || request.Parts > ChatMaxParts ||
		request.Part < 1 || request.Part > request.Parts {
		return ErrChatInvalidBatch
	}
	return nil
}

// NewChatMessageID returns an identifier that sorts by creation time, so a
// listing ordered by createdAt then by ID is stable when two messages share a
// timestamp.
func NewChatMessageID(at time.Time) (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("store: chat message id: %w", err)
	}
	return fmt.Sprintf("%016x%s", at.UnixMicro(), hex.EncodeToString(suffix[:])), nil
}
