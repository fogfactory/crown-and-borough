package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSimulateVictoryClaimThenKillHandsTheFiefToTheHeir(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	holder := models.NobleID("N3")
	state.Fiefs = []models.Fief{{ID: "F1", OwnerID: "P2", HolderNobleID: &holder, Title: models.FiefTitleDuchy}}
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
	}
	balance := allianceBalance()

	simulation, err := SimulateVictory(state, balance, []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.Fiefs[0].OwnerID; got != "P1" {
		t.Fatalf("fief owner after the claimed noble died = %q, want P1", got)
	}
	if got := simulation.Projected["P1"].Score.Titles; got != simulation.Current["P1"].Score.Titles+1 {
		t.Fatalf("P1 titles = %d, want one more than %d", got, simulation.Current["P1"].Score.Titles)
	}
	if len(state.Fiefs) != 1 || state.Fiefs[0].OwnerID != "P2" {
		t.Fatal("simulation mutated the source state")
	}
}

func TestSimulateVictoryKillOfTheKingCrownsTheHeirOfAClaim(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	king := models.NobleID("N3")
	state.King = &king
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
		state.Nobles[index].Sex = models.SexMale
	}

	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.King; got == nil || *got != "N2" {
		t.Fatalf("king after the claimed king died = %v, want N2", got)
	}
	if state.King == nil || *state.King != "N3" {
		t.Fatal("simulation mutated the source state")
	}
}

func TestSimulateVictoryKillOfTheLastNobleReturnsTheFiefToTheCrown(t *testing.T) {
	state := allianceState()
	king := models.NobleID("N1")
	state.King = &king
	holder := models.NobleID("N3")
	state.Nobles = state.Nobles[:3]
	state.Fiefs = append(state.Fiefs, models.Fief{ID: "F2", OwnerID: "P2", HolderNobleID: &holder, Title: models.FiefTitleBarony})
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
	}

	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationKill, Noble: "CN3"}})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.Fiefs[1]; got.OwnerID != "P1" || !got.Crown || got.HolderNobleID != nil {
		t.Fatalf("fief after the last noble of P2 died = %#v, want a vacant crown fief of P1", got)
	}
	if state.Fiefs[1].OwnerID != "P2" || state.Fiefs[1].Crown {
		t.Fatal("simulation mutated the source state")
	}
}

func rankSimulationState() *models.GameState {
	state := allianceState()
	state.Fiefs = nil
	for index, id := range []models.NobleID{"N1", "N2", "N3", "N4"} {
		holder := id
		owner := models.PlayerID("P1")
		if index >= 2 {
			owner = "P2"
		}
		state.Fiefs = append(state.Fiefs, models.Fief{
			ID: models.FiefID("F" + string(id[1:])), OwnerID: owner, HolderNobleID: &holder,
			CapitalTerritoryID: models.TerritoryID("T" + string(id[1:])), Title: models.FiefTitleBarony,
		})
	}
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
	}
	// B (N2-N3) is older than A (N1-N4); both weigh the same.
	state.Marriages = []models.Marriage{
		{NobleA: "N1", NobleB: "N4", Turn: 2},
		{NobleA: "N2", NobleB: "N3", Turn: 1},
	}
	return state
}

