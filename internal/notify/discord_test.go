package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

const testWebhookURL = "https://discord.com/api/webhooks/123456789/SeCrEt-tOkEn_42"

// fakeDiscord is a webhook server. The client under test is built with a
// transport that sends every request to it, whatever the (valid) URL.
type fakeDiscord struct {
	mu       sync.Mutex
	server   *httptest.Server
	requests []fakeRequest
	// replies are the statuses of the next requests; 204 once empty.
	replies []fakeReply
}

type fakeRequest struct {
	Path    string
	Content string
	Mention DiscordMention
	Raw     map[string]json.RawMessage
}

type fakeReply struct {
	status     int
	retryAfter string
	location   string
}

func newFakeDiscord(t *testing.T) *fakeDiscord {
	t.Helper()
	fake := &fakeDiscord{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var raw map[string]json.RawMessage
		_ = json.Unmarshal(body, &raw)
		request := fakeRequest{Path: r.URL.Path, Raw: raw}
		_ = json.Unmarshal(raw["content"], &request.Content)
		_ = json.Unmarshal(raw["allowed_mentions"], &request.Mention)
		fake.mu.Lock()
		fake.requests = append(fake.requests, request)
		reply := fakeReply{status: http.StatusNoContent}
		if len(fake.replies) > 0 {
			reply, fake.replies = fake.replies[0], fake.replies[1:]
		}
		fake.mu.Unlock()
		if reply.retryAfter != "" {
			w.Header().Set("Retry-After", reply.retryAfter)
		}
		if reply.location != "" {
			w.Header().Set("Location", reply.location)
		}
		w.WriteHeader(reply.status)
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeDiscord) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeDiscord) last(t *testing.T) fakeRequest {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		t.Fatal("no request reached the webhook server")
	}
	return f.requests[len(f.requests)-1]
}

type redirectToFake struct{ target *url.URL }

func (r redirectToFake) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.URL.Scheme = r.target.Scheme
	clone.URL.Host = r.target.Host
	return http.DefaultTransport.RoundTrip(clone)
}

func (f *fakeDiscord) client(t *testing.T) (*DiscordClient, *[]time.Duration) {
	t.Helper()
	target, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := newDiscordClient(redirectToFake{target: target})
	var waits []time.Duration
	client.sleep = func(_ context.Context, wait time.Duration) error {
		waits = append(waits, wait)
		return nil
	}
	return client, &waits
}

func TestValidateDiscordWebhookURL(t *testing.T) {
	valid := []string{
		testWebhookURL,
		"https://discordapp.com/api/webhooks/1/abc",
	}
	for _, raw := range valid {
		if err := ValidateDiscordWebhookURL(raw); err != nil {
			t.Errorf("%q rejected: %v", raw, err)
		}
	}
	invalid := []string{
		"",
		"http://discord.com/api/webhooks/1/abc",
		"https://discord.com.evil.example/api/webhooks/1/abc",
		"https://evil.example/api/webhooks/1/abc",
		"https://evil.example@discord.com/api/webhooks/1/abc",
		"https://discord.com@evil.example/api/webhooks/1/abc",
		"https://discord.com:8443/api/webhooks/1/abc",
		"https://discord.com/api/webhooks/1/abc?wait=true",
		"https://discord.com/api/webhooks/1/abc#frag",
		"https://discord.com/api/webhooks/1/abc/extra",
		"https://discord.com/api/webhooks/1/abc/",
		"https://discord.com/api/webhooks/abc/abc",
		"https://discord.com/api/webhooks//abc",
		"https://discord.com/api/webhooks/1/",
		"https://DISCORD.com/api/webhooks/1/abc",
		"https://discord.com/api/v9/webhooks/1/abc",
		"https://ptb.discord.com/api/webhooks/1/abc",
		"https://127.0.0.1/api/webhooks/1/abc",
		" https://discord.com/api/webhooks/1/abc",
		"https://discord.com/api/webhooks/1/abc\n",
		"https://discord.com/api/webhooks/1/ab%2fc",
		"https://discord.com\\@evil.example/api/webhooks/1/abc",
	}
	for _, raw := range invalid {
		if !errors.Is(ValidateDiscordWebhookURL(raw), ErrInvalidWebhookURL) {
			t.Errorf("%q accepted", raw)
		}
	}
}

