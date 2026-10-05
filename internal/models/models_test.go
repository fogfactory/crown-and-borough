package models_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func ptrID(id string) *models.PlayerID {
	p := models.PlayerID(id)
	return &p
}

func ptrArmyID(id string) *models.ArmyID {
	a := models.ArmyID(id)
	return &a
}

func ptrChainID(id string) *models.ChainID {
	c := models.ChainID(id)
	return &c
}

func ptrInfraID(id string) *models.InfraID {
	infrastructureID := models.InfraID(id)
	return &infrastructureID
}

func ptrNobleID(id string) *models.NobleID {
	nobleID := models.NobleID(id)
	return &nobleID
}

// validState returns a complete, valid state: 2 players, 4 territories in a
// ring with commune trigrams, 2 armies, 1 noble, 1 mill level 2 and 1 castle,
// 1 neutral territory, and stock on its castle. Tests mutate a fresh instance to build
// negative cases.
func validState() *models.GameState {
	g := models.NewGameState()
	g.ID = "game-1"
	g.Seed = "seed-1"
	g.Players = []models.Player{
		{ID: "P1", Name: "Alpha", Color: "red"},
		{ID: "P2", Name: "Beta", Color: "blue"},
	}
	g.Territories = []models.Territory{
		{ID: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"BCL", "FOU"}},
		{ID: "BCL", Name: "Boisclair", Terrain: models.TerrainForest, Adjacencies: []models.TerritoryID{"ROS", "BRU"}},
		{ID: "BRU", Name: "Bruyères", Terrain: models.TerrainHill, Adjacencies: []models.TerritoryID{"BCL", "FOU"}},
		{ID: "FOU", Name: "Fougères", Terrain: models.TerrainSwamp, Adjacencies: []models.TerritoryID{"BRU", "ROS"}},
	}
	g.Armies = []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "ROS", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BCL", Size: 2},
	}
	g.NextArmyID = 3
	g.Nobles = []models.Noble{
		{ID: "N1", Sex: models.SexMale, Code: "HUG", Name: "Hugues", OwnerID: "P1", LocationID: "ROS", Status: models.NobleStatusFree},
	}
	g.Infrastructures = []models.Infrastructure{
		{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "ROS"},
		{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BRU"},
	}
	g.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"ROS": {Resources: 0, Army: ptrArmyID("A1"), Infrastructures: ptrInfraID("I1")},
		"BCL": {Resources: 0, Army: ptrArmyID("A2")},
		"BRU": {Resources: 5, Infrastructures: ptrInfraID("I2")},
		"FOU": {Resources: 0},
	}
	return g
}

func TestTerrainIsValid(t *testing.T) {
	for _, valid := range []models.Terrain{
		models.TerrainPlain, models.TerrainForest, models.TerrainHill,
		models.TerrainMountain, models.TerrainSwamp,
	} {
		if !valid.IsValid() {
			t.Errorf("Terrain %q: want valid", valid)
		}
	}
	for _, invalid := range []models.Terrain{"", "MARSH", "any", "forest ", "plain2"} {
		if invalid.IsValid() {
			t.Errorf("Terrain %q: want invalid", invalid)
		}
	}
}

func TestSeasonIsValid(t *testing.T) {
	for _, valid := range []models.Season{
		models.SeasonSpring, models.SeasonSummer, models.SeasonAutumn, models.SeasonWinter,
	} {
		if !valid.IsValid() {
			t.Errorf("Season %q: want valid", valid)
		}
	}
	for _, invalid := range []models.Season{"", "snow", "SPRING", "spring "} {
		if invalid.IsValid() {
			t.Errorf("Season %q: want invalid", invalid)
		}
	}
}

func TestInfraTypeIsValid(t *testing.T) {
	for _, valid := range []models.InfraType{
		models.InfraTypeMill, models.InfraTypeSupplyDepot, models.InfraTypeCastle, models.InfraTypeVillage,
	} {
		if !valid.IsValid() {
			t.Errorf("InfraType %q: want valid", valid)
		}
	}
	for _, invalid := range []models.InfraType{"", "bank", "mill ", "Mill", "castle\t"} {
		if invalid.IsValid() {
			t.Errorf("InfraType %q: want invalid", invalid)
		}
	}
}

