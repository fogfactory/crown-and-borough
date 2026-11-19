package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

// Every way of presenting a code that does not open a game (unknown, malformed,
// empty, belonging to another game) must be indistinguishable to the
// client, and no clear-text code may reach the logs.
func TestInvalidInvitationsAreIndistinguishableAndNeverLogged(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	gameStore, rules := newGamesTestStore(t)
	actors := map[string]store.Actor{
		"alice-token": {ID: "alice-uid"}, "bob-token": {ID: "bob-uid"}, "carol-token": {ID: "carol-uid"},
	}
	resolve := BearerActorResolver(func(token string) (store.Actor, error) {
		if actor, ok := actors[token]; ok {
			return actor, nil
		}
		return store.Actor{}, ErrUnauthorized
	})
	games := NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{Actor: resolve})
	mux := http.NewServeMux()
	mux.Handle("/api/games", games)
	mux.Handle("/api/games/", games)

	create := func(token string) gameDetailView {
		response := requestAuthenticated(t, mux, http.MethodPost, "/api/games", `{"name":"g","players":2}`, token)
		var game gameDetailView
		if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &game) != nil {
			t.Fatalf("create = %d: %s", response.Code, response.Body.String())
		}
		return game
	}
	first, second := create("alice-token"), create("carol-token")

	type attempt struct{ method, path, body string }
	forms := func(code string) []attempt {
		id := "/api/games/" + string(first.ID)
		return []attempt{
			{http.MethodPost, id + "/join", `{"inviteCode":"` + code + `"}`},
			{http.MethodGet, id + "/join-preview?inviteCode=" + code, ""},
			{http.MethodGet, id + "/seed-candidates/0/map?inviteCode=" + code, ""},
		}
	}
	for index, form := range forms("ZZZZZZ") {
		unknown := requestAuthenticated(t, mux, form.method, form.path, form.body, "bob-token")
		if unknown.Code != http.StatusForbidden || !strings.Contains(unknown.Body.String(), `"invalid_invitation"`) {
			t.Fatalf("unknown code, form %d = %d: %s", index, unknown.Code, unknown.Body.String())
		}
		for name, code := range map[string]string{"foreign game": second.InviteCode, "malformed": "x", "empty": ""} {
			other := forms(code)[index]
			response := requestAuthenticated(t, mux, other.method, other.path, other.body, "bob-token")
			if response.Code != unknown.Code || response.Body.String() != unknown.Body.String() {
				t.Fatalf("%s, form %d differs from unknown code: %d %s vs %d %s", name, index, response.Code, response.Body.String(), unknown.Code, unknown.Body.String())
			}
		}
	}
	// A non-member holding the right code is not turned away, and none of the
	// attempts left a code in the logs.
	if response := requestAuthenticated(t, mux, http.MethodGet, "/api/games/"+string(first.ID)+"/join-preview?inviteCode="+first.InviteCode, "", "bob-token"); response.Code != http.StatusOK {
		t.Fatalf("valid code = %d: %s", response.Code, response.Body.String())
	}
	for _, code := range []string{first.InviteCode, second.InviteCode} {
		if strings.Contains(logs.String(), code) {
			t.Fatalf("log contains an invitation code: %s", logs.String())
		}
	}
}
