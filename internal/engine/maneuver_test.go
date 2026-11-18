package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func maneuverOrder(position models.TerritoryID, loop bool, destinations ...any) models.Order {
	order := models.Order{
		Type:             models.OrderTypeManeuver,
		PositionID:       position,
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{},
	}
	if loop {
		order.Liaison = models.LiaisonModeLoop
	}
	for index := 0; index < len(destinations); index += 2 {
		order.TargetIDs = append(order.TargetIDs, destinations[index].(models.TerritoryID))
		order.Counts = append(order.Counts, destinations[index+1].(int))
	}
	return order
}

func holdAt(position models.TerritoryID) models.Order {
	return models.Order{Type: models.OrderTypeHold, PositionID: position, TargetIDs: []models.TerritoryID{}}
}

func attackFrom(source, target models.TerritoryID) models.Order {
	return models.Order{Type: models.OrderTypeAttack, PositionID: source, TargetIDs: []models.TerritoryID{target}}
}

const rest = models.ManeuverRest

func maneuverMap() []models.Territory {
	return []models.Territory{
		territory("AAA", "AAA", "BBB", "CCC"),
		territory("BBB", "BBB", "AAA", "CCC"),
		territory("CCC", "CCC", "AAA", "BBB"),
	}
}

func resolveManeuverState(t *testing.T, state *models.GameState) models.GameState {
	t.Helper()
	keepTestArmiesSupplied(state)
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return *resolution.State
}

func armyAt(state *models.GameState, territoryID models.TerritoryID) *models.Army {
	for index := range state.Armies {
		if state.Armies[index].TerritoryID == territoryID {
			return &state.Armies[index]
		}
	}
	return nil
}

func TestManeuverSplitsArmyAndChainFollowsLastGroup(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2), holdAt("BBB"))
	// The last group is BBB: the chain leaves with it.
	result := resolveManeuverState(t, state)
	origin, arrived := armyAt(&result, "AAA"), armyAt(&result, "BBB")
	if origin == nil || origin.Size != 1 || origin.ChainID != nil {
		t.Fatalf("origin = %+v, want size 1 without chain", origin)
	}
	if arrived == nil || arrived.Size != 2 || arrived.ChainID == nil {
		t.Fatalf("arrived = %+v, want size 2 carrying the chain", arrived)
	}
	if noble := nobleByID(t, &result, "N1"); noble.LocationID != "BBB" {
		t.Errorf("noble at %q, want BBB (follows the chain)", noble.LocationID)
	}
}

func TestManeuverRemainingGroupCarriesChainAndNobles(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2, models.TerritoryID("AAA"), rest), holdAt("AAA"))
	result := resolveManeuverState(t, state)
	origin, arrived := armyAt(&result, "AAA"), armyAt(&result, "BBB")
	if origin == nil || origin.Size != 1 || origin.ChainID == nil {
		t.Fatalf("origin = %+v, want size 1 keeping the chain", origin)
	}
	if arrived == nil || arrived.Size != 2 || arrived.ChainID != nil {
		t.Fatalf("arrived = %+v, want size 2 without chain", arrived)
	}
	if noble := nobleByID(t, &result, "N1"); noble.LocationID != "AAA" {
		t.Errorf("noble at %q, want AAA", noble.LocationID)
	}
}

func TestManeuverWholeArmyActsAsJoin(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest), holdAt("BBB"))
	result := resolveManeuverState(t, state)
	if armyAt(&result, "AAA") != nil {
		t.Fatal("origin must be empty")
	}
	if army := armyAt(&result, "BBB"); army == nil || army.ID != "A1" || army.Size != 3 || army.ChainID == nil {
		t.Fatalf("army = %+v, want A1 size 3 with chain at BBB", army)
	}
}

func TestManeuverToEnemyDestinationStaysAndDiscardsChain(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2), holdAt("BBB"))
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "AAA"); army == nil || army.Size != 3 || army.ChainID != nil {
		t.Fatalf("origin = %+v, want the full army without chain", army)
	}
	if army := armyAt(&result, "BBB"); army == nil || army.OwnerID != "P2" || army.Size != 1 {
		t.Fatalf("enemy army = %+v, must be untouched", army)
	}
}

func TestManeuverFusesWithAlliedArmy(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2), holdAt("BBB"))
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "BBB"); army == nil || army.ID != "A2" || army.Size != 3 || army.ChainID == nil {
		t.Fatalf("host = %+v, want A2 size 3 carrying the chain", army)
	}
}

