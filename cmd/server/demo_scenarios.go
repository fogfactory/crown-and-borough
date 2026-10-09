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
	"appeasement": {
		description: "spring, P1 has a bishop, a cardinal and an abbess; rebel armies of 1 to 3 troops stand in their regions and elsewhere; P1 has plenty of R",
		apply:       forgeAppeasement,
	},
	"tithe": {
		description: "spring, P1 has a bishop and a cardinal (see the appeasement scenario) and two Tax cards: tithe orders on bishoprics",
		apply:       forgeTithe,
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

// forgeAppeasement sets up the revolt appeasement cases (specs/religieux.md):
// P1's first noble is the bishop of the region it stands in, its second noble
// a cardinal (and bishop of another region), and a new abbess stands beside
// the cardinal. Rebel armies wait in four places: 2 troops and 1 troop in the
// bishop's region, 3 troops in the cardinal's, 1 troop in a region where P1 has
// no cleric. The orders to try, with the noble codes printed by the game:
//
//	P AG <bishop> <rebels in his region>      free rite, may kill
//	P AP <bishop> <rebels in his region>      paid, 2^size R
//	P AP <bishop> <rebels elsewhere>          rejected: not his bishopric
//	P AP <cardinal> <rebels anywhere>         paid, anywhere
//	P AG <abbess> <rebels where she stands>   the only way open to her
func forgeAppeasement(state *models.GameState) error {
	p1 := ownedNobles(state, "P1")
	if len(p1) < 2 || len(state.Regions) < 3 {
		return fmt.Errorf("the base game lacks nobles or regions")
	}
	regionOf := func(territoryID models.TerritoryID) int {
		for index, region := range state.Regions {
			for _, id := range region.Territories {
				if id == territoryID {
					return index
				}
			}
		}
		return -1
	}
	bishop, cardinal := state.Nobles[p1[0]], state.Nobles[p1[1]]
	bishopRegion, cardinalRegion := regionOf(bishop.LocationID), regionOf(cardinal.LocationID)
	if bishopRegion < 0 || cardinalRegion < 0 {
		return fmt.Errorf("a noble stands outside every region")
	}
	if cardinalRegion == bishopRegion {
		// Their own bishoprics must differ: the cardinal takes the next region.
		cardinalRegion = (bishopRegion + 1) % len(state.Regions)
	}
	state.Bishops = []models.Bishop{
		{Region: state.Regions[bishopRegion].ID, Noble: bishop.ID},
		{Region: state.Regions[cardinalRegion].ID, Noble: cardinal.ID},
	}
	state.Cardinals = []models.NobleID{cardinal.ID}
	state.Nobles = append(state.Nobles, models.Noble{
		ID: "N901", Code: "ZAB", Name: "Aliénor", Sex: models.SexFemale, OwnerID: "P1",
		LocationID: cardinal.LocationID, Status: models.NobleStatusFree,
		Dignities: []models.Dignity{models.DignityAbbess}, AbbeyRegion: state.Regions[cardinalRegion].Seed,
	})
	occupied := make(map[models.TerritoryID]bool)
	for _, army := range state.Armies {
		occupied[army.TerritoryID] = true
	}
	freeIn := func(region int) (models.TerritoryID, bool) {
		for _, id := range state.Regions[region].Territories {
			if !occupied[id] {
				occupied[id] = true
				return id, true
			}
		}
		return "", false
	}
	other := 0
	for other == bishopRegion || other == cardinalRegion {
		other++
	}
	next := 1
	for _, rebel := range []struct {
		region int
		size   int
	}{{bishopRegion, 2}, {bishopRegion, 1}, {cardinalRegion, 3}, {other, 1}} {
		territoryID, found := freeIn(rebel.region)
		if !found {
			return fmt.Errorf("region %d has no free territory for a rebel army", rebel.region)
		}
		armyID := models.ArmyID(fmt.Sprintf("ZR%d", next))
		next++
		state.Armies = append(state.Armies, models.Army{ID: armyID, OwnerID: models.NeutralPlayerID, TerritoryID: territoryID, Size: rebel.size})
		territoryState := state.TerritoryStates[territoryID]
		territoryState.Army = &armyID
		state.TerritoryStates[territoryID] = territoryState
		fmt.Printf("demo appeasement: %d rebel troops at %s (region %s)\n", rebel.size, territoryID, state.Regions[rebel.region].Seed)
	}
	state.NextArmyID += len(state.Armies)
	for territoryID, territoryState := range state.TerritoryStates {
		if owner, controlled := state.TerritoryController(territoryID); controlled && owner == "P1" && territoryState.Infrastructures != nil {
			territoryState.Resources = 40
			state.TerritoryStates[territoryID] = territoryState
		}
	}
	fmt.Printf("demo appeasement: bishop %s (region %s), cardinal %s (bishop of %s), abbess ZAB stands at %s\n",
		bishop.Code, state.Regions[bishopRegion].Seed, cardinal.Code, state.Regions[cardinalRegion].Seed, cardinal.LocationID)
	return nil
}

// forgeTithe reuses the appeasement clerics (a bishop and a cardinal of P1),
// hands P1 two Tax cards and makes the bishop also the baron of a fief whose
// capital is the seed of his own bishopric, so the same territory can be taxed
// as a fief (seigneurial tax) or as a bishopric (tithe). The orders to try:
//
//	P TX <bishop> <seed>     seigneurial tax on his fief
//	P DI <bishop> <seed>     tithe on his bishopric
//	P DI <cardinal> <seed>   tithe on any bishopric
func forgeTithe(state *models.GameState) error {
	if err := forgeAppeasement(state); err != nil {
		return err
	}
	giveSpecialCards(state, "P1", models.CardKindSeigneurialTax, models.CardKindSeigneurialTax)
	p1 := ownedNobles(state, "P1")
	bishop := state.Nobles[p1[0]]
	region, found := state.RegionOfTerritory(bishop.LocationID)
	if !found {
		return fmt.Errorf("the bishop stands outside every region")
	}
	territories := []models.TerritoryID{region.Seed}
	for _, territory := range state.Territories {
		if territory.ID != region.Seed && len(territories) < models.FiefMinTerritories && containsTerritory(territory.Adjacencies, region.Seed) {
			territories = append(territories, territory.ID)
		}
	}
	if len(territories) < models.FiefMinTerritories {
		return fmt.Errorf("the bishopric seed %s has too few neighbors for a fief", region.Seed)
	}
	title, _ := models.FiefTitleForSize(len(territories))
	holder := bishop.ID
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "F901", Title: title, CapitalTerritoryID: region.Seed,
		Territories: territories, OwnerID: "P1", HolderNobleID: &holder,
	})
	fmt.Printf("demo tithe: %s is bishop of %s and holds a fief whose capital is %s\n", bishop.Code, region.Seed, region.Seed)
	return nil
}

func containsTerritory(list []models.TerritoryID, target models.TerritoryID) bool {
	for _, id := range list {
		if id == target {
			return true
		}
	}
	return false
}
