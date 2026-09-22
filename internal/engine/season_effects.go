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
	bonusEffects := make(map[models.TerritoryID]map[models.CardKind]bool)
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
	played := make(map[models.TerritoryID]map[models.CardKind]int)
	for _, intent := range intents {
		if intent.order.Kind == models.CardKindRevolt {
			continue
		}
		seed := intent.order.RegionSeed
		if played[seed] == nil {
			played[seed] = make(map[models.CardKind]int)
		}
		played[seed][intent.order.Kind]++
	}
	for _, intent := range intents {
		if intent.order.Kind == models.CardKindRevolt {
			continue
		}
		seed := intent.order.RegionSeed
		kind := intent.order.Kind
		count := played[seed][kind]
		if count == 0 {
			continue
		}
		played[seed][kind] = 0
		// One regional bonus applies per (region, kind) at most, after the
		// first card has consumed its matching calamity if one is active.
		canceledKind, cancels := kind.CanceledCalamity()
		if cancels && active[seed][canceledKind] {
			delete(active[seed], canceledKind)
			ctx.events = append(ctx.events, Event{Type: EventTypeCalamityCanceled, Phase: phaseForSeason(ctx.state.Season), CardKind: canceledKind, RegionSeed: seed, Season: ctx.state.Season, Year: ctx.state.Year()})
		} else {
			cancels = false
		}
		if !cancels || count > 1 {
			ctx.bonusMillRegions[seed]++
			ctx.bonusRationRegions[seed]++
			if bonusEffects[seed] == nil {
				bonusEffects[seed] = make(map[models.CardKind]bool)
			}
			bonusEffects[seed][kind] = true
			ctx.events = append(ctx.events, Event{Type: EventTypeBonusEffect, Phase: phaseForSeason(ctx.state.Season), CardKind: kind, RegionSeed: seed, Season: ctx.state.Season, Year: ctx.state.Year()})
		}
	}
	for _, calamity := range calamities {
		if active[calamity.RegionSeed][calamity.Kind] {
			ctx.events = append(ctx.events, Event{Type: EventTypeCalamityApplied, Phase: phaseForSeason(ctx.state.Season), CardKind: calamity.Kind, RegionSeed: calamity.RegionSeed, Season: calamity.Season, Year: calamity.Year})
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
	activeEffects := make(map[models.TerritoryID]map[models.CardKind]bool)
	for regionSeed, kinds := range active {
		activeEffects[regionSeed] = make(map[models.CardKind]bool, len(kinds))
		for kind := range kinds {
			activeEffects[regionSeed][kind] = true
		}
	}
	for regionSeed, kinds := range bonusEffects {
		if activeEffects[regionSeed] == nil {
			activeEffects[regionSeed] = make(map[models.CardKind]bool)
		}
		for kind := range kinds {
			activeEffects[regionSeed][kind] = true
		}
	}
	regionSeeds := make([]models.TerritoryID, 0, len(activeEffects))
	for regionSeed := range activeEffects {
		regionSeeds = append(regionSeeds, regionSeed)
	}
	sort.Slice(regionSeeds, func(i, j int) bool { return regionSeeds[i] < regionSeeds[j] })
	ctx.state.ActiveRegionEffects = []models.ActiveRegionEffect{}
	for _, regionSeed := range regionSeeds {
		kinds := make([]models.CardKind, 0, len(activeEffects[regionSeed]))
		for kind := range activeEffects[regionSeed] {
			kinds = append(kinds, kind)
		}
		sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
		for _, kind := range kinds {
			ctx.state.ActiveRegionEffects = append(ctx.state.ActiveRegionEffects, models.ActiveRegionEffect{
				Kind: kind, RegionSeed: regionSeed, Season: ctx.state.Season, Year: ctx.state.Year(),
			})
		}
	}
	for _, intent := range intents {
		if intent.order.Kind == models.CardKindRevolt {
			applyRevolt(ctx, intent.order.TargetTerritoryID, intent.order.ID, intent.playerID)
		}
	}
	resolvePlagueMortality(ctx)
	emitFamineLosses(ctx)
}

// emitFamineLosses reports the production the bad harvest calamity suppresses:
// one regional summary followed by one detail line per disabled mill and per
// settlement losing its infrastructure rations.
func emitFamineLosses(ctx *resolutionContext) {
	seeds := make([]models.TerritoryID, 0, len(ctx.famineRegions))
	for seed := range ctx.famineRegions {
		seeds = append(seeds, seed)
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i] < seeds[j] })
	for _, seed := range seeds {
		productionLost := 0
		rationsLost := 0
		details := make([]Event, 0)
		for _, territoryID := range regionTerritories(ctx, seed) {
			infrastructure := ctx.infrastructureAt(territoryID)
			if infrastructure == nil {
				continue
			}
			switch infrastructure.Type {
			case models.InfraTypeMill:
				lost := infrastructure.Level + ctx.bonusMillRegions[seed]*ctx.balance.SpecialOrders.Effects.BonusMillProduction
				productionLost += lost
				details = append(details, Event{
					Type: EventTypeFamineLoss, Phase: phaseForSeason(ctx.state.Season),
					CardKind: models.CardKindFamine, RegionSeed: seed, TerritoryID: territoryID,
					InfrastructureType: models.InfraTypeMill, Level: infrastructure.Level,
					Production: lost, Season: ctx.state.Season, Year: ctx.state.Year(),
				})
			case models.InfraTypeCastle, models.InfraTypeVillage:
				rationsLost += ctx.balance.InfraRationsBonus
				details = append(details, Event{
					Type: EventTypeFamineLoss, Phase: phaseForSeason(ctx.state.Season),
					CardKind: models.CardKindFamine, RegionSeed: seed, TerritoryID: territoryID,
					InfrastructureType: infrastructure.Type, RationsLost: ctx.balance.InfraRationsBonus,
					Season: ctx.state.Season, Year: ctx.state.Year(),
				})
			}
		}
		if productionLost == 0 && rationsLost == 0 {
			continue
		}
		ctx.events = append(ctx.events, Event{
			Type: EventTypeFamineLoss, Phase: phaseForSeason(ctx.state.Season),
			CardKind: models.CardKindFamine, RegionSeed: seed,
			Production: productionLost, RationsLost: rationsLost,
			Season: ctx.state.Season, Year: ctx.state.Year(),
		})
		ctx.events = append(ctx.events, details...)
	}
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
		before := army.Size
		army.Size = max(1, (army.Size+divisor-1)/divisor)
		ctx.startArmiesByID[army.ID] = *army
		ctx.events = append(ctx.events, Event{Type: EventTypeCalamityApplied, Phase: phaseForSeason(ctx.state.Season), CardKind: models.CardKindPlague, RegionSeed: regionSeed, ArmyID: army.ID, OwnerID: army.OwnerID, SizeBefore: before, SizeAfter: army.Size, Season: ctx.state.Season, Year: ctx.state.Year()})
	}
}

