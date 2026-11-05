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

// CardinalPurchaseCap is the maximum number of cardinals bought with R that can
// be in play at once.
func CardinalPurchaseCap(state *models.GameState, balance assetgen.Balance) int {
	return models.CardinalCap(len(state.Players), balance.Religion.CardinalPurchaseBase, balance.Religion.CardinalPurchasePlayersPerExtra)
}

// CardinalCardCount is the number of cardinal cards in the noble deck, which
// bounds the cardinals obtained by card.
func CardinalCardCount(players int, religion assetgen.ReligionBalance) int {
	return models.CardinalCap(players, religion.CardinalCardBase, religion.CardinalCardPlayersPerExtra)
}

// PurchasedCardinals counts the cardinals in play that were not obtained by a
// card: a card cardinal carries the cardinal dignity, a purchased one does not.
func PurchasedCardinals(state *models.GameState) int {
	purchased := 0
	for _, id := range state.Cardinals {
		for _, noble := range state.Nobles {
			if noble.ID == id && !noble.Has(models.DignityCardinal) {
				purchased++
			}
		}
	}
	return purchased
}
