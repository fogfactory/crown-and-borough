package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// drawNobleOrder is T N (specs/succession.md § Deck de nobles): it adds the
// top card of the shared noble deck to the player's hand, at most once per
// player and winter, reshuffling the discard pile when the draw pile is
// empty. It is free. The hand limit is shared with the special-orders hand:
// the order is rejected with hand_limit_reached when both hands together
// already hold special_orders.hand_limit cards.
type drawNobleOrder struct{ order models.WinterOrder }

func (order drawNobleOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	if resolution.nobleDraws[playerID] {
		resolution.rejectWinterOrder(playerID, order.order, "noble_draw_already_used")
		return
	}
	if resolution.handSize(playerID) >= resolution.balance.SpecialOrders.HandLimit {
		resolution.rejectWinterOrder(playerID, order.order, "hand_limit_reached")
		return
	}
	cardID, drawn := resolution.drawNobleCard()
	if !drawn {
		resolution.rejectWinterOrder(playerID, order.order, "noble_deck_empty")
		return
	}
	if resolution.nobleDraws == nil {
		resolution.nobleDraws = map[models.PlayerID]bool{}
	}
	resolution.nobleDraws[playerID] = true
	deck := resolution.state.NobleDeck
	deck.Hands[playerID] = append(deck.Hands[playerID], cardID)
	// The event does not name the card: the winter report is public and a
	// hand is private.
	resolution.events = append(resolution.events, Event{
		Type:    EventTypeNobleDraw,
		Phase:   winterPhase,
		OwnerID: playerID,
		OrderID: order.order.ID,
	})
}

// dignityOrder is D N XXX CCC: it plays the dignity card CCC from the
// player's hand on their noble XXX. It is free.
type dignityOrder struct{ order models.WinterOrder }

func (order dignityOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	if noble == nil || noble.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_owned")
		return
	}
	card, handIndex, inHand := resolution.state.NobleDeck.HandCard(playerID, models.NobleCardKindDignity, winterOrder.CardCode)
	if !inHand {
		resolution.rejectWinterOrder(playerID, winterOrder, "card_not_in_hand")
		return
	}
	if noble.Has(card.Dignity) {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_already_"+string(card.Dignity))
		return
	}
	resolution.consumeNobleCard(playerID, handIndex, noble.ID)
	noble.Dignities = append(noble.Dignities, card.Dignity)
	resolution.events = append(resolution.events, Event{
		Type:      EventTypeDignity,
		Phase:     winterPhase,
		OwnerID:   playerID,
		OrderID:   winterOrder.ID,
		NobleID:   noble.ID,
		NobleCode: models.NobleCode(noble.Code),
		NobleName: resolution.state.NobleDisplayName(*noble),
		Dignity:   card.Dignity,
	})
}

// discardNobleCardOrder is D C CCC: it discards the card CCC (a noble
// trigram or a dignity code) from the player's noble hand without playing it.
// It is free, has no per-winter limit and puts the card on the discard pile
// unchanged: a noble card keeps its name and code, still out of play until
// the discard pile is reshuffled. The slot it frees can be used by a later
// T N of the same sheet.
type discardNobleCardOrder struct{ order models.WinterOrder }

func (order discardNobleCardOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	deck := resolution.state.NobleDeck
	_, handIndex, inHand := deck.HandCardByCode(playerID, order.order.CardCode)
	if !inHand {
		resolution.rejectWinterOrder(playerID, order.order, "card_not_in_hand")
		return
	}
	hand := deck.Hands[playerID]
	cardID := hand[handIndex]
	deck.Hands[playerID] = append(hand[:handIndex:handIndex], hand[handIndex+1:]...)
	deck.Discard = append(deck.Discard, cardID)
	// Like noble_draw, the event does not name the card.
	resolution.events = append(resolution.events, Event{
		Type:    EventTypeNobleDiscard,
		Phase:   winterPhase,
		OwnerID: playerID,
		OrderID: order.order.ID,
	})
}

// consumeNobleCard moves the card at handIndex of the player's hand to the
// played list, on the noble it recruited or that carries its dignity.
func (ctx *resolutionContext) consumeNobleCard(playerID models.PlayerID, handIndex int, nobleID models.NobleID) {
	deck := ctx.state.NobleDeck
	hand := deck.Hands[playerID]
	cardID := hand[handIndex]
	deck.Hands[playerID] = append(hand[:handIndex:handIndex], hand[handIndex+1:]...)
	deck.Played = append(deck.Played, models.NobleCardPlay{Card: cardID, Noble: nobleID})
}

// handSize counts the cards a player holds, the special-orders hand and the
// noble hand (noble and dignity cards) together: the hand limit applies to
// their sum.
func (ctx *resolutionContext) handSize(playerID models.PlayerID) int {
	size := 0
	if ctx.state.SpecialDeck != nil {
		size += len(ctx.state.SpecialDeck.Hands[playerID])
	}
	if ctx.state.NobleDeck != nil {
		size += len(ctx.state.NobleDeck.Hands[playerID])
	}
	return size
}