func TestNobleStatusIsValid(t *testing.T) {
	for _, valid := range []models.NobleStatus{
		models.NobleStatusFree, models.NobleStatusHostage, models.NobleStatusDungeon,
	} {
		if !valid.IsValid() {
			t.Errorf("NobleStatus %q: want valid", valid)
		}
	}
	for _, invalid := range []models.NobleStatus{"", "captured", "FREE", "free "} {
		if invalid.IsValid() {
			t.Errorf("NobleStatus %q: want invalid", invalid)
		}
	}
}

func TestSex(t *testing.T) {
	for _, valid := range []models.Sex{models.SexMale, models.SexFemale} {
		if !valid.IsValid() {
			t.Errorf("Sex %q: want valid", valid)
		}
	}
	for _, invalid := range []models.Sex{"", "M", "MALE", "other"} {
		if invalid.IsValid() {
			t.Errorf("Sex %q: want invalid", invalid)
		}
		if invalid.CanHoldReligiousOrRoyalTitle() {
			t.Errorf("Sex %q: unknown sex must not hold religious or royal titles", invalid)
		}
	}
	if !models.SexMale.CanHoldReligiousOrRoyalTitle() {
		t.Error("male noble: want eligible for religious and royal titles")
	}
	if models.SexFemale.CanHoldReligiousOrRoyalTitle() {
		t.Error("female noble: want ineligible for religious and royal titles")
	}
}

func TestNobleDisplayName(t *testing.T) {
	male := models.Noble{ID: "N1", Name: "Guillaume de Rosemont", Sex: models.SexMale}
	female := models.Noble{ID: "N2", Name: "Mahaut de Rosemont", Sex: models.SexFemale}
	holds := func(noble models.Noble, title models.FiefTitle) models.Fief {
		id := noble.ID
		return models.Fief{Title: title, HolderNobleID: &id}
	}
	tests := []struct {
		name  string
		fiefs []models.Fief
		noble models.Noble
		want  string
	}{
		{"male without fief", nil, male, "Sieur Guillaume de Rosemont"},
		{"female without fief", nil, female, "Dame Mahaut de Rosemont"},
		{"baron", []models.Fief{holds(male, models.FiefTitleBarony)}, male, "Baron Guillaume de Rosemont"},
		{"baroness", []models.Fief{holds(female, models.FiefTitleBarony)}, female, "Baronne Mahaut de Rosemont"},
		{"count", []models.Fief{holds(male, models.FiefTitleCounty)}, male, "Comte Guillaume de Rosemont"},
		{"countess", []models.Fief{holds(female, models.FiefTitleCounty)}, female, "Comtesse Mahaut de Rosemont"},
		{"marquis", []models.Fief{holds(male, models.FiefTitleMarquisate)}, male, "Marquis Guillaume de Rosemont"},
		{"marquise", []models.Fief{holds(female, models.FiefTitleMarquisate)}, female, "Marquise Mahaut de Rosemont"},
		{"duke", []models.Fief{holds(male, models.FiefTitleDuchy)}, male, "Duc Guillaume de Rosemont"},
		{"duchess", []models.Fief{holds(female, models.FiefTitleDuchy)}, female, "Duchesse Mahaut de Rosemont"},
		{"highest of several fiefs", []models.Fief{holds(male, models.FiefTitleCounty), holds(male, models.FiefTitleDuchy), holds(male, models.FiefTitleBarony)}, male, "Duc Guillaume de Rosemont"},
		{"fief held by someone else", []models.Fief{holds(female, models.FiefTitleDuchy)}, male, "Sieur Guillaume de Rosemont"},
		{"vacant fief", []models.Fief{{Title: models.FiefTitleDuchy}}, male, "Sieur Guillaume de Rosemont"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &models.GameState{Fiefs: tt.fiefs}
			if got := state.NobleDisplayName(tt.noble); got != tt.want {
				t.Errorf("NobleDisplayName = %q, want %q", got, tt.want)
			}
		})
	}
	var nilState *models.GameState
	if got, want := nilState.NobleDisplayName(male), "Sieur Guillaume de Rosemont"; got != want {
		t.Errorf("nil state NobleDisplayName = %q, want %q", got, want)
	}
}

