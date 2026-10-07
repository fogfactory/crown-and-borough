package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// trialCardDefinition is the "P PR HHH" order (specs/dames.md § Carte de
// procès, #260): HHH is the code of the noble put on trial, not necessarily
// an adversary's. The card is consumed when it is played; whether the trial is
// well founded is only decided at the very end of the turn (resolveTrials),
// once every other effect of the turn has taken place.
type trialCardDefinition struct{}

type trialOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (trialCardDefinition) Kind() models.CardKind { return models.CardKindTrial }

func (trialCardDefinition) CanPlay(ctx *ExecutionContext, order models.DeckOrder) (bool, string) {
	if ctx.season == models.SeasonWinter {
		return false, "deck_order_out_of_season"
	}
	if ctx.resolution.noblesByID[order.TargetNobleID] == nil {
		return false, "trial_requires_noble"
	}
	return true, ""
}

func (trialCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return trialOrder{playerID: playerID, order: order}
}

func (order trialOrder) Apply(ctx *ExecutionContext) {
	definition := trialCardDefinition{}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	ctx.resolution.applyDeckCardOrder(order.playerID, order.order)
}

// resolveTrials judges the trial cards played this turn, in submission order.
// A trial on a noble who can be tried directly executes her and opens a revolt
// opportunity on the region where she stood, from the next action season; any
// other trial, including one on a noble already gone, is unfounded: the card
// is spent and nothing else happens.
func resolveTrials(ctx *resolutionContext) {
	for _, intent := range ctx.deckIntents {
		if intent.order.Kind != models.CardKindTrial {
			continue
		}
		noble := ctx.noblesByID[intent.order.TargetNobleID]
		event := Event{
			Type: EventTypeTrial, Phase: phaseForSeason(ctx.state.Season), OwnerID: intent.playerID,
			OrderID: intent.order.ID, CardKind: models.CardKindTrial, NobleID: intent.order.TargetNobleID,
			Season: ctx.state.Season, Year: ctx.state.Year(),
		}
		if noble == nil {
			// The target is gone (executed by another trial or dead this
			// turn): nothing identifies her, so the verdict stays generic.
			event.Reason = "trial_unfounded"
			ctx.events = append(ctx.events, event)
			continue
		}
		_, married := ctx.state.MarriageOf(noble.ID)
		event.NobleCode = models.NobleCode(noble.Code)
		event.NobleName = ctx.state.NobleDisplayName(*noble)
		event.TerritoryID = noble.LocationID
		dignity, liable := noble.TrialDignity(married)
		if !liable {
			event.Reason = "trial_unfounded"
			ctx.events = append(ctx.events, event)
			continue
		}
		region := regionForTerritory(ctx, noble.LocationID)
		event.Reason = "trial_executed"
		// The trial reveals the dignity, hidden or not.
		event.Dignity = dignity
		event.RegionSeed = region
		ctx.events = append(ctx.events, event)
		ctx.executeNoble(*noble)
		ctx.state.TrialRevoltWindows = append(ctx.state.TrialRevoltWindows, models.TrialRevoltWindow{
			RegionSeed: region, FromTurn: nextActionTurn(ctx.state.Turn),
		})
	}
}

// executeNoble removes a noble put to death by a trial from play, with the
// aftermath of any death (cards released, fiefs vacated, claims settled).
func (ctx *resolutionContext) executeNoble(noble models.Noble) {
	ctx.state.Nobles = removeNoble(ctx.state.Nobles, noble.ID)
	ctx.releaseNobleCards(noble)
	ctx.state.RemovedNobles = append(ctx.state.RemovedNobles, models.RemovedNoble{
		ID: noble.ID, Code: noble.Code, Name: noble.Name, Sex: noble.Sex, OwnerID: noble.OwnerID,
		Cause: models.DeathCauseExecution, Turn: ctx.state.Turn,
	})
	ctx.rebuildIndexes()
	ctx.vacateFiefsOfMissingHolders()
	ctx.settleClaimsOfDead()
}

func removeNoble(nobles []models.Noble, id models.NobleID) []models.Noble {
	remaining := make([]models.Noble, 0, len(nobles))
	for _, noble := range nobles {
		if noble.ID != id {
			remaining = append(remaining, noble)
		}
	}
	return remaining
}

// nextActionTurn returns the first turn after the given one that is not a
// winter.
func nextActionTurn(turn int) int {
	next := turn + 1
	if models.SeasonForTurn(next) == models.SeasonWinter {
		next++
	}
	return next
}

// revoltEligibleByTrial reports whether a trial executed last turn (or before
// the winter in between) opens revolt on the territory's region this turn.
func (ctx *resolutionContext) revoltEligibleByTrial(territoryID models.TerritoryID) bool {
	region := regionForTerritory(ctx, territoryID)
	for _, window := range ctx.state.TrialRevoltWindows {
		if window.RegionSeed == region && window.FromTurn == ctx.state.Turn {
			return true
		}
	}
	return false
}

// pruneTrialRevoltWindows drops the windows that can no longer open on the
// current turn or a later one.
func pruneTrialRevoltWindows(windows []models.TrialRevoltWindow, currentTurn int) []models.TrialRevoltWindow {
	if len(windows) == 0 {
		return nil
	}
	kept := make([]models.TrialRevoltWindow, 0, len(windows))
	for _, window := range windows {
		if window.FromTurn >= currentTurn {
			kept = append(kept, window)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}