func TestDiscordClientPostsContentAndMentions(t *testing.T) {
	fake := newFakeDiscord(t)
	client, _ := fake.client(t)
	err := client.Send(context.Background(), DiscordMessage{URL: testWebhookURL, Content: "hello", Mention: DiscordMention{}})
	if err != nil {
		t.Fatal(err)
	}
	request := fake.last(t)
	if request.Path != "/api/webhooks/123456789/SeCrEt-tOkEn_42" || request.Content != "hello" {
		t.Fatalf("request = %+v", request)
	}
	// An absent allowed_mentions would let Discord parse every mention.
	if string(request.Raw["allowed_mentions"]) != `{"parse":[]}` {
		t.Fatalf("allowed_mentions = %s", request.Raw["allowed_mentions"])
	}
}

func TestDiscordClientRefusesAForeignURLWithoutAnyRequest(t *testing.T) {
	fake := newFakeDiscord(t)
	client, _ := fake.client(t)
	err := client.Send(context.Background(), DiscordMessage{URL: "http://169.254.169.254/api/webhooks/1/a", Content: "x"})
	if !errors.Is(err, ErrInvalidWebhookURL) || fake.count() != 0 {
		t.Fatalf("err = %v, requests = %d", err, fake.count())
	}
}

func TestDiscordClientDoesNotFollowRedirects(t *testing.T) {
	fake := newFakeDiscord(t)
	fake.replies = []fakeReply{{status: http.StatusTemporaryRedirect, location: "http://127.0.0.1:1/internal"}}
	client, _ := fake.client(t)
	err := client.Send(context.Background(), DiscordMessage{URL: testWebhookURL, Content: "x"})
	if !errors.Is(err, ErrDiscordFailed) || fake.count() != 1 {
		t.Fatalf("err = %v, requests = %d", err, fake.count())
	}
}

func TestDiscordClientHonoursBoundedRetryAfter(t *testing.T) {
	fake := newFakeDiscord(t)
	fake.replies = []fakeReply{{status: 429, retryAfter: "1.5"}}
	client, waits := fake.client(t)
	if err := client.Send(context.Background(), DiscordMessage{URL: testWebhookURL, Content: "x"}); err != nil {
		t.Fatal(err)
	}
	if fake.count() != 2 || len(*waits) != 1 || (*waits)[0] != 1500*time.Millisecond {
		t.Fatalf("requests = %d, waits = %v", fake.count(), *waits)
	}
}

func TestDiscordClientGivesUpOnLongOrRepeated429(t *testing.T) {
	fake := newFakeDiscord(t)
	fake.replies = []fakeReply{{status: 429, retryAfter: "60"}}
	client, waits := fake.client(t)
	err := client.Send(context.Background(), DiscordMessage{URL: testWebhookURL, Content: "x"})
	if !errors.Is(err, ErrDiscordRateLimited) || fake.count() != 1 || len(*waits) != 0 {
		t.Fatalf("long wait: err = %v, requests = %d", err, fake.count())
	}

	fake = newFakeDiscord(t)
	fake.replies = []fakeReply{{status: 429, retryAfter: "1"}, {status: 429, retryAfter: "1"}, {status: 429, retryAfter: "1"}, {status: 429, retryAfter: "1"}}
	client, _ = fake.client(t)
	err = client.Send(context.Background(), DiscordMessage{URL: testWebhookURL, Content: "x"})
	if !errors.Is(err, ErrDiscordRateLimited) || fake.count() != maxDiscordAttempts {
		t.Fatalf("repeated 429: err = %v, requests = %d", err, fake.count())
	}
}