func TestDeathCauseIsValid(t *testing.T) {
	for _, valid := range []models.DeathCause{
		models.DeathCauseNatural, models.DeathCauseExecution, models.DeathCauseAssassination,
	} {
		if !valid.IsValid() {
			t.Errorf("DeathCause %q: want valid", valid)
		}
	}
	for _, invalid := range []models.DeathCause{"", "poison", "NATURAL"} {
		if invalid.IsValid() {
			t.Errorf("DeathCause %q: want invalid", invalid)
		}
	}
}

func TestWinterOrderTypeIsValid(t *testing.T) {
	for _, valid := range []models.WinterOrderType{
		models.WinterOrderTypeRecruitNoble,
		models.WinterOrderTypeRecruitTroop,
		models.WinterOrderTypeBuild,
		models.WinterOrderTypeElectCapital,
		models.WinterOrderTypeLiberateNoble,
		models.WinterOrderTypeHostage,
		models.WinterOrderTypeDungeon,
		models.WinterOrderTypeTransfer,
		models.WinterOrderTypeFoundFief,
		models.WinterOrderTypeAssignFief,
	} {
		if !valid.IsValid() {
			t.Errorf("WinterOrderType %q: want valid", valid)
		}
	}
	for _, invalid := range []models.WinterOrderType{"", "recruit", "E C"} {
		if invalid.IsValid() {
			t.Errorf("WinterOrderType %q: want invalid", invalid)
		}
	}
}

func TestValidateAllowsActionSeasonCacheOutsideSettlement(t *testing.T) {
	state := validState()
	state.TerritoryStates["FOU"] = models.TerritoryState{Resources: 3}
	if err := state.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want ordinary territory cache to be valid", err)
	}
}

func TestSeasonForTurn(t *testing.T) {
	cases := []struct {
		turn int
		want models.Season
	}{
		{1, models.SeasonSpring},
		{2, models.SeasonSummer},
		{3, models.SeasonAutumn},
		{4, models.SeasonWinter},
		{5, models.SeasonSpring},
		{8, models.SeasonWinter},
		{9, models.SeasonSpring},
	}
	for _, tc := range cases {
		if got := models.SeasonForTurn(tc.turn); got != tc.want {
			t.Errorf("SeasonForTurn(%d) = %q, want %q", tc.turn, got, tc.want)
		}
	}
}

func TestYear(t *testing.T) {
	cases := []struct {
		turn int
		want int
	}{
		{1, 1}, {4, 1}, {5, 2}, {8, 2}, {9, 3},
	}
	for _, tc := range cases {
		g := models.NewGameState()
		g.Turn = tc.turn
		if got := g.Year(); got != tc.want {
			t.Errorf("Year() with turn %d = %d, want %d", tc.turn, got, tc.want)
		}
	}
}

