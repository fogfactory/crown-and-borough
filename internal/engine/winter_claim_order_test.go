package engine

import (
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// claimTestState: P1's man HUG (N1) is married to P2's woman ANN (N2) since
// turn 2. P1's noble KID (N3, placed turn 3) and P2's noble LEO (N4, placed
// turn 3) were placed during the marriage. The state is at turn 5.
func claimTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := deckOrdersState(t)
	state.Turn = 12
	addNoble(state, "N1", "HUG", "P1", "AAA")
	addNoble(state, "N2", "ANN", "P2", "AAA")
	addNoble(state, "N3", "KID", "P1", "AAA")
	addNoble(state, "N4", "LEO", "P2", "AAA")
	state.Nobles[1].Sex = models.SexFemale
	state.Nobles[2].PlacedTurn = 8
	state.Nobles[3].PlacedTurn = 8
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N2", Turn: 8}}
	giveClaimCard(state, "P1")
	giveClaimCard(state, "P2")
	return state
}

func giveClaimCard(state *models.GameState, playerID models.PlayerID) models.NobleCardID {
	return giveCard(state, playerID, models.NobleCard{Kind: models.NobleCardKindClaim, Code: models.ClaimCardCode})
}

func claimSheetOrder(id models.OrderID, heir, target models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeClaim, NobleCode: heir, SpouseCode: target}
}

func TestClaimOrderRecordsClaim(t *testing.T) {
	state := claimTestState(t)
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {claimSheetOrder("O1", "KID", "ANN")}})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %v, want none", reasons)
	}
	want := models.Claim{Heir: "N3", Target: "N2", Spouse: "N1", Turn: 12}
	if claims := resolution.State.Claims; len(claims) != 1 || claims[0] != want {
		t.Errorf("claims = %+v, want [%+v]", claims, want)
	}
	if events := eventsOfType(resolution.Events, EventTypeClaim); len(events) != 1 || events[0].NobleCode != "KID" || events[0].SpouseNobleCode != "ANN" {
		t.Errorf("claim events = %+v, want KID claiming ANN", events)
	}
	if len(state.Claims) != 0 {
		t.Error("ResolveWinter mutated its input claims")
	}
}

func TestClaimOrderRejections(t *testing.T) {
	for name, test := range map[string]struct {
		prepare func(*models.GameState)
		heir    models.NobleCode
		target  models.NobleCode
		want    string
	}{
		"heir placed before the marriage": {prepare: func(s *models.GameState) { s.Nobles[2].PlacedTurn = 4 }, heir: "KID", target: "ANN", want: "claim_requires_marriage"},
		"starting noble":                  {prepare: func(s *models.GameState) { s.Nobles[2].PlacedTurn = 0 }, heir: "KID", target: "ANN", want: "claim_requires_marriage"},
		"heir placed after the spouse died": {prepare: func(s *models.GameState) {
			s.RemovedNobles = []models.RemovedNoble{{ID: "N9", Code: "OLD", Name: "Old", Sex: models.SexMale, OwnerID: "P1", Cause: models.DeathCauseNatural, Turn: 4}}
			s.Marriages = []models.Marriage{{NobleA: "N9", NobleB: "N2", Turn: 0}}
		}, heir: "KID", target: "ANN", want: "claim_requires_marriage"},
		"heir not owned":   {prepare: func(*models.GameState) {}, heir: "LEO", target: "ANN", want: "noble_not_owned"},
		"own target":       {prepare: func(*models.GameState) {}, heir: "KID", target: "HUG", want: "claim_on_own_noble"},
		"unmarried target": {prepare: func(*models.GameState) {}, heir: "KID", target: "LEO", want: "claim_requires_marriage"},
		"bastard heir":     {prepare: func(s *models.GameState) { s.Nobles[2].Dignities = []models.Dignity{models.DignityBastard} }, heir: "KID", target: "ANN", want: "claim_by_bastard"},
		"heir already claims": {prepare: func(s *models.GameState) {
			s.Claims = []models.Claim{{Heir: "N3", Target: "N4", Spouse: "N1", Turn: 8}}
		}, heir: "KID", target: "ANN", want: "claim_already_staked"},
	} {
		t.Run(name, func(t *testing.T) {
			state := claimTestState(t)
			test.prepare(state)
			before := len(state.Claims)
			resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {claimSheetOrder("O1", test.heir, test.target)}})
			if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != test.want {
				t.Errorf("rejections = %v, want %s", reasons, test.want)
			}
			if len(resolution.State.Claims) != before {
				t.Errorf("claims = %+v, want unchanged", resolution.State.Claims)
			}
		})
	}
}

