package engine

import (
	"encoding/json"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type winterOrderInvalidCase struct {
	name       string
	order      models.WinterOrder
	setup      func(*models.GameState)
	balance    func(*assetgen.Balance)
	wantReason string
}

func TestWinterOrderApplyRejectsInvalidCases(t *testing.T) {
	cases := []winterOrderInvalidCase{
		{
			name:       "recruit noble unknown territory",
			order:      models.WinterOrder{Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "ZZZ"},
			wantReason: "unknown_territory",
		},
		{
			name:  "recruit noble without settlement",
			order: models.WinterOrder{Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "AAA"},
			setup: func(state *models.GameState) {
				setTerritoryOwner(state, "AAA", "P1")
			},
			wantReason: "noble_requires_settlement",
		},
		{
			name:  "recruit noble without owned army",
			order: models.WinterOrder{Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "AAA"},
			setup: func(state *models.GameState) {
				setTerritoryOwner(state, "AAA", "P1")
				addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
			},
			wantReason: "noble_requires_owned_army",
		},
		{
			name:  "recruit noble insufficient resources",
			order: models.WinterOrder{Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "AAA"},
			setup: func(state *models.GameState) {
				setTerritoryOwner(state, "AAA", "P1")
				state.Armies = append(state.Armies, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1})
				state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: playerPointer("P1"), Army: armyPointer("A1"), Resources: 0, Infrastructures: []models.InfraID{"I1"}}
				addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
			},
			wantReason: "insufficient_resources",
		},
		{
			name:       "recruit troop without noble",
			order:      models.WinterOrder{Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "AAA"},
			setup:      controlledArmyAndSettlement,
			wantReason: "troop_requires_adjacent_noble",
		},
		{
			name:  "recruit troop with enemy army",
			order: models.WinterOrder{Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "AAA"},
			setup: func(state *models.GameState) {
				setTerritoryOwner(state, "AAA", "P1")
				state.Armies = []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}}
				state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: playerPointer("P1"), Army: armyPointer("A1"), Resources: 1, Infrastructures: []models.InfraID{"I1"}}
				addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
				addNoble(state, "N1", "ONE", "P1", "AAA")
			},
			wantReason: "territory_occupied_by_other_player",
		},
		{
			name:  "recruit troop insufficient resources",
			order: models.WinterOrder{Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "AAA"},
			setup: func(state *models.GameState) {
				controlledArmyAndSettlement(state)
				addNoble(state, "N1", "ONE", "P1", "AAA")
				state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: playerPointer("P1"), Army: armyPointer("A1"), Resources: 0, Infrastructures: []models.InfraID{"I1"}}
			},
			wantReason: "insufficient_resources",
		},
		{
			name:       "build unknown infrastructure",
			order:      models.WinterOrder{Type: models.WinterOrderTypeBuild, TerritoryID: "AAA", InfraType: "village"},
			setup:      controlledSettlement,
			wantReason: "invalid_infrastructure",
		},
		{
			name:  "build mill without productive neighbor",
			order: models.WinterOrder{Type: models.WinterOrderTypeBuild, TerritoryID: "AAA", InfraType: models.InfraTypeMill},
			setup: func(state *models.GameState) {
				setTerritoryOwner(state, "AAA", "P1")
			},
			wantReason: "mill_requires_productive_neighbor",
		},
		{
			name:  "build with occupied structure",
			order: models.WinterOrder{Type: models.WinterOrderTypeBuild, TerritoryID: "AAA", InfraType: models.InfraTypeCastle},
			setup: func(state *models.GameState) {
				setTerritoryOwner(state, "AAA", "P1")
				addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
				setTerritoryResources(state, "AAA", 10)
			},
			wantReason: "structure_present",
		},
		{
			name:  "elect capital without castle",
			order: models.WinterOrder{Type: models.WinterOrderTypeElectCapital, TerritoryID: "AAA"},
			setup: func(state *models.GameState) {
				controlledSettlement(state)
			},
			wantReason: "capital_requires_controlled_castle",
		},
		{
			name:       "elect capital unknown territory",
			order:      models.WinterOrder{Type: models.WinterOrderTypeElectCapital, TerritoryID: "ZZZ"},
			wantReason: "unknown_territory",
		},
		{
			name:       "hostage unknown noble",
			order:      models.WinterOrder{Type: models.WinterOrderTypeHostage, NobleCode: "BAD"},
			setup:      holderArmy,
			wantReason: "unknown_noble",
		},
		{
			name:  "dungeon free noble",
			order: models.WinterOrder{Type: models.WinterOrderTypeDungeon, NobleCode: "ONE"},
			setup: func(state *models.GameState) {
				holderArmy(state)
				addNoble(state, "N1", "ONE", "P1", "AAA")
			},
			wantReason: "noble_not_prisoner",
		},
		{
			name:  "hostage noble not held",
			order: models.WinterOrder{Type: models.WinterOrderTypeHostage, NobleCode: "ONE"},
			setup: func(state *models.GameState) {
				state.Armies = []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}}
				state.TerritoryStates["AAA"] = models.TerritoryState{Army: armyPointer("A1")}
				addNoble(state, "N1", "ONE", "P2", "AAA")
				setNobleStatus(state, "N1", models.NobleStatusHostage)
			},
			wantReason: "noble_not_held",
		},
		{
			name:       "liberate unknown noble",
			order:      models.WinterOrder{Type: models.WinterOrderTypeLiberateNoble, NobleCode: "BAD"},
			setup:      holderArmy,
			wantReason: "unknown_noble",
		},
		{
			name:  "liberate without owner capital",
			order: models.WinterOrder{Type: models.WinterOrderTypeLiberateNoble, NobleCode: "ONE"},
			setup: func(state *models.GameState) {
				state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}}
				state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: playerPointer("P1"), Army: armyPointer("A1"), Resources: 10}
				addNoble(state, "N1", "ONE", "P2", "AAA")
				setNobleStatus(state, "N1", models.NobleStatusHostage)
			},
			wantReason: "no_capital",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			state := winterTestState(t, []models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")}, nil)
			if test.setup != nil {
				test.setup(state)
			}
			balance := testBalance()
			if test.balance != nil {
				test.balance(&balance)
			}
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatalf("marshal initial state: %v", err)
			}
			ctx := newResolutionContext(state, balance)
			executable := newExecutableWinterOrder(test.order)
			if executable == nil {
				t.Fatalf("newExecutableWinterOrder(%q) returned nil", test.order.Type)
			}
			executable.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", firstNameRNG: newWinterRNG(state.Seed, state.Turn)})
			if event := firstRejectedEvent(t, ctx.events); event.Reason != test.wantReason {
				t.Fatalf("rejection reason = %q, want %q", event.Reason, test.wantReason)
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatalf("marshal final state: %v", err)
			}
			if string(after) != string(before) {
				t.Fatalf("rejected order mutated state: got %s, want %s", after, before)
			}
		})
	}
}

func controlledSettlement(state *models.GameState) {
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryResources(state, "AAA", 10)
}

func controlledArmyAndSettlement(state *models.GameState) {
	controlledSettlement(state)
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}}
	state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: playerPointer("P1"), Army: armyPointer("A1"), Resources: 10, Infrastructures: []models.InfraID{"I1"}}
}

func holderArmy(state *models.GameState) {
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}}
	state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: playerPointer("P2"), Army: armyPointer("A1"), Resources: 10}
}

func playerPointer(playerID models.PlayerID) *models.PlayerID {
	return &playerID
}

func armyPointer(armyID models.ArmyID) *models.ArmyID {
	return &armyID
}
