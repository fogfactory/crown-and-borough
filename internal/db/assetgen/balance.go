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
	TerritoryIncome         int                    `json:"territory_income" yaml:"territory_income"`
	VillageIncome           int                    `json:"village_income" yaml:"village_income"`
	SupplyRange             int                    `json:"supply_range" yaml:"supply_range"`
	DepotRangeBonus         int                    `json:"depot_range_bonus" yaml:"depot_range_bonus"`
	CostBase                int                    `json:"cost_base" yaml:"cost_base"`
	PillageBonus            int                    `json:"pillage_bonus" yaml:"pillage_bonus"`
	NobleCommandBonus       int                    `json:"noble_command_bonus" yaml:"noble_command_bonus"`
	CastleDefenseBonus      int                    `json:"castle_defense_bonus" yaml:"castle_defense_bonus"`
	CityDefenseBonus        int                    `json:"city_defense_bonus" yaml:"city_defense_bonus"`
	RationTerrain           map[models.Terrain]int `json:"ration_terrain" yaml:"ration_terrain"`
	WinterStockDivisor      int                    `json:"winter_stock_divisor" yaml:"winter_stock_divisor"`
	VillageStockCap         int                    `json:"village_stock_cap" yaml:"village_stock_cap"`
	CastleStockCap          int                    `json:"castle_stock_cap" yaml:"castle_stock_cap"`
	ProsperityLossThreshold int                    `json:"prosperity_loss_threshold" yaml:"prosperity_loss_threshold"`
	Costs                   Costs                  `json:"costs" yaml:"costs"`
	Victory                 VictoryBalance         `json:"victory" yaml:"victory"`
	Alliance                AllianceBalance        `json:"alliance" yaml:"alliance"`
	NobleLimit              int                    `json:"noble_limit" yaml:"noble_limit"`
	NobleLimitMax           int                    `json:"noble_limit_max" yaml:"noble_limit_max"`
	StartingNobles          int                    `json:"starting_nobles" yaml:"starting_nobles"`
	StartingTroops          int                    `json:"starting_troops" yaml:"starting_troops"`
	StartingOutposts        int                    `json:"starting_outposts" yaml:"starting_outposts"`
	StartingResources       int                    `json:"starting_resources" yaml:"starting_resources"`
	SpecialOrders           SpecialOrdersBalance   `json:"special_orders" yaml:"special_orders"`
	Religion                ReligionBalance        `json:"religion" yaml:"religion"`
	FirstNames              []Asset                `json:"-" yaml:"-"`
}

// VictoryBalance derives the title-score supremacy thresholds from the board
// size (titres.md § Seuil de victoire et fin de partie). A threshold is the
// share of the game territories (SoloTerritoryPercent / AllianceTerritoryPercent)
// a player or alliance should hold through fiefs, divided by ReferenceFiefSize,
// the average number of territories behind one title (a barony has 3, a duchy
// 6 or more). The result is approximate by design: fief sizes vary.
type VictoryBalance struct {
	SoloTerritoryPercent     int `json:"solo_territory_percent" yaml:"solo_territory_percent"`
	AllianceTerritoryPercent int `json:"alliance_territory_percent" yaml:"alliance_territory_percent"`
	ReferenceFiefSize        int `json:"reference_fief_size" yaml:"reference_fief_size"`
}

// AllianceBalance holds the alliance weight parameters (specs/succession.md
// § Poids d'alliance). SuccessionRanks[i] is the weight of the noble at line
// position i; the last entry applies to every later position. TitleRanks maps
// each fief title to its weight (no title weighs 0). DensityBonus is added per
// additional alliance between the same two players, without cap.
type AllianceBalance struct {
	SuccessionRanks []int                    `json:"succession_ranks" yaml:"succession_ranks"`
	TitleRanks      map[models.FiefTitle]int `json:"title_ranks" yaml:"title_ranks"`
	DensityBonus    int                      `json:"density_bonus" yaml:"density_bonus"`
}