func TestClaimsOnSameCoupleStackWithWifeFamilyFirst(t *testing.T) {
	// Both families claim the same couple the same winter: both claims stand.
	for name, sheets := range map[string]map[models.PlayerID][]models.WinterOrder{
		"husband first": {"P1": {claimSheetOrder("O1", "KID", "ANN")}, "P2": {claimSheetOrder("O1", "LEO", "HUG")}},
		"wife first":    {"P2": {claimSheetOrder("O1", "LEO", "HUG")}, "P1": {claimSheetOrder("O1", "KID", "ANN")}},
	} {
		t.Run(name, func(t *testing.T) {
			resolution := resolveNobleDeckWinter(t, claimTestState(t), sheets)
			if reasons := rejectionReasons(resolution.Events); len(reasons) != 0 {
				t.Fatalf("rejections = %v, want none", reasons)
			}
			if len(resolution.State.Claims) != 2 {
				t.Fatalf("claims = %+v, want both claims stacked", resolution.State.Claims)
			}
			// ANN and HUG each have one claim on them; the wife's family (P2,
			// LEO on HUG) ranks first on its own target, so check the flag.
			for _, claim := range resolution.State.Claims {
				if want := claim.Heir == "N4"; claim.WifeSide != want {
					t.Errorf("claim %+v: WifeSide = %v, want %v", claim, claim.WifeSide, want)
				}
			}
		})
	}
}

func TestClaimsOnSameNobleRankByAgeThenWifeFamily(t *testing.T) {
	g := models.NewGameState()
	g.Claims = []models.Claim{
		{Heir: "N5", Target: "N2", Turn: 12},
		{Heir: "N6", Target: "N2", Turn: 12, WifeSide: true},
		{Heir: "N7", Target: "N2", Turn: 8},
		{Heir: "N8", Target: "N9", Turn: 4},
	}
	var heirs []models.NobleID
	for _, claim := range g.ClaimsOn("N2") {
		heirs = append(heirs, claim.Heir)
	}
	if want := []models.NobleID{"N7", "N6", "N5"}; !slices.Equal(heirs, want) {
		t.Errorf("claim ranking = %v, want %v", heirs, want)
	}
}

func TestBastardDignityOnHeirVoidsClaimButNotOnParent(t *testing.T) {
	state := claimTestState(t)
	state.Claims = []models.Claim{{Heir: "N3", Target: "N2", Spouse: "N1", Turn: 8}}
	giveDignityCard(state, "P1")
	giveDignityCard(state, "P1")
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {
		{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "HUG", CardCode: "BAS"},
	}})
	if len(resolution.State.Claims) != 1 {
		t.Fatalf("claims = %+v, want the claim kept when its spouse becomes a bastard", resolution.State.Claims)
	}
	resolution = resolveNobleDeckWinter(t, resolution.State, map[models.PlayerID][]models.WinterOrder{"P1": {
		{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "KID", CardCode: "BAS"},
	}})
	if len(resolution.State.Claims) != 0 {
		t.Errorf("claims = %+v, want the claim voided by the heir's dignity", resolution.State.Claims)
	}
}

func TestMarriageCoveringBounds(t *testing.T) {
	state := claimTestState(t)
	state.RemovedNobles = []models.RemovedNoble{{ID: "N8", Code: "DED", Name: "Ded", Sex: models.SexMale, OwnerID: "P1", Cause: models.DeathCauseNatural, Turn: 8}}
	state.Marriages = nil
	state.Marriages = append(state.Marriages, models.Marriage{NobleA: "N8", NobleB: "N4", Turn: 4})
	for turn, want := range map[int]bool{0: false, 4: true, 8: true, 12: false} {
		if _, found := state.MarriageCovering("N4", "P1", turn); found != want {
			t.Errorf("turn %d covered = %v, want %v", turn, found, want)
		}
	}
}

