package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestComputeScoresCountsCategoriesAndCaptiveHolder(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Players: []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
			{ID: "BBB"},
			{ID: "CCC"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			// A captive noble's point goes to whoever physically holds it, the
			// army stationed on its territory (#215), so AAA and CCC each carry
			// the army that will hold N2 and N3 below.
			"AAA": {Resources: 3, Infrastructures: infraPointer("I1"), Army: armyPointer("A1")},
			"BBB": {Resources: 2, Infrastructures: infraPointer("I2"), Army: armyPointer("A3")},
			"CCC": {Resources: 1, Infrastructures: infraPointer("I3"), Army: armyPointer("A2")},
		},
		Infrastructures: []models.Infrastructure{
			{ID: "I1", Type: models.InfraTypeCastle, TerritoryID: "AAA"},
			{ID: "I2", Type: models.InfraTypeMill, TerritoryID: "BBB"},
			{ID: "I3", Type: models.InfraTypeVillage, TerritoryID: "CCC"},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 4},
			{ID: "A2", OwnerID: p2, TerritoryID: "CCC", Size: 2},
			{ID: "A3", OwnerID: p1, TerritoryID: "BBB", Size: 1},
		},
		Nobles: []models.Noble{
			{ID: "N1", OwnerID: p1, LocationID: "AAA", Status: models.NobleStatusFree},
			{ID: "N2", OwnerID: p2, LocationID: "AAA", Status: models.NobleStatusHostage},
			{ID: "N3", OwnerID: p1, LocationID: "CCC", Status: models.NobleStatusDungeon},
			{ID: "N4", OwnerID: p2, LocationID: "CCC", Status: models.NobleStatusFree},
		},
	}

	scores := ComputeScores(state)
	if got, want := scores[p1], (ScoreBreakdown{Territories: 2, Castles: 5, Mills: 1, Nobles: 4, Troops: 5, Resources: 5, Total: 22}); got != want {
		t.Fatalf("P1 score = %#v, want %#v", got, want)
	}
	if got, want := scores[p2], (ScoreBreakdown{Territories: 1, Villages: 2, Nobles: 4, Troops: 2, Resources: 1, Total: 10}); got != want {
		t.Fatalf("P2 score = %#v, want %#v", got, want)
	}
}

// TestComputeScoresFortifiedVillageScoresAsVillage locks in #193: a fortified
// village still scores Villages += 2, not Castles += 5, since it keeps its
// InfraType and is never a distinct infrastructure.
func TestComputeScoresFortifiedVillageScoresAsVillage(t *testing.T) {
	p1 := models.PlayerID("P1")
	state := &models.GameState{
		Players:     []models.Player{{ID: p1}},
		Territories: []models.Territory{{ID: "AAA"}},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {Infrastructures: infraPointer("I1"), Army: armyPointer("A1")},
		},
		Armies: []models.Army{{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 1}},
		Infrastructures: []models.Infrastructure{
			{ID: "I1", Type: models.InfraTypeVillage, TerritoryID: "AAA", Fortified: true},
		},
	}

	scores := ComputeScores(state)
	if got, want := scores[p1], (ScoreBreakdown{Territories: 1, Villages: 2, Troops: 1, Total: 4}); got != want {
		t.Fatalf("P1 score = %#v, want %#v", got, want)
	}
}

// TestComputeScoresCaptiveGoesToHolderNotController checks #215: a hostage
// or dungeon noble's point goes to whoever physically holds it (the army
// stationed on its territory), not to that territory's controller, since
// control outside a fief is now ephemeral and can lapse -- or belong to a
// third party via a fief -- while the captor's army still stands there.
func TestComputeScoresCaptiveGoesToHolderNotController(t *testing.T) {
	p1, p2, p3 := models.PlayerID("P1"), models.PlayerID("P2"), models.PlayerID("P3")
	state := &models.GameState{
		Players:     []models.Player{{ID: p1}, {ID: p2}, {ID: p3}},
		Territories: []models.Territory{{ID: "AAA"}},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			// AAA is controlled by P1 (e.g. a fief member) but P2's army is the
			// one physically standing on it, holding N1 hostage.
			"AAA": {Army: armyPointer("A1")},
		},
		Fiefs: []models.Fief{{
			ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
			Territories: []models.TerritoryID{"AAA"}, OwnerID: p1,
		}},
		Armies: []models.Army{{ID: "A1", OwnerID: p2, TerritoryID: "AAA", Size: 1}},
		Nobles: []models.Noble{{ID: "N1", OwnerID: p3, LocationID: "AAA", Status: models.NobleStatusHostage}},
	}

	scores := ComputeScores(state)
	if got := scores[p2].Nobles; got != 2 {
		t.Errorf("P2 nobles score = %d, want 2 (the holder, not AAA's controller)", got)
	}
	if got := scores[p1].Nobles; got != 0 {
		t.Errorf("P1 nobles score = %d, want 0 (AAA's controller does not hold N1)", got)
	}
}

// TestComputeScoresCaptiveFallsBackToAnchorOwnerWithoutArmy verifies that a
// hostage or dungeon noble left on anchored ground (a fief member or a
// player's own capital) with no army present still scores for that anchor's
// owner, instead of being dropped for lack of a physical holder (#215): an
// anchored territory stays controlled indefinitely without an army, so a
// captive left behind there is not stranded on nobody's land.
func TestComputeScoresCaptiveFallsBackToAnchorOwnerWithoutArmy(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Players:     []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{{ID: "AAA"}},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			// AAA is a fief member of P1 with no army present.
			"AAA": {},
		},
		Fiefs: []models.Fief{{
			ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
			Territories: []models.TerritoryID{"AAA"}, OwnerID: p1,
		}},
		Nobles: []models.Noble{{ID: "N1", OwnerID: p2, LocationID: "AAA", Status: models.NobleStatusHostage}},
	}

	scores := ComputeScores(state)
	if got := scores[p1].Nobles; got != 2 {
		t.Errorf("P1 nobles score = %d, want 2 (falls back to the anchor owner)", got)
	}
	if got := scores[p2].Nobles; got != 0 {
		t.Errorf("P2 nobles score = %d, want 0 (owner is not the holder)", got)
	}
}

