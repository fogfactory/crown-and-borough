package engine

import (
	"slices"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// incomeGroup is the report grouping of territory income inside one
// destination: one line per fief, and one line per region for the territories
// held outside any fief (region is empty for a fief line).
type incomeGroup struct {
	fief   models.FiefID
	region models.TerritoryID
}

// territoryIncomeReport aggregates one player's territory income by
// (destination, incomeGroup) for the turn report: one line per fief, and one
// per region for the territories held outside any fief. Destinations differ per territory
// only when the player controls no valid capital: each territory then finds
// its own closest controlled castle or village (see
// territoryIncomeDestination), so several destinations can appear for the
// same player the same turn. group.fief is empty for territory income routed
// outside any fief; it is otherwise kept as a distinct grouping key from
// destinationID so a fief whose capital happens to also be the player's own
// capital still gets its own report line (titres.md, #196).
type territoryIncomeReport struct {
	ownerID       models.PlayerID
	group         incomeGroup
	members       []models.TerritoryID
	destinationID models.TerritoryID
	fiefTitle     models.FiefTitle
	territories   int
	villages      int
	villageIncome []IncomeVillage
	// territoryAmount is the part of base+bonus the territories bring, villages apart.
	territoryAmount int
	base            int
	bonus           int
	suppressed      int
	lost            bool
}

// TerritoryIncomeForecast is one controlled territory's projected income and
// its destination, for the front's per-territory "rapporte X R à YYY" line.
// Amount is the harvest-adjusted credited amount (0 when suppressed);
// Destination is empty when the income would be lost.
type TerritoryIncomeForecast struct {
	Amount      int
	Destination models.TerritoryID
}

// resolveTerritoryIncome credits every controlled territory's income to its
// destination first within the end-of-turn ravitaillement block resolveSupply
// runs (#208), before rations and mill production, on the territories'
// post-combat, post-movement, post-control ownership: a territory captured
// earlier this same turn already credits its new owner. It is never called in
// winter (see ResolveWinter, and Resolve's guard against winter states).
// Income bypasses the supply network entirely, so it cannot be intercepted;
// the harvest calamity and bonus apply in the income-producing territory's
// own region, exactly as they do for any other source's base production.
func resolveTerritoryIncome(ctx *resolutionContext) {
	reports, _ := computeTerritoryIncome(ctx)
	creditTerritoryIncome(ctx, reports)
	emitTerritoryIncomeEvents(ctx, reports)
	resolveRoyalIncome(ctx)
}

// computeTerritoryIncome computes, but does not credit, every controlled
// territory's harvest-adjusted income, both grouped by (owner, destination,
// fief) for the turn report and individually per territory for the front's
// per-territory forecast. It is the seam both ForecastIncome and
// ForecastTerritoryIncome reuse to project income without touching state.
func computeTerritoryIncome(ctx *resolutionContext) (map[models.PlayerID]map[models.TerritoryID]map[incomeGroup]*territoryIncomeReport, map[models.TerritoryID]TerritoryIncomeForecast) {
	reports := make(map[models.PlayerID]map[models.TerritoryID]map[incomeGroup]*territoryIncomeReport)
	perTerritory := make(map[models.TerritoryID]TerritoryIncomeForecast)
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, territoryID := range sortedStateTerritoryIDs(ctx) {
			if !controlledBy(ctx.controllerNow, ownerID, territoryID) {
				continue
			}
			village := ctx.infrastructureAt(territoryID)
			parts := territoryIncomeParts(ctx, territoryID)
			fief := ctx.territoryFief(ownerID, territoryID)
			taxed := fief != nil && ctx.taxedFiefsThisTurn[fief.ID] && ctx.taxedFiefOwnerAtApply[fief.ID] == fief.OwnerID
			if taxed {
				// Seigneurial tax doubles the fief's territorial income,
				// village included, for the turn (titres.md "Taxe
				// seigneuriale"); it never touches mill production, which
				// this parts value does not carry. Income now credits after
				// captures (#208): if the fief's capital changed hands this
				// same turn, fief.OwnerID no longer matches the player who
				// applied the tax, so the doubling is silently dropped for
				// this turn rather than crediting the conqueror's income
				// with a tax it never played.
				parts.base *= 2
				parts.bonus *= 2
			}
			destinationID := ctx.territoryIncomeDestination(ownerID, territoryID, fief)
			perTerritory[territoryID] = TerritoryIncomeForecast{Amount: parts.total(), Destination: destinationID}
			group := incomeGroup{}
			if fief != nil {
				group.fief = fief.ID
			} else {
				group.region = regionForTerritory(ctx, territoryID)
			}
			if reports[ownerID] == nil {
				reports[ownerID] = make(map[models.TerritoryID]map[incomeGroup]*territoryIncomeReport)
			}
			if reports[ownerID][destinationID] == nil {
				reports[ownerID][destinationID] = make(map[incomeGroup]*territoryIncomeReport)
			}
			report, exists := reports[ownerID][destinationID][group]
			if !exists {
				report = &territoryIncomeReport{
					ownerID: ownerID, destinationID: destinationID, group: group, lost: destinationID == "",
				}
				if fief != nil {
					report.fiefTitle = fief.Title
					// A copy: the report must not alias the live fief.
					report.members = slices.Clone(fief.Territories)
				}
				reports[ownerID][destinationID][group] = report
			}
			report.territories++
			villageAmount := 0
			if village != nil && village.Type == models.InfraTypeVillage {
				report.villages++
				if village.Level > 0 {
					// The village's own part of parts, adjusted and taxed the same way.
					share := harvestAdjustedParts(ctx, territoryID, ctx.villageProduction(territoryID))
					amount := share.base + share.bonus
					if taxed {
						amount *= 2
					}
					villageAmount = amount
					report.villageIncome = append(report.villageIncome, IncomeVillage{
						Territory: territoryID, Level: village.Level, Amount: amount,
					})
				}
			}
			report.territoryAmount += parts.base + parts.bonus - villageAmount
			report.base += parts.base
			report.bonus += parts.bonus
			report.suppressed += parts.suppressed
		}
	}
	return reports, perTerritory
}