func claimDeathState(t *testing.T) *models.GameState {
	t.Helper()
	state := effectTestState()
	state.Territories = append(state.Territories, models.Territory{ID: "CCC", Name: "CCC", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"BBB"}})
	state.Territories[1].Adjacencies = append(state.Territories[1].Adjacencies, "CCC")
	state.TerritoryStates["CCC"] = models.TerritoryState{}
	// One plague region per territory: only the lord on AAA dies.
	state.Regions = []models.Region{
		{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}},
		{ID: "BBB", Seed: "BBB", Territories: []models.TerritoryID{"BBB", "CCC"}},
	}
	addNoble(state, "N1", "HUG", "P1", "BBB")
	addNoble(state, "N2", "ANN", "P2", "AAA")
	addNoble(state, "N3", "ELE", "P1", "BBB")
	holder := models.NobleID("N2")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P2", HolderNobleID: &holder,
	}}
	state.Claims = []models.Claim{{Heir: "N3", Target: "N2", Spouse: "N1"}}
	setCurrentCalamity(state, models.CardKindPlague, "AAA")
	return state
}

func resolveClaimDeath(state *models.GameState) *resolutionContext {
	balance := testBalance()
	balance.SpecialOrders.Effects.PlagueNobleMortalityPercentage = 100
	ctx := newResolutionContext(state, balance)
	resolveSeasonEffects(ctx)
	return ctx
}

func TestClaimedFiefPassesToHeirWhenTargetDies(t *testing.T) {
	ctx := resolveClaimDeath(claimDeathState(t))
	if len(ctx.state.Nobles) != 2 {
		t.Fatalf("nobles = %+v, want N2 dead only", ctx.state.Nobles)
	}
	fief := ctx.state.Fiefs[0]
	if fief.OwnerID != "P1" || fief.HolderNobleID == nil || *fief.HolderNobleID != "N3" {
		t.Errorf("fief = %+v, want it owned by P1 and held by the heir N3", fief)
	}
	if len(ctx.state.Claims) != 0 {
		t.Errorf("claims = %+v, want the honoured claim settled", ctx.state.Claims)
	}
	events := eventsOfType(ctx.events, EventTypeFiefConquered)
	if len(events) != 1 || events[0].Reason != "claim" || events[0].NobleCode != "ELE" {
		t.Errorf("fief events = %+v, want one claim transfer to ELE", events)
	}
	validateTestState(t, ctx.state)
}

func TestClaimOnDeadTargetWithoutFiefIsDropped(t *testing.T) {
	state := claimDeathState(t)
	state.Fiefs = nil
	ctx := resolveClaimDeath(state)
	if len(ctx.state.Claims) != 0 {
		t.Errorf("claims = %+v, want none", ctx.state.Claims)
	}
}

func TestVacantFiefWithoutClaimStaysVacant(t *testing.T) {
	state := claimDeathState(t)
	state.Claims = nil
	ctx := resolveClaimDeath(state)
	if fief := ctx.state.Fiefs[0]; fief.OwnerID != "P2" || fief.HolderNobleID != nil {
		t.Errorf("fief = %+v, want it vacant with P2", fief)
	}
}

func TestStackedClaimFallsBackToNextLivingHeir(t *testing.T) {
	state := claimDeathState(t)
	addNoble(state, "N4", "LEO", "P1", "BBB")
	// The older claim's heir is dead (not in state): the next one takes the fief.
	state.Claims = []models.Claim{
		{Heir: "N9", Target: "N2", Spouse: "N1", Turn: 4},
		{Heir: "N4", Target: "N2", Spouse: "N1", Turn: 8},
		{Heir: "N3", Target: "N2", Spouse: "N1", Turn: 12},
	}
	ctx := newResolutionContext(state, testBalance())
	if heir := ctx.claimHeirOf("N2"); heir == nil || heir.ID != "N4" {
		t.Errorf("heir = %+v, want N4, the oldest living claim", heir)
	}
}

