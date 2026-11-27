package api

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/notify"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

const apiWebhookURL = "https://discord.com/api/webhooks/123456789/SeCrEt-tOkEn_42"

type recordingSender struct {
	mu       sync.Mutex
	messages []notify.DiscordMessage
	err      error
}

func (s *recordingSender) Send(_ context.Context, message notify.DiscordMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, message)
	return s.err
}

type discordTestEnv struct {
	chatTestEnv
	sender   *recordingSender
	webhooks *store.MemoryChatBackend
	service  *notify.DiscordService
}

// newDiscordTestEnv is the chat game (alice hosts, bob and cleo play) with the
// Discord routes mounted.
func newDiscordTestEnv(t *testing.T, mounted bool) discordTestEnv {
	t.Helper()
	env := newChatTestEnv(t, false)
	_, rules := newGamesTestStore(t)
	sender := &recordingSender{}
	webhooks := store.NewMemoryChatBackend()
	service := notify.NewDiscordService(webhooks, sender, "https://game.example.test")
	options := GamesHandlerOptions{Actor: DevActorResolver("alice")}
	if mounted {
		options.Discord = service
	}
	env.handler = NewGamesHandlerWithOptions(env.games, rules, options)
	return discordTestEnv{chatTestEnv: env, sender: sender, webhooks: webhooks, service: service}
}

func (e discordTestEnv) path(suffix string) string {
	return "/api/games/" + string(e.gameID) + "/discord-webhook" + suffix
}

func (e discordTestEnv) request(t *testing.T, method, suffix, player, body string) (int, string) {
	t.Helper()
	recorder := requestGames(t, e.handler, method, e.path(suffix)+"?player="+player, body)
	return recorder.Code, recorder.Body.String()
}

func saveBody(url string) string {
	return `{"url":"` + url + `","language":"en","templates":{"newTurn":"@here turn {turn}","publicMessage":""}}`
}

func TestDiscordWebhookIsHostOnly(t *testing.T) {
	env := newDiscordTestEnv(t, true)
	for _, c := range []struct{ method, suffix, body string }{
		{http.MethodGet, "", ""},
		{http.MethodPut, "", saveBody(apiWebhookURL)},
		{http.MethodDelete, "", ""},
		{http.MethodPost, "/test", ""},
		{http.MethodPost, "/preview", `{"event":"newTurn","language":"en","template":""}`},
	} {
		status, body := env.request(t, c.method, c.suffix, "bob", c.body)
		if status != http.StatusForbidden || !strings.Contains(body, "not_creator") {
			t.Errorf("%s %s as a player = %d %s", c.method, c.suffix, status, body)
		}
	}
	if _, ok, _ := env.webhooks.DiscordWebhook(context.Background(), env.gameID); ok {
		t.Fatal("a player saved a webhook")
	}
	if len(env.sender.messages) != 0 {
		t.Fatal("a player triggered a send")
	}
}

