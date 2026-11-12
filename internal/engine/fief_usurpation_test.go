package engine

import (
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// captureCapitalAs runs captureFiefCapital for an army of ownerID standing on
// territoryID, the way pass 1 of updateTerritorialControl does, and reports
// whether the fief changed hands.
func captureCapitalAs(ctx *resolutionContext, territoryID models.TerritoryID, ownerID models.PlayerID, control map[models.TerritoryID]models.PlayerID) bool {
	army := &models.Army{ID: "ACAP", OwnerID: ownerID, TerritoryID: territoryID, Size: 3}
	return ctx.captureFiefCapital(ctx.fiefByCapital(territoryID), army, control)
}

// usurpationState is the three-territory fief of fiefControlTestState owned by
// P1 and held by N1, with P2 and P3 each owning a noble.
func usurpationState(t *testing.T) *models.GameState {
	t.Helper()
	state := fiefControlTestState(t, nil)
	addNoble(state, "N1", "HUG", "P1", "BBB")
	addNoble(state, "N2", "ANN", "P2", "AAA")
	addNoble(state, "N3", "BEA", "P3", "CCC")
	holder := models.NobleID("N1")
	state.Fiefs[0].HolderNobleID = &holder
	return state
}

func fiefAfterControlPass(t *testing.T, state *models.GameState, armies ...models.Army) (*resolutionContext, models.Fief) {
	t.Helper()
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	for _, army := range armies {
		placeCurrentArmyOnly(ctx, army.ID, army.OwnerID, army.TerritoryID, army.Size)
	}
	updateTerritorialControl(ctx)
	return ctx, ctx.state.Fiefs[0]
}

func requireTitles(t *testing.T, fief models.Fief, owner models.PlayerID, holder, exile models.NobleID, usurped bool) {
	t.Helper()
	if fief.OwnerID != owner || fief.Usurped != usurped {
		t.Errorf("fief = %#v, want owner %s, usurped %v", fief, owner, usurped)
	}
	check := func(label string, got *models.NobleID, want models.NobleID) {
		switch {
		case want == "" && got != nil:
			t.Errorf("%s = %q, want none", label, *got)
		case want != "" && (got == nil || *got != want):
			t.Errorf("%s = %v, want %q", label, got, want)
		}
	}
	check("holder", fief.HolderNobleID, holder)
	check("exile", fief.ExiledNobleID, exile)
}

func TestConquestWithANobleUsurpsAHeldFief(t *testing.T) {
	state := usurpationState(t)
	ctx, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 5})
	requireTitles(t, fief, "P2", "N2", "N1", true)
	conquered := eventsOfType(ctx.events, EventTypeFiefConquered)
	if len(conquered) != 1 || conquered[0].Reason != "usurpation" || conquered[0].NobleID != "N2" || conquered[0].ExiledNobleID != "N1" {
		t.Fatalf("conquered events = %#v, want one usurpation of N1 by N2", conquered)
	}
	if err := ctx.state.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}
}

func TestConquestUsurperIsFirstFreeNobleInLine(t *testing.T) {
	state := usurpationState(t)
	// N6 comes after N2 in P2's line, whatever the order of the nobles: N2
	// takes the fief.
	state.Nobles = append([]models.Noble{{
		ID: "N6", Code: "ZOE", Name: "ZOE", OwnerID: "P2", LocationID: "AAA", Status: models.NobleStatusFree, Sex: models.SexMale,
	}}, state.Nobles...)
	_, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 5})
	requireTitles(t, fief, "P2", "N2", "N1", true)
}

func TestArmyWithoutNobleOccupiesACapitalWithoutConquering(t *testing.T) {
	state := usurpationState(t)
	ctx, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P3", TerritoryID: "AAA", Size: 5})
	requireTitles(t, fief, "P1", "N1", "", false)
	if got, _ := ctx.controllerNow("AAA"); got != "P1" {
		t.Errorf("AAA controller = %q, want P1 (occupied, not conquered)", got)
	}
	army := ctx.armiesByID["A1"]
	if !ctx.occupiedAgainstController("AAA", army) {
		t.Error("the capital is not occupied against its controller")
	}
	if len(eventsOfType(ctx.events, EventTypeFiefConquered)) != 0 || len(eventsOfType(ctx.events, EventTypeControlChanged)) != 0 {
		t.Errorf("events = %#v, want no conquest nor control change", ctx.events)
	}
	occupied := eventsOfType(ctx.events, EventTypeFiefMemberOccupied)
	if len(occupied) != 1 || occupied[0].Reason != "capital" || occupied[0].TerritoryID != "AAA" || occupied[0].CaptorPlayerID != "P3" {
		t.Errorf("occupied events = %#v, want one for the capital", occupied)
	}
}

