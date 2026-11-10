package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// chatGames is a GameStore serving one hand-built game, so a test controls
// which houses are still in play without playing turns.
type chatGames struct {
	GameStore
	snapshot GameSnapshot
}

func (g *chatGames) Get(_ context.Context, actor Actor, _ GameID) (GameSnapshot, error) {
	if _, ok := g.snapshot.ViewerFor(actor); !ok {
		return GameSnapshot{}, ErrNotMember
	}
	return g.snapshot, nil
}

func newChatFixture(t *testing.T) (*ChatService, *chatGames, *time.Time) {
	t.Helper()
	capital := models.InfraID("castle")
	games := &chatGames{snapshot: GameSnapshot{
		ID:        "g1",
		CreatedBy: "host",
		Players: []PlayerSlot{
			{ID: "P1", ActorID: "alice"},
			{ID: "P2", ActorID: "bob"},
			{ID: "P3", ActorID: "cleo"},
			{ID: "P4", ActorID: "slot:P4"},
		},
		State: &models.GameState{
			Turn:   6,
			Season: models.SeasonSummer,
			Players: []models.Player{
				{ID: "P1", CapitalCastleID: &capital},
				{ID: "P2", CapitalCastleID: &capital},
				{ID: "P3"}, // eliminated
				{ID: "P4", CapitalCastleID: &capital},
			},
		},
	}}
	service := NewChatService(games, NewMemoryChatBackend())
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	service.SetClock(func() time.Time { return clock })
	return service, games, &clock
}

func TestChatPostStampsAuthorHouseAndTurnServerSide(t *testing.T) {
	service, _, clock := newChatFixture(t)
	posted, err := service.Post(context.Background(), Actor{ID: "alice"}, "g1", ChatPostRequest{Channel: ChatPublicChannel, Body: "hello"})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	message := posted.Message
	if message.AuthorHouseID != "P1" || message.Channel != ChatPublicChannel || message.ID == "" {
		t.Errorf("message = %#v", message)
	}
	if want := (ChatTurn{Number: 6, Season: models.SeasonSummer, Year: 2}); message.Turn != want {
		t.Errorf("turn = %#v, want %#v", message.Turn, want)
	}
	if !message.CreatedAt.Equal(*clock) {
		t.Errorf("createdAt = %v, want %v", message.CreatedAt, *clock)
	}
}

