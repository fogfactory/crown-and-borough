package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// The tithe (religieux.md "Dîme") is the Impôts card played by a bishop, a
// cardinal or the pope on a bishopric: "P TX HHH XXX", HHH being the issuing
// cleric and XXX the bishopric's seed village. It diverts the bishopric's mill
// production to the capital of the player who played it.

// Tithe priority tiers: the local bishop outranks the cardinals, who outrank
// the pope.
const (
	titheTierBishop = iota
	titheTierCardinal
	titheTierPope
)

// isTitheOrder reports whether a tax order is a tithe rather than a
// seigneurial tax.
func (ctx *resolutionContext) isTitheOrder(order models.DeckOrder) bool {
	return order.Kind == models.CardKindSeigneurialTax && order.Tithe
}

// titheTier is the priority tier of a cleric levying the tithe on the
// bishopric seeded at regionSeed: the bishop of that very bishopric first,
// then cardinals, then the pope.
func (ctx *resolutionContext) titheTier(nobleID models.NobleID, regionSeed models.TerritoryID) int {
	if bishopric, isBishop := ctx.state.BishopricOf(nobleID); isBishop {
		for _, region := range ctx.state.Regions {
			if region.ID == bishopric && region.Seed == regionSeed {
				return titheTierBishop
			}
		}
	}
	if ctx.state.VotingReligiousTitle(nobleID) == models.ReligiousTitlePope {
		return titheTierPope
	}
	return titheTierCardinal
}

// titheRejection returns why the order cannot be played as a tithe, or "".
func titheRejection(ctx *resolutionContext, playerID models.PlayerID, order models.DeckOrder) (bool, string) {
	issuer := ctx.noblesByID[order.TargetNobleID]
	if issuer == nil || issuer.OwnerID != playerID {
		return false, "tithe_requires_own_cleric"
	}
	if !ctx.isRegionSeed(order.TargetTerritoryID) {
		return false, "tithe_requires_bishopric"
	}
	switch ctx.state.VotingReligiousTitle(issuer.ID) {
	case models.ReligiousTitleNone:
		return false, "tithe_requires_cleric"
	case models.ReligiousTitleCardinal, models.ReligiousTitlePope:
		return true, ""
	}
	if ctx.titheTier(issuer.ID, order.TargetTerritoryID) != titheTierBishop {
		return false, "tithe_requires_own_bishopric"
	}
	return true, ""
}

type titheClaim struct {
	playerID models.PlayerID
	order    models.DeckOrder
	tier     int
}

// applyTithes settles the tithes played this turn, bishopric by bishopric: only
// the best-ranked tier collects; within a tier, the players share equally (one
// share per player). Every other tithe card is consumed without effect.
func applyTithes(ctx *resolutionContext, intents []deckOrderIntent) {
	claims := make(map[models.TerritoryID][]titheClaim)
	for _, intent := range intents {
		if !ctx.isTitheOrder(intent.order) {
			continue
		}
		if ok, reason := titheRejection(ctx, intent.playerID, intent.order); !ok {
			ctx.events = append(ctx.events, Event{
				Type: EventTypeRejected, Phase: phaseForSeason(ctx.state.Season),
				OwnerID: intent.playerID, OrderID: intent.order.ID, Reason: reason,
			})
			continue
		}
		seed := intent.order.TargetTerritoryID
		claims[seed] = append(claims[seed], titheClaim{
			playerID: intent.playerID, order: intent.order,
			tier: ctx.titheTier(intent.order.TargetNobleID, seed),
		})
	}
	seeds := make([]models.TerritoryID, 0, len(claims))
	for seed := range claims {
		seeds = append(seeds, seed)
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i] < seeds[j] })
	for _, seed := range seeds {
		best := claims[seed][0].tier
		for _, claim := range claims[seed] {
			best = min(best, claim.tier)
		}
		collectors := make(map[models.PlayerID]bool)
		for _, claim := range claims[seed] {
			reason := ""
			switch {
			case claim.tier != best:
				reason = "tithe_outranked"
			case collectors[claim.playerID]:
				reason = "tithe_already_applied"
			}
			if reason != "" {
				ctx.events = append(ctx.events, Event{
					Type: EventTypeCardCanceled, Phase: phaseForSeason(ctx.state.Season),
					CardKind: models.CardKindSeigneurialTax, TerritoryID: seed, RegionSeed: seed,
					OwnerID: claim.playerID, Reason: reason, Season: ctx.state.Season, Year: ctx.state.Year(),
				})
				continue
			}
			collectors[claim.playerID] = true
		}
		players := make([]models.PlayerID, 0, len(collectors))
		for playerID := range collectors {
			players = append(players, playerID)
		}
		sort.Slice(players, func(i, j int) bool { return players[i] < players[j] })
		ctx.tithesThisTurn[seed] = players
		ctx.state.TithedRegions = append(ctx.state.TithedRegions, models.TithedRegion{RegionSeed: seed, Turn: ctx.state.Turn})
		for _, playerID := range players {
			ctx.events = append(ctx.events, Event{
				Type: EventTypeBonusEffect, Phase: phaseForSeason(ctx.state.Season),
				CardKind: models.CardKindSeigneurialTax, TerritoryID: seed, RegionSeed: seed,
				OwnerID: playerID, Reason: "tithe", Season: ctx.state.Season, Year: ctx.state.Year(),
			})
		}
	}
}