func TestDiscordClientReportsAGoneWebhook(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusUnauthorized} {
		fake := newFakeDiscord(t)
		fake.replies = []fakeReply{{status: status}}
		client, _ := fake.client(t)
		err := client.Send(context.Background(), DiscordMessage{URL: testWebhookURL, Content: "x"})
		if !errors.Is(err, ErrWebhookGone) || fake.count() != 1 {
			t.Fatalf("status %d: err = %v, requests = %d", status, err, fake.count())
		}
	}
}

func TestDiscordClientErrorsNeverQuoteTheURL(t *testing.T) {
	// A closed server makes the transport fail with an error that quotes the
	// URL in net/http.
	fake := newFakeDiscord(t)
	client, _ := fake.client(t)
	fake.server.Close()
	err := client.Send(context.Background(), DiscordMessage{URL: testWebhookURL, Content: "x"})
	if err == nil || strings.Contains(err.Error(), "SeCrEt") || strings.Contains(err.Error(), "webhooks") {
		t.Fatalf("err = %v", err)
	}
	message := DiscordMessage{URL: testWebhookURL, Content: "x"}
	if strings.Contains(strings.Join([]string{message.String()}, ""), "SeCrEt") {
		t.Fatal("DiscordMessage prints its URL")
	}
}

func TestValidateDiscordTemplate(t *testing.T) {
	for _, template := range []string{"", "{game} {turn} {season} {url}", "@here new turn", "<@&123> go", "no variable"} {
		if err := ValidateDiscordTemplate(template); err != nil {
			t.Errorf("%q rejected: %v", template, err)
		}
	}
	var unknown *UnknownVariableError
	for _, template := range []string{"{player}", "{message}", "{Game}", "{}", "{ url }", "{order}"} {
		if err := ValidateDiscordTemplate(template); !errors.As(err, &unknown) {
			t.Errorf("%q accepted: %v", template, err)
		}
	}
	// Variables count at their longest: eight links cannot fit.
	if err := ValidateDiscordTemplate(strings.Repeat("{url}", 8)); !errors.Is(err, ErrTemplateTooLong) {
		t.Errorf("long template: %v", err)
	}
	if err := ValidateDiscordTemplate(strings.Repeat("a", 2001)); !errors.Is(err, ErrTemplateTooLong) {
		t.Errorf("2001 characters: %v", err)
	}
	if err := ValidateDiscordTemplate(strings.Repeat("a", 2000)); err != nil {
		t.Errorf("2000 characters: %v", err)
	}
}

func TestDiscordMentionsComeFromTheTemplateOnly(t *testing.T) {
	cases := []struct {
		template string
		parse    []string
		roles    []string
	}{
		{"nothing", []string{}, nil},
		{"@everyone turn {turn}", []string{"everyone"}, nil},
		{"hey @here", []string{"everyone"}, nil},
		{"<@&111> and <@&222> and <@&111>", []string{}, []string{"111", "222"}},
		{"<@123456> user mention", []string{}, nil},
		{"<@&{turn}>", []string{}, nil},
		{"@he{game}", []string{}, nil},
	}
	for _, c := range cases {
		got := DiscordMentionsOf(c.template)
		if strings.Join(got.Parse, ",") != strings.Join(c.parse, ",") || strings.Join(got.Roles, ",") != strings.Join(c.roles, ",") {
			t.Errorf("%q: %+v, want parse %v roles %v", c.template, got, c.parse, c.roles)
		}
		payload, _ := json.Marshal(got)
		if strings.Contains(string(payload), "users") {
			t.Errorf("%q: users in %s", c.template, payload)
		}
	}
}

