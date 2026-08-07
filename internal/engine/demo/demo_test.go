package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestDemoStateDeterministic(t *testing.T) {
	assets := loadTestAssets(t)
	mapData := generateTestMap(t, assets)

	first, err := DemoState("demo-determinism", assets, mapData, 4)
	if err != nil {
		t.Fatalf("first DemoState: %v", err)
	}
	second, err := DemoState("demo-determinism", assets, mapData, 4)
	if err != nil {
		t.Fatalf("second DemoState: %v", err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first state: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second state: %v", err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("same seed produced different GameState JSON")
	}
}

func TestDemoStateStagesAPlausibleMidGame(t *testing.T) {
	assets := loadTestAssets(t)

	for _, players := range []int{2, 3, 4, 5} {
		t.Run("players", func(t *testing.T) {
			mapData := generatePlayerTestMap(t, assets, players)
			state, err := DemoState("demo-mid-game", assets, mapData, players)
			if err != nil {
				t.Fatalf("DemoState(%d): %v", players, err)
			}
			if state.ID != "demo" || state.Turn != demoTurn || state.Season != models.SeasonSpring {
				t.Errorf("state metadata = %+v, want demo turn 5 spring", state)
			}
			if len(state.Players) != players {
				t.Errorf("players = %d, want %d", len(state.Players), players)
			}
			if len(state.Nobles) != players {
				t.Errorf("nobles = %d, want %d", len(state.Nobles), players)
			}
			if len(state.Armies) != players*2 {
				t.Errorf("armies = %d, want %d", len(state.Armies), players*2)
			}
			assertArmySizes(t, state, players*2)
			assertStateReferences(t, state, mapData)
			assertPlayerStaging(t, state)
			assertVillageMaterialization(t, state, mapData)
		})
	}
}

func TestDemoStateSupportsGeneratedMaps(t *testing.T) {
	assets := loadTestAssets(t)
	for _, seed := range []string{"crown-and-borough-dev", "alpha", "beta", "gamma", "delta"} {
		for _, players := range []int{2, 3, 4, 5} {
			t.Run(seed, func(t *testing.T) {
				mapData := generatePlayerMap(t, seed, assets, players)
				if _, err := DemoState(seed, assets, mapData, players); err != nil {
					t.Fatalf("DemoState(%d): %v", players, err)
				}
			})
		}
	}
}

func TestDemoStateSkipsCollidingNobleCodes(t *testing.T) {
	assets := assetgen.Assets{Prenoms: []assetgen.Asset{
		{Code: "BER", Name: "Bérenger"},
		{Code: "BEN", Name: "Bernier"},
		{Code: "BET", Name: "Berthe"},
		{Code: "BRT", Name: "Bertrand"},
		{Code: "ALI", Name: "Alice"},
	}}
	state, err := DemoState("noble-collisions", assets, generateTestMap(t, loadTestAssets(t)), 2)
	if err != nil {
		t.Fatalf("DemoState: %v", err)
	}
	codes := make(map[string]bool, len(state.Nobles))
	for _, noble := range state.Nobles {
		if codes[noble.Code] {
			t.Fatalf("duplicate noble code %q", noble.Code)
		}
		codes[noble.Code] = true
	}
	if !codes["BER"] || !codes["ALI"] {
		t.Errorf("noble codes = %v, want the unique BER and ALI candidates", codes)
	}
}

func TestDemoStateJSONRoundTrip(t *testing.T) {
	assets := loadTestAssets(t)
	state, err := DemoState("demo-round-trip", assets, generateTestMap(t, assets), 4)
	if err != nil {
		t.Fatalf("DemoState: %v", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	var decoded models.GameState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	if !reflect.DeepEqual(*state, decoded) {
		t.Fatalf("GameState changed after JSON round trip\nwant: %#v\ngot: %#v", *state, decoded)
	}
	if err := decoded.Validate(); err != nil {
		t.Errorf("round-tripped state validation: %v", err)
	}
}

func TestDemoFreshnessDeterministicAndBounded(t *testing.T) {
	assets := loadTestAssets(t)
	state, err := DemoState("demo-freshness", assets, generateTestMap(t, assets), 4)
	if err != nil {
		t.Fatalf("DemoState: %v", err)
	}
	first := DemoFreshness(state)
	second := DemoFreshness(state)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("DemoFreshness changed for the same state")
	}
	if len(first) != len(state.Territories) {
		t.Errorf("freshness coverage = %d, want %d", len(first), len(state.Territories))
	}
	stale := 0
	for _, territory := range state.Territories {
		value := first[territory.ID]
		if value != state.Turn && value != state.Turn-2 {
			t.Errorf("freshness[%s] = %d, want %d or %d", territory.ID, value, state.Turn, state.Turn-2)
		}
		if value == state.Turn-2 {
			stale++
		}
	}
	if stale < 1 || stale > 2 {
		t.Errorf("stale territories = %d, want 1 or 2", stale)
	}
}

func TestDemoStateFallsBackWhenVillagesAreScarce(t *testing.T) {
	assets := loadTestAssets(t)
	mapData := fallbackMapData()
	state, err := DemoState("fallback-starts", assets, mapData, 2)
	if err != nil {
		t.Fatalf("DemoState: %v", err)
	}

	castles := 0
	castleOnVillage := 0
	for _, infrastructure := range state.Infrastructures {
		if infrastructure.Type != models.InfraTypeCastle {
			continue
		}
		castles++
		for _, territory := range mapData.Territories {
			if territory.ID == string(infrastructure.TerritoryID) && territory.Village {
				castleOnVillage++
			}
		}
	}
	if castles != 2 || castleOnVillage != 1 {
		t.Errorf("castles = %d, castles on the only village = %d, want 2 and 1", castles, castleOnVillage)
	}
}

func TestDemoStateFallsBackToOneArmyWithoutAdjacentControlledTerritory(t *testing.T) {
	assets := loadTestAssets(t)
	mapData := fallbackMapData()
	for index := 1; index < 3; index++ {
		mapData.Territories[index].Village = true
	}

	var fallback *models.GameState
	for index := 0; index < 20; index++ {
		state, err := DemoState(fmt.Sprintf("fallback-army-%d", index), assets, mapData, 2)
		if err != nil {
			t.Fatalf("DemoState fallback candidate %d: %v", index, err)
		}
		armiesByOwner := make(map[models.PlayerID][]models.Army)
		for _, army := range state.Armies {
			armiesByOwner[army.OwnerID] = append(armiesByOwner[army.OwnerID], army)
		}
		for _, armies := range armiesByOwner {
			if len(armies) == 1 && armies[0].Size == 1 {
				fallback = state
				break
			}
		}
		if fallback != nil {
			break
		}
	}
	if fallback == nil {
		t.Fatal("no deterministic fallback seed produced a player without an adjacent controlled territory")
	}
	if err := fallback.Validate(); err != nil {
		t.Fatalf("fallback state validation: %v", err)
	}
	for _, army := range fallback.Armies {
		if army.Size < 1 {
			t.Errorf("fallback army %s has invalid size %d", army.ID, army.Size)
		}
	}
}

func TestControlledArmyLocationRequiresAdjacency(t *testing.T) {
	mapData := fallbackMapData()
	territoriesByID := make(map[models.TerritoryID]mapgen.Territory, len(mapData.Territories))
	for _, territory := range mapData.Territories {
		territoriesByID[models.TerritoryID(territory.ID)] = territory
	}
	if location := controlledArmyLocation("T01", []models.TerritoryID{"T01", "T03"}, territoriesByID, newRNG("location-test", "army")); location != "" {
		t.Errorf("non-adjacent army location = %s, want none", location)
	}
}

func TestDemoStateRejectsUnsupportedInput(t *testing.T) {
	assets := loadTestAssets(t)
	mapData := generateTestMap(t, assets)
	for _, players := range []int{1, 17} {
		if _, err := DemoState("invalid-player-count", assets, mapData, players); err == nil {
			t.Errorf("DemoState(%d) = nil error, want error", players)
		}
	}
	if _, err := DemoState("empty-map", assets, mapgen.MapData{}, 2); err == nil {
		t.Error("DemoState(empty map) = nil error, want error")
	}
}

func assertStateReferences(t *testing.T, state *models.GameState, mapData mapgen.MapData) {
	t.Helper()
	if err := state.Validate(); err != nil {
		t.Fatalf("generated state validation: %v", err)
	}
	territories := make(map[models.TerritoryID]bool, len(mapData.Territories))
	for _, territory := range mapData.Territories {
		territories[models.TerritoryID(territory.ID)] = true
	}
	if len(state.TerritoryStates) != len(territories) {
		t.Errorf("territory state coverage = %d, want %d", len(state.TerritoryStates), len(territories))
	}
	for _, army := range state.Armies {
		if !territories[army.TerritoryID] {
			t.Errorf("army %s references unknown territory %s", army.ID, army.TerritoryID)
		}
	}
	for territoryID, territoryState := range state.TerritoryStates {
		if territoryState.Army == nil {
			continue
		}
		found := false
		for _, army := range state.Armies {
			if army.ID == *territoryState.Army {
				found = true
				if army.TerritoryID != territoryID {
					t.Errorf("army %s is indexed by %s but stationed in %s", army.ID, territoryID, army.TerritoryID)
				}
			}
		}
		if !found {
			t.Errorf("territory %s references missing army %s", territoryID, *territoryState.Army)
		}
	}
	for _, noble := range state.Nobles {
		if !territories[noble.LocationID] {
			t.Errorf("noble %s references unknown territory %s", noble.ID, noble.LocationID)
		}
	}
	for _, infrastructure := range state.Infrastructures {
		if !territories[infrastructure.TerritoryID] {
			t.Errorf("infrastructure %s references unknown territory %s", infrastructure.ID, infrastructure.TerritoryID)
		}
	}
}

func assertArmySizes(t *testing.T, state *models.GameState, wantCount int) {
	t.Helper()
	if len(state.Armies) != wantCount {
		return
	}
	seenIDs := make(map[models.ArmyID]bool, len(state.Armies))
	seenTerritories := make(map[models.TerritoryID]bool, len(state.Armies))
	totalSize := 0
	for _, army := range state.Armies {
		if seenIDs[army.ID] {
			t.Errorf("duplicate army id %s", army.ID)
		}
		seenIDs[army.ID] = true
		if seenTerritories[army.TerritoryID] {
			t.Errorf("multiple armies on %s", army.TerritoryID)
		}
		seenTerritories[army.TerritoryID] = true
		if army.Size != 1 && army.Size != 2 {
			t.Errorf("army %s size = %d, want 1 or 2", army.ID, army.Size)
		}
		totalSize += army.Size
	}
	if totalSize != wantCount+wantCount/2 {
		t.Errorf("total army size = %d, want %d", totalSize, wantCount+wantCount/2)
	}
}

func assertPlayerStaging(t *testing.T, state *models.GameState) {
	t.Helper()
	infrastructures := make(map[models.InfraID]models.Infrastructure, len(state.Infrastructures))
	for _, infrastructure := range state.Infrastructures {
		infrastructures[infrastructure.ID] = infrastructure
	}
	for _, player := range state.Players {
		controlled := 0
		castles := 0
		castleTerritory := models.TerritoryID("")
		totalSize := 0
		armyLocations := make(map[models.TerritoryID]bool)
		nobles := 0
		for territoryID, territoryState := range state.TerritoryStates {
			if territoryState.OwnerID == nil || *territoryState.OwnerID != player.ID {
				continue
			}
			controlled++
			for _, infrastructureID := range territoryState.Infrastructures {
				if infrastructure := infrastructures[infrastructureID]; infrastructure.Type == models.InfraTypeCastle {
					castles++
					castleTerritory = territoryID
				}
			}
			if territoryState.Army != nil {
				for _, army := range state.Armies {
					if army.ID == *territoryState.Army && army.OwnerID == player.ID {
						totalSize += army.Size
						armyLocations[territoryID] = true
					}
				}
			}
		}
		for _, noble := range state.Nobles {
			if noble.OwnerID == player.ID {
				nobles++
			}
		}
		if controlled < 2 || controlled > 3 {
			t.Errorf("%s controls %d territories, want 2 or 3", player.ID, controlled)
		}
		if castles != 1 {
			t.Errorf("%s castles = %d, want 1", player.ID, castles)
		}
		if totalSize != 3 || len(armyLocations) != 2 {
			t.Errorf("%s total army size = %d across %d armies, want 3 across 2", player.ID, totalSize, len(armyLocations))
		}
		if nobles != 1 {
			t.Errorf("%s nobles = %d, want 1", player.ID, nobles)
		}
		for _, army := range state.Armies {
			if army.OwnerID == player.ID && army.Size > 1 && !adjacentTerritories(state, castleTerritory, army.TerritoryID) {
				t.Errorf("%s army %s is not adjacent to its castle", player.ID, army.ID)
			}
		}
	}

	seenInfrastructureTerritories := make(map[models.TerritoryID]bool, len(state.Infrastructures))
	for _, infrastructure := range state.Infrastructures {
		if seenInfrastructureTerritories[infrastructure.TerritoryID] {
			t.Errorf("multiple infrastructures on %s", infrastructure.TerritoryID)
		}
		seenInfrastructureTerritories[infrastructure.TerritoryID] = true
	}
	seenCodes := make(map[string]bool, len(state.Nobles))
	for _, noble := range state.Nobles {
		if seenCodes[noble.Code] {
			t.Errorf("duplicate noble code %s", noble.Code)
		}
		seenCodes[noble.Code] = true
	}
	mills := 0
	extras := map[models.InfraType]int{}
	for _, infrastructure := range state.Infrastructures {
		extras[infrastructure.Type]++
		if infrastructure.Type != models.InfraTypeMill {
			continue
		}
		mills++
		if !hasAdjacentCastle(state, infrastructure.TerritoryID) {
			t.Errorf("mill %s is not adjacent to a castle", infrastructure.ID)
		}
	}
	if mills != 1 {
		t.Errorf("mills = %d, want 1", mills)
	}
	for _, infrastructureType := range []models.InfraType{models.InfraTypeWatchtower, models.InfraTypeSupplyDepot} {
		if extras[infrastructureType] != 1 {
			t.Errorf("%s infrastructures = %d, want 1", infrastructureType, extras[infrastructureType])
		}
	}
}

func adjacentTerritories(state *models.GameState, first, second models.TerritoryID) bool {
	for _, territory := range state.Territories {
		if territory.ID != first {
			continue
		}
		for _, adjacent := range territory.Adjacencies {
			if adjacent == second {
				return true
			}
		}
	}
	return false
}

func hasAdjacentCastle(state *models.GameState, territoryID models.TerritoryID) bool {
	castles := make(map[models.TerritoryID]bool)
	for _, infrastructure := range state.Infrastructures {
		if infrastructure.Type == models.InfraTypeCastle {
			castles[infrastructure.TerritoryID] = true
		}
	}
	for _, territory := range state.Territories {
		if territory.ID != territoryID {
			continue
		}
		for _, adjacent := range territory.Adjacencies {
			if castles[adjacent] {
				return true
			}
		}
	}
	return false
}

func assertVillageMaterialization(t *testing.T, state *models.GameState, mapData mapgen.MapData) {
	t.Helper()
	infrastructures := make(map[models.InfraID]models.Infrastructure, len(state.Infrastructures))
	for _, infrastructure := range state.Infrastructures {
		infrastructures[infrastructure.ID] = infrastructure
	}
	for _, territory := range mapData.Territories {
		if !territory.Village {
			continue
		}
		stateForTerritory := state.TerritoryStates[models.TerritoryID(territory.ID)]
		if len(stateForTerritory.Infrastructures) != 1 {
			t.Errorf("village territory %s has %d infrastructures, want one village or castle", territory.ID, len(stateForTerritory.Infrastructures))
			continue
		}
		infrastructure := infrastructures[stateForTerritory.Infrastructures[0]]
		if infrastructure.Type != models.InfraTypeVillage && infrastructure.Type != models.InfraTypeCastle {
			t.Errorf("village territory %s has %s, want village or castle", territory.ID, infrastructure.Type)
		}
		if infrastructure.Type == models.InfraTypeVillage && stateForTerritory.OwnerID != nil {
			t.Errorf("remaining village %s is controlled by %s", territory.ID, *stateForTerritory.OwnerID)
		}
	}
}

func loadTestAssets(t *testing.T) assetgen.Assets {
	t.Helper()
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	return assets
}

func generateTestMap(t *testing.T, assets assetgen.Assets) mapgen.MapData {
	return generatePlayerTestMap(t, assets, 5)
}

func generatePlayerTestMap(t *testing.T, assets assetgen.Assets, players int) mapgen.MapData {
	return generatePlayerMap(t, "demo-map", assets, players)
}

func generatePlayerMap(t *testing.T, seed string, assets assetgen.Assets, players int) mapgen.MapData {
	t.Helper()
	mapData, err := mapgen.Generate(seed, assets, mapgen.Config{
		Width:        1000,
		Height:       700,
		SiteCount:    mapgen.TerritoriesPerPlayer * players,
		VillageCount: players + 1,
	})
	if err != nil {
		t.Fatalf("generate map: %v", err)
	}
	return mapData
}

func fallbackMapData() mapgen.MapData {
	territories := make([]mapgen.Territory, 8)
	codes := []string{"AAA", "AAB", "AAC", "AAD", "AAE", "AAF", "AAG", "AAH"}
	for index := range territories {
		previous := (index + len(territories) - 1) % len(territories)
		next := (index + 1) % len(territories)
		territories[index] = mapgen.Territory{
			ID:          "T0" + string(rune('1'+index)),
			Code:        codes[index],
			Name:        "Territory " + string(rune('A'+index)),
			Terrain:     models.TerrainPlain,
			Village:     index == 0,
			Adjacencies: []string{"T0" + string(rune('1'+previous)), "T0" + string(rune('1'+next))},
			Impassable:  []string{},
		}
	}
	return mapgen.MapData{Territories: territories}
}