func TestManeuverFusesWithAlliedAttackerThatWins(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
		{ID: "A3", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "CCC")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2))
	addChain(t, state, "A2", "N2", attackFrom("CCC", "BBB"))
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "BBB"); army == nil || army.OwnerID != "P1" || army.Size != 4 {
		t.Fatalf("BBB = %+v, want the winning attacker fused with the group (size 4)", army)
	}
}

func TestManeuverDestinationContestedWithoutWinnerStays(t *testing.T) {
	state := testState(t, []models.Territory{
		territory("AAA", "AAA", "BBB"),
		territory("BBB", "BBB", "AAA", "CCC", "DDD"),
		territory("CCC", "CCC", "BBB"),
		territory("DDD", "DDD", "BBB"),
	}, []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
		{ID: "A3", OwnerID: "P3", TerritoryID: "DDD", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2))
	addNoble(state, "N2", "TWO", "P2", "CCC")
	addChain(t, state, "A2", "N2", attackFrom("CCC", "BBB"))
	addNoble(state, "N3", "THR", "P3", "DDD")
	addChain(t, state, "A3", "N3", attackFrom("DDD", "BBB"))
	result := resolveManeuverState(t, state)
	if armyAt(&result, "BBB") != nil {
		t.Fatalf("BBB must stay empty after a standoff, got %+v", armyAt(&result, "BBB"))
	}
	if army := armyAt(&result, "AAA"); army == nil || army.Size != 3 {
		t.Fatalf("origin = %+v, want the full army", army)
	}
}

func TestManeuverOriginAttackedWithRemainingTroopsDefendsWithRemainder(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "CCC")
	addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1))
	addChain(t, state, "A2", "N2", attackFrom("CCC", "AAA"))
	result := resolveManeuverState(t, state)
	// 2 remaining troops hold against 1 attacker and its noble (force 2); the
	// group still leaves.
	if army := armyAt(&result, "AAA"); army == nil || army.OwnerID != "P1" || army.Size != 2 {
		t.Fatalf("AAA = %+v, want P1 remainder of 2", army)
	}
	if army := armyAt(&result, "BBB"); army == nil || army.OwnerID != "P1" || army.Size != 1 {
		t.Fatalf("BBB = %+v, want the departing group", army)
	}
}

func TestManeuverOriginAttackedWithoutRemainderIsDislodged(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
		{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "CCC")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest), holdAt("BBB"))
	addChain(t, state, "A2", "N2", attackFrom("CCC", "AAA"))
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "AAA"); army == nil || army.OwnerID != "P2" {
		t.Fatalf("AAA = %+v, want the attacker", army)
	}
	// The dislodged army retreats whole and does not carry out the maneuver.
	total := 0
	for _, army := range result.Armies {
		if army.OwnerID == "P1" {
			total += army.Size
		}
	}
	if total != 2 {
		t.Errorf("P1 troops = %d, want the whole army of 2 preserved", total)
	}
	if outcome, found := findOutcome(resultEvents(t, state), "O1"); !found || outcome.Reason != "dislodged" {
		t.Errorf("M outcome = %#v, want dislodged", outcome)
	}
}

func resultEvents(t *testing.T, state *models.GameState) []Event {
	t.Helper()
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return resolution.Events
}

func TestManeuverNobleAssignments(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 4}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "AAA")
	addNoble(state, "N3", "THR", "P1", "AAA")
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest)
	order.NobleAssignments["BBB"] = []models.NobleCode{"ONE"}
	addChain(t, state, "A1", "N1", order)
	result := resolveManeuverState(t, state)
	for id, want := range map[models.NobleID]models.TerritoryID{"N1": "BBB", "N2": "CCC", "N3": "CCC"} {
		if got := nobleByID(t, &result, id).LocationID; got != want {
			t.Errorf("%s at %s, want %s", id, got, want)
		}
	}
}

func TestManeuverNobleOfFailedGroupStays(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest)
	order.NobleAssignments["BBB"] = []models.NobleCode{"ONE"}
	addChain(t, state, "A1", "N1", order)
	result := resolveManeuverState(t, state)
	if got := nobleByID(t, &result, "N1").LocationID; got != "AAA" {
		t.Errorf("noble at %s, want AAA (its group did not arrive)", got)
	}
	if army := armyAt(&result, "CCC"); army == nil || army.Size != 2 {
		t.Errorf("CCC = %+v, want the rest group", army)
	}
}