func TestRenderNeutralizesVariables(t *testing.T) {
	vars := DiscordVars{Game: "@everyone <@&99> {url} @here", Turn: 3, Season: models.SeasonWinter, URL: "https://app.example.test/games/g1"}
	content := RenderDiscordTemplate("{game} | {season} {turn} | {url}", store.NotificationLanguageEnglish, vars)
	if strings.Contains(content, "@everyone") || strings.Contains(content, "@here") || strings.Contains(content, "<@&99>") {
		t.Fatalf("content = %q", content)
	}
	if !strings.HasSuffix(content, "| winter 3 | https://app.example.test/games/g1") {
		t.Fatalf("content = %q", content)
	}
	// A value is never read as a variable.
	if strings.Count(content, "https://app.example.test") != 1 {
		t.Fatalf("variable in a value was expanded: %q", content)
	}
	if got := RenderDiscordTemplate(strings.Repeat("a", 3000), "fr", vars); len([]rune(got)) != DiscordMaxContentRunes {
		t.Fatalf("length = %d", len([]rune(got)))
	}
}

func TestDefaultTemplatesMentionNobody(t *testing.T) {
	for _, kind := range []Kind{KindNewTurn, KindPublicMessage} {
		for _, language := range []string{"fr", "en"} {
			template := DefaultDiscordTemplate(kind, language)
			if err := ValidateDiscordTemplate(template); err != nil {
				t.Errorf("%s/%s: %v", kind, language, err)
			}
			mention := DiscordMentionsOf(template)
			if strings.Contains(template, "@") || len(mention.Parse) != 0 || len(mention.Roles) != 0 {
				t.Errorf("%s/%s mentions: %q", kind, language, template)
			}
		}
	}
}

type discordFixture struct {
	fake     *fakeDiscord
	webhooks *store.MemoryChatBackend
	service  *DiscordService
	notifier *Notifier
	clock    *time.Time
	waits    *[]time.Duration
}

func newDiscordFixture(t *testing.T) discordFixture {
	t.Helper()
	fake := newFakeDiscord(t)
	client, waits := fake.client(t)
	webhooks := store.NewMemoryChatBackend()
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	service := NewDiscordService(webhooks, client, "https://game.example.test/")
	service.SetClock(func() time.Time { return clock })
	f := newFixture(t)
	f.notifier.SetDiscord(service)
	f.notifier.SetClock(func() time.Time { return clock })
	return discordFixture{fake: fake, webhooks: webhooks, service: service, notifier: f.notifier, clock: &clock, waits: waits}
}

func (f discordFixture) configure(t *testing.T, templates store.DiscordTemplates, language string) {
	t.Helper()
	if _, err := f.service.Save(context.Background(), "g1", DiscordSave{URL: testWebhookURL, Language: language, Templates: templates}); err != nil {
		t.Fatal(err)
	}
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	log.SetOutput(&buffer)
	t.Cleanup(func() { log.SetOutput(io.Discard) })
	return &buffer
}

func TestNewTurnIsAnnouncedWithoutPrivateContent(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "en")
	snapshot := turnSnapshot(store.StatusPlaying)
	snapshot.State.Turn = 7
	snapshot.State.Season = models.SeasonSummer
	f.notifier.TurnStarted(context.Background(), snapshot)
	request := f.fake.last(t)
	want := "A new turn has started in « Les Trois Rivières » (summer, turn 7). https://game.example.test/games/g1"
	if request.Content != want || len(request.Mention.Parse) != 0 || len(request.Mention.Roles) != 0 {
		t.Fatalf("request = %+v", request)
	}
}

func TestNewTurnNotAnnouncedOnceFinished(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "fr")
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusFinished))
	if f.fake.count() != 0 {
		t.Fatal("a finished game was announced")
	}
}

func TestCustomTemplateMentionsAreExactlyTheHosts(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{NewTurn: "@here <@&555> tour {turn} de {game}"}, "fr")
	snapshot := turnSnapshot(store.StatusPlaying)
	snapshot.Name = "@everyone"
	f.notifier.TurnStarted(context.Background(), snapshot)
	request := f.fake.last(t)
	if strings.Join(request.Mention.Parse, ",") != "everyone" || strings.Join(request.Mention.Roles, ",") != "555" {
		t.Fatalf("mentions = %+v", request.Mention)
	}
	if strings.Contains(request.Content, "@everyone") {
		t.Fatalf("game name mention not neutralized: %q", request.Content)
	}
}

