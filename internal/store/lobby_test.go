package store

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine"
)

func TestResolveSeedBorda(t *testing.T) {
	candidates := SeedCandidates("game", 3)
	cases := []struct {
		name     string
		rankings [][]int
		want     int
	}{
		{"no vote picks the first candidate", nil, 0},
		{"only non-voters", [][]int{nil, {}}, 0},
		{"unanimous", [][]int{{2, 0, 1}, {2, 1, 0}}, 2},
		{"points beat first places", [][]int{{0, 1, 2}, {1, 2, 0}, {1, 0, 2}}, 1},
		{"a single voter decides", [][]int{{1, 2, 0}}, 1},
	}
	for _, tc := range cases {
		if got := ResolveSeed("game", candidates, tc.rankings); got != candidates[tc.want] {
			t.Errorf("%s: seed = %s, want candidate %d", tc.name, got, tc.want)
		}
	}
}

func TestResolveSeedTieBreakIsDeterministicAndOrderIndependent(t *testing.T) {
	candidates := SeedCandidates("tie-game", 3)
	// Two opposed votes tie candidates 0 and 2 (3 points each).
	votes := [][]int{{0, 1, 2}, {2, 1, 0}}
	first := ResolveSeed("tie-game", candidates, votes)
	reversed := ResolveSeed("tie-game", candidates, [][]int{votes[1], votes[0]})
	if first != reversed {
		t.Fatalf("seed depends on the vote order: %s vs %s", first, reversed)
	}
	if first == candidates[1] {
		t.Fatal("the middle candidate cannot win a tie it is not part of")
	}
	// The tie is broken by the game id: other ids can pick the other candidate.
	picks := map[string]bool{}
	for _, id := range []GameID{"a", "b", "c", "d", "e", "f", "g", "h"} {
		picks[ResolveSeed(id, SeedCandidates(id, 3), votes)] = true
		if ResolveSeed(id, SeedCandidates(id, 3), votes) != ResolveSeed(id, SeedCandidates(id, 3), votes) {
			t.Fatal("tie-break is not deterministic")
		}
	}
	if len(picks) < 2 {
		t.Fatal("the tie-break never varies with the game id")
	}
}

func TestValidateSeedRanking(t *testing.T) {
	for _, ranking := range [][]int{nil, {}, {0, 1, 2}, {2, 0, 1}} {
		if err := ValidateSeedRanking(ranking, 3); err != nil {
			t.Errorf("%v rejected: %v", ranking, err)
		}
	}
	for _, ranking := range [][]int{{0}, {0, 1}, {0, 0, 1}, {0, 1, 3}, {-1, 0, 1}, {0, 1, 2, 3}} {
		if !errors.Is(ValidateSeedRanking(ranking, 3), ErrInvalidSeedRanking) {
			t.Errorf("%v accepted", ranking)
		}
	}
}

func newLobbyGame(t *testing.T, request CreateRequest) (*MemoryStore, GameSnapshot, string) {
	t.Helper()
	gameStore := newTestStore(t)
	ctx := context.Background()
	request.StrictMembership = true
	if request.Players == nil {
		request.Players = []engine.PlayerInit{{}, {}, {}}
	}
	created, err := gameStore.Create(ctx, Actor{ID: "alice"}, request)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	secret, err := gameStore.CreateInvitation(ctx, Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	return gameStore, created, secret.Code
}

func TestOnlineGameHasNoBoardUntilTheLastSlotIsTaken(t *testing.T) {
	gameStore, created, code := newLobbyGame(t, CreateRequest{})
	ctx := context.Background()
	alice := Actor{ID: "alice"}
	if created.Started() || created.State != nil || len(created.Map.Territories) != 0 || created.Seed != "" {
		t.Fatalf("lobby snapshot = started %v seed %q territories %d", created.Started(), created.Seed, len(created.Map.Territories))
	}
	if created.Lobby == nil || len(created.Lobby.SeedCandidates) != 3 {
		t.Fatalf("lobby view = %#v", created.Lobby)
	}
	if _, err := gameStore.Join(ctx, Actor{ID: "bob"}, created.ID, code); err != nil {
		t.Fatalf("join: %v", err)
	}
	for name, err := range map[string]error{
		"state":   func() error { _, e := gameStore.State(ctx, alice, created.ID); return e }(),
		"map":     func() error { _, e := gameStore.Map(ctx, alice, created.ID); return e }(),
		"supply":  func() error { _, e := gameStore.Supply(ctx, alice, created.ID, "T1", ""); return e }(),
		"submit":  func() error { _, e := gameStore.Submit(ctx, alice, created.ID, SubmitRequest{}); return e }(),
		"resolve": func() error { _, e := gameStore.Resolve(ctx, alice, created.ID); return e }(),
	} {
		if !errors.Is(err, ErrGameNotStarted) {
			t.Errorf("%s in the lobby = %v, want ErrGameNotStarted", name, err)
		}
	}
	listed, err := gameStore.List(ctx, alice)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %v, %v", listed, err)
	}
	joined, err := gameStore.Join(ctx, Actor{ID: "carol"}, created.ID, code)
	if err != nil {
		t.Fatalf("last join: %v", err)
	}
	if !joined.Snapshot.Started() || joined.Snapshot.Seed == "" || len(joined.Snapshot.Map.Territories) == 0 || joined.Snapshot.Lobby != nil {
		t.Fatalf("started snapshot = %#v", joined.Snapshot.Seed)
	}
	if _, err := gameStore.State(ctx, alice, created.ID); err != nil {
		t.Fatalf("state once started: %v", err)
	}
}