// ReligionBalance holds the religious title parameters (specs/religieux.md).
// Two independent caps bound the cardinals: the purchased ones at
// CardinalPurchaseBase plus one per full CardinalPurchasePlayersPerExtra
// players, and the number of cardinal cards in the noble deck at
// CardinalCardBase plus one per full CardinalCardPlayersPerExtra players.
type ReligionBalance struct {
	CardinalCost                    int `json:"cardinal_cost" yaml:"cardinal_cost"`
	CardinalPurchaseBase            int `json:"cardinal_purchase_base" yaml:"cardinal_purchase_base"`
	CardinalPurchasePlayersPerExtra int `json:"cardinal_purchase_players_per_extra" yaml:"cardinal_purchase_players_per_extra"`
	CardinalCardBase                int `json:"cardinal_card_base" yaml:"cardinal_card_base"`
	CardinalCardPlayersPerExtra     int `json:"cardinal_card_players_per_extra" yaml:"cardinal_card_players_per_extra"`
	ExcommunicationsPerWinter       int `json:"excommunications_per_winter" yaml:"excommunications_per_winter"`
	ExcommunicationsPerTargetPlayer int `json:"excommunications_per_target_player" yaml:"excommunications_per_target_player"`
	// AppeasementSuccessRolls and AppeasementDeathRolls are the faces of the
	// d6 of the free revolt appeasement rite: the lowest AppeasementSuccessRolls
	// faces succeed, the highest AppeasementDeathRolls faces fail and kill the
	// cleric, the faces between fail. AppeasementCostBase raised to the size of
	// the rebel army is the price of the safe, paid appeasement.
	AppeasementSuccessRolls int        `json:"appeasement_success_rolls" yaml:"appeasement_success_rolls"`
	AppeasementDeathRolls   int        `json:"appeasement_death_rolls" yaml:"appeasement_death_rolls"`
	AppeasementCostBase     int        `json:"appeasement_cost_base" yaml:"appeasement_cost_base"`
	Votes                   VoteWeight `json:"votes" yaml:"votes"`
}

// VoteWeight is the number of voices of a territory held (Seat for the seat of
// the bishopric in its own bishop election) and of the highest religious title
// of a noble.
type VoteWeight struct {
	Territory int `json:"territory" yaml:"territory"`
	Seat      int `json:"seat" yaml:"seat"`
	Bishop    int `json:"bishop" yaml:"bishop"`
	Cardinal  int `json:"cardinal" yaml:"cardinal"`
	Pope      int `json:"pope" yaml:"pope"`
}

// TitleVotes returns the voices of a noble whose highest religious title is t.
func (v VoteWeight) TitleVotes(t models.ReligiousTitle) int {
	switch t {
	case models.ReligiousTitleBishop:
		return v.Bishop
	case models.ReligiousTitleCardinal:
		return v.Cardinal
	case models.ReligiousTitlePope:
		return v.Pope
	}
	return 0
}

type SpecialOrdersBalance struct {
	HandLimit          int                     `json:"hand_limit" yaml:"hand_limit"`
	DrawOrdersLimit    int                     `json:"draw_orders_limit" yaml:"draw_orders_limit"`
	DeckSize           int                     `json:"deck_size" yaml:"deck_size"`
	CalamityPercentage int                     `json:"calamity_percentage" yaml:"calamity_percentage"`
	CalamitySlots      map[models.Season]int   `json:"calamity_slots" yaml:"calamity_slots"`
	CalamityWeights    map[models.CardKind]int `json:"calamity_weights" yaml:"calamity_weights"`
	BonusWeights       map[models.CardKind]int `json:"bonus_weights" yaml:"bonus_weights"`
	Effects            SpecialOrderEffects     `json:"effects" yaml:"effects"`
}

type SpecialOrderEffects struct {
	PlagueArmyDivisor              int `json:"plague_army_divisor" yaml:"plague_army_divisor"`
	PlagueNobleMortalityPercentage int `json:"plague_noble_mortality_percentage" yaml:"plague_noble_mortality_percentage"`
	RevoltArmyMinSize              int `json:"revolt_army_min_size" yaml:"revolt_army_min_size"`
	RevoltArmyMaxSize              int `json:"revolt_army_max_size" yaml:"revolt_army_max_size"`
}

// Costs groups all resource costs used by winter investments.
type Costs struct {
	Castle           int   `json:"castle" yaml:"castle"`
	MillLevels       []int `json:"mill_levels" yaml:"mill_levels"`
	Troop            int   `json:"troop" yaml:"troop"`
	SupplyDepot      int   `json:"supply_depot" yaml:"supply_depot"`
	FiefPerTerritory int   `json:"fief_per_territory" yaml:"fief_per_territory"`
}