func TestNewGameState(t *testing.T) {
	g := models.NewGameState()
	if g.Turn != 1 {
		t.Errorf("Turn = %d, want 1", g.Turn)
	}
	if g.Season != models.SeasonSpring {
		t.Errorf("Season = %q, want %q", g.Season, models.SeasonSpring)
	}
	if g.YearCount != models.DefaultGameYears {
		t.Errorf("YearCount = %d, want %d", g.YearCount, models.DefaultGameYears)
	}
	if g.NextArmyID != 1 {
		t.Errorf("NextArmyID = %d, want 1", g.NextArmyID)
	}
	if err := g.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateYearCount(t *testing.T) {
	for _, yearCount := range []int{-1, models.MaximumGameYears + 1} {
		state := models.NewGameState()
		state.YearCount = yearCount
		if err := state.Validate(); err == nil {
			t.Fatalf("Validate() with year count %d = nil, want error", yearCount)
		}
	}
}

func TestValidateValidState(t *testing.T) {
	g := validState()
	g.Players[0].CapitalCastleID = ptrInfraID("I2")
	if err := g.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateAssignedChain(t *testing.T) {
	g := validState()
	g.Armies[0].ChainID = ptrChainID("C1")
	g.Chains = []models.Chain{{
		ID:           "C1",
		NobleID:      "N1",
		ArmyID:       "A1",
		CurrentIndex: 0,
		Orders: []models.Order{
			{ID: "O1", Type: models.OrderTypeAttack, ArmyID: "A1", PositionID: "ROS", TargetIDs: []models.TerritoryID{"BCL"}, NobleAssignments: nil, Liaison: models.LiaisonModeSingle},
			{ID: "O2", Type: models.OrderTypeDisperse, ArmyID: "A1", PositionID: "BCL", TargetIDs: []models.TerritoryID{"BCL"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BCL": {"HUG"}}, Liaison: models.LiaisonModeLoop},
		},
	}}
	g.NextChainID = 2
	if err := g.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(g *models.GameState)
		want   string
	}{
		{"turn zero", func(g *models.GameState) { g.Turn = 0 }, "turn"},
		{"invalid season", func(g *models.GameState) { g.Season = "snow" }, "invalid season"},
		{"season mismatch with turn", func(g *models.GameState) { g.Turn = 2 }, "season"},
		{"duplicate player id", func(g *models.GameState) {
			g.Players = append(g.Players, models.Player{ID: "P1", Name: "Alpha2", Color: "green"})
		}, "duplicate id"},
		{"duplicate territory id", func(g *models.GameState) {
			g.Territories = append(g.Territories, models.Territory{ID: "ROS", Name: "X", Terrain: models.TerrainPlain})
		}, "duplicate id"},
		{"invalid territory id", func(g *models.GameState) {
			g.Territories[0].ID = "RO1"
		}, "invalid id"},
		{"invalid terrain", func(g *models.GameState) { g.Territories[1].Terrain = "MARSH" }, "invalid terrain"},
		{"asymmetric adjacency", func(g *models.GameState) { g.Territories[0].Adjacencies = []models.TerritoryID{"BCL"} }, "asymmetric"},
		{"adjacency to unknown territory", func(g *models.GameState) {
			g.Territories[0].Adjacencies = []models.TerritoryID{"BCL", "ZZZ"}
		}, "does not exist"},
		{"self adjacency", func(g *models.GameState) {
			g.Territories[0].Adjacencies = []models.TerritoryID{"BCL", "ROS"}
		}, "self-adjacency"},
		{"duplicate adjacency", func(g *models.GameState) {
			g.Territories[0].Adjacencies = []models.TerritoryID{"BCL", "BCL"}
		}, "duplicate adjacency"},
		{"duplicate army id", func(g *models.GameState) {
			g.Armies = append(g.Armies, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "BCL", Size: 1})
		}, "duplicate id"},
		{"army size zero", func(g *models.GameState) { g.Armies[0].Size = 0 }, "size"},
		{"army unknown owner", func(g *models.GameState) { g.Armies[0].OwnerID = "P9" }, "unknown owner"},
		{"army unknown territory", func(g *models.GameState) { g.Armies[0].TerritoryID = "ZZZ" }, "unknown territory"},
		{"army not referenced by territory state", func(g *models.GameState) {
			g.TerritoryStates["ROS"] = models.TerritoryState{Resources: 0, Infrastructures: ptrInfraID("I1")}
		}, "does not reference it"},
		{"state references army stationed elsewhere", func(g *models.GameState) {
			g.TerritoryStates["BCL"] = models.TerritoryState{Resources: 0, Army: ptrArmyID("A1")}
		}, "stationed in"},
		{"state references unknown army", func(g *models.GameState) {
			g.TerritoryStates["BCL"] = models.TerritoryState{Resources: 0, Army: ptrArmyID("A9")}
		}, "unknown army"},
		{"duplicate noble id", func(g *models.GameState) {
			g.Nobles = append(g.Nobles, models.Noble{Sex: models.SexMale, ID: "N1", Code: "ANN", Name: "Anne", OwnerID: "P2", LocationID: "BCL"})
		}, "duplicate id"},
		{"duplicate noble code", func(g *models.GameState) {
			g.Nobles = append(g.Nobles, models.Noble{Sex: models.SexMale, ID: "N2", Code: "HUG", Name: "Hugues II", OwnerID: "P2", LocationID: "BCL"})
		}, "duplicate code"},
		{"noble unknown owner", func(g *models.GameState) { g.Nobles[0].OwnerID = "P9" }, "unknown owner"},
		{"noble unknown territory", func(g *models.GameState) { g.Nobles[0].LocationID = "ZZZ" }, "unknown territory"},
		{"noble invalid sex", func(g *models.GameState) { g.Nobles[0].Sex = "" }, "invalid sex"},
		{"noble invalid status", func(g *models.GameState) { g.Nobles[0].Status = "captured" }, "invalid status"},
		{"noble negative last emission turn", func(g *models.GameState) { g.Nobles[0].LastEmissionTurn = -1 }, "last emission turn"},
		{"noble future last emission turn", func(g *models.GameState) { g.Nobles[0].LastEmissionTurn = g.Turn + 1 }, "last emission turn"},
		{"removed noble duplicate id", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles,
				models.RemovedNoble{Sex: models.SexMale, ID: "N9", Code: "ANN", OwnerID: "P1", Cause: models.DeathCauseNatural},
				models.RemovedNoble{Sex: models.SexMale, ID: "N9", Code: "BEA", OwnerID: "P1", Cause: models.DeathCauseNatural},
			)
		}, "duplicate or empty id"},
		{"removed noble still exists", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles, models.RemovedNoble{Sex: models.SexMale, ID: "N1", Code: "ANN", OwnerID: "P1", Cause: models.DeathCauseNatural})
		}, "still exists"},
		{"removed noble invalid sex", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles, models.RemovedNoble{ID: "N9", Code: "ANN", OwnerID: "P1", Cause: models.DeathCauseNatural})
		}, "invalid sex"},
		{"removed noble invalid code", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles, models.RemovedNoble{Sex: models.SexMale, ID: "N9", Code: "ann", OwnerID: "P1", Cause: models.DeathCauseNatural})
		}, "invalid code"},
		{"removed noble duplicate code", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles, models.RemovedNoble{Sex: models.SexMale, ID: "N9", Code: "HUG", OwnerID: "P1", Cause: models.DeathCauseNatural})
		}, "duplicate code"},
		{"removed noble unknown owner", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles, models.RemovedNoble{Sex: models.SexMale, ID: "N9", Code: "ANN", OwnerID: "P9", Cause: models.DeathCauseNatural})
		}, "unknown owner"},
		{"removed noble invalid cause", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles, models.RemovedNoble{Sex: models.SexMale, ID: "N9", Code: "ANN", OwnerID: "P1", Cause: "poison"})
		}, "invalid death cause"},
		{"removed noble future turn", func(g *models.GameState) {
			g.RemovedNobles = append(g.RemovedNobles, models.RemovedNoble{Sex: models.SexMale, ID: "N9", Code: "ANN", OwnerID: "P1", Cause: models.DeathCauseNatural, Turn: g.Turn + 1})
		}, "must be between 0"},
		{"next chain id zero", func(g *models.GameState) { g.NextChainID = 0 }, "next chain id"},
		{"next army id zero", func(g *models.GameState) { g.NextArmyID = 0 }, "next army id"},
		{"next army id collides with stored army", func(g *models.GameState) { g.NextArmyID = 2 }, "next army id"},
		{"duplicate infrastructure id", func(g *models.GameState) {
			g.Infrastructures = append(g.Infrastructures, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "BCL"})
		}, "duplicate id"},
		{"invalid infra type", func(g *models.GameState) { g.Infrastructures[0].Type = "bank" }, "invalid type"},
		{"infra level zero", func(g *models.GameState) { g.Infrastructures[0].Level = 0 }, "level"},
		{"infra unknown territory", func(g *models.GameState) { g.Infrastructures[0].TerritoryID = "ZZZ" }, "unknown territory"},
		{"capital castle does not exist", func(g *models.GameState) { g.Players[0].CapitalCastleID = ptrInfraID("I9") }, "capital castle"},
		{"capital references noncastle", func(g *models.GameState) { g.Players[0].CapitalCastleID = ptrInfraID("I1") }, "capital infrastructure"},
		{"infra not listed in territory state", func(g *models.GameState) {
			g.TerritoryStates["ROS"] = models.TerritoryState{Resources: 0, Army: ptrArmyID("A1")}
		}, "does not list it"},
		{"state lists infra built elsewhere", func(g *models.GameState) {
			g.TerritoryStates["BCL"] = models.TerritoryState{Resources: 0, Army: ptrArmyID("A2"), Infrastructures: ptrInfraID("I2")}
		}, "built in"},
		{"state lists unknown infra", func(g *models.GameState) {
			g.TerritoryStates["BCL"] = models.TerritoryState{Resources: 0, Army: ptrArmyID("A2"), Infrastructures: ptrInfraID("I9")}
		}, "unknown infrastructure"},
		{"missing territory state entry", func(g *models.GameState) {
			delete(g.TerritoryStates, "FOU")
		}, "missing TerritoryState"},
		{"orphan territory state entry", func(g *models.GameState) {
			g.TerritoryStates["ZZZ"] = models.TerritoryState{}
		}, "references unknown territory"},
		{"negative resources", func(g *models.GameState) {
			g.TerritoryStates["ROS"] = models.TerritoryState{Resources: -1, Army: ptrArmyID("A1"), Infrastructures: ptrInfraID("I1")}
		}, "negative resources"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validState()
			tc.mutate(g)
			err := g.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestValidateChainErrors(t *testing.T) {
	validChain := func(g *models.GameState) {
		g.Armies[0].ChainID = ptrChainID("C1")
		g.Chains = []models.Chain{{
			ID: "C1", NobleID: "N1", ArmyID: "A1", CurrentIndex: 0,
			Orders: []models.Order{{
				ID: "O1", Type: models.OrderTypeAttack, ArmyID: "A1", PositionID: "ROS",
				TargetIDs: []models.TerritoryID{"BCL"}, Liaison: models.LiaisonModeSingle,
			}},
		}}
		g.NextChainID = 2
	}
	cases := []struct {
		name   string
		mutate func(g *models.GameState)
		want   string
	}{
		{"empty chain", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].Orders = nil
		}, "no orders"},
		{"duplicate chain id", func(g *models.GameState) {
			validChain(g)
			g.Chains = append(g.Chains, g.Chains[0])
		}, "duplicate id"},
		{"unknown chain noble", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].NobleID = "N9"
		}, "unknown noble"},
		{"unknown chain army", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].ArmyID = "A9"
		}, "unknown army"},
		{"out of range current index", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].CurrentIndex = 1
		}, "current index"},
		{"invalid order type", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].Orders[0].Type = "march"
		}, "invalid type"},
		{"invalid liaison", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].Orders[0].Liaison = "retry"
		}, "invalid liaison"},
		{"unknown order position", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].Orders[0].PositionID = "ZZZ"
		}, "unknown position"},
		{"unknown order target", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].Orders[0].TargetIDs = []models.TerritoryID{"ZZZ"}
		}, "unknown target"},
		{"chain noble does not own army", func(g *models.GameState) {
			validChain(g)
			g.Chains[0].ArmyID = "A2"
			g.Chains[0].Orders[0].ArmyID = "A2"
			g.Armies[0].ChainID = nil
			g.Armies[1].ChainID = ptrChainID("C1")
		}, "does not own army"},
		{"next chain id collides with stored chain", func(g *models.GameState) {
			validChain(g)
			g.NextChainID = 1
		}, "next chain id"},
		{"missing inverse army link", func(g *models.GameState) {
			validChain(g)
			g.Armies[0].ChainID = nil
		}, "does not reference it"},
		{"unknown army chain link", func(g *models.GameState) {
			validChain(g)
			g.Armies[0].ChainID = ptrChainID("C9")
		}, "references unknown chain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validState()
			tc.mutate(g)
			err := g.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

// TestTrigramInvariants pins the trigram requirement for territories and nobles.
func TestTrigramInvariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(g *models.GameState)
		want   string
	}{
		{"territory with 2-letter id", func(g *models.GameState) { g.Territories[0].ID = "RO" }, "3 uppercase"},
		{"territory with 4-letter id", func(g *models.GameState) { g.Territories[1].ID = "FROS" }, "3 uppercase"},
		{"territory with lowercase id", func(g *models.GameState) { g.Territories[0].ID = "ros" }, "3 uppercase"},
		{"territory id with digit", func(g *models.GameState) { g.Territories[1].ID = "FR0" }, "3 uppercase"},
		{"noble with 4-letter code", func(g *models.GameState) { g.Nobles[0].Code = "HUGU" }, "3 uppercase"},
		{"noble with lowercase code", func(g *models.GameState) { g.Nobles[0].Code = "hug" }, "3 uppercase"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validState()
			tc.mutate(g)
			err := g.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error %q does not contain %q", err, tc.want)
			}
		})
	}
}

