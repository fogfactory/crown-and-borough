package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// receivedChain is a chain assigned to an army by this turn's submissions.
type receivedChain struct {
	player models.PlayerID
	noble  models.NobleCode
	chain  models.Chain
}

// unplannedSupports applies to supports the own-army rule of the attack cards
// (designatedAttackPossible): the offensive support "XXX S YYY - ZZZ" that opens
// a chain received this turn, for an army YYY of the same player, needs that
// army to attack ZZZ. Supporting another player's army stays free, even an
// empty territory, since a support may be given blind (specs/gdd.md). Supports
// further down a chain depend on where the army will stand and remain
// execution-time checks. working holds the received chains.
func unplannedSupports(working *models.GameState, balance assetgen.Balance, received []receivedChain) []InputError {
	var supportErrors []InputError
	var ctx *resolutionContext
	for _, entry := range received {
		if len(entry.chain.Orders) == 0 {
			continue
		}
		order := entry.chain.Orders[0]
		if order.Type != models.OrderTypeSupport || len(order.TargetIDs) != 2 {
			continue
		}
		if ctx == nil {
			ctx = newResolutionContext(working, balance)
		}
		supported := ctx.startArmyAt(order.TargetIDs[0])
		if supported == nil || supported.OwnerID != entry.player || ctx.plannedAttack(entry.player, order.TargetIDs[0], order.TargetIDs[1]) {
			continue
		}
		supportErrors = append(supportErrors, newInputError(entry.player, entry.noble, order.Line, "support_no_attack", i18n.ErrorSupportNoAttack, string(order.TargetIDs[0]), string(order.TargetIDs[1])))
	}
	return supportErrors
}