func TestNobleCarriedAsHostageDoesNotConquer(t *testing.T) {
	state := usurpationState(t)
	setNobleStatus(state, "N2", models.NobleStatusHostage)
	ctx, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 5})
	requireTitles(t, fief, "P1", "N1", "", false)
	if len(eventsOfType(ctx.events, EventTypeFiefConquered)) != 0 {
		t.Errorf("events = %#v, want no conquest", ctx.events)
	}
}

func TestNobleWithoutArmyDoesNotConquer(t *testing.T) {
	state := usurpationState(t)
	_, fief := fiefAfterControlPass(t, state)
	requireTitles(t, fief, "P1", "N1", "", false)
}

func TestConquestOfAVacantFiefNeedsANobleAndStaysVacant(t *testing.T) {
	state := usurpationState(t)
	state.Fiefs[0].HolderNobleID = nil
	state.Fiefs[0].Crown = true
	// P3's army has no noble: mere occupation.
	_, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P3", TerritoryID: "AAA", Size: 5})
	requireTitles(t, fief, "P1", "", "", false)

	state = usurpationState(t)
	state.Fiefs[0].HolderNobleID = nil
	state.Fiefs[0].Crown = true
	_, fief = fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 5})
	requireTitles(t, fief, "P2", "", "", false)
	if fief.Crown {
		t.Error("conquest left the crown mark on the fief")
	}
}

func TestRetakingTheCapitalRestoresTheExile(t *testing.T) {
	state := usurpationState(t)
	// P2 holds the fief through N2, usurper; N1 of P1 is exiled.
	holder, exile := models.NobleID("N2"), models.NobleID("N1")
	state.Fiefs[0].OwnerID = "P2"
	state.Fiefs[0].HolderNobleID = &holder
	state.Fiefs[0].ExiledNobleID = &exile
	state.Fiefs[0].Usurped = true
	// A noble-less army of P1 is enough.
	ctx, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 4})
	requireTitles(t, fief, "P1", "N1", "", false)
	conquered := eventsOfType(ctx.events, EventTypeFiefConquered)
	if len(conquered) != 1 || conquered[0].Reason != "restoration" || conquered[0].NobleID != "N1" {
		t.Fatalf("conquered events = %#v, want one restoration of N1", conquered)
	}
	for _, territoryID := range []models.TerritoryID{"AAA", "BBB", "CCC"} {
		if got, _ := ctx.controllerNow(territoryID); got != "P1" {
			t.Errorf("%s controller = %q, want P1", territoryID, got)
		}
	}
}

func TestUsurpedThenConqueredDropsTheUsurperAndKeepsTheExile(t *testing.T) {
	state := usurpationState(t)
	holder, exile := models.NobleID("N2"), models.NobleID("N1")
	state.Fiefs[0].OwnerID = "P2"
	state.Fiefs[0].HolderNobleID = &holder
	state.Fiefs[0].ExiledNobleID = &exile
	state.Fiefs[0].Usurped = true
	setNobleLocation(state, "N3", "AAA")
	_, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P3", TerritoryID: "AAA", Size: 5})
	// N3 takes it; N2 loses everything; N1 stays the exile.
	requireTitles(t, fief, "P3", "N3", "N1", true)
}

func TestRetakingBySameOwnerAsUsurperDoesNothing(t *testing.T) {
	state := usurpationState(t)
	// P1 owns both the exile N1 and the usurper N4.
	addNoble(state, "N4", "DAN", "P1", "AAA")
	holder, exile := models.NobleID("N4"), models.NobleID("N1")
	state.Fiefs[0].HolderNobleID = &holder
	state.Fiefs[0].ExiledNobleID = &exile
	state.Fiefs[0].Usurped = true
	ctx, fief := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 4})
	requireTitles(t, fief, "P1", "N4", "N1", true)
	if len(eventsOfType(ctx.events, EventTypeFiefConquered)) != 0 {
		t.Errorf("events = %#v, want nothing", ctx.events)
	}
}

func TestOccupiedCapitalRefusesDepotAndInvestment(t *testing.T) {
	state := usurpationState(t)
	ctx, _ := fiefAfterControlPass(t, state, models.Army{ID: "A1", OwnerID: "P3", TerritoryID: "AAA", Size: 5})
	order := models.WinterOrder{ID: "O1", Type: models.WinterOrderTypeBuild}
	if !ctx.rejectIfOccupied("P1", order, "AAA") {
		t.Error("rejectIfOccupied = false, want the occupied capital to refuse investments")
	}
}

