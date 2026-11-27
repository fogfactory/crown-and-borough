package notify

import (
	"context"
	"errors"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// ProfileReader gives the notifier the email and settings of a player.
type ProfileReader interface {
	GetProfile(ctx context.Context, uid string) (store.PlayerProfile, error)
}

// Notifier turns game events into emails. Failures are logged and never
// returned: a notification must not change the result of the request that
// caused it.
type Notifier struct {
	mailer   Mailer
	profiles ProfileReader
	state    store.NotificationStateStore
	baseURL  string
	now      func() time.Time
	// discord announces to the Discord webhook of a game; nil when the
	// channel is not mounted (hotseat, store without webhooks).
	discord *DiscordService
	// profileCache holds the profiles read lately. claimCache holds the
	// (game, player) pairs whose message email was just claimed or refused:
	// neither outcome can change before the player reads the chat again.
	profileCache ttlCache[string, store.PlayerProfile]
	claimCache   ttlCache[claimKey, struct{}]
}

type claimKey struct {
	game store.GameID
	uid  string
}

// maxConcurrentRecipients bounds the recipients of one event handled at the
// same time, and so the simultaneous connections to the mail server.
const maxConcurrentRecipients = 4

// releaseTimeout bounds giving a message email claim back after a failed send.
const releaseTimeout = 2 * time.Second

// NewNotifier builds a notifier; baseURL is the public URL of the application
// (PUBLIC_APP_URL) the links of the emails start with.
func NewNotifier(mailer Mailer, profiles ProfileReader, state store.NotificationStateStore, baseURL string) *Notifier {
	return &Notifier{mailer: mailer, profiles: profiles, state: state, baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), now: time.Now}
}

// SetDiscord mounts the Discord channel: new turns and public room messages
// are also announced to the webhook of the game, if it has one.
func (n *Notifier) SetDiscord(discord *DiscordService) { n.discord = discord }

// announce runs the Discord announcement beside the emails, which a slow mail
// server must not delay. The returned function waits for it.
func (n *Notifier) announce(ctx context.Context, kind Kind, game DiscordGame) (wait func()) {
	if n.discord == nil {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		n.discord.Announce(ctx, kind, game)
	}()
	return func() { <-done }
}

// discordGameOfSnapshot and discordGameOfChat say what an announcement tells of
// the game an event belongs to.
func discordGameOfSnapshot(snapshot store.GameSnapshot) DiscordGame {
	return DiscordGame{ID: snapshot.ID, Name: snapshot.Name, Turn: snapshot.State.Turn, Season: snapshot.State.Season}
}

func discordGameOfChat(result store.ChatPostResult) DiscordGame {
	return DiscordGame{ID: result.GameID, Name: result.GameName, Turn: result.Turn, Season: result.Season}
}

// DiscordNotifier announces to the Discord webhooks only: the notifier of a
// server that sends no emails.
type DiscordNotifier struct{ discord *DiscordService }

// NewDiscordNotifier builds a notifier that only announces to Discord.
func NewDiscordNotifier(discord *DiscordService) *DiscordNotifier {
	return &DiscordNotifier{discord: discord}
}

// TurnStarted announces the new turn of a game that is still being played.
func (d *DiscordNotifier) TurnStarted(ctx context.Context, snapshot store.GameSnapshot) {
	if snapshot.Status != store.StatusPlaying || !snapshot.Started() {
		return
	}
	d.discord.Announce(ctx, KindNewTurn, discordGameOfSnapshot(snapshot))
}

// MessagePosted announces the messages of the public room; private ones never
// reach Discord.
func (d *DiscordNotifier) MessagePosted(ctx context.Context, result store.ChatPostResult) {
	if result.Message.Channel != store.ChatPublicChannel {
		return
	}
	d.discord.Announce(ctx, KindPublicMessage, discordGameOfChat(result))
}

// SetClock replaces the clock; for tests.
func (n *Notifier) SetClock(now func() time.Time) { n.now = now }

// TurnStarted emails the opted-in houses of a game whose turn was just
// resolved. Nothing is sent once the game is finished, nor to eliminated
// houses or the observer.
func (n *Notifier) TurnStarted(ctx context.Context, snapshot store.GameSnapshot) {
	if snapshot.Status != store.StatusPlaying || !snapshot.Started() {
		return
	}
	var uids []string
	for _, slot := range snapshot.Players {
		if !store.IsAssignedActor(strings.TrimSpace(slot.ActorID)) || !engine.PlayerAlive(snapshot.State, slot.ID) {
			continue
		}
		if !slices.Contains(uids, slot.ActorID) {
			uids = append(uids, slot.ActorID)
		}
	}
	wait := n.announce(ctx, KindNewTurn, discordGameOfSnapshot(snapshot))
	n.notify(ctx, KindNewTurn, snapshot.ID, snapshot.Name, uids)
	wait()
}

// MessagePosted emails the opted-in houses a chat message concerns, at most
// one email per player until they read their chat again.
func (n *Notifier) MessagePosted(ctx context.Context, result store.ChatPostResult) {
	kind := KindPublicMessage
	wait := func() {}
	if result.Message.Channel != store.ChatPublicChannel {
		kind = KindPrivateMessage
	} else {
		// Only the public room reaches Discord: private messages never do.
		wait = n.announce(ctx, KindPublicMessage, discordGameOfChat(result))
	}
	n.notify(ctx, kind, result.GameID, result.GameName, result.RecipientUIDs)
	wait()
}