func TestSimulateVictoryRankUpChangesTheActiveHead(t *testing.T) {
	state := rankSimulationState()
	balance := allianceBalance()
	head, _ := ActiveHeadMarriage(state, balance, "P1")
	if head.NobleA != "N2" {
		t.Fatalf("head before = %v, want the older marriage N2-N3", head)
	}

	simulation, err := SimulateVictory(state, balance, []SimulationAction{
		// Nobles placed above must hold an equal or higher title first.
		{Kind: SimulationRank, Fief: "T1"},
		{Kind: SimulationRank, Fief: "T3"},
		{Kind: SimulationRank, Fief: "T4"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	head, _ = ActiveHeadMarriage(simulation.State, balance, "P1")
	if head.NobleA != "N1" {
		t.Fatalf("head after the rank up = %v, want N1-N4", head)
	}
	if got := simulation.State.Fiefs[3].Title; got != models.FiefTitleCounty {
		t.Fatalf("fief title = %q, want county", got)
	}
	if state.Fiefs[3].Title != models.FiefTitleBarony {
		t.Fatal("simulation mutated the source state")
	}
}

func TestSimulateVictoryRankErrors(t *testing.T) {
	state := rankSimulationState()
	state.Fiefs[0].Title = models.FiefTitleDuchy
	for name, fief := range map[string]models.TerritoryID{"top rank": "T1", "unknown": "T9"} {
		if _, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationRank, Fief: fief}}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSimulateVictoryCrownAndDethrone(t *testing.T) {
	state := rankSimulationState()
	for index := range state.Nobles {
		state.Nobles[index].Sex = models.SexMale
	}
	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationCrown, Noble: "CN1"},
		{Kind: SimulationCrown, Noble: "CN2"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.King; got == nil || *got != "N2" {
		t.Fatalf("king = %v, want N2", got)
	}
	if state.King != nil {
		t.Fatal("simulation mutated the source state")
	}
	simulation, err = SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationCrown, Noble: "CN1"},
		{Kind: SimulationDethrone},
	})
	if err != nil || simulation.State.King != nil {
		t.Fatalf("dethrone: king = %v, err = %v", simulation.State.King, err)
	}
	if _, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationDethrone}}); err == nil {
		t.Fatal("dethroning a vacant throne should fail")
	}
}

func TestSimulateVictoryRankBlockedBySuccessionLine(t *testing.T) {
	// N1 is above N2 in the line and holds only a barony.
	_, err := SimulateVictory(rankSimulationState(), allianceBalance(), []SimulationAction{{Kind: SimulationRank, Fief: "T2"}})
	if err == nil {
		t.Fatal("expected the rank up of the second in line to be refused")
	}
}

func TestSimulateVictoryCrownRefusesIneligibleNobles(t *testing.T) {
	build := func() *models.GameState {
		state := rankSimulationState()
		for index := range state.Nobles {
			state.Nobles[index].Sex = models.SexMale
		}
		return state
	}
	king := models.NobleID("N1")
	pope := models.NobleID("N2")
	cases := map[string]struct {
		mutate func(*models.GameState)
		noble  models.NobleCode
	}{
		"woman":         {func(s *models.GameState) { s.Nobles[3].Sex = models.SexFemale }, "CN4"},
		"already king":  {func(s *models.GameState) { s.King = &king }, "CN1"},
		"pope":          {func(s *models.GameState) { s.Pope = &pope }, "CN2"},
		"bastard":       {func(s *models.GameState) { s.Nobles[2].Dignities = []models.Dignity{models.DignityBastard} }, "CN3"},
		"excommunicate": {func(s *models.GameState) { s.Excommunications = []models.Excommunication{{Noble: "N4"}} }, "CN4"},
		"unknown noble": {func(*models.GameState) {}, "CN9"},
	}
	for name, tc := range cases {
		state := build()
		tc.mutate(state)
		if _, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationCrown, Noble: tc.noble}}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSimulateVictoryCrownedHeirChangesTheHeadMarriage(t *testing.T) {
	state := allianceState()
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
		state.Nobles[index].Sex = models.SexMale
	}
	king := models.NobleID("N3")
	state.King = &king
	state.Fiefs = append(state.Fiefs,
		models.Fief{ID: "F2", OwnerID: "P2", HolderNobleID: ptrNoble("N4"), Title: models.FiefTitleDuchy},
		models.Fief{ID: "F3", OwnerID: "P2", HolderNobleID: ptrNoble("N5"), Title: models.FiefTitleCounty})
	state.Marriages = []models.Marriage{
		{NobleA: "N1", NobleB: "N3", Turn: 1},
		{NobleA: "N2", NobleB: "N4", Turn: 2},
		{NobleA: "N1", NobleB: "N5", Turn: 3},
	}
	balance := allianceBalance()

	vacant, err := SimulateVictory(state, balance, []SimulationAction{{Kind: SimulationKill, Noble: "CN3"}})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if head, _ := ActiveHeadMarriage(vacant.State, balance, "P1"); head.NobleB != "N5" {
		t.Fatalf("head with a vacant throne = %+v, want N1-N5", head)
	}

	crowned, err := SimulateVictory(state, balance, []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if head, _ := ActiveHeadMarriage(crowned.State, balance, "P1"); head.NobleA != "N2" {
		t.Fatalf("head once N2 is crowned = %+v, want N2-N4", head)
	}
	if got, want := crowned.Projected["P1"].Score.Titles, vacant.Projected["P1"].Score.Titles+1; got != want {
		t.Fatalf("P1 titles = %d, want %d (the crown)", got, want)
	}
}

