package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// Reasons carried by an ambush event: a noble captured from the army that
// retreated, and an ambush that caught nobody.
const (
	AmbushCaptureReason = "ambush_capture"
	AmbushVoidReason    = "ambush_void"
)

// ambushCardDefinition is the ambush card (specs/ordres-speciaux.md
// § Embuscade, #416): "P EM AAA BBB" designates the attack from AAA to BBB, the
// player's own plan or the possible attack of another player's army. If that
// attack is victorious and the defending army retreats with a noble, the noble
// first in the line of succession of its owner is captured and held by the
// attacking army. The card is consumed when it is played; the effect is
// settled with the retreats (settleAmbushes).
type ambushCardDefinition struct{}

type ambushOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (ambushCardDefinition) Kind() models.CardKind { return models.CardKindAmbush }

func (ambushCardDefinition) CanPlay(ctx *ExecutionContext, order models.DeckOrder) (bool, string) {
	if ctx.season == models.SeasonWinter {
		return false, "deck_order_out_of_season"
	}
	if !ctx.resolution.designatedAttackPossible(ctx.playerID, order.SourceTerritoryID, order.TargetTerritoryID) {
		return false, "ambush_no_attack"
	}
	return true, ""
}

func (ambushCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return ambushOrder{playerID: playerID, order: order}
}

func (order ambushOrder) Apply(ctx *ExecutionContext) {
	if applicable, reason := (ambushCardDefinition{}).CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	if !ctx.resolution.applyDeckCardOrder(order.playerID, order.order) {
		return
	}
	ctx.resolution.pendingAmbushes = append(ctx.resolution.pendingAmbushes, deckOrderIntent{playerID: order.playerID, order: order.order})
}

// applyPendingAmbushes matches each ambush card with the attack intents
// enumerated for the turn. An attack that did not materialize (a canceled or
// invalid order) leaves the card consumed with no effect.
func applyPendingAmbushes(ctx *resolutionContext) {
	for _, pending := range ctx.pendingAmbushes {
		if army := ctx.designatedAttacker(pending.order); army != nil {
			if _, taken := ctx.ambushAttacks[army.ID]; !taken {
				ctx.ambushAttacks[army.ID] = pending
				continue
			}
		}
		ctx.events = append(ctx.events, ctx.attackCardEvent(EventTypeAmbush, models.CardKindAmbush, pending, AmbushVoidReason))
	}
}

// settleAmbushes runs once the retreats are known. An ambush captures when its
// attack dislodged the defending army and that army retreated (it was not
// destroyed) with a free noble of another player; otherwise the card has no
// effect.
func (ctx *resolutionContext) settleAmbushes(plans map[models.ArmyID]*retreatPlan) {
	for _, armyID := range sortedArmyMap(ctx.ambushAttacks) {
		pending := ctx.ambushAttacks[armyID]
		event := ctx.attackCardEvent(EventTypeAmbush, models.CardKindAmbush, pending, AmbushVoidReason)
		event.ArmyID = armyID
		if plan, noble := ctx.ambushVictim(armyID, plans); noble != nil {
			if occupier := ctx.currentArmyAt(plan.dislodged.originID); occupier != nil && occupier.OwnerID == ctx.startArmiesByID[armyID].OwnerID {
				ctx.captureAmbushed(noble, occupier, plan.dislodged.originID)
				event.Reason = AmbushCaptureReason
				event.NobleID, event.NobleCode = noble.ID, models.NobleCode(noble.Code)
				event.NobleName = ctx.state.NobleDisplayName(*noble)
				event.ArmyID = occupier.ID
			}
		}
		ctx.events = append(ctx.events, event)
	}
}

// ambushVictim returns the retreat of the army dislodged by the ambushing
// attack and the noble it carried that comes first in the line of succession
// of its owner. Only free nobles of other players qualify.
func (ctx *resolutionContext) ambushVictim(attackerID models.ArmyID, plans map[models.ArmyID]*retreatPlan) (*retreatPlan, *models.Noble) {
	attacker := ctx.startArmiesByID[attackerID]
	for _, defenderID := range sortedArmyMap(plans) {
		plan := plans[defenderID]
		if plan.destroyReason != "" || plan.dislodged.attackerOriginID != attacker.TerritoryID {
			continue
		}
		var victim *models.Noble
		bestRank := 0
		for _, nobleID := range plan.dislodged.nobleIDs {
			noble := ctx.noblesByID[nobleID]
			if noble == nil || noble.OwnerID == attacker.OwnerID || noble.Status != models.NobleStatusFree {
				continue
			}
			if rank := ctx.successionRank(noble); victim == nil || rank < bestRank {
				victim, bestRank = noble, rank
			}
		}
		return plan, victim
	}
	return nil, nil
}

// successionRank is the position of the noble in the line of succession of its
// owner (0 comes first).
func (ctx *resolutionContext) successionRank(noble *models.Noble) int {
	line := ctx.state.SuccessionLine(noble.OwnerID)
	for index, member := range line {
		if member.ID == noble.ID {
			return index
		}
	}
	return len(line)
}

// captureAmbushed makes the noble a hostage (or a prisoner when its dignity
// calls for it) of the ambushing army, which holds the territory the noble's
// army was driven from.
func (ctx *resolutionContext) captureAmbushed(noble *models.Noble, occupier *models.Army, territoryID models.TerritoryID) {
	previousStatus, source := noble.Status, noble.LocationID
	noble.LocationID = territoryID
	noble.Status = models.NobleStatusHostage
	if noble.CapturedToDungeon() {
		noble.Status = models.NobleStatusDungeon
	}
	ctx.events = append(ctx.events, Event{
		Type: EventTypeCapture, Phase: 4, ArmyID: occupier.ID, TerritoryID: territoryID, SourceID: source,
		NobleID: noble.ID, PreviousStatus: previousStatus, Status: noble.Status, CaptorPlayerID: occupier.OwnerID,
	})
	ctx.unmaskEon(noble, 4)
}
