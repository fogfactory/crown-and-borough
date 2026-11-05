package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// kingDeathState is claimDeathState with N2 (P2), the lord on AAA that the
// plague kills, wearing the crown. N3 (P1) holds a claim on him.
func kingDeathState(t *testing.T) *models.GameState {
	t.Helper()
	state := claimDeathState(t)
	king := models.NobleID("N2")
	state.King = &king
	return state
}

func setSex(state *models.GameState, id models.NobleID, sex models.Sex) {
	for index := range state.Nobles {
		if state.Nobles[index].ID == id {
			state.Nobles[index].Sex = sex
		}
	}
}

func requireKing(t *testing.T, state *models.GameState, want models.NobleID) {
	t.Helper()
	switch {
	case want == "" && state.King != nil:
		t.Fatalf("king = %q, want a vacant throne", *state.King)
	case want != "" && (state.King == nil || *state.King != want):
		t.Fatalf("king = %v, want %q", state.King, want)
	}
}

func TestCrownPassesToTheHeirOfAClaimOnTheKing(t *testing.T) {
	ctx := resolveClaimDeath(kingDeathState(t))
	requireKing(t, ctx.state, "N3")
	if fief := ctx.state.Fiefs[0]; fief.HolderNobleID == nil || *fief.HolderNobleID != "N3" {
		t.Errorf("fief = %+v, want it held by the crowned heir too", fief)
	}
	events := eventsOfType(ctx.events, EventTypeInvestiture)
	if len(events) != 1 || events[0].Reason != "claim" || events[0].NobleCode != "ELE" || events[0].Election.Kind != models.ElectionKing {
		t.Errorf("investitures = %+v, want one royal claim investiture of ELE", events)
	}
	validateTestState(t, ctx.state)
}

func TestCrownPassesToTheHeirOfAClaimOnTheQueen(t *testing.T) {
	state := kingDeathState(t)
	addNoble(state, "N4", "QUE", "P1", "BBB")
	setSex(state, "N4", models.SexFemale)
	addNoble(state, "N5", "RAL", "P2", "BBB")
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N4"}}
	// The heir of the queen's house-mate claims the queen; the heir N3 on the
	// king is a woman and cannot reign.
	setSex(state, "N3", models.SexFemale)
	state.Claims = []models.Claim{
		{Heir: "N3", Target: "N2", Spouse: "N4", Turn: 0},
		{Heir: "N5", Target: "N4", Spouse: "N2", Turn: 1},
	}
	ctx := resolveClaimDeath(state)
	requireKing(t, ctx.state, "N5")
	if fief := ctx.state.Fiefs[0]; fief.HolderNobleID == nil || *fief.HolderNobleID != "N3" {
		t.Errorf("fief = %+v, want it kept by the woman who claimed the king", fief)
	}
	validateTestState(t, ctx.state)
}

func TestClaimsOnTheCoupleAreRankedTogether(t *testing.T) {
	state := kingDeathState(t)
	addNoble(state, "N4", "QUE", "P1", "BBB")
	setSex(state, "N4", models.SexFemale)
	addNoble(state, "N5", "RAL", "P2", "BBB")
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N4"}}
	state.Claims = []models.Claim{
		{Heir: "N3", Target: "N2", Spouse: "N4", Turn: 1},
		{Heir: "N5", Target: "N4", Spouse: "N2", Turn: 0},
	}
	ctx := resolveClaimDeath(state)
	requireKing(t, ctx.state, "N5")
}

func TestCrownSkipsIneligibleHeirsInTheRanking(t *testing.T) {
	state := kingDeathState(t)
	addNoble(state, "N4", "LEO", "P1", "BBB")
	setSex(state, "N3", models.SexFemale)
	state.Claims = []models.Claim{
		{Heir: "N3", Target: "N2", Spouse: "N1", Turn: 0},
		{Heir: "N9", Target: "N2", Spouse: "N1", Turn: 0},
		{Heir: "N4", Target: "N2", Spouse: "N1", Turn: 1},
	}
	ctx := resolveClaimDeath(state)
	requireKing(t, ctx.state, "N4")
}

