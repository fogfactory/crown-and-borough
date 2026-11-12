package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func renounceTitleOrderOf(code models.NobleCode, capital models.TerritoryID) models.WinterOrder {
	return models.WinterOrder{ID: "O1", Type: models.WinterOrderTypeRenounceTitle, NobleCode: code, TerritoryID: capital}
}

func applyRenounce(state *models.GameState, playerID models.PlayerID, order models.WinterOrder) *resolutionContext {
	ctx := newResolutionContext(state, testBalance())
	executeWinterOrder(ctx, playerID, order, nil)
	return ctx
}

func TestRenounceClaimEndsItAndAllowsANewOneInTheSameWinter(t *testing.T) {
	state := claimTestState(t)
	secondCard := giveClaimCard(state, "P1")
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {
		claimSheetOrder("O1", "KID", "ANN"),
		{ID: "O2", Type: models.WinterOrderTypeRenounceClaim, NobleCode: "KID"},
		claimSheetOrder("O3", "KID", "ANN"),
	}})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %v, want none", reasons)
	}
	if claims := resolution.State.Claims; len(claims) != 1 || claims[0].Heir != "N3" {
		t.Errorf("claims = %+v, want the new claim of N3", claims)
	}
	expired := resolution.State.ExpiredClaims
	if len(expired) != 1 || expired[0].Outcome != models.ClaimRenounced {
		t.Fatalf("expired claims = %+v, want one renounced", expired)
	}
	// The first card went back to the discard pile; the other one is now
	// staked on KID. Both orders are free.
	deck := resolution.State.NobleDeck
	if len(deck.Discard) != 1 || len(deck.Played) != 1 || deck.Played[0].Card != secondCard || deck.Played[0].Noble != "N3" {
		t.Errorf("discard = %v, played = %+v, want one discarded card and %s staked on N3", deck.Discard, deck.Played, secondCard)
	}
	renounced := eventsOfType(resolution.Events, EventTypeTitleRenounced)
	if len(renounced) != 1 || renounced[0].Reason != "claim" || renounced[0].NobleCode != "KID" || renounced[0].SpouseNobleName != "Dame ANN" {
		t.Errorf("renounced events = %+v, want one claim renunciation", renounced)
	}
}

func TestRenounceClaimRejections(t *testing.T) {
	for name, test := range map[string]struct {
		code models.NobleCode
		want string
	}{
		"no claim":      {code: "KID", want: "no_claim_to_renounce"},
		"not own noble": {code: "LEO", want: "noble_not_owned"},
		"unknown noble": {code: "ZZZ", want: "unknown_noble"},
	} {
		t.Run(name, func(t *testing.T) {
			state := claimTestState(t)
			resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {
				{ID: "O1", Type: models.WinterOrderTypeRenounceClaim, NobleCode: test.code},
			}})
			if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != test.want {
				t.Errorf("rejections = %v, want [%s]", reasons, test.want)
			}
		})
	}
}

func TestRenounceExiledTitleMakesTheUsurperOfficial(t *testing.T) {
	state := usurpedFiefState(t)
	ctx := applyRenounce(state, "P1", renounceTitleOrderOf("HUG", "AAA"))
	requireTitles(t, ctx.state.Fiefs[0], "P2", "N2", "", false)
	renounced := eventsOfType(ctx.events, EventTypeTitleRenounced)
	if len(renounced) != 1 || renounced[0].Reason != "exiled" || renounced[0].NobleCode != "HUG" || renounced[0].OwnerID != "P2" {
		t.Errorf("events = %+v, want one exiled renunciation", ctx.events)
	}
	// No restoration is possible any more: a mere army of P1 does nothing.
	placeCurrentArmyOnly(ctx, "A1", "P1", "AAA", 3)
	updateTerritorialControl(ctx)
	requireTitles(t, ctx.state.Fiefs[0], "P2", "N2", "", false)
}

func TestRenounceUsurpedTitleRestoresTheExile(t *testing.T) {
	state := usurpedFiefState(t)
	ctx := applyRenounce(state, "P2", renounceTitleOrderOf("ANN", "AAA"))
	requireTitles(t, ctx.state.Fiefs[0], "P1", "N1", "", false)
	renounced := eventsOfType(ctx.events, EventTypeTitleRenounced)
	if len(renounced) != 1 || renounced[0].Reason != "usurped" || renounced[0].PreviousOwnerID != "P2" || renounced[0].OwnerID != "P1" {
		t.Errorf("events = %+v, want one usurped renunciation handing the fief to P1", ctx.events)
	}
}

func TestRenounceTitleRejections(t *testing.T) {
	for name, test := range map[string]struct {
		player models.PlayerID
		order  models.WinterOrder
		want   string
	}{
		"uncontested noble": {player: "P3", order: renounceTitleOrderOf("BEA", "AAA"), want: "noble_not_contested"},
		"not own noble":     {player: "P1", order: renounceTitleOrderOf("ANN", "AAA"), want: "noble_not_owned"},
		"unknown fief":      {player: "P1", order: renounceTitleOrderOf("HUG", "CCC"), want: "fief_not_found"},
		"unknown noble":     {player: "P1", order: renounceTitleOrderOf("ZZZ", "AAA"), want: "unknown_noble"},
	} {
		t.Run(name, func(t *testing.T) {
			state := usurpedFiefState(t)
			ctx := applyRenounce(state, test.player, test.order)
			requireRejected(t, ctx, "O1", test.want)
			requireTitles(t, ctx.state.Fiefs[0], "P2", "N2", "N1", true)
		})
	}
}

func TestRenounceTitleOfAnOrdinaryHolderIsRejected(t *testing.T) {
	state := usurpationState(t)
	ctx := applyRenounce(state, "P1", renounceTitleOrderOf("HUG", "AAA"))
	requireRejected(t, ctx, "O1", "noble_not_contested")
}
