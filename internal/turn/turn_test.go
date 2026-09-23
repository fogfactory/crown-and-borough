package turn

import (
	"errors"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestNormalizeSubmissionAssignsEveryOrderToThePlayer(t *testing.T) {
	input := engine.OrdersInput{
		Chains:  []engine.ChainSubmission{{Noble: "HUG", Text: "HUG\nROS A BOI"}},
		Winter:  []engine.WinterSubmission{{Lines: "R T ROS"}},
		Special: []engine.DeckSubmission{{Player: "P1", Text: "P BH ROS"}},
	}
	normalized, err := NormalizeSubmission("P1", input)
	if err != nil {
		t.Fatalf("NormalizeSubmission = %v", err)
	}
	if normalized.Chains[0].Player != "P1" || normalized.Winter[0].Player != "P1" || normalized.Special[0].Player != "P1" {
		t.Fatalf("normalized = %#v, want every order assigned to P1", normalized)
	}
	if input.Chains[0].Player != "" {
		t.Fatal("NormalizeSubmission mutated its input")
	}
}

func TestNormalizeSubmissionRejectsOrdersOfAnotherPlayer(t *testing.T) {
	for name, input := range map[string]engine.OrdersInput{
		"chain":   {Chains: []engine.ChainSubmission{{Player: "P2", Text: "HUG\nROS A BOI"}}},
		"winter":  {Winter: []engine.WinterSubmission{{Player: "P2", Lines: "R T ROS"}}},
		"special": {Special: []engine.DeckSubmission{{Player: "P2", Text: "P BH ROS"}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NormalizeSubmission("P1", input)
			var inputErrors *engine.InputErrors
			if !errors.As(err, &inputErrors) || len(inputErrors.Errors) != 1 || inputErrors.Errors[0].Code != "foreign_player_order" {
				t.Fatalf("NormalizeSubmission error = %#v, want one foreign_player_order input error", err)
			}
		})
	}
}

func progressState() *models.GameState {
	return &models.GameState{
		Season:  models.SeasonSpring,
		Players: []models.Player{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}, {ID: "P4"}},
		Armies: []models.Army{
			{ID: "A1", OwnerID: "P1", Size: 1},
			{ID: "A2", OwnerID: "P2", Size: 1},
			{ID: "A3", OwnerID: "P3", Size: 1},
		},
		Nobles: []models.Noble{
			{ID: "N1", OwnerID: "P1", Status: models.NobleStatusFree},
			{ID: "N2", OwnerID: "P2", Status: models.NobleStatusDungeon},
			{ID: "N3", OwnerID: "P3", Status: models.NobleStatusDungeon},
			{ID: "N4", OwnerID: "P4", Status: models.NobleStatusFree},
		},
		SpecialDeck: &models.SpecialDeck{
			Hands: map[models.PlayerID][]models.SpecialCardID{"P3": {"C1"}},
		},
	}
}

func TestProgressWaitsOnlyForPlayersWhoCanOrder(t *testing.T) {
	state := progressState()
	none := func(models.PlayerID) bool { return false }

	// P1 has a free noble, P2 only a dungeon noble, P3 a card in hand and P4 is
	// eliminated despite its free noble.
	submitted, remaining := Progress(state, none)
	if len(submitted) != 0 || !reflect.DeepEqual(remaining, []models.PlayerID{"P1", "P3"}) {
		t.Fatalf("action progress = %v / %v, want [] / [P1 P3]", submitted, remaining)
	}

	state.Season = models.SeasonWinter
	submitted, remaining = Progress(state, func(playerID models.PlayerID) bool { return playerID == "P1" })
	if !reflect.DeepEqual(submitted, []models.PlayerID{"P1"}) || !reflect.DeepEqual(remaining, []models.PlayerID{"P2", "P3"}) {
		t.Fatalf("winter progress = %v / %v, want [P1] / [P2 P3]", submitted, remaining)
	}
}

func TestCombineSubmissionsFollowsPlayerOrder(t *testing.T) {
	state := progressState()
	combined := CombineSubmissions(state, map[models.PlayerID]engine.OrdersInput{
		"P3": {Chains: []engine.ChainSubmission{{Player: "P3"}}, Special: []engine.DeckSubmission{{Player: "P3"}}},
		"P1": {Chains: []engine.ChainSubmission{{Player: "P1"}}, Winter: []engine.WinterSubmission{{Player: "P1"}}},
	})
	if len(combined.Chains) != 2 || combined.Chains[0].Player != "P1" || combined.Chains[1].Player != "P3" {
		t.Fatalf("chains = %#v, want P1 then P3", combined.Chains)
	}
	if len(combined.Winter) != 1 || len(combined.Special) != 1 {
		t.Fatalf("combined = %#v, want one winter and one special order", combined)
	}
}

func TestCloneOrdersKeepsSpecialOrders(t *testing.T) {
	input := engine.OrdersInput{Special: []engine.DeckSubmission{{Player: "P1", Text: "P BH ROS"}}}
	clone := CloneOrders(input)
	if !reflect.DeepEqual(clone.Special, input.Special) {
		t.Fatalf("clone special = %#v, want %#v", clone.Special, input.Special)
	}
	clone.Special[0].Text = "changed"
	if input.Special[0].Text != "P BH ROS" {
		t.Fatal("CloneOrders shares the special order slice")
	}
}

func TestOutcomeReportsSoleSurvivor(t *testing.T) {
	state := progressState()
	if finished, _ := Outcome(state); finished {
		t.Fatal("Outcome = finished with three players alive")
	}
	state.Armies = state.Armies[:1]
	finished, winner := Outcome(state)
	if !finished || winner == nil || *winner != "P1" {
		t.Fatalf("Outcome = %v, %v; want finished with winner P1", finished, winner)
	}
}
