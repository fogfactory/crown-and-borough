package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func resolveSeasonEffects(ctx *resolutionContext) {
	calamities := currentSeasonCalamities(ctx)
	active := make(map[models.TerritoryID]map[models.CardKind]bool)
	for _, calamity := range calamities {
		if active[calamity.RegionSeed] == nil {
			active[calamity.RegionSeed] = make(map[models.CardKind]bool)
		}
		active[calamity.RegionSeed][calamity.Kind] = true
	}
	ctx.badWeatherRegions = make(map[models.TerritoryID]bool)
	ctx.famineRegions = make(map[models.TerritoryID]bool)
	ctx.bonusMillRegions = make(map[models.TerritoryID]int)
	ctx.bonusRationRegions = make(map[models.TerritoryID]int)
	effective := make(map[models.TerritoryID]map[models.CardKind]bool)
	intents := append([]deckOrderIntent(nil), ctx.deckIntents...)
	sort.SliceStable(intents, func(i, j int) bool {
		if intents[i].order.RegionSeed != intents[j].order.RegionSeed {
			return intents[i].order.RegionSeed < intents[j].order.RegionSeed
		}
		if intents[i].order.Kind != intents[j].order.Kind {
			return intents[i].order.Kind < intents[j].order.Kind
		}
		if intents[i].playerID != intents[j].playerID {
			return intents[i].playerID < intents[j].playerID
		}
		return intents[i].order.ID < intents[j].order.ID
	})
	for _, intent := range intents {
		seed := intent.order.RegionSeed
		if effective[seed] == nil {
			effective[seed] = make(map[models.CardKind]bool)
		}
		if effective[seed][intent.order.Kind] {
			continue
		}
		effective[seed][intent.order.Kind] = true
		canceled := false
		if canceledKind, ok := intent.order.Kind.CanceledCalamity(); ok && active[seed][canceledKind] {
			delete(active[seed], canceledKind)
			canceled = true
		}
		if !canceled {
			ctx.bonusMillRegions[seed]++
			ctx.bonusRationRegions[seed]++
		}
	}
	for _, calamity := range calamities {
		if active[calamity.RegionSeed][calamity.Kind] {
			switch calamity.Kind {
			case models.CardKindBadWeather:
				ctx.badWeatherRegions[calamity.RegionSeed] = true
			case models.CardKindFamine:
				ctx.famineRegions[calamity.RegionSeed] = true
			case models.CardKindPlague:
				applyPlague(ctx, calamity.RegionSeed)
			}
		}
	}
	for _, intent := range intents {
		if intent.order.Kind == models.CardKindRevolt && effective[intent.order.RegionSeed][intent.order.Kind] {
			applyRevolt(ctx, intent.order.RegionSeed, intent.order.ID)
		}
	}
	resolvePlagueMortality(ctx)
}

func copyTerritoryFlags(source map[models.TerritoryID]bool) map[models.TerritoryID]bool {
	copy := make(map[models.TerritoryID]bool, len(source))
	for territoryID, value := range source {
		copy[territoryID] = value
	}
	return copy
}

func currentSeasonCalamities(ctx *resolutionContext) []models.Calamity {
	augury, exists := ctx.state.Auguries[ctx.state.Year()]
	if !exists {
		return nil
	}
	result := make([]models.Calamity, 0)
	for _, calamity := range augury.Calamities {
		if calamity.Season == ctx.state.Season {
			result = append(result, calamity)
		}
	}
	return result
}

func regionForTerritory(ctx *resolutionContext, territoryID models.TerritoryID) models.TerritoryID {
	for _, region := range ctx.state.Regions {
		if containsTerritory(region.Territories, territoryID) {
			return region.Seed
		}
	}
	return ""
}

func containsTerritory(territories []models.TerritoryID, target models.TerritoryID) bool {
	for _, territoryID := range territories {
		if territoryID == target {
			return true
		}
	}
	return false
}

func applyPlague(ctx *resolutionContext, regionSeed models.TerritoryID) {
	divisor := ctx.balance.SpecialOrders.Effects.PlagueArmyDivisor
	if divisor < 1 {
		return
	}
	for _, territoryID := range regionTerritories(ctx, regionSeed) {
		army := ctx.currentArmyAt(territoryID)
		if army == nil {
			continue
		}
		army.Size = max(1, (army.Size+divisor-1)/divisor)
		ctx.startArmiesByID[army.ID] = *army
	}
}