// applyRevolt consumes one revolt card on the target territory: it rolls for
// reinforcements and adds them to the common neutral army building there, or
// defers the fight to the post-movement revolt pass while the territory is
// held by a player army. A revolt whose famine has been canceled in the
// meantime is annulled with the card.
func applyRevolt(ctx *resolutionContext, targetTerritory models.TerritoryID, orderID models.OrderID, playerID models.PlayerID) {
	if targetTerritory == "" {
		return
	}
	regionSeed := regionForTerritory(ctx, targetTerritory)
	if !ctx.famineRegions[regionSeed] {
		// The famine was countered before the revolt applied: the card returns
		// to its player's hand and the annulment is credited to them.
		ctx.events = append(ctx.events, Event{
			Type: EventTypeCardCanceled, Phase: phaseForSeason(ctx.state.Season),
			CardKind: models.CardKindRevolt, RegionSeed: regionSeed, TerritoryID: targetTerritory,
			OwnerID: playerID, Season: ctx.state.Season, Year: ctx.state.Year(),
		})
		ctx.restoreDeckCard(playerID, models.CardKindRevolt)
		return
	}
	roll := ctx.rollRevoltSize(orderID)
	army := ctx.currentArmyAt(targetTerritory)
	if army != nil && army.OwnerID != models.NeutralPlayerID {
		ctx.pendingRevoltSizes[targetTerritory] += roll
		ctx.events = append(ctx.events, Event{Type: EventTypeNeutralArmy, Phase: phaseForSeason(ctx.state.Season), CardKind: models.CardKindRevolt, RegionSeed: regionForTerritory(ctx, targetTerritory), TerritoryID: targetTerritory, Troops: roll, Season: ctx.state.Season, Year: ctx.state.Year()})
		return
	}
	if army != nil {
		army.Size += roll
		ctx.startArmiesByID[army.ID] = *army
	} else {
		placeRevoltArmy(ctx, targetTerritory, roll)
	}
	ctx.events = append(ctx.events, Event{Type: EventTypeNeutralArmy, Phase: phaseForSeason(ctx.state.Season), CardKind: models.CardKindRevolt, RegionSeed: regionForTerritory(ctx, targetTerritory), TerritoryID: targetTerritory, Troops: roll, Season: ctx.state.Season, Year: ctx.state.Year()})
}