func TestPublicMessageAnnouncementIsRateLimitedPerGameAndTurnsAreNot(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "fr")
	ctx := context.Background()
	f.notifier.MessagePosted(ctx, publicMessage())
	f.notifier.MessagePosted(ctx, publicMessage())
	if f.fake.count() != 1 {
		t.Fatalf("requests = %d, want 1 within the interval", f.fake.count())
	}
	content := f.fake.last(t).Content
	if strings.Contains(content, "secret plan") || !strings.Contains(content, "salon public") {
		t.Fatalf("content = %q", content)
	}
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying))
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying))
	if f.fake.count() != 3 {
		t.Fatalf("requests = %d, turns must not be limited", f.fake.count())
	}
	*f.clock = f.clock.Add(DiscordPublicInterval)
	f.notifier.MessagePosted(ctx, publicMessage())
	if f.fake.count() != 4 {
		t.Fatalf("requests = %d, want an announcement after the interval", f.fake.count())
	}
}

func TestFailedPublicAnnouncementGivesTheClaimBack(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "fr")
	f.fake.replies = []fakeReply{{status: http.StatusInternalServerError}}
	f.notifier.MessagePosted(context.Background(), publicMessage())
	f.notifier.MessagePosted(context.Background(), publicMessage())
	if f.fake.count() != 2 {
		t.Fatalf("requests = %d, want the second message to try again", f.fake.count())
	}
}

func TestPrivateMessagesNeverReachDiscord(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "fr")
	f.notifier.MessagePosted(context.Background(), privateMessage("bob"))
	if f.fake.count() != 0 {
		t.Fatalf("a private message was announced: %+v", f.fake.last(t))
	}
}

func TestNothingIsSentWithoutAnActiveWebhook(t *testing.T) {
	f := newDiscordFixture(t)
	ctx := context.Background()
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying)) // none configured
	f.configure(t, store.DiscordTemplates{}, "fr")
	if err := f.service.Delete(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying)) // removed
	if f.fake.count() != 0 {
		t.Fatalf("requests = %d", f.fake.count())
	}
}

func TestGoneWebhookIsDisabledLoggedAndNeverRetried(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "fr")
	logs := captureLog(t)
	f.fake.replies = []fakeReply{{status: http.StatusNotFound}}
	ctx := context.Background()
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying))
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying))
	if f.fake.count() != 1 {
		t.Fatalf("requests = %d, a disabled webhook must not be tried again", f.fake.count())
	}
	status, err := f.service.Status(ctx, "g1")
	if err != nil || !status.Configured || !status.Disabled || status.DisabledReason != store.DiscordDisabledGone {
		t.Fatalf("status = %+v, %v", status, err)
	}
	if !strings.Contains(logs.String(), "disabled") {
		t.Fatalf("log = %q", logs.String())
	}
	if strings.Contains(logs.String(), "SeCrEt") || strings.Contains(logs.String(), "123456789") {
		t.Fatalf("the webhook URL was logged: %q", logs.String())
	}
}

func TestDiscordFailureNeverLeaksTheURLInLogs(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "fr")
	logs := captureLog(t)
	f.fake.server.Close()
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	if logs.Len() == 0 || strings.Contains(logs.String(), "SeCrEt") {
		t.Fatalf("log = %q", logs.String())
	}
}

func TestSaveRetiresDisabledStateAndKeepsURLWhenOmitted(t *testing.T) {
	f := newDiscordFixture(t)
	ctx := context.Background()
	f.configure(t, store.DiscordTemplates{}, "fr")
	f.fake.replies = []fakeReply{{status: http.StatusNotFound}}
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying))

	// Changing the settings only keeps the URL and the disabled state.
	status, err := f.service.Save(ctx, "g1", DiscordSave{Language: "en", Templates: store.DiscordTemplates{NewTurn: "turn {turn}"}})
	if err != nil || !status.Disabled || status.Language != "en" || status.Templates.NewTurn != "turn {turn}" {
		t.Fatalf("status = %+v, %v", status, err)
	}
	// A new URL enables it again.
	status, err = f.service.Save(ctx, "g1", DiscordSave{URL: "https://discord.com/api/webhooks/9/NewToken", Language: "en"})
	if err != nil || status.Disabled || !status.Configured {
		t.Fatalf("status = %+v, %v", status, err)
	}
}

