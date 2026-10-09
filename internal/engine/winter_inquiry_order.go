package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// inquiryOrderEntry is Q HHH NNN in the management stage: inquiries resolve in
// stage 3 (resolveInquiries), so applying them here is a no-op.
type inquiryOrderEntry struct{}

func (inquiryOrderEntry) Apply(*ExecutionContext) {}

// resolveInquiries is winter stage 3 (specs/religieux.md § Enquête): the
// Q HHH NNN orders, players by identifier then sheet order. HHH is a cardinal
// or the pope of the player with an active title, who investigates once per
// winter. The cost is taken here, even when the target hides nothing.
func (ctx *resolutionContext) resolveInquiries(orders map[models.PlayerID][]models.WinterOrder) {
	for _, playerID := range sortedPlayerIDs(ctx.state.Players) {
		inquired := map[models.NobleID]bool{}
		for _, order := range orders[playerID] {
			if order.Type != models.WinterOrderTypeInquiry {
				continue
			}
			inquirer := ctx.noblesByID[ctx.noblesByCode[order.NobleCode]]
			target := ctx.noblesByID[ctx.noblesByCode[order.TargetCode]]
			switch {
			case inquirer == nil || target == nil:
				ctx.rejectWinterOrder(playerID, order, "unknown_noble")
				continue
			case inquirer.OwnerID != playerID:
				ctx.rejectWinterOrder(playerID, order, "noble_not_owned")
				continue
			case !ctx.canInquire(inquirer.ID):
				ctx.rejectWinterOrder(playerID, order, "not_cardinal")
				continue
			case inquired[inquirer.ID]:
				ctx.rejectWinterOrder(playerID, order, "inquiry_limit")
				continue
			}
			payFrom := inquirer.LocationID
			if capitalID, _, hasCapital := ctx.capitalTerritory(playerID); hasCapital {
				payFrom = capitalID
			}
			cost := InquiryCost(ctx.state, ctx.balance.Religion.InquiryCost, target.ID)
			spent, paid := ctx.payWinterCost(playerID, payFrom, cost)
			if !paid {
				ctx.rejectWinterOrder(playerID, order, "insufficient_resources")
				continue
			}
			inquired[inquirer.ID] = true
			ctx.events = append(ctx.events, Event{
				Type:          EventTypeInquiry,
				Phase:         winterPhase,
				OwnerID:       playerID,
				OrderID:       order.ID,
				NobleID:       target.ID,
				NobleCode:     models.NobleCode(target.Code),
				NobleName:     ctx.state.NobleDisplayName(*target),
				ResourceSpent: spent,
			})
			ctx.revealHiddenDignity(target, spent)
		}
	}
}

// canInquire reports whether the noble is a cardinal or the pope with an
// active title (not excommunicated nor in a dungeon).
func (ctx *resolutionContext) canInquire(id models.NobleID) bool {
	title := ctx.state.VotingReligiousTitle(id)
	return title == models.ReligiousTitleCardinal || title == models.ReligiousTitlePope
}

// InquiryCost is the R an inquiry on the noble costs: the most expensive of
// its own highest title (fief, bishopric, cardinalate, papacy), the cost of its
// spouse minus one, and the cost of an untitled noble. The spouse's cost is
// read on the spouse's own titles, without the marriage reduction.
func InquiryCost(state *models.GameState, costs assetgen.InquiryCost, id models.NobleID) int {
	cost := max(titleInquiryCost(state, costs, id), costs.Untitled)
	if marriage, married := state.MarriageOf(id); married {
		spouse := marriage.NobleA
		if spouse == id {
			spouse = marriage.NobleB
		}
		cost = max(cost, titleInquiryCost(state, costs, spouse)-1)
	}
	return cost
}

// titleInquiryCost is the cost of the most expensive title the noble holds, 0
// when it holds none.
func titleInquiryCost(state *models.GameState, costs assetgen.InquiryCost, id models.NobleID) int {
	cost := 0
	if fief := state.HighestHeldFief(id); fief != nil {
		switch fief.Title {
		case models.FiefTitleBarony:
			cost = costs.Baron
		case models.FiefTitleCounty:
			cost = costs.Count
		case models.FiefTitleMarquisate:
			cost = costs.Marquis
		case models.FiefTitleDuchy:
			cost = costs.Duke
		}
	}
	switch state.ReligiousTitleOf(id) {
	case models.ReligiousTitlePope:
		cost = max(cost, costs.Pope)
	case models.ReligiousTitleCardinal:
		cost = max(cost, costs.Cardinal)
	case models.ReligiousTitleBishop:
		cost = max(cost, costs.Bishop)
	}
	return cost
}

// revealHiddenDignity reveals the hidden dignity of an investigated noble to
// every player. A chevalier d'Éon or a Sorcière is excommunicated ex officio;
// the other hidden dignities keep their bonuses. A noble hiding nothing, or
// already revealed, leaves the cost spent without any effect or information.
func (ctx *resolutionContext) revealHiddenDignity(noble *models.Noble, spent int) {
	if noble.DignityRevealed {
		return
	}
	for _, dignity := range noble.Dignities {
		if !dignity.Effect().Hidden {
			continue
		}
		noble.DignityRevealed = true
		switch dignity {
		case models.DignityChevalierDEon:
			ctx.unmaskEon(noble, winterPhase)
		case models.DignityWitch:
			ctx.excommunicateExOfficio(noble.ID)
		}
		if dignity == models.DignityChevalierDEon {
			// unmaskEon emitted the public unmasking event.
			continue
		}
		ctx.events = append(ctx.events, Event{
			Type:      EventTypeDignityRevealed,
			Phase:     winterPhase,
			OwnerID:   noble.OwnerID,
			NobleID:   noble.ID,
			NobleCode: models.NobleCode(noble.Code),
			NobleName: ctx.state.NobleDisplayName(*noble),
			Dignity:   dignity,
			Cost:      spent,
		})
	}
}
