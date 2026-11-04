package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ArmyFamineRisk is one player army ForecastFamineRisk determines would
// starve this turn if nothing changes. Deficit is the ration shortfall that
// goes unmet (demand minus what its assigned source or local production
// covers), not a troop count: a famine costs an army that was already
// starving every troop beyond what its rations feed (see resolveFamine).
type ArmyFamineRisk struct {
	ArmyID      models.ArmyID      `json:"armyId"`
	TerritoryID models.TerritoryID `json:"territoryId"`
	Size        int                `json:"size"`
	Deficit     int                `json:"deficit"`
	// SuppliedStrength is the size the rations the army reaches can feed.
	SuppliedStrength int `json:"suppliedStrength"`
	// ExpectedLoss is the troops the army sheds this turn when it was already
	// starving (all of them when SuppliedStrength is zero); a first deficit
	// only caps its strength at SuppliedStrength − 1.
	ExpectedLoss int `json:"expectedLoss"`
	// PillageInfrastructure is the infrastructure an already starving army
	// pillages on its territory; SavedByPillage is set when that pillage
	// covers its demand, which spares every troop.
	PillageInfrastructure models.InfraType `json:"pillageInfrastructure,omitempty"`
	SavedByPillage        bool             `json:"savedByPillage,omitempty"`
}

// FamineRiskForecast is one player's projected ravitaillement for the next
// action turn: NetConsumption is the summed ration demand actually drawn
// from stock or the supply network (an army fully fed by local production
// contributes 0, and an army ArmiesAtRisk flags contributes 0 too, since
// nothing is actually fed to it), and ArmiesAtRisk lists the ones that would
// starve.
type FamineRiskForecast struct {
	NetConsumption int
	ArmiesAtRisk   []ArmyFamineRisk
}

// ForecastFamineRisk computes, for every player, the ravitaillement outcome
// their armies would have on the next action turn if nothing changes before
// then: the net ration demand drawn from stock or the supply network, and
// which armies would starve. It ignores any calamity or bonus card already
// drawn this turn, like ForecastIncome/ForecastTerritoryIncome (the command
// post projection must never leak an undrawn harvest card's effect), and
// necessarily assumes no order changes anything else before resolution — most
// of all that every army stays exactly where it is now, since ravitaillement
// resolves on post-movement, post-combat positions (#208) that this snapshot,
// taken before any order runs, cannot foresee (no attack, join, dispersal,
// transfer, or newly built infrastructure) — it is a snapshot of "if nothing
// moves and orders stay exactly as currently drafted", not a guarantee.
// Ravitaillement never happens in winter, so a winter state always forecasts
// nil.
//
// Unlike a simplified per-army estimate, this replays the same allocation
// resolveSupply performs — assignSupply and allocateSupply — on a disposable
// clone, so the armies it flags are exactly the ones that would starve,
// including when two armies share a single supply source that cannot feed
// both of them.
func ForecastFamineRisk(state *models.GameState, balance assetgen.Balance) map[models.PlayerID]FamineRiskForecast {
	if state == nil || state.Season == models.SeasonWinter {
		return nil
	}
	clone := cloneGameState(state)
	ctx := newResolutionContext(clone, balance)
	resolveTerritoryIncome(ctx)
	receivedRations := resolveRations(ctx)

	forecasts := make(map[models.PlayerID]FamineRiskForecast, len(ctx.state.Players))
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		sources := controlledSupplySources(ctx, ownerID)
		assignments, direct := assignSupply(ctx, ownerID, sources, receivedRations)
		assignedFamine := allocateSupply(ctx, sources, assignments)

		received := make(map[models.ArmyID]int, len(assignedFamine))
		famined := make(map[models.ArmyID]bool, len(assignedFamine))
		for _, candidate := range assignedFamine {
			famined[candidate.army.ID] = true
			received[candidate.army.ID] = candidate.received
		}

		forecast := FamineRiskForecast{}
		for _, assignment := range assignments {
			if famined[assignment.army.ID] {
				forecast.NetConsumption += received[assignment.army.ID]
				continue
			}
			forecast.NetConsumption += assignment.demand
		}
		for _, candidate := range direct {
			forecast.ArmiesAtRisk = append(forecast.ArmiesAtRisk, ctx.armyFamineRisk(candidate))
		}
		for _, candidate := range assignedFamine {
			forecast.ArmiesAtRisk = append(forecast.ArmiesAtRisk, ctx.armyFamineRisk(candidate))
		}
		sortArmyFamineRisks(forecast.ArmiesAtRisk)
		forecasts[ownerID] = forecast
	}
	return forecasts
}

func sortArmyFamineRisks(risks []ArmyFamineRisk) {
	sort.SliceStable(risks, func(i, j int) bool { return risks[i].TerritoryID < risks[j].TerritoryID })
}

func (ctx *resolutionContext) armyFamineRisk(candidate famineCandidate) ArmyFamineRisk {
	supplied := suppliedStrength(candidate.army.Size, candidate.missing, ctx.balance.CostBase)
	risk := ArmyFamineRisk{
		ArmyID:           candidate.army.ID,
		TerritoryID:      candidate.army.TerritoryID,
		Size:             candidate.army.Size,
		Deficit:          candidate.missing,
		SuppliedStrength: supplied,
	}
	if !ctx.famished[candidate.army.ID] {
		return risk
	}
	// A second deficit pillages the territory's infrastructure first, and a
	// pillage that covers the demand saves the army (resolveFamine).
	if infrastructure := ctx.infrastructureAt(candidate.army.TerritoryID); infrastructure != nil {
		risk.PillageInfrastructure = infrastructure.Type
		risk.SavedByPillage = ctx.balance.PillageBonus >= candidate.demand
	}
	if !risk.SavedByPillage {
		risk.ExpectedLoss = max(0, candidate.army.Size-supplied)
	}
	return risk
}