type rawBalance struct {
	TerritoryIncome         *int              `yaml:"territory_income"`
	VillageIncome           *int              `yaml:"village_income"`
	SupplyRange             *int              `yaml:"supply_range"`
	DepotRangeBonus         *int              `yaml:"depot_range_bonus"`
	CostBase                *int              `yaml:"cost_base"`
	PillageBonus            *int              `yaml:"pillage_bonus"`
	NobleCommandBonus       *int              `yaml:"noble_command_bonus"`
	CastleDefenseBonus      *int              `yaml:"castle_defense_bonus"`
	CityDefenseBonus        *int              `yaml:"city_defense_bonus"`
	RationTerrain           map[string]*int   `yaml:"ration_terrain"`
	WinterStockDivisor      *int              `yaml:"winter_stock_divisor"`
	VillageStockCap         *int              `yaml:"village_stock_cap"`
	CastleStockCap          *int              `yaml:"castle_stock_cap"`
	ProsperityLossThreshold *int              `yaml:"prosperity_loss_threshold"`
	Costs                   *rawCosts         `yaml:"costs"`
	Victory                 *rawVictory       `yaml:"victory"`
	Alliance                *rawAlliance      `yaml:"alliance"`
	NobleLimit              *int              `yaml:"noble_limit"`
	NobleLimitMax           *int              `yaml:"noble_limit_max"`
	StartingNobles          *int              `yaml:"starting_nobles"`
	StartingTroops          *int              `yaml:"starting_troops"`
	StartingOutposts        *int              `yaml:"starting_outposts"`
	StartingResources       *int              `yaml:"starting_resources"`
	SpecialOrders           *rawSpecialOrders `yaml:"special_orders"`
	Religion                *rawReligion      `yaml:"religion"`
}

type rawReligion struct {
	CardinalCost                    *int `yaml:"cardinal_cost"`
	CardinalPurchaseBase            *int `yaml:"cardinal_purchase_base"`
	CardinalPurchasePlayersPerExtra *int `yaml:"cardinal_purchase_players_per_extra"`
	CardinalCardBase                *int `yaml:"cardinal_card_base"`
	CardinalCardPlayersPerExtra     *int `yaml:"cardinal_card_players_per_extra"`
	ExcommunicationsPerWinter       *int `yaml:"excommunications_per_winter"`
	ExcommunicationsPerTargetPlayer *int `yaml:"excommunications_per_target_player"`
	AppeasementSuccessRolls         *int `yaml:"appeasement_success_rolls"`
	AppeasementDeathRolls           *int `yaml:"appeasement_death_rolls"`
	AppeasementCostBase             *int `yaml:"appeasement_cost_base"`
	Votes                           *struct {
		Territory *int `yaml:"territory"`
		Seat      *int `yaml:"seat"`
		Bishop    *int `yaml:"bishop"`
		Cardinal  *int `yaml:"cardinal"`
		Pope      *int `yaml:"pope"`
	} `yaml:"votes"`
}

type rawAlliance struct {
	SuccessionRanks []*int          `yaml:"succession_ranks"`
	TitleRanks      map[string]*int `yaml:"title_ranks"`
	DensityBonus    *int            `yaml:"density_bonus"`
}

type rawVictory struct {
	SoloTerritoryPercent     *int `yaml:"solo_territory_percent"`
	AllianceTerritoryPercent *int `yaml:"alliance_territory_percent"`
	ReferenceFiefSize        *int `yaml:"reference_fief_size"`
}

type rawSpecialOrders struct {
	HandLimit          *int                    `yaml:"hand_limit"`
	DrawOrdersLimit    *int                    `yaml:"draw_orders_limit"`
	DeckSize           *int                    `yaml:"deck_size"`
	CalamityPercentage *int                    `yaml:"calamity_percentage"`
	CalamitySlots      map[string]*int         `yaml:"calamity_slots"`
	CalamityWeights    map[string]*int         `yaml:"calamity_weights"`
	BonusWeights       map[string]*int         `yaml:"bonus_weights"`
	Effects            *rawSpecialOrderEffects `yaml:"effects"`
}

type rawSpecialOrderEffects struct {
	PlagueArmyDivisor              *int `yaml:"plague_army_divisor"`
	PlagueNobleMortalityPercentage *int `yaml:"plague_noble_mortality_percentage"`
	RevoltArmyMinSize              *int `yaml:"revolt_army_min_size"`
	RevoltArmyMaxSize              *int `yaml:"revolt_army_max_size"`
}