func TestSeedVoteDecidesTheGameSeedWhateverTheJoinOrder(t *testing.T) {
	play := func(order []string) GameSnapshot {
		gameStore, created, code := newLobbyGame(t, CreateRequest{})
		ctx := context.Background()
		// Ids differ between runs: compare candidate indexes, not seeds.
		votes := map[string][]int{"alice": {2, 0, 1}, "bob": {2, 1, 0}, "carol": nil}
		if _, err := gameStore.SetSeedVote(ctx, Actor{ID: "alice"}, created.ID, votes["alice"]); err != nil {
			t.Fatalf("vote: %v", err)
		}
		var last JoinResult
		for _, uid := range order {
			var err error
			if last, err = gameStore.Join(ctx, Actor{ID: uid}, created.ID, code); err != nil {
				t.Fatalf("join: %v", err)
			}
			if votes[uid] != nil && !last.Snapshot.Started() {
				if _, err := gameStore.SetSeedVote(ctx, Actor{ID: uid}, created.ID, votes[uid]); err != nil {
					t.Fatalf("vote: %v", err)
				}
			}
		}
		return last.Snapshot
	}
	first, second := play([]string{"bob", "carol"}), play([]string{"carol", "bob"})
	// bob votes after joining in the first run only, carol is a non-voter
	// (and the last joiner in one run): candidate 2 still wins both.
	for _, snapshot := range [...]GameSnapshot{first, second} {
		if want := SeedCandidates(snapshot.ID, 3)[2]; snapshot.Seed != want {
			t.Fatalf("seed = %s, want candidate 2 (%s)", snapshot.Seed, want)
		}
	}
}

func TestSeedVoteRules(t *testing.T) {
	gameStore, created, code := newLobbyGame(t, CreateRequest{})
	ctx := context.Background()
	alice := Actor{ID: "alice"}
	if _, err := gameStore.SetSeedVote(ctx, alice, created.ID, []int{0, 0, 1}); !errors.Is(err, ErrInvalidSeedRanking) {
		t.Fatalf("duplicate ranking = %v", err)
	}
	if _, err := gameStore.SetSeedVote(ctx, Actor{ID: "mallory"}, created.ID, []int{0, 1, 2}); !errors.Is(err, ErrNotMember) {
		t.Fatalf("stranger = %v", err)
	}
	snapshot, err := gameStore.SetSeedVote(ctx, alice, created.ID, []int{1, 0, 2})
	if err != nil || !slices.Equal(snapshot.Lobby.SeedRanking, []int{1, 0, 2}) {
		t.Fatalf("vote = %v, %v", snapshot.Lobby, err)
	}
	if _, err := gameStore.Join(ctx, Actor{ID: "bob"}, created.ID, code); err != nil {
		t.Fatal(err)
	}
	// Other members never see this vote, and the vote may change or be withdrawn.
	bobView, err := gameStore.Get(ctx, Actor{ID: "bob"}, created.ID)
	if err != nil || len(bobView.Lobby.SeedRanking) != 0 {
		t.Fatalf("bob sees %v, %v", bobView.Lobby, err)
	}
	snapshot, err = gameStore.SetSeedVote(ctx, alice, created.ID, nil)
	if err != nil || len(snapshot.Lobby.SeedRanking) != 0 {
		t.Fatalf("withdrawn vote = %v, %v", snapshot.Lobby, err)
	}
	if _, err := gameStore.Join(ctx, Actor{ID: "carol"}, created.ID, code); err != nil {
		t.Fatal(err)
	}
	if _, err := gameStore.SetSeedVote(ctx, alice, created.ID, []int{0, 1, 2}); !errors.Is(err, ErrGameStarted) {
		t.Fatalf("vote after start = %v", err)
	}
}

