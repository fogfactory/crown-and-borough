//go:build demo

package main

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// Forged-state scenarios of the hotseat demo (scripts/demo.sh <scenario>).
// A scenario edits the state of the freshly created game (4 players, seed
// crown-and-borough-dev); the store validates the result. Add one here when a
// feature needs a situation that takes many turns to reach.
var demoScenarios = map[string]struct {
	description string
	apply       func(*models.GameState) error
}{
	"winter": {
		description: "winter, P1 is pope with a bishop-astrologer, a Witch, cards in hand; P2 holds a cardinal and an excommunicated noble",
		apply:       forgeWinter,
	},
	"action": {
		description: "spring, P1 holds bonus cards; a famine is active, a bad weather is announced (one bent by a ritual)",
		apply:       forgeAction,
	},
}

func init() {
	demoRegistered = func(name string, memory *store.MemoryStore) error {
		scenario, known := demoScenarios[name]
		if !known {
			names := make([]string, 0, len(demoScenarios))
			for known := range demoScenarios {
				names = append(names, known)
			}
			sort.Strings(names)
			return fmt.Errorf("unknown scenario (have %v)", names)
		}
		var failure error
		if err := memory.Forge(func(state *models.GameState) { failure = scenario.apply(state) }); err != nil {
			return err
		}
		return failure
	}
}

// ownedNobles returns the indexes of the nobles of a player, in state order.
func ownedNobles(state *models.GameState, player models.PlayerID) []int {
	var indexes []int
	for index, noble := range state.Nobles {
		if noble.OwnerID == player {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

// giveSpecialCards moves the first card of each kind from the draw pile to the
// player's hand.
func giveSpecialCards(state *models.GameState, player models.PlayerID, kinds ...models.CardKind) {
	deck := state.SpecialDeck
	if deck == nil {
		return
	}
	for _, kind := range kinds {
		for _, card := range deck.Cards {
			if card.Kind != kind {
				continue
			}
			index := -1
			for position, id := range deck.DrawPile {
				if id == card.ID {
					index = position
					break
				}
			}
			if index < 0 {
				continue
			}
			deck.DrawPile = append(deck.DrawPile[:index], deck.DrawPile[index+1:]...)
			deck.Hands[player] = append(deck.Hands[player], card.ID)
			break
		}
	}
}

func giveNobleCards(state *models.GameState, player models.PlayerID, cards ...models.NobleCard) {
	if state.NobleDeck == nil {
		return
	}
	for index, card := range cards {
		card.ID = models.NobleCardID(fmt.Sprintf("KD%d", index+1))
		state.NobleDeck.Cards = append(state.NobleDeck.Cards, card)
		state.NobleDeck.Hands[player] = append(state.NobleDeck.Hands[player], card.ID)
	}
}

func forgeWinter(state *models.GameState) error {
	state.Season = models.SeasonWinter
	state.Turn = 4
	p1, p2 := ownedNobles(state, "P1"), ownedNobles(state, "P2")
	if len(p1) < 2 || len(p2) < 2 || len(state.Regions) < 3 {
		return fmt.Errorf("the base game lacks nobles or regions")
	}
	pope, bishop := state.Nobles[p1[0]].ID, state.Nobles[p1[1]].ID
	cardinal, excommunicated := state.Nobles[p2[0]].ID, state.Nobles[p2[1]].ID
	state.Bishops = []models.Bishop{
		{Region: state.Regions[0].ID, Noble: pope},
		{Region: state.Regions[1].ID, Noble: bishop},
		{Region: state.Regions[2].ID, Noble: cardinal},
	}
	state.Cardinals = []models.NobleID{pope, cardinal}
	state.Pope = &pope
	state.Excommunications = []models.Excommunication{{Noble: excommunicated, Reason: models.ExcommunicationPapal, By: pope}}
	// P1's second noble becomes a lady astrologer; a Witch joins P1's court.
	state.Nobles[p1[1]].Sex = models.SexFemale
	state.Nobles[p1[1]].Dignities = []models.Dignity{models.DignityAstrologer}
	state.Nobles = append(state.Nobles, models.Noble{
		ID: "N900", Code: "ZOE", Name: "Zoé", Sex: models.SexFemale, OwnerID: "P1",
		LocationID: state.Nobles[p1[0]].LocationID, Status: models.NobleStatusFree,
		Dignities: []models.Dignity{models.DignityWitch},
	})
	giveSpecialCards(state, "P1", models.CardKindFairWeather, models.CardKindRevolt, models.CardKindTrial, models.CardKindSeigneurialTax)
	giveNobleCards(state, "P1",
		models.NobleCard{Kind: models.NobleCardKindNoble, Code: "ZAL", Name: "Albert", Sex: models.SexMale},
		models.NobleCard{Kind: models.NobleCardKindDignity, Code: models.DignityBastardCardCode, Dignity: models.DignityBastard},
	)
	return nil
}

func forgeAction(state *models.GameState) error {
	if len(state.Regions) < 3 {
		return fmt.Errorf("the base game lacks regions")
	}
	year := state.Year()
	state.Auguries[year] = models.YearAugury{
		Year:       year,
		Capacities: map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonAutumn: 1},
		Revealed:   true,
		Calamities: []models.Calamity{
			{CardID: "ZF1", Kind: models.CardKindFamine, Year: year, Season: models.SeasonSpring, RegionSeed: state.Regions[0].Seed},
			{CardID: "ZB1", Kind: models.CardKindBadWeather, Year: year, Season: models.SeasonSummer, RegionSeed: state.Regions[1].Seed, Ritual: true},
			{CardID: "ZB2", Kind: models.CardKindBadWeather, Year: year, Season: models.SeasonAutumn, RegionSeed: state.Regions[2].Seed},
		},
	}
	if state.SpecialDeck != nil {
		for _, card := range []models.SpecialCard{
			{ID: "ZF1", Kind: models.CardKindFamine},
			{ID: "ZB1", Kind: models.CardKindBadWeather},
			{ID: "ZB2", Kind: models.CardKindBadWeather},
		} {
			state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, card)
		}
	}
	giveSpecialCards(state, "P1", models.CardKindFairWeather, models.CardKindAbundantHarvest, models.CardKindRevolt)
	return nil
}
