package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func nobleIDPtr(id models.NobleID) *models.NobleID {
	return &id
}

// TestVacantFiefAutoAssignedAtWinterEnd verifies that a fief still vacant at
// the end of winter is no longer dissolved: it is attributed by default to
// its owner's free noble with the lowest trigram, with a warning event
// (titres.md "Perte et vacance d'un fief").
func TestVacantFiefAutoAssignedAtWinterEnd(t *testing.T) {
	state := foundFiefTestState(t)
	addNoble(state, "N2", "ABC", "P1", "AAA")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
	}}
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if len(resolution.State.Fiefs) != 1 {
		t.Fatalf("fiefs = %#v, want the fief to survive, auto-assigned", resolution.State.Fiefs)
	}
	// N2/ABC sorts before N1/HUG: it is the default holder.
	if holder := resolution.State.Fiefs[0].HolderNobleID; holder == nil || *holder != "N2" {
		t.Fatalf("holder = %v, want N2 (ABC, lowest trigram)", holder)
	}
	assigned := eventsOfType(resolution.Events, EventTypeFiefAutoAssigned)
	if len(assigned) != 1 || assigned[0].FiefID != "F1" || assigned[0].NobleID != "N2" || assigned[0].NobleCode != "ABC" {
		t.Fatalf("auto-assigned events = %#v", assigned)
	}
	if len(eventsOfType(resolution.Events, EventTypeFiefDissolved)) != 0 {
		t.Errorf("events = %#v, want no dissolution", resolution.Events)
	}
}

// TestVacantFiefStaysVacantWithoutFreeNoble verifies that a fief whose owner
// has no free noble is neither dissolved nor attributed: it simply stays
// vacant, still producing and scoring, until a noble is free or its capital's
// castle falls (titres.md "Perte et vacance d'un fief").
func TestVacantFiefStaysVacantWithoutFreeNoble(t *testing.T) {
	state := foundFiefTestState(t)
	setNobleStatus(state, "N1", models.NobleStatusHostage)
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
	}}
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if len(resolution.State.Fiefs) != 1 || resolution.State.Fiefs[0].HolderNobleID != nil {
		t.Fatalf("fiefs = %#v, want the fief to stay vacant, undissolved", resolution.State.Fiefs)
	}
	if len(eventsOfType(resolution.Events, EventTypeFiefDissolved)) != 0 {
		t.Errorf("events = %#v, want no dissolution", resolution.Events)
	}
	if len(eventsOfType(resolution.Events, EventTypeFiefAutoAssigned)) != 0 {
		t.Errorf("events = %#v, want no auto-assignment", resolution.Events)
	}
}

// TestFiefAssignedSameWinterSurvivesDissolution verifies that a T A order
// attributing a vacant fief in the same winter it would otherwise be
// dissolved keeps it alive.
func TestFiefAssignedSameWinterSurvivesDissolution(t *testing.T) {
	state := foundFiefTestState(t)
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
	}}
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeAssignFief, NobleCode: "HUG", TerritoryID: "AAA"}},
	})
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if len(resolution.State.Fiefs) != 1 {
		t.Fatalf("fiefs = %#v, want the assigned fief to survive", resolution.State.Fiefs)
	}
	if resolution.State.Fiefs[0].HolderNobleID == nil || *resolution.State.Fiefs[0].HolderNobleID != "N1" {
		t.Fatalf("fief = %#v, want N1 as holder", resolution.State.Fiefs[0])
	}
	if len(eventsOfType(resolution.Events, EventTypeFiefDissolved)) != 0 {
		t.Errorf("events = %#v, want no dissolution", resolution.Events)
	}
}

// TestTransferFiefOnCapitalCapture verifies that conquering a fief's capital
// hands the whole fief, vacant, to the conqueror, including transferring
// control of every other member still held by the previous owner (titres.md
// "Contrôle et occupation", #196). transferFiefOnCapitalCapture itself is a
// no-op when called directly on a non-capital territory: fief_control_test.go
// covers the full updateTerritorialControl behavior for occupied members.
func TestTransferFiefOnCapitalCapture(t *testing.T) {
	t.Run("capital capture transfers the fief, vacant, and every other member", func(t *testing.T) {
		state := foundFiefTestState(t)
		holder := models.NobleID("N1")
		state.Fiefs = []models.Fief{{
			ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
			Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1", HolderNobleID: &holder,
		}}
		ctx := newResolutionContext(state, testBalance())
		ctx.transferFiefOnCapitalCapture("AAA", "P2")

		if len(ctx.state.Fiefs) != 1 {
			t.Fatalf("fiefs = %#v, want the fief to survive, transferred", ctx.state.Fiefs)
		}
		fief := ctx.state.Fiefs[0]
		if fief.OwnerID != "P2" {
			t.Errorf("owner = %q, want P2", fief.OwnerID)
		}
		if fief.HolderNobleID != nil {
			t.Errorf("holder = %v, want nil (vacant)", fief.HolderNobleID)
		}
		conquered := eventsOfType(ctx.events, EventTypeFiefConquered)
		if len(conquered) != 1 || conquered[0].PreviousOwnerID != "P1" || conquered[0].OwnerID != "P2" {
			t.Fatalf("conquered events = %#v", conquered)
		}
		for _, memberID := range []models.TerritoryID{"BBB", "CCC"} {
			state := ctx.state.TerritoryStates[memberID]
			if state.OwnerID == nil || *state.OwnerID != "P2" {
				t.Errorf("%s owner = %v, want P2 (transitive control)", memberID, state.OwnerID)
			}
		}
		changed := eventsOfType(ctx.events, EventTypeControlChanged)
		if len(changed) != 2 {
			t.Fatalf("control_changed events = %#v, want one per non-capital member", changed)
		}
		for _, event := range changed {
			if event.Reason != "fief_transferred" || event.OwnerID != "P2" || event.PreviousOwnerID != "P1" {
				t.Errorf("control_changed event = %#v, want reason fief_transferred P1->P2", event)
			}
		}
	})

	t.Run("non-capital territory changing hands does not transfer the fief", func(t *testing.T) {
		state := foundFiefTestState(t)
		holder := models.NobleID("N1")
		state.Fiefs = []models.Fief{{
			ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
			Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1", HolderNobleID: &holder,
		}}
		ctx := newResolutionContext(state, testBalance())
		ctx.transferFiefOnCapitalCapture("BBB", "P2")

		if len(ctx.state.Fiefs) != 1 || ctx.state.Fiefs[0].OwnerID != "P1" || ctx.state.Fiefs[0].HolderNobleID == nil {
			t.Fatalf("fiefs = %#v, want the fief unchanged", ctx.state.Fiefs)
		}
		if len(eventsOfType(ctx.events, EventTypeFiefConquered)) != 0 {
			t.Errorf("events = %#v, want no conquest", ctx.events)
		}
	})
}

