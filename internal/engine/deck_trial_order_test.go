package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func trialTestState(t *testing.T, dignity models.Dignity, sex models.Sex) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}}}
	addNoble(state, "N1", "ELE", "P2", "BBB")
	state.Nobles[0].Sex = sex
	if dignity != "" {
		state.Nobles[0].Dignities = []models.Dignity{dignity}
	}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindTrial}, {ID: "C2", Kind: models.CardKindTrial}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1", "C2"}},
	}
	return state
}

func playTrial() map[models.PlayerID][]models.DeckOrder {
	return map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindTrial, TargetNobleID: "N1"}},
	}
}

func TestTrialExecutesAnUnmarriedLadyWithAVisibleDignityAndOpensRevolt(t *testing.T) {
	state := trialTestState(t, models.DignityHerbalist, models.SexFemale)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playTrial())
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if len(resolution.State.Nobles) != 0 {
		t.Fatalf("nobles = %#v, want the lady executed", resolution.State.Nobles)
	}
	removed := resolution.State.RemovedNobles
	if len(removed) != 1 || removed[0].ID != "N1" || removed[0].Cause != models.DeathCauseExecution {
		t.Fatalf("removed nobles = %#v, want an execution", removed)
	}
	windows := resolution.State.TrialRevoltWindows
	if len(windows) != 1 || windows[0].RegionSeed != "AAA" || windows[0].FromTurn != state.Turn+1 {
		t.Fatalf("revolt windows = %#v, want the region opened from the next turn", windows)
	}
	trials := eventsOfType(resolution.Events, EventTypeTrial)
	if len(trials) != 1 || trials[0].Reason != "trial_executed" {
		t.Fatalf("trial events = %#v", trials)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 1 {
		t.Fatalf("P1 hand = %#v, want the played card consumed", got)
	}
}

func TestTrialRevealsAHiddenDignityAndExecutes(t *testing.T) {
	state := trialTestState(t, models.DignityWitch, models.SexFemale)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playTrial())
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	trials := eventsOfType(resolution.Events, EventTypeTrial)
	if len(resolution.State.Nobles) != 0 || len(trials) != 1 || trials[0].Reason != "trial_executed" || trials[0].Dignity != models.DignityWitch {
		t.Fatalf("nobles = %#v, trial events = %#v; want the witch executed and revealed", resolution.State.Nobles, trials)
	}
}

func TestTrialIsUnfoundedOnIneligibleTargets(t *testing.T) {
	cases := map[string]struct {
		dignity models.Dignity
		sex     models.Sex
		married bool
	}{
		"man":             {models.DignityHerbalist, models.SexMale, false},
		"no dignity":      {"", models.SexFemale, false},
		"abbess":          {models.DignityAbbess, models.SexFemale, false},
		"married lady":    {models.DignityCastellan, models.SexFemale, true},
		"bastard (woman)": {models.DignityBastard, models.SexFemale, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			state := trialTestState(t, tc.dignity, tc.sex)
			if tc.married {
				addNoble(state, "N2", "GUY", "P1", "BBB")
				state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N2"}}
			}
			validateTestState(t, state)
			resolution, err := ResolveWithDeckOrders(state, testBalance(), playTrial())
			if err != nil {
				t.Fatalf("ResolveWithDeckOrders: %v", err)
			}
			if len(resolution.State.Nobles) != len(state.Nobles) || len(resolution.State.TrialRevoltWindows) != 0 {
				t.Fatalf("state = %#v, want nobody executed and no revolt window", resolution.State.Nobles)
			}
			trials := eventsOfType(resolution.Events, EventTypeTrial)
			if len(trials) != 1 || trials[0].Reason != "trial_unfounded" {
				t.Fatalf("trial events = %#v", trials)
			}
			if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 1 {
				t.Fatalf("P1 hand = %#v, want the card spent", got)
			}
		})
	}
}

func TestTwoTrialsOnTheSameLadyExecuteOnlyOnce(t *testing.T) {
	state := trialTestState(t, models.DignityAstrologer, models.SexFemale)
	validateTestState(t, state)
	deckOrders := playTrial()
	deckOrders["P1"] = append(deckOrders["P1"], models.DeckOrder{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindTrial, TargetNobleID: "N1"})
	resolution, err := ResolveWithDeckOrders(state, testBalance(), deckOrders)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if len(resolution.State.RemovedNobles) != 1 || len(resolution.State.TrialRevoltWindows) != 1 {
		t.Fatalf("removed = %#v windows = %#v", resolution.State.RemovedNobles, resolution.State.TrialRevoltWindows)
	}
}

func TestTrialIsRejectedInWinter(t *testing.T) {
	state := trialTestState(t, models.DignityHerbalist, models.SexFemale)
	definition := trialCardDefinition{}
	ctx := newResolutionContext(state, testBalance())
	order := models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: models.CardKindTrial, TargetNobleID: "N1"}
	if ok, reason := definition.CanPlay(&ExecutionContext{resolution: ctx, playerID: "P1", season: models.SeasonWinter}, order); ok || reason != "deck_order_out_of_season" {
		t.Fatalf("CanPlay in winter = %v %q", ok, reason)
	}
	order.TargetNobleID = "NOPE"
	if ok, reason := definition.CanPlay(&ExecutionContext{resolution: ctx, playerID: "P1", season: models.SeasonSpring}, order); ok || reason != "trial_requires_noble" {
		t.Fatalf("CanPlay on unknown noble = %v %q", ok, reason)
	}
}

func TestExecutionOpensRevoltOnTheRegionForTheNextActionSeason(t *testing.T) {
	state := trialTestState(t, models.DignityHerbalist, models.SexFemale)
	state.Turn = 3
	state.Season = models.SeasonForTurn(3)
	state.TrialRevoltWindows = []models.TrialRevoltWindow{{RegionSeed: "AAA", FromTurn: 3}}
	state.SpecialDeck.Cards = []models.SpecialCard{{ID: "C3", Kind: models.CardKindRevolt}}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C3"}
	validateTestState(t, state)
	_, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "BBB"}},
	})
	if err != nil {
		t.Fatalf("revolt on the region of a recent execution: %v", err)
	}
	state.TrialRevoltWindows = []models.TrialRevoltWindow{{RegionSeed: "AAA", FromTurn: 4}}
	if _, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "BBB"}},
	}); err == nil {
		t.Fatal("revolt before the window opens: want a rejection")
	}
}

func TestNextActionTurnSkipsWinter(t *testing.T) {
	for turn, want := range map[int]int{1: 2, 2: 3, 3: 5, 4: 5, 7: 9} {
		if got := nextActionTurn(turn); got != want {
			t.Errorf("nextActionTurn(%d) = %d, want %d", turn, got, want)
		}
	}
}

func TestParseTrialOrder(t *testing.T) {
	state := trialTestState(t, models.DignityHerbalist, models.SexFemale)
	for _, line := range []string{"P PR ELE", "P TR ELE"} {
		parsed, parseErrors := orders.ParseDeckOrders(line, state)
		if len(parseErrors) != 0 || len(parsed) != 1 || parsed[0].Kind != models.CardKindTrial || parsed[0].TargetNobleID != "N1" {
			t.Fatalf("%q = %#v %#v", line, parsed, parseErrors)
		}
	}
	if _, parseErrors := orders.ParseDeckOrders("P PR ZZZ", state); len(parseErrors) == 0 {
		t.Fatal("unknown noble code: want a parse error")
	}
}
