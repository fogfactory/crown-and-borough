package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestAuthenticatedProfileInvitationAndMembershipFlow(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	actors := map[string]store.Actor{
		"alice-token": {ID: "alice-uid", Email: "alice@example.com"},
		"bob-token":   {ID: "bob-uid", Email: "bob@example.com"},
	}
	resolve := BearerActorResolver(func(token string) (store.Actor, error) {
		actor, ok := actors[token]
		if !ok {
			return store.Actor{}, ErrUnauthorized
		}
		return actor, nil
	})
	profiles := gameStore.(store.ProfileStore)
	games := NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{
		Actor:          resolve,
		Profiles:       profiles,
		RequireProfile: true,
		InviteBaseURL:  "https://play.example.test",
	})
	auth := NewAuthHandler(profiles, resolve)
	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth)
	mux.Handle("/api/games", games)
	mux.Handle("/api/games/", games)

	if response := requestAuthenticated(t, mux, http.MethodGet, "/api/auth/me", "", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("profile without bearer = %d: %s", response.Code, response.Body.String())
	}
	profile := requestAuthenticated(t, mux, http.MethodGet, "/api/auth/me", "", "alice-token")
	if profile.Code != http.StatusOK || !strings.Contains(profile.Body.String(), `"uid":"alice-uid"`) {
		t.Fatalf("initial profile = %d: %s", profile.Code, profile.Body.String())
	}
	withoutName := requestAuthenticated(t, mux, http.MethodPost, "/api/games", `{"name":"friends","players":2}`, "alice-token")
	if withoutName.Code != http.StatusBadRequest || !strings.Contains(withoutName.Body.String(), `"profile_required"`) {
		t.Fatalf("create without profile name = %d: %s", withoutName.Code, withoutName.Body.String())
	}
	updated := requestAuthenticated(t, mux, http.MethodPut, "/api/auth/me", `{"displayName":"Alice"}`, "alice-token")
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"displayName":"Alice"`) {
		t.Fatalf("profile update = %d: %s", updated.Code, updated.Body.String())
	}
	if response := requestAuthenticated(t, mux, http.MethodPut, "/api/auth/me", `{"displayName":""}`, "alice-token"); response.Code != http.StatusBadRequest {
		t.Fatalf("empty profile name = %d: %s", response.Code, response.Body.String())
	}

	created := requestAuthenticated(t, mux, http.MethodPost, "/api/games?player=spoofed", `{"name":"friends","players":2}`, "alice-token")
	if created.Code != http.StatusCreated {
		t.Fatalf("create authenticated game = %d: %s", created.Code, created.Body.String())
	}
	var game gameDetailView
	if err := json.Unmarshal(created.Body.Bytes(), &game); err != nil {
		t.Fatalf("decode created game: %v", err)
	}
	if game.ID == "" || len(game.InviteCode) != store.InvitationCodeLength || !strings.Contains(game.InviteURL, game.InviteCode) {
		t.Fatalf("created invitation = %#v", game)
	}
	if game.CurrentPlayer != "P1" || !game.CanInvite || !game.InviteAvailable {
		t.Fatalf("creator projection = %#v", game)
	}
	if strings.Contains(created.Body.String(), store.InvitationCodeHash(game.InviteCode)) {
		t.Fatal("create response exposed the invitation hash")
	}

	if response := requestAuthenticated(t, mux, http.MethodGet, "/api/games/"+string(game.ID)+"/state?player=alice-uid", "", "bob-token"); response.Code != http.StatusForbidden {
		t.Fatalf("non-member state access = %d: %s", response.Code, response.Body.String())
	}
	if response := requestAuthenticated(t, mux, http.MethodPut, "/api/auth/me", `{"displayName":"Bob"}`, "bob-token"); response.Code != http.StatusOK {
		t.Fatalf("Bob profile = %d: %s", response.Code, response.Body.String())
	}
	if response := requestAuthenticated(t, mux, http.MethodPost, "/api/games/"+string(game.ID)+"/join", `{"inviteCode":"wrong1"}`, "bob-token"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid invitation = %d: %s", response.Code, response.Body.String())
	}
	joined := requestAuthenticated(t, mux, http.MethodPost, "/api/games/"+string(game.ID)+"/join", `{"inviteCode":"`+game.InviteCode+`"}`, "bob-token")
	if joined.Code != http.StatusCreated || !strings.Contains(joined.Body.String(), `"joined":true`) {
		t.Fatalf("join = %d: %s", joined.Code, joined.Body.String())
	}
	var joinedGame gameJoinResponse
	if err := json.Unmarshal(joined.Body.Bytes(), &joinedGame); err != nil {
		t.Fatalf("decode joined game: %v", err)
	}
	if joinedGame.CurrentPlayer != "P2" || joinedGame.CanInvite || joinedGame.InviteAvailable {
		t.Fatalf("joiner projection = %#v", joinedGame)
	}
	if response := requestAuthenticated(t, mux, http.MethodPost, "/api/games/"+string(game.ID)+"/resolve", `{}`, "bob-token"); response.Code != http.StatusForbidden {
		t.Fatalf("non-creator resolve access = %d: %s", response.Code, response.Body.String())
	}
	idempotent := requestAuthenticated(t, mux, http.MethodPost, "/api/games/"+string(game.ID)+"/join", `{"inviteCode":"`+game.InviteCode+`"}`, "bob-token")
	if idempotent.Code != http.StatusOK || !strings.Contains(idempotent.Body.String(), `"joined":false`) {
		t.Fatalf("idempotent join = %d: %s", idempotent.Code, idempotent.Body.String())
	}
	if response := requestAuthenticated(t, mux, http.MethodGet, "/api/games/"+string(game.ID)+"/state", "", "bob-token"); response.Code != http.StatusOK {
		t.Fatalf("member state = %d: %s", response.Code, response.Body.String())
	}
	if response := requestAuthenticated(t, mux, http.MethodPost, "/api/games/"+string(game.ID)+"/orders", `{"chains":[{"player":"P2","noble":"ABC","text":""}],"winter":[]}`, "bob-token"); response.Code != http.StatusBadRequest {
		t.Fatalf("client supplied order player = %d: %s", response.Code, response.Body.String())
	}
	if response := requestAuthenticated(t, mux, http.MethodGet, "/api/games/"+string(game.ID)+"/invite", "", "bob-token"); response.Code != http.StatusForbidden {
		t.Fatalf("non-creator invite access = %d: %s", response.Code, response.Body.String())
	}
	if response := requestAuthenticated(t, mux, http.MethodGet, "/api/games/"+string(game.ID)+"/invite", "", "alice-token"); response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"game_full"`) {
		t.Fatalf("full creator invite access = %d: %s", response.Code, response.Body.String())
	}
}