func TestThroneStaysVacantWithoutAnEligibleHeir(t *testing.T) {
	for name, mutate := range map[string]func(*models.GameState){
		"no claim":         func(state *models.GameState) { state.Claims = nil },
		"heir is a woman":  func(state *models.GameState) { setSex(state, "N3", models.SexFemale) },
		"heir is the pope": func(state *models.GameState) { pope := models.NobleID("N3"); state.Pope = &pope },
		"claim on another noble": func(state *models.GameState) {
			addNoble(state, "N4", "LEO", "P2", "BBB")
			state.Claims = []models.Claim{{Heir: "N4", Target: "N1", Spouse: "N3"}}
		},
		"heir is a bastard": func(state *models.GameState) { state.Nobles[2].Dignities = []models.Dignity{models.DignityBastard} },
	} {
		t.Run(name, func(t *testing.T) {
			state := kingDeathState(t)
			mutate(state)
			ctx := resolveClaimDeath(state)
			requireKing(t, ctx.state, "")
			if events := eventsOfType(ctx.events, EventTypeInvestiture); len(events) != 0 {
				t.Errorf("investitures = %+v, want none", events)
			}
		})
	}
}

func TestExecutedKingPassesTheCrownToo(t *testing.T) {
	state := kingDeathState(t)
	ctx := newResolutionContext(state, testBalance())
	ctx.executeNoble(*ctx.noblesByID["N2"])
	requireKing(t, ctx.state, "N3")
	validateTestState(t, ctx.state)
}

func TestAnotherNoblesDeathDoesNotTouchTheCrown(t *testing.T) {
	state := kingDeathState(t)
	king := models.NobleID("N1")
	state.King = &king
	ctx := resolveClaimDeath(state)
	requireKing(t, ctx.state, "N1")
}

func expiredOutcomes(state *models.GameState) map[models.NobleID]models.ClaimOutcome {
	outcomes := make(map[models.NobleID]models.ClaimOutcome, len(state.ExpiredClaims))
	for _, expired := range state.ExpiredClaims {
		outcomes[expired.Heir] = expired.Outcome
	}
	return outcomes
}

func TestServedClaimsAreKeptAsHonouredAndTheOthersAsLapsed(t *testing.T) {
	state := kingDeathState(t)
	addNoble(state, "N4", "QUE", "P1", "BBB")
	setSex(state, "N4", models.SexFemale)
	addNoble(state, "N5", "RAL", "P2", "BBB")
	addNoble(state, "N6", "GUY", "P1", "BBB")
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N4"}}
	// N3 (a woman) gets the fief, N5 (claim on the queen) the crown, N6 is
	// served by nothing: its claim comes after N3's on the same fief.
	setSex(state, "N3", models.SexFemale)
	state.Claims = []models.Claim{
		{Heir: "N3", Target: "N2", Spouse: "N4", Turn: 0},
		{Heir: "N5", Target: "N4", Spouse: "N2", Turn: 1},
		{Heir: "N6", Target: "N2", Spouse: "N4", Turn: 1},
	}
	ctx := resolveClaimDeath(state)
	if len(ctx.state.Claims) != 0 {
		t.Errorf("claims = %+v, want none left alive", ctx.state.Claims)
	}
	want := map[models.NobleID]models.ClaimOutcome{"N3": models.ClaimHonoured, "N5": models.ClaimHonoured, "N6": models.ClaimLapsed}
	got := expiredOutcomes(ctx.state)
	if len(got) != len(want) {
		t.Fatalf("expired = %+v, want %+v", ctx.state.ExpiredClaims, want)
	}
	for heir, outcome := range want {
		if got[heir] != outcome {
			t.Errorf("claim of %s = %q, want %q", heir, got[heir], outcome)
		}
	}
	for _, expired := range ctx.state.ExpiredClaims {
		if expired.EndedTurn != ctx.state.Turn {
			t.Errorf("expired %+v: ended turn, want %d", expired, ctx.state.Turn)
		}
	}
	validateTestState(t, ctx.state)
}

func TestWidowOfAKingDiedByPlagueStaysQueen(t *testing.T) {
	state := kingDeathState(t)
	addNoble(state, "N4", "QUE", "P1", "BBB")
	setSex(state, "N4", models.SexFemale)
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N4"}}
	// No heir: the throne stays vacant, the widow keeps the queen's title.
	state.Claims = nil
	ctx := resolveClaimDeath(state)
	requireKing(t, ctx.state, "")
	if !ctx.state.IsQueen("N4") || !ctx.state.IsDowagerQueen("N4") {
		t.Fatalf("queen = %v, want N4 as a dowager queen", ctx.state.Queen)
	}
	validateTestState(t, ctx.state)
}