func TestDiscordWebhookLifecycleNeverReturnsTheURL(t *testing.T) {
	env := newDiscordTestEnv(t, true)

	status, body := env.request(t, http.MethodGet, "", "alice", "")
	if status != http.StatusOK || !strings.Contains(body, `"configured":false`) {
		t.Fatalf("initial GET = %d %s", status, body)
	}

	status, body = env.request(t, http.MethodPut, "", "alice", saveBody(apiWebhookURL))
	if status != http.StatusOK || !strings.Contains(body, `"configured":true`) || !strings.Contains(body, `"language":"en"`) || !strings.Contains(body, "@here turn {turn}") {
		t.Fatalf("PUT = %d %s", status, body)
	}
	// Settings change without the URL.
	status, body2 := env.request(t, http.MethodPut, "", "alice", `{"language":"fr","templates":{"newTurn":"","publicMessage":"x"}}`)
	if status != http.StatusOK || !strings.Contains(body2, `"language":"fr"`) {
		t.Fatalf("PUT without url = %d %s", status, body2)
	}
	status, body3 := env.request(t, http.MethodGet, "", "alice", "")
	for _, got := range []string{body, body2, body3} {
		if status != http.StatusOK || strings.Contains(got, "SeCrEt") || strings.Contains(got, "discord.com") || strings.Contains(got, "123456789") {
			t.Fatalf("response leaks the URL: %s", got)
		}
	}

	status, body = env.request(t, http.MethodPost, "/test", "alice", "")
	if status != http.StatusOK || len(env.sender.messages) != 1 || env.sender.messages[0].URL != apiWebhookURL {
		t.Fatalf("test = %d %s, sent %d", status, body, len(env.sender.messages))
	}
	if strings.Contains(body, "SeCrEt") {
		t.Fatalf("test response leaks the URL: %s", body)
	}

	status, body = env.request(t, http.MethodDelete, "", "alice", "")
	if status != http.StatusOK || !strings.Contains(body, `"configured":false`) {
		t.Fatalf("DELETE = %d %s", status, body)
	}
	if _, ok, _ := env.webhooks.DiscordWebhook(context.Background(), env.gameID); ok {
		t.Fatal("the webhook is still stored")
	}
	// Removing twice is fine; testing without a webhook is a conflict.
	if status, _ = env.request(t, http.MethodDelete, "", "alice", ""); status != http.StatusOK {
		t.Fatalf("second DELETE = %d", status)
	}
	if status, body = env.request(t, http.MethodPost, "/test", "alice", ""); status != http.StatusConflict || !strings.Contains(body, "webhook_not_configured") {
		t.Fatalf("test without webhook = %d %s", status, body)
	}
}

func TestDiscordWebhookSaveErrors(t *testing.T) {
	env := newDiscordTestEnv(t, true)
	for _, c := range []struct{ body, code string }{
		{saveBody("https://evil.example/api/webhooks/1/a"), "invalid_webhook_url"},
		{saveBody("http://discord.com/api/webhooks/1/a"), "invalid_webhook_url"},
		{`{"url":"` + apiWebhookURL + `","language":"de"}`, "invalid_language"},
		{`{"url":"` + apiWebhookURL + `","templates":{"newTurn":"{author} won"}}`, "invalid_template"},
		{`{"url":"` + apiWebhookURL + `","templates":{"newTurn":"` + strings.Repeat("{url}", 9) + `"}}`, "template_too_long"},
		{`{"language":"fr"}`, "webhook_not_configured"},
		{`{"url":"` + apiWebhookURL + `","extra":1}`, "invalid_discord_request"},
	} {
		status, body := env.request(t, http.MethodPut, "", "alice", c.body)
		if status < 400 || !strings.Contains(body, c.code) {
			t.Errorf("body %s => %d %s, want %s", c.body, status, body, c.code)
		}
		if strings.Contains(body, "SeCrEt") {
			t.Errorf("error leaks the token: %s", body)
		}
	}
	status, body := env.request(t, http.MethodPut, "", "alice", `{"url":"`+apiWebhookURL+`","templates":{"newTurn":"{author}"}}`)
	if status != http.StatusBadRequest || !strings.Contains(body, `"variable":"author"`) {
		t.Errorf("unknown variable = %d %s", status, body)
	}
}

func TestDiscordWebhookPreview(t *testing.T) {
	env := newDiscordTestEnv(t, true)
	status, body := env.request(t, http.MethodPost, "/preview", "alice", `{"event":"newTurn","language":"en","template":"@everyone <@&42> {game} {season} {turn}"}`)
	if status != http.StatusOK || !strings.Contains(body, `"mentionEveryone":true`) || !strings.Contains(body, `"mentionRoles":["42"]`) || !strings.Contains(body, "autumn 12") {
		t.Fatalf("preview = %d %s", status, body)
	}
	if status, body = env.request(t, http.MethodPost, "/preview", "alice", `{"event":"x","language":"en","template":""}`); status != http.StatusBadRequest || !strings.Contains(body, "invalid_event") {
		t.Fatalf("bad event = %d %s", status, body)
	}
	if len(env.sender.messages) != 0 {
		t.Fatal("a preview sent a message")
	}
}

