package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// territoryIncomeReport aggregates one player's territory income by
// destination for the turn report. Destinations differ per territory only
// when the player controls no valid capital: each territory then finds its
// own closest controlled castle or village (see territoryIncomeDestination),
// so several destinations can appear for the same player the same turn.
type territoryIncomeReport struct {
	ownerID       models.PlayerID
	destinationID models.TerritoryID
	territories   int
	villages      int
	base          int
	bonus         int
	suppressed    int
	lost          bool
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
// destination before any other production this turn: it is never called in
// winter (see ResolveWinter, and Resolve's guard against winter states).
// Income bypasses the supply network entirely, so it cannot be intercepted;
// the harvest calamity and bonus apply in the income-producing territory's
// own region, exactly as they do for any other source's base production.
func resolveTerritoryIncome(ctx *resolutionContext) {
	reports, _ := computeTerritoryIncome(ctx)
	creditTerritoryIncome(ctx, reports)
	emitTerritoryIncomeEvents(ctx, reports)
}

// computeTerritoryIncome computes, but does not credit, every controlled
// territory's harvest-adjusted income, both grouped by (owner, destination)
// for the turn report and individually per territory for the front's
// per-territory forecast. It is the seam both ForecastIncome and
// ForecastTerritoryIncome reuse to project income without touching state.
func computeTerritoryIncome(ctx *resolutionContext) (map[models.PlayerID]map[models.TerritoryID]*territoryIncomeReport, map[models.TerritoryID]TerritoryIncomeForecast) {
	reports := make(map[models.PlayerID]map[models.TerritoryID]*territoryIncomeReport)
	perTerritory := make(map[models.TerritoryID]TerritoryIncomeForecast)
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, territoryID := range sortedStateTerritoryIDs(ctx) {
			state := ctx.state.TerritoryStates[territoryID]
			if state.OwnerID == nil || *state.OwnerID != ownerID {
				continue
			}
			hasVillage := ctx.hasInfrastructure(territoryID, models.InfraTypeVillage)
			parts := territoryIncomeParts(ctx, territoryID, hasVillage)
			destinationID := ctx.territoryIncomeDestination(ownerID, territoryID)
			perTerritory[territoryID] = TerritoryIncomeForecast{Amount: parts.total(), Destination: destinationID}
			if reports[ownerID] == nil {
				reports[ownerID] = make(map[models.TerritoryID]*territoryIncomeReport)
			}
			report, exists := reports[ownerID][destinationID]
			if !exists {
				report = &territoryIncomeReport{ownerID: ownerID, destinationID: destinationID, lost: destinationID == ""}
				reports[ownerID][destinationID] = report
			}
			report.territories++
			if hasVillage {
				report.villages++
			}
			report.base += parts.base
			report.bonus += parts.bonus
			report.suppressed += parts.suppressed
		}
	}
	return reports, perTerritory
}

func creditTerritoryIncome(ctx *resolutionContext, reports map[models.PlayerID]map[models.TerritoryID]*territoryIncomeReport) {
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, destinationID := range sortedTerritoryMap(reports[ownerID]) {
			report := reports[ownerID][destinationID]
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

func emitTerritoryIncomeEvents(ctx *resolutionContext, reports map[models.PlayerID]map[models.TerritoryID]*territoryIncomeReport) {
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, destinationID := range sortedTerritoryMap(reports[ownerID]) {
			report := reports[ownerID][destinationID]
			ctx.events = append(ctx.events, Event{
				Type:                 EventTypeIncome,
				Phase:                0,
				OwnerID:              ownerID,
				DestinationID:        destinationID,
				TerritoryCount:       report.territories,
				VillageCount:         report.villages,
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

func (ctx *resolutionContext) destinationStock(destinationID models.TerritoryID) int {
	if destinationID == "" {
		return 0
	}
	return ctx.state.TerritoryStates[destinationID].Resources
}

// territoryIncomeParts computes the harvest-adjusted income of one controlled
// territory: territory_income, plus village_income when it carries a
// village. Bad harvest suppresses it, good harvest doubles it, in the
// territory's own region - the same rule harvest cards apply to any other
// source's base production.
func territoryIncomeParts(ctx *resolutionContext, territoryID models.TerritoryID, hasVillage bool) sourceProductionParts {
	base := ctx.balance.TerritoryIncome
	if hasVillage {
		base += ctx.balance.VillageIncome
	}
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

// territoryIncomeDestination is the credited target of one controlled
// territory's income: the player's capital when it is valid, else the
// closest controlled castle, else the closest controlled village (BFS over
// crossable borders, trigram tie-break), else none when the income is lost.
// Fief capitals are out of scope for this seam; #196 will extend it to try
// the fief's capital before the player's own.
func (ctx *resolutionContext) territoryIncomeDestination(ownerID models.PlayerID, territoryID models.TerritoryID) models.TerritoryID {
	if capitalTerritoryID, _, hasCapital := ctx.capitalTerritory(ownerID); hasCapital {
		return capitalTerritoryID
	}
	if castleID := ctx.closestControlledTerritory(territoryID, ownerID, func(candidateID models.TerritoryID) bool {
		return ctx.hasInfrastructure(candidateID, models.InfraTypeCastle)
	}); castleID != "" {
		return castleID
	}
	return ctx.closestControlledTerritory(territoryID, ownerID, func(candidateID models.TerritoryID) bool {
		return ctx.hasInfrastructure(candidateID, models.InfraTypeVillage)
	})
}

// ForecastIncome computes the normal territory income each player would
// receive on the next action turn, ignoring any calamity or bonus card
// already drawn this turn: the command post projection must never leak an
// undrawn harvest card's effect. It runs the same resolveTerritoryIncome
// computation used during resolution, on a throwaway clone, so it never
// mutates state and always matches the actual resolution logic.
func ForecastIncome(state *models.GameState, balance assetgen.Balance) []Event {
	if state == nil {
		return nil
	}
	clone := cloneGameState(state)
	ctx := newResolutionContext(clone, balance)
	resolveTerritoryIncome(ctx)
	return append([]Event(nil), ctx.events...)
}

// ForecastTerritoryIncome computes each controlled territory's normal
// projected income and destination, ignoring any calamity or bonus card
// already drawn this turn, for the territory detail panel's "rapporte X R à
// YYY" line. It performs no mutation.
func ForecastTerritoryIncome(state *models.GameState, balance assetgen.Balance) map[models.TerritoryID]TerritoryIncomeForecast {
	if state == nil {
		return nil
	}
	clone := cloneGameState(state)
	ctx := newResolutionContext(clone, balance)
	_, perTerritory := computeTerritoryIncome(ctx)
	return perTerritory
}
