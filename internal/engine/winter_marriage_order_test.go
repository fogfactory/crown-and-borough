package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// marriageTestState has a man HUG (P1) and a woman ANN (P2), plus a woman
// EVE (P3) and a second man LEO (P2), all free.
func marriageTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := winterTestState(t, []models.Territory{
		territory("AAA", "AAA", "BBB"),
		territory("BBB", "BBB", "AAA"),
	}, nil)
	addNoble(state, "N1", "HUG", "P1", "AAA")
	addNoble(state, "N2", "ANN", "P2", "BBB")
	addNoble(state, "N3", "EVE", "P3", "BBB")
	addNoble(state, "N4", "LEO", "P2", "BBB")
	for _, id := range []models.NobleID{"N2", "N3"} {
		for index := range state.Nobles {
			if state.Nobles[index].ID == id {
				state.Nobles[index].Sex = models.SexFemale
			}
		}
	}
	return state
}

func marriageSheetOrder(id models.OrderID, noble, spouse models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeMarriage, NobleCode: noble, SpouseCode: spouse}
}

func resolveMarriageSheets(t *testing.T, state *models.GameState, sheets map[models.PlayerID][]models.WinterOrder) Resolution {
	t.Helper()
	resolution, err := ResolveWinter(state, testBalance(), sheets)
	if err != nil {
		t.Fatalf("ResolveWinter() = %v", err)
	}
	return resolution
}

func TestMarriageConcludedWhenReciprocal(t *testing.T) {
	state := marriageTestState(t)
	resolution := resolveMarriageSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {marriageSheetOrder("O1", "HUG", "ANN")},
		"P2": {marriageSheetOrder("O1", "ANN", "HUG")},
	})
	if len(resolution.State.Marriages) != 1 {
		t.Fatalf("marriages = %#v, want one", resolution.State.Marriages)
	}
	marriage := resolution.State.Marriages[0]
	if marriage.NobleA != "N1" || marriage.NobleB != "N2" || marriage.Turn != state.Turn {
		t.Errorf("marriage = %#v, want N1/N2 at turn %d", marriage, state.Turn)
	}
	events := eventsOfType(resolution.Events, EventTypeMarriage)
	if len(events) != 1 || events[0].SpouseNobleCode != "ANN" || events[0].SpouseOwnerID != "P2" || events[0].OwnerID != "P1" {
		t.Fatalf("marriage events = %#v", events)
	}
	if rejected := eventsOfType(resolution.Events, EventTypeRejected); len(rejected) != 0 {
		t.Errorf("rejections = %#v, want none", rejected)
	}
	if len(state.Marriages) != 0 {
		t.Errorf("input state mutated: %#v", state.Marriages)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	if len(report.Marriages) != 1 || report.Marriages[0].NobleCode != "HUG" || report.Marriages[0].SpouseCode != "ANN" || report.Marriages[0].Outcome != OutcomeSuccess {
		t.Errorf("report marriages = %#v", report.Marriages)
	}
}

func TestMarriageRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*models.GameState)
		sheets map[models.PlayerID][]models.WinterOrder
		player models.PlayerID
		reason string
	}{
		{
			name:   "not reciprocated",
			sheets: map[models.PlayerID][]models.WinterOrder{"P1": {marriageSheetOrder("O1", "HUG", "ANN")}},
			player: "P1", reason: "marriage_not_reciprocated",
		},
		{
			name: "mismatched partner",
			sheets: map[models.PlayerID][]models.WinterOrder{
				"P1": {marriageSheetOrder("O1", "HUG", "ANN")},
				"P3": {marriageSheetOrder("O1", "EVE", "HUG")},
			},
			player: "P1", reason: "marriage_not_reciprocated",
		},
		{
			name: "not own noble",
			sheets: map[models.PlayerID][]models.WinterOrder{
				"P1": {marriageSheetOrder("O1", "ANN", "HUG")},
			},
			player: "P1", reason: "noble_not_owned",
		},
		{
			name: "same owner",
			sheets: map[models.PlayerID][]models.WinterOrder{
				"P2": {marriageSheetOrder("O1", "ANN", "LEO")},
			},
			player: "P2", reason: "marriage_same_owner",
		},
		{
			name: "same sex",
			sheets: map[models.PlayerID][]models.WinterOrder{
				"P1": {marriageSheetOrder("O1", "HUG", "LEO")},
				"P2": {marriageSheetOrder("O1", "LEO", "HUG")},
			},
			player: "P1", reason: "marriage_same_sex",
		},
		{
			name:   "hostage",
			mutate: func(state *models.GameState) { setNobleStatus(state, "N2", models.NobleStatusHostage) },
			sheets: map[models.PlayerID][]models.WinterOrder{
				"P1": {marriageSheetOrder("O1", "HUG", "ANN")},
				"P2": {marriageSheetOrder("O1", "ANN", "HUG")},
			},
			player: "P1", reason: "noble_not_free",
		},
		{
			name: "already married",
			mutate: func(state *models.GameState) {
				state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
			},
			sheets: map[models.PlayerID][]models.WinterOrder{
				"P1": {marriageSheetOrder("O1", "HUG", "ANN")},
				"P2": {marriageSheetOrder("O1", "ANN", "HUG")},
			},
			player: "P1", reason: "noble_already_married",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := marriageTestState(t)
			if tc.mutate != nil {
				tc.mutate(state)
			}
			resolution := resolveMarriageSheets(t, state, tc.sheets)
			if len(resolution.State.Marriages) != len(state.Marriages) {
				t.Fatalf("marriages = %#v, want unchanged %#v", resolution.State.Marriages, state.Marriages)
			}
			var found bool
			for _, event := range eventsOfType(resolution.Events, EventTypeRejected) {
				if event.OwnerID == tc.player && event.Reason == tc.reason {
					found = true
				}
			}
			if !found {
				t.Errorf("events = %#v, want %s rejected %q", resolution.Events, tc.player, tc.reason)
			}
		})
	}
}

