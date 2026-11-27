package engine

import (
	"slices"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// newForecastContext builds the disposable context the command post
// projections run on. It applies the weather of the current season the way the
// resolution would: the calamities already scheduled (announced to every
// player) and the regional bonus cards (fair weather, abundant harvest) in
// deckOrders, which cancel them or double the production. Nothing else the
// deck orders do is projected here (taxes are added by the interface, and the
// plague, revolts and other event cards are never anticipated).
func newForecastContext(
	state *models.GameState,
	balance assetgen.Balance,
	deckOrders map[models.PlayerID][]models.DeckOrder,
) *resolutionContext {
	ctx := newResolutionContext(cloneGameState(state), balance)
	ctx.forecast = true
	bonuses := make(map[models.PlayerID][]models.DeckOrder, len(deckOrders))
	for playerID, playerOrders := range deckOrders {
		for _, order := range playerOrders {
			if order.Type == models.DeckOrderTypePlay && order.Kind.IsRegionalBonus() {
				bonuses[playerID] = append(bonuses[playerID], order)
			}
		}
	}
	if len(bonuses) != 0 {
		resolveDeckOrders(ctx, bonuses)
	}
	resolveSeasonEffects(ctx)
	return ctx
}

// ForecastEffect is one weather effect behind the command post projections,
// as the forecast context applied it.
type ForecastEffect struct {
	Kind   ForecastEffectKind `json:"kind"`
	Region models.TerritoryID `json:"region"`
	// Income and Rations are what the effect adds (positive) or costs
	// (negative) the viewer in the region: territory, village and mill income
	// in R, and terrain rations. Zero for a canceled calamity.
	Income  int `json:"income,omitempty"`
	Rations int `json:"rations,omitempty"`
}

// ForecastEffectKind names what happens to a region's production.
type ForecastEffectKind string

const (
	// ForecastEffectFamine: the bad harvest suppresses territory income and
	// terrain rations.
	ForecastEffectFamine ForecastEffectKind = "famine"
	// ForecastEffectFamineCanceled: an abundant harvest card cancels it.
	ForecastEffectFamineCanceled ForecastEffectKind = "famine_canceled"
	// ForecastEffectAbundantHarvest doubles territory income and rations.
	ForecastEffectAbundantHarvest ForecastEffectKind = "abundant_harvest"
	// ForecastEffectBadWeather halts the mills.
	ForecastEffectBadWeather ForecastEffectKind = "bad_weather"
	// ForecastEffectBadWeatherCanceled: a fair weather card cancels it.
	ForecastEffectBadWeatherCanceled ForecastEffectKind = "bad_weather_canceled"
	// ForecastEffectFairWeather doubles the mills' production.
	ForecastEffectFairWeather ForecastEffectKind = "fair_weather"
)

// forecastEffectOrder is the order the effects of one region are listed in.
var forecastEffectOrder = []ForecastEffectKind{
	ForecastEffectFamine, ForecastEffectFamineCanceled, ForecastEffectAbundantHarvest,
	ForecastEffectBadWeather, ForecastEffectBadWeatherCanceled, ForecastEffectFairWeather,
}

// ForecastWeatherEffects lists the weather effects the projections apply for
// the current season, sorted by region: the scheduled calamities, the ones the
// drafted bonus cards cancel, and the bonuses they add. It explains every
// difference with the normal figures, from the same computation that produces
// them. Winter has none. The amounts are the viewer's own (none without one).
func ForecastWeatherEffects(
	state *models.GameState,
	balance assetgen.Balance,
	deckOrders map[models.PlayerID][]models.DeckOrder,
	viewer *models.PlayerID,
) []ForecastEffect {
	if state == nil || state.Season == models.SeasonWinter {
		return []ForecastEffect{}
	}
	ctx := newForecastContext(state, balance, deckOrders)
	effects := []ForecastEffect{}
	addAll := func(kind ForecastEffectKind, regions map[models.TerritoryID]bool) {
		for region := range regions {
			effects = append(effects, ForecastEffect{Kind: kind, Region: region})
		}
	}
	addAll(ForecastEffectFamine, ctx.famineRegions)
	addAll(ForecastEffectAbundantHarvest, ctx.goodHarvestRegions)
	addAll(ForecastEffectBadWeather, ctx.badWeatherRegions)
	addAll(ForecastEffectFairWeather, ctx.fairWeatherRegions)
	for _, event := range ctx.events {
		if event.Type != EventTypeCalamityCanceled {
			continue
		}
		switch event.CardKind {
		case models.CardKindFamine:
			effects = append(effects, ForecastEffect{Kind: ForecastEffectFamineCanceled, Region: event.RegionSeed})
		case models.CardKindBadWeather:
			effects = append(effects, ForecastEffect{Kind: ForecastEffectBadWeatherCanceled, Region: event.RegionSeed})
		}
	}
	if viewer != nil && len(effects) != 0 {
		deltas := viewerWeatherDeltas(ctx, *viewer)
		for index := range effects {
			effects[index].Income, effects[index].Rations = deltas[effects[index].Region].amounts(effects[index].Kind)
		}
	}
	sort.Slice(effects, func(i, j int) bool {
		if effects[i].Region != effects[j].Region {
			return effects[i].Region < effects[j].Region
		}
		return slices.Index(forecastEffectOrder, effects[i].Kind) < slices.Index(forecastEffectOrder, effects[j].Kind)
	})
	return effects
}

// weatherDelta is what the weather adds (positive) or costs (negative) the
// viewer in one region: territory and village income, mill production and
// terrain rations. A region never has both a suppression and a doubling.
type weatherDelta struct {
	income, mills, rations int
}

// viewerWeatherDeltas sums, per region, the production of the territories and
// mills the viewer controls that the weather of ctx suppresses or doubles.
func viewerWeatherDeltas(ctx *resolutionContext, viewer models.PlayerID) map[models.TerritoryID]weatherDelta {
	deltas := make(map[models.TerritoryID]weatherDelta)
	controlled := func(territoryID models.TerritoryID) bool {
		owner := controllerPointer(ctx.controllerNow, territoryID)
		return owner != nil && *owner == viewer
	}
	for _, territory := range ctx.state.Territories {
		if !controlled(territory.ID) {
			continue
		}
		income := territoryIncomeParts(ctx, territory.ID)
		rations := rationProductionBreakdown(ctx, territory.ID)
		region := regionForTerritory(ctx, territory.ID)
		delta := deltas[region]
		delta.income += income.bonus - income.suppressed
		delta.rations += rations.bonus - rations.suppressed
		deltas[region] = delta
	}
	for _, mill := range computeMillProduction(ctx) {
		if !controlled(mill.millID) {
			continue
		}
		region := regionForTerritory(ctx, mill.millID)
		delta := deltas[region]
		delta.mills += mill.bonus - mill.suppressed
		deltas[region] = delta
	}
	return deltas
}

// amounts returns the income and rations the effect adds (positive) or costs
// (negative) in its region; a canceled calamity changes nothing.
func (delta weatherDelta) amounts(kind ForecastEffectKind) (income, rations int) {
	switch kind {
	case ForecastEffectFamine, ForecastEffectAbundantHarvest:
		return delta.income, delta.rations
	case ForecastEffectBadWeather, ForecastEffectFairWeather:
		return delta.mills, 0
	}
	return 0, 0
}