func TestChatPostValidatesTheMessage(t *testing.T) {
	tooLong := strings.Repeat("é", ChatMaxBodyRunes+1)
	atLimit := strings.Repeat("é", ChatMaxBodyRunes) // 1000 bytes, 500 runes
	cases := []struct {
		name    string
		actor   string
		request ChatPostRequest
		want    error
	}{
		{"empty", "alice", ChatPostRequest{Channel: "public", Body: ""}, ErrChatEmptyBody},
		{"blank", "alice", ChatPostRequest{Channel: "public", Body: " \n\t"}, ErrChatEmptyBody},
		{"too long in runes", "alice", ChatPostRequest{Channel: "public", Body: tooLong}, ErrChatBodyTooLong},
		{"exactly 500 runes", "alice", ChatPostRequest{Channel: "public", Body: atLimit}, nil},
		{"no channel", "alice", ChatPostRequest{Body: "x"}, ErrChatInvalidChannel},
		{"unknown house", "alice", ChatPostRequest{Channel: "P9", Body: "x"}, ErrChatInvalidRecipient},
		{"oneself", "alice", ChatPostRequest{Channel: "P1", Body: "x"}, ErrChatInvalidRecipient},
		{"unclaimed house", "alice", ChatPostRequest{Channel: "P4", Body: "x"}, ErrChatInvalidRecipient},
		{"non member", "mallory", ChatPostRequest{Channel: "public", Body: "x"}, ErrNotMember},
		{"observer host", "host", ChatPostRequest{Channel: "public", Body: "x"}, ErrNotMember},
		{"batch id alone", "alice", ChatPostRequest{Channel: "public", Body: "x", BatchID: "b"}, ErrChatInvalidBatch},
		{"part without batch", "alice", ChatPostRequest{Channel: "public", Body: "x", Part: 1, Parts: 2}, ErrChatInvalidBatch},
		{"part beyond parts", "alice", ChatPostRequest{Channel: "public", Body: "x", BatchID: "b", Part: 3, Parts: 2}, ErrChatInvalidBatch},
		{"too many parts", "alice", ChatPostRequest{Channel: "public", Body: "x", BatchID: "b", Part: 1, Parts: ChatMaxParts + 1}, ErrChatInvalidBatch},
		{"coherent batch", "alice", ChatPostRequest{Channel: "public", Body: "x", BatchID: "b", Part: 2, Parts: 3}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service, games, _ := newChatFixture(t)
			if tc.name == "observer host" {
				games.snapshot.SpectatorUID = "host"
				tc.want = ErrChatNoHouse
			}
			_, err := service.Post(context.Background(), Actor{ID: tc.actor}, "g1", tc.request)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestChatRateLimitsEachPlayerToOneMessagePerSecond(t *testing.T) {
	service, _, clock := newChatFixture(t)
	ctx := context.Background()
	post := func(actor string) error {
		_, err := service.Post(ctx, Actor{ID: actor}, "g1", ChatPostRequest{Channel: "public", Body: "x"})
		return err
	}
	if err := post("alice"); err != nil {
		t.Fatal(err)
	}
	if err := post("alice"); !errors.Is(err, ErrChatRateLimited) {
		t.Fatalf("second message = %v, want rate limit", err)
	}
	if err := post("bob"); err != nil {
		t.Fatalf("another player is not limited: %v", err)
	}
	*clock = clock.Add(999 * time.Millisecond)
	if err := post("alice"); !errors.Is(err, ErrChatRateLimited) {
		t.Fatalf("after 999ms = %v, want rate limit", err)
	}
	*clock = clock.Add(time.Millisecond)
	if err := post("alice"); err != nil {
		t.Fatalf("after one second: %v", err)
	}
}

func TestChatPrivateConversationsAreSeenByTheirParticipantsAndTheObserverOnly(t *testing.T) {
	service, games, clock := newChatFixture(t)
	ctx := context.Background()
	// P3 stays in play for this test.
	capital := models.InfraID("castle")
	games.snapshot.State.Players[2].CapitalCastleID = &capital

	if _, err := service.Post(ctx, Actor{ID: "alice"}, "g1", ChatPostRequest{Channel: "P2", Body: "secret"}); err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(time.Second)
	if _, err := service.Post(ctx, Actor{ID: "bob"}, "g1", ChatPostRequest{Channel: "P1", Body: "reply"}); err != nil {
		t.Fatal(err)
	}

	key := ChatConversationKey("P2", "P1")
	if key != "P1_P2" {
		t.Fatalf("conversation key = %q, want P1_P2", key)
	}
	history := func(actor, channel string) ([]ChatMessage, error) {
		return service.History(ctx, Actor{ID: actor}, "g1", ChatHistoryRequest{Channel: channel})
	}
	if messages, err := history("alice", "P2"); err != nil || len(messages) != 2 || messages[0].Body != "reply" || messages[1].Body != "secret" {
		t.Fatalf("alice sees %#v, %v", messages, err)
	}
	if messages, err := history("bob", "P1"); err != nil || len(messages) != 2 {
		t.Fatalf("bob sees %#v, %v", messages, err)
	}
	// A third player addressing the same pair sees only their own thread.
	if messages, err := history("cleo", "P1"); err != nil || len(messages) != 0 {
		t.Fatalf("cleo's conversation with P1 = %#v, %v", messages, err)
	}
	// Naming the key of someone else's conversation is refused.
	if _, err := history("cleo", key); !errors.Is(err, ErrChatInvalidRecipient) {
		t.Fatalf("cleo reading %s = %v, want refusal", key, err)
	}
	if _, err := history("mallory", key); !errors.Is(err, ErrNotMember) {
		t.Fatalf("outsider = %v, want not member", err)
	}
	// The game creator who plays a house gets no more than any player.
	games.snapshot.Players = append(games.snapshot.Players, PlayerSlot{ID: "P5", ActorID: "host"})
	games.snapshot.CreatedBy = "host"
	if _, err := history("host", key); !errors.Is(err, ErrChatInvalidRecipient) {
		t.Fatalf("playing creator reading %s = %v, want refusal", key, err)
	}
	// The observer, who plays no house, reads every conversation by key.
	games.snapshot.Players = games.snapshot.Players[:len(games.snapshot.Players)-1]
	games.snapshot.SpectatorUID = "host"
	if messages, err := history("host", key); err != nil || len(messages) != 2 {
		t.Fatalf("observer sees %#v, %v", messages, err)
	}
	if _, err := history("host", "P1_P9"); !errors.Is(err, ErrChatInvalidChannel) {
		t.Fatalf("observer on unknown key = %v", err)
	}
}

func TestChatEliminatedPlayersWriteButNewConversationsNeedAnActiveRecipient(t *testing.T) {
	service, _, clock := newChatFixture(t)
	ctx := context.Background()
	// P3 (cleo) is eliminated: she can write publicly and to an active house
	// only through a conversation that already exists.
	if _, err := service.Post(ctx, Actor{ID: "cleo"}, "g1", ChatPostRequest{Channel: "public", Body: "I am out"}); err != nil {
		t.Fatalf("eliminated player writes publicly: %v", err)
	}
	*clock = clock.Add(time.Second)
	if _, err := service.Post(ctx, Actor{ID: "cleo"}, "g1", ChatPostRequest{Channel: "P1", Body: "psst"}); err != nil {
		t.Fatalf("eliminated player writes to an active house: %v", err)
	}
	*clock = clock.Add(time.Second)
	if _, err := service.Post(ctx, Actor{ID: "alice"}, "g1", ChatPostRequest{Channel: "P3", Body: "still there?"}); err != nil {
		t.Fatalf("existing conversation stays usable: %v", err)
	}
	// Nobody opens a new conversation with the eliminated house.
	*clock = clock.Add(time.Second)
	if _, err := service.Post(ctx, Actor{ID: "bob"}, "g1", ChatPostRequest{Channel: "P3", Body: "hi"}); !errors.Is(err, ErrChatInvalidRecipient) {
		t.Fatalf("new conversation with eliminated house = %v, want refusal", err)
	}
}

func TestChatMarkReadMovesCursorsForwardOnly(t *testing.T) {
	service, _, clock := newChatFixture(t)
	ctx := context.Background()
	// A zero-time update returns the stored cursor untouched.
	stored := func(uid, channel string) time.Time {
		cursor, _ := service.backend.SetChatReadCursor(ctx, "g1", uid, ChatReadCursor{Channel: channel})
		return cursor.LastReadAt
	}
	first, err := service.MarkRead(ctx, Actor{ID: "alice"}, "g1", "P2")
	if err != nil {
		t.Fatal(err)
	}
	if first.Channel != "P1_P2" {
		t.Fatalf("cursor channel = %q", first.Channel)
	}
	*clock = clock.Add(-time.Hour)
	second, err := service.MarkRead(ctx, Actor{ID: "alice"}, "g1", "P2")
	if err != nil || !second.LastReadAt.Equal(first.LastReadAt) {
		t.Fatalf("cursor moved back: %v, %v", second, err)
	}
	if _, err := service.MarkRead(ctx, Actor{ID: "alice"}, "g1", ChatPublicChannel); err != nil {
		t.Fatal(err)
	}
	if stored("alice", ChatPublicChannel).IsZero() {
		t.Error("public cursor not stored")
	}
	if !stored("bob", "P1_P2").IsZero() {
		t.Error("bob's cursor was written by alice's call")
	}
	if _, err := service.MarkRead(ctx, Actor{ID: "alice"}, "g1", "P1"); !errors.Is(err, ErrChatInvalidRecipient) {
		t.Fatalf("own house = %v", err)
	}
}

func TestChatHistoryPaginatesNewestFirstByCursor(t *testing.T) {
	service, _, clock := newChatFixture(t)
	ctx := context.Background()
	for i := 0; i < 7; i++ {
		*clock = clock.Add(2 * time.Second)
		if _, err := service.Post(ctx, Actor{ID: "alice"}, "g1", ChatPostRequest{Channel: "public", Body: string(rune('a' + i))}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := service.History(ctx, Actor{ID: "bob"}, "g1", ChatHistoryRequest{Channel: "public", Limit: 3})
	if err != nil || len(page) != 3 || page[0].Body != "g" || page[2].Body != "e" {
		t.Fatalf("first page = %#v, %v", page, err)
	}
	next, err := service.History(ctx, Actor{ID: "bob"}, "g1", ChatHistoryRequest{Channel: "public", Limit: 3, Before: page[2].Cursor()})
	if err != nil || len(next) != 3 || next[0].Body != "d" || next[2].Body != "b" {
		t.Fatalf("second page = %#v, %v", next, err)
	}
}

func TestNewChatMessageIDSortsByTime(t *testing.T) {
	early, _ := NewChatMessageID(time.Unix(1000, 0))
	late, _ := NewChatMessageID(time.Unix(1001, 0))
	if early >= late {
		t.Errorf("%q should sort before %q", early, late)
	}
}

func TestChatHistoryCursorNeverSkipsMessagesSharingATimestamp(t *testing.T) {
	service, _, _ := newChatFixture(t) // the clock is frozen: every message shares one createdAt
	ctx := context.Background()
	backend := service.backend
	for _, body := range []string{"a", "b", "c", "d", "e"} {
		if _, err := backend.AppendChatMessage(ctx, "g1", "uid-"+body, ChatMessage{Channel: ChatPublicChannel, Body: body, CreatedAt: time.Unix(1000, 0).UTC()}, nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	var seen []string
	var cursor ChatCursor
	for range 5 {
		page, err := service.History(ctx, Actor{ID: "alice"}, "g1", ChatHistoryRequest{Channel: ChatPublicChannel, Limit: 2, Before: cursor})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for _, message := range page {
			seen = append(seen, message.Body)
		}
		cursor = page[len(page)-1].Cursor()
	}
	if len(seen) != 5 {
		t.Fatalf("pages returned %v, want the 5 messages exactly once", seen)
	}
}
