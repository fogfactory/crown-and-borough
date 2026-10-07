package store

import (
	"context"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// trialGame is a real 3-player hotseat game, seeded with a lady of P2 carrying
// the given dignity and one trial card (plus an optional revolt card) in P1's
// hand, then driven only through Submit like the browser does.
type trialGame struct {
	t     *testing.T
	store *MemoryStore
	id    GameID
	where models.TerritoryID
}

func newTrialGame(t *testing.T, dignity models.Dignity, mutate func(*models.GameState)) *trialGame {
	t.Helper()
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "P1"}, CreateRequest{
		Seed:    "trial-hotseat",
		Players: []engine.PlayerInit{{Name: "One"}, {Name: "Two"}, {Name: "Three"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	game, err := gameStore.game(created.ID)
	if err != nil {
		t.Fatalf("lookup game: %v", err)
	}
	game.mu.Lock()
	defer game.mu.Unlock()
	state := game.state
	var where models.TerritoryID
	for _, noble := range state.Nobles {
		if noble.OwnerID == "P2" {
			where = noble.LocationID
			break
		}
	}
	lady := models.Noble{
		ID: "N900", Code: "ZZA", Name: "Zaza", Sex: models.SexFemale, OwnerID: "P2",
		LocationID: where, Status: models.NobleStatusFree, Dignities: []models.Dignity{dignity},
	}
	if dignity == models.DignityAbbess {
		lady.AbbeyRegion = state.Regions[0].Seed
	}
	state.Nobles = append(state.Nobles, lady)
	if state.SpecialDeck == nil {
		t.Fatal("no special deck")
	}
	for _, card := range []models.SpecialCard{{ID: "C901", Kind: models.CardKindTrial}, {ID: "C902", Kind: models.CardKindRevolt}} {
		state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, card)
	}
	state.SpecialDeck.Hands["P1"] = append(state.SpecialDeck.Hands["P1"], "C901")
	state.SpecialDeck.Hands["P2"] = append(state.SpecialDeck.Hands["P2"], "C902")
	if mutate != nil {
		mutate(state)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("seeded state invalid: %v", err)
	}
	return &trialGame{t: t, store: gameStore, id: created.ID, where: where}
}

// playTurn submits one special order per player ("" = none) and resolves.
func (g *trialGame) playTurn(orders map[models.PlayerID]string) (GameSnapshot, error) {
	g.t.Helper()
	var result SubmitResult
	for _, player := range []models.PlayerID{"P1", "P2", "P3"} {
		request := SubmitRequest{}
		if text := orders[player]; text != "" {
			request.Special = []engine.DeckSubmission{{Player: player, Text: text}}
		}
		var err error
		result, err = g.store.Submit(context.Background(), Actor{ID: string(player)}, g.id, request)
		if err != nil {
			return GameSnapshot{}, err
		}
	}
	if !result.Resolved {
		g.t.Fatalf("turn did not resolve: %#v", result.Status)
	}
	return result.Snapshot, nil
}

func hasNoble(state *models.GameState, id models.NobleID) bool {
	for _, noble := range state.Nobles {
		if noble.ID == id {
			return true
		}
	}
	return false
}

func trialEffect(t *testing.T, snapshot GameSnapshot) engine.SeasonEffectReport {
	t.Helper()
	reports := snapshot.Reports
	for _, effect := range reports[len(reports)-1].Report.SeasonEffects {
		if effect.Kind == engine.EventTypeTrial {
			return effect
		}
	}
	t.Fatalf("no trial effect in the last report: %#v", reports[len(reports)-1].Report.SeasonEffects)
	return engine.SeasonEffectReport{}
}

func TestHotseatTrialExecutesAnOpponentLadyThenRevoltIsAllowedNextTurn(t *testing.T) {
	g := newTrialGame(t, models.DignityHerbalist, nil)
	// Revolt is not allowed on the lady's region before she is executed.
	if _, err := g.playTurn(map[models.PlayerID]string{"P2": "P RE " + string(g.where)}); err == nil {
		t.Fatal("revolt without famine, tax or execution: want a rejection")
	}
	snapshot, err := g.playTurn(map[models.PlayerID]string{"P1": "P PR ZZA"})
	if err != nil {
		t.Fatalf("trial turn: %v", err)
	}
	if hasNoble(snapshot.State, "N900") {
		t.Fatal("the herbalist survived the trial")
	}
	effect := trialEffect(t, snapshot)
	if effect.Reason != "trial_executed" || effect.Dignity != models.DignityHerbalist || effect.Noble != "ZZA" {
		t.Fatalf("trial effect = %#v", effect)
	}
	var removed models.RemovedNoble
	for _, entry := range snapshot.State.RemovedNobles {
		if entry.ID == "N900" {
			removed = entry
		}
	}
	if removed.Cause != models.DeathCauseExecution {
		t.Fatalf("removed noble = %#v, want an execution", removed)
	}
	if got := snapshot.State.SpecialDeck.Hands["P1"]; containsCard(got, "C901") {
		t.Fatalf("P1 hand = %v, want the trial card spent", got)
	}
	// Turn 2: the window is open, P2 avenges with a revolt on the region.
	if _, err := g.playTurn(map[models.PlayerID]string{"P2": "P RE " + string(g.where)}); err != nil {
		t.Fatalf("revolt in the opened window: %v", err)
	}
}

func TestHotseatTrialRevealsAHiddenDignity(t *testing.T) {
	g := newTrialGame(t, models.DignityWitch, nil)
	snapshot, err := g.playTurn(map[models.PlayerID]string{"P1": "P PR ZZA"})
	if err != nil {
		t.Fatalf("trial turn: %v", err)
	}
	effect := trialEffect(t, snapshot)
	if hasNoble(snapshot.State, "N900") || effect.Reason != "trial_executed" || effect.Dignity != models.DignityWitch {
		t.Fatalf("witch still alive or not revealed: %#v", effect)
	}
}

func TestHotseatTrialIsUnfoundedOnProtectedTargets(t *testing.T) {
	cases := map[string]struct {
		dignity models.Dignity
		mutate  func(*models.GameState)
	}{
		"abbess": {models.DignityAbbess, nil},
		"married lady": {models.DignityCastellan, func(state *models.GameState) {
			state.Nobles = append(state.Nobles, models.Noble{
				ID: "N901", Code: "ZZB", Name: "Zozo", Sex: models.SexMale, OwnerID: "P3",
				LocationID: state.Nobles[len(state.Nobles)-1].LocationID, Status: models.NobleStatusFree,
			})
			state.Marriages = append(state.Marriages, models.Marriage{NobleA: "N900", NobleB: "N901", Turn: 1})
		}},
		"lady without dignity": {models.DignityBastard, nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			g := newTrialGame(t, tc.dignity, tc.mutate)
			snapshot, err := g.playTurn(map[models.PlayerID]string{"P1": "P PR ZZA"})
			if err != nil {
				t.Fatalf("trial turn: %v", err)
			}
			if !hasNoble(snapshot.State, "N900") {
				t.Fatal("a protected target was executed")
			}
			if effect := trialEffect(t, snapshot); effect.Reason != "trial_unfounded" {
				t.Fatalf("trial effect = %#v, want unfounded", effect)
			}
			if containsCard(snapshot.State.SpecialDeck.Hands["P1"], "C901") {
				t.Fatal("the trial card was not spent")
			}
			if len(snapshot.State.TrialRevoltWindows) != 0 {
				t.Fatalf("windows = %#v, want none", snapshot.State.TrialRevoltWindows)
			}
		})
	}
}

func TestHotseatTrialRejectsUnknownNoble(t *testing.T) {
	g := newTrialGame(t, models.DignityHerbalist, nil)
	_, err := g.store.Submit(context.Background(), Actor{ID: "P1"}, g.id, SubmitRequest{
		Special: []engine.DeckSubmission{{Player: "P1", Text: "P PR QQQ"}},
	})
	if err == nil || !strings.Contains(err.Error(), "QQQ") {
		t.Fatalf("unknown noble error = %v, want a parse error naming it", err)
	}
}

func TestHotseatTrialMayTargetOwnLady(t *testing.T) {
	g := newTrialGame(t, models.DignityAstrologer, func(state *models.GameState) {
		for index := range state.Nobles {
			if state.Nobles[index].ID == "N900" {
				state.Nobles[index].OwnerID = "P1"
			}
		}
	})
	snapshot, err := g.playTurn(map[models.PlayerID]string{"P1": "P PR ZZA"})
	if err != nil {
		t.Fatalf("trial turn: %v", err)
	}
	if hasNoble(snapshot.State, "N900") {
		t.Fatal("own lady was not executed")
	}
}

func TestHotseatTrialInAutumnOpensRevoltInSpring(t *testing.T) {
	g := newTrialGame(t, models.DignityHerbalist, func(state *models.GameState) {
		state.Turn = 3
		state.Season = models.SeasonForTurn(3)
	})
	snapshot, err := g.playTurn(map[models.PlayerID]string{"P1": "P PR ZZA"})
	if err != nil {
		t.Fatalf("autumn trial: %v", err)
	}
	windows := snapshot.State.TrialRevoltWindows
	if len(windows) != 1 || windows[0].FromTurn != 5 {
		t.Fatalf("windows = %#v, want one from turn 5 (spring, winter skipped)", windows)
	}
}

func containsCard(hand []models.SpecialCardID, id models.SpecialCardID) bool {
	for _, card := range hand {
		if card == id {
			return true
		}
	}
	return false
}
