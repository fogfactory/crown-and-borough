package assetgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Balance contains every editable numerical game rule. FirstNames is loaded
// alongside balance.json so pure engine resolvers can create nobles without I/O.
type Balance struct {
	BaseProduction     int                    `json:"base_production"`
	SupplyRange        int                    `json:"supply_range"`
	DepotRangeBonus    int                    `json:"depot_range_bonus"`
	InfraRationsBonus  int                    `json:"infra_rations_bonus"`
	CostBase           int                    `json:"cost_base"`
	PillageBonus       int                    `json:"pillage_bonus"`
	CastleDefenseBonus int                    `json:"castle_defense_bonus"`
	RationTerrain      map[models.Terrain]int `json:"ration_terrain"`
	WinterStockDivisor int                    `json:"winter_stock_divisor"`
	VillageStockCap    int                    `json:"village_stock_cap"`
	CastleStockCap     int                    `json:"castle_stock_cap"`
	Costs              Costs                  `json:"costs"`
	Travel             Travel                 `json:"travel"`
	StartingNobles     int                    `json:"starting_nobles"`
	StartingTroops     int                    `json:"starting_troops"`
	StartingResources  int                    `json:"starting_resources"`
	FirstNames         []Asset                `json:"-"`
}

// Costs groups all resource costs used by winter investments.
type Costs struct {
	Castle      int `json:"castle"`
	Mill        int `json:"mill"`
	Troop       int `json:"troop"`
	Noble       int `json:"noble"`
	PostRelay   int `json:"post_relay"`
	Watchtower  int `json:"watchtower"`
	SupplyDepot int `json:"supply_depot"`
	Liberation  int `json:"liberation"`
}

// Travel contains P2 messenger values that must remain editable from the
// shared balance asset even before the travel resolver exists.
type Travel struct {
	TerrainCosts map[models.Terrain]float64 `json:"terrain_costs"`
	RelayDivisor float64                    `json:"relay_divisor"`
}

type rawBalance struct {
	BaseProduction     *int            `json:"base_production"`
	SupplyRange        *int            `json:"supply_range"`
	DepotRangeBonus    *int            `json:"depot_range_bonus"`
	InfraRationsBonus  *int            `json:"infra_rations_bonus"`
	CostBase           *int            `json:"cost_base"`
	PillageBonus       *int            `json:"pillage_bonus"`
	CastleDefenseBonus *int            `json:"castle_defense_bonus"`
	RationTerrain      map[string]*int `json:"ration_terrain"`
	WinterStockDivisor *int            `json:"winter_stock_divisor"`
	VillageStockCap    *int            `json:"village_stock_cap"`
	CastleStockCap     *int            `json:"castle_stock_cap"`
	Costs              *rawCosts       `json:"costs"`
	Travel             *rawTravel      `json:"travel"`
	StartingNobles     *int            `json:"starting_nobles"`
	StartingTroops     *int            `json:"starting_troops"`
	StartingResources  *int            `json:"starting_resources"`
}

type rawCosts struct {
	Castle      *int `json:"castle"`
	Mill        *int `json:"mill"`
	Troop       *int `json:"troop"`
	Noble       *int `json:"noble"`
	PostRelay   *int `json:"post_relay"`
	Watchtower  *int `json:"watchtower"`
	SupplyDepot *int `json:"supply_depot"`
	Liberation  *int `json:"liberation"`
}

type rawTravel struct {
	TerrainCosts map[string]*float64 `json:"terrain_costs"`
	RelayDivisor *float64            `json:"relay_divisor"`
}

var balanceTerrains = [...]models.Terrain{
	models.TerrainPlain,
	models.TerrainForest,
	models.TerrainHill,
	models.TerrainMountain,
	models.TerrainSwamp,
}

// LoadBalance reads balance.json and the noble first-name asset from dir. The
// balance file permits documentation comments while requiring every setting.
func LoadBalance(dir string) (Balance, error) {
	path := filepath.Join(dir, "balance.json")
	source, err := os.ReadFile(path)
	if err != nil {
		return Balance{}, fmt.Errorf("assetgen: %s: %w", path, err)
	}
	stripped, err := stripJSONComments(source)
	if err != nil {
		return Balance{}, fmt.Errorf("assetgen: %s: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(stripped))
	decoder.DisallowUnknownFields()
	var raw rawBalance
	if err := decoder.Decode(&raw); err != nil {
		return Balance{}, fmt.Errorf("assetgen: %s: invalid JSON: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Balance{}, fmt.Errorf("assetgen: %s: invalid JSON: multiple values", path)
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
	travel, err := raw.travel(path)
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
		CastleDefenseBonus: castleDefenseBonus,
		RationTerrain:      rationTerrain,
		WinterStockDivisor: winterStockDivisor,
		VillageStockCap:    villageStockCap,
		CastleStockCap:     castleStockCap,
		Costs:              costs,
		Travel:             travel,
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
	postRelay, err := requiredNonNegativeInt(path, "costs.post_relay", raw.Costs.PostRelay)
	if err != nil {
		return Costs{}, err
	}
	watchtower, err := requiredNonNegativeInt(path, "costs.watchtower", raw.Costs.Watchtower)
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
		PostRelay:   postRelay,
		Watchtower:  watchtower,
		SupplyDepot: supplyDepot,
		Liberation:  liberation,
	}, nil
}

func (raw rawBalance) travel(path string) (Travel, error) {
	if raw.Travel == nil {
		return Travel{}, missingBalanceValue(path, "travel")
	}
	if raw.Travel.RelayDivisor == nil {
		return Travel{}, missingBalanceValue(path, "travel.relay_divisor")
	}
	if *raw.Travel.RelayDivisor <= 0 {
		return Travel{}, fmt.Errorf("assetgen: %s: value %q must be > 0", path, "travel.relay_divisor")
	}
	terrainCosts, err := requiredFloatTerrainMap(path, "travel.terrain_costs", raw.Travel.TerrainCosts)
	if err != nil {
		return Travel{}, err
	}
	return Travel{TerrainCosts: terrainCosts, RelayDivisor: *raw.Travel.RelayDivisor}, nil
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

func requiredFloatTerrainMap(path, name string, values map[string]*float64) (map[models.Terrain]float64, error) {
	if values == nil {
		return nil, missingBalanceValue(path, name)
	}
	result := make(map[models.Terrain]float64, len(balanceTerrains))
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

func stripJSONComments(source []byte) ([]byte, error) {
	stripped := make([]byte, 0, len(source))
	inString := false
	escaped := false
	for index := 0; index < len(source); {
		character := source[index]
		if inString {
			stripped = append(stripped, character)
			index++
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		if character == '"' {
			inString = true
			stripped = append(stripped, character)
			index++
			continue
		}
		if character != '/' || index+1 == len(source) {
			stripped = append(stripped, character)
			index++
			continue
		}
		next := source[index+1]
		switch next {
		case '/':
			index += 2
			for index < len(source) && source[index] != '\n' {
				index++
			}
		case '*':
			index += 2
			closed := false
			for index < len(source) {
				if source[index] == '\n' {
					stripped = append(stripped, '\n')
				}
				if source[index] == '*' && index+1 < len(source) && source[index+1] == '/' {
					index += 2
					closed = true
					break
				}
				index++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated block comment")
			}
		default:
			stripped = append(stripped, character)
			index++
		}
	}
	return stripped, nil
}
