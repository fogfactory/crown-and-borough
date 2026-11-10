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

// SetClock replaces the clock; for tests.
func (n *Notifier) SetClock(now func() time.Time) { n.now = now }

// TurnStarted emails the opted-in houses of a game whose turn was just
// resolved. Nothing is sent once the game is finished, nor to eliminated
// houses or the observer.
func (n *Notifier) TurnStarted(ctx context.Context, snapshot store.GameSnapshot) {
	if snapshot.Status != store.StatusPlaying || snapshot.State == nil {
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
	n.notify(ctx, KindNewTurn, snapshot.ID, snapshot.Name, uids)
}

// MessagePosted emails the opted-in houses a chat message concerns, at most
// one email per player until they read their chat again.
func (n *Notifier) MessagePosted(ctx context.Context, result store.ChatPostResult) {
	kind := KindPublicMessage
	if result.Message.Channel != store.ChatPublicChannel {
		kind = KindPrivateMessage
	}
	n.notify(ctx, kind, result.GameID, result.GameName, result.RecipientUIDs)
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
	subject, body := renderEmail(kind, preferences.Language, strings.Join(strings.Fields(gameName), " "), gameURL(n.baseURL, id), profileURL(n.baseURL))
	err = n.mailer.Send(ctx, Email{To: profile.Email, Subject: subject, Body: body, Reference: reference})
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
}

// ActivityWriteInterval is the minimum delay between two writes for a player.
const ActivityWriteInterval = 2 * time.Minute

// activityWriteTimeout bounds the write so a slow database cannot slow a
// request noticeably.
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
// interval ago. Errors are logged and ignored.
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
	writeContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), activityWriteTimeout)
	defer cancel()
	if err := t.store.TouchActivity(writeContext, uid, now.UTC()); err != nil {
		log.Printf("notify: record activity uid=%s: %v", uid, err)
		// Let the next request retry.
		t.mu.Lock()
		delete(t.last, uid)
		t.mu.Unlock()
	}
}
