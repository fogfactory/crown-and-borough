package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// electionOrderEntry is K E / K P (candidacy) and V E / V P (vote). It only
// records the order: eligibility, the open registry and the ballots are
// checked together in the election stage (resolveWinterElections), once the
// earlier stages have changed the winter's state.
type electionOrderEntry struct{ order models.WinterOrder }

func (entry electionOrderEntry) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	resolution.electionOrders = append(resolution.electionOrders, electionOrder{
		playerID: ctx.playerID,
		order:    entry.order,
	})
}
