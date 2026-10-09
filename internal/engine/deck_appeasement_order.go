package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// appeaseOrder is the revolt appeasement of a cleric (specs/religieux.md
// § Apaisement de révolte, #237). It plays no card. Two ways exist:
//
//   - the free rite (P AG HHH TER): any bishop, cardinal, pope or abbess acts
//     in the region where they stand and rolls a d6, which may kill them;
//   - the paid appeasement (P AP HHH TER): a bishop in their own bishopric, a
//     cardinal or the pope anywhere pay appeasement_cost_base^size R, safely.
//
// Either way a success removes the rebel NEUTRAL army at once, before the
// revolt cards of the same turn are applied.
type appeaseOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

// appeasementRejection returns why a cleric cannot play the order, or "" if
// the order is legal. Conditions that depend on the board during the turn
// (a rebel army to calm, the price) are checked when the order is applied.
func appeasementRejection(ctx *resolutionContext, season models.Season, playerID models.PlayerID, order models.DeckOrder) string {
	if season == models.SeasonWinter {
		return "deck_order_out_of_season"
	}
	noble := ctx.noblesByID[order.TargetNobleID]
	if noble == nil || noble.OwnerID != playerID {
		return "appeasement_requires_own_cleric"
	}
	target := regionForTerritory(ctx, order.TargetTerritoryID)
	if target == "" {
		return "appeasement_unknown_territory"
	}
	title := ctx.state.VotingReligiousTitle(noble.ID)
	if order.Type == models.DeckOrderTypeAppeasePaid {
		switch title {
		case models.ReligiousTitleCardinal, models.ReligiousTitlePope:
			return ""
		case models.ReligiousTitleBishop:
			if bishop, found := ctx.state.BishopOf(models.RegionID(target)); found && bishop == noble.ID {
				return ""
			}
			return "appeasement_requires_own_bishopric"
		}
		return "appeasement_requires_cleric"
	}
	abbess := noble.Has(models.DignityAbbess) && noble.Status != models.NobleStatusDungeon
	if title == models.ReligiousTitleNone && !abbess {
		return "appeasement_requires_cleric"
	}
	if regionForTerritory(ctx, noble.LocationID) != target {
		return "appeasement_requires_own_region"
	}
	return ""
}

func (order appeaseOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	if reason := appeasementRejection(resolution, ctx.season, order.playerID, order.order); reason != "" {
		resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	territoryID := order.order.TargetTerritoryID
	rebels := resolution.currentArmyAt(territoryID)
	if rebels == nil || rebels.OwnerID != models.NeutralPlayerID {
		resolution.rejectDeckOrderReason(order.playerID, order.order, "appeasement_no_rebel_army")
		return
	}
	noble := *resolution.noblesByID[order.order.TargetNobleID]
	event := Event{
		Type: EventTypeRevoltAppeased, Phase: phaseForSeason(resolution.state.Season),
		OwnerID: order.playerID, NobleID: noble.ID, NobleCode: models.NobleCode(noble.Code),
		NobleName: resolution.state.NobleDisplayName(noble), TerritoryID: territoryID,
		RegionSeed: regionForTerritory(resolution, territoryID), Troops: rebels.Size,
		Season: resolution.state.Season, Year: resolution.state.Year(),
	}
	if order.order.Type == models.DeckOrderTypeAppeasePaid {
		cost := appeasementCost(resolution.balance.Religion.AppeasementCostBase, rebels.Size)
		if _, paid := resolution.payFromSources(resolution.winterPaymentSources(order.playerID, territoryID), cost); !paid {
			resolution.rejectDeckOrderReason(order.playerID, order.order, "appeasement_insufficient_resources")
			return
		}
		event.Reason, event.Cost = "paid", cost
		resolution.removeRebelArmy(rebels.ID)
		resolution.events = append(resolution.events, event)
		return
	}
	religion := resolution.balance.Religion
	roll := newAppeasementRNG(resolution.state.Seed, resolution.state.Turn, noble.ID).IntN(6) + 1
	switch {
	case roll <= religion.AppeasementSuccessRolls:
		event.Reason = "succeeded"
		resolution.removeRebelArmy(rebels.ID)
		resolution.events = append(resolution.events, event)
	case roll > 6-religion.AppeasementDeathRolls:
		event.Reason = "failed_death"
		resolution.events = append(resolution.events, event)
		resolution.removeNobleInAction(noble)
	default:
		event.Reason = "failed"
		resolution.events = append(resolution.events, event)
	}
}

// appeasementCost is base^size, the price of calming a rebel army of that size.
func appeasementCost(base, size int) int {
	cost := 1
	for ; size > 0; size-- {
		cost *= base
	}
	return cost
}

// removeRebelArmy disbands a NEUTRAL army at once.
func (ctx *resolutionContext) removeRebelArmy(armyID models.ArmyID) {
	armies := ctx.state.Armies[:0]
	for _, army := range ctx.state.Armies {
		if army.ID != armyID {
			armies = append(armies, army)
		}
	}
	ctx.state.Armies = armies
	delete(ctx.startArmiesByID, armyID)
	ctx.rebuildIndexes()
}

// removeNobleInAction kills a noble during an action season, before the
// movement phases: the chain they emitted this turn is dropped like a plague
// victim's.
func (ctx *resolutionContext) removeNobleInAction(noble models.Noble) {
	chains := ctx.state.Chains[:0]
	dropped := make(map[models.ChainID]bool)
	for _, chain := range ctx.state.Chains {
		if chain.NobleID == noble.ID && noble.LastEmissionTurn == ctx.state.Turn {
			dropped[chain.ID] = true
			continue
		}
		chains = append(chains, chain)
	}
	ctx.state.Chains = chains
	for index := range ctx.state.Armies {
		if chainID := ctx.state.Armies[index].ChainID; chainID != nil && dropped[*chainID] {
			ctx.state.Armies[index].ChainID = nil
		}
	}
	ctx.removeNoble(noble, models.DeathCauseMartyr)
}

func newAppeasementRNG(seed string, turn int, nobleID models.NobleID) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|appeasement|%d|%s", seed, turn, nobleID)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}