func setNobleLocation(state *models.GameState, id models.NobleID, territoryID models.TerritoryID) {
	for i := range state.Nobles {
		if state.Nobles[i].ID == id {
			state.Nobles[i].LocationID = territoryID
		}
	}
}

// usurpedFiefState: P2 holds the fief through the usurper N2, N1 of P1 is the
// exile, and N3 (P3) is a claim heir that may be staked on either of them.
func usurpedFiefState(t *testing.T) *models.GameState {
	t.Helper()
	state := usurpationState(t)
	holder, exile := models.NobleID("N2"), models.NobleID("N1")
	state.Fiefs[0].OwnerID = "P2"
	state.Fiefs[0].HolderNobleID = &holder
	state.Fiefs[0].ExiledNobleID = &exile
	state.Fiefs[0].Usurped = true
	return state
}

// killTogether removes the nobles in one settlement, like a plague does.
func killTogether(ctx *resolutionContext, ids ...models.NobleID) {
	for _, id := range ids {
		ctx.state.Nobles = removeNoble(ctx.state.Nobles, id)
	}
	ctx.rebuildIndexes()
	ctx.settleAfterDeaths()
}

func TestUsurperDeathRestoresTheExileAndIgnoresClaimsOnTheUsurper(t *testing.T) {
	state := usurpedFiefState(t)
	state.Claims = []models.Claim{{Heir: "N3", Target: "N2", Spouse: "N1"}}
	ctx := newResolutionContext(state, testBalance())
	killTogether(ctx, "N2")
	requireTitles(t, ctx.state.Fiefs[0], "P1", "N1", "", false)
	if len(ctx.state.Claims) != 0 || len(ctx.state.ExpiredClaims) != 1 || ctx.state.ExpiredClaims[0].Outcome != models.ClaimLapsed {
		t.Errorf("claims = %+v expired = %+v, want the claim on the usurper lapsed", ctx.state.Claims, ctx.state.ExpiredClaims)
	}
	conquered := eventsOfType(ctx.events, EventTypeFiefConquered)
	if len(conquered) != 1 || conquered[0].Reason != "usurper_died" || conquered[0].OwnerID != "P1" || conquered[0].PreviousOwnerID != "P2" {
		t.Errorf("conquered events = %+v, want one restoration to P1", conquered)
	}
	if err := ctx.state.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}
}

func TestExileDeathLegitimisesTheUsurper(t *testing.T) {
	state := usurpedFiefState(t)
	ctx := newResolutionContext(state, testBalance())
	killTogether(ctx, "N1")
	requireTitles(t, ctx.state.Fiefs[0], "P2", "N2", "", false)
	if settled := eventsOfType(ctx.events, EventTypeFiefTitleSettled); len(settled) != 1 || settled[0].Reason != "usurper_legitimised" {
		t.Errorf("events = %+v, want one usurper_legitimised", ctx.events)
	}
}

func TestExileDeathWithAClaimMakesTheHeirTheNewExile(t *testing.T) {
	state := usurpedFiefState(t)
	state.Claims = []models.Claim{{Heir: "N3", Target: "N1", Spouse: "N4"}}
	ctx := newResolutionContext(state, testBalance())
	killTogether(ctx, "N1")
	requireTitles(t, ctx.state.Fiefs[0], "P2", "N2", "N3", true)
	if len(ctx.state.ExpiredClaims) != 1 || ctx.state.ExpiredClaims[0].Outcome != models.ClaimHonoured {
		t.Errorf("expired claims = %+v, want the claim honoured", ctx.state.ExpiredClaims)
	}
	// The new exile is restorable by its own owner's mere army.
	placeCurrentArmyOnly(ctx, "A1", "P3", "AAA", 3)
	updateTerritorialControl(ctx)
	requireTitles(t, ctx.state.Fiefs[0], "P3", "N3", "", false)
}

func TestBothDeadLeaveTheFiefVacantWithItsOwnerEvenWithAClaimOnTheExile(t *testing.T) {
	state := usurpedFiefState(t)
	state.Claims = []models.Claim{{Heir: "N3", Target: "N1", Spouse: "N4"}}
	ctx := newResolutionContext(state, testBalance())
	killTogether(ctx, "N1", "N2")
	requireTitles(t, ctx.state.Fiefs[0], "P2", "", "", false)
	if len(eventsOfType(ctx.events, EventTypeFiefVacated)) != 1 {
		t.Errorf("events = %+v, want one fief_vacated", ctx.events)
	}
	if len(ctx.state.ExpiredClaims) != 1 || ctx.state.ExpiredClaims[0].Outcome != models.ClaimLapsed {
		t.Errorf("expired claims = %+v, want the claim not honoured", ctx.state.ExpiredClaims)
	}
}

