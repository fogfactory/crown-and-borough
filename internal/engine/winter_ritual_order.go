package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// ritualOrder is S R NNN CAL | S R NNN N (specs/dames.md § Sorcière): the
// Witch NNN of the player either calls a calamity kind (CAL) or fixes the
// season (N: 1 spring, 2 summer, 3 autumn) of a calamity of next year, not
// both. The calamity then falls on the region where the Witch stands. The
// order is only recorded here: it bends the draw of the calamities, which
// happens at the end of the winter.
type ritualOrder struct{ order models.WinterOrder }

// pendingRitual is the ritual ordered this winter.
type pendingRitual struct {
	playerID models.PlayerID
	order    models.WinterOrder
	noble    models.NobleID
	// region is the seed village of the region the Witch stands in.
	region models.TerritoryID
	used   bool
}

func (order ritualOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	noble := resolution.noblesByID[nobleID]
	if !exists || noble == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	if noble.OwnerID != playerID || !noble.CanPerformRitual() {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_witch")
		return
	}
	if resolution.ritual != nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "ritual_already_used")
		return
	}
	validSeason := winterOrder.Season == models.SeasonSpring || winterOrder.Season == models.SeasonSummer || winterOrder.Season == models.SeasonAutumn
	if (winterOrder.Calamity != "" && !winterOrder.Calamity.IsCalamity()) ||
		(winterOrder.Calamity != "") == (winterOrder.Season != "") ||
		(winterOrder.Season != "" && !validSeason) {
		resolution.rejectWinterOrder(playerID, winterOrder, "ritual_invalid_target")
		return
	}
	region := regionForTerritory(resolution, noble.LocationID)
	if region == "" {
		resolution.rejectWinterOrder(playerID, winterOrder, "ritual_no_region")
		return
	}
	resolution.ritual = &pendingRitual{playerID: playerID, order: winterOrder, noble: noble.ID, region: region}
	resolution.events = append(resolution.events, Event{
		Type:      EventTypeRitual,
		Phase:     winterPhase,
		OwnerID:   playerID,
		OrderID:   winterOrder.ID,
		NobleID:   noble.ID,
		NobleCode: models.NobleCode(noble.Code),
		NobleName: resolution.state.NobleDisplayName(*noble),
		Reason:    "declared",
	})
}

// ritualFor returns the ritual that bends the calamity being scheduled, if
// any. A called kind bends the first calamity of that kind; a fixed season
// bends the first calamity of the winter, provided the season is still free.
func (ctx *resolutionContext) ritualFor(kind models.CardKind, first bool, free []models.Season) *pendingRitual {
	ritual := ctx.ritual
	if ritual == nil || ritual.used {
		return nil
	}
	if ritual.order.Calamity != "" {
		if ritual.order.Calamity == kind {
			return ritual
		}
		return nil
	}
	if !first {
		return nil
	}
	for _, season := range free {
		if season == ritual.order.Season {
			return ritual
		}
	}
	return nil
}

func (ctx *resolutionContext) completeRitual(ritual *pendingRitual, kind models.CardKind, season models.Season) {
	ritual.used = true
	ctx.emitRitualOutcome(ritual, "succeeded", kind, season)
}

// concludeRitual reports a ritual that bent no calamity as failed.
func (ctx *resolutionContext) concludeRitual() {
	if ritual := ctx.ritual; ritual != nil && !ritual.used {
		ritual.used = true
		ctx.emitRitualOutcome(ritual, "failed", ritual.order.Calamity, ritual.order.Season)
	}
}

func (ctx *resolutionContext) emitRitualOutcome(ritual *pendingRitual, reason string, kind models.CardKind, season models.Season) {
	noble := ctx.noblesByID[ritual.noble]
	event := Event{
		Type:       EventTypeRitual,
		Phase:      winterPhase,
		OwnerID:    ritual.playerID,
		NobleID:    ritual.noble,
		Reason:     reason,
		CardKind:   kind,
		Season:     season,
		RegionSeed: ritual.region,
	}
	if noble != nil {
		event.NobleCode = models.NobleCode(noble.Code)
		event.NobleName = ctx.state.NobleDisplayName(*noble)
	}
	ctx.events = append(ctx.events, event)
}
