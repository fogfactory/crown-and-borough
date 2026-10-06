package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// marriageOrder is M N XXX YYY (specs/succession.md § Conclusion d'un
// mariage). It only records the declaration: a marriage needs the matching
// M N YYY XXX of the other player, and every condition that depends on the
// state of the winter is checked once all individual orders have run, in
// resolveMarriages.
type marriageOrder struct{ order models.WinterOrder }

type pendingMarriage struct {
	playerID models.PlayerID
	order    models.WinterOrder
}

func (order marriageOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	_, nobleExists := resolution.noblesByCode[order.order.NobleCode]
	_, spouseExists := resolution.noblesByCode[order.order.SpouseCode]
	if !nobleExists || !spouseExists {
		resolution.rejectWinterOrder(ctx.playerID, order.order, "unknown_noble")
		return
	}
	resolution.pendingMarriages = append(resolution.pendingMarriages, pendingMarriage{
		playerID: ctx.playerID,
		order:    order.order,
	})
}

// resolveMarriages concludes the declared marriages once every other winter
// order has been applied. Consent is simultaneous: a declaration marries only
// when another one names the same two nobles the other way round. Declarations
// are processed in resolution order (players by id, then sheet order), so when
// a noble appears in several reciprocal pairs only the first is concluded.
func (ctx *resolutionContext) resolveMarriages() {
	declarations := ctx.pendingMarriages
	ctx.pendingMarriages = nil
	eligible := make([]bool, len(declarations))
	for index, declaration := range declarations {
		reason := ctx.marriageRejection(declaration)
		if reason != "" {
			ctx.rejectWinterOrder(declaration.playerID, declaration.order, reason)
			continue
		}
		eligible[index] = true
	}
	done := make([]bool, len(declarations))
	refused := map[[2]models.NobleID]bool{}
	for index, declaration := range declarations {
		if !eligible[index] || done[index] {
			continue
		}
		done[index] = true
		noble := ctx.noblesByID[ctx.noblesByCode[declaration.order.NobleCode]]
		spouse := ctx.noblesByID[ctx.noblesByCode[declaration.order.SpouseCode]]
		if _, married := ctx.state.MarriageOf(noble.ID); married {
			ctx.rejectWinterOrder(declaration.playerID, declaration.order, "noble_already_married")
			continue
		}
		if _, married := ctx.state.MarriageOf(spouse.ID); married {
			ctx.rejectWinterOrder(declaration.playerID, declaration.order, "noble_already_married")
			continue
		}
		match := -1
		for other := index + 1; other < len(declarations); other++ {
			candidate := declarations[other]
			if eligible[other] && !done[other] &&
				candidate.order.NobleCode == declaration.order.SpouseCode &&
				candidate.order.SpouseCode == declaration.order.NobleCode {
				match = other
				break
			}
		}
		if match < 0 {
			ctx.rejectWinterOrder(declaration.playerID, declaration.order, "marriage_not_reciprocated")
			ctx.announceRefusedMarriage(noble, spouse, refused)
			continue
		}
		done[match] = true
		ctx.state.Marriages = append(ctx.state.Marriages, models.Marriage{
			NobleA: noble.ID, NobleB: spouse.ID, Turn: ctx.state.Turn,
		})
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeMarriage,
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

// announceRefusedMarriage tells the player who did not order the marriage
// that it was refused, and makes the failed negotiation public (a rumor in
// the report of every other player). A pair is announced once, however many
// times it was declared.
func (ctx *resolutionContext) announceRefusedMarriage(noble, spouse *models.Noble, announced map[[2]models.NobleID]bool) {
	key := [2]models.NobleID{noble.ID, spouse.ID}
	if announced[key] {
		return
	}
	announced[key] = true
	ctx.events = append(ctx.events, Event{
		Type:            EventTypeMarriageRefused,
		Phase:           winterPhase,
		OwnerID:         spouse.OwnerID,
		NobleID:         spouse.ID,
		NobleCode:       models.NobleCode(spouse.Code),
		NobleName:       ctx.state.NobleDisplayName(*spouse),
		SpouseNobleID:   noble.ID,
		SpouseNobleCode: models.NobleCode(noble.Code),
		SpouseNobleName: ctx.state.NobleDisplayName(*noble),
		SpouseOwnerID:   noble.OwnerID,
	})
}

// marriageRejection returns the reason a declaration cannot lead to a
// marriage, or "" when both nobles may marry.
func (ctx *resolutionContext) marriageRejection(declaration pendingMarriage) string {
	noble := ctx.noblesByID[ctx.noblesByCode[declaration.order.NobleCode]]
	spouse := ctx.noblesByID[ctx.noblesByCode[declaration.order.SpouseCode]]
	switch {
	case noble == nil || spouse == nil:
		return "unknown_noble"
	case noble.OwnerID != declaration.playerID:
		return "noble_not_owned"
	case spouse.OwnerID == noble.OwnerID:
		return "marriage_same_owner"
	case noble.Sex == spouse.Sex:
		return "marriage_same_sex"
	case noble.Status != models.NobleStatusFree || spouse.Status != models.NobleStatusFree:
		return "noble_not_free"
	case marriageForbidden(ctx, noble) || marriageForbidden(ctx, spouse):
		return "marriage_forbidden"
	}
	if _, married := ctx.state.MarriageOf(noble.ID); married {
		return "noble_already_married"
	}
	if _, married := ctx.state.MarriageOf(spouse.ID); married {
		return "noble_already_married"
	}
	return ""
}

// marriageForbidden reports whether a dignity forbids the noble to marry
// (specs/dames.md § Dignités, the Bloqué dignities).
func marriageForbidden(_ *resolutionContext, noble *models.Noble) bool {
	return noble.MarriageBlockedByDignity()
}
