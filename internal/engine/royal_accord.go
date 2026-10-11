package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// annexAccordKey identifies an annexation accord: the capital of the annexing
// fief and the annexed territory.
type annexAccordKey struct {
	capital   models.TerritoryID
	territory models.TerritoryID
}

// royalAccords holds the royal authorizations in force this winter
// (politique.md § Pouvoirs royaux): W X lets the fief of a capital annex a
// territory of another fief, W R makes a rank change free. The royal orders
// fill it before the management orders that read it; while no king or regent
// grants any, it stays empty and every order needing an accord is rejected.
type royalAccords struct {
	annex map[annexAccordKey]bool
	rank  map[models.TerritoryID]bool
}

func (ctx *resolutionContext) hasAnnexAccord(capitalID, territoryID models.TerritoryID) bool {
	return ctx.royalAccords.annex[annexAccordKey{capital: capitalID, territory: territoryID}]
}

func (ctx *resolutionContext) hasRankAccord(capitalID models.TerritoryID) bool {
	return ctx.royalAccords.rank[capitalID]
}
