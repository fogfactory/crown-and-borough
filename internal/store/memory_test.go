package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestMemoryStoreCreatesIndependentGamesAndFiltersList(t *testing.T) {
	gameStore := newTestStore(t)
	first, err := gameStore.Create(context.Background(), Actor{ID: "alice"}, CreateRequest{
		Name:    "Alice's game",
		Seed:    "same-seed",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create first game: %v", err)
	}
	second, err := gameStore.Create(context.Background(), Actor{ID: "bob"}, CreateRequest{
		Name:    "Bob's game",
		Seed:    "same-seed",
		Players: []engine.PlayerInit{{Name: "Bob"}, {Name: "Carol"}},
	})
	if err != nil {
		t.Fatalf("create second game: %v", err)
	}
	if first.ID == second.ID || !isUUIDv4(string(first.ID)) || !isUUIDv4(string(second.ID)) {
		t.Fatalf("game ids = %q and %q, want distinct UUID v4 values", first.ID, second.ID)
	}
	firstMap, err := gameStore.Map(context.Background(), Actor{ID: "alice"}, first.ID)
	if err != nil {
		t.Fatalf("first map: %v", err)
	}
	secondMap, err := gameStore.Map(context.Background(), Actor{ID: "bob"}, second.ID)
	if err != nil {
		t.Fatalf("second map: %v", err)
	}
	firstJSON, _ := json.Marshal(firstMap)
	secondJSON, _ := json.Marshal(secondMap)
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("same seed produced different maps")
	}

	aliceGames, err := gameStore.List(context.Background(), Actor{ID: "alice"})
	if err != nil {
		t.Fatalf("list for Alice: %v", err)
	}
	if len(aliceGames) != 1 || aliceGames[0].ID != first.ID {
		t.Fatalf("Alice games = %#v, want only %s", aliceGames, first.ID)
	}
	bobGames, err := gameStore.List(context.Background(), Actor{ID: "bob"})
	if err != nil {
		t.Fatalf("list for Bob: %v", err)
	}
	if len(bobGames) != 1 || bobGames[0].ID != second.ID {
		t.Fatalf("Bob games = %#v, want only %s", bobGames, second.ID)
	}

	firstResult, err := gameStore.Submit(context.Background(), Actor{ID: "alice"}, first.ID, SubmitRequest{})
	if err != nil {
		t.Fatalf("submit first game: %v", err)
	}
	if firstResult.Status != "pending" || firstResult.Snapshot.State.Turn != 1 {
		t.Fatalf("first submit = %#v, want pending turn 1", firstResult)
	}
	secondState, err := gameStore.State(context.Background(), Actor{ID: "bob"}, second.ID)
	if err != nil {
		t.Fatalf("second state: %v", err)
	}
	if secondState.State.Turn != 1 || len(secondState.Submissions) != 0 {
		t.Fatalf("second game changed after first submit: turn=%d submissions=%#v", secondState.State.Turn, secondState.Submissions)
	}
}

func TestMemoryStoreSubmissionReplacementAndAutomaticResolution(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "submission-test",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	first, err := gameStore.Submit(context.Background(), Actor{ID: "P1"}, created.ID, SubmitRequest{})
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if first.Status != "pending" || first.Snapshot.Revision != 2 || len(first.Snapshot.Submissions) != 1 {
		t.Fatalf("first submit = %#v, want pending revision 2", first)
	}
	second, err := gameStore.Submit(context.Background(), Actor{ID: "P1"}, created.ID, SubmitRequest{ExpectedRevision: 2})
	if err != nil {
		t.Fatalf("replacement: %v", err)
	}
	if second.Status != "pending" || second.Snapshot.Revision != 3 || len(second.Snapshot.Submissions) != 1 {
		t.Fatalf("replacement = %#v, want pending revision 3", second)
	}

	resolved, err := gameStore.Submit(context.Background(), Actor{ID: "P2"}, created.ID, SubmitRequest{ExpectedRevision: 3})
	if err != nil {
		t.Fatalf("last submit: %v", err)
	}
	if resolved.Status != "resolved" || !resolved.Resolved || resolved.Snapshot.State.Turn != 2 {
		t.Fatalf("resolved = %#v, want resolved turn 2", resolved)
	}
	if resolved.Snapshot.Revision != 4 || len(resolved.Snapshot.Reports) != 1 || len(resolved.Snapshot.Submissions) != 0 {
		t.Fatalf("resolved revision/reports/submissions = %d/%d/%d", resolved.Snapshot.Revision, len(resolved.Snapshot.Reports), len(resolved.Snapshot.Submissions))
	}

	if _, err := gameStore.Submit(context.Background(), Actor{ID: "P1"}, created.ID, SubmitRequest{ExpectedRevision: 2}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale submission error = %v, want revision conflict", err)
	}
}

