package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestCloneGameStateDeepCopiesPrivacySnapshots(t *testing.T) {
	state := models.NewGameState()
	state.Privacy.ChainKnowledge["P1"] = map[models.ChainID]models.ChainSnapshot{
		"C1": {
			ID: "C1", NobleID: "N1", ArmyID: "A1", CurrentIndex: 0,
			Orders: []models.Order{{
				ID: "O1", ArmyID: "A1", PositionID: "ROS",
				TargetIDs: []models.TerritoryID{"BOI"},
				NobleAssignments: map[models.TerritoryID][]models.NobleCode{
					"BOI": {"N1"},
				},
			}},
		},
	}
	state.Privacy.CombatParticipation["P1"] = map[string]bool{"combat-BOI": true}

	clone := cloneGameState(state)
	clone.Privacy.ChainKnowledge["P1"]["C1"].Orders[0].TargetIDs[0] = "FOU"
	clone.Privacy.ChainKnowledge["P1"]["C1"].Orders[0].NobleAssignments["BOI"][0] = "ALI"
	clone.Privacy.CombatParticipation["P1"]["combat-BOI"] = false

	snapshot := state.Privacy.ChainKnowledge["P1"]["C1"]
	if snapshot.Orders[0].TargetIDs[0] != "BOI" {
		t.Error("privacy snapshot target was aliased by clone")
	}
	if snapshot.Orders[0].NobleAssignments["BOI"][0] != "N1" {
		t.Error("privacy snapshot noble assignment was aliased by clone")
	}
	if !state.Privacy.CombatParticipation["P1"]["combat-BOI"] {
		t.Error("combat participation map was aliased by clone")
	}
}

func TestCloneGameStateDeepCopiesDeckRegionsAndAuguries(t *testing.T) {
	state := models.NewGameState()
	state.Regions = []models.Region{{
		ID:          "ROS",
		Seed:        "ROS",
		Territories: []models.TerritoryID{"ROS", "BOI"},
	}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindPlague}, {ID: "C2", Kind: models.CardKindFairWeather}},
		DrawPile: []models.SpecialCardID{"C1"},
		Discard:  []models.SpecialCardID{"C2"},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C2"}},
	}
	state.Auguries = map[int]models.YearAugury{
		2: {
			Year:       2,
			Capacities: map[models.Season]int{models.SeasonSpring: 1},
			Calamities: []models.Calamity{{CardID: "C1", Kind: models.CardKindPlague, Year: 2, Season: models.SeasonSpring, RegionSeed: "ROS"}},
		},
	}

	clone := cloneGameState(state)
	clone.Regions[0].Territories[0] = "BOI"
	clone.SpecialDeck.DrawPile[0] = "C2"
	clone.SpecialDeck.Discard[0] = "C1"
	clone.SpecialDeck.Hands["P1"][0] = "C1"
	augury := clone.Auguries[2]
	augury.Capacities[models.SeasonSpring] = 2
	augury.Calamities[0].RegionSeed = "BOI"
	clone.Auguries[2] = augury

	if state.Regions[0].Territories[0] != "ROS" {
		t.Error("region territories were aliased by clone")
	}
	if state.SpecialDeck.DrawPile[0] != "C1" || state.SpecialDeck.Discard[0] != "C2" {
		t.Error("deck piles were aliased by clone")
	}
	if state.SpecialDeck.Hands["P1"][0] != "C2" {
		t.Error("deck hand was aliased by clone")
	}
	if state.Auguries[2].Capacities[models.SeasonSpring] != 1 || state.Auguries[2].Calamities[0].RegionSeed != "ROS" {
		t.Error("augury was aliased by clone")
	}
}
