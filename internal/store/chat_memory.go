package store

import (
	"context"
	"slices"
	"sync"
	"time"
)

// MemoryChatBackend is the in-memory ChatBackend used by tests.
type MemoryChatBackend struct {
	mu       sync.Mutex
	messages map[GameID][]ChatMessage
	lastSent map[chatSenderKey]time.Time
	cursors  map[chatCursorKey]time.Time
}

type chatSenderKey struct {
	game GameID
	uid  string
}

type chatCursorKey struct {
	game    GameID
	uid     string
	channel string
}

var _ ChatBackend = (*MemoryChatBackend)(nil)

func NewMemoryChatBackend() *MemoryChatBackend {
	return &MemoryChatBackend{
		messages: make(map[GameID][]ChatMessage),
		lastSent: make(map[chatSenderKey]time.Time),
		cursors:  make(map[chatCursorKey]time.Time),
	}
}

func (b *MemoryChatBackend) AppendChatMessage(_ context.Context, id GameID, authorUID string, message ChatMessage, _ []string, minInterval time.Duration) (ChatMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sender := chatSenderKey{game: id, uid: authorUID}
	if last, ok := b.lastSent[sender]; ok && message.CreatedAt.Sub(last) < minInterval {
		return ChatMessage{}, ErrChatRateLimited
	}
	messageID, err := NewChatMessageID(message.CreatedAt)
	if err != nil {
		return ChatMessage{}, err
	}
	message.ID = messageID
	b.lastSent[sender] = message.CreatedAt
	b.messages[id] = append(b.messages[id], message)
	return message, nil
}

func (b *MemoryChatBackend) SetChatReadCursor(_ context.Context, id GameID, uid string, cursor ChatReadCursor) (ChatReadCursor, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := chatCursorKey{game: id, uid: uid, channel: cursor.Channel}
	if current, ok := b.cursors[key]; ok && current.After(cursor.LastReadAt) {
		cursor.LastReadAt = current
	}
	b.cursors[key] = cursor.LastReadAt
	return cursor, nil
}

func (b *MemoryChatBackend) ListChatMessages(_ context.Context, id GameID, channel string, before ChatCursor, limit int) ([]ChatMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var inChannel []ChatMessage
	for _, message := range b.messages[id] {
		if message.Channel == channel && (before.IsZero() || message.Cursor().Compare(before) < 0) {
			inChannel = append(inChannel, message)
		}
	}
	// Newest first, like the Firestore query: createdAt then ID, descending.
	slices.SortFunc(inChannel, func(left, right ChatMessage) int {
		return right.Cursor().Compare(left.Cursor())
	})
	return inChannel[:min(limit, len(inChannel))], nil
}
