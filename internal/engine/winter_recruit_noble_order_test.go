package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func recruitTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := winterTestState(t, []models.Territory{territory("AAA", "Aaa")}, []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryOwner(state, "AAA", "P1")
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: state.TerritoryStates["AAA"].Army, Infrastructures: infraPointer("I1")}
	return state
}

func applyRecruit(state *models.GameState, balance func(*assetgen.Balance), cardCode string) *resolutionContext {
	b := testBalance()
	if balance != nil {
		balance(&b)
	}
	ctx := newResolutionContext(state, b)
	recruitNobleOrder{order: models.WinterOrder{ID: "O1", Type: models.WinterOrderTypeRecruitNoble, CardCode: cardCode, TerritoryID: "AAA"}}.
		Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	return ctx
}

func TestRecruitNobleOrderPlaysCardFromHandForFree(t *testing.T) {
	state := recruitTestState(t)
	giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	ctx := applyRecruit(state, nil, "ELE")
	if len(state.Nobles) != 1 {
		t.Fatalf("nobles = %d, want 1", len(state.Nobles))
	}
	noble := state.Nobles[0]
	if noble.Code != "ELE" || noble.Name != "Eleonore de Aaa" || noble.Sex != models.SexFemale || noble.OwnerID != "P1" || noble.Status != models.NobleStatusFree {
		t.Fatalf("noble = %#v, want the card's identity as a free P1 noble", noble)
	}
	recruits := eventsOfType(ctx.events, EventTypeRecruit)
	if len(recruits) != 1 || recruits[0].ResourceSpent != 0 {
		t.Fatalf("recruit events = %#v, want one free recruit", ctx.events)
	}
	if hand := state.NobleDeck.Hands["P1"]; len(hand) != 0 {
		t.Fatalf("hand = %v, want the card consumed", hand)
	}
	if played := state.NobleDeck.Played; len(played) != 1 || played[0].Noble != noble.ID {
		t.Fatalf("played = %v, want the card tracked on the recruited noble", played)
	}
	validateTestState(t, state)
}

func TestRecruitNobleOrderRejectsCardNotInHand(t *testing.T) {
	state := recruitTestState(t)
	giveNobleCard(state, "P2", "ELE", "Eleonore", models.SexFemale)
	pileNobleCard(state, "GUI", "Guy", models.SexMale)
	giveDignityCard(state, "P1")
	for _, code := range []string{"ELE", "GUI", "BAS", "XXX"} {
		ctx := applyRecruit(state, nil, code)
		rejected := eventsOfType(ctx.events, EventTypeRejected)
		if len(rejected) != 1 || rejected[0].Reason != "card_not_in_hand" || len(state.Nobles) != 0 {
			t.Fatalf("card %s: events = %#v nobles = %v, want a single card_not_in_hand rejection", code, ctx.events, state.Nobles)
		}
	}
}

func TestRecruitNobleOrderRespectsNobleLimit(t *testing.T) {
	state := recruitTestState(t)
	for index, code := range []string{"AAA", "BBB", "CCC", "DDD"} {
		addNoble(state, models.NobleID("N"+string(rune('1'+index))), code, "P1", "AAA")
	}
	giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	ctx := applyRecruit(state, nil, "ELE")
	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "noble_limit_reached" || len(state.Nobles) != 4 {
		t.Fatalf("events = %#v, want noble_limit_reached", ctx.events)
	}
	if len(state.NobleDeck.Hands["P1"]) != 1 {
		t.Fatal("a rejected recruit consumed the card")
	}
}
