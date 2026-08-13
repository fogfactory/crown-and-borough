package assetgen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"gopkg.in/yaml.v3"
)

// Balance contains every editable numerical game rule. FirstNames is loaded
// alongside balance.yaml so pure engine resolvers can create nobles without I/O.
type Balance struct {
	BaseProduction     int                    `json:"base_production" yaml:"base_production"`
	SupplyRange        int                    `json:"supply_range" yaml:"supply_range"`
	DepotRangeBonus    int                    `json:"depot_range_bonus" yaml:"depot_range_bonus"`
	InfraRationsBonus  int                    `json:"infra_rations_bonus" yaml:"infra_rations_bonus"`
	CostBase           int                    `json:"cost_base" yaml:"cost_base"`
	PillageBonus       int                    `json:"pillage_bonus" yaml:"pillage_bonus"`
	NobleCommandBonus  int                    `json:"noble_command_bonus" yaml:"noble_command_bonus"`
	CastleDefenseBonus int                    `json:"castle_defense_bonus" yaml:"castle_defense_bonus"`
	RationTerrain      map[models.Terrain]int `json:"ration_terrain" yaml:"ration_terrain"`
	WinterStockDivisor int                    `json:"winter_stock_divisor" yaml:"winter_stock_divisor"`
	VillageStockCap    int                    `json:"village_stock_cap" yaml:"village_stock_cap"`
	CastleStockCap     int                    `json:"castle_stock_cap" yaml:"castle_stock_cap"`
	Costs              Costs                  `json:"costs" yaml:"costs"`
	StartingNobles     int                    `json:"starting_nobles" yaml:"starting_nobles"`
	StartingTroops     int                    `json:"starting_troops" yaml:"starting_troops"`
	StartingResources  int                    `json:"starting_resources" yaml:"starting_resources"`
	FirstNames         []Asset                `json:"-" yaml:"-"`
}

// Costs groups all resource costs used by winter investments.
type Costs struct {
	Castle      int `json:"castle" yaml:"castle"`
	Mill        int `json:"mill" yaml:"mill"`
	Troop       int `json:"troop" yaml:"troop"`
	Noble       int `json:"noble" yaml:"noble"`
	SupplyDepot int `json:"supply_depot" yaml:"supply_depot"`
	Liberation  int `json:"liberation" yaml:"liberation"`
}

type rawBalance struct {
	BaseProduction     *int            `yaml:"base_production"`
	SupplyRange        *int            `yaml:"supply_range"`
	DepotRangeBonus    *int            `yaml:"depot_range_bonus"`
	InfraRationsBonus  *int            `yaml:"infra_rations_bonus"`
	CostBase           *int            `yaml:"cost_base"`
	PillageBonus       *int            `yaml:"pillage_bonus"`
	NobleCommandBonus  *int            `yaml:"noble_command_bonus"`
	CastleDefenseBonus *int            `yaml:"castle_defense_bonus"`
	RationTerrain      map[string]*int `yaml:"ration_terrain"`
	WinterStockDivisor *int            `yaml:"winter_stock_divisor"`
	VillageStockCap    *int            `yaml:"village_stock_cap"`
	CastleStockCap     *int            `yaml:"castle_stock_cap"`
	Costs              *rawCosts       `yaml:"costs"`
	StartingNobles     *int            `yaml:"starting_nobles"`
	StartingTroops     *int            `yaml:"starting_troops"`
	StartingResources  *int            `yaml:"starting_resources"`
}

type rawCosts struct {
	Castle      *int `yaml:"castle"`
	Mill        *int `yaml:"mill"`
	Troop       *int `yaml:"troop"`
	Noble       *int `yaml:"noble"`
	SupplyDepot *int `yaml:"supply_depot"`
	Liberation  *int `yaml:"liberation"`
}

var balanceTerrains = [...]models.Terrain{
	models.TerrainPlain,
	models.TerrainForest,
	models.TerrainHill,
	models.TerrainMountain,
	models.TerrainSwamp,
}

