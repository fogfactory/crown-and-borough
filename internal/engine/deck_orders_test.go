package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestCardDefinitionsRegisterBonusOrders(t *testing.T) {
	for _, kind := range []models.CardKind{models.CardKindFairWeather, models.CardKindAbundantHarvest, models.CardKindRevolt} {
		definition := cardDefinitions[kind]
		if definition == nil || definition.Kind() != kind {
			t.Errorf("definition[%q] = %#v, want matching definition", kind, definition)
		}
		if order := newExecutableDeckOrder("P1", models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: kind}); order == nil {
			t.Errorf("newExecutableDeckOrder(%q) = nil", kind)
		}
	}
	for _, kind := range []models.CardKind{models.CardKindPlague, models.CardKindBadWeather, models.CardKindFamine} {
		if order := newExecutableDeckOrder("P1", models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: kind}); order != nil {
			t.Errorf("newExecutableDeckOrder(%q) = %T, want nil", kind, order)
		}
	}
	if definition := cardDefinitions[models.CardKindRevolt]; definition == nil {
		t.Fatal("revolt definition is not registered")
	} else if canPlay, _ := definition.CanPlay(&ExecutionContext{season: models.SeasonWinter}, models.DeckOrder{RegionSeed: "ROS"}); canPlay {
		t.Fatal("revolt definition is not registered with a winter restriction")
	}
}

func TestDeckCardOrderApplyConsumesFirstMatchingCard(t *testing.T) {
	state := models.NewGameState()
	state.SpecialDeck = &models.SpecialDeck{
		Cards:   []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}, {ID: "C2", Kind: models.CardKindFairWeather}},
		Hands:   map[models.PlayerID][]models.SpecialCardID{"P1": {"C1", "C2"}},
		Discard: []models.SpecialCardID{},
	}
	ctx := newResolutionContext(state, testBalance())
	order := newExecutableDeckOrder("P1", models.DeckOrder{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindFairWeather})
	order.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", season: models.SeasonSpring})
	if got := state.SpecialDeck.Hands["P1"]; len(got) != 1 || got[0] != "C2" {
		t.Fatalf("hand = %#v, want [C2]", got)
	}
	if got := state.SpecialDeck.Discard; len(got) != 1 || got[0] != "C1" {
		t.Fatalf("discard = %#v, want [C1]", got)
	}
	if len(ctx.deckIntents) != 1 || ctx.deckIntents[0].order.Kind != models.CardKindFairWeather {
		t.Fatalf("deck intents = %#v, want one fair weather intent", ctx.deckIntents)
	}
}

func TestDeckCardOrderApplyRejectsMissingCardWithoutMutation(t *testing.T) {
	state := models.NewGameState()
	state.SpecialDeck = &models.SpecialDeck{
		Cards:   []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}},
		Hands:   map[models.PlayerID][]models.SpecialCardID{"P1": {}},
		Discard: []models.SpecialCardID{},
	}
	ctx := newResolutionContext(state, testBalance())
	order := newExecutableDeckOrder("P1", models.DeckOrder{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindFairWeather})
	order.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", season: models.SeasonSpring})
	if len(state.SpecialDeck.Hands["P1"]) != 0 || len(state.SpecialDeck.Discard) != 0 || len(ctx.deckIntents) != 0 {
		t.Fatalf("missing card mutated state: %#v/%#v/%#v", state.SpecialDeck.Hands, state.SpecialDeck.Discard, ctx.deckIntents)
	}
	if len(eventsOfType(ctx.events, EventTypeRejected)) != 1 || ctx.events[0].Reason != "no_card_for_kind" {
		t.Fatalf("events = %#v, want no_card_for_kind rejection", ctx.events)
	}
}

func TestDeckCardOrdersRejectWinter(t *testing.T) {
	for _, kind := range []models.CardKind{models.CardKindFairWeather, models.CardKindAbundantHarvest, models.CardKindRevolt} {
		t.Run(string(kind), func(t *testing.T) {
			state := models.NewGameState()
			state.SpecialDeck = &models.SpecialDeck{
				Cards:   []models.SpecialCard{{ID: "C1", Kind: kind}},
				Hands:   map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}},
				Discard: []models.SpecialCardID{},
			}
			ctx := newResolutionContext(state, testBalance())
			order := newExecutableDeckOrder("P1", models.DeckOrder{ID: "O1", Type: models.DeckOrderTypePlay, Kind: kind})
			order.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", season: models.SeasonWinter})
			if len(state.SpecialDeck.Hands["P1"]) != 1 || len(state.SpecialDeck.Discard) != 0 {
				t.Fatalf("winter rejection consumed card: %#v/%#v", state.SpecialDeck.Hands, state.SpecialDeck.Discard)
			}
			if len(ctx.events) != 1 || ctx.events[0].Reason != "deck_order_out_of_season" {
				t.Fatalf("events = %#v, want deck_order_out_of_season", ctx.events)
			}
		})
	}
}

func TestRevoltCardRequiresActiveFamine(t *testing.T) {
	state := models.NewGameState()
	state.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"ROS"}}}
	state.Auguries[1] = models.YearAugury{
		Year:       1,
		Capacities: map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonWinter: 1},
		Calamities: []models.Calamity{{Kind: models.CardKindFamine, Season: models.SeasonSpring, Year: 1, RegionSeed: "ROS"}},
	}
	ctx := newResolutionContext(state, testBalance())
	definition := cardDefinitions[models.CardKindRevolt]
	if ok, reason := definition.CanPlay(&ExecutionContext{resolution: ctx, season: models.SeasonSpring}, models.DeckOrder{RegionSeed: "ROS"}); !ok || reason != "" {
		t.Fatalf("revolt with famine = %t/%q, want true/empty", ok, reason)
	}
	if ok, reason := definition.CanPlay(&ExecutionContext{resolution: ctx, season: models.SeasonSpring}, models.DeckOrder{RegionSeed: "XXX"}); ok || reason != "revolt_requires_famine" {
		t.Fatalf("revolt without famine = %t/%q, want false/revolt_requires_famine", ok, reason)
	}
}
