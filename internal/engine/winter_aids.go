package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// WinterAids lists, for one viewer, the winter orders that can possibly succeed
// on the winter snapshot, so the client offers only possible orders and prices
// them. It reads no order and nothing private to another player.
type WinterAids struct {
	// Excommunicable are the nobles the viewer's pope could excommunicate;
	// empty unless the viewer owns an active pope.
	Excommunicable []ExcommunicationTarget `json:"excommunicable"`
	// Liftable are the nobles under a papal excommunication the viewer's pope
	// can lift.
	Liftable []models.NobleCode `json:"liftable"`
	// BuyableCardinals are the viewer's bishops that can be promoted by purchase.
	BuyableCardinals []models.NobleCode `json:"buyableCardinals"`
	CardinalCost     int                `json:"cardinalCost"`
	// Inquirers are the viewer's cardinals and pope able to order an inquiry;
	// InquiryCosts is the R an inquiry costs, per noble.
	Inquirers    []models.NobleCode       `json:"inquirers"`
	InquiryCosts map[models.NobleCode]int `json:"inquiryCosts"`
	// FiefSites are the groups of territories a fief can be founded on.
	FiefSites            []FiefSite `json:"fiefSites"`
	FiefCostPerTerritory int        `json:"fiefCostPerTerritory"`
	// Rituals are the Witches of the viewer that can order a ritual, with the
	// region (seed village) the calamity would fall on.
	Rituals []RitualOption `json:"rituals"`
}

// RitualOption is a Witch able to perform a ritual.
type RitualOption struct {
	Noble  models.NobleCode   `json:"noble"`
	Region models.TerritoryID `json:"region"`
}

// ExcommunicationTarget is a noble the pope can excommunicate. Blocker is the
// papal excommunicated noble of the same player whose lifting must come first
// (one excommunicated at a time per opposing player), empty when none.
type ExcommunicationTarget struct {
	Code    models.NobleCode `json:"code"`
	Blocker models.NobleCode `json:"blocker,omitempty"`
}

// FiefSite is a connected group of territories the viewer controls, free of
// fiefs and of foreign armies, that holds at least three territories and a
// castle. Edges are the crossable borders inside the group.
type FiefSite struct {
	Territories []models.TerritoryID    `json:"territories"`
	Castles     []models.TerritoryID    `json:"castles"`
	Edges       [][2]models.TerritoryID `json:"edges"`
}

// ForecastWinterAids returns the aids of the viewer; nil outside winter.
func ForecastWinterAids(state *models.GameState, balance assetgen.Balance, viewer models.PlayerID) *WinterAids {
	if state == nil || state.Season != models.SeasonWinter {
		return nil
	}
	ctx := newResolutionContext(cloneGameState(state), balance)
	aids := &WinterAids{
		Excommunicable:       []ExcommunicationTarget{},
		Liftable:             []models.NobleCode{},
		BuyableCardinals:     []models.NobleCode{},
		CardinalCost:         balance.Religion.CardinalCost,
		Inquirers:            []models.NobleCode{},
		InquiryCosts:         map[models.NobleCode]int{},
		FiefSites:            []FiefSite{},
		FiefCostPerTerritory: balance.Costs.FiefPerTerritory,
		Rituals:              []RitualOption{},
	}
	ctx.papalAids(viewer, aids)
	ctx.cardinalAids(viewer, aids)
	ctx.inquiryAids(viewer, aids)
	ctx.fiefAids(viewer, aids)
	ctx.ritualAids(viewer, aids)
	return aids
}

func (ctx *resolutionContext) papalAids(viewer models.PlayerID, aids *WinterAids) {
	pope := ctx.activePope(viewer)
	if pope == nil {
		return
	}
	for i := range ctx.state.Nobles {
		noble := &ctx.state.Nobles[i]
		if noble.ID == pope.ID {
			continue
		}
		excommunication, excommunicated := ctx.state.ExcommunicationOf(noble.ID)
		if excommunicated {
			if excommunication.Reason == models.ExcommunicationPapal {
				aids.Liftable = append(aids.Liftable, models.NobleCode(noble.Code))
			}
			continue
		}
		target := ExcommunicationTarget{Code: models.NobleCode(noble.Code)}
		if noble.OwnerID != viewer {
			for _, existing := range ctx.state.Excommunications {
				holder := ctx.noblesByID[existing.Noble]
				if existing.Reason == models.ExcommunicationPapal && holder != nil && holder.OwnerID == noble.OwnerID {
					target.Blocker = models.NobleCode(holder.Code)
					break
				}
			}
		}
		aids.Excommunicable = append(aids.Excommunicable, target)
	}
}

