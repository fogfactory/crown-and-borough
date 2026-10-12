package engine

import (
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ambushState is a P2 army of the given size on BBB attacking AAA, where P1 has
// a one-troop army carrying the nobles N2 and N3 (N2 first in the line of
// succession) and an empty retreat CCC next to AAA. P2 holds an Ambush card.
func ambushState(t *testing.T, attackerSize int, withRetreat bool) *models.GameState {
	t.Helper()
	territories := []models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")}
	if withRetreat {
		territories = []models.Territory{territory("AAA", "AAA", "BBB", "CCC"), territory("BBB", "BBB", "AAA"), territory("CCC", "CCC", "AAA")}
	}
	state := testState(t, territories,
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: attackerSize}},
	)
	addAnchorArmy(t, state, "A2", "P1", "AAA")
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P2", "BBB")
	addNoble(state, "N3", "TRI", "P1", "AAA")
	addNoble(state, "N2", "TWA", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindAmbush}},
		DrawPile: []models.SpecialCardID{}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P2": {"C1"}},
	}
	return state
}

func ambushOrders(source, target models.TerritoryID) map[models.PlayerID][]models.DeckOrder {
	return map[models.PlayerID][]models.DeckOrder{"P2": {{
		ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindAmbush,
		SourceTerritoryID: source, TargetTerritoryID: target,
	}}}
}

func TestAmbushCapturesTheFirstNobleInTheSuccessionLine(t *testing.T) {
	state := ambushState(t, 3, true)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), ambushOrders("BBB", "AAA"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if got := armyByID(t, resolution.State, "A1").TerritoryID; got != "AAA" {
		t.Fatalf("A1 is on %s, want the attack to succeed", got)
	}
	captured := nobleByID(t, resolution.State, "N2")
	if captured.Status != models.NobleStatusHostage || captured.LocationID != "AAA" || captured.OwnerID != "P1" {
		t.Fatalf("N2 = %#v, want a P1 hostage held on AAA by the P2 army", captured)
	}
	if other := nobleByID(t, resolution.State, "N3"); other.Status != models.NobleStatusFree || other.LocationID != "CCC" {
		t.Fatalf("N3 = %#v, want free on the retreat CCC", other)
	}
	events := eventsOfType(resolution.Events, EventTypeAmbush)
	if len(events) != 1 || events[0].Reason != AmbushCaptureReason || events[0].NobleCode != "TWA" || events[0].OwnerID != "P2" {
		t.Fatalf("ambush events = %#v", events)
	}
	if got := resolution.State.SpecialDeck.Hands["P2"]; len(got) != 0 {
		t.Fatalf("hand = %#v, want the card consumed", got)
	}
}

func TestAmbushHasNoEffectWithoutVictoryRetreatOrNoble(t *testing.T) {
	cases := map[string]func() *models.GameState{
		// Force 1 (+1 noble) only ties the one-troop army: no victory.
		"attack fails": func() *models.GameState { return ambushState(t, 1, true) },
		// Nowhere to retreat: the defending army is destroyed.
		"no retreat": func() *models.GameState { return ambushState(t, 3, false) },
		"no noble": func() *models.GameState {
			state := ambushState(t, 3, true)
			state.Nobles = state.Nobles[:1]
			return state
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			state := build()
			validateTestState(t, state)
			resolution, err := ResolveWithDeckOrders(state, testBalance(), ambushOrders("BBB", "AAA"))
			if err != nil {
				t.Fatalf("ResolveWithDeckOrders: %v", err)
			}
			events := eventsOfType(resolution.Events, EventTypeAmbush)
			if len(events) != 1 || events[0].Reason != AmbushVoidReason {
				t.Fatalf("ambush events = %#v, want one void ambush", events)
			}
			if got := resolution.State.SpecialDeck.Hands["P2"]; len(got) != 0 {
				t.Fatalf("hand = %#v, want the card consumed", got)
			}
		})
	}
}

func TestAmbushIsRejectedWithoutAPlannedAttack(t *testing.T) {
	state := ambushState(t, 3, true)
	_, err := ResolveWithDeckOrders(state, testBalance(), ambushOrders("BBB", "BBB"))
	if err == nil || !strings.Contains(err.Error(), "ambush_no_attack") {
		t.Fatalf("error = %v, want ambush_no_attack", err)
	}
	if got := state.SpecialDeck.Hands["P2"]; len(got) != 1 {
		t.Fatalf("hand = %#v, want the card kept", got)
	}
}

