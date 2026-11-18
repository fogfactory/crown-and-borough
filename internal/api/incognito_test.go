package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

// lobbyEnv is an online API with chat, as seen by the members of one game.
type lobbyEnv struct {
	t       *testing.T
	handler http.Handler
	base    string
	game    gameDetailView
}

func newLobbyEnv(t *testing.T, body string) lobbyEnv {
	t.Helper()
	games, rules := newGamesTestStore(t)
	service := store.NewChatService(games, store.NewMemoryChatBackend())
	handler := NewGamesHandlerWithOptions(games, rules, GamesHandlerOptions{
		Actor:            DevActorResolver("alice"),
		StrictMembership: true,
		Chat:             service,
	})
	game := createGameHTTP(t, handler, "alice", body)
	env := lobbyEnv{t: t, handler: handler, base: "/api/games/" + string(game.ID), game: game}
	env.expect(http.MethodPost, "/confirm", "alice", "", http.StatusOK, "")
	return env
}

func (e lobbyEnv) call(method, path, actor, body string) (int, map[string]any) {
	e.t.Helper()
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	recorder := requestGames(e.t, e.handler, method, e.base+path+separator+"player="+actor, body)
	var decoded map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func (e lobbyEnv) expect(method, path, actor, body string, status int, code string) {
	e.t.Helper()
	got, response := e.call(method, path, actor, body)
	if got != status || (code != "" && response["error"] != code) {
		e.t.Fatalf("%s %s as %s = %d %v, want %d %s", method, path, actor, got, response, status, code)
	}
}

func (e lobbyEnv) join(actor string) {
	e.t.Helper()
	invite := requestGames(e.t, e.handler, http.MethodGet, e.base+"/invite?player=alice", "")
	var invitation invitationView
	if err := json.Unmarshal(invite.Body.Bytes(), &invitation); err != nil || invitation.InviteCode == "" {
		e.t.Fatalf("invite = %d %s", invite.Code, invite.Body.String())
	}
	if status, response := e.call(http.MethodPost, "/join", actor, `{"inviteCode":"`+invitation.InviteCode+`"}`); status != http.StatusCreated {
		e.t.Fatalf("join = %d %v", status, response)
	}
}

func TestLobbyHidesTheBoardAndAcceptsVotesAndNames(t *testing.T) {
	env := newLobbyEnv(t, `{"name":"Hidden","players":3,"incognito":true}`)
	if !env.game.Incognito || env.game.Started || env.game.CurrentPlayer != "" || env.game.Lobby == nil || len(env.game.Lobby.SeedCandidates) != 3 {
		t.Fatalf("created lobby = %#v", env.game)
	}
	// Nothing reveals the board, and nothing can be played, in the lobby.
	env.expect(http.MethodGet, "/state", "alice", "", http.StatusConflict, "game_not_started")
	env.expect(http.MethodGet, "/map", "alice", "", http.StatusConflict, "game_not_started")
	env.expect(http.MethodPost, "/orders", "alice", `{"chains":[]}`, http.StatusConflict, "game_not_started")
	env.expect(http.MethodPost, "/resolve", "alice", "", http.StatusConflict, "game_not_started")
	env.expect(http.MethodPost, "/messages", "alice", `{"channel":"public","body":"hi"}`, http.StatusConflict, "game_not_started")
	if status, response := env.call(http.MethodGet, "", "alice", ""); status != http.StatusOK || response["started"] != false {
		t.Fatalf("lobby detail = %d %v", status, response)
	}
	if status, response := env.call(http.MethodGet, "", "alice", ""); status != http.StatusOK || response["turn"] != float64(0) {
		t.Fatalf("lobby calendar = %d %v", status, response)
	}

	// Votes: a permutation only, own ranking only.
	env.expect(http.MethodPut, "/seed-vote", "alice", `{"ranking":[0,0,1]}`, http.StatusBadRequest, "invalid_seed_ranking")
	status, voted := env.call(http.MethodPut, "/seed-vote", "alice", `{"ranking":[2,0,1]}`)
	lobby, _ := voted["lobby"].(map[string]any)
	if status != http.StatusOK || len(lobby["seedRanking"].([]any)) != 3 {
		t.Fatalf("vote = %d %v", status, voted)
	}
	env.join("bob")
	if _, detail := env.call(http.MethodGet, "", "bob", ""); len(detail["lobby"].(map[string]any)["seedRanking"].([]any)) != 0 {
		t.Fatalf("bob sees another vote: %v", detail["lobby"])
	}
	env.expect(http.MethodPost, "/house-name", "bob", `{"name":"House Tyrell"}`, http.StatusOK, "")
	env.expect(http.MethodPost, "/house-name", "alice", `{"name":"house tyrell"}`, http.StatusConflict, "house_name_taken")

	// The candidate maps reveal the map only, to members only.
	status, candidate := env.call(http.MethodGet, "/seed-candidates/1/map", "alice", "")
	if status != http.StatusOK || len(candidate["territories"].([]any)) == 0 {
		t.Fatalf("candidate map = %d", status)
	}
	// The castles are shown as a set; the houses they go to are not.
	starts, _ := candidate["starts"].([]any)
	if len(starts) == 0 || !sort.SliceIsSorted(starts, func(i, j int) bool { return starts[i].(string) < starts[j].(string) }) {
		t.Fatalf("candidate map starts = %v, want the sorted start castles", candidate["starts"])
	}
	// The browser may keep the map, but the authorization is checked every time.
	first := requestGames(t, env.handler, http.MethodGet, env.base+"/seed-candidates/1/map?player=alice", "")
	etag := first.Header().Get("ETag")
	if etag == "" || !strings.Contains(first.Header().Get("Cache-Control"), "private") {
		t.Fatalf("candidate map headers = %v", first.Header())
	}
	revalidated := requestGamesWithHeaders(t, env.handler, http.MethodGet, env.base+"/seed-candidates/1/map?player=alice", "", map[string]string{"If-None-Match": etag})
	if revalidated.Code != http.StatusNotModified || revalidated.Body.Len() != 0 {
		t.Fatalf("revalidation = %d %q", revalidated.Code, revalidated.Body.String())
	}
	stranger := requestGamesWithHeaders(t, env.handler, http.MethodGet, env.base+"/seed-candidates/1/map?player=mallory", "", map[string]string{"If-None-Match": etag})
	if stranger.Code != http.StatusForbidden {
		t.Fatalf("a stranger holding the ETag got %d", stranger.Code)
	}
	env.expect(http.MethodGet, "/seed-candidates/1/map", "mallory", "", http.StatusForbidden, "invalid_invitation")
	env.expect(http.MethodGet, "/seed-candidates/9/map", "alice", "", http.StatusNotFound, "not_found")

	// Lobby edits are silent: the public chat stays empty.
	env.join("carol")
	recorder := requestGames(t, env.handler, http.MethodGet, env.base+"/messages?channel=public&player=bob", "")
	if strings.Contains(recorder.Body.String(), `"kind"`) {
		t.Fatalf("lobby edits reached the chat: %s", recorder.Body.String())
	}
	env.expect(http.MethodPut, "/seed-vote", "alice", `{"ranking":[0,1,2]}`, http.StatusConflict, "game_started")
	if status, _ := env.call(http.MethodGet, "/state", "alice", ""); status != http.StatusOK {
		t.Fatalf("state once started = %d", status)
	}

	// Once started, a rename is announced in the public chat until the first
	// order is submitted.
	status, renamed := env.call(http.MethodPost, "/house-name", "alice", `{"name":"  House   Stark "}`)
	player, _ := renamed["player"].(map[string]any)
	if status != http.StatusOK || player["name"] != "House Stark" {
		t.Fatalf("rename = %d %v", status, renamed)
	}
	recorder = requestGames(t, env.handler, http.MethodGet, env.base+"/messages?channel=public&player=bob", "")
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
	env.expect(http.MethodPost, "/orders", "alice", `{"chains":[]}`, http.StatusOK, "")
	env.expect(http.MethodPost, "/house-name", "alice", `{"name":"House Lannister"}`, http.StatusConflict, "house_name_locked")
}

func TestOrdinaryOnlineGameHasALobbyToo(t *testing.T) {
	env := newLobbyEnv(t, `{"players":2}`)
	if env.game.Incognito || env.game.Started || env.game.Lobby == nil {
		t.Fatalf("ordinary lobby = %#v", env.game)
	}
	env.expect(http.MethodGet, "/map", "alice", "", http.StatusConflict, "game_not_started")
	env.join("bob")
	status, detail := env.call(http.MethodGet, "", "bob", "")
	if status != http.StatusOK || detail["started"] != true || detail["lobby"] != nil {
		t.Fatalf("started detail = %d %v", status, detail)
	}
	env.expect(http.MethodPost, "/house-name", "bob", `{"name":"House Stark"}`, http.StatusOK, "")
}

type turnNotifier struct{ started []store.GameID }

func (n *turnNotifier) TurnStarted(_ context.Context, snapshot store.GameSnapshot) {
	n.started = append(n.started, snapshot.ID)
}

func (n *turnNotifier) MessagePosted(context.Context, store.ChatPostResult) {}

func TestJoinCarriesTheLobbyEntryAndNotifiesWhenItStartsTheGame(t *testing.T) {
	games, rules := newGamesTestStore(t)
	notifier := &turnNotifier{}
	handler := NewGamesHandlerWithOptions(games, rules, GamesHandlerOptions{
		Actor:            DevActorResolver("alice"),
		StrictMembership: true,
		Notifier:         notifier,
	})
	game := createGameHTTP(t, handler, "alice", `{"players":2}`)
	env := lobbyEnv{t: t, handler: handler, base: "/api/games/" + string(game.ID), game: game}
	env.expect(http.MethodPost, "/confirm", "alice", "", http.StatusOK, "")
	invite := requestGames(t, handler, http.MethodGet, env.base+"/invite?player=alice", "")
	var invitation invitationView
	if err := json.Unmarshal(invite.Body.Bytes(), &invitation); err != nil {
		t.Fatal(err)
	}

	// Someone holding the code, not yet a member, previews the lobby and a map.
	status, preview := env.call(http.MethodGet, "/join-preview?inviteCode="+invitation.InviteCode, "bob", "")
	if status != http.StatusOK || preview["member"] != false || len(preview["seedCandidates"].([]any)) != 3 {
		t.Fatalf("preview = %d %v", status, preview)
	}
	if status, _ := env.call(http.MethodGet, "/seed-candidates/0/map?inviteCode="+invitation.InviteCode, "bob", ""); status != http.StatusOK {
		t.Fatalf("map by invitation = %d", status)
	}
	env.expect(http.MethodGet, "/join-preview?inviteCode=WRONG1", "bob", "", http.StatusForbidden, "invalid_invitation")
	env.expect(http.MethodGet, "/seed-candidates/0/map", "bob", "", http.StatusForbidden, "invalid_invitation")
	if len(notifier.started) != 0 {
		t.Fatal("notified before the game started")
	}

	env.expect(http.MethodPost, "/join", "bob", `{"inviteCode":"`+invitation.InviteCode+`","seedRanking":[0,0,1]}`, http.StatusBadRequest, "invalid_seed_ranking")
	status, joined := env.call(http.MethodPost, "/join", "bob", `{"inviteCode":"`+invitation.InviteCode+`","houseName":"House Tyrell","seedRanking":[2,1,0]}`)
	player, _ := joined["player"].(map[string]any)
	if status != http.StatusCreated || joined["started"] != true || player["name"] != "House Tyrell" {
		t.Fatalf("join = %d %v", status, joined)
	}
	if len(notifier.started) != 1 || notifier.started[0] != game.ID {
		t.Fatalf("turn notifications = %v, want one for the game", notifier.started)
	}
}