func TestManeuverLoopRetriesUndeliveredGroupsWithEditedOrder(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", maneuverOrder("AAA", true, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest))
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "BBB"); army == nil || army.Size != 1 {
		t.Fatalf("BBB = %+v, want the delivered group", army)
	}
	origin := armyAt(&result, "AAA")
	if origin == nil || origin.Size != 2 || origin.ChainID == nil {
		t.Fatalf("origin = %+v, want size 2 keeping the chain", origin)
	}
	chain := chainOf(&result, "A1")
	order := chain.Orders[chain.CurrentIndex]
	if len(order.TargetIDs) != 1 || order.TargetIDs[0] != "CCC" || order.Counts[0] != rest {
		t.Fatalf("retried order = %+v, want only the CCC+ group", order)
	}
}

func TestManeuverCrossedAndConvergentMovements(t *testing.T) {
	t.Run("crossed", func(t *testing.T) {
		state := testState(t, maneuverMap(), []models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 2},
		})
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest))
		addChain(t, state, "A2", "N2", maneuverOrder("BBB", false, models.TerritoryID("AAA"), rest))
		result := resolveManeuverState(t, state)
		if a, b := armyAt(&result, "AAA"), armyAt(&result, "BBB"); a == nil || b == nil || a.Size != 2 || b.Size != 3 {
			t.Fatalf("AAA = %+v, BBB = %+v, want sizes swapped", a, b)
		}
	})
	t.Run("same owner converges", func(t *testing.T) {
		state := testState(t, maneuverMap(), []models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 3},
		})
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "CCC")
		addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest))
		addChain(t, state, "A2", "N2", maneuverOrder("CCC", false, models.TerritoryID("BBB"), rest))
		result := resolveManeuverState(t, state)
		if army := armyAt(&result, "BBB"); army == nil || army.Size != 5 {
			t.Fatalf("BBB = %+v, want a single stacked army of 5", army)
		}
	})
	t.Run("different owners do not arrive", func(t *testing.T) {
		state := testState(t, maneuverMap(), []models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 3},
		})
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "CCC")
		addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest))
		addChain(t, state, "A2", "N2", maneuverOrder("CCC", false, models.TerritoryID("BBB"), rest))
		result := resolveManeuverState(t, state)
		if armyAt(&result, "BBB") != nil {
			t.Fatalf("BBB = %+v, want empty", armyAt(&result, "BBB"))
		}
		if a, c := armyAt(&result, "AAA"), armyAt(&result, "CCC"); a == nil || c == nil || a.Size != 2 || c.Size != 3 {
			t.Fatalf("armies must stay home: %+v %+v", a, c)
		}
	})
}

func TestManeuverChainCollisionKeepsMostRecentChain(t *testing.T) {
	var maneuverChain, hostChain models.ChainID
	build := func(t *testing.T, maneuverEmitted, hostEmitted int) models.GameState {
		state := testState(t, maneuverMap(), []models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
		})
		state.Turn = 9
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest), holdAt("BBB"))
		addChainOrders(t, state, "A2", "N2", holdAt("BBB"), holdAt("BBB"))
		state.Chains[0].EmittedTurn = maneuverEmitted
		state.Chains[1].EmittedTurn = hostEmitted
		maneuverChain, hostChain = state.Chains[0].ID, state.Chains[1].ID
		return resolveManeuverState(t, state)
	}
	t.Run("more recent maneuver chain wins", func(t *testing.T) {
		result := build(t, 8, 6)
		if army := armyAt(&result, "BBB"); army == nil || army.ChainID == nil || *army.ChainID != maneuverChain {
			t.Fatalf("BBB = %+v, want the maneuver chain", army)
		}
	})
	t.Run("more recent host chain wins", func(t *testing.T) {
		result := build(t, 6, 8)
		if army := armyAt(&result, "BBB"); army == nil || army.ChainID == nil || *army.ChainID != hostChain {
			t.Fatalf("BBB = %+v, want the host chain", army)
		}
	})
	t.Run("same turn cancels both", func(t *testing.T) {
		result := build(t, 7, 7)
		if army := armyAt(&result, "BBB"); army == nil || army.ChainID != nil {
			t.Fatalf("BBB = %+v, want no chain", army)
		}
		if len(result.Chains) != 0 {
			t.Fatalf("chains = %+v, want none", result.Chains)
		}
	})
}

func TestManeuverChainEndingWithManeuverDoesNotCollide(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest))
	addChainOrders(t, state, "A2", "N2", holdAt("BBB"), holdAt("BBB"))
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "BBB"); army == nil || army.Size != 4 || army.ChainID == nil {
		t.Fatalf("BBB = %+v, want the host chain untouched", army)
	}
}