func TestSaveValidates(t *testing.T) {
	f := newDiscordFixture(t)
	ctx := context.Background()
	if _, err := f.service.Save(ctx, "g1", DiscordSave{Language: "fr"}); !errors.Is(err, ErrWebhookNotConfigured) {
		t.Errorf("no url, none stored: %v", err)
	}
	if _, err := f.service.Save(ctx, "g1", DiscordSave{URL: "https://evil.example/api/webhooks/1/a"}); !errors.Is(err, ErrInvalidWebhookURL) {
		t.Errorf("foreign url: %v", err)
	}
	if _, err := f.service.Save(ctx, "g1", DiscordSave{URL: testWebhookURL, Language: "de"}); !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("language: %v", err)
	}
	var unknown *UnknownVariableError
	if _, err := f.service.Save(ctx, "g1", DiscordSave{URL: testWebhookURL, Templates: store.DiscordTemplates{PublicMessage: "{body}"}}); !errors.As(err, &unknown) || unknown.Name != "body" {
		t.Errorf("unknown variable: %v", err)
	}
	if config, ok, _ := f.webhooks.DiscordWebhook(ctx, "g1"); ok {
		t.Errorf("an invalid save stored %+v", config)
	}
}

func TestStatusNeverCarriesTheURL(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{NewTurn: "x {turn}"}, "en")
	status, err := f.service.Status(context.Background(), "g1")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(status)
	if strings.Contains(string(payload), "SeCrEt") || strings.Contains(string(payload), "discord.com") {
		t.Fatalf("status = %s", payload)
	}
}

func TestTestMessageMentionsNobodyAndIsRateLimited(t *testing.T) {
	f := newDiscordFixture(t)
	ctx := context.Background()
	f.configure(t, store.DiscordTemplates{NewTurn: "@everyone"}, "en")
	game := DiscordGame{ID: "g1", Name: "Rivers"}
	if err := f.service.Test(ctx, game, "", ""); err != nil {
		t.Fatal(err)
	}
	request := f.fake.last(t)
	if len(request.Mention.Parse) != 0 || len(request.Mention.Roles) != 0 || !strings.Contains(request.Content, "Rivers") {
		t.Fatalf("request = %+v", request)
	}
	if err := f.service.Test(ctx, game, "", ""); !errors.Is(err, ErrTestRateLimited) {
		t.Fatalf("second test: %v", err)
	}
	if err := f.service.Test(ctx, DiscordGame{ID: "other"}, "", ""); !errors.Is(err, ErrWebhookNotConfigured) {
		t.Fatalf("unconfigured game: %v", err)
	}
}

func TestTestReenablesAndDisables(t *testing.T) {
	f := newDiscordFixture(t)
	ctx := context.Background()
	f.configure(t, store.DiscordTemplates{}, "fr")
	f.fake.replies = []fakeReply{{status: http.StatusUnauthorized}}
	if err := f.service.Test(ctx, DiscordGame{ID: "g1"}, "", ""); !errors.Is(err, ErrWebhookGone) {
		t.Fatalf("test: %v", err)
	}
	if status, _ := f.service.Status(ctx, "g1"); !status.Disabled {
		t.Fatal("a refused test did not disable the webhook")
	}
	*f.clock = f.clock.Add(time.Minute)
	if err := f.service.Test(ctx, DiscordGame{ID: "g1"}, "", ""); err != nil {
		t.Fatal(err)
	}
	if status, _ := f.service.Status(ctx, "g1"); status.Disabled {
		t.Fatal("a successful test did not enable the webhook again")
	}
}