func TestClaimCardIsConsumedAndReturnsToDiscard(t *testing.T) {
	state := claimTestState(t)
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {claimSheetOrder("O1", "KID", "ANN")}})
	deck := resolution.State.NobleDeck
	if len(deck.Hands["P1"]) != 0 || len(deck.Played) != 1 || deck.Played[0].Noble != "N3" {
		t.Fatalf("hand = %v, played = %v, want the claim card played on the heir", deck.Hands["P1"], deck.Played)
	}
	// The heir becomes a bastard: the claim is void and the card is discarded.
	giveDignityCard(resolution.State, "P1")
	next := resolution.State
	next.Turn += 4
	voided := resolveNobleDeckWinter(t, next, map[models.PlayerID][]models.WinterOrder{"P1": {
		{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "KID", CardCode: "BAS"},
	}})
	deck = voided.State.NobleDeck
	if len(voided.State.Claims) != 0 || len(deck.Played) != 1 || len(deck.Discard) != 1 {
		t.Errorf("claims = %v, played = %v, discard = %v, want the claim card discarded", voided.State.Claims, deck.Played, deck.Discard)
	}
}

func TestClaimWithoutCardIsRejected(t *testing.T) {
	state := claimTestState(t)
	state.NobleDeck.Discard = append(state.NobleDeck.Discard, state.NobleDeck.Hands["P1"]...)
	state.NobleDeck.Hands["P1"] = nil
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {claimSheetOrder("O1", "KID", "ANN")}})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != "card_not_in_hand" {
		t.Errorf("rejections = %v, want card_not_in_hand", reasons)
	}
	if len(resolution.State.Claims) != 0 {
		t.Errorf("claims = %+v, want none", resolution.State.Claims)
	}
}

func TestClaimCardPlayedWhenTargetDiesGoesToDiscard(t *testing.T) {
	state := claimDeathState(t)
	ensureNobleDeck(state)
	cardID := giveClaimCard(state, "P1")
	state.NobleDeck.Hands["P1"] = nil
	state.NobleDeck.Played = []models.NobleCardPlay{{Card: cardID, Noble: "N3"}}
	ctx := resolveClaimDeath(state)
	if len(ctx.state.NobleDeck.Played) != 0 || len(ctx.state.NobleDeck.Discard) != 1 {
		t.Errorf("played = %v, discard = %v, want the card back on the discard pile", ctx.state.NobleDeck.Played, ctx.state.NobleDeck.Discard)
	}
	validateTestState(t, ctx.state)
}

// actionSeasonClaimState is claimTestState moved to a spring turn.
func actionSeasonClaimState(t *testing.T) *models.GameState {
	t.Helper()
	state := claimTestState(t)
	state.Turn = 13
	state.Season = models.SeasonSpring
	return state
}

func TestCharacterCardOrdersCanBePlayedInActionSeasons(t *testing.T) {
	state := actionSeasonClaimState(t)
	giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	validateTestState(t, state)
	resolution, err := ResolveWithOrders(state, testBalance(), nil, map[models.PlayerID][]models.WinterOrder{"P1": {
		claimSheetOrder("O1", "KID", "ANN"),
		{ID: "O2", Type: models.WinterOrderTypeRecruitNoble, CardCode: "ELE", TerritoryID: "AAA"},
	}})
	if err != nil {
		t.Fatalf("ResolveWithOrders: %v", err)
	}
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %v, want none", reasons)
	}
	if len(resolution.State.Claims) != 1 {
		t.Errorf("claims = %+v, want the claim recorded in spring", resolution.State.Claims)
	}
	last := resolution.State.Nobles[len(resolution.State.Nobles)-1]
	if last.Code != "ELE" || last.PlacedTurn != 13 {
		t.Errorf("recruited noble = %+v, want ELE placed on turn 13", last)
	}
}

func TestResolveTurnAcceptsOnlyCardOrdersOutsideWinter(t *testing.T) {
	state := actionSeasonClaimState(t)
	// A second living player keeps the game from being finished.
	state.Territories = append(state.Territories, territory("BBB", "Bbb", "AAA"))
	state.Territories[0].Adjacencies = append(state.Territories[0].Adjacencies, "BBB")
	state.Armies = append(state.Armies, models.Army{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1})
	state.TerritoryStates["BBB"] = models.TerritoryState{Army: armyPointer("A2")}
	state.NextArmyID = 3
	for line, wantErr := range map[string]bool{"C N KID ANN": false, "R T AAA": true, "T N": true, "D C CLM": true} {
		_, err := ResolveTurn(state, testBalance(), OrdersInput{Winter: []WinterSubmission{{Player: "P1", Lines: line}}})
		if (err != nil) != wantErr {
			t.Errorf("ResolveTurn(%q) error = %v, want error %v", line, err, wantErr)
		}
	}
}
