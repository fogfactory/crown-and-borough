package notify

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

type fakeMailer struct {
	mu   sync.Mutex
	sent []Email
	err  error
}

func (m *fakeMailer) Send(_ context.Context, email Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, email)
	return nil
}

func (m *fakeMailer) recipients() string {
	var to []string
	for _, email := range m.sent {
		to = append(to, email.To)
	}
	return strings.Join(to, ",")
}

type fixture struct {
	notifier *Notifier
	mailer   *fakeMailer
	profiles *store.MemoryProfileStore
	state    *store.MemoryChatBackend
	clock    *time.Time
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	mailer := &fakeMailer{}
	profiles := store.NewMemoryProfileStore()
	state := store.NewMemoryChatBackend()
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	notifier := NewNotifier(mailer, profiles, state, "https://game.example.test/")
	notifier.SetClock(func() time.Time { return clock })
	return fixture{notifier: notifier, mailer: mailer, profiles: profiles, state: state, clock: &clock}
}

// player creates a profile with the given settings; an email is derived from
// the uid.
func (f fixture) player(t *testing.T, uid string, preferences store.NotificationPreferences) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.profiles.EnsureProfile(ctx, store.Actor{ID: uid, Email: uid + "@example.test"}); err != nil {
		t.Fatal(err)
	}
	if preferences == (store.NotificationPreferences{}) {
		return
	}
	update := store.NotificationsUpdate{
		NewTurn: &preferences.NewTurn, PrivateMessage: &preferences.PrivateMessage,
		PublicMessage: &preferences.PublicMessage, ReduceWhenActive: &preferences.ReduceWhenActive,
	}
	if preferences.Language != "" {
		update.Language = &preferences.Language
	}
	if _, err := f.profiles.UpdateNotifications(ctx, uid, update); err != nil {
		t.Fatal(err)
	}
}

func turnSnapshot(status store.Status) store.GameSnapshot {
	capital := models.InfraID("castle")
	return store.GameSnapshot{
		ID:     "g1",
		Name:   "Les Trois Rivières",
		Status: status,
		Players: []store.PlayerSlot{
			{ID: "P1", ActorID: "alice"},
			{ID: "P2", ActorID: "bob"},
			{ID: "P3", ActorID: "cleo"}, // eliminated
			{ID: "P4", ActorID: "slot:P4"},
		},
		SpectatorUID: "host",
		State: &models.GameState{Players: []models.Player{
			{ID: "P1", CapitalCastleID: &capital},
			{ID: "P2", CapitalCastleID: &capital},
			{ID: "P3"},
			{ID: "P4", CapitalCastleID: &capital},
		}},
	}
}

func publicMessage(recipients ...string) store.ChatPostResult {
	return store.ChatPostResult{
		Message:       store.ChatMessage{Channel: store.ChatPublicChannel, Body: "secret plan"},
		GameID:        "g1",
		GameName:      "Les Trois Rivières",
		RecipientUIDs: recipients,
	}
}

func privateMessage(recipients ...string) store.ChatPostResult {
	result := publicMessage(recipients...)
	result.Message.Channel = "P1_P2"
	return result
}

func TestTurnStartedEmailsOnlyOptedInLivingPlayers(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{NewTurn: true})
	f.player(t, "bob", store.NotificationPreferences{})                      // default profile
	f.player(t, "cleo", store.NotificationPreferences{NewTurn: true})        // eliminated
	f.player(t, "host", store.NotificationPreferences{NewTurn: true})        // observer
	f.player(t, "dave", store.NotificationPreferences{PrivateMessage: true}) // other opt-in only
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	if got := f.mailer.recipients(); got != "alice@example.test" {
		t.Fatalf("recipients = %q", got)
	}
	email := f.mailer.sent[0]
	if !strings.Contains(email.Body, "https://game.example.test/games/g1") || !strings.Contains(email.Body, "https://game.example.test/profile") {
		t.Fatalf("body lacks the links: %q", email.Body)
	}
	if !strings.Contains(email.Subject, "Les Trois Rivières") || !strings.Contains(email.Subject, "nouveau tour") {
		t.Fatalf("subject = %q (default language is French)", email.Subject)
	}
}