func TestAmbushIsParsed(t *testing.T) {
	state := ambushState(t, 3, true)
	parsed, parseErrors := orders.ParseDeckOrders("P EM BBB AAA\nP AM BBB AAA\n", state)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseDeckOrders: %v", parseErrors)
	}
	for _, order := range parsed {
		if order.Kind != models.CardKindAmbush || order.SourceTerritoryID != "BBB" || order.TargetTerritoryID != "AAA" {
			t.Fatalf("order = %#v", order)
		}
	}
	for _, line := range []string{"P EM BBB", "P EM BBB ZZZ", "P EM BBB AAA CCC"} {
		if _, parseErrors := orders.ParseDeckOrders(line+"\n", state); len(parseErrors) == 0 {
			t.Errorf("%q must be rejected", line)
		}
	}
}

func TestAmbushLeavesTheBastardLastInTheLineAndSendsItToTheDungeon(t *testing.T) {
	state := ambushState(t, 3, true)
	for index := range state.Nobles {
		switch state.Nobles[index].ID {
		case "N2":
			state.Nobles[index].Dignities = []models.Dignity{models.DignityBastard}
		}
	}
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), ambushOrders("BBB", "AAA"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	// N2 is a bastard, so it comes last in the line: N3 is captured instead.
	if noble := nobleByID(t, resolution.State, "N3"); noble.Status != models.NobleStatusHostage {
		t.Fatalf("N3 = %#v, want the first of the line captured", noble)
	}
	if noble := nobleByID(t, resolution.State, "N2"); noble.Status != models.NobleStatusFree {
		t.Fatalf("N2 = %#v, want the bastard left free", noble)
	}
}

func TestAmbushSecondCardOnTheSameAttackIsWasted(t *testing.T) {
	state := ambushState(t, 3, true)
	state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: "C2", Kind: models.CardKindAmbush})
	state.SpecialDeck.Hands["P2"] = []models.SpecialCardID{"C1", "C2"}
	validateTestState(t, state)
	orders := ambushOrders("BBB", "AAA")
	second := orders["P2"][0]
	second.ID = "O2"
	orders["P2"] = append(orders["P2"], second)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), orders)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	reasons := map[string]int{}
	for _, event := range eventsOfType(resolution.Events, EventTypeAmbush) {
		reasons[event.Reason]++
	}
	if reasons[AmbushCaptureReason] != 1 || reasons[AmbushVoidReason] != 1 {
		t.Fatalf("ambush reasons = %v, want one capture and one void", reasons)
	}
	captured := 0
	for _, noble := range resolution.State.Nobles {
		if noble.OwnerID == "P1" && noble.Status != models.NobleStatusFree {
			captured++
		}
	}
	if captured != 1 {
		t.Fatalf("captured nobles = %d, want exactly one", captured)
	}
}

func TestAmbushOnAnotherPlayersAttackCapturesForTheAttacker(t *testing.T) {
	state := ambushState(t, 3, true)
	state.SpecialDeck.Hands = map[models.PlayerID][]models.SpecialCardID{"P3": {"C1"}}
	validateTestState(t, state)
	orders := ambushOrders("BBB", "AAA")
	orders["P3"], orders["P2"] = orders["P2"], nil
	resolution, err := ResolveWithDeckOrders(state, testBalance(), orders)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	captured := nobleByID(t, resolution.State, "N2")
	if captured.Status != models.NobleStatusHostage || captured.LocationID != "AAA" {
		t.Fatalf("N2 = %#v, want a hostage held on AAA", captured)
	}
	if holder := armyByID(t, resolution.State, "A1"); holder.TerritoryID != "AAA" || holder.OwnerID != "P2" {
		t.Fatalf("A1 = %#v, want the P2 attacker to hold the hostage", holder)
	}
	events := eventsOfType(resolution.Events, EventTypeAmbush)
	if len(events) != 1 || events[0].Reason != AmbushCaptureReason || events[0].OwnerID != "P3" {
		t.Fatalf("ambush events = %#v", events)
	}
}

func TestAmbushIsRejectedWhenNoAttackIsPossible(t *testing.T) {
	cases := map[string]struct{ source, target models.TerritoryID }{
		"origin without army":   {"CCC", "AAA"},
		"origins not adjacent":  {"BBB", "CCC"},
		"own army without plan": {"AAA", "BBB"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			state := ambushState(t, 3, true)
			state.SpecialDeck.Hands = map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}}
			orders := ambushOrders(test.source, test.target)
			orders["P1"], orders["P2"] = orders["P2"], nil
			_, err := ResolveWithDeckOrders(state, testBalance(), orders)
			if err == nil || !strings.Contains(err.Error(), "ambush_no_attack") {
				t.Fatalf("error = %v, want ambush_no_attack", err)
			}
		})
	}
}
