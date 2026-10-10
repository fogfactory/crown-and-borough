package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// dissolutionOrderEntry is X D in the management stage: dissolutions resolve in
// stage 4 (resolveDissolutions), so applying them here is a no-op.
type dissolutionOrderEntry struct{}

func (dissolutionOrderEntry) Apply(*ExecutionContext) {}

// dissolutionDeclaration is one X D order naming the marriage it targets.
type dissolutionDeclaration struct {
	playerID models.PlayerID
	order    models.WinterOrder
	marriage int
	papal    bool
}

// resolveDissolutions is winter stage 4 (specs/religieux.md § Dissolution de
// mariage): the pope's X D NNN, with the request of the owner of one spouse
// (a plain X D NNN on his own noble). A single order suffices when the pope's
// owner owns one of the spouses. Spouses are freed, no Claim is withdrawn.
func (ctx *resolutionContext) resolveDissolutions(orders map[models.PlayerID][]models.WinterOrder) {
	var declarations []dissolutionDeclaration
	for _, playerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, order := range orders[playerID] {
			if order.Type != models.WinterOrderTypeDissolveMarriage {
				continue
			}
			target := ctx.noblesByID[ctx.noblesByCode[order.NobleCode]]
			if target == nil {
				ctx.rejectWinterOrder(playerID, order, "unknown_noble")
				continue
			}
			index := ctx.activeMarriageIndex(target.ID)
			if index < 0 {
				ctx.rejectWinterOrder(playerID, order, "not_married")
				continue
			}
			papal := ctx.activePope(playerID) != nil
			if !papal && target.OwnerID != playerID {
				ctx.rejectWinterOrder(playerID, order, "noble_not_owned")
				continue
			}
			declarations = append(declarations, dissolutionDeclaration{playerID, order, index, papal})
		}
	}
	dissolved := map[int]bool{}
	for _, declaration := range declarations {
		if dissolved[declaration.marriage] {
			continue
		}
		marriage := ctx.state.Marriages[declaration.marriage]
		papalOrder, requested := false, false
		for _, other := range declarations {
			if other.marriage != declaration.marriage {
				continue
			}
			papalOrder = papalOrder || other.papal
			for _, spouse := range []models.NobleID{marriage.NobleA, marriage.NobleB} {
				if ctx.noblesByID[spouse].OwnerID == other.playerID {
					requested = true
				}
			}
		}
		if !papalOrder || !requested {
			reason := "dissolution_not_requested"
			if !papalOrder {
				reason = "dissolution_not_granted"
			}
			ctx.rejectWinterOrder(declaration.playerID, declaration.order, reason)
			continue
		}
		dissolved[declaration.marriage] = true
		ctx.state.Marriages[declaration.marriage].Dissolved = true
		ctx.state.Marriages[declaration.marriage].DissolvedTurn = ctx.state.Turn
		noble, spouse := ctx.noblesByID[marriage.NobleA], ctx.noblesByID[marriage.NobleB]
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeMarriageDissolved,
			Phase:           winterPhase,
			OwnerID:         noble.OwnerID,
			OrderID:         declaration.order.ID,
			NobleID:         noble.ID,
			NobleCode:       models.NobleCode(noble.Code),
			NobleName:       ctx.state.NobleDisplayName(*noble),
			SpouseNobleID:   spouse.ID,
			SpouseNobleCode: models.NobleCode(spouse.Code),
			SpouseNobleName: ctx.state.NobleDisplayName(*spouse),
			SpouseOwnerID:   spouse.OwnerID,
		})
	}
}

// activeMarriageIndex is the index in state.Marriages of the active marriage
// of the noble, -1 when it has none.
func (ctx *resolutionContext) activeMarriageIndex(id models.NobleID) int {
	for index, marriage := range ctx.state.Marriages {
		if (marriage.NobleA == id || marriage.NobleB == id) && marriage.Active(ctx.state) {
			return index
		}
	}
	return -1
}
