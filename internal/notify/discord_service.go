package notify

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

var (
	// ErrWebhookNotConfigured is returned when an operation needs a webhook
	// and the game has none.
	ErrWebhookNotConfigured = errors.New("notify: no discord webhook configured")
	// ErrInvalidLanguage is returned for a language other than fr or en.
	ErrInvalidLanguage = errors.New("notify: invalid language")
	// ErrTestRateLimited is returned when a test message was sent a moment ago.
	ErrTestRateLimited = errors.New("notify: discord test sent too recently")
)

const (
	// DiscordPublicInterval is the minimum delay between two public room
	// announcements of a game. Turn announcements are not limited.
	DiscordPublicInterval = 5 * time.Minute
	// discordTestInterval is the minimum delay between two test messages of a
	// game on one server instance.
	discordTestInterval = 10 * time.Second
	// discordDisableTimeout bounds disabling a webhook once it failed.
	discordDisableTimeout = 2 * time.Second
	// Event names of the API (templates, preview).
	EventNewTurn       = "newTurn"
	EventPublicMessage = "publicMessage"
)

// DiscordGame is what an announcement says about a game.
type DiscordGame struct {
	ID     store.GameID
	Name   string
	Turn   int
	Season models.Season
}

// DiscordStatus is what a host reads back: never the URL.
type DiscordStatus struct {
	Configured     bool                   `json:"configured"`
	Language       string                 `json:"language,omitempty"`
	Templates      store.DiscordTemplates `json:"templates"`
	Disabled       bool                   `json:"disabled"`
	DisabledReason string                 `json:"disabledReason,omitempty"`
	// Defaults are the templates used where the host wrote none, by language.
	Defaults map[string]store.DiscordTemplates `json:"defaults"`
}

// DiscordSave is a save request. An empty URL keeps the stored one.
type DiscordSave struct {
	URL       string
	Language  string
	Templates store.DiscordTemplates
}

// DiscordPreview is an announcement rendered with example values.
type DiscordPreview struct {
	Content         string   `json:"content"`
	Length          int      `json:"length"`
	MentionEveryone bool     `json:"mentionEveryone"`
	MentionRoles    []string `json:"mentionRoles"`
}

// DiscordService owns the Discord webhook of the games: the host's operations
// and the announcements. Failures of Discord are logged, never returned by
// Announce. It does not import any database.
type DiscordService struct {
	webhooks store.DiscordWebhookStore
	sender   Sender[DiscordMessage]
	baseURL  string
	now      func() time.Time

	mu       sync.Mutex
	lastTest map[store.GameID]time.Time
	// active remembers, per game, whether a webhook is active: the game view
	// every player reads asks it, and saving, removing or testing the webhook
	// through this instance refreshes it at once.
	active ttlCache[store.GameID, bool]
}

func NewDiscordService(webhooks store.DiscordWebhookStore, sender Sender[DiscordMessage], baseURL string) *DiscordService {
	return &DiscordService{
		webhooks: webhooks,
		sender:   sender,
		baseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		now:      time.Now,
		lastTest: make(map[store.GameID]time.Time),
	}
}

// SetClock replaces the clock; for tests.
func (s *DiscordService) SetClock(now func() time.Time) { s.now = now }

func normalizeLanguage(language string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "", store.NotificationLanguageFrench:
		return store.NotificationLanguageFrench, nil
	case store.NotificationLanguageEnglish:
		return store.NotificationLanguageEnglish, nil
	}
	return "", ErrInvalidLanguage
}

// discordDefaults are the default templates by language; they never change.
var discordDefaults = func() map[string]store.DiscordTemplates {
	defaults := map[string]store.DiscordTemplates{}
	for _, language := range []string{store.NotificationLanguageFrench, store.NotificationLanguageEnglish} {
		defaults[language] = store.DiscordTemplates{
			NewTurn:       DefaultDiscordTemplate(KindNewTurn, language),
			PublicMessage: DefaultDiscordTemplate(KindPublicMessage, language),
		}
	}
	return defaults
}()

// normalizeTemplates trims the templates of a host and checks each of them.
func normalizeTemplates(templates store.DiscordTemplates) (store.DiscordTemplates, error) {
	templates.NewTurn = strings.TrimSpace(templates.NewTurn)
	templates.PublicMessage = strings.TrimSpace(templates.PublicMessage)
	for _, template := range []string{templates.NewTurn, templates.PublicMessage} {
		if err := ValidateDiscordTemplate(template); err != nil {
			return store.DiscordTemplates{}, err
		}
	}
	return templates, nil
}