// rollRevoltSize draws the rebel reinforcement for one revolt card between the
// balance bounds, deterministically per played order.
func (ctx *resolutionContext) rollRevoltSize(orderID models.OrderID) int {
	minSize := ctx.balance.SpecialOrders.Effects.RevoltArmyMinSize
	maxSize := ctx.balance.SpecialOrders.Effects.RevoltArmyMaxSize
	if minSize < 1 {
		minSize = 1
	}
	if maxSize < minSize {
		maxSize = minSize
	}
	size := minSize
	if maxSize > minSize {
		size += newRevoltRNG(ctx.state.Seed, ctx.state.Turn, orderID).IntN(maxSize - minSize + 1)
	}
	return size
}

func placeRevoltArmy(ctx *resolutionContext, territoryID models.TerritoryID, size int) {
	army := models.Army{ID: ctx.allocateArmyID(), OwnerID: models.NeutralPlayerID, TerritoryID: territoryID, Size: size}
	ctx.state.Armies = append(ctx.state.Armies, army)
	ctx.startArmiesByID[army.ID] = army
	ctx.rebuildIndexes()
}

// resolveRevoltCombats resolves the deferred revolts against the armies that
// still hold their target territories once every movement is settled. A
// winning rebellion dislodges the holder with the standard retreat rules; a
// losing or tied rebellion is crushed without leaving an army behind.
func resolveRevoltCombats(ctx *resolutionContext) {
	if len(ctx.pendingRevoltSizes) == 0 {
		return
	}
	for _, territoryID := range sortedTerritoryMap(ctx.pendingRevoltSizes) {
		rebelForce := ctx.pendingRevoltSizes[territoryID]
		occupant := ctx.currentArmyAt(territoryID)
		if occupant == nil {
			placeRevoltArmy(ctx, territoryID, rebelForce)
			ctx.events = append(ctx.events, Event{Type: EventTypeNeutralArmy, Phase: 4, CardKind: models.CardKindRevolt, RegionSeed: regionForTerritory(ctx, territoryID), TerritoryID: territoryID, Troops: rebelForce, Season: ctx.state.Season, Year: ctx.state.Year()})
			continue
		}
		resolveRevoltCombat(ctx, territoryID, rebelForce, occupant)
	}
	ctx.pendingRevoltSizes = make(map[models.TerritoryID]int)
}

func resolveRevoltCombat(ctx *resolutionContext, territoryID models.TerritoryID, rebelForce int, occupant *models.Army) {
	defense := occupant.Size + nobleCommandBonus(ctx, *occupant)
	if ctx.hasCastle(territoryID) {
		defense += ctx.balance.CastleDefenseBonus
	}
	rebels := CombatContender{OwnerID: models.NeutralPlayerID, Force: rebelForce}
	defenders := CombatContender{ArmyID: occupant.ID, OwnerID: occupant.OwnerID, Force: defense, NobleBonus: nobleCommandBonus(ctx, *occupant), Defender: true}
	result := contestResult{
		territoryID: territoryID,
		defenderID:  occupant.ID,
		baseDefense: defense - nobleCommandBonus(ctx, *occupant),
		defense:     defense,
		castleBonus: 0,
		contenders:  []CombatContender{rebels, defenders},
	}
	if ctx.hasCastle(territoryID) {
		result.castleBonus = ctx.balance.CastleDefenseBonus
		result.baseDefense = defense - nobleCommandBonus(ctx, *occupant)
	}
	if rebelForce > defense {
		rebel := placeRevoltArmyForCombat(ctx, territoryID, rebelForce)
		result.winnerID = rebel.ID
		result.dislodgedArmyID = occupant.ID
		result.attackerOriginID = territoryID
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeCombat,
			Phase:           4,
			TerritoryID:     territoryID,
			BaseDefense:     result.baseDefense,
			Defense:         defense,
			CastleBonus:     result.castleBonus,
			Contenders:      []CombatContender{rebels, defenders},
			WinnerArmyID:    rebel.ID,
			DislodgedArmyID: occupant.ID,
			Reason:          "attack_wins",
		})
		ctx.dislodgeRevoltOccupant(territoryID, occupant)
	} else {
		rebel := placeRevoltArmyForCombat(ctx, territoryID, rebelForce)
		if rebelForce == defense {
			result.standoff = true
		}
		ctx.events = append(ctx.events, Event{
			Type:        EventTypeCombat,
			Phase:       4,
			TerritoryID: territoryID,
			BaseDefense: result.baseDefense,
			Defense:     defense,
			CastleBonus: result.castleBonus,
			Contenders:  []CombatContender{rebels, defenders},
			Reason:      "defense_holds",
		})
		// A crushed rebellion still retreats like any defeated army instead of
		// vanishing on the spot.
		ctx.retreatRebelArmy(rebel, territoryID)
	}
}