func requestAuthenticated(t *testing.T, handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	return recorder
}

func newReleaseNotesTestMux(t *testing.T) http.Handler {
	t.Helper()
	profiles := store.NewMemoryProfileStore()
	resolve := BearerActorResolver(func(token string) (store.Actor, error) {
		if token != "alice-token" {
			return store.Actor{}, ErrUnauthorized
		}
		return store.Actor{ID: "alice-uid", Email: "alice@example.com"}, nil
	})
	mux := http.NewServeMux()
	mux.Handle("/api/auth/", NewAuthHandler(profiles, resolve))
	return mux
}

func TestReleaseNotesPreferencesUpdate(t *testing.T) {
	mux := newReleaseNotesTestMux(t)
	const path = "/api/auth/me/release-notes"

	if response := requestAuthenticated(t, mux, http.MethodPut, path, `{"lastSeenVersion":"1.2.0"}`, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("without bearer = %d", response.Code)
	}
	if response := requestAuthenticated(t, mux, http.MethodGet, path, "", "alice-token"); response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET = %d", response.Code)
	}
	for _, body := range []string{`{}`, `{"lastSeenVersion":"dev"}`, `{"lastSeenVersion":""}`, `{"unknown":1}`, `not json`} {
		if response := requestAuthenticated(t, mux, http.MethodPut, path, body, "alice-token"); response.Code != http.StatusBadRequest {
			t.Fatalf("body %q = %d: %s", body, response.Code, response.Body.String())
		}
	}

	var decoded profileResponse
	response := requestAuthenticated(t, mux, http.MethodPut, path, `{"lastSeenVersion":"1.2.0"}`, "alice-token")
	if response.Code != http.StatusOK {
		t.Fatalf("update version = %d: %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Player.LastSeenVersion != "1.2.0" || decoded.Player.HideReleaseNotes {
		t.Fatalf("after version update: %+v", decoded.Player)
	}

	response = requestAuthenticated(t, mux, http.MethodPut, path, `{"hideReleaseNotes":true}`, "alice-token")
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil || response.Code != http.StatusOK {
		t.Fatalf("update hide = %d: %s", response.Code, response.Body.String())
	}
	if decoded.Player.LastSeenVersion != "1.2.0" || !decoded.Player.HideReleaseNotes {
		t.Fatalf("hide update must keep the version: %+v", decoded.Player)
	}

	response = requestAuthenticated(t, mux, http.MethodGet, "/api/auth/me", "", "alice-token")
	if !strings.Contains(response.Body.String(), `"lastSeenVersion":"1.2.0"`) || !strings.Contains(response.Body.String(), `"hideReleaseNotes":true`) {
		t.Fatalf("profile does not expose preferences: %s", response.Body.String())
	}
}