func (ctx *resolutionContext) cardinalAids(viewer models.PlayerID, aids *WinterAids) {
	if PurchasedCardinals(ctx.state) >= CardinalPurchaseCap(ctx.state, ctx.balance) {
		return
	}
	for i := range ctx.state.Nobles {
		noble := &ctx.state.Nobles[i]
		if noble.OwnerID == viewer && ctx.state.IsBishop(noble.ID) && !ctx.state.IsCardinal(noble.ID) {
			aids.BuyableCardinals = append(aids.BuyableCardinals, models.NobleCode(noble.Code))
		}
	}
}

func (ctx *resolutionContext) inquiryAids(viewer models.PlayerID, aids *WinterAids) {
	for i := range ctx.state.Nobles {
		noble := &ctx.state.Nobles[i]
		code := models.NobleCode(noble.Code)
		aids.InquiryCosts[code] = InquiryCost(ctx.state, ctx.balance.Religion.InquiryCost, noble.ID)
		if noble.OwnerID == viewer && ctx.canInquire(noble.ID) {
			aids.Inquirers = append(aids.Inquirers, code)
		}
	}
}

// fiefAids groups the territories a fief could be founded on into connected
// sites, ignoring those the conditions of T F exclude (not controlled, already
// in a fief, occupied by another army).
func (ctx *resolutionContext) fiefAids(viewer models.PlayerID, aids *WinterAids) {
	eligible := map[models.TerritoryID]bool{}
	for _, territory := range ctx.state.Territories {
		id := territory.ID
		if ctx.controlsTerritory(viewer, id) &&
			ctx.fiefContaining(id) == nil &&
			!ctx.occupiedAgainstStartController(id, ctx.currentArmyAt(id)) {
			eligible[id] = true
		}
	}
	ids := make([]models.TerritoryID, 0, len(eligible))
	for id := range eligible {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	visited := map[models.TerritoryID]bool{}
	for _, start := range ids {
		if visited[start] {
			continue
		}
		site := FiefSite{Territories: []models.TerritoryID{}, Castles: []models.TerritoryID{}, Edges: [][2]models.TerritoryID{}}
		visited[start] = true
		queue := []models.TerritoryID{start}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			site.Territories = append(site.Territories, current)
			if infrastructure := ctx.infrastructureAt(current); infrastructure != nil && infrastructure.Type == models.InfraTypeCastle {
				site.Castles = append(site.Castles, current)
			}
			for _, neighbor := range ctx.sortedNeighbors(current) {
				if !eligible[neighbor] {
					continue
				}
				if current < neighbor {
					site.Edges = append(site.Edges, [2]models.TerritoryID{current, neighbor})
				}
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		sort.Slice(site.Territories, func(i, j int) bool { return site.Territories[i] < site.Territories[j] })
		sort.Slice(site.Castles, func(i, j int) bool { return site.Castles[i] < site.Castles[j] })
		if len(site.Territories) >= models.FiefMinTerritories && len(site.Castles) > 0 {
			aids.FiefSites = append(aids.FiefSites, site)
		}
	}
}

func (ctx *resolutionContext) ritualAids(viewer models.PlayerID, aids *WinterAids) {
	for i := range ctx.state.Nobles {
		noble := &ctx.state.Nobles[i]
		if noble.OwnerID != viewer || !noble.CanPerformRitual() {
			continue
		}
		if region := regionForTerritory(ctx, noble.LocationID); region != "" {
			aids.Rituals = append(aids.Rituals, RitualOption{Noble: models.NobleCode(noble.Code), Region: region})
		}
	}
}
