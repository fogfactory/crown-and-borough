package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// Sources of the royal income flows, carried by the income event's Reason and
// by IncomeReport.Source.
const (
	incomeSourceRoyalTax       = "royal_tax"
	incomeSourceRoyalRentKing  = "royal_rent_king"
	incomeSourceRoyalRentQueen = "royal_rent_queen"
)

// resolveRoyalIncome credits the royal flows of the turn, right after the
// territory income: the supplement of the king's tax on the fiefs taxed this
// turn, then the rent of the king and of the queen (politique.md § Rente).
func resolveRoyalIncome(ctx *resolutionContext) {
	creditRoyalTaxSupplements(ctx)
	creditRoyalRent(ctx)
}

// creditRoyalTaxSupplements creates, for each fief the royal tax applied to,
// the amount of its territorial income a second time (harvest included) and
// delivers it to the capital of the player who played the tax. The fief's own
// income is untouched.
func creditRoyalTaxSupplements(ctx *resolutionContext) {
	for _, fief := range ctx.state.Fiefs {
		playerID := ctx.royalTaxedFiefs[fief.ID]
		if playerID == "" {
			continue
		}
		amount := 0
		for _, territoryID := range fief.Territories {
			if !controlledBy(ctx.controllerNow, fief.OwnerID, territoryID) {
				continue
			}
			parts := territoryIncomeParts(ctx, territoryID, ctx.hasInfrastructure(territoryID, models.InfraTypeVillage))
			amount += parts.base + parts.bonus
		}
		creditRoyalFlow(ctx, playerID, amount, incomeSourceRoyalTax, fief.CapitalTerritoryID)
	}
}

type royalRent struct {
	playerID models.PlayerID
	amount   int
	source   string
}

// royalRents lists the rents of the turn: 1 R per constituted fief to the
// king's player, and half of it, rounded up, to the queen's player. A held
// king keeps his rent; a vacant throne pays none, but a dowager queen still
// gets hers.
func royalRents(state *models.GameState) []royalRent {
	rent := len(state.Fiefs)
	if rent == 0 {
		return nil
	}
	owners := make(map[models.NobleID]models.PlayerID, len(state.Nobles))
	for _, noble := range state.Nobles {
		owners[noble.ID] = noble.OwnerID
	}
	var rents []royalRent
	if state.King != nil {
		if owner, ok := owners[*state.King]; ok {
			rents = append(rents, royalRent{owner, rent, incomeSourceRoyalRentKing})
		}
	}
	if state.Queen != nil {
		if owner, ok := owners[*state.Queen]; ok {
			rents = append(rents, royalRent{owner, (rent + 1) / 2, incomeSourceRoyalRentQueen})
		}
	}
	return rents
}

// ForecastRoyalRent returns the rent each player would receive this turn.
func ForecastRoyalRent(state *models.GameState) map[models.PlayerID]int {
	totals := make(map[models.PlayerID]int)
	if state == nil {
		return totals
	}
	for _, rent := range royalRents(state) {
		totals[rent.playerID] += rent.amount
	}
	return totals
}

func creditRoyalRent(ctx *resolutionContext) {
	for _, rent := range royalRents(ctx.state) {
		creditRoyalFlow(ctx, rent.playerID, rent.amount, rent.source, "")
	}
}

// creditRoyalFlow credits amount to the capital of playerID and reports it as
// a flow of its own: it is not the income of any fief of the player. taxed is
// the capital of the fief a royal tax supplement comes from, empty for a rent.
func creditRoyalFlow(ctx *resolutionContext, playerID models.PlayerID, amount int, source string, taxed models.TerritoryID) {
	if amount <= 0 {
		return
	}
	destinationID, _, hasCapital := ctx.capitalTerritory(playerID)
	if hasCapital {
		state := ctx.state.TerritoryStates[destinationID]
		state.Resources += amount
		ctx.state.TerritoryStates[destinationID] = state
	}
	ctx.events = append(ctx.events, Event{
		Type: EventTypeIncome, Phase: 6, OwnerID: playerID, DestinationID: destinationID,
		TerritoryID: taxed, Production: amount, BaseProduction: amount, Lost: !hasCapital, Reason: source,
		StockAfter: ctx.destinationStock(destinationID), Season: ctx.state.Season, Year: ctx.state.Year(),
	})
}
