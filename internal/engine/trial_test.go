package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestExecutionOpensRevoltOnTheRegionForTheNextActionSeason(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}}}
	state.Turn = 3
	state.Season = models.SeasonForTurn(3)
	state.TrialRevoltWindows = []models.TrialRevoltWindow{{RegionSeed: "AAA", FromTurn: 3}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C3", Kind: models.CardKindRevolt}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C3"}},
	}
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

func TestTrialVerdictOfTheDirectTrial(t *testing.T) {
	cases := map[string]struct {
		dignity models.Dignity
		sex     models.Sex
		married bool
		liable  bool
	}{
		"lady with a visible dignity": {models.DignityHerbalist, models.SexFemale, false, true},
		"lady with a hidden dignity":  {models.DignityWitch, models.SexFemale, false, true},
		"man":                         {models.DignityHerbalist, models.SexMale, false, false},
		"no dignity":                  {"", models.SexFemale, false, false},
		"abbess":                      {models.DignityAbbess, models.SexFemale, false, false},
		"married lady":                {models.DignityCastellan, models.SexFemale, true, false},
		"bastard (woman)":             {models.DignityBastard, models.SexFemale, false, false},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			state := testState(t, []models.Territory{territory("AAA", "AAA")}, nil)
			addNoble(state, "N1", "ELE", "P2", "AAA")
			state.Nobles[0].Sex = test.sex
			if test.dignity != "" {
				state.Nobles[0].Dignities = []models.Dignity{test.dignity}
			}
			if test.married {
				addNoble(state, "N2", "GUY", "P1", "AAA")
				state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N2"}}
			}
			dignity, liable := (&resolutionContext{state: state}).trialVerdict(&state.Nobles[0])
			if liable != test.liable || liable && dignity != test.dignity {
				t.Fatalf("trialVerdict = %q/%t, want %q/%t", dignity, liable, test.dignity, test.liable)
			}
		})
	}
}

func TestWinterTrialOfALadyRevealsHerDignityAndOpensRevolt(t *testing.T) {
	state := winterTrialState(t)
	state.Excommunications = nil
	for index := range state.Nobles {
		if state.Nobles[index].ID == "N2" {
			state.Nobles[index].Sex = models.SexFemale
			state.Nobles[index].Dignities = []models.Dignity{models.DignityWitch}
		}
	}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO"), trialBy("O2", "OTO", "LEO")},
	})
	trials := eventsOfType(resolution.Events, EventTypeTrial)
	if hasNoble(resolution.State, "N2") || len(trials) != 1 || trials[0].Reason != "trial_executed" || trials[0].Dignity != models.DignityWitch {
		t.Fatalf("nobles = %#v, trial events = %#v; want the witch executed and revealed", resolution.State.Nobles, trials)
	}
	windows := resolution.State.TrialRevoltWindows
	if len(windows) != 1 || windows[0].FromTurn != nextActionTurn(state.Turn) {
		t.Fatalf("revolt windows = %#v, want one opened from turn %d", windows, nextActionTurn(state.Turn))
	}
}