type rawCosts struct {
	Castle           *int   `yaml:"castle"`
	MillLevels       []*int `yaml:"mill_levels"`
	Troop            *int   `yaml:"troop"`
	SupplyDepot      *int   `yaml:"supply_depot"`
	FiefPerTerritory *int   `yaml:"fief_per_territory"`
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
	territoryIncome, err := requiredNonNegativeInt(path, "territory_income", raw.TerritoryIncome)
	if err != nil {
		return Balance{}, err
	}
	villageIncome, err := requiredNonNegativeInt(path, "village_income", raw.VillageIncome)
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
	cityDefenseBonus, err := requiredNonNegativeInt(path, "city_defense_bonus", raw.CityDefenseBonus)
	if err != nil {
		return Balance{}, err
	}
	nobleLimit, err := requiredPositiveInt(path, "noble_limit", raw.NobleLimit)
	if err != nil {
		return Balance{}, err
	}
	nobleLimitMax, err := requiredPositiveInt(path, "noble_limit_max", raw.NobleLimitMax)
	if err != nil {
		return Balance{}, err
	}
	if nobleLimitMax < nobleLimit {
		return Balance{}, fmt.Errorf("assetgen: %s: noble_limit_max (%d) must be >= noble_limit (%d)", path, nobleLimitMax, nobleLimit)
	}
	startingNobles, err := requiredNonNegativeInt(path, "starting_nobles", raw.StartingNobles)
	if err != nil {
		return Balance{}, err
	}
	if startingNobles > nobleLimit {
		return Balance{}, fmt.Errorf("assetgen: %s: starting_nobles (%d) must be <= noble_limit (%d)", path, startingNobles, nobleLimit)
	}
	startingTroops, err := requiredNonNegativeInt(path, "starting_troops", raw.StartingTroops)
	if err != nil {
		return Balance{}, err
	}
	startingOutposts, err := requiredNonNegativeInt(path, "starting_outposts", raw.StartingOutposts)
	if err != nil {
		return Balance{}, err
	}
	startingResources, err := requiredNonNegativeInt(path, "starting_resources", raw.StartingResources)
	if err != nil {
		return Balance{}, err
	}
	specialOrders, err := raw.specialOrders(path)
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
	prosperityLossThreshold, err := requiredPositiveInt(path, "prosperity_loss_threshold", raw.ProsperityLossThreshold)
	if err != nil {
		return Balance{}, err
	}
	costs, err := raw.costs(path)
	if err != nil {
		return Balance{}, err
	}
	victory, err := raw.victory(path)
	if err != nil {
		return Balance{}, err
	}
	alliance, err := raw.alliance(path)
	if err != nil {
		return Balance{}, err
	}
	religion, err := raw.religion(path)
	if err != nil {
		return Balance{}, err
	}
	return Balance{
		TerritoryIncome:         territoryIncome,
		VillageIncome:           villageIncome,
		SupplyRange:             supplyRange,
		DepotRangeBonus:         depotRangeBonus,
		CostBase:                costBase,
		PillageBonus:            pillageBonus,
		NobleCommandBonus:       nobleCommandBonus,
		CastleDefenseBonus:      castleDefenseBonus,
		CityDefenseBonus:        cityDefenseBonus,
		RationTerrain:           rationTerrain,
		WinterStockDivisor:      winterStockDivisor,
		VillageStockCap:         villageStockCap,
		CastleStockCap:          castleStockCap,
		ProsperityLossThreshold: prosperityLossThreshold,
		Costs:                   costs,
		Victory:                 victory,
		Alliance:                alliance,
		NobleLimit:              nobleLimit,
		NobleLimitMax:           nobleLimitMax,
		StartingNobles:          startingNobles,
		StartingTroops:          startingTroops,
		StartingOutposts:        startingOutposts,
		StartingResources:       startingResources,
		SpecialOrders:           specialOrders,
		Religion:                religion,
	}, nil
}

