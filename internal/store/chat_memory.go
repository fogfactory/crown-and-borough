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
	acked    map[chatCursorKey]time.Time
	// activity and messageEmails back NotificationStateStore.
	activity      map[string]time.Time
	messageEmails map[chatSenderKey]time.Time
	// webhooks backs DiscordWebhookStore.
	webhooks map[GameID]DiscordWebhookConfig
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

var (
	_ ChatBackend            = (*MemoryChatBackend)(nil)
	_ NotificationStateStore = (*MemoryChatBackend)(nil)
)

func NewMemoryChatBackend() *MemoryChatBackend {
	return &MemoryChatBackend{
		messages: make(map[GameID][]ChatMessage),
		lastSent: make(map[chatSenderKey]time.Time),
		cursors:  make(map[chatCursorKey]time.Time),
		acked:    make(map[chatCursorKey]time.Time),

		activity:      make(map[string]time.Time),
		messageEmails: make(map[chatSenderKey]time.Time),
		webhooks:      make(map[GameID]DiscordWebhookConfig),
	}
}

func (b *MemoryChatBackend) TouchActivity(_ context.Context, uid string, at time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if at.After(b.activity[uid]) {
		b.activity[uid] = at
	}
	return nil
}

func (b *MemoryChatBackend) LastActivity(_ context.Context, uid string) (time.Time, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.activity[uid], nil
}

func (b *MemoryChatBackend) ClaimMessageEmail(_ context.Context, id GameID, uid string, at time.Time) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var reads []time.Time
	for key, readAt := range b.cursors {
		if key.game == id && key.uid == uid {
			reads = append(reads, readAt)
		}
	}
	key := chatSenderKey{game: id, uid: uid}
	if !MessageEmailClaimable(b.messageEmails[key], reads) {
		return false, nil
	}
	b.messageEmails[key] = at
	return true, nil
}

func (b *MemoryChatBackend) ReleaseMessageEmail(_ context.Context, id GameID, uid string, at time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := chatSenderKey{game: id, uid: uid}
	if b.messageEmails[key].Equal(at) {
		delete(b.messageEmails, key)
	}
	return nil
}

func (b *MemoryChatBackend) AppendChatMessage(_ context.Context, id GameID, authorUID string, message ChatMessage, _ []string, minInterval time.Duration) (ChatMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sender := chatSenderKey{game: id, uid: authorUID}
	limited := minInterval > 0
	if last, ok := b.lastSent[sender]; limited && ok && message.CreatedAt.Sub(last) < minInterval {
		return ChatMessage{}, ErrChatRateLimited
	}
	messageID, err := NewChatMessageID(message.CreatedAt)
	if err != nil {
		return ChatMessage{}, err
	}
	message.ID = messageID
	if limited {
		b.lastSent[sender] = message.CreatedAt
	}
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
	cursor.AcknowledgedAt = b.acked[key]
	return cursor, nil
}

func (b *MemoryChatBackend) SetChatAcknowledged(_ context.Context, id GameID, uid string, channel string, at time.Time) (ChatReadCursor, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := chatCursorKey{game: id, uid: uid, channel: channel}
	if current := b.acked[key]; current.After(at) {
		at = current
	}
	b.acked[key] = at
	return ChatReadCursor{Channel: channel, LastReadAt: b.cursors[key], AcknowledgedAt: at}, nil
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

var _ DiscordWebhookStore = (*MemoryChatBackend)(nil)

func (b *MemoryChatBackend) DiscordWebhook(_ context.Context, id GameID) (DiscordWebhookConfig, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	config, ok := b.webhooks[id]
	return config, ok, nil
}

func (b *MemoryChatBackend) SaveDiscordWebhook(_ context.Context, id GameID, config DiscordWebhookConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.webhooks == nil {
		b.webhooks = make(map[GameID]DiscordWebhookConfig)
	}
	config.LastPublicAt = b.webhooks[id].LastPublicAt
	b.webhooks[id] = config
	return nil
}

func (b *MemoryChatBackend) DeleteDiscordWebhook(_ context.Context, id GameID) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.webhooks, id)
	return nil
}

func (b *MemoryChatBackend) DisableDiscordWebhook(_ context.Context, id GameID, url, reason string, at time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	config, ok := b.webhooks[id]
	if !ok || config.URL != url {
		return nil
	}
	config.Disabled = true
	config.DisabledReason = reason
	config.DisabledAt = at
	b.webhooks[id] = config
	return nil
}

func (b *MemoryChatBackend) ClaimDiscordPublic(_ context.Context, id GameID, at time.Time, minInterval time.Duration) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	config, ok := b.webhooks[id]
	if !ok {
		return false, nil
	}
	if !config.LastPublicAt.IsZero() && at.Sub(config.LastPublicAt) < minInterval {
		return false, nil
	}
	config.LastPublicAt = at
	b.webhooks[id] = config
	return true, nil
}

func (b *MemoryChatBackend) ReleaseDiscordPublic(_ context.Context, id GameID, at time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if config, ok := b.webhooks[id]; ok && config.LastPublicAt.Equal(at) {
		config.LastPublicAt = time.Time{}
		b.webhooks[id] = config
	}
	return nil
}