func applyRevolt(ctx *resolutionContext, regionSeed models.TerritoryID, orderID models.OrderID) {
	count := ctx.balance.SpecialOrders.Effects.RevoltArmyCount
	minSize := ctx.balance.SpecialOrders.Effects.RevoltArmyMinSize
	maxSize := ctx.balance.SpecialOrders.Effects.RevoltArmyMaxSize
	if count < 1 || minSize < 1 || maxSize < minSize {
		return
	}
	candidates := make([]models.TerritoryID, 0)
	for _, territoryID := range regionTerritories(ctx, regionSeed) {
		if ctx.currentArmyAt(territoryID) == nil {
			candidates = append(candidates, territoryID)
		}
	}
	limit := min(count, len(candidates))
	rng := newRevoltRNG(ctx.state.Seed, ctx.state.Turn, orderID)
	for index := 0; index < limit; index++ {
		size := minSize
		if maxSize > minSize {
			size += rng.IntN(maxSize - minSize + 1)
		}
		army := models.Army{ID: ctx.allocateArmyID(), OwnerID: models.NeutralPlayerID, TerritoryID: candidates[index], Size: size}
		ctx.state.Armies = append(ctx.state.Armies, army)
		armyID := army.ID
		state := ctx.state.TerritoryStates[army.TerritoryID]
		state.Army = &armyID
		ctx.state.TerritoryStates[army.TerritoryID] = state
		ctx.startArmiesByID[army.ID] = army
		ctx.startArmyAtTerritory[army.TerritoryID] = army.ID
		ctx.rebuildIndexes()
	}
}

func regionTerritories(ctx *resolutionContext, seed models.TerritoryID) []models.TerritoryID {
	for _, region := range ctx.state.Regions {
		if region.Seed == seed {
			return append([]models.TerritoryID(nil), region.Territories...)
		}
	}
	return nil
}

func newRevoltRNG(seed string, turn int, orderID models.OrderID) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|revolt|%d|%s", seed, turn, orderID)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func resolvePlagueMortality(ctx *resolutionContext) {
	mortality := ctx.balance.SpecialOrders.Effects.PlagueNobleMortalityPercentage
	if mortality <= 0 || len(ctx.startNoblesByID) == 0 {
		return
	}
	plagueRegions := make(map[models.TerritoryID]bool)
	for _, calamity := range currentSeasonCalamities(ctx) {
		if calamity.Kind == models.CardKindPlague {
			plagueRegions[calamity.RegionSeed] = true
		}
	}
	if len(plagueRegions) == 0 {
		return
	}
	dead := make(map[models.NobleID]bool)
	for nobleID, startNoble := range ctx.startNoblesByID {
		if plagueRegions[regionForTerritory(ctx, startNoble.LocationID)] && newPlagueRNG(ctx.state.Seed, ctx.state.Turn, nobleID).IntN(100) < mortality {
			dead[nobleID] = true
			ctx.plagueDeaths = append(ctx.plagueDeaths, startNoble)
		}
	}
	if len(dead) == 0 {
		return
	}
	remainingNobles := ctx.state.Nobles[:0]
	for _, noble := range ctx.state.Nobles {
		if !dead[noble.ID] {
			remainingNobles = append(remainingNobles, noble)
		}
	}
	ctx.state.Nobles = remainingNobles
	removedChains := make(map[models.ChainID]bool)
	remainingChains := ctx.state.Chains[:0]
	for _, noble := range ctx.plagueDeaths {
		ctx.state.RemovedNobleIDs = append(ctx.state.RemovedNobleIDs, noble.ID)
	}
	for _, chain := range ctx.state.Chains {
		if !dead[chain.NobleID] {
			remainingChains = append(remainingChains, chain)
			continue
		}
		startNoble := ctx.startNoblesByID[chain.NobleID]
		if startNoble.LastEmissionTurn == ctx.state.Turn {
			removedChains[chain.ID] = true
			continue
		}
		remainingChains = append(remainingChains, chain)
	}
	ctx.state.Chains = remainingChains
	for index := range ctx.state.Armies {
		if ctx.state.Armies[index].ChainID != nil && removedChains[*ctx.state.Armies[index].ChainID] {
			ctx.state.Armies[index].ChainID = nil
		}
	}
	ctx.rebuildIndexes()
}

func newPlagueRNG(seed string, turn int, nobleID models.NobleID) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|plague-noble|%d|%s", seed, turn, nobleID)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}