func placeRevoltArmyForCombat(ctx *resolutionContext, territoryID models.TerritoryID, size int) *models.Army {
	army := models.Army{ID: ctx.allocateArmyID(), OwnerID: models.NeutralPlayerID, TerritoryID: territoryID, Size: size}
	ctx.state.Armies = append(ctx.state.Armies, army)
	ctx.startArmiesByID[army.ID] = army
	return &army
}

// retreatRebelArmy moves a crushed rebel army to an adjacent territory with
// the standard retreat priorities; a rebellion with nowhere to go is
// destroyed. Neutral armies carry no nobles.
func (ctx *resolutionContext) retreatRebelArmy(rebel *models.Army, territoryID models.TerritoryID) {
	displaced := &dislodgedArmy{
		army:             *rebel,
		originID:         territoryID,
		attackerOriginID: territoryID,
	}
	buckets := ctx.classifyRetreatDestinations(displaced)
	destination := models.TerritoryID("")
	kind := ""
	for _, candidateID := range buckets.controlledEmpty {
		if ctx.currentArmyAt(candidateID) == nil {
			destination = candidateID
			kind = RetreatDestinationControlledEmpty
			break
		}
	}
	if destination == "" {
		for _, candidateID := range buckets.emptyOther {
			if ctx.currentArmyAt(candidateID) == nil {
				destination = candidateID
				kind = RetreatDestinationEmpty
				break
			}
		}
	}
	if destination == "" && len(buckets.friendlyArmies) > 0 {
		hostID := buckets.friendlyArmies[0]
		host := ctx.armiesByID[hostID]
		if host != nil {
			n := rebel.Size
			troopsMerged := 1
			if n > 1 {
				troopsMerged = n - 1
			}
			host.Size += troopsMerged
			ctx.events = append(ctx.events, Event{
				Type:             EventTypeRetreat,
				Phase:            4,
				ArmyID:           rebel.ID,
				SourceID:         territoryID,
				DestinationID:    host.TerritoryID,
				AttackerOriginID: territoryID,
				DestinationKind:  RetreatDestinationFriendlyArmy,
				HostArmyID:       host.ID,
				TroopsMerged:     troopsMerged,
				Outcome:          OutcomeSuccess,
			})
			ctx.removeRevoltArmy(territoryID, rebel)
			return
		}
	}
	if destination == "" {
		ctx.removeRevoltArmy(territoryID, rebel)
		ctx.events = append(ctx.events, Event{
			Type:             EventTypeArmyDestroyed,
			Phase:            4,
			ArmyID:           rebel.ID,
			TerritoryID:      territoryID,
			AttackerOriginID: territoryID,
			Reason:           "no_retreat_destination",
		})
		ctx.rebuildIndexes()
		return
	}
	ctx.removeRevoltArmy(territoryID, rebel)
	army := *rebel
	army.TerritoryID = destination
	ctx.state.Armies = append(ctx.state.Armies, army)
	ctx.events = append(ctx.events, Event{
		Type:             EventTypeRetreat,
		Phase:            4,
		ArmyID:           army.ID,
		SourceID:         territoryID,
		DestinationID:    destination,
		AttackerOriginID: territoryID,
		DestinationKind:  kind,
		Outcome:          OutcomeSuccess,
	})
}