func (raw rawBalance) religion(path string) (ReligionBalance, error) {
	if raw.Religion == nil {
		return ReligionBalance{}, missingBalanceValue(path, "religion")
	}
	r := raw.Religion
	var out ReligionBalance
	var err error
	if out.CardinalCost, err = requiredPositiveInt(path, "religion.cardinal_cost", r.CardinalCost); err != nil {
		return ReligionBalance{}, err
	}
	if out.CardinalPurchaseBase, err = requiredPositiveInt(path, "religion.cardinal_purchase_base", r.CardinalPurchaseBase); err != nil {
		return ReligionBalance{}, err
	}
	if out.CardinalPurchasePlayersPerExtra, err = requiredPositiveInt(path, "religion.cardinal_purchase_players_per_extra", r.CardinalPurchasePlayersPerExtra); err != nil {
		return ReligionBalance{}, err
	}
	if out.CardinalCardBase, err = requiredPositiveInt(path, "religion.cardinal_card_base", r.CardinalCardBase); err != nil {
		return ReligionBalance{}, err
	}
	if out.CardinalCardPlayersPerExtra, err = requiredPositiveInt(path, "religion.cardinal_card_players_per_extra", r.CardinalCardPlayersPerExtra); err != nil {
		return ReligionBalance{}, err
	}
	if out.ExcommunicationsPerWinter, err = requiredPositiveInt(path, "religion.excommunications_per_winter", r.ExcommunicationsPerWinter); err != nil {
		return ReligionBalance{}, err
	}
	if out.ExcommunicationsPerTargetPlayer, err = requiredPositiveInt(path, "religion.excommunications_per_target_player", r.ExcommunicationsPerTargetPlayer); err != nil {
		return ReligionBalance{}, err
	}
	if out.AppeasementSuccessRolls, err = requiredPositiveInt(path, "religion.appeasement_success_rolls", r.AppeasementSuccessRolls); err != nil {
		return ReligionBalance{}, err
	}
	if out.AppeasementDeathRolls, err = requiredPositiveInt(path, "religion.appeasement_death_rolls", r.AppeasementDeathRolls); err != nil {
		return ReligionBalance{}, err
	}
	if out.AppeasementSuccessRolls+out.AppeasementDeathRolls > 6 {
		return ReligionBalance{}, fmt.Errorf("assetgen: %s: religion.appeasement_success_rolls and appeasement_death_rolls must fit on a d6", path)
	}
	if out.AppeasementCostBase, err = requiredPositiveInt(path, "religion.appeasement_cost_base", r.AppeasementCostBase); err != nil {
		return ReligionBalance{}, err
	}
	if r.Votes == nil {
		return ReligionBalance{}, missingBalanceValue(path, "religion.votes")
	}
	votes := r.Votes
	if out.Votes.Territory, err = requiredPositiveInt(path, "religion.votes.territory", votes.Territory); err != nil {
		return ReligionBalance{}, err
	}
	if out.Votes.Seat, err = requiredPositiveInt(path, "religion.votes.seat", votes.Seat); err != nil {
		return ReligionBalance{}, err
	}
	if out.Votes.Seat < out.Votes.Territory {
		return ReligionBalance{}, fmt.Errorf("assetgen: %s: religion.votes.seat must not be below religion.votes.territory", path)
	}
	if out.Votes.Bishop, err = requiredPositiveInt(path, "religion.votes.bishop", votes.Bishop); err != nil {
		return ReligionBalance{}, err
	}
	if out.Votes.Cardinal, err = requiredPositiveInt(path, "religion.votes.cardinal", votes.Cardinal); err != nil {
		return ReligionBalance{}, err
	}
	if out.Votes.Pope, err = requiredPositiveInt(path, "religion.votes.pope", votes.Pope); err != nil {
		return ReligionBalance{}, err
	}
	if out.Votes.Bishop > out.Votes.Cardinal || out.Votes.Cardinal > out.Votes.Pope {
		return ReligionBalance{}, fmt.Errorf("assetgen: %s: religion.votes must not decrease from bishop to cardinal to pope", path)
	}
	return out, nil
}

