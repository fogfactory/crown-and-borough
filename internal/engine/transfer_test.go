package engine

import (
	"errors"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestResolveTransferFailsForArmyStarvingSinceLastTurn covers #208's point
// that famished_sender now reads the sending army's persisted
// models.Army.Starving flag, carried over from last turn's ravitaillement,
// not a flag computed within this same turn (which no longer runs before
// movement at all).
func TestResolveTransferFailsForArmyStarvingSinceLastTurn(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	validateTestState(t, state)

	turn1, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("turn 1 Resolve: %v", err)
	}
	if army := armyByID(t, turn1.State, "A1"); !army.Starving {
		t.Fatalf("A1 after turn 1 = %+v, want starving", army)
	}

	state2 := cloneGameState(turn1.State)
	state2.Territories = append(state2.Territories, territory("BBB", "BBB", "AAA"))
	state2.TerritoryStates["BBB"] = models.TerritoryState{}
	for i := range state2.Territories {
		if state2.Territories[i].ID == "AAA" {
			state2.Territories[i].Adjacencies = append(state2.Territories[i].Adjacencies, "BBB")
		}
	}
	// A recipient army of another player at BBB, so the transfer order
	// itself is otherwise valid: only the sender's persisted famine should
	// block it.
	state2.Armies = append(state2.Armies, models.Army{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1})
	recipientState := state2.TerritoryStates["BBB"]
	recipientArmyID := models.ArmyID("A2")
	recipientState.Army = &recipientArmyID
	state2.TerritoryStates["BBB"] = recipientState
	state2.NextArmyID = nextArmyID(state2.Armies)
	addNoble(state2, "N1", "ONE", "P1", "AAA")
	addChain(t, state2, "A1", "N1", models.Order{
		Type: models.OrderTypeTransfer, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Amount: 1,
	})
	validateTestState(t, state2)

	turn2, err := Resolve(state2, testBalance())
	if err != nil {
		t.Fatalf("turn 2 Resolve: %v", err)
	}
	outcome, found := findOutcome(turn2.Events, "O1")
	if !found || outcome.Reason != "famished_sender" {
		t.Errorf("transfer outcome = %#v, found=%t, want famished_sender", outcome, found)
	}
}

func TestResolveTransferToRecipientArmyAndConsumesLocalCacheFirst(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setTerritoryResources(state, "AAA", 4)
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeTransfer,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB"},
		Amount:     1,
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 3 {
		t.Errorf("source resources = %d, want 3", got)
	}
	if got := resolution.State.TerritoryStates["BBB"].Resources; got != 1 {
		t.Errorf("target resources = %d, want 1", got)
	}
	event, ok := findEvent(resolution.Events, EventTypeTransfer)
	if !ok || event.OtherArmyID != "A2" || event.ResourceAmount != 1 || event.Partial {
		t.Errorf("transfer event = %#v, want full transfer to A2", event)
	}
}

func TestResolveTransferShortageDoesNotBreakSingleChain(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1",
		models.Order{Type: models.OrderTypeTransfer, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Amount: 1},
		models.Order{Type: models.OrderTypeHold, PositionID: "AAA"},
	)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	outcome, ok := findOutcome(resolution.Events, "O1")
	if !ok || outcome.Outcome != OutcomeFailure || outcome.Reason != "insufficient_resources" || outcome.Progression != ProgressionAdvanced {
		t.Errorf("short transfer outcome = %#v, want non-breaking failure", outcome)
	}
	chain := chainOf(resolution.State, "A1")
	if chain == nil || chain.CurrentIndex != 1 || chain.Orders[chain.CurrentIndex].Type != models.OrderTypeHold {
		t.Fatalf("chain after shortage = %#v, want advanced to hold", chain)
	}
}

func TestResolveLoopTransferUsesPartialFinalShipment(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	// Preset below the requested amount (the army's transfer capacity, 4):
	// transfers now execute before ravitaillement (#208), so nothing else
	// draws from AAA's stock first any more, and the shortfall must come
	// from the preset itself to still force a partial shipment.
	setTerritoryResources(state, "AAA", 3)
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeTransfer,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB"},
		Amount:     4,
		Liaison:    models.LiaisonModeLoop,
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 0 {
		t.Errorf("source resources = %d, want exhausted cache", got)
	}
	if got := resolution.State.TerritoryStates["BBB"].Resources; got != 3 {
		t.Errorf("target resources = %d, want partial shipment of 3", got)
	}
	event, ok := findEvent(resolution.Events, EventTypeTransfer)
	if !ok || event.ResourceAmount != 3 || !event.Partial {
		t.Errorf("partial transfer event = %#v", event)
	}
	if chainOf(resolution.State, "A1") != nil {
		t.Fatal("completed loop transfer should consume its chain")
	}
}