func creditTerritoryIncome(ctx *resolutionContext, reports map[models.PlayerID]map[models.TerritoryID]map[incomeGroup]*territoryIncomeReport) {
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, destinationID := range sortedTerritoryMap(reports[ownerID]) {
			for _, group := range sortedIncomeGroups(reports[ownerID][destinationID]) {
				report := reports[ownerID][destinationID][group]
				amount := report.base + report.bonus
				if destinationID == "" || amount == 0 {
					continue
				}
				state := ctx.state.TerritoryStates[destinationID]
				state.Resources += amount
				ctx.state.TerritoryStates[destinationID] = state
			}
		}
	}
}

func emitTerritoryIncomeEvents(ctx *resolutionContext, reports map[models.PlayerID]map[models.TerritoryID]map[incomeGroup]*territoryIncomeReport) {
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, destinationID := range sortedTerritoryMap(reports[ownerID]) {
			for _, group := range sortedIncomeGroups(reports[ownerID][destinationID]) {
				report := reports[ownerID][destinationID][group]
				ctx.events = append(ctx.events, Event{
					Type:                 EventTypeIncome,
					Phase:                6,
					OwnerID:              ownerID,
					DestinationID:        destinationID,
					FiefID:               report.group.fief,
					FiefTitle:            report.fiefTitle,
					RegionSeed:           report.group.region,
					FiefTerritories:      report.members,
					TerritoryCount:       report.territories,
					VillageCount:         report.villages,
					IncomeVillages:       report.villageIncome,
					TerritoryAmount:      report.territoryAmount,
					BaseProduction:       report.base,
					BonusProduction:      report.bonus,
					SuppressedProduction: report.suppressed,
					Production:           report.base + report.bonus,
					Lost:                 report.lost,
					StockAfter:           ctx.destinationStock(destinationID),
					Season:               ctx.state.Season,
					Year:                 ctx.state.Year(),
				})
			}
		}
	}
}

// sortedIncomeGroups returns the keys of a map[incomeGroup]V in deterministic
// order, empty id (income routed outside any fief) sorting first.
func sortedIncomeGroups[V any](values map[incomeGroup]V) []incomeGroup {
	groups := make([]incomeGroup, 0, len(values))
	for group := range values {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].fief != groups[j].fief {
			return groups[i].fief < groups[j].fief
		}
		return groups[i].region < groups[j].region
	})
	return groups
}

func (ctx *resolutionContext) destinationStock(destinationID models.TerritoryID) int {
	if destinationID == "" {
		return 0
	}
	return ctx.state.TerritoryStates[destinationID].Resources
}

// territoryIncomeParts computes the harvest-adjusted income of one controlled
// territory: territory_income, plus village_income per level of the village
// it carries (nothing for a level-0 village). Bad harvest suppresses it, good harvest doubles it, in the
// territory's own region - the same rule harvest cards apply to any other
// source's base production.
func territoryIncomeParts(ctx *resolutionContext, territoryID models.TerritoryID) sourceProductionParts {
	base := ctx.balance.TerritoryIncome + ctx.villageProduction(territoryID)
	return harvestAdjustedParts(ctx, territoryID, base)
}