// message renders the announcement of a game from a template; it mentions what
// the template itself mentions.
func (s *DiscordService) message(config store.DiscordWebhookConfig, game DiscordGame, template string) DiscordMessage {
	return DiscordMessage{
		URL:     config.URL,
		Content: RenderDiscordTemplate(template, config.Language, DiscordVars{Game: game.Name, Turn: game.Turn, Season: game.Season, URL: gameURL(s.baseURL, game.ID)}),
		Mention: DiscordMentionsOf(template),
	}
}

func statusOf(config store.DiscordWebhookConfig, ok bool) DiscordStatus {
	status := DiscordStatus{Defaults: discordDefaults}
	if !ok {
		return status
	}
	status.Configured = true
	status.Language = config.Language
	status.Templates = config.Templates
	status.Disabled = config.Disabled
	status.DisabledReason = config.DisabledReason
	return status
}

// Active reports whether the game announces to Discord; it is the only thing
// the other players learn.
func (s *DiscordService) Active(ctx context.Context, id store.GameID) (bool, error) {
	if active, cached := s.active.get(id, s.now()); cached {
		return active, nil
	}
	config, ok, err := s.webhooks.DiscordWebhook(ctx, id)
	if err != nil {
		return false, err
	}
	active := ok && config.Active()
	s.active.put(id, active, s.now())
	return active, nil
}

func (s *DiscordService) Status(ctx context.Context, id store.GameID) (DiscordStatus, error) {
	config, ok, err := s.webhooks.DiscordWebhook(ctx, id)
	if err != nil {
		return DiscordStatus{}, err
	}
	return statusOf(config, ok), nil
}

// Save adds or replaces the webhook. A new URL re-enables a disabled webhook.
func (s *DiscordService) Save(ctx context.Context, id store.GameID, request DiscordSave) (DiscordStatus, error) {
	language, err := normalizeLanguage(request.Language)
	if err != nil {
		return DiscordStatus{}, err
	}
	templates, err := normalizeTemplates(request.Templates)
	if err != nil {
		return DiscordStatus{}, err
	}
	config, ok, err := s.webhooks.DiscordWebhook(ctx, id)
	if err != nil {
		return DiscordStatus{}, err
	}
	if rawURL := strings.TrimSpace(request.URL); rawURL != "" {
		if err := ValidateDiscordWebhookURL(rawURL); err != nil {
			return DiscordStatus{}, err
		}
		config = store.DiscordWebhookConfig{URL: rawURL}
	} else if !ok {
		return DiscordStatus{}, ErrWebhookNotConfigured
	}
	config.Language = language
	config.Templates = templates
	config.UpdatedAt = s.now().UTC()
	err = s.webhooks.SaveDiscordWebhook(ctx, id, config)
	s.active.remove(id)
	if err != nil {
		return DiscordStatus{}, err
	}
	return statusOf(config, true), nil
}

func (s *DiscordService) Delete(ctx context.Context, id store.GameID) error {
	err := s.webhooks.DeleteDiscordWebhook(ctx, id)
	s.active.remove(id)
	return err
}

// Preview renders a template with example values. An empty template previews
// the default one.
func (s *DiscordService) Preview(event, language, template string) (DiscordPreview, error) {
	language, err := normalizeLanguage(language)
	if err != nil {
		return DiscordPreview{}, err
	}
	kind, err := kindOfEvent(event)
	if err != nil {
		return DiscordPreview{}, err
	}
	template = strings.TrimSpace(template)
	if err := ValidateDiscordTemplate(template); err != nil {
		return DiscordPreview{}, err
	}
	if template == "" {
		template = DefaultDiscordTemplate(kind, language)
	}
	example := "The Western Wolves"
	if language == store.NotificationLanguageFrench {
		example = "Les Loups de l'Ouest"
	}
	content := RenderDiscordTemplate(template, language, DiscordVars{
		Game: example, Turn: 12, Season: models.SeasonAutumn, URL: gameURL(s.baseURL, "example"),
	})
	mention := DiscordMentionsOf(template)
	preview := DiscordPreview{
		Content:         content,
		Length:          utf8.RuneCountInString(content),
		MentionEveryone: len(mention.Parse) > 0,
		MentionRoles:    mention.Roles,
	}
	return preview, nil
}

// ErrUnknownEvent is returned for an event other than newTurn or publicMessage.
var ErrUnknownEvent = errors.New("notify: unknown discord event")

func kindOfEvent(event string) (Kind, error) {
	switch event {
	case EventNewTurn:
		return KindNewTurn, nil
	case EventPublicMessage:
		return KindPublicMessage, nil
	}
	return "", ErrUnknownEvent
}