func TestManeuverBadWeather(t *testing.T) {
	state := effectTestState()
	state.Regions = []models.Region{
		{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}},
		{ID: "BBB", Seed: "BBB", Territories: []models.TerritoryID{"BBB"}},
	}
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 3}}
	state.TerritoryStates["BBB"] = models.TerritoryState{Army: armyPointer("A1")}
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)

	// Every destination hit: invalid, the chain pauses.
	record := &orderRecord{armyID: "A1", order: maneuverOrder("BBB", false, models.TerritoryID("AAA"), 1)}
	ctx.enumerateOrder(record, state.Armies[0], false)
	if record.outcome != OutcomeInvalid || record.reason != "bad_weather" {
		t.Fatalf("record = %#v, want bad_weather invalidation", record)
	}
	// A hit destination is dropped, the others go on.
	record = &orderRecord{armyID: "A1", order: maneuverOrder("BBB", false, models.TerritoryID("AAA"), 1, models.TerritoryID("BBB"), rest)}
	ctx.enumerateOrder(record, state.Armies[0], false)
	intent := ctx.maneuvers["A1"]
	if record.outcome != "" || intent == nil || intent.remaining != 3 || len(intent.groups) != 0 {
		t.Fatalf("record = %#v intent = %#v, want the hit group dropped and 3 troops staying", record, intent)
	}
}

func joinFrom(source, target models.TerritoryID) models.Order {
	return models.Order{Type: models.OrderTypeJoin, PositionID: source, TargetIDs: []models.TerritoryID{target}}
}

func troopsOf(state *models.GameState, owner models.PlayerID) int {
	total := 0
	for _, army := range state.Armies {
		if army.OwnerID == owner {
			total += army.Size
		}
	}
	return total
}

func TestManeuverKeepsReinforcementsThatReachTheOrigin(t *testing.T) {
	for name, destinations := range map[string][]any{
		"partial": {models.TerritoryID("BBB"), 1},
		"whole":   {models.TerritoryID("BBB"), rest},
	} {
		t.Run(name, func(t *testing.T) {
			state := testState(t, maneuverMap(), []models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
				{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
			})
			addNoble(state, "N1", "ONE", "P1", "AAA")
			addNoble(state, "N2", "TWO", "P1", "CCC")
			addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, destinations...))
			addChain(t, state, "A2", "N2", joinFrom("CCC", "AAA"))
			result := resolveManeuverState(t, state)
			if got := troopsOf(&result, "P1"); got != 5 {
				t.Fatalf("P1 troops = %d, want 5 (the joining army must not vanish): %+v", got, result.Armies)
			}
			if army := armyAt(&result, "AAA"); army == nil || army.Size < 2 {
				t.Fatalf("AAA = %+v, want the joining troops kept", army)
			}
		})
	}
}

func TestManeuverChainLeavingWithItsOwnGroupIsNotAHostChain(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 2},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest), holdAt("BBB"))
	addChainOrders(t, state, "A2", "N2", maneuverOrder("BBB", false, models.TerritoryID("CCC"), 1), holdAt("CCC"))
	arriving, leaving := state.Chains[0].ID, state.Chains[1].ID
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "BBB"); army == nil || army.Size != 4 || army.ChainID == nil || *army.ChainID != arriving {
		t.Fatalf("BBB = %+v, want the arriving chain on 4 troops", army)
	}
	if army := armyAt(&result, "CCC"); army == nil || army.ChainID == nil || *army.ChainID != leaving {
		t.Fatalf("CCC = %+v, want the chain that left with its group", army)
	}
}

func TestManeuverLoopWithoutTroopsLeftDiscardsTheChain(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", maneuverOrder("AAA", true, models.TerritoryID("BBB"), 3, models.TerritoryID("CCC"), 1))
	result := resolveManeuverState(t, state)
	if len(result.Chains) != 0 {
		t.Fatalf("chains = %+v, want the chain discarded (nothing left on its origin)", result.Chains)
	}
	if army := armyAt(&result, "BBB"); army == nil || army.Size != 3 || army.ChainID != nil {
		t.Fatalf("BBB = %+v, want the whole army without chain", army)
	}
}

func TestManeuverNobleFollowsTheLastArrivedGroupWhenNoTroopStays(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2, models.TerritoryID("CCC"), rest))
	result := resolveManeuverState(t, state)
	// The carrier group CCC+ received no troops and the whole army left: a
	// noble cannot stay alone on the emptied origin, it follows the group
	// that arrived.
	if got := nobleByID(t, &result, "N1").LocationID; got != "BBB" {
		t.Errorf("noble at %s, want BBB (the last group that arrived)", got)
	}
}

