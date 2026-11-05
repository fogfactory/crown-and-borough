package engine

import (
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

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
// player's hand on the noble XXX, whoever owns it: any dignity can be played
// on any noble, secret ones included. Only the card must come from the
// player's own hand. It is free.
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
	if noble == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	card, handIndex, inHand := resolution.state.NobleDeck.HandCard(playerID, models.NobleCardKindDignity, winterOrder.CardCode)
	if !inHand {
		resolution.rejectWinterOrder(playerID, winterOrder, "card_not_in_hand")
		return
	}
	if card.Dignity == models.DignityCardinal {
		resolution.playCardinalCard(playerID, winterOrder, noble, handIndex)
		return
	}
	// A hidden dignity is only played on one of the player's own nobles: the
	// order must not tell whether a lady of another player already hides one.
	if card.Dignity.Effect().Hidden && noble.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "dignity_hidden_own_only")
		return
	}
	_, married := resolution.state.MarriageOf(noble.ID)
	if reason := card.Dignity.CanReceive(*noble, married); reason != "" {
		resolution.rejectWinterOrder(playerID, winterOrder, reason)
		return
	}
	effect := card.Dignity.Effect()
	if effect.NeedsRegion != (winterOrder.TerritoryID != "") {
		resolution.rejectWinterOrder(playerID, winterOrder, "dignity_region_required")
		return
	}
	if effect.NeedsRegion && !resolution.isRegionSeed(winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "dignity_region_unknown")
		return
	}
	poolIndex := -1
	if effect.ChangesSexToMale {
		poolIndex = resolution.firstFreeMaleName()
		if poolIndex < 0 {
			resolution.rejectWinterOrder(playerID, winterOrder, "no_free_name")
			return
		}
	}
	resolution.consumeNobleCard(playerID, handIndex, noble.ID)
	noble.Dignities = append(noble.Dignities, card.Dignity)
	if effect.NeedsRegion {
		noble.AbbeyRegion = winterOrder.TerritoryID
	}
	if effect.ChangesSexToMale {
		// Silent: a new male noble, drawn from the unused names, replaces the
		// lady in public; her identity stays known to her owner only.
		deck := resolution.state.NobleDeck
		identity := deck.NamePool[poolIndex]
		deck.NamePool = slices.Delete(deck.NamePool, poolIndex, poolIndex+1)
		noble.SecretCode, noble.SecretName, noble.SecretSex = noble.Code, noble.Name, noble.Sex
		delete(resolution.noblesByCode, winterOrder.NobleCode)
		noble.Code, noble.Name, noble.Sex = identity.Code, identity.Name, models.SexMale
		resolution.noblesByCode[models.NobleCode(noble.Code)] = noble.ID
	}
	if card.Dignity.Effect().VoidsClaims {
		resolution.voidClaimOf(noble.ID)
	}
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

// discardHandCard moves the card at handIndex of the player's hand straight
// to the discard pile, spent without ever being played on a noble (a Claim
// lost against a married chevalier d'Éon, specs/dames.md § Chevalier d'Éon).
func (ctx *resolutionContext) discardHandCard(playerID models.PlayerID, handIndex int) {
	deck := ctx.state.NobleDeck
	hand := deck.Hands[playerID]
	cardID := hand[handIndex]
	deck.Hands[playerID] = append(hand[:handIndex:handIndex], hand[handIndex+1:]...)
	deck.Discard = append(deck.Discard, cardID)
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

// isRegionSeed reports whether the territory is the seed of a region
// (bishopric).
func (ctx *resolutionContext) isRegionSeed(territoryID models.TerritoryID) bool {
	for _, region := range ctx.state.Regions {
		if region.Seed == territoryID {
			return true
		}
	}
	return false
}

// firstFreeMaleName returns the index in the name pool of the first unused
// male name, or -1.
func (ctx *resolutionContext) firstFreeMaleName() int {
	if ctx.state.NobleDeck == nil {
		return -1
	}
	return slices.IndexFunc(ctx.state.NobleDeck.NamePool, func(name models.NobleName) bool {
		return name.Sex == models.SexMale
	})
}
