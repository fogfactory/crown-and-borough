package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// Reasons of an EventTypeArmyRevolt: the summed roll reached the army's size
// and turned it neutral, or fell short and left it to its owner.
const (
	ArmyRevoltNeutralizedReason = "neutralized"
	ArmyRevoltSuppressedReason  = "suppressed"
)

// starvingArmyAt returns the player army on territoryID that starved last
// turn (models.Army.Starving), the army a Révolte card played there targets,
// or nil.
func (ctx *resolutionContext) starvingArmyAt(territoryID models.TerritoryID) *models.Army {
	army := ctx.currentArmyAt(territoryID)
	if army == nil || army.OwnerID == models.NeutralPlayerID || !ctx.famished[army.ID] {
		return nil
	}
	return army
}

// resolveArmyRevolts resolves together every revolt card played on a starving
// army, before any movement (gdd.md "Révolte"): when the summed roll is at
// least the army's size, the army turns NEUTRAL with its troops, its chain is
// dropped and its owner's free nobles on the territory die. A smaller total
// leaves the army as it was.
func resolveArmyRevolts(ctx *resolutionContext) {
	for _, armyID := range sortedArmyMap(ctx.armyRevoltRolls) {
		army := ctx.armiesByID[armyID]
		total := ctx.armyRevoltRolls[armyID]
		event := Event{
			Type: EventTypeArmyRevolt, Phase: phaseForSeason(ctx.state.Season),
			CardKind: models.CardKindRevolt, RegionSeed: regionForTerritory(ctx, army.TerritoryID),
			TerritoryID: army.TerritoryID, ArmyID: army.ID, PreviousOwnerID: army.OwnerID,
			Troops: total, SizeBefore: army.Size, Reason: ArmyRevoltSuppressedReason,
			Season: ctx.state.Season, Year: ctx.state.Year(),
		}
		if total >= army.Size {
			event.Reason = ArmyRevoltNeutralizedReason
			ctx.neutralizeArmy(armyID)
		}
		ctx.events = append(ctx.events, event)
	}
	clear(ctx.armyRevoltRolls)
}

// neutralizeArmy hands a revolted army to NEUTRAL: it keeps its troops and its
// territory, loses its chain and its starving status (neutral armies follow
// their own famine rule), and its owner's free nobles on the territory die.
func (ctx *resolutionContext) neutralizeArmy(armyID models.ArmyID) {
	army := ctx.armiesByID[armyID]
	ownerID, territoryID := army.OwnerID, army.TerritoryID
	if army.ChainID != nil {
		ctx.removeChain(*army.ChainID)
	}
	army.OwnerID = models.NeutralPlayerID
	army.Starving = false
	army.StarvingStrength = 0
	ctx.startArmiesByID[armyID] = *army
	delete(ctx.famished, armyID)
	ctx.killFreeNobles(ownerID, territoryID, models.DeathCauseRevolt)
}