func TestTurnStartedDefaultProfilesReceiveNothing(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{})
	f.player(t, "bob", store.NotificationPreferences{})
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	f.notifier.MessagePosted(context.Background(), publicMessage("alice", "bob"))
	f.notifier.MessagePosted(context.Background(), privateMessage("bob"))
	if len(f.mailer.sent) != 0 {
		t.Fatalf("sent %d emails to default profiles", len(f.mailer.sent))
	}
}

func TestTurnStartedSendsNothingOnceTheGameIsFinished(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{NewTurn: true})
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusFinished))
	if len(f.mailer.sent) != 0 {
		t.Fatalf("sent %d emails after the game finished", len(f.mailer.sent))
	}
}

func TestTurnStartedUsesTheEnglishTemplateWhenAsked(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{NewTurn: true, Language: "en"})
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	if len(f.mailer.sent) != 1 || !strings.Contains(f.mailer.sent[0].Subject, "new turn") {
		t.Fatalf("sent = %+v", f.mailer.sent)
	}
}

func TestPrivateAndPublicMessageOptInsAreSeparate(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{PrivateMessage: true})
	f.player(t, "bob", store.NotificationPreferences{PublicMessage: true})
	ctx := context.Background()

	f.notifier.MessagePosted(ctx, privateMessage("bob"))
	if len(f.mailer.sent) != 0 {
		t.Fatalf("private message sent to a public-only player: %q", f.mailer.recipients())
	}
	f.notifier.MessagePosted(ctx, privateMessage("alice"))
	if got := f.mailer.recipients(); got != "alice@example.test" {
		t.Fatalf("after private message: %q", got)
	}
	f.notifier.MessagePosted(ctx, publicMessage("alice"))
	if len(f.mailer.sent) != 1 {
		t.Fatalf("public message sent to a private-only player: %q", f.mailer.recipients())
	}
	f.notifier.MessagePosted(ctx, publicMessage("alice", "bob"))
	if got := f.mailer.recipients(); got != "alice@example.test,bob@example.test" {
		t.Fatalf("after public message: %q", got)
	}
}

func TestMessageEmailsNeverCarryTheContent(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{PrivateMessage: true})
	f.notifier.MessagePosted(context.Background(), privateMessage("alice"))
	if len(f.mailer.sent) != 1 {
		t.Fatalf("sent = %d", len(f.mailer.sent))
	}
	email := f.mailer.sent[0]
	if strings.Contains(email.Subject+email.Body, "secret plan") {
		t.Fatalf("email leaks the message: %+v", email)
	}
	if strings.Contains(email.Reference, "@") {
		t.Fatalf("log reference holds an address: %q", email.Reference)
	}
}

func TestReduceWhenActiveSkipsActivePlayersOnly(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{NewTurn: true, PublicMessage: true, ReduceWhenActive: true})
	f.player(t, "bob", store.NotificationPreferences{NewTurn: true, ReduceWhenActive: true})
	ctx := context.Background()
	if err := f.state.TouchActivity(ctx, "alice", f.clock.Add(-2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := f.state.TouchActivity(ctx, "bob", f.clock.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	f.notifier.TurnStarted(ctx, turnSnapshot(store.StatusPlaying))
	if got := f.mailer.recipients(); got != "bob@example.test" {
		t.Fatalf("turn recipients = %q, want only the inactive player", got)
	}

	// A skipped message email does not consume the anti-spam claim.
	f.notifier.MessagePosted(ctx, publicMessage("alice"))
	if len(f.mailer.sent) != 1 {
		t.Fatalf("active player got a message email: %q", f.mailer.recipients())
	}
	*f.clock = f.clock.Add(10 * time.Minute)
	f.notifier.MessagePosted(ctx, publicMessage("alice"))
	if got := f.mailer.recipients(); got != "bob@example.test,alice@example.test" {
		t.Fatalf("once inactive: %q", got)
	}
}

func TestMessageEmailIsSentOncePerUnreadStretch(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{PrivateMessage: true, PublicMessage: true})
	ctx := context.Background()

	f.notifier.MessagePosted(ctx, privateMessage("alice"))
	*f.clock = f.clock.Add(time.Minute)
	f.notifier.MessagePosted(ctx, privateMessage("alice"))
	f.notifier.MessagePosted(ctx, publicMessage("alice"))
	if len(f.mailer.sent) != 1 {
		t.Fatalf("emails before reading = %d, want 1", len(f.mailer.sent))
	}

	// Reading any channel of the game re-arms the notification.
	*f.clock = f.clock.Add(time.Minute)
	if _, err := f.state.SetChatReadCursor(ctx, "g1", "alice", store.ChatReadCursor{Channel: "public", LastReadAt: *f.clock}); err != nil {
		t.Fatal(err)
	}
	*f.clock = f.clock.Add(time.Minute)
	f.notifier.MessagePosted(ctx, publicMessage("alice"))
	f.notifier.MessagePosted(ctx, publicMessage("alice"))
	if len(f.mailer.sent) != 2 {
		t.Fatalf("emails after reading = %d, want 2", len(f.mailer.sent))
	}

	// The marker is per game.
	other := publicMessage("alice")
	other.GameID = "g2"
	f.notifier.MessagePosted(ctx, other)
	if len(f.mailer.sent) != 3 {
		t.Fatalf("emails for another game = %d, want 3", len(f.mailer.sent))
	}
}

func TestConcurrentMessagesSendOneEmail(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{PublicMessage: true})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.notifier.MessagePosted(context.Background(), publicMessage("alice"))
		}()
	}
	wg.Wait()
	if len(f.mailer.sent) != 1 {
		t.Fatalf("concurrent posts sent %d emails, want 1", len(f.mailer.sent))
	}
}