// Test sends a test message, which mentions nobody. A webhook that answers is
// enabled again; one Discord refuses is disabled.
func (s *DiscordService) Test(ctx context.Context, game DiscordGame, rawURL, rawLanguage string) error {
	// A URL given with the test is the one of the form: it is tried as is,
	// and nothing about the stored webhook changes.
	rawURL = strings.TrimSpace(rawURL)
	adhoc := rawURL != ""
	var config store.DiscordWebhookConfig
	if adhoc {
		if err := ValidateDiscordWebhookURL(rawURL); err != nil {
			return err
		}
		language, err := normalizeLanguage(rawLanguage)
		if err != nil {
			return err
		}
		config = store.DiscordWebhookConfig{URL: rawURL, Language: language}
	} else {
		stored, ok, err := s.webhooks.DiscordWebhook(ctx, game.ID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrWebhookNotConfigured
		}
		config = stored
	}
	now := s.now()
	s.mu.Lock()
	if last, seen := s.lastTest[game.ID]; seen && now.Sub(last) < discordTestInterval {
		s.mu.Unlock()
		return ErrTestRateLimited
	}
	if len(s.lastTest) >= 256 {
		for other, at := range s.lastTest {
			if now.Sub(at) >= discordTestInterval {
				delete(s.lastTest, other)
			}
		}
	}
	previousTest, hadPreviousTest := s.lastTest[game.ID]
	s.lastTest[game.ID] = now
	s.mu.Unlock()
	// A test that did not leave must not hold back the retry of the host.
	forgetTest := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.lastTest[game.ID] != now {
			return
		}
		if hadPreviousTest {
			s.lastTest[game.ID] = previousTest
		} else {
			delete(s.lastTest, game.ID)
		}
	}
	err := s.sender.Send(ctx, s.message(config, game, discordTestTemplate(config.Language)))
	if errors.Is(err, ErrWebhookGone) {
		forgetTest()
		if !adhoc {
			s.disable(ctx, game.ID, config)
		}
		return err
	}
	if err != nil {
		forgetTest()
		log.Printf("notify: discord test game=%s: %v", game.ID, err)
		return err
	}
	if !adhoc && config.Disabled {
		config.Disabled, config.DisabledReason, config.DisabledAt = false, "", time.Time{}
		err := s.webhooks.SaveDiscordWebhook(ctx, game.ID, config)
		s.active.remove(game.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// Announce posts the announcement of an event: a new turn or a public room
// message. Public room announcements are limited per game. It returns nothing:
// a failure is logged and the webhook disabled when Discord refuses it.
func (s *DiscordService) Announce(ctx context.Context, kind Kind, game DiscordGame) {
	if kind != KindNewTurn && kind != KindPublicMessage {
		return
	}
	reference := string(kind) + " game=" + string(game.ID)
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("notify: discord %s: panic: %v", reference, recovered)
		}
	}()
	if ctx.Err() != nil {
		return
	}
	config, ok, err := s.webhooks.DiscordWebhook(ctx, game.ID)
	if err != nil {
		log.Printf("notify: discord %s: read webhook: %v", reference, err)
		return
	}
	if !ok || !config.Active() {
		return
	}
	at := s.now().UTC()
	claimed := false
	if kind == KindPublicMessage {
		// The configuration just read already tells most refusals, which
		// spares a transaction per message of a busy room.
		if !config.LastPublicAt.IsZero() && at.Sub(config.LastPublicAt) < DiscordPublicInterval {
			return
		}
		if claimed, err = s.webhooks.ClaimDiscordPublic(ctx, game.ID, at, DiscordPublicInterval); err != nil {
			log.Printf("notify: discord %s: claim: %v", reference, err)
			return
		}
		if !claimed {
			return
		}
	}
	template := config.Templates.NewTurn
	if kind == KindPublicMessage {
		template = config.Templates.PublicMessage
	}
	if template == "" {
		template = DefaultDiscordTemplate(kind, config.Language)
	}
	err = s.sender.Send(ctx, s.message(config, game, template))
	if err == nil {
		return
	}
	log.Printf("notify: discord %s: %v", reference, err)
	if errors.Is(err, ErrWebhookGone) {
		s.disable(ctx, game.ID, config)
		return
	}
	if claimed {
		// The announcement did not leave: give the claim back.
		releaseContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
		defer cancel()
		if releaseErr := s.webhooks.ReleaseDiscordPublic(releaseContext, game.ID, at); releaseErr != nil {
			log.Printf("notify: discord %s: release claim: %v", reference, releaseErr)
		}
	}
}

// disable marks a webhook Discord refused; it is never tried again until the
// host saves a new URL or a test succeeds.
func (s *DiscordService) disable(ctx context.Context, id store.GameID, config store.DiscordWebhookConfig) {
	disableContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), discordDisableTimeout)
	defer cancel()
	err := s.webhooks.DisableDiscordWebhook(disableContext, id, config.URL, store.DiscordDisabledGone, s.now().UTC())
	s.active.remove(id)
	if err != nil {
		log.Printf("notify: discord game=%s: disable webhook: %v", id, err)
		return
	}
	log.Printf("notify: discord game=%s: webhook refused by Discord, disabled", id)
}