func TestMemoryStoreRejectsInvalidSubmissionAtomically(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "invalid-submission",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = gameStore.Submit(context.Background(), Actor{ID: "P1"}, created.ID, SubmitRequest{
		Chains: []engine.ChainSubmission{{Text: "BAD EXTRA"}},
	})
	var inputErrors *engine.InputErrors
	if !errors.As(err, &inputErrors) {
		t.Fatalf("invalid submit error = %v, want InputErrors", err)
	}
	state, err := gameStore.State(context.Background(), Actor{ID: "P1"}, created.ID)
	if err != nil {
		t.Fatalf("state after invalid submit: %v", err)
	}
	if state.Revision != 1 || len(state.Submissions) != 0 || state.State.Turn != 1 {
		t.Fatalf("state changed after invalid submission: revision=%d submissions=%d turn=%d", state.Revision, len(state.Submissions), state.State.Turn)
	}
}

func TestMemoryStoreForcedResolutionAndSeasonCycle(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "season-test",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	forced, err := gameStore.Resolve(context.Background(), Actor{ID: "P1"}, created.ID)
	if err != nil {
		t.Fatalf("forced resolve: %v", err)
	}
	if forced.Status != "resolved" || !forced.Forced || forced.Snapshot.State.Turn != 2 {
		t.Fatalf("forced result = %#v", forced)
	}

	wantSeasons := []models.Season{models.SeasonSummer, models.SeasonAutumn, models.SeasonWinter, models.SeasonSpring}
	for index, wantSeason := range wantSeasons {
		p1, err := gameStore.Submit(context.Background(), Actor{ID: "P1"}, created.ID, SubmitRequest{})
		if err != nil {
			t.Fatalf("turn %d P1 submit: %v", index+2, err)
		}
		if p1.Status != "pending" {
			t.Fatalf("turn %d P1 status = %q, want pending", index+2, p1.Status)
		}
		p2, err := gameStore.Submit(context.Background(), Actor{ID: "P2"}, created.ID, SubmitRequest{})
		if err != nil {
			t.Fatalf("turn %d P2 submit: %v", index+2, err)
		}
		if p2.Report == nil || p2.Report.Report.Header.Season != wantSeason {
			t.Fatalf("turn %d report = %#v, want season %s", index+2, p2.Report, wantSeason)
		}
	}
}

func TestMemoryStoreRejectsForcedSubmissionByNonCreator(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "forced-submission-owner",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := gameStore.Submit(context.Background(), Actor{ID: "P2"}, created.ID, SubmitRequest{Force: true}); !errors.Is(err, ErrNotCreator) {
		t.Fatalf("forced submission error = %v, want not creator", err)
	}
	current, err := gameStore.Get(context.Background(), Actor{ID: "P1"}, created.ID)
	if err != nil {
		t.Fatalf("read after rejected submission: %v", err)
	}
	if current.Revision != created.Revision || len(current.Submissions) != 0 {
		t.Fatalf("state changed after rejected submission: revision=%d submissions=%d", current.Revision, len(current.Submissions))
	}
}