// TestComputeScoresFiefs verifies that each fief contributes exactly 1 point
// to its owner, vacant or not, until it is dissolved (titres.md).
func TestComputeScoresFiefs(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Players:         []models.Player{{ID: p1}, {ID: p2}},
		Territories:     []models.Territory{{ID: "AAA"}, {ID: "BBB"}, {ID: "CCC"}},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{},
		Fiefs: []models.Fief{
			{ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: p1},
			{ID: "F2", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: p1},
		},
	}
	scores := ComputeScores(state)
	if got := scores[p1].Fiefs; got != 2 {
		t.Errorf("P1 fiefs score = %d, want 2 (vacant still counts)", got)
	}
	// The fief members are controlled territories, so they score too.
	if got := scores[p1].Total; got != 5 {
		t.Errorf("P1 total = %d, want 5 (3 controlled territories and 2 fiefs)", got)
	}
	if got := scores[p2].Fiefs; got != 0 {
		t.Errorf("P2 fiefs score = %d, want 0", got)
	}

	state.Fiefs = nil
	scores = ComputeScores(state)
	if got := scores[p1].Fiefs; got != 0 {
		t.Errorf("P1 fiefs score after dissolution = %d, want 0", got)
	}
}

func TestWinnerForFinishedGameUsesScoreAtYearLimit(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Turn:      5,
		YearCount: 1,
		Players:   []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
			{ID: "BBB"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {Resources: 1, Army: armyPointer("A1")},
			"BBB": {Army: armyPointer("A2")},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: p2, TerritoryID: "BBB", Size: 1},
		},
	}

	if !GameFinished(state) {
		t.Fatal("state should be finished at the year limit")
	}
	winner := WinnerForFinishedGame(state)
	if winner == nil || *winner != p1 {
		t.Fatalf("winner = %v, want %s", winner, p1)
	}
}

func TestWinnerForFinishedGamePrefersSoleSurvivor(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Turn:      5,
		YearCount: 1,
		Players:   []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {Army: armyPointer("A1")},
		},
		Armies: []models.Army{{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 1}},
		Nobles: []models.Noble{{ID: "N2", OwnerID: p2, LocationID: "AAA", Status: models.NobleStatusFree}},
	}

	winner := WinnerForFinishedGame(state)
	if winner == nil || *winner != p1 {
		t.Fatalf("winner = %v, want sole survivor %s", winner, p1)
	}
}

func TestWinnerForFinishedGameReturnsNoWinnerForExactTie(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Turn:      5,
		YearCount: 1,
		Players:   []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
			{ID: "BBB"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {Army: armyPointer("A1")},
			"BBB": {Army: armyPointer("A2")},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: p2, TerritoryID: "BBB", Size: 1},
		},
	}

	if winner := WinnerForFinishedGame(state); winner != nil {
		t.Fatalf("winner = %v, want no winner for exact tie", winner)
	}
}

func TestPlayerMustSubmit(t *testing.T) {
	state := &models.GameState{
		Season:  models.SeasonSpring,
		Players: []models.Player{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}},
		Armies: []models.Army{
			{ID: "A1", OwnerID: "P1", Size: 1},
			{ID: "A2", OwnerID: "P2", Size: 1},
		},
		Nobles: []models.Noble{
			{ID: "N1", OwnerID: "P1", Status: models.NobleStatusHostage},
			{ID: "N2", OwnerID: "P2", Status: models.NobleStatusDungeon},
			{ID: "N3", OwnerID: "P3", Status: models.NobleStatusFree},
		},
	}
	for _, test := range []struct {
		season models.Season
		player models.PlayerID
		want   bool
	}{
		{models.SeasonSpring, "P1", true},  // a hostage noble can still emit
		{models.SeasonSpring, "P2", false}, // a dungeon noble cannot emit
		{models.SeasonSpring, "P3", false}, // eliminated despite a free noble
		{models.SeasonWinter, "P2", true},  // winter orders need no noble
		{models.SeasonWinter, "P3", false},
	} {
		state.Season = test.season
		if got := PlayerMustSubmit(state, test.player); got != test.want {
			t.Errorf("PlayerMustSubmit(%s, %s) = %v, want %v", test.season, test.player, got, test.want)
		}
	}
}

func TestPlayerMustSubmitWaitsForPlayerWithCardInHand(t *testing.T) {
	state := &models.GameState{
		Season:  models.SeasonSummer,
		Players: []models.Player{{ID: "P1"}},
		Armies:  []models.Army{{ID: "A1", OwnerID: "P1", Size: 1}},
		Nobles:  []models.Noble{{ID: "N1", OwnerID: "P1", Status: models.NobleStatusDungeon}},
	}
	if PlayerMustSubmit(state, "P1") {
		t.Fatal("PlayerMustSubmit = true without an emitting noble or a card")
	}
	state.SpecialDeck = &models.SpecialDeck{Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}}}
	if !PlayerMustSubmit(state, "P1") {
		t.Fatal("PlayerMustSubmit = false with a playable card in hand")
	}
}