func TestSimulateVictoryKillOfAUsurperRestoresTheExile(t *testing.T) {
	state := allianceState()
	// N3 of P2 holds the fief as usurper; N1 of P1 is the exile, whose title
	// counts for nothing.
	holder, exile := models.NobleID("N3"), models.NobleID("N1")
	state.Fiefs = []models.Fief{{
		ID: "F1", OwnerID: "P2", HolderNobleID: &holder, ExiledNobleID: &exile, Title: models.FiefTitleDuchy,
	}}
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
	}

	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	fief := simulation.State.Fiefs[0]
	if fief.OwnerID != "P1" || fief.HolderNobleID == nil || *fief.HolderNobleID != "N1" || fief.ExiledNobleID != nil {
		t.Fatalf("fief = %+v, want it restored to N1 of P1", fief)
	}
	if got, want := simulation.Projected["P1"].Score.Titles, simulation.Current["P1"].Score.Titles+1; got != want {
		t.Errorf("P1 titles = %d, want %d once restored", got, want)
	}
}

func TestSimulateVictoryClaimOnAFiefWithAGarrisonNobleUsurpsIt(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	holder := models.NobleID("N3")
	state.Fiefs = []models.Fief{{
		ID: "F1", OwnerID: "P2", HolderNobleID: &holder, Title: models.FiefTitleDuchy, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA"},
	}}
	// N4 of P2 garrisons the capital with an army.
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 2}}
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
		state.Nobles[index].Status = models.NobleStatusFree
	}
	setNobleLocation(state, "N4", "AAA")

	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	fief := simulation.State.Fiefs[0]
	if fief.OwnerID != "P2" || fief.HolderNobleID == nil || *fief.HolderNobleID != "N4" || fief.ExiledNobleID == nil || *fief.ExiledNobleID != "N2" || !fief.Usurped() {
		t.Fatalf("fief = %+v, want N4 of P2 as usurper and N2 exiled", fief)
	}
}

// contestedFiefState is allianceState with a third house (N6 of P3), free
// nobles and codes "C"+ID, and no fief but the one a test adds on the capital
// "AAA".
func contestedFiefState() *models.GameState {
	state := allianceState()
	state.Players = append(state.Players, models.Player{ID: "P3"})
	state.Nobles = append(state.Nobles, models.Noble{ID: "N6", OwnerID: "P3", Sex: models.SexMale})
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
		state.Nobles[index].Status = models.NobleStatusFree
	}
	state.Fiefs = nil
	return state
}

// withFief gives the capital "AAA" a fief of owner held by holder (empty for a
// vacant fief), with exile (empty for none).
func withFief(state *models.GameState, owner models.PlayerID, holder, exile models.NobleID) {
	fief := models.Fief{
		ID: "F1", OwnerID: owner, Title: models.FiefTitleDuchy, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA"},
	}
	if holder != "" {
		fief.HolderNobleID = &holder
	}
	if exile != "" {
		fief.ExiledNobleID = &exile
	}
	state.Fiefs = []models.Fief{fief}
}

func simulateFief(t *testing.T, state *models.GameState, actions ...SimulationAction) models.Fief {
	t.Helper()
	simulation, err := SimulateVictory(state, allianceBalance(), actions)
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	return simulation.State.Fiefs[0]
}

func requireFief(t *testing.T, fief models.Fief, owner models.PlayerID, holder, exile models.NobleID) {
	t.Helper()
	gotHolder, gotExile := models.NobleID(""), models.NobleID("")
	if fief.HolderNobleID != nil {
		gotHolder = *fief.HolderNobleID
	}
	if fief.ExiledNobleID != nil {
		gotExile = *fief.ExiledNobleID
	}
	if fief.OwnerID != owner || gotHolder != holder || gotExile != exile {
		t.Fatalf("fief = owner %q holder %q exile %q, want owner %q holder %q exile %q",
			fief.OwnerID, gotHolder, gotExile, owner, holder, exile)
	}
}

