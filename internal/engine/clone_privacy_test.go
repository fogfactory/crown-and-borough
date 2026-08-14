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