// dislodgeRevoltOccupant removes the defeated holder from the territory and
// applies the standard retreat priorities; a holder with no retreat is
// destroyed and its nobles are captured by the rebels.
func (ctx *resolutionContext) dislodgeRevoltOccupant(territoryID models.TerritoryID, occupant *models.Army) {
	displaced := &dislodgedArmy{
		army:             *occupant,
		originID:         territoryID,
		attackerOriginID: territoryID,
		nobleIDs:         append([]models.NobleID(nil), ctx.noblesAt(territoryID)...),
	}
	buckets := ctx.classifyRetreatDestinations(displaced)
	destination := models.TerritoryID("")
	kind := ""
	for _, candidateID := range buckets.controlledEmpty {
		if ctx.currentArmyAt(candidateID) == nil {
			destination = candidateID
			kind = RetreatDestinationControlledEmpty
			break
		}
	}
	if destination == "" {
		for _, candidateID := range buckets.emptyOther {
			if ctx.currentArmyAt(candidateID) == nil {
				destination = candidateID
				kind = RetreatDestinationEmpty
				break
			}
		}
	}
	if destination == "" && len(buckets.friendlyArmies) > 0 {
		hostID := buckets.friendlyArmies[0]
		host := ctx.armiesByID[hostID]
		if host != nil {
			n := occupant.Size
			troopsMerged := 1
			troopsLost := 0
			if n > 1 {
				troopsMerged = n - 1
				troopsLost = 1
			}
			host.Size += troopsMerged
			ctx.moveNobles(displaced.nobleIDs, host.TerritoryID, host.ID)
			ctx.events = append(ctx.events, Event{
				Type:             EventTypeRetreat,
				Phase:            4,
				ArmyID:           occupant.ID,
				SourceID:         territoryID,
				DestinationID:    host.TerritoryID,
				AttackerOriginID: territoryID,
				DestinationKind:  RetreatDestinationFriendlyArmy,
				HostArmyID:       host.ID,
				TroopsMerged:     troopsMerged,
				TroopsLost:       troopsLost,
				Outcome:          OutcomeSuccess,
			})
			ctx.removeRevoltArmy(territoryID, occupant)
			return
		}
	}
	if destination == "" {
		ctx.destroyRevoltOccupant(territoryID, occupant, displaced)
		return
	}
	ctx.removeRevoltArmy(territoryID, occupant)
	army := *occupant
	army.TerritoryID = destination
	ctx.state.Armies = append(ctx.state.Armies, army)
	ctx.moveNobles(displaced.nobleIDs, destination, army.ID)
	ctx.events = append(ctx.events, Event{
		Type:             EventTypeRetreat,
		Phase:            4,
		ArmyID:           army.ID,
		SourceID:         territoryID,
		DestinationID:    destination,
		AttackerOriginID: territoryID,
		DestinationKind:  kind,
		Outcome:          OutcomeSuccess,
	})
}

func (ctx *resolutionContext) removeRevoltArmy(territoryID models.TerritoryID, occupant *models.Army) {
	remaining := make([]models.Army, 0, len(ctx.state.Armies)-1)
	for _, army := range ctx.state.Armies {
		if army.ID != occupant.ID {
			remaining = append(remaining, army)
			continue
		}
		// Keep the displaced army out of the live list; its retreat copy was
		// appended by the caller.
	}
	ctx.state.Armies = remaining
	delete(ctx.startArmiesByID, occupant.ID)
	ctx.rebuildIndexes()
}

func (ctx *resolutionContext) destroyRevoltOccupant(territoryID models.TerritoryID, occupant *models.Army, displaced *dislodgedArmy) {
	remaining := make([]models.Army, 0, len(ctx.state.Armies)-1)
	for _, army := range ctx.state.Armies {
		if army.ID != occupant.ID {
			remaining = append(remaining, army)
		}
	}
	ctx.state.Armies = remaining
	delete(ctx.startArmiesByID, occupant.ID)
	ctx.events = append(ctx.events, Event{
		Type:             EventTypeArmyDestroyed,
		Phase:            4,
		ArmyID:           occupant.ID,
		TerritoryID:      territoryID,
		AttackerOriginID: territoryID,
		Reason:           "no_retreat_destination",
	})
	ctx.captureNoblesAfterDestruction(&retreatPlan{dislodged: displaced})
	ctx.rebuildIndexes()
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
		region := regionForTerritory(ctx, startNoble.LocationID)
		if !plagueRegions[region] {
			continue
		}
		if newPlagueRNG(ctx.state.Seed, ctx.state.Turn, nobleID).IntN(100) < mortality {
			dead[nobleID] = true
			ctx.plagueDeaths = append(ctx.plagueDeaths, startNoble)
			ctx.events = append(ctx.events, Event{Type: EventTypePlagueDeath, Phase: phaseForSeason(ctx.state.Season), RegionSeed: region, NobleID: nobleID, NobleCode: models.NobleCode(startNoble.Code), NobleName: startNoble.Name, TerritoryID: startNoble.LocationID, Season: ctx.state.Season, Year: ctx.state.Year()})
		} else {
			ctx.events = append(ctx.events, Event{Type: EventTypePlagueSurvived, Phase: phaseForSeason(ctx.state.Season), RegionSeed: region, NobleID: nobleID, NobleCode: models.NobleCode(startNoble.Code), NobleName: startNoble.Name, TerritoryID: startNoble.LocationID, Season: ctx.state.Season, Year: ctx.state.Year()})
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
