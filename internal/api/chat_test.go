package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

type chatTestEnv struct {
	handler http.Handler
	games   store.GameStore
	gameID  store.GameID
	clock   *time.Time
}

// newChatTestEnv starts a three-player online game (alice P1, bob P2, cleo
// P3) and a host who only observes.
func newChatTestEnv(t *testing.T, withChat bool) chatTestEnv {
	t.Helper()
	games, rules := newGamesTestStore(t)
	memory := games.(*store.MemoryStore)
	ctx := context.Background()
	created, err := memory.Create(ctx, store.Actor{ID: "alice"}, store.CreateRequest{
		Name:             "Chat game",
		Seed:             "chat",
		Players:          []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}, {Name: "Cleo"}},
		StrictMembership: true,
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	invitation, err := memory.CreateInvitation(ctx, store.Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("invitation: %v", err)
	}
	for _, uid := range []string{"bob", "cleo"} {
		if _, err := memory.Join(ctx, store.Actor{ID: uid}, created.ID, invitation.Code); err != nil {
			t.Fatalf("join %s: %v", uid, err)
		}
	}
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	options := GamesHandlerOptions{Actor: DevActorResolver("alice")}
	if withChat {
		service := store.NewChatService(memory, store.NewMemoryChatBackend())
		service.SetClock(func() time.Time { return clock })
		options.Chat = service
	}
	handler := NewGamesHandlerWithOptions(games, rules, options)
	env := chatTestEnv{handler: handler, games: games, gameID: created.ID, clock: &clock}
	return env
}

func (e chatTestEnv) post(t *testing.T, actor, body string) (int, map[string]any) {
	t.Helper()
	return e.call(t, http.MethodPost, "/api/games/"+string(e.gameID)+"/messages?player="+actor, body)
}

func (e chatTestEnv) call(t *testing.T, method, path, body string) (int, map[string]any) {
	t.Helper()
	recorder := requestGames(t, e.handler, method, path, body)
	var decoded map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func TestChatMessageRouteRejectsInvalidRequests(t *testing.T) {
	env := newChatTestEnv(t, true)
	cases := []struct {
		name   string
		actor  string
		body   string
		status int
		code   string
	}{
		{"not a member", "mallory", `{"channel":"public","body":"hi"}`, http.StatusForbidden, "not_member"},
		{"empty body", "alice", `{"channel":"public","body":""}`, http.StatusBadRequest, "empty_message"},
		{"too long", "alice", `{"channel":"public","body":"` + strings.Repeat("a", 501) + `"}`, http.StatusBadRequest, "message_too_long"},
		{"recipient outside the game", "alice", `{"channel":"P9","body":"hi"}`, http.StatusBadRequest, "invalid_recipient"},
		{"recipient is oneself", "alice", `{"channel":"P1","body":"hi"}`, http.StatusBadRequest, "invalid_recipient"},
		{"missing channel", "alice", `{"body":"hi"}`, http.StatusBadRequest, "invalid_channel"},
		{"incoherent batch", "alice", `{"channel":"public","body":"hi","batchId":"b","part":3,"parts":2}`, http.StatusBadRequest, "invalid_batch"},
		{"unknown field", "alice", `{"channel":"public","body":"hi","author":"P2"}`, http.StatusBadRequest, "invalid_message_request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := env.post(t, tc.actor, tc.body)
			if status != tc.status || body["error"] != tc.code {
				t.Fatalf("status %d body %v, want %d %s", status, body, tc.status, tc.code)
			}
		})
	}
}

func TestChatMessageRouteReturnsTheStoredMessage(t *testing.T) {
	env := newChatTestEnv(t, true)
	status, body := env.post(t, "bob", `{"channel":"P1","body":"hello","batchId":"b1","part":1,"parts":2}`)
	if status != http.StatusCreated {
		t.Fatalf("status %d: %v", status, body)
	}
	if body["authorHouseId"] != "P2" || body["channel"] != "P1_P2" || body["body"] != "hello" {
		t.Fatalf("message = %v", body)
	}
	turn, _ := body["turn"].(map[string]any)
	if turn["number"] != float64(1) || turn["season"] != "spring" || turn["year"] != float64(1) {
		t.Fatalf("turn = %v", turn)
	}
	for _, leak := range []string{"uid", "email", "participantUids", "bob"} {
		for key, value := range body {
			if strings.Contains(strings.ToLower(key), strings.ToLower(leak)) || (key != "body" && value == leak) {
				t.Errorf("message exposes the account: %s=%v", key, value)
			}
		}
	}
}