// titheCollectors are the players of a tithed bishopric who can receive its
// mill production: those holding a capital.
func (ctx *resolutionContext) titheCollectors(regionSeed models.TerritoryID) []models.PlayerID {
	var collectors []models.PlayerID
	for _, playerID := range ctx.tithesThisTurn[regionSeed] {
		// Only a capital that is still a supply source can receive the
		// production (see controlledSupplySources); otherwise it stays on the mills.
		capital, _, ok := ctx.capitalTerritory(playerID)
		if ok && controlledBy(ctx.controllerNow, playerID, capital) && !ctx.occupiedAgainstController(capital, ctx.currentArmyAt(capital)) {
			collectors = append(collectors, playerID)
		}
	}
	return collectors
}

// titheDiversion is the part of a mill's production, in whole units, diverted
// to the tithing players of its bishopric; the remainder of the equal division
// stays on the normal flow.
func (ctx *resolutionContext) titheDiversion(regionSeed models.TerritoryID, total int) int {
	collectors := len(ctx.titheCollectors(regionSeed))
	if collectors == 0 {
		return 0
	}
	return total / collectors * collectors
}

// titheProductionAtCapital is the mill production tithing players deliver to
// the capital castle at territoryID this turn.
func (ctx *resolutionContext) titheProductionAtCapital(territoryID models.TerritoryID) int {
	received := 0
	for seed := range ctx.tithesThisTurn {
		collectors := ctx.titheCollectors(seed)
		isCollector := false
		for _, playerID := range collectors {
			if capital, _, ok := ctx.capitalTerritory(playerID); ok && capital == territoryID {
				isCollector = true
			}
		}
		if !isCollector {
			continue
		}
		for _, millID := range regionTerritoriesBySeed(ctx, seed) {
			infrastructure := ctx.infrastructureAt(millID)
			if infrastructure == nil || infrastructure.Type != models.InfraTypeMill || !ctx.millActive(millID) {
				continue
			}
			production, bonus, _ := millRawProduction(ctx, millID, infrastructure.Level)
			received += (production + bonus) / len(collectors)
		}
	}
	return received
}

func regionTerritoriesBySeed(ctx *resolutionContext, seed models.TerritoryID) []models.TerritoryID {
	for _, region := range ctx.state.Regions {
		if region.Seed == seed {
			return region.Territories
		}
	}
	return nil
}

// revoltEligibleByTithe reports whether the territory's bishopric was tithed
// this turn, last turn, or by a tithe order co-submitted while validating.
func (ctx *resolutionContext) revoltEligibleByTithe(territoryID models.TerritoryID) bool {
	seed := regionForTerritory(ctx, territoryID)
	if seed == "" {
		return false
	}
	if len(ctx.tithesThisTurn[seed]) > 0 || ctx.pendingTitheRegions[seed] {
		return true
	}
	for _, tithed := range ctx.state.TithedRegions {
		if tithed.RegionSeed == seed && (tithed.Turn == ctx.state.Turn || tithed.Turn == ctx.state.Turn-1) {
			return true
		}
	}
	return false
}

// pruneTithedRegions drops the tithe windows that can no longer cover the
// current or next turn.
func pruneTithedRegions(tithed []models.TithedRegion, currentTurn int) []models.TithedRegion {
	var kept []models.TithedRegion
	for _, entry := range tithed {
		if entry.Turn >= currentTurn-1 {
			kept = append(kept, entry)
		}
	}
	return kept
}
