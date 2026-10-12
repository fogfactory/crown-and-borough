package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// trialVerdict returns the dignity a trial reveals when the noble can be put
// to death: a lady liable to the direct trial, or any excommunicated noble
// (specs/dames.md § Procès). The dignity is empty for a noble who
// holds none.
func (ctx *resolutionContext) trialVerdict(noble *models.Noble) (models.Dignity, bool) {
	_, married := ctx.state.MarriageOf(noble.ID)
	if dignity, liable := noble.TrialDignity(married); liable {
		return dignity, true
	}
	if _, excommunicated := ctx.state.ExcommunicationOf(noble.ID); excommunicated {
		if len(noble.Dignities) > 0 {
			return noble.Dignities[0], true
		}
		return "", true
	}
	return "", false
}

// judgeTrial emits the verdict of a trial on the noble (nil when she is gone)
// and, when it is well founded, executes her.
func (ctx *resolutionContext) judgeTrial(event Event, noble *models.Noble) {
	if noble == nil {
		// The target is gone (executed by another trial or dead this
		// turn): nothing identifies her, so the verdict stays generic.
		event.Reason = "trial_unfounded"
		ctx.events = append(ctx.events, event)
		return
	}
	event.NobleCode = models.NobleCode(noble.Code)
	event.NobleName = ctx.state.NobleDisplayName(*noble)
	event.TerritoryID = noble.LocationID
	dignity, liable := ctx.trialVerdict(noble)
	if !liable {
		event.Reason = "trial_unfounded"
		ctx.events = append(ctx.events, event)
		return
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

// executeNoble removes a noble put to death by a trial from play, with the
// aftermath of any death (cards released, fiefs vacated, claims settled).
func (ctx *resolutionContext) executeNoble(noble models.Noble) {
	ctx.removeNoble(noble, models.DeathCauseExecution)
}

// removeNoble removes a dead noble from play with the aftermath of any death.
func (ctx *resolutionContext) removeNoble(noble models.Noble, cause models.DeathCause) {
	ctx.state.Nobles = removeNoble(ctx.state.Nobles, noble.ID)
	ctx.releaseNobleCards(noble)
	ctx.state.RemovedNobles = append(ctx.state.RemovedNobles, models.RemovedNoble{
		ID: noble.ID, Code: noble.Code, Name: noble.Name, Sex: noble.Sex, OwnerID: noble.OwnerID,
		Cause: cause, Turn: ctx.state.Turn,
	})
	ctx.rebuildIndexes()
	ctx.vacateFiefsOfMissingHolders()
	ctx.state.DropReligiousTitlesOfMissingNobles()
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