func TestMailerFailuresAreSwallowed(t *testing.T) {
	f := newFixture(t)
	f.mailer.err = errors.New("smtp down")
	f.player(t, "alice", store.NotificationPreferences{NewTurn: true})
	f.player(t, "bob", store.NotificationPreferences{NewTurn: true})
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	f.mailer.err = nil
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	if len(f.mailer.sent) != 2 {
		t.Fatalf("sent = %d, want 2 after recovery", len(f.mailer.sent))
	}
}

func TestPlayersWithoutProfileOrEmailAreSkipped(t *testing.T) {
	f := newFixture(t)
	// alice has no profile at all; bob has settings but no email.
	if _, err := f.profiles.EnsureProfile(context.Background(), store.Actor{ID: "bob"}); err != nil {
		t.Fatal(err)
	}
	on := true
	if _, err := f.profiles.UpdateNotifications(context.Background(), "bob", store.NotificationsUpdate{NewTurn: &on}); err != nil {
		t.Fatal(err)
	}
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
	if len(f.mailer.sent) != 0 {
		t.Fatalf("sent = %q", f.mailer.recipients())
	}
}

type countingToucher struct {
	mu     sync.Mutex
	writes map[string][]time.Time
	err    error
}

func (c *countingToucher) TouchActivity(_ context.Context, uid string, at time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.writes[uid] = append(c.writes[uid], at)
	return nil
}

func TestActivityTrackerThrottlesWritesPerPlayer(t *testing.T) {
	toucher := &countingToucher{writes: map[string][]time.Time{}}
	tracker := NewActivityTracker(toucher)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tracker.SetClock(func() time.Time { return now })
	ctx := context.Background()

	tracker.Touch(ctx, "alice")
	now = now.Add(time.Minute)
	tracker.Touch(ctx, "alice")
	tracker.Touch(ctx, "bob")
	if len(toucher.writes["alice"]) != 1 || len(toucher.writes["bob"]) != 1 {
		t.Fatalf("writes = %v", toucher.writes)
	}
	now = now.Add(ActivityWriteInterval)
	tracker.Touch(ctx, "alice")
	if len(toucher.writes["alice"]) != 2 {
		t.Fatalf("alice writes = %v", toucher.writes["alice"])
	}
}

func TestActivityTrackerRetriesAfterAFailedWrite(t *testing.T) {
	toucher := &countingToucher{writes: map[string][]time.Time{}, err: errors.New("firestore down")}
	tracker := NewActivityTracker(toucher)
	ctx := context.Background()
	tracker.Touch(ctx, "alice") // must not panic or return an error
	toucher.err = nil
	tracker.Touch(ctx, "alice")
	if len(toucher.writes["alice"]) != 1 {
		t.Fatalf("writes = %v", toucher.writes)
	}
}

// countingProfiles counts the profile reads that reach the store.
type countingProfiles struct {
	ProfileReader
	reads atomic.Int32
}

func (c *countingProfiles) GetProfile(ctx context.Context, uid string) (store.PlayerProfile, error) {
	c.reads.Add(1)
	return c.ProfileReader.GetProfile(ctx, uid)
}