func TestDiscordTestFailuresMapToStatuses(t *testing.T) {
	env := newDiscordTestEnv(t, true)
	if status, _ := env.request(t, http.MethodPut, "", "alice", saveBody(apiWebhookURL)); status != http.StatusOK {
		t.Fatal("save failed")
	}
	env.sender.err = notify.ErrWebhookGone
	if status, body := env.request(t, http.MethodPost, "/test", "alice", ""); status != http.StatusConflict || !strings.Contains(body, "webhook_gone") {
		t.Fatalf("gone = %d %s", status, body)
	}
	// A test that did not leave does not hold back the retry.
	env.sender.err = notify.ErrDiscordFailed
	if status, body := env.request(t, http.MethodPost, "/test", "alice", ""); status != http.StatusBadGateway || !strings.Contains(body, "discord_unreachable") {
		t.Fatalf("failed = %d %s", status, body)
	}
	// Two successful tests in a row are limited; later ones are not.
	env.sender.err = nil
	if status, body := env.request(t, http.MethodPost, "/test", "alice", ""); status != http.StatusOK {
		t.Fatalf("first success = %d %s", status, body)
	}
	if status, body := env.request(t, http.MethodPost, "/test", "alice", ""); status != http.StatusTooManyRequests || !strings.Contains(body, "discord_rate_limited") {
		t.Fatalf("immediate second test = %d %s", status, body)
	}
	later := time.Now().Add(time.Minute)
	env.service.SetClock(func() time.Time { return later })
	if status, body := env.request(t, http.MethodPost, "/test", "alice", ""); status != http.StatusOK {
		t.Fatalf("later = %d %s", status, body)
	}
}

func TestGameViewTellsEveryPlayerAWebhookIsActive(t *testing.T) {
	env := newDiscordTestEnv(t, true)
	view := func(player string) string {
		recorder := requestGames(t, env.handler, http.MethodGet, "/api/games/"+string(env.gameID)+"?player="+player, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("detail = %d", recorder.Code)
		}
		return recorder.Body.String()
	}
	if got := view("bob"); !strings.Contains(got, `"discordWebhook":false`) {
		t.Fatalf("before: %s", got)
	}
	if status, _ := env.request(t, http.MethodPut, "", "alice", saveBody(apiWebhookURL)); status != http.StatusOK {
		t.Fatal("save failed")
	}
	for _, player := range []string{"alice", "bob", "cleo"} {
		got := view(player)
		if !strings.Contains(got, `"discordWebhook":true`) || strings.Contains(got, "SeCrEt") || strings.Contains(got, "discord.com") {
			t.Fatalf("%s sees %s", player, got)
		}
	}
	if status, _ := env.request(t, http.MethodDelete, "", "alice", ""); status != http.StatusOK {
		t.Fatal("delete failed")
	}
	if got := view("bob"); !strings.Contains(got, `"discordWebhook":false`) {
		t.Fatalf("after removal: %s", got)
	}
}

func TestDiscordRoutesAbsentWhenNotMounted(t *testing.T) {
	env := newDiscordTestEnv(t, false)
	for _, c := range []struct{ method, suffix string }{
		{http.MethodGet, ""}, {http.MethodPut, ""}, {http.MethodDelete, ""}, {http.MethodPost, "/test"}, {http.MethodPost, "/preview"},
	} {
		status, body := env.request(t, c.method, c.suffix, "alice", "{}")
		if status != http.StatusNotFound || !strings.Contains(body, "discord_disabled") {
			t.Errorf("%s %s = %d %s", c.method, c.suffix, status, body)
		}
	}
	recorder := requestGames(t, env.handler, http.MethodGet, "/api/games/"+string(env.gameID)+"?player=bob", "")
	if !strings.Contains(recorder.Body.String(), `"discordWebhook":false`) {
		t.Fatalf("view = %s", recorder.Body.String())
	}
}