// The web client (ProfileData in web/src/types.ts) relies on these exact keys.
func TestProfileContractKeys(t *testing.T) {
	mux := newReleaseNotesTestMux(t)
	response := requestAuthenticated(t, mux, http.MethodGet, "/api/auth/me", "", "alice-token")
	var decoded struct {
		Player map[string]any `json:"player"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"uid": "alice-uid", "email": "alice@example.com", "displayName": "",
		"lastSeenVersion": "", "hideReleaseNotes": false,
		"notifications": map[string]any{
			"newTurn": false, "privateMessage": false, "publicMessage": false,
			"reduceWhenActive": false, "language": "",
		},
	}
	if !reflect.DeepEqual(decoded.Player, want) {
		t.Fatalf("profile contract = %#v, want %#v", decoded.Player, want)
	}
}

func TestNotificationsPreferencesUpdate(t *testing.T) {
	mux := newReleaseNotesTestMux(t)
	const path = "/api/auth/me/notifications"

	if response := requestAuthenticated(t, mux, http.MethodPut, path, `{"newTurn":true}`, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("without bearer = %d", response.Code)
	}
	if response := requestAuthenticated(t, mux, http.MethodGet, path, "", "alice-token"); response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET = %d", response.Code)
	}
	for _, body := range []string{`{}`, `{"language":"de"}`, `{"newTurn":"yes"}`, `{"unknown":1}`, `not json`} {
		if response := requestAuthenticated(t, mux, http.MethodPut, path, body, "alice-token"); response.Code != http.StatusBadRequest {
			t.Fatalf("body %q = %d: %s", body, response.Code, response.Body.String())
		}
	}

	var decoded profileResponse
	response := requestAuthenticated(t, mux, http.MethodPut, path, `{"newTurn":true,"privateMessage":true,"language":"EN"}`, "alice-token")
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil || response.Code != http.StatusOK {
		t.Fatalf("update = %d: %s", response.Code, response.Body.String())
	}
	want := store.NotificationPreferences{NewTurn: true, PrivateMessage: true, Language: "en"}
	if decoded.Player.Notifications != want {
		t.Fatalf("after update: %+v", decoded.Player.Notifications)
	}

	// A partial update keeps the other settings.
	response = requestAuthenticated(t, mux, http.MethodPut, path, `{"publicMessage":true,"newTurn":false}`, "alice-token")
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil || response.Code != http.StatusOK {
		t.Fatalf("partial update = %d: %s", response.Code, response.Body.String())
	}
	want = store.NotificationPreferences{PrivateMessage: true, PublicMessage: true, Language: "en"}
	if decoded.Player.Notifications != want {
		t.Fatalf("after partial update: %+v", decoded.Player.Notifications)
	}

	response = requestAuthenticated(t, mux, http.MethodGet, "/api/auth/me", "", "alice-token")
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil || decoded.Player.Notifications != want {
		t.Fatalf("GET /me does not expose the settings: %s", response.Body.String())
	}
}