func (n *Notifier) notify(ctx context.Context, kind Kind, id store.GameID, gameName string, uids []string) {
	eventAt := n.now().UTC()
	// Recipients are independent: handle them side by side so a slow mail
	// server does not use up the caller's time budget on the first ones.
	var group sync.WaitGroup
	slots := make(chan struct{}, maxConcurrentRecipients)
	for _, uid := range uids {
		group.Add(1)
		slots <- struct{}{}
		go func() {
			defer group.Done()
			defer func() { <-slots }()
			reference := string(kind) + " game=" + string(id) + " uid=" + uid
			// A panic here would not reach the caller's recover and would
			// stop the server.
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Printf("notify: %s: panic: %v", reference, recovered)
				}
			}()
			if ctx.Err() != nil {
				return
			}
			if err := n.notifyOne(ctx, kind, id, gameName, uid, eventAt, reference); err != nil {
				log.Printf("notify: %s: %v", reference, err)
			}
		}()
	}
	group.Wait()
}

func (n *Notifier) notifyOne(ctx context.Context, kind Kind, id store.GameID, gameName, uid string, eventAt time.Time, reference string) error {
	profile, err := n.profile(ctx, uid)
	if errors.Is(err, store.ErrProfileNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	preferences := profile.Notifications
	// Cheap checks first: the activity is only read when the reduce option
	// may need it.
	var lastActive time.Time
	if preferences.ReduceWhenActive {
		if lastActive, err = n.state.LastActivity(ctx, uid); err != nil {
			return err
		}
	}
	if !ShouldSend(preferences, profile.Email, kind, lastActive, eventAt) {
		return nil
	}
	// Without a mail server nothing is sent: the anti-spam marker is left
	// alone so the first real email is not held back.
	_, silent := n.mailer.(LogMailer)
	claimed := false
	key := claimKey{game: id, uid: uid}
	if kind.IsMessage() && !silent {
		if _, recent := n.claimCache.get(key, n.now()); recent {
			return nil
		}
		if claimed, err = n.state.ClaimMessageEmail(ctx, id, uid, eventAt); err != nil {
			return err
		}
		n.claimCache.put(key, struct{}{}, n.now())
		if !claimed {
			return nil
		}
	}
	subject, body := renderEmail(kind, preferences.Language, safeGameName(gameName), gameURL(n.baseURL, id), profileURL(n.baseURL))
	email := Email{To: profile.Email, Subject: subject, Body: body, Reference: reference}
	err = n.mailer.Send(ctx, email)
	if err != nil && ctx.Err() == nil {
		// One more try while there is time: messages that arrived during
		// this send rely on it, as the claim made them skip theirs.
		log.Printf("notify: %s: retrying after: %v", reference, err)
		err = n.mailer.Send(ctx, email)
	}
	if err != nil && claimed {
		// The email did not leave: give the claim back so the next message
		// tries again. ctx may be the reason of the failure, hence a new one.
		n.claimCache.remove(key)
		releaseContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
		defer cancel()
		if releaseErr := n.state.ReleaseMessageEmail(releaseContext, id, uid, eventAt); releaseErr != nil {
			log.Printf("notify: %s: release claim: %v", reference, releaseErr)
		}
	}
	return err
}

// profile reads a player's profile through the cache.
func (n *Notifier) profile(ctx context.Context, uid string) (store.PlayerProfile, error) {
	if cached, ok := n.profileCache.get(uid, n.now()); ok {
		return cached, nil
	}
	profile, err := n.profiles.GetProfile(ctx, uid)
	if err != nil {
		return store.PlayerProfile{}, err
	}
	n.profileCache.put(uid, profile, n.now())
	return profile, nil
}

// ActivityToucher is what the activity tracker writes through.
type ActivityToucher interface {
	TouchActivity(ctx context.Context, uid string, at time.Time) error
}

// ActivityTracker records the last authenticated request of each player. A
// player is written at most once per interval and server instance, so the cost
// stays negligible.
type ActivityTracker struct {
	store    ActivityToucher
	interval time.Duration
	now      func() time.Time
	mu       sync.Mutex
	last     map[string]time.Time
	writes   sync.WaitGroup
}

// ActivityWriteInterval is the minimum delay between two writes for a player.
const ActivityWriteInterval = 2 * time.Minute

// activityWriteTimeout bounds the write of an activity record.
const activityWriteTimeout = 2 * time.Second

func NewActivityTracker(toucher ActivityToucher) *ActivityTracker {
	return &ActivityTracker{
		store:    toucher,
		interval: ActivityWriteInterval,
		now:      time.Now,
		last:     make(map[string]time.Time),
	}
}

// SetClock replaces the clock; for tests.
func (t *ActivityTracker) SetClock(now func() time.Time) { t.now = now }

// Touch records uid as active now unless it was written less than one
// interval ago. It does not wait for the write; errors are logged and ignored.
func (t *ActivityTracker) Touch(ctx context.Context, uid string) {
	now := t.now()
	t.mu.Lock()
	previous, seen := t.last[uid]
	if seen && now.Sub(previous) < t.interval {
		t.mu.Unlock()
		return
	}
	if len(t.last) >= cachePruneSize {
		// Entries older than the interval no longer throttle anything.
		for other, at := range t.last {
			if now.Sub(at) >= t.interval {
				delete(t.last, other)
			}
		}
	}
	t.last[uid] = now
	t.mu.Unlock()
	// The write runs beside the request instead of delaying it; the request
	// usually outlasts it, which keeps the CPU available to finish.
	t.writes.Add(1)
	go func() {
		defer t.writes.Done()
		writeContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), activityWriteTimeout)
		defer cancel()
		if err := t.store.TouchActivity(writeContext, uid, now.UTC()); err != nil {
			log.Printf("notify: record activity uid=%s: %v", uid, err)
			// Let the next request retry.
			t.mu.Lock()
			delete(t.last, uid)
			t.mu.Unlock()
		}
	}()
}

// Wait blocks until the writes started so far are done; for tests.
func (t *ActivityTracker) Wait() { t.writes.Wait() }