// harvestAdjustedParts splits a base amount produced by territoryID into its
// harvest-adjusted parts: suppressed entirely under a bad harvest, doubled
// (as a bonus) under an abundant harvest, unaffected otherwise.
func harvestAdjustedParts(ctx *resolutionContext, territoryID models.TerritoryID, base int) sourceProductionParts {
	region := regionForTerritory(ctx, territoryID)
	switch {
	case ctx.famineRegions[region]:
		return sourceProductionParts{suppressed: base}
	case ctx.goodHarvestRegions[region]:
		return sourceProductionParts{base: base, bonus: base}
	default:
		return sourceProductionParts{base: base}
	}
}

// territoryFief returns the fief territoryID belongs to when it is owned by
// ownerID (its capital included), or nil otherwise: a territory transferred
// to a fief's conqueror this same turn already reports the conqueror as
// owner (captureFiefCapital), so this stays a simple membership
// check rather than a positional one.
func (ctx *resolutionContext) territoryFief(ownerID models.PlayerID, territoryID models.TerritoryID) *models.Fief {
	fief := ctx.fiefContaining(territoryID)
	if fief == nil || fief.OwnerID != ownerID {
		return nil
	}
	return fief
}

// territoryIncomeDestination is the credited target of one controlled
// territory's income: the fief's capital when territoryID belongs to fief
// (the caller's own territoryFief(ownerID, territoryID) lookup, passed in
// rather than redone here, even when that fief capital is itself occupied by
// another player: income is never intercepted, titres.md), else the player's
// own capital when it is valid, else the closest controlled castle, else the
// closest controlled village (BFS over crossable borders, trigram
// tie-break), else none when the income is lost.
func (ctx *resolutionContext) territoryIncomeDestination(ownerID models.PlayerID, territoryID models.TerritoryID, fief *models.Fief) models.TerritoryID {
	if fief != nil {
		return fief.CapitalTerritoryID
	}
	if capitalTerritoryID, _, hasCapital := ctx.capitalTerritory(ownerID); hasCapital {
		return capitalTerritoryID
	}
	if castleID := ctx.closestControlledTerritory(territoryID, ownerID, ctx.controllerNow, func(candidateID models.TerritoryID) bool {
		return ctx.hasInfrastructure(candidateID, models.InfraTypeCastle) && ctx.hasActiveSettlement(candidateID)
	}); castleID != "" {
		return castleID
	}
	return ctx.closestControlledTerritory(territoryID, ownerID, ctx.controllerNow, func(candidateID models.TerritoryID) bool {
		return ctx.hasInfrastructure(candidateID, models.InfraTypeVillage) && ctx.hasActiveSettlement(candidateID)
	})
}

// ForecastIncome computes the normal territory income each player would
// receive on the next action turn, including the calamity already scheduled
// for the current season (announced to every player) but never a bonus card
// still to be played. It runs the same resolveTerritoryIncome
// computation used during resolution, on a throwaway clone, so it never
// mutates state and always matches the actual resolution logic; it assumes
// nothing moves before resolution, since income now credits after captures
// (#208) but this projection can only see the current, pre-order control.
func ForecastIncome(state *models.GameState, balance assetgen.Balance) []Event {
	if state == nil {
		return nil
	}
	ctx := newForecastContext(state, balance, nil)
	resolveTerritoryIncome(ctx)
	return append([]Event(nil), ctx.events...)
}

// ForecastTerritoryIncome computes each controlled territory's normal
// projected income and destination, including the scheduled calamity like
// ForecastIncome, for the territory detail panel's "rapporte X R à
// YYY" line. It performs no mutation, and like ForecastIncome assumes
// nothing moves before resolution.
func ForecastTerritoryIncome(state *models.GameState, balance assetgen.Balance) map[models.TerritoryID]TerritoryIncomeForecast {
	return ForecastTerritoryIncomeWith(state, balance, nil)
}

// ForecastTerritoryIncomeWith is ForecastTerritoryIncome projected with the regional bonus cards the viewer
// drafted (see newForecastContext).
func ForecastTerritoryIncomeWith(state *models.GameState, balance assetgen.Balance, deckOrders map[models.PlayerID][]models.DeckOrder) map[models.TerritoryID]TerritoryIncomeForecast {
	if state == nil {
		return nil
	}
	ctx := newForecastContext(state, balance, deckOrders)
	_, perTerritory := computeTerritoryIncome(ctx)
	return perTerritory
}