func TestPreviewUsesExampleValues(t *testing.T) {
	f := newDiscordFixture(t)
	preview, err := f.service.Preview(EventNewTurn, "en", "@here <@&42> {game} {season} {turn}")
	if err != nil {
		t.Fatal(err)
	}
	if !preview.MentionEveryone || len(preview.MentionRoles) != 1 || preview.MentionRoles[0] != "42" || !strings.Contains(preview.Content, "autumn 12") {
		t.Fatalf("preview = %+v", preview)
	}
	defaults, err := f.service.Preview(EventPublicMessage, "fr", "")
	if err != nil || defaults.MentionEveryone || !strings.Contains(defaults.Content, "salon public") {
		t.Fatalf("default preview = %+v, %v", defaults, err)
	}
	if _, err := f.service.Preview("nope", "fr", ""); !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("event: %v", err)
	}
	if f.fake.count() != 0 {
		t.Fatal("a preview sent something")
	}
}

func TestTestWithFormURLStoresNothing(t *testing.T) {
	f := newDiscordFixture(t)
	ctx := context.Background()
	game := DiscordGame{ID: "g1", Name: "Rivers"}
	// No webhook stored: the URL of the form is tried as is.
	if err := f.service.Test(ctx, game, testWebhookURL, "en"); err != nil {
		t.Fatal(err)
	}
	if status, _ := f.service.Status(ctx, "g1"); status.Configured {
		t.Fatal("a test with the URL of the form stored it")
	}
	// A refusal neither disables nor changes a stored webhook.
	f.configure(t, store.DiscordTemplates{}, "fr")
	*f.clock = f.clock.Add(time.Minute)
	f.fake.replies = []fakeReply{{status: http.StatusNotFound}}
	if err := f.service.Test(ctx, game, testWebhookURL, "en"); !errors.Is(err, ErrWebhookGone) {
		t.Fatalf("test: %v", err)
	}
	if status, _ := f.service.Status(ctx, "g1"); !status.Configured || status.Disabled {
		t.Fatalf("status = %+v", status)
	}
	if err := f.service.Test(ctx, game, "https://evil.example/x", "en"); !errors.Is(err, ErrInvalidWebhookURL) {
		t.Fatalf("invalid URL: %v", err)
	}
}

func TestDiscordNotifierAnnouncesWithoutMailer(t *testing.T) {
	f := newDiscordFixture(t)
	f.configure(t, store.DiscordTemplates{}, "en")
	notifier := NewDiscordNotifier(f.service)
	notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusFinished))
	if f.fake.count() != 0 {
		t.Fatal("a finished game was announced")
	}
	notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	if f.fake.count() != 1 {
		t.Fatalf("announcements = %d, want 1", f.fake.count())
	}
	notifier.MessagePosted(context.Background(), store.ChatPostResult{GameID: "g1", Message: store.ChatMessage{Channel: "private:x"}})
	if f.fake.count() != 1 {
		t.Fatal("a private message was announced")
	}
}

func TestActiveIsCachedAndRefreshedByTheHostsChanges(t *testing.T) {
	f := newDiscordFixture(t)
	ctx := context.Background()
	active := func() bool {
		t.Helper()
		value, err := f.service.Active(ctx, "g1")
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	if active() {
		t.Fatal("a game without webhook is active")
	}
	// Written behind the service: the cached answer holds until it expires.
	if err := f.webhooks.SaveDiscordWebhook(ctx, "g1", store.DiscordWebhookConfig{URL: testWebhookURL}); err != nil {
		t.Fatal(err)
	}
	if active() {
		t.Fatal("the cached answer was not used")
	}
	*f.clock = f.clock.Add(cacheTTL + time.Second)
	if !active() {
		t.Fatal("an expired answer was not read again")
	}
	// Changes made through the service show at once.
	if err := f.service.Delete(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	if active() {
		t.Fatal("a removed webhook is still active")
	}
	f.configure(t, store.DiscordTemplates{}, "fr")
	if !active() {
		t.Fatal("a saved webhook is not active")
	}
}
