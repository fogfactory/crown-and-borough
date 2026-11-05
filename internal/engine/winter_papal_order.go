package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// papalOrderEntry is X E / X L in the management stage: these orders already
// resolved in stage 1 (resolvePapalSanctions), so applying them again is a
// no-op.
type papalOrderEntry struct{}

func (papalOrderEntry) Apply(*ExecutionContext) {}

// resolvePapalSanctions is winter stage 1 (specs/hiver.md): the pope's X E
// (excommunicate) and X L (lift) orders, players by identifier then sheet
// order, before any other order. Only the pope's owner can give them while the
// pope holds an active title (not in a dungeon).
func (ctx *resolutionContext) resolvePapalSanctions(orders map[models.PlayerID][]models.WinterOrder) {
	excommunicated := false
	for _, playerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, order := range orders[playerID] {
			switch order.Type {
			case models.WinterOrderTypeExcommunicate:
				ctx.pronounceExcommunication(playerID, order, &excommunicated)
			case models.WinterOrderTypeLiftExcommunication:
				ctx.liftExcommunication(playerID, order)
			}
		}
	}
}

// activePope returns the pope when playerID owns him and his title is active.
func (ctx *resolutionContext) activePope(playerID models.PlayerID) *models.Noble {
	if ctx.state.Pope == nil {
		return nil
	}
	pope := ctx.noblesByID[*ctx.state.Pope]
	if pope == nil || pope.OwnerID != playerID || ctx.state.VotingReligiousTitle(pope.ID) != models.ReligiousTitlePope {
		return nil
	}
	return pope
}

func (ctx *resolutionContext) papalTarget(playerID models.PlayerID, order models.WinterOrder) (*models.Noble, *models.Noble, string) {
	pope := ctx.activePope(playerID)
	if pope == nil {
		return nil, nil, "not_pope"
	}
	target := ctx.noblesByID[ctx.noblesByCode[order.NobleCode]]
	if target == nil {
		return nil, nil, "unknown_noble"
	}
	return pope, target, ""
}

func (ctx *resolutionContext) pronounceExcommunication(playerID models.PlayerID, order models.WinterOrder, done *bool) {
	pope, target, reason := ctx.papalTarget(playerID, order)
	switch {
	case reason != "":
		ctx.rejectWinterOrder(playerID, order, reason)
		return
	case target.ID == pope.ID:
		ctx.rejectWinterOrder(playerID, order, "cannot_excommunicate_self")
		return
	case *done:
		ctx.rejectWinterOrder(playerID, order, "excommunication_limit")
		return
	}
	if _, already := ctx.state.ExcommunicationOf(target.ID); already {
		ctx.rejectWinterOrder(playerID, order, "already_excommunicated")
		return
	}
	if target.OwnerID != playerID {
		for _, existing := range ctx.state.Excommunications {
			if existing.Reason != models.ExcommunicationPapal {
				continue
			}
			if holder := ctx.noblesByID[existing.Noble]; holder != nil && holder.OwnerID == target.OwnerID {
				ctx.rejectWinterOrder(playerID, order, "excommunication_slot_taken")
				return
			}
		}
	}
	*done = true
	ctx.excommunicate(models.Excommunication{
		Noble:  target.ID,
		Reason: models.ExcommunicationPapal,
		By:     pope.ID,
		Turn:   ctx.state.Turn,
	})
	ctx.emitExcommunicationEvent(EventTypeExcommunication, order, target)
}

func (ctx *resolutionContext) liftExcommunication(playerID models.PlayerID, order models.WinterOrder) {
	_, target, reason := ctx.papalTarget(playerID, order)
	if reason != "" {
		ctx.rejectWinterOrder(playerID, order, reason)
		return
	}
	excommunication, excommunicated := ctx.state.ExcommunicationOf(target.ID)
	switch {
	case !excommunicated:
		ctx.rejectWinterOrder(playerID, order, "not_excommunicated")
		return
	case excommunication.Reason != models.ExcommunicationPapal:
		ctx.rejectWinterOrder(playerID, order, "excommunication_not_liftable")
		return
	}
	ctx.state.LiftExcommunication(target.ID)
	ctx.emitExcommunicationEvent(EventTypeExcommunicationLifted, order, target)
}

func (ctx *resolutionContext) emitExcommunicationEvent(eventType EventType, order models.WinterOrder, noble *models.Noble) {
	ctx.events = append(ctx.events, Event{
		Type:      eventType,
		Phase:     winterPhase,
		OwnerID:   noble.OwnerID,
		OrderID:   order.ID,
		NobleID:   noble.ID,
		NobleCode: models.NobleCode(noble.Code),
		NobleName: ctx.state.NobleDisplayName(*noble),
	})
}