func requireSimulationError(t *testing.T, state *models.GameState, actions ...SimulationAction) {
	t.Helper()
	if _, err := SimulateVictory(state, allianceBalance(), actions); err == nil {
		t.Fatalf("SimulateVictory(%+v) succeeded, want an error", actions)
	}
}

func TestSimulateVictoryConquerHeldFiefUsurpsIt(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "")
	fief := simulateFief(t, state, SimulationAction{Kind: SimulationConquer, Noble: "CN1", Fief: "AAA"})
	requireFief(t, fief, "P1", "N1", "N3")
	if state.Fiefs[0].OwnerID != "P2" {
		t.Fatal("simulation mutated the source state")
	}
}

func TestSimulateVictoryConquerVacantFiefPassesVacant(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "", "")
	state.Fiefs[0].Crown = true
	fief := simulateFief(t, state, SimulationAction{Kind: SimulationConquer, Noble: "CN1", Fief: "AAA"})
	requireFief(t, fief, "P1", "", "")
	if fief.Crown {
		t.Error("a conquered fief keeps its crown mark")
	}
}

func TestSimulateVictoryConquerUsurpedFiefDropsTheUsurper(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "N2")
	fief := simulateFief(t, state, SimulationAction{Kind: SimulationConquer, Noble: "CN6", Fief: "AAA"})
	requireFief(t, fief, "P3", "N6", "N2")
}

func TestSimulateVictoryConquerByTheExilesHouseRestoresIt(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "N2")
	// N1 shares the house of the exile N2: no usurpation, a restoration.
	fief := simulateFief(t, state, SimulationAction{Kind: SimulationConquer, Noble: "CN1", Fief: "AAA"})
	requireFief(t, fief, "P1", "N2", "")
}

func TestSimulateVictoryConquerErrors(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "")
	requireSimulationError(t, state, SimulationAction{Kind: SimulationConquer, Noble: "CN4", Fief: "AAA"}) // own fief
	requireSimulationError(t, state, SimulationAction{Kind: SimulationConquer, Noble: "CN1", Fief: "ZZZ"}) // unknown fief
	requireSimulationError(t, state, SimulationAction{Kind: SimulationConquer, Noble: "NOPE", Fief: "AAA"})
	state.Nobles[0].Status = models.NobleStatusHostage
	requireSimulationError(t, state, SimulationAction{Kind: SimulationConquer, Noble: "CN1", Fief: "AAA"}) // not free
}

func TestSimulateVictoryRestoreGivesTheFiefBackToTheExile(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "N2")
	fief := simulateFief(t, state, SimulationAction{Kind: SimulationRestore, Fief: "AAA"})
	requireFief(t, fief, "P1", "N2", "")
}

func TestSimulateVictoryRestoreErrors(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "")
	requireSimulationError(t, state, SimulationAction{Kind: SimulationRestore, Fief: "AAA"}) // no exile
	requireSimulationError(t, state, SimulationAction{Kind: SimulationRestore, Fief: "ZZZ"})
	// The exile and the usurper share a house: only a death or a renunciation resolves it.
	withFief(state, "P2", "N3", "N4")
	requireSimulationError(t, state, SimulationAction{Kind: SimulationRestore, Fief: "AAA"})
}

func TestSimulateVictoryRenounceTitle(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "N2")

	// The exile abandons its title: the usurper becomes official.
	fief := simulateFief(t, state, SimulationAction{Kind: SimulationRenounceTitle, Noble: "CN2", Fief: "AAA"})
	requireFief(t, fief, "P2", "N3", "")
	// The usurper gives the title back: the exile recovers the fief.
	fief = simulateFief(t, state, SimulationAction{Kind: SimulationRenounceTitle, Noble: "CN3", Fief: "AAA"})
	requireFief(t, fief, "P1", "N2", "")

	requireSimulationError(t, state, SimulationAction{Kind: SimulationRenounceTitle, Noble: "CN1", Fief: "AAA"})
	requireSimulationError(t, state, SimulationAction{Kind: SimulationRenounceTitle, Noble: "CN2", Fief: "ZZZ"})
}