func TestResolveTransferRejectsBlockedIntermediateArmy(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "MID"),
		supplyTerritory("MID", "MID", models.TerrainPlain, "AAA", "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "MID"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "MID", Size: 1},
		{ID: "A3", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setTerritoryResources(state, "AAA", 2)
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeTransfer,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB"},
		Amount:     1,
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	outcome, ok := findOutcome(resolution.Events, "O1")
	if !ok || outcome.Outcome != OutcomeInvalid || outcome.Reason != "transfer_path_blocked" {
		t.Errorf("blocked transfer outcome = %#v", outcome)
	}
	if chainOf(resolution.State, "A1") != nil {
		t.Fatal("blocked transfer should break its chain")
	}
}

func TestResolveTurnRejectsCurrentlyBlockedInitialTransferAtSubmission(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "MID"),
		supplyTerritory("MID", "MID", models.TerrainPlain, "AAA", "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "MID"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "MID", Size: 1},
		{ID: "A3", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setTerritoryResources(state, "AAA", 2)
	validateTestState(t, state)

	_, err := ResolveTurn(state, testBalance(), OrdersInput{Chains: []ChainSubmission{{
		Player: "P1",
		Noble:  "ONE",
		Text:   "ONE\nAAA T BBB 1",
	}}})
	var inputErrors *InputErrors
	if !errors.As(err, &inputErrors) {
		t.Fatalf("ResolveTurn error = %v, want submission input errors", err)
	}
	if len(inputErrors.Errors) != 1 || inputErrors.Errors[0].Code != "transfer_path_blocked" {
		t.Fatalf("input errors = %#v, want transfer_path_blocked", inputErrors.Errors)
	}
}

func TestResolveWinterTransferUsesSettlementPaymentAndOnlySettlementTarget(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	state.Turn = 4
	state.Season = models.SeasonWinter
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	setTerritoryResources(state, "AAA", 5)
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeTransfer, SourceID: "AAA", TargetID: "BBB", Amount: 3}},
	})
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 1 {
		t.Errorf("winter source resources = %d, want 1 after payment and conservation", got)
	}
	if got := resolution.State.TerritoryStates["BBB"].Resources; got != 2 {
		t.Errorf("winter target resources = %d, want 2 after conservation", got)
	}
	if events := eventsOfType(resolution.Events, EventTypeTransfer); len(events) != 1 || events[0].ResourceAmount != 3 {
		t.Errorf("winter transfer events = %#v", events)
	}
}

func TestResolveWinterKeepsDepotStockAndLosesBareCache(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	state.Turn = 4
	state.Season = models.SeasonWinter
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "AAA"})
	setTerritoryResources(state, "AAA", 5)
	setTerritoryResources(state, "BBB", 5)
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 5 {
		t.Errorf("depot stock = %d, want integral conservation", got)
	}
	if got := resolution.State.TerritoryStates["BBB"].Resources; got != 0 {
		t.Errorf("bare cache stock = %d, want winter loss", got)
	}
}

func findEvent(events []Event, eventType EventType) (Event, bool) {
	for _, event := range events {
		if event.Type == eventType {
			return event, true
		}
	}
	return Event{}, false
}

// TestResolveTransferRejectsTargetOccupiedByNonController verifies that a
// transfer aimed at an army stationed on a territory it does not itself
// control (only occupies) is rejected with a dedicated reason, rather than
// silently handing resources to an army with no claim on the territory
// (titres.md, #196).
func TestResolveTransferRejectsTargetOccupiedByNonController(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		supplyTerritory("CCC", "CCC", models.TerrainPlain),
		supplyTerritory("DDD", "DDD", models.TerrainPlain),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	// BBB is a member of P3's barony, only occupied (not controlled) by P2's
	// army.
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "CCC",
		Territories: []models.TerritoryID{"CCC", "BBB", "DDD"}, OwnerID: "P3",
	}}
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setTerritoryResources(state, "AAA", 4)
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeTransfer,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB"},
		Amount:     1,
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	outcome, ok := findOutcome(resolution.Events, "O1")
	if !ok || outcome.Outcome != OutcomeInvalid || outcome.Reason != "transfer_target_occupied" {
		t.Fatalf("outcome = %#v, want invalid transfer_target_occupied", outcome)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 4 {
		t.Errorf("source resources = %d, want unchanged 4 (no prelevement)", got)
	}
}

// TestResolveTransferRejectsAbandonedCastleTarget checks #215: a transfer
// aimed at an empty castle with no controller (an enemy's, abandoned once
// its army left, outside every fief and capital) is invalid, exactly like a
// target with no settlement at all: control.controlsTerritory rejects it
// through the ordinary "no controller" branch, no dedicated code needed.
func TestResolveTransferRejectsAbandonedCastleTarget(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
	})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setTerritoryResources(state, "AAA", 4)
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeTransfer,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB"},
		Amount:     1,
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	outcome, ok := findOutcome(resolution.Events, "O1")
	if !ok || outcome.Outcome != OutcomeInvalid || outcome.Reason != "invalid_transfer_destination" {
		t.Fatalf("outcome = %#v, want invalid invalid_transfer_destination", outcome)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 4 {
		t.Errorf("source resources = %d, want unchanged 4 (no prelevement)", got)
	}
}

// TestResolveWinterTransferRejectsOccupiedSource verifies that a winter
// transfer order cannot debit a settlement occupied against its controller
// (titres.md, #196).
func TestResolveWinterTransferRejectsOccupiedSource(t *testing.T) {
	state := testState(t, []models.Territory{
		supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
		supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		supplyTerritory("CCC", "CCC", models.TerrainPlain),
		supplyTerritory("DDD", "DDD", models.TerrainPlain),
	}, []models.Army{
		{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1},
	})
	state.Turn = 4
	state.Season = models.SeasonWinter
	// AAA is a member of P1's barony, occupied by P2's army.
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "CCC",
		Territories: []models.TerritoryID{"CCC", "AAA", "DDD"}, OwnerID: "P1",
	}}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	setTerritoryResources(state, "AAA", 5)
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeTransfer, SourceID: "AAA", TargetID: "BBB", Amount: 3}},
	})
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	rejected := eventsOfType(resolution.Events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "territory_occupied_by_other_player" {
		t.Fatalf("rejected events = %#v, want territory_occupied_by_other_player", rejected)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 3 {
		t.Errorf("source resources after winter conservation = %d, want 3 (ceil(5/2)), no transfer debit", got)
	}
}