func TestMemoryStoreConcurrentSubmissionsResolveOnce(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "concurrent-submission",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var wait sync.WaitGroup
	results := make(chan SubmitResult, 2)
	errorsCh := make(chan error, 2)
	for _, actor := range []string{"P1", "P2"} {
		actor := actor
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, submitErr := gameStore.Submit(context.Background(), Actor{ID: actor}, created.ID, SubmitRequest{})
			if submitErr != nil {
				errorsCh <- submitErr
				return
			}
			results <- result
		}()
	}
	wait.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent submit: %v", err)
	}
	count := 0
	for range results {
		count++
	}
	if count != 2 {
		t.Fatalf("successful submissions = %d, want 2", count)
	}
	final, err := gameStore.State(context.Background(), Actor{ID: "P1"}, created.ID)
	if err != nil {
		t.Fatalf("final state: %v", err)
	}
	if final.State.Turn != 2 || len(final.Reports) != 1 || final.Revision != 3 {
		t.Fatalf("final state = turn %d reports %d revision %d, want 2/1/3", final.State.Turn, len(final.Reports), final.Revision)
	}
}

func TestMemoryStoreTreatsEliminatedPlayersAsSubmittedAndSetsWinner(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "elimination-test",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	game, err := gameStore.game(created.ID)
	if err != nil {
		t.Fatalf("lookup game: %v", err)
	}
	game.mu.Lock()
	var p2Capital models.InfraID
	for index := range game.state.Players {
		if game.state.Players[index].ID != "P2" {
			continue
		}
		if game.state.Players[index].CapitalCastleID != nil {
			p2Capital = *game.state.Players[index].CapitalCastleID
		}
		game.state.Players[index].CapitalCastleID = nil
	}
	if p2Capital == "" {
		game.mu.Unlock()
		t.Fatal("P2 has no starting capital")
	}
	for territoryID, territory := range game.state.TerritoryStates {
		if territory.OwnerID != nil && *territory.OwnerID == "P2" {
			territory.OwnerID = nil
			territory.Army = nil
			game.state.TerritoryStates[territoryID] = territory
		}
	}
	armies := game.state.Armies[:0]
	for _, army := range game.state.Armies {
		if army.OwnerID != "P2" {
			armies = append(armies, army)
		}
	}
	game.state.Armies = armies
	if err := game.state.Validate(); err != nil {
		game.mu.Unlock()
		t.Fatalf("eliminated fixture is invalid: %v", err)
	}
	game.mu.Unlock()

	result, err := gameStore.Submit(context.Background(), Actor{ID: "P1"}, created.ID, SubmitRequest{})
	if err != nil {
		t.Fatalf("P1 submit: %v", err)
	}
	if result.Status != "resolved" || result.Snapshot.Status != StatusFinished || result.Snapshot.Winner == nil || *result.Snapshot.Winner != "P1" {
		t.Fatalf("elimination result = status %q/%q winner %v, want resolved/finished/P1", result.Status, result.Snapshot.Status, result.Snapshot.Winner)
	}
}

func TestMemoryStoreRevisionGuardsConcurrentForcedResolutions(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "forced-race",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var wait sync.WaitGroup
	errorsCh := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, resolveErr := gameStore.ResolveAt(context.Background(), Actor{ID: "P1"}, created.ID, created.Revision)
			errorsCh <- resolveErr
		}()
	}
	wait.Wait()
	close(errorsCh)
	successes := 0
	conflicts := 0
	for resolveErr := range errorsCh {
		switch {
		case resolveErr == nil:
			successes++
		case errors.Is(resolveErr, ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("forced resolution error = %v", resolveErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("forced resolution outcomes = successes %d conflicts %d, want 1/1", successes, conflicts)
	}
	final, err := gameStore.State(context.Background(), Actor{ID: "P1"}, created.ID)
	if err != nil {
		t.Fatalf("final state: %v", err)
	}
	if final.State.Turn != 2 || len(final.Reports) != 1 {
		t.Fatalf("final forced state = turn %d reports %d, want 2/1", final.State.Turn, len(final.Reports))
	}
}

func newTestStore(t *testing.T) *MemoryStore {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	return NewMemoryStore(balance, assets)
}

func isUUIDv4(value string) bool {
	return len(value) == 36 && value[14] == '4' && (value[19] == '8' || value[19] == '9' || value[19] == 'a' || value[19] == 'b')
}