func TestSimulateVictoryRenounceClaim(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "")
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationRenounceClaim, Noble: "CN2"},
		// With the claim gone, the death leaves the fief vacant.
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if len(simulation.State.Claims) != 0 || len(simulation.State.ExpiredClaims) != 1 || simulation.State.ExpiredClaims[0].Outcome != models.ClaimRenounced {
		t.Fatalf("claims = %+v expired = %+v, want the claim renounced", simulation.State.Claims, simulation.State.ExpiredClaims)
	}
	requireFief(t, simulation.State.Fiefs[0], "P2", "", "")
	requireSimulationError(t, state, SimulationAction{Kind: SimulationRenounceClaim, Noble: "CN2"})
}

func TestSimulateVictoryKillUsurpedFiefTogetherOrAlone(t *testing.T) {
	// A claim of N6 bears on the exile N2.
	build := func() *models.GameState {
		state := contestedFiefState()
		withFief(state, "P2", "N3", "N2")
		state.Claims = []models.Claim{{Heir: "N6", Target: "N2", Turn: 1}}
		return state
	}
	// Both die in one settlement: the fief is vacant with the usurper's house and
	// the claim is not honoured.
	state := build()
	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationKill, Noble: "CN3", AlsoKill: "CN2"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	requireFief(t, simulation.State.Fiefs[0], "P2", "", "")
	if len(simulation.State.Claims) != 0 || simulation.State.ExpiredClaims[0].Outcome != models.ClaimLapsed {
		t.Errorf("claims = %+v expired = %+v, want the claim lapsed", simulation.State.Claims, simulation.State.ExpiredClaims)
	}
	if len(simulation.State.RemovedNobles) != 2 {
		t.Errorf("removed nobles = %d, want 2", len(simulation.State.RemovedNobles))
	}

	// The usurper dies alone: the exile recovers the fief.
	requireFief(t, simulateFief(t, build(), SimulationAction{Kind: SimulationKill, Noble: "CN3"}), "P1", "N2", "")
	// The exile dies alone: the heir of the claim on it is the new exile.
	requireFief(t, simulateFief(t, build(), SimulationAction{Kind: SimulationKill, Noble: "CN2"}), "P2", "N3", "N6")
	// Without a claim on it, the usurper becomes official.
	state = build()
	state.Claims = nil
	requireFief(t, simulateFief(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN2"}), "P2", "N3", "")
}

func TestSimulateVictoryKillWithAGarrisonAssumption(t *testing.T) {
	// N1 of P1 holds a claim on N3, holder of the fief of P2; N4 of P2 stands
	// on the capital with an army.
	build := func() *models.GameState {
		state := contestedFiefState()
		withFief(state, "P2", "N3", "")
		state.Claims = []models.Claim{{Heir: "N1", Target: "N3", Turn: 1}}
		state.Armies = []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 2}}
		setNobleLocation(state, "N4", "AAA")
		return state
	}
	// Real positions: the garrison noble usurps.
	requireFief(t, simulateFief(t, build(), SimulationAction{Kind: SimulationKill, Noble: "CN3"}), "P2", "N4", "N1")
	// Assuming nobody garrisons: the heir takes the title.
	requireFief(t, simulateFief(t, build(), SimulationAction{Kind: SimulationKill, Noble: "CN3", NoGarrison: true}), "P1", "N1", "")
	// Assuming a noble of a third house garrisons.
	state := build()
	state.Armies = nil
	requireFief(t, simulateFief(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3"}), "P1", "N1", "")
	requireFief(t, simulateFief(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", Garrison: "CN6"}), "P3", "N6", "N1")
	// A garrison of the heir's own house changes nothing.
	requireFief(t, simulateFief(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", Garrison: "CN2"}), "P1", "N1", "")
}

func TestSimulateVictoryKillOptionErrors(t *testing.T) {
	state := contestedFiefState()
	withFief(state, "P2", "N3", "")
	requireSimulationError(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", AlsoKill: "CN3"})
	requireSimulationError(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", AlsoKill: "NOPE"})
	requireSimulationError(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", Garrison: "CN3"})
	requireSimulationError(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", Garrison: "NOPE"})
	requireSimulationError(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", Garrison: "CN6", NoGarrison: true})
	state.Nobles[5].Status = models.NobleStatusDungeon
	requireSimulationError(t, state, SimulationAction{Kind: SimulationKill, Noble: "CN3", Garrison: "CN6"})
}
