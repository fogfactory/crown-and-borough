package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestIncognitoGameLifecycleOverHTTP(t *testing.T) {
	games, rules := newGamesTestStore(t)
	service := store.NewChatService(games, store.NewMemoryChatBackend())
	handler := NewGamesHandlerWithOptions(games, rules, GamesHandlerOptions{
		Actor:            DevActorResolver("alice"),
		StrictMembership: true,
		Chat:             service,
	})
	created := createGameHTTP(t, handler, "alice", `{"name":"Hidden","players":2,"incognito":true}`)
	if !created.Incognito || created.Started || created.CurrentPlayer != "" {
		t.Fatalf("created = incognito %v started %v house %q", created.Incognito, created.Started, created.CurrentPlayer)
	}
	base := "/api/games/" + string(created.ID)
	call := func(method, path, actor, body string) (int, map[string]any) {
		t.Helper()
		recorder := requestGames(t, handler, method, base+path+"?player="+actor, body)
		var decoded map[string]any
		_ = json.Unmarshal(recorder.Body.Bytes(), &decoded)
		return recorder.Code, decoded
	}
	expectConflict := func(method, path, actor, body, code string) {
		t.Helper()
		if status, response := call(method, path, actor, body); status != http.StatusConflict || response["error"] != code {
			t.Fatalf("%s %s = %d %v, want 409 %s", method, path, status, response, code)
		}
	}

	// Nothing reveals the placement, and nothing can be played, before the
	// last slot is taken.
	expectConflict(http.MethodGet, "/state", "alice", "", "game_not_started")
	expectConflict(http.MethodGet, "/map", "alice", "", "game_not_started")
	expectConflict(http.MethodPost, "/orders", "alice", `{"chains":[]}`, "game_not_started")
	expectConflict(http.MethodPost, "/resolve", "alice", "", "game_not_started")
	expectConflict(http.MethodPost, "/house-name", "alice", `{"name":"House Stark"}`, "game_not_started")

	status, joined := call(http.MethodPost, "/join", "bob", `{"inviteCode":"`+inviteCodeOf(t, handler, base)+`"}`)
	if status != http.StatusCreated || joined["started"] != true || joined["incognito"] != true {
		t.Fatalf("join = %d %v", status, joined)
	}
	if status, _ := call(http.MethodGet, "/state", "alice", ""); status != http.StatusOK {
		t.Fatalf("state once started = %d", status)
	}

	status, renamed := call(http.MethodPost, "/house-name", "alice", `{"name":"  House   Stark "}`)
	player, _ := renamed["player"].(map[string]any)
	if status != http.StatusOK || player["name"] != "House Stark" {
		t.Fatalf("rename = %d %v", status, renamed)
	}
	expectConflict(http.MethodPost, "/house-name", "bob", `{"name":"house stark"}`, "house_name_taken")
	if status, response := call(http.MethodPost, "/house-name", "bob", `{"name":""}`); status != http.StatusBadRequest || response["error"] != "invalid_house_name" {
		t.Fatalf("empty name = %d %v", status, response)
	}

	recorder := requestGames(t, handler, http.MethodGet, base+"/messages?channel=public&player=bob", "")
	var history struct {
		Messages []store.ChatMessage `json:"messages"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &history); err != nil || len(history.Messages) != 1 {
		t.Fatalf("history = %s (%v)", recorder.Body.String(), err)
	}
	if message := history.Messages[0]; message.Kind != store.ChatKindHouseRenamed || message.Body != "House Stark" || message.PreviousName == "" {
		t.Fatalf("rename event = %#v", message)
	}
	if strings.Contains(recorder.Body.String(), "alice") {
		t.Fatalf("history leaks the account: %s", recorder.Body.String())
	}

	if status, _ := call(http.MethodPost, "/orders", "alice", `{"chains":[]}`); status != http.StatusOK {
		t.Fatalf("submit = %d", status)
	}
	expectConflict(http.MethodPost, "/house-name", "alice", `{"name":"House Lannister"}`, "house_name_locked")
}

func TestHouseNameRouteRefusesOrdinaryGames(t *testing.T) {
	games, rules := newGamesTestStore(t)
	handler := NewGamesHandlerWithOptions(games, rules, GamesHandlerOptions{Actor: DevActorResolver("alice")})
	created := createGameHTTP(t, handler, "alice", `{"players":2}`)
	recorder := requestGames(t, handler, http.MethodPost, "/api/games/"+string(created.ID)+"/house-name?player=alice", `{"name":"House Stark"}`)
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), `"not_incognito"`) {
		t.Fatalf("rename = %d %s", recorder.Code, recorder.Body.String())
	}
}

// inviteCodeOf asks for a fresh invitation code of the game as its creator.
func inviteCodeOf(t *testing.T, handler http.Handler, base string) string {
	t.Helper()
	recorder := requestGames(t, handler, http.MethodGet, base+"/invite?player=alice", "")
	var invitation invitationView
	if err := json.Unmarshal(recorder.Body.Bytes(), &invitation); err != nil || invitation.InviteCode == "" {
		t.Fatalf("invite = %d %s", recorder.Code, recorder.Body.String())
	}
	return invitation.InviteCode
}
