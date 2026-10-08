package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TitleVotes returns the voices a player's nobles carry through their religious
// titles, wherever they stand (specs/religieux.md § Votes). Only the highest
// title of each noble counts, and only while its religious standing is active:
// a noble in a dungeon or excommunicated votes nothing.
func TitleVotes(state *models.GameState, playerID models.PlayerID, balance assetgen.Balance) int {
	votes := 0
	for _, noble := range state.Nobles {
		if noble.OwnerID == playerID {
			votes += balance.Religion.Votes.TitleVotes(state.VotingReligiousTitle(noble.ID))
		}
	}
	return votes
}

// CardinalCap is the maximum number of cardinals the game allows.
func CardinalCap(state *models.GameState, balance assetgen.Balance) int {
	return models.CardinalCap(len(state.Players), balance.Religion.CardinalCapMargin)
}