// TestNeutralRevoltNeverTransfersFief verifies that a rebel (NEUTRAL) army
// dislodging the holder of a fief capital never triggers a transfer: revolts
// never take positional control (updateTerritorialControl skips NEUTRAL
// armies outright), so the capture path is never reached.
func TestNeutralRevoltNeverTransfersFief(t *testing.T) {
	state := foundFiefTestState(t)
	holder := models.NobleID("N1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1", HolderNobleID: &holder,
	}}
	placeArmyAt(state, "A9", models.NeutralPlayerID, "AAA", 5)
	ctx := newResolutionContext(state, testBalance())
	updateTerritorialControl(ctx)

	if len(ctx.state.Fiefs) != 1 || ctx.state.Fiefs[0].OwnerID != "P1" || ctx.state.Fiefs[0].HolderNobleID == nil {
		t.Fatalf("fiefs = %#v, want the fief unchanged by the neutral army", ctx.state.Fiefs)
	}
	if len(eventsOfType(ctx.events, EventTypeFiefConquered)) != 0 {
		t.Errorf("events = %#v, want no conquest", ctx.events)
	}
}

// TestDissolveFiefOnCapitalCastleLoss verifies that losing a fief capital's
// castle to pillage dissolves the fief immediately (titres.md deliberately
// chose immediate dissolution over suspending the city bonus).
func TestDissolveFiefOnCapitalCastleLoss(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "BBB", "P1")
	setTerritoryOwner(state, "CCC", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addNoble(state, "N1", "HUG", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "AAA"})
	holder := models.NobleID("N1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1", HolderNobleID: &holder,
	}}
	state.Turn = 1
	state.Season = models.SeasonSpring
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve(pillage) = %v", err)
	}
	if len(resolution.State.Fiefs) != 0 {
		t.Fatalf("fiefs = %#v, want none (dissolved)", resolution.State.Fiefs)
	}
	dissolved := eventsOfType(resolution.Events, EventTypeFiefDissolved)
	if len(dissolved) != 1 || dissolved[0].FiefID != "F1" || dissolved[0].Reason != "capital_castle_lost" {
		t.Fatalf("dissolved events = %#v", dissolved)
	}
}

// TestPlagueVacatesFiefWhenHolderDies verifies that a plague killing a
// fief's titulaire leaves the fief with its owner, vacant.
func TestPlagueVacatesFiefWhenHolderDies(t *testing.T) {
	state := effectTestState()
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setCurrentCalamity(state, models.CardKindPlague, "AAA")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB"}, OwnerID: "P1", HolderNobleID: nobleIDPtr("N1"),
	}}
	balance := testBalance()
	balance.SpecialOrders.Effects.PlagueNobleMortalityPercentage = 100
	ctx := newResolutionContext(state, balance)
	resolveSeasonEffects(ctx)

	if len(ctx.state.Fiefs) != 1 {
		t.Fatalf("fiefs = %#v, want the fief to stay, vacant", ctx.state.Fiefs)
	}
	if ctx.state.Fiefs[0].HolderNobleID != nil {
		t.Errorf("holder = %v, want nil (vacant)", ctx.state.Fiefs[0].HolderNobleID)
	}
	if ctx.state.Fiefs[0].OwnerID != "P1" {
		t.Errorf("owner = %q, want P1 (unchanged)", ctx.state.Fiefs[0].OwnerID)
	}
	vacated := eventsOfType(ctx.events, EventTypeFiefVacated)
	if len(vacated) != 1 || vacated[0].FiefID != "F1" {
		t.Fatalf("vacated events = %#v", vacated)
	}
}

// TestHostageDoesNotVacateFief verifies that capturing a titulaire (hostage
// or dungeon) has no effect on their fief: the noble still exists, only its
// status changes (titres.md).
func TestHostageDoesNotVacateFief(t *testing.T) {
	state := foundFiefTestState(t)
	setNobleStatus(state, "N1", models.NobleStatusHostage)
	holder := models.NobleID("N1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1", HolderNobleID: &holder,
	}}
	ctx := newResolutionContext(state, testBalance())
	ctx.vacateFiefsOfMissingHolders()

	if ctx.state.Fiefs[0].HolderNobleID == nil || *ctx.state.Fiefs[0].HolderNobleID != "N1" {
		t.Errorf("holder = %v, want N1 unchanged", ctx.state.Fiefs[0].HolderNobleID)
	}
	if len(eventsOfType(ctx.events, EventTypeFiefVacated)) != 0 {
		t.Errorf("events = %#v, want no vacation", ctx.events)
	}
}