func TestMarriageNotReciprocatedIsRefusedForTheOtherPlayer(t *testing.T) {
	state := marriageTestState(t)
	resolution := resolveMarriageSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {marriageSheetOrder("O1", "HUG", "ANN"), marriageSheetOrder("O2", "HUG", "ANN")},
	})
	refused := eventsOfType(resolution.Events, EventTypeMarriageRefused)
	if len(refused) != 1 || refused[0].OwnerID != "P2" || refused[0].NobleCode != "ANN" ||
		refused[0].SpouseNobleCode != "HUG" || refused[0].SpouseOwnerID != "P1" {
		t.Fatalf("refusals = %#v, want one for P2 (ANN) about HUG", refused)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	if len(report.Marriages) != 1 || report.Marriages[0].Outcome != OutcomeFailure ||
		report.Marriages[0].NobleCode != "HUG" || report.Marriages[0].SpouseCode != "ANN" {
		t.Fatalf("report marriages = %#v, want one failed HUG/ANN negotiation", report.Marriages)
	}
	var seen bool
	for _, investment := range report.Winter.Investments {
		if investment.Kind == EventTypeMarriageRefused && investment.Player == "P2" && investment.Reason == "marriage_refused" {
			seen = true
		}
	}
	if !seen {
		t.Errorf("winter investments = %#v, want a marriage_refused entry for P2", report.Winter.Investments)
	}
}

func TestMarriageFirstReciprocalPairWinsForSharedNoble(t *testing.T) {
	state := marriageTestState(t)
	resolution := resolveMarriageSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {marriageSheetOrder("O1", "HUG", "ANN"), marriageSheetOrder("O2", "HUG", "EVE")},
		"P2": {marriageSheetOrder("O1", "ANN", "HUG")},
		"P3": {marriageSheetOrder("O1", "EVE", "HUG")},
	})
	if len(resolution.State.Marriages) != 1 || resolution.State.Marriages[0].NobleB != "N2" {
		t.Fatalf("marriages = %#v, want only HUG/ANN", resolution.State.Marriages)
	}
	reasons := map[string]int{}
	for _, event := range eventsOfType(resolution.Events, EventTypeRejected) {
		reasons[event.Reason]++
	}
	if reasons["noble_already_married"] != 2 {
		t.Errorf("rejections = %v, want the two losing orders rejected as already married", reasons)
	}
}

func TestValidateRejectsInvalidMarriages(t *testing.T) {
	cases := map[string][]models.Marriage{
		"same sex":    {{NobleA: "N1", NobleB: "N4", Turn: 1}},
		"same owner":  {{NobleA: "N2", NobleB: "N4", Turn: 1}},
		"twice":       {{NobleA: "N1", NobleB: "N2", Turn: 1}, {NobleA: "N1", NobleB: "N3", Turn: 1}},
		"unknown":     {{NobleA: "N1", NobleB: "N99", Turn: 1}},
		"future turn": {{NobleA: "N1", NobleB: "N2", Turn: 99}},
	}
	for name, marriages := range cases {
		t.Run(name, func(t *testing.T) {
			state := marriageTestState(t)
			state.Marriages = marriages
			if err := state.Validate(); err == nil {
				t.Errorf("Validate() = nil, want an error for marriages %#v", marriages)
			}
		})
	}
}
