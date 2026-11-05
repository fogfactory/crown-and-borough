package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// trialOrderEntry is J NNN in the management stage: trials are filed in stage 3
// (fileWinterTrials) and judged in stage 9, so applying them here is a no-op.
type trialOrderEntry struct{}

func (trialOrderEntry) Apply(*ExecutionContext) {}

// winterTrial is a trial backed by two distinct cardinals, filed at stage 3.
// The public verdict names neither the backing players nor their orders.
type winterTrial struct {
	target models.NobleID
}

// fileWinterTrials is winter stage 3 for the J HHH NNN orders (specs/religieux.md
// § Procès à deux cardinaux): HHH is a cardinal of the player, active (neither
// excommunicated nor in a dungeon) at this stage, who backs the trial of NNN once
// per winter; the pope counts only as a cardinal. A target backed by two distinct
// cardinals, of the same player or of two, is filed for the judgment of stage 9;
// a lone cardinal, or orders on different targets, stay without effect.
func (ctx *resolutionContext) fileWinterTrials(orders map[models.PlayerID][]models.WinterOrder) {
	type backing struct {
		cardinal models.NobleID
	}
	backers := map[models.NobleID][]backing{}
	var targets []models.NobleID
	used := map[models.NobleID]bool{}
	for _, playerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, order := range orders[playerID] {
			if order.Type != models.WinterOrderTypeTrial {
				continue
			}
			cardinal := ctx.noblesByID[ctx.noblesByCode[order.NobleCode]]
			target := ctx.noblesByID[ctx.noblesByCode[order.TargetCode]]
			switch {
			case cardinal == nil || target == nil:
				ctx.rejectWinterOrder(playerID, order, "unknown_noble")
				continue
			case cardinal.OwnerID != playerID:
				ctx.rejectWinterOrder(playerID, order, "noble_not_owned")
				continue
			case !ctx.state.IsCardinal(cardinal.ID) || ctx.state.VotingReligiousTitle(cardinal.ID) == models.ReligiousTitleNone:
				ctx.rejectWinterOrder(playerID, order, "not_cardinal")
				continue
			case used[cardinal.ID]:
				ctx.rejectWinterOrder(playerID, order, "trial_limit")
				continue
			}
			used[cardinal.ID] = true
			ctx.events = append(ctx.events, Event{
				Type: EventTypeTrialFiled, Phase: winterPhase, OwnerID: playerID, OrderID: order.ID,
				NobleID: target.ID, NobleCode: models.NobleCode(target.Code),
				NobleName: ctx.state.NobleDisplayName(*target),
			})
			if len(backers[target.ID]) == 0 {
				targets = append(targets, target.ID)
			}
			backers[target.ID] = append(backers[target.ID], backing{cardinal: cardinal.ID})
		}
	}
	for _, target := range targets {
		sponsors := backers[target]
		if len(sponsors) < 2 {
			continue
		}
		ctx.winterTrials = append(ctx.winterTrials, winterTrial{target: target})
	}
}

// judgeWinterTrials is winter stage 9: the filed trials, in increasing order of
// the target's code, on the final state of the winter. The eligibility of the
// target is read here (an excommunication or an unmasking of the winter makes
// her liable, a marriage of the winter protects a lady).
func (ctx *resolutionContext) judgeWinterTrials() {
	sort.SliceStable(ctx.winterTrials, func(i, j int) bool {
		return ctx.trialCode(ctx.winterTrials[i]) < ctx.trialCode(ctx.winterTrials[j])
	})
	for _, trial := range ctx.winterTrials {
		event := Event{
			Type: EventTypeTrial, Phase: winterPhase, NobleID: trial.target,
			Season: ctx.state.Season, Year: ctx.state.Year(),
		}
		ctx.judgeTrial(event, ctx.noblesByID[trial.target])
	}
	ctx.winterTrials = nil
}

func (ctx *resolutionContext) trialCode(trial winterTrial) string {
	if noble := ctx.noblesByID[trial.target]; noble != nil {
		return noble.Code
	}
	return ""
}