// LoadBalance reads balance.yaml and the noble first-name asset from dir. The
// balance file permits documentation comments while requiring every setting.
func LoadBalance(dir string) (Balance, error) {
	path := filepath.Join(dir, "balance.yaml")
	source, err := os.ReadFile(path)
	if err != nil {
		return Balance{}, fmt.Errorf("assetgen: %s: %w", path, err)
	}
	if len(bytes.TrimSpace(source)) == 0 {
		return Balance{}, fmt.Errorf("assetgen: %s: empty file", path)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(source))
	decoder.KnownFields(true)
	var raw rawBalance
	if err := decoder.Decode(&raw); err != nil {
		if err == io.EOF {
			return Balance{}, fmt.Errorf("assetgen: %s: empty file", path)
		}
		return Balance{}, fmt.Errorf("assetgen: %s: invalid YAML: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Balance{}, fmt.Errorf("assetgen: %s: invalid YAML: multiple documents", path)
		}
		return Balance{}, fmt.Errorf("assetgen: %s: invalid YAML: %w", path, err)
	}

	balance, err := raw.balance(path)
	if err != nil {
		return Balance{}, err
	}
	firstNames, err := loadAssets(filepath.Join(dir, "prenoms.csv"))
	if err != nil {
		return Balance{}, err
	}
	balance.FirstNames = firstNames
	return balance, nil
}

func (raw rawBalance) balance(path string) (Balance, error) {
	baseProduction, err := requiredNonNegativeInt(path, "base_production", raw.BaseProduction)
	if err != nil {
		return Balance{}, err
	}
	supplyRange, err := requiredNonNegativeInt(path, "supply_range", raw.SupplyRange)
	if err != nil {
		return Balance{}, err
	}
	depotRangeBonus, err := requiredNonNegativeInt(path, "depot_range_bonus", raw.DepotRangeBonus)
	if err != nil {
		return Balance{}, err
	}
	infraRationsBonus, err := requiredNonNegativeInt(path, "infra_rations_bonus", raw.InfraRationsBonus)
	if err != nil {
		return Balance{}, err
	}
	costBase, err := requiredPositiveInt(path, "cost_base", raw.CostBase)
	if err != nil {
		return Balance{}, err
	}
	pillageBonus, err := requiredNonNegativeInt(path, "pillage_bonus", raw.PillageBonus)
	if err != nil {
		return Balance{}, err
	}
	nobleCommandBonus, err := requiredNonNegativeInt(path, "noble_command_bonus", raw.NobleCommandBonus)
	if err != nil {
		return Balance{}, err
	}
	castleDefenseBonus, err := requiredNonNegativeInt(path, "castle_defense_bonus", raw.CastleDefenseBonus)
	if err != nil {
		return Balance{}, err
	}
	startingNobles, err := requiredNonNegativeInt(path, "starting_nobles", raw.StartingNobles)
	if err != nil {
		return Balance{}, err
	}
	startingTroops, err := requiredNonNegativeInt(path, "starting_troops", raw.StartingTroops)
	if err != nil {
		return Balance{}, err
	}
	startingResources, err := requiredNonNegativeInt(path, "starting_resources", raw.StartingResources)
	if err != nil {
		return Balance{}, err
	}
	rationTerrain, err := requiredIntTerrainMap(path, "ration_terrain", raw.RationTerrain)
	if err != nil {
		return Balance{}, err
	}
	winterStockDivisor, err := requiredPositiveInt(path, "winter_stock_divisor", raw.WinterStockDivisor)
	if err != nil {
		return Balance{}, err
	}
	villageStockCap, err := requiredNonNegativeInt(path, "village_stock_cap", raw.VillageStockCap)
	if err != nil {
		return Balance{}, err
	}
	castleStockCap, err := requiredNonNegativeInt(path, "castle_stock_cap", raw.CastleStockCap)
	if err != nil {
		return Balance{}, err
	}
	costs, err := raw.costs(path)
	if err != nil {
		return Balance{}, err
	}
	return Balance{
		BaseProduction:     baseProduction,
		SupplyRange:        supplyRange,
		DepotRangeBonus:    depotRangeBonus,
		InfraRationsBonus:  infraRationsBonus,
		CostBase:           costBase,
		PillageBonus:       pillageBonus,
		NobleCommandBonus:  nobleCommandBonus,
		CastleDefenseBonus: castleDefenseBonus,
		RationTerrain:      rationTerrain,
		WinterStockDivisor: winterStockDivisor,
		VillageStockCap:    villageStockCap,
		CastleStockCap:     castleStockCap,
		Costs:              costs,
		StartingNobles:     startingNobles,
		StartingTroops:     startingTroops,
		StartingResources:  startingResources,
	}, nil
}