func TestBothDeadFiefOfAPlayerWithoutNobleReturnsToTheCrown(t *testing.T) {
	state := usurpedFiefState(t)
	king := models.NobleID("N3")
	state.King = &king
	ctx := newResolutionContext(state, testBalance())
	killTogether(ctx, "N1", "N2")
	if fief := ctx.state.Fiefs[0]; !fief.Crown || fief.OwnerID != "P3" || fief.HolderNobleID != nil {
		t.Errorf("fief = %+v, want a vacant crown fief of P3", fief)
	}
}

// capitalAssaultState: P1 holds the fief {AAA, CCC, DDD} through N1 at CCC,
// and P2's army A1 stands on BBB, next to the capital AAA, with the noble N2
// of P2 (when withNoble) attacking AAA.
func capitalAssaultState(t *testing.T, withNoble bool) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA", "DDD"),
			territory("DDD", "DDD", "CCC"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: 4}},
	)
	keepTestArmiesSupplied(state)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addNoble(state, "N1", "HUG", "P1", "CCC")
	holder := models.NobleID("N1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "CCC", "DDD"}, OwnerID: "P1", HolderNobleID: &holder,
	}}
	chainNoble := models.NobleID("N2")
	addNoble(state, "N2", "ANN", "P2", "BBB")
	if !withNoble {
		// The only noble of P2 stays behind, far from the army.
		setNobleLocation(state, "N2", "DDD")
	}
	addChain(t, state, "A1", chainNoble, models.Order{
		Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"},
	})
	return state
}

func TestResolveArmyArrivingWithANobleConquersTheCapital(t *testing.T) {
	state := capitalAssaultState(t, true)
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	requireTitles(t, resolution.State.Fiefs[0], "P2", "N2", "N1", true)
	if err := resolution.State.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}
}

func TestResolveArmyArrivingWithoutANobleOnlyOccupiesTheCapital(t *testing.T) {
	state := capitalAssaultState(t, false)
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
		t.Fatalf("A1 = %+v, want it on the capital AAA", army)
	}
	requireTitles(t, resolution.State.Fiefs[0], "P1", "N1", "", false)
	occupied := eventsOfType(resolution.Events, EventTypeFiefMemberOccupied)
	if len(occupied) != 1 || occupied[0].Reason != "capital" {
		t.Errorf("occupied events = %+v, want the capital occupied", occupied)
	}
}

func TestNobleLessArmyAlreadyOnTheCapitalRestoresANewExileAtTheNextControlPass(t *testing.T) {
	state := usurpedFiefState(t)
	state.Claims = []models.Claim{{Heir: "N3", Target: "N1", Spouse: "N4"}}
	ctx := newResolutionContext(state, testBalance())
	// P3's army, without any noble, stands on the capital before the exile
	// dies: the claim only makes N3 the new exile, nothing is restored yet.
	placeCurrentArmyOnly(ctx, "A1", "P3", "AAA", 3)
	setNobleLocation(ctx.state, "N3", "CCC")
	killTogether(ctx, "N1")
	requireTitles(t, ctx.state.Fiefs[0], "P2", "N2", "N3", true)

	updateTerritorialControl(ctx)
	requireTitles(t, ctx.state.Fiefs[0], "P3", "N3", "", false)
	conquered := eventsOfType(ctx.events, EventTypeFiefConquered)
	if len(conquered) != 1 || conquered[0].Reason != "restoration" {
		t.Errorf("conquered events = %+v, want one restoration", conquered)
	}
}

// sameOwnerUsurpedState: P1 owns both the usurper N4 and the exile N1, and the
// castle on the fief's capital AAA is P1's capital.
func sameOwnerUsurpedState(t *testing.T) *models.GameState {
	t.Helper()
	state := usurpationState(t)
	addNoble(state, "N4", "DAN", "P1", "BBB")
	holder, exile := models.NobleID("N4"), models.NobleID("N1")
	state.Fiefs[0].HolderNobleID = &holder
	state.Fiefs[0].ExiledNobleID = &exile
	state.Fiefs[0].Usurped = true
	capital := models.InfraID("I1")
	state.Players[0].CapitalCastleID = &capital
	return state
}

func TestUsurperDeathKeepsTheCapitalWhenBothTitlesBelongToTheSamePlayer(t *testing.T) {
	ctx := newResolutionContext(sameOwnerUsurpedState(t), testBalance())
	killTogether(ctx, "N4")
	requireTitles(t, ctx.state.Fiefs[0], "P1", "N1", "", false)
	if ctx.state.Players[0].CapitalCastleID == nil {
		t.Error("P1 lost its capital although the fief never changed owner")
	}
}