func TestChatMessageRouteRateLimitsWithTheErrorContract(t *testing.T) {
	env := newChatTestEnv(t, true)
	if status, body := env.post(t, "alice", `{"channel":"public","body":"one"}`); status != http.StatusCreated {
		t.Fatalf("first = %d %v", status, body)
	}
	recorder := requestGames(t, env.handler, http.MethodPost, "/api/games/"+string(env.gameID)+"/messages?player=alice", `{"channel":"public","body":"two"}`)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("second = %d: %s", recorder.Code, recorder.Body.String())
	}
	var body struct{ Error, Message string }
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Error != "rate_limited" || body.Message == "" {
		t.Fatalf("429 body = %s", recorder.Body.String())
	}
	if recorder.Header().Get("Retry-After") != "1" {
		t.Errorf("Retry-After = %q", recorder.Header().Get("Retry-After"))
	}
	if status, _ := env.post(t, "bob", `{"channel":"public","body":"other player"}`); status != http.StatusCreated {
		t.Fatalf("another player is limited too: %d", status)
	}
}

func TestChatMessagesDoNotTouchOrdersOrGameState(t *testing.T) {
	env := newChatTestEnv(t, true)
	actor := store.Actor{ID: "alice"}
	before, err := env.games.Get(context.Background(), actor, env.gameID)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"channel":"public","body":"a"}`, `{"channel":"P2","body":"b"}`} {
		if status, resp := env.post(t, "alice", body); status != http.StatusCreated {
			t.Fatalf("post = %d %v", status, resp)
		}
		*env.clock = env.clock.Add(time.Second)
	}
	if status, resp := env.call(t, http.MethodPost, "/api/games/"+string(env.gameID)+"/messages/read?player=alice", `{"channel":"public"}`); status != http.StatusOK {
		t.Fatalf("read = %d %v", status, resp)
	}
	after, err := env.games.Get(context.Background(), actor, env.gameID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != before.Revision || len(after.Submissions) != 0 {
		t.Fatalf("revision %d -> %d, submissions %v", before.Revision, after.Revision, after.Submissions)
	}
	beforeState, _ := json.Marshal(before.State)
	afterState, _ := json.Marshal(after.State)
	if string(beforeState) != string(afterState) {
		t.Fatal("the game state changed")
	}
}

func TestChatMarkReadRoute(t *testing.T) {
	env := newChatTestEnv(t, true)
	status, body := env.call(t, http.MethodPost, "/api/games/"+string(env.gameID)+"/messages/read?player=cleo", `{"channel":"P2"}`)
	if status != http.StatusOK || body["channel"] != "P2_P3" || body["lastReadAt"] == nil {
		t.Fatalf("read = %d %v", status, body)
	}
	if status, body := env.call(t, http.MethodPost, "/api/games/"+string(env.gameID)+"/messages/read?player=cleo", `{"channel":"P3"}`); status != http.StatusBadRequest || body["error"] != "invalid_recipient" {
		t.Fatalf("own house = %d %v", status, body)
	}
	if status, _ := env.call(t, http.MethodPost, "/api/games/"+string(env.gameID)+"/messages/read?player=mallory", `{"channel":"public"}`); status != http.StatusForbidden {
		t.Fatalf("outsider = %d", status)
	}
}

func TestChatHistoryRouteIsolatesPrivateConversations(t *testing.T) {
	env := newChatTestEnv(t, true)
	env.post(t, "alice", `{"channel":"P2","body":"for bob only"}`)
	*env.clock = env.clock.Add(time.Second)
	env.post(t, "alice", `{"channel":"public","body":"for all"}`)

	list := func(actor, channel string) (int, []any) {
		status, body := env.call(t, http.MethodGet, "/api/games/"+string(env.gameID)+"/messages?player="+actor+"&channel="+channel, "")
		messages, _ := body["messages"].([]any)
		return status, messages
	}
	if status, messages := list("bob", "P1"); status != http.StatusOK || len(messages) != 1 {
		t.Fatalf("bob/P1 = %d %v", status, messages)
	}
	if status, messages := list("cleo", "P1"); status != http.StatusOK || len(messages) != 0 {
		t.Fatalf("cleo/P1 = %d %v", status, messages)
	}
	if status, _ := list("cleo", "P1_P2"); status != http.StatusBadRequest {
		t.Fatalf("cleo reading another conversation by key = %d", status)
	}
	if status, messages := list("cleo", "public"); status != http.StatusOK || len(messages) != 1 {
		t.Fatalf("cleo/public = %d %v", status, messages)
	}
	if status, _ := list("mallory", "public"); status != http.StatusForbidden {
		t.Fatalf("outsider = %d", status)
	}
	// The creator plays P1: no more access than any player.
	if status, _ := list("alice", "P2_P3"); status != http.StatusBadRequest {
		t.Fatalf("playing creator reading by key = %d", status)
	}
}

func TestChatIsAbsentWithoutAChatService(t *testing.T) {
	env := newChatTestEnv(t, false)
	for _, call := range []struct{ method, path, body string }{
		{http.MethodPost, "/messages", `{"channel":"public","body":"hi"}`},
		{http.MethodPost, "/messages/read", `{"channel":"public"}`},
		{http.MethodGet, "/messages?channel=public", ""},
	} {
		status, body := env.call(t, call.method, "/api/games/"+string(env.gameID)+call.path, call.body)
		if status != http.StatusNotFound || body["error"] != "chat_disabled" {
			t.Errorf("%s %s = %d %v, want 404 chat_disabled", call.method, call.path, status, body)
		}
	}
}