func TestProfilesAreReadOncePerCacheWindow(t *testing.T) {
	f := newFixture(t)
	f.player(t, "bob", store.NotificationPreferences{})
	profiles := &countingProfiles{ProfileReader: f.profiles}
	f.notifier.profiles = profiles
	ctx := context.Background()

	f.notifier.MessagePosted(ctx, publicMessage("bob"))
	f.notifier.MessagePosted(ctx, publicMessage("bob"))
	if got := profiles.reads.Load(); got != 1 {
		t.Fatalf("profile reads = %d, want 1 within the cache window", got)
	}

	// A setting saved meanwhile applies once the window has passed.
	enabled := true
	if _, err := f.profiles.UpdateNotifications(ctx, "bob", store.NotificationsUpdate{PublicMessage: &enabled}); err != nil {
		t.Fatal(err)
	}
	*f.clock = f.clock.Add(cacheTTL)
	f.notifier.MessagePosted(ctx, publicMessage("bob"))
	if got := profiles.reads.Load(); got != 2 {
		t.Fatalf("profile reads = %d, want 2 after the cache window", got)
	}
	if got := f.mailer.recipients(); got != "bob@example.test" {
		t.Fatalf("recipients = %q, want bob once the new setting is read", got)
	}
}

// countingState counts the message email claims that reach the store.
type countingState struct {
	store.NotificationStateStore
	claims atomic.Int32
}

func (c *countingState) ClaimMessageEmail(ctx context.Context, id store.GameID, uid string, at time.Time) (bool, error) {
	c.claims.Add(1)
	return c.NotificationStateStore.ClaimMessageEmail(ctx, id, uid, at)
}

func TestMessageClaimsAreNotRepeatedWithinTheCacheWindow(t *testing.T) {
	f := newFixture(t)
	f.player(t, "bob", store.NotificationPreferences{PrivateMessage: true})
	state := &countingState{NotificationStateStore: f.state}
	f.notifier.state = state
	ctx := context.Background()

	for range 3 {
		f.notifier.MessagePosted(ctx, privateMessage("bob"))
	}
	if got := state.claims.Load(); got != 1 {
		t.Fatalf("claims = %d, want 1 within the cache window", got)
	}
	if got := len(f.mailer.sent); got != 1 {
		t.Fatalf("emails = %d, want 1", got)
	}
	*f.clock = f.clock.Add(cacheTTL)
	f.notifier.MessagePosted(ctx, privateMessage("bob"))
	if got := state.claims.Load(); got != 2 {
		t.Fatalf("claims = %d, want 2 after the cache window", got)
	}
	if got := len(f.mailer.sent); got != 1 {
		t.Fatalf("emails = %d, want still 1: the chat was not read", got)
	}
}

func TestFailedMessageEmailGivesItsClaimBack(t *testing.T) {
	f := newFixture(t)
	f.player(t, "bob", store.NotificationPreferences{PrivateMessage: true})
	ctx := context.Background()

	f.mailer.err = errors.New("smtp down")
	f.notifier.MessagePosted(ctx, privateMessage("bob"))
	f.mailer.err = nil
	f.notifier.MessagePosted(ctx, privateMessage("bob"))
	if got := len(f.mailer.sent); got != 1 {
		t.Fatalf("emails = %d, want 1: the message after a failed send must notify", got)
	}
	f.notifier.MessagePosted(ctx, privateMessage("bob"))
	if got := len(f.mailer.sent); got != 1 {
		t.Fatalf("emails = %d, want still 1 once an email left", got)
	}
}

func TestMessagesAreNotClaimedWithoutAMailServer(t *testing.T) {
	f := newFixture(t)
	f.player(t, "bob", store.NotificationPreferences{PrivateMessage: true})
	state := &countingState{NotificationStateStore: f.state}
	f.notifier.state = state
	f.notifier.mailer = LogMailer{}
	f.notifier.MessagePosted(context.Background(), privateMessage("bob"))
	if got := state.claims.Load(); got != 0 {
		t.Fatalf("claims = %d, want 0 while no email can be sent", got)
	}
}

type panickingMailer struct{}

func (panickingMailer) Send(context.Context, Email) error { panic("boom") }

func TestAPanicForOneRecipientIsContained(t *testing.T) {
	f := newFixture(t)
	f.player(t, "alice", store.NotificationPreferences{NewTurn: true})
	f.notifier.mailer = panickingMailer{}
	f.notifier.TurnStarted(context.Background(), turnSnapshot(store.StatusPlaying))
}