func TestRenouncingTheUsurpedTitleKeepsTheCapitalWhenBothTitlesBelongToTheSamePlayer(t *testing.T) {
	ctx := applyRenounce(sameOwnerUsurpedState(t), "P1", renounceTitleOrderOf("DAN", "AAA"))
	requireTitles(t, ctx.state.Fiefs[0], "P1", "N1", "", false)
	if ctx.state.Players[0].CapitalCastleID == nil {
		t.Error("P1 lost its capital although the fief never changed owner")
	}
}

func TestRenouncingTheExiledTitleLeavesLaterClaimsOnTheExileWithoutEffect(t *testing.T) {
	state := usurpedFiefState(t)
	state.Claims = []models.Claim{{Heir: "N3", Target: "N1", Spouse: "N4"}}
	ctx := applyRenounce(state, "P1", renounceTitleOrderOf("HUG", "AAA"))
	killTogether(ctx, "N1")
	requireTitles(t, ctx.state.Fiefs[0], "P2", "N2", "", false)
}

func TestRenouncedTitleBlocksLaterTitleOrdersOfTheSameSheet(t *testing.T) {
	// The exile gives up the title, then tries to raise the fief.
	state := usurpedFiefState(t)
	ctx := newResolutionContext(state, testBalance())
	executeWinterOrder(ctx, "P1", renounceTitleOrderOf("HUG", "AAA"), nil)
	rank := models.WinterOrder{ID: "O2", Type: models.WinterOrderTypeChangeRank, NobleCode: "HUG", TerritoryID: "AAA"}
	annex := models.WinterOrder{ID: "O3", Type: models.WinterOrderTypeAnnexTerritory, NobleCode: "HUG", TerritoryID: "AAA", TargetID: "BBB"}
	executeWinterOrder(ctx, "P1", rank, nil)
	executeWinterOrder(ctx, "P1", annex, nil)
	if reasons := rejectionReasons(ctx.events); len(reasons) != 2 {
		t.Errorf("rejections = %v, want the rank and annex orders refused", reasons)
	}

	// The usurper gives the fief back, then tries the same.
	state = usurpedFiefState(t)
	ctx = newResolutionContext(state, testBalance())
	executeWinterOrder(ctx, "P2", renounceTitleOrderOf("ANN", "AAA"), nil)
	rank.NobleCode, annex.NobleCode = "ANN", "ANN"
	executeWinterOrder(ctx, "P2", rank, nil)
	executeWinterOrder(ctx, "P2", annex, nil)
	if reasons := rejectionReasons(ctx.events); len(reasons) != 2 {
		t.Errorf("rejections = %v, want the rank and annex orders refused", reasons)
	}
}

func TestTwoArmiesOnTheSameCapitalResolveInArmyIdOrder(t *testing.T) {
	// A state the engine never produces (one army per territory), kept
	// deterministic anyway: the second conquest drops the first usurper.
	state := usurpationState(t)
	setNobleLocation(state, "N3", "AAA")
	ctx := newResolutionContext(state, testBalance())
	placeCurrentArmyOnly(ctx, "A1", "P2", "AAA", 3)
	placeCurrentArmyOnly(ctx, "A2", "P3", "AAA", 3)
	updateTerritorialControl(ctx)
	requireTitles(t, ctx.state.Fiefs[0], "P3", "N3", "N1", true)
}

func TestClaimUsurpationByAGarrisonExilesAMaskedEonHeirUnderItsPublicIdentity(t *testing.T) {
	state := claimDeathState(t)
	for i := range state.Nobles {
		if state.Nobles[i].ID == "N3" {
			state.Nobles[i].SecretCode, state.Nobles[i].SecretName = "ZEL", "Zelie"
			state.Nobles[i].Dignities = []models.Dignity{models.DignityChevalierDEon}
		}
	}
	addNoble(state, "N5", "GAR", "P2", "AAA")
	ctx := killN2(t, state, models.Army{ID: "G1", OwnerID: "P2", TerritoryID: "AAA", Size: 3})
	requireTitles(t, ctx.state.Fiefs[0], "P2", "N5", "N3", true)
	events := eventsOfType(ctx.events, EventTypeFiefConquered)
	if len(events) != 1 || events[0].ExiledNobleCode != "ELE" || strings.Contains(events[0].ExiledNobleName, "Zelie") {
		t.Errorf("events = %+v, want the exile named by its public identity", events)
	}
}
