package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// newIncognitoGame creates a two-house incognito game for alice, whose profile
// is named, and returns its invitation code.
func newIncognitoGame(t *testing.T) (*MemoryStore, GameSnapshot, string) {
	t.Helper()
	gameStore := newTestStore(t)
	ctx := context.Background()
	for uid, name := range map[string]string{"alice": "Alice Account", "bob": "Bob Account"} {
		if _, err := gameStore.EnsureProfile(ctx, Actor{ID: uid, Email: uid + "@example.com"}); err != nil {
			t.Fatalf("ensure profile: %v", err)
		}
		if _, err := gameStore.UpdateProfile(ctx, uid, name); err != nil {
			t.Fatalf("set profile name: %v", err)
		}
	}
	created, err := gameStore.Create(ctx, Actor{ID: "alice"}, CreateRequest{
		Players:          []engine.PlayerInit{{Name: "Alice Account"}, {Name: "Other"}},
		StrictMembership: true,
		Incognito:        true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	secret, err := gameStore.CreateInvitation(ctx, Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	confirmCreator(t, gameStore, created.ID)
	return gameStore, created, secret.Code
}

func TestIncognitoGameKeepsPlaceholderNames(t *testing.T) {
	gameStore, created, code := newIncognitoGame(t)
	ctx := context.Background()
	if !created.Incognito || created.Started() {
		t.Fatalf("created incognito/started = %v/%v, want true/false", created.Incognito, created.Started())
	}
	joined, err := gameStore.Join(ctx, Actor{ID: "bob"}, created.ID, code, JoinOptions{})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if !joined.Snapshot.Started() {
		t.Fatal("game with every slot taken has not started")
	}
	// A profile rename must not reach the game either.
	if _, err := gameStore.UpdateProfile(ctx, "alice", "Alicia Account"); err != nil {
		t.Fatalf("rename profile: %v", err)
	}
	snapshot, err := gameStore.Get(ctx, Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for index, want := range []string{"P1", "P2"} {
		if snapshot.Players[index].Name != want || snapshot.State.Players[index].Name != want {
			t.Fatalf("house %d = slot %q/state %q, want %q", index, snapshot.Players[index].Name, snapshot.State.Players[index].Name, want)
		}
	}
}

func TestIncognitoGameWaitsForEverySlot(t *testing.T) {
	gameStore, created, code := newIncognitoGame(t)
	ctx := context.Background()
	if _, err := gameStore.Submit(ctx, Actor{ID: "alice"}, created.ID, SubmitRequest{}); !errors.Is(err, ErrGameNotStarted) {
		t.Fatalf("submit before start = %v, want ErrGameNotStarted", err)
	}
	if _, err := gameStore.Resolve(ctx, Actor{ID: "alice"}, created.ID); !errors.Is(err, ErrGameNotStarted) {
		t.Fatalf("resolve before start = %v, want ErrGameNotStarted", err)
	}
	if _, err := gameStore.Join(ctx, Actor{ID: "bob"}, created.ID, code, JoinOptions{}); err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, err := gameStore.Submit(ctx, Actor{ID: "alice"}, created.ID, SubmitRequest{}); err != nil {
		t.Fatalf("submit after start: %v", err)
	}
}

func TestOrdinaryGameShowsTheAccountNameBesideTheHouseName(t *testing.T) {
	gameStore := newTestStore(t)
	ctx := context.Background()
	if _, err := gameStore.EnsureProfile(ctx, Actor{ID: "bob"}); err != nil {
		t.Fatal(err)
	}
	if _, err := gameStore.UpdateProfile(ctx, "bob", "Bob Account"); err != nil {
		t.Fatal(err)
	}
	created, err := gameStore.Create(ctx, Actor{ID: "alice"}, CreateRequest{
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Other"}}, StrictMembership: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := gameStore.CreateInvitation(ctx, Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	confirmCreator(t, gameStore, created.ID)
	joined, err := gameStore.Join(ctx, Actor{ID: "bob"}, created.ID, secret.Code, JoinOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if joined.Player.Name != "Bob Account" || joined.Player.AccountName != "Bob Account" || joined.Snapshot.Incognito {
		t.Fatalf("ordinary game player = %#v", joined.Player)
	}
	// The house name is its own: renaming the profile moves the account name only.
	if _, err := gameStore.RenameHouse(ctx, Actor{ID: "bob"}, created.ID, "Bobbins"); err != nil {
		t.Fatalf("rename in an ordinary game: %v", err)
	}
	if _, err := gameStore.UpdateProfile(ctx, "bob", "Robert Account"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := gameStore.Get(ctx, Actor{ID: "bob"}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	slot := playerSlotForID(snapshot.Players, joined.Player.ID)
	if slot.Name != "Bobbins" || slot.AccountName != "Robert Account" {
		t.Fatalf("slot after profile rename = %#v", slot)
	}
}

func TestRenameHouseRules(t *testing.T) {
	gameStore, created, code := newIncognitoGame(t)
	ctx := context.Background()
	alice, bob := Actor{ID: "alice"}, Actor{ID: "bob"}
	if _, err := gameStore.Join(ctx, bob, created.ID, code, JoinOptions{}); err != nil {
		t.Fatalf("join: %v", err)
	}

	result, err := gameStore.RenameHouse(ctx, alice, created.ID, "  House   Stark ")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if result.Player.Name != "House Stark" || result.PreviousName != string(result.Player.ID) {
		t.Fatalf("rename result = %q (was %q)", result.Player.Name, result.PreviousName)
	}
	for _, player := range result.Snapshot.State.Players {
		if player.ID == result.Player.ID && player.Name != "House Stark" {
			t.Fatalf("state name = %q", player.Name)
		}
	}
	if _, err := gameStore.RenameHouse(ctx, bob, created.ID, "house stark"); !errors.Is(err, ErrHouseNameTaken) {
		t.Fatalf("duplicate name = %v, want ErrHouseNameTaken", err)
	}
	if _, err := gameStore.RenameHouse(ctx, bob, created.ID, ""); !errors.Is(err, ErrInvalidDisplayName) {
		t.Fatalf("empty name = %v, want ErrInvalidDisplayName", err)
	}
	if _, err := gameStore.RenameHouse(ctx, Actor{ID: "mallory"}, created.ID, "Intruders"); !errors.Is(err, ErrNotMember) {
		t.Fatalf("stranger = %v, want ErrNotMember", err)
	}
	if _, err := gameStore.Submit(ctx, alice, created.ID, SubmitRequest{}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := gameStore.RenameHouse(ctx, alice, created.ID, "House Lannister"); !errors.Is(err, ErrNameLocked) {
		t.Fatalf("rename after submit = %v, want ErrNameLocked", err)
	}
	// The other house submitted nothing: it may still rename.
	if _, err := gameStore.RenameHouse(ctx, bob, created.ID, "House Tyrell"); err != nil {
		t.Fatalf("other house rename: %v", err)
	}
}

func TestChatPostHouseRenamedBypassesRateLimitAndValidation(t *testing.T) {
	service, _, clock := newChatFixture(t)
	ctx := context.Background()
	alice := Actor{ID: "alice"}
	if _, err := service.Post(ctx, alice, "g1", ChatPostRequest{Channel: ChatPublicChannel, Body: "hello"}); err != nil {
		t.Fatalf("post: %v", err)
	}
	*clock = clock.Add(time.Millisecond)
	message, err := service.PostHouseRenamed(ctx, alice, "g1", "P1", "House Stark")
	if err != nil {
		t.Fatalf("announce rename: %v", err)
	}
	if message.Kind != ChatKindHouseRenamed || message.Body != "House Stark" || message.PreviousName != "P1" ||
		message.Channel != ChatPublicChannel || message.AuthorHouseID != "P1" {
		t.Fatalf("rename message = %#v", message)
	}
	history, err := service.History(ctx, alice, "g1", ChatHistoryRequest{Channel: ChatPublicChannel})
	if err != nil || len(history) != 2 || history[0].Kind != ChatKindHouseRenamed {
		t.Fatalf("history = %#v, %v", history, err)
	}
	// The system message does not count against the author's rate limit.
	*clock = clock.Add(ChatMinInterval)
	if _, err := service.Post(ctx, alice, "g1", ChatPostRequest{Channel: ChatPublicChannel, Body: "again"}); err != nil {
		t.Fatalf("post after rename: %v", err)
	}
}

// seatsAfterJoins plays a three-house incognito game where the guests join in
// the given order, and returns the house each member ended up with.
func seatsAfterJoins(t *testing.T, guests []string) map[string]string {
	t.Helper()
	gameStore := newTestStore(t)
	ctx := context.Background()
	created, err := gameStore.Create(ctx, Actor{ID: "alice"}, CreateRequest{
		Seed:             "seat-seed",
		Players:          []engine.PlayerInit{{}, {}, {}},
		StrictMembership: true,
		Incognito:        true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	secret, err := gameStore.CreateInvitation(ctx, Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	confirmCreator(t, gameStore, created.ID)
	for index, guest := range guests {
		result, err := gameStore.Join(ctx, Actor{ID: guest}, created.ID, secret.Code, JoinOptions{})
		if err != nil {
			t.Fatalf("join %s: %v", guest, err)
		}
		last := index == len(guests)-1
		if result.Snapshot.Started() != last {
			t.Fatalf("started after join %d = %v", index+1, !last)
		}
		_, stateErr := gameStore.State(ctx, Actor{ID: guest}, created.ID)
		_, mapErr := gameStore.Map(ctx, Actor{ID: guest}, created.ID)
		_, supplyErr := gameStore.Supply(ctx, Actor{ID: guest}, created.ID, "T1", "")
		if last {
			if stateErr != nil || mapErr != nil {
				t.Fatalf("state/map once started = %v/%v", stateErr, mapErr)
			}
			continue
		}
		for _, err := range []error{stateErr, mapErr, supplyErr} {
			if !errors.Is(err, ErrGameNotStarted) {
				t.Fatalf("projection before start = %v, want ErrGameNotStarted", err)
			}
		}
	}
	seats := make(map[string]string)
	for _, uid := range append([]string{"alice"}, guests...) {
		snapshot, err := gameStore.Get(ctx, Actor{ID: uid}, created.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		house, ok := snapshot.PlayerFor(Actor{ID: uid})
		if !ok {
			t.Fatalf("%s holds no house", uid)
		}
		seats[uid] = string(house)
	}
	return seats
}

func TestIncognitoSeatsAreDrawnAtTheLastJoinWhateverTheJoinOrder(t *testing.T) {
	first := seatsAfterJoins(t, []string{"bob", "carol"})
	second := seatsAfterJoins(t, []string{"carol", "bob"})
	for uid, house := range first {
		if second[uid] != house {
			t.Fatalf("%s holds %s or %s depending on the join order", uid, house, second[uid])
		}
	}
	distinct := map[string]bool{}
	for _, house := range first {
		distinct[house] = true
	}
	if len(distinct) != 3 {
		t.Fatalf("seats = %v, want three distinct houses", first)
	}
}

func TestIncognitoSeatsDependOnTheSeed(t *testing.T) {
	members := []string{"alice", "bob", "carol", "dave", "erin", "frank"}
	slots := func(seed string) []PlayerSlot {
		players := make([]PlayerSlot, len(members))
		for index, uid := range members {
			players[index] = PlayerSlot{ID: models.PlayerID(rune('1' + index)), ActorID: uid}
		}
		SeatIncognitoPlayers(seed, players)
		return players
	}
	again := slots("one")
	for index, slot := range slots("one") {
		if slot.ActorID != again[index].ActorID {
			t.Fatal("the draw is not deterministic")
		}
	}
	differs := false
	for index, slot := range slots("two") {
		if slot.ActorID != again[index].ActorID {
			differs = true
		}
	}
	if !differs {
		t.Fatal("two seeds drew the same seats")
	}
}