func TestFixedSeedGamesDoNotVote(t *testing.T) {
	gameStore, created, code := newLobbyGame(t, CreateRequest{Seed: "imposed"})
	ctx := context.Background()
	if len(created.Lobby.SeedCandidates) != 0 {
		t.Fatalf("candidates = %v", created.Lobby.SeedCandidates)
	}
	for _, uid := range []string{"bob", "carol"} {
		if _, err := gameStore.Join(ctx, Actor{ID: uid}, created.ID, code); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := gameStore.Get(ctx, Actor{ID: "alice"}, created.ID)
	if err != nil || snapshot.Seed != "imposed" {
		t.Fatalf("seed = %q, %v", snapshot.Seed, err)
	}
}

func TestLobbyHouseNamesAreSilentAndFollowTheMemberThroughTheDraw(t *testing.T) {
	gameStore, created, code := newLobbyGame(t, CreateRequest{Incognito: true, Seed: "draw"})
	ctx := context.Background()
	names := map[string]string{"alice": "House Stark", "bob": "House Tyrell", "carol": "House Arryn"}
	if _, err := gameStore.RenameHouse(ctx, Actor{ID: "alice"}, created.ID, names["alice"]); err != nil {
		t.Fatalf("lobby rename: %v", err)
	}
	if _, err := gameStore.Join(ctx, Actor{ID: "bob"}, created.ID, code); err != nil {
		t.Fatal(err)
	}
	if _, err := gameStore.RenameHouse(ctx, Actor{ID: "bob"}, created.ID, "house stark"); !errors.Is(err, ErrHouseNameTaken) {
		t.Fatalf("duplicate pending name = %v", err)
	}
	if _, err := gameStore.RenameHouse(ctx, Actor{ID: "bob"}, created.ID, names["bob"]); err != nil {
		t.Fatal(err)
	}
	// carol takes the last slot without entering a name: placeholder house.
	last, err := gameStore.Join(ctx, Actor{ID: "carol"}, created.ID, code)
	if err != nil {
		t.Fatal(err)
	}
	for _, uid := range []string{"alice", "bob"} {
		snapshot, err := gameStore.Get(ctx, Actor{ID: uid}, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		house, _ := snapshot.PlayerFor(Actor{ID: uid})
		if got := playerSlotForID(snapshot.Players, house).Name; got != names[uid] {
			t.Errorf("%s holds house %s named %q, want %q", uid, house, got, names[uid])
		}
		for _, player := range snapshot.State.Players {
			if player.ID == house && player.Name != names[uid] {
				t.Errorf("state name of %s = %q", house, player.Name)
			}
		}
	}
	if last.Player.Name != string(last.Player.ID) {
		t.Fatalf("a member without a name keeps the placeholder, got %q", last.Player.Name)
	}
}

func TestSeedCandidateMapRevealsOnlyTheMap(t *testing.T) {
	gameStore, created, _ := newLobbyGame(t, CreateRequest{})
	ctx := context.Background()
	mapData, err := gameStore.SeedCandidateMap(ctx, Actor{ID: "alice"}, created.ID, "", 1)
	if err != nil || len(mapData.Territories) == 0 {
		t.Fatalf("candidate map = %d territories, %v", len(mapData.Territories), err)
	}
	if _, err := gameStore.SeedCandidateMap(ctx, Actor{ID: "alice"}, created.ID, "", 3); !errors.Is(err, ErrInvalidCandidate) {
		t.Fatalf("unknown candidate = %v", err)
	}
	if _, err := gameStore.SeedCandidateMap(ctx, Actor{ID: "mallory"}, created.ID, "", 0); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("stranger = %v", err)
	}
}

func TestLastJoinerNameAndRankingCountBeforeTheGameStarts(t *testing.T) {
	gameStore, created, code := newLobbyGame(t, CreateRequest{})
	ctx := context.Background()
	if _, err := gameStore.SetSeedVote(ctx, Actor{ID: "alice"}, created.ID, []int{0, 1, 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := gameStore.JoinWith(ctx, Actor{ID: "bob"}, created.ID, code, JoinOptions{SeedRanking: []int{0, 1, 2}, HouseName: "House Tyrell"}); err != nil {
		t.Fatal(err)
	}
	// Alice and Bob prefer candidate 0, Carol's ranking tips the count to 2.
	last, err := gameStore.JoinWith(ctx, Actor{ID: "carol"}, created.ID, code, JoinOptions{SeedRanking: []int{2, 1, 0}, HouseName: "House Arryn"})
	if err != nil {
		t.Fatal(err)
	}
	if !last.Started {
		t.Fatal("the last join did not report the start")
	}
	// Borda: candidate 0 = 2+2+0, candidate 2 = 0+0+2... candidate 0 wins.
	if want := SeedCandidates(created.ID, 3)[0]; last.Snapshot.Seed != want {
		t.Fatalf("seed = %s, want candidate 0 %s", last.Snapshot.Seed, want)
	}
	if last.Player.Name != "House Arryn" {
		t.Fatalf("last joiner's house = %q", last.Player.Name)
	}
	// Carol's vote counted: without it, the same seed; flip it to prove it.
	gameStore2, created2, code2 := newLobbyGame(t, CreateRequest{})
	for _, vote := range []struct {
		uid     string
		ranking []int
	}{{"bob", []int{1, 0, 2}}, {"carol", []int{1, 2, 0}}} {
		if _, err := gameStore2.JoinWith(ctx, Actor{ID: vote.uid}, created2.ID, code2, JoinOptions{SeedRanking: vote.ranking}); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := gameStore2.Get(ctx, Actor{ID: "alice"}, created2.ID)
	if err != nil || snapshot.Seed != SeedCandidates(created2.ID, 3)[1] {
		t.Fatalf("two voters (the last joiner among them) pick candidate 1: seed %q, %v", snapshot.Seed, err)
	}
}

func TestJoinWithRejectsABadEntryWithoutTakingTheSlot(t *testing.T) {
	gameStore, created, code := newLobbyGame(t, CreateRequest{})
	ctx := context.Background()
	if _, err := gameStore.RenameHouse(ctx, Actor{ID: "alice"}, created.ID, "House Stark"); err != nil {
		t.Fatal(err)
	}
	bob := Actor{ID: "bob"}
	if _, err := gameStore.JoinWith(ctx, bob, created.ID, code, JoinOptions{SeedRanking: []int{0, 0, 1}}); !errors.Is(err, ErrInvalidSeedRanking) {
		t.Fatalf("bad ranking = %v", err)
	}
	if _, err := gameStore.JoinWith(ctx, bob, created.ID, code, JoinOptions{HouseName: "house stark"}); !errors.Is(err, ErrHouseNameTaken) {
		t.Fatalf("taken name = %v", err)
	}
	preview, err := gameStore.JoinPreview(ctx, bob, created.ID, code)
	if err != nil || preview.Seated != 1 || preview.Member {
		t.Fatalf("a rejected join took the slot: %#v, %v", preview, err)
	}
}

func TestInviteCodeOpensThePreviewAndTheCandidateMapsToNonMembers(t *testing.T) {
	gameStore, created, code := newLobbyGame(t, CreateRequest{})
	ctx := context.Background()
	stranger := Actor{ID: "dave"}
	preview, err := gameStore.JoinPreview(ctx, stranger, created.ID, code)
	if err != nil || preview.Started || preview.Member || preview.Players != 3 || len(preview.SeedCandidates) != 3 {
		t.Fatalf("preview = %#v, %v", preview, err)
	}
	if _, err := gameStore.JoinPreview(ctx, stranger, created.ID, "ZZZZZZ"); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("wrong code = %v", err)
	}
	if _, err := gameStore.JoinPreview(ctx, stranger, created.ID, ""); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("no code = %v", err)
	}
	mapData, err := gameStore.SeedCandidateMap(ctx, stranger, created.ID, code, 2)
	if err != nil || len(mapData.Territories) == 0 {
		t.Fatalf("map by invitation = %d, %v", len(mapData.Territories), err)
	}
	// A code of another game opens nothing.
	_, other, otherCode := newLobbyGame(t, CreateRequest{})
	_ = other
	if _, err := gameStore.JoinPreview(ctx, stranger, created.ID, otherCode); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("foreign code = %v", err)
	}
}