// validFiefState extends validState with a 3-territory fief (BRU as capital,
// where the castle already sits, plus BCL and ROS) held by P1 with no
// titleholder, so tests can attach a holder or mutate a single invariant.
func validFiefState() *models.GameState {
	g := validState()
	g.Fiefs = []models.Fief{{
		ID:                 "F1",
		Title:              models.FiefTitleBarony,
		CapitalTerritoryID: "BRU",
		Territories:        []models.TerritoryID{"BRU", "BCL", "ROS"},
		OwnerID:            "P1",
	}}
	return g
}

func TestValidateFiefValid(t *testing.T) {
	g := validFiefState()
	if err := g.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateFiefNilIsAccepted(t *testing.T) {
	g := validState()
	if g.Fiefs != nil {
		t.Fatalf("validState() already sets Fiefs = %#v, want nil", g.Fiefs)
	}
	if err := g.Validate(); err != nil {
		t.Errorf("Validate() with nil Fiefs = %v, want nil", err)
	}
}

func TestValidateFiefWithHolder(t *testing.T) {
	g := validFiefState()
	g.Fiefs[0].HolderNobleID = ptrNobleID("N1")
	if err := g.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateFiefErrors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(g *models.GameState)
		want   string
	}{
		{"empty id", func(g *models.GameState) { g.Fiefs[0].ID = "" }, "empty id"},
		{"duplicate id", func(g *models.GameState) {
			g.Fiefs = append(g.Fiefs, g.Fiefs[0])
		}, "duplicate id"},
		{"too few territories", func(g *models.GameState) {
			g.Fiefs[0].Territories = []models.TerritoryID{"BRU", "BCL"}
		}, "at least 3 territories"},
		{"title does not match size", func(g *models.GameState) {
			g.Fiefs[0].Title = models.FiefTitleCounty
		}, "does not match"},
		{"capital not first", func(g *models.GameState) {
			g.Fiefs[0].Territories = []models.TerritoryID{"BCL", "BRU", "ROS"}
		}, "must be the capital"},
		{"unknown territory", func(g *models.GameState) {
			g.Fiefs[0].Territories = []models.TerritoryID{"BRU", "BCL", "ZZZ"}
		}, "unknown territory"},
		{"duplicate territory", func(g *models.GameState) {
			g.Fiefs[0].Territories = []models.TerritoryID{"BRU", "BCL", "BCL"}
		}, "duplicate territory"},
		{"territory already in another fief", func(g *models.GameState) {
			g.Fiefs = append(g.Fiefs, models.Fief{
				ID: "F2", Title: models.FiefTitleBarony, CapitalTerritoryID: "BRU",
				Territories: []models.TerritoryID{"BRU", "BCL", "ROS"}, OwnerID: "P1",
			})
		}, "already belongs to fief"},
		{"unknown owner", func(g *models.GameState) { g.Fiefs[0].OwnerID = "P9" }, "unknown owner"},
		{"capital castle inside another player's fief", func(g *models.GameState) {
			// A capital is a permanent anchor of its player, but a fief
			// member is controlled by the fief's owner first: the two must
			// not disagree.
			g.Players[1].CapitalCastleID = ptrInfraID("I2")
		}, "is not controlled by its owner"},
		{"unknown holder noble", func(g *models.GameState) {
			g.Fiefs[0].HolderNobleID = ptrNobleID("N9")
		}, "unknown holder noble"},
		{"holder noble owned by another player", func(g *models.GameState) {
			g.Nobles = append(g.Nobles, models.Noble{Sex: models.SexMale, ID: "N2", Code: "ANN", Name: "Anne", OwnerID: "P2", LocationID: "BCL", Status: models.NobleStatusFree})
			g.Fiefs[0].HolderNobleID = ptrNobleID("N2")
		}, "belongs to"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validFiefState()
			tc.mutate(g)
			err := g.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestFiefTitleForSize(t *testing.T) {
	cases := []struct {
		size  int
		title models.FiefTitle
		ok    bool
	}{
		{0, "", false},
		{2, "", false},
		{3, models.FiefTitleBarony, true},
		{4, models.FiefTitleCounty, true},
		{5, models.FiefTitleMarquisate, true},
		{6, models.FiefTitleDuchy, true},
		{9, models.FiefTitleDuchy, true},
	}
	for _, tc := range cases {
		title, ok := models.FiefTitleForSize(tc.size)
		if title != tc.title || ok != tc.ok {
			t.Errorf("FiefTitleForSize(%d) = (%q, %v), want (%q, %v)", tc.size, title, ok, tc.title, tc.ok)
		}
	}
}

func TestGameStateJSONRoundTrip(t *testing.T) {
	g := validState()
	g.Armies[0].ChainID = ptrChainID("C1")
	g.Chains = []models.Chain{{
		ID: "C1", NobleID: "N1", ArmyID: "A1", CurrentIndex: 0,
		Orders: []models.Order{{
			ID: "O1", Type: models.OrderTypeDisperse, ArmyID: "A1", PositionID: "ROS",
			TargetIDs:        []models.TerritoryID{"ROS"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"ROS": {"HUG", "*"}},
			Liaison:          models.LiaisonModeLoop,
		}},
	}}
	g.Nobles[0].LastEmissionTurn = 1
	g.NextChainID = 2
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if bytes.Contains(data, []byte(`"lieuDit"`)) {
		t.Error("JSON still contains the removed lieuDit flag")
	}
	infraJSON, err := json.Marshal(g.Infrastructures[0])
	if err != nil {
		t.Fatalf("Marshal infrastructure: %v", err)
	}
	if bytes.Contains(infraJSON, []byte(`"owner"`)) {
		t.Errorf("infrastructure JSON %s still contains an owner field", infraJSON)
	}
	if !bytes.Contains(data, []byte(`"armies"`)) || !bytes.Contains(data, []byte(`"army":"A1"`)) || !bytes.Contains(data, []byte(`"army":null`)) {
		t.Errorf("GameState JSON %s does not contain the army list and nullable territory army pointers", data)
	}
	var got models.GameState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(*g, got) {
		t.Fatalf("round-trip mismatch:\nwant %+v\ngot  %+v", *g, got)
	}
	if err := got.Validate(); err != nil {
		t.Errorf("Validate() of round-tripped state = %v, want nil", err)
	}
}

func TestNewGameStateJSONEmptyCollections(t *testing.T) {
	data, err := json.Marshal(models.NewGameState())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{
		`"players":[]`, `"territories":[]`, `"nobles":[]`, `"armies":[]`,
		`"chains":[]`, `"infrastructures":[]`, `"territoryStates":{}`,
	} {
		if !bytes.Contains(data, []byte(key)) {
			t.Errorf("JSON %s does not contain %s", data, key)
		}
	}
}