func (raw rawBalance) victory(path string) (VictoryBalance, error) {
	if raw.Victory == nil {
		return VictoryBalance{}, missingBalanceValue(path, "victory")
	}
	solo, err := requiredPositiveInt(path, "victory.solo_territory_percent", raw.Victory.SoloTerritoryPercent)
	if err != nil {
		return VictoryBalance{}, err
	}
	alliance, err := requiredPositiveInt(path, "victory.alliance_territory_percent", raw.Victory.AllianceTerritoryPercent)
	if err != nil {
		return VictoryBalance{}, err
	}
	size, err := requiredPositiveInt(path, "victory.reference_fief_size", raw.Victory.ReferenceFiefSize)
	if err != nil {
		return VictoryBalance{}, err
	}
	if solo > 100 || alliance > 100 {
		return VictoryBalance{}, fmt.Errorf("%s: victory territory percentages must not exceed 100", path)
	}
	if alliance <= solo {
		return VictoryBalance{}, fmt.Errorf("%s: victory.alliance_territory_percent must be strictly above victory.solo_territory_percent", path)
	}
	return VictoryBalance{SoloTerritoryPercent: solo, AllianceTerritoryPercent: alliance, ReferenceFiefSize: size}, nil
}

func (raw rawBalance) alliance(path string) (AllianceBalance, error) {
	if raw.Alliance == nil {
		return AllianceBalance{}, missingBalanceValue(path, "alliance")
	}
	if len(raw.Alliance.SuccessionRanks) == 0 {
		return AllianceBalance{}, missingBalanceValue(path, "alliance.succession_ranks")
	}
	ranks := make([]int, len(raw.Alliance.SuccessionRanks))
	for index, value := range raw.Alliance.SuccessionRanks {
		rank, err := requiredNonNegativeInt(path, fmt.Sprintf("alliance.succession_ranks[%d]", index), value)
		if err != nil {
			return AllianceBalance{}, err
		}
		if index > 0 && rank > ranks[index-1] {
			return AllianceBalance{}, fmt.Errorf("assetgen: %s: alliance.succession_ranks must not increase along the line", path)
		}
		ranks[index] = rank
	}
	titles := []models.FiefTitle{models.FiefTitleBarony, models.FiefTitleCounty, models.FiefTitleMarquisate, models.FiefTitleDuchy}
	titleRanks := make(map[models.FiefTitle]int, len(titles))
	for key := range raw.Alliance.TitleRanks {
		if !models.FiefTitle(key).IsValid() {
			return AllianceBalance{}, fmt.Errorf("assetgen: %s: alliance.title_ranks: unknown fief title %q", path, key)
		}
	}
	for _, title := range titles {
		rank, err := requiredNonNegativeInt(path, "alliance.title_ranks."+string(title), raw.Alliance.TitleRanks[string(title)])
		if err != nil {
			return AllianceBalance{}, err
		}
		titleRanks[title] = rank
	}
	density, err := requiredNonNegativeInt(path, "alliance.density_bonus", raw.Alliance.DensityBonus)
	if err != nil {
		return AllianceBalance{}, err
	}
	return AllianceBalance{SuccessionRanks: ranks, TitleRanks: titleRanks, DensityBonus: density}, nil
}

func (raw rawBalance) costs(path string) (Costs, error) {
	if raw.Costs == nil {
		return Costs{}, missingBalanceValue(path, "costs")
	}
	castle, err := requiredNonNegativeInt(path, "costs.castle", raw.Costs.Castle)
	if err != nil {
		return Costs{}, err
	}
	if raw.Costs.MillLevels == nil || len(raw.Costs.MillLevels) == 0 {
		return Costs{}, missingBalanceValue(path, "costs.mill_levels")
	}
	millLevels := make([]int, len(raw.Costs.MillLevels))
	for index, value := range raw.Costs.MillLevels {
		millLevels[index], err = requiredNonNegativeInt(path, fmt.Sprintf("costs.mill_levels[%d]", index), value)
		if err != nil {
			return Costs{}, err
		}
	}
	troop, err := requiredNonNegativeInt(path, "costs.troop", raw.Costs.Troop)
	if err != nil {
		return Costs{}, err
	}
	supplyDepot, err := requiredNonNegativeInt(path, "costs.supply_depot", raw.Costs.SupplyDepot)
	if err != nil {
		return Costs{}, err
	}
	fiefPerTerritory, err := requiredNonNegativeInt(path, "costs.fief_per_territory", raw.Costs.FiefPerTerritory)
	if err != nil {
		return Costs{}, err
	}
	return Costs{
		Castle:           castle,
		MillLevels:       millLevels,
		Troop:            troop,
		SupplyDepot:      supplyDepot,
		FiefPerTerritory: fiefPerTerritory,
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