func TestManeuverChainCollisionWithAChainHandedOverThisTurn(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
		{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
	})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "CCC")
	addChainOrders(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest), holdAt("BBB"))
	addChainOrders(t, state, "A2", "N2", models.Order{
		Type: models.OrderTypeDisperse, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{},
	}, holdAt("BBB"))
	// Same emission turn: both chains have orders left, so both are cancelled
	// and the turn must still resolve to a valid state.
	result := resolveManeuverState(t, state)
	if army := armyAt(&result, "BBB"); army == nil || army.ChainID != nil || army.Size != 4 {
		t.Fatalf("BBB = %+v, want the 4 troops without chain", army)
	}
	if len(result.Chains) != 0 {
		t.Fatalf("chains = %+v, want none", result.Chains)
	}
}

func TestManeuverGroupIsNotStruckTwiceByThePlague(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 8}})
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}}}
	setCurrentCalamity(state, models.CardKindPlague, "AAA")
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2))
	keepTestArmiesSupplied(state)
	validateTestState(t, state)
	resolution, err := Resolve(state, plagueBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// 8 troops are struck once (4); the group that leaves is part of those 4.
	if got := troopsOf(resolution.State, "P1"); got != 4 {
		t.Fatalf("P1 troops = %d, want 4 (struck once): %+v", got, resolution.State.Armies)
	}
}

func TestManeuverLeavingEntirelyStillReceivesDefensiveSupport(t *testing.T) {
	build := func(t *testing.T, supported bool) models.GameState {
		state := testState(t, []models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "DDD"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
			territory("DDD", "DDD", "AAA"),
		}, []models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
			{ID: "A2", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		})
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "DDD")
		addNoble(state, "N3", "THR", "P1", "CCC")
		addChain(t, state, "A1", "N1", maneuverOrder("AAA", false, models.TerritoryID("BBB"), rest))
		addChain(t, state, "A2", "N2", attackFrom("DDD", "AAA"))
		if supported {
			addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
		}
		return resolveManeuverState(t, state)
	}
	// Supported (force 1 + 1 noble = 2 against the attacker's 1 + 1 = 2): the
	// attack is stopped and the army redeploys to BBB.
	held := build(t, true)
	if army := armyAt(&held, "AAA"); army != nil {
		t.Fatalf("AAA = %+v, want it empty: the army left and the attack was stopped", army)
	}
	if army := armyAt(&held, "BBB"); army == nil || army.OwnerID != "P1" || army.Size != 3 {
		t.Fatalf("BBB = %+v, want the whole army redeployed", army)
	}
	if army := armyAt(&held, "DDD"); army == nil || army.OwnerID != "P2" {
		t.Fatalf("DDD = %+v, want the attacker bounced back to its origin", army)
	}
	// Without support nothing defends the territory: the attack takes it.
	fallen := build(t, false)
	if army := armyAt(&fallen, "AAA"); army == nil || army.OwnerID != "P2" {
		t.Fatalf("AAA = %+v, want the attacker in place", army)
	}
}

func TestManeuverPreciseNobleSplitLikeDispersion(t *testing.T) {
	// AAA M BBB/2*HUG CCC* DDD+ : two troops and HUG to BBB, one troop and the
	// remaining nobles to CCC, the rest of the troops and the chain to DDD.
	state := testState(t, []models.Territory{
		territory("AAA", "AAA", "BBB", "CCC", "DDD"),
		territory("BBB", "BBB", "AAA"),
		territory("CCC", "CCC", "AAA"),
		territory("DDD", "DDD", "AAA"),
	}, []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 6}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "HUG", "P1", "AAA")
	addNoble(state, "N3", "TWO", "P1", "AAA")
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2, models.TerritoryID("CCC"), 1, models.TerritoryID("DDD"), rest)
	order.NobleAssignments["BBB"] = []models.NobleCode{"HUG"}
	order.NobleAssignments["CCC"] = []models.NobleCode{"*"}
	addChainOrders(t, state, "A1", "N1", order, holdAt("DDD"))
	result := resolveManeuverState(t, state)
	for territory, size := range map[models.TerritoryID]int{"BBB": 2, "CCC": 1, "DDD": 3} {
		if army := armyAt(&result, territory); army == nil || army.Size != size {
			t.Errorf("%s = %+v, want %d troops", territory, army, size)
		}
	}
	for id, want := range map[models.NobleID]models.TerritoryID{"N2": "BBB", "N1": "CCC", "N3": "CCC"} {
		if got := nobleByID(t, &result, id).LocationID; got != want {
			t.Errorf("%s at %s, want %s", id, got, want)
		}
	}
	if army := armyAt(&result, "DDD"); army == nil || army.ChainID == nil {
		t.Errorf("DDD = %+v, want the chain", army)
	}
	if army := armyAt(&result, "AAA"); army != nil {
		t.Errorf("AAA = %+v, want it empty", army)
	}
}