func (raw rawBalance) costs(path string) (Costs, error) {
	if raw.Costs == nil {
		return Costs{}, missingBalanceValue(path, "costs")
	}
	castle, err := requiredNonNegativeInt(path, "costs.castle", raw.Costs.Castle)
	if err != nil {
		return Costs{}, err
	}
	mill, err := requiredNonNegativeInt(path, "costs.mill", raw.Costs.Mill)
	if err != nil {
		return Costs{}, err
	}
	troop, err := requiredNonNegativeInt(path, "costs.troop", raw.Costs.Troop)
	if err != nil {
		return Costs{}, err
	}
	noble, err := requiredNonNegativeInt(path, "costs.noble", raw.Costs.Noble)
	if err != nil {
		return Costs{}, err
	}
	supplyDepot, err := requiredNonNegativeInt(path, "costs.supply_depot", raw.Costs.SupplyDepot)
	if err != nil {
		return Costs{}, err
	}
	liberation, err := requiredNonNegativeInt(path, "costs.liberation", raw.Costs.Liberation)
	if err != nil {
		return Costs{}, err
	}
	return Costs{
		Castle:      castle,
		Mill:        mill,
		Troop:       troop,
		Noble:       noble,
		SupplyDepot: supplyDepot,
		Liberation:  liberation,
	}, nil
}

func requiredNonNegativeInt(path, name string, value *int) (int, error) {
	if value == nil {
		return 0, missingBalanceValue(path, name)
	}
	if *value < 0 {
		return 0, fmt.Errorf("assetgen: %s: value %q must be >= 0", path, name)
	}
	return *value, nil
}

func requiredPositiveInt(path, name string, value *int) (int, error) {
	if value == nil {
		return 0, missingBalanceValue(path, name)
	}
	if *value <= 0 {
		return 0, fmt.Errorf("assetgen: %s: value %q must be > 0", path, name)
	}
	return *value, nil
}

func missingBalanceValue(path, name string) error {
	return fmt.Errorf("assetgen: %s: missing required value %q", path, name)
}

func requiredIntTerrainMap(path, name string, values map[string]*int) (map[models.Terrain]int, error) {
	if values == nil {
		return nil, missingBalanceValue(path, name)
	}
	result := make(map[models.Terrain]int, len(balanceTerrains))
	for _, terrain := range balanceTerrains {
		value, exists := values[string(terrain)]
		if !exists || value == nil {
			return nil, missingBalanceValue(path, name+"."+string(terrain))
		}
		if *value < 0 {
			return nil, fmt.Errorf("assetgen: %s: value %q must be >= 0", path, name+"."+string(terrain))
		}
		result[terrain] = *value
	}
	for terrain := range values {
		if !isBalanceTerrain(terrain) {
			return nil, fmt.Errorf("assetgen: %s: invalid terrain %q in %s", path, terrain, name)
		}
	}
	return result, nil
}

func isBalanceTerrain(value string) bool {
	for _, terrain := range balanceTerrains {
		if string(terrain) == value {
			return true
		}
	}
	return false
}
