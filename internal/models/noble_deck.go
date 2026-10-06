package models

import "fmt"

// NobleCardID identifies a card of the noble deck ("K1", "K2", ...).
type NobleCardID string

// NobleCardKind is the kind of a noble deck card.
type NobleCardKind string

const (
	// NobleCardKindNoble recruits the noble it carries when played (R N).
	NobleCardKindNoble NobleCardKind = "noble"
	// NobleCardKindDignity confers a dignity on an owned noble when played
	// (D N).
	NobleCardKindDignity NobleCardKind = "dignity"
	// NobleCardKindClaim is consumed by a claim order (C N) played on one of
	// the player's nobles, its heir.
	NobleCardKindClaim NobleCardKind = "claim"
)

// ClaimCardCode is the code of the claim cards, as typed in D C CLM.
const ClaimCardCode = "CLM"

// IsValid reports whether the kind is a known value.
func (k NobleCardKind) IsValid() bool {
	return k == NobleCardKindNoble || k == NobleCardKindDignity || k == NobleCardKindClaim
}

// NobleCard is one card of the noble deck (specs/succession.md § Deck de
// nobles). A claim card carries no identity: it is played on an heir. A noble card carries the identity of the noble it recruits (Code
// is its trigram); a dignity card carries the Dignity it confers and its
// card code (several dignity cards of the same dignity share it).
type NobleCard struct {
	ID      NobleCardID   `json:"id"`
	Kind    NobleCardKind `json:"kind"`
	Code    string        `json:"code"`
	Name    string        `json:"name,omitempty"`
	Sex     Sex           `json:"sex,omitempty"`
	Dignity Dignity       `json:"dignity,omitempty"`
}

// NobleName is an unused name of assets/prenoms.csv: it can still be given
// to a replacement noble card.
type NobleName struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Sex  Sex    `json:"sex"`
}

// NobleCardPlay records a played card that is still in effect: the card lies
// on the living noble it recruited (noble card) or that carries the dignity
// it conferred (dignity card).
type NobleCardPlay struct {
	Card  NobleCardID `json:"card"`
	Noble NobleID     `json:"noble"`
}

// NobleDeck is the single deck of nobles shared by every player. A card is
// in the draw pile, in the discard pile, in one player's hand, or played on
// a living noble. When the draw pile runs out, the discard pile is shuffled
// into it (Reshuffles counts these shuffles, which seed them). A noble card
// leaves the deck for good when its noble dies, and a new card of the same
// sex, taken from NamePool, goes to the discard pile; a dignity card goes to
// the discard pile when its carrier dies or loses the dignity. A player can
// also discard any card of their hand (D C), which then goes to the discard
// pile unchanged.
type NobleDeck struct {
	Cards      []NobleCard                `json:"cards"`
	DrawPile   []NobleCardID              `json:"drawPile"`
	Discard    []NobleCardID              `json:"discard"`
	Hands      map[PlayerID][]NobleCardID `json:"hands"`
	Played     []NobleCardPlay            `json:"played"`
	Reshuffles int                        `json:"reshuffles"`
	// NamePool holds the names no card, noble or dead noble uses yet, in the
	// order replacement cards take them.
	NamePool []NobleName `json:"namePool"`
}

// Card returns the card with the given ID.
func (d *NobleDeck) Card(id NobleCardID) (NobleCard, bool) {
	if d == nil {
		return NobleCard{}, false
	}
	for _, card := range d.Cards {
		if card.ID == id {
			return card, true
		}
	}
	return NobleCard{}, false
}

// HandCard finds, in the player's hand, the card of the given kind whose
// code matches, and returns it with its index in the hand.
func (d *NobleDeck) HandCard(playerID PlayerID, kind NobleCardKind, code string) (NobleCard, int, bool) {
	if d == nil {
		return NobleCard{}, -1, false
	}
	for index, id := range d.Hands[playerID] {
		if card, exists := d.Card(id); exists && card.Kind == kind && card.Code == code {
			return card, index, true
		}
	}
	return NobleCard{}, -1, false
}

// HandCardByCode finds, in the player's hand, the first card of either kind
// whose code matches. Noble codes are trigrams that never equal a dignity
// card code (those are reserved when the deck is built), so the match is
// unambiguous for noble cards.
func (d *NobleDeck) HandCardByCode(playerID PlayerID, code string) (NobleCard, int, bool) {
	if d == nil {
		return NobleCard{}, -1, false
	}
	for index, id := range d.Hands[playerID] {
		if card, exists := d.Card(id); exists && card.Code == code {
			return card, index, true
		}
	}
	return NobleCard{}, -1, false
}

func validateNobleDeck(deck *NobleDeck, players map[PlayerID]bool, nobleCodes map[string]NobleID, nobles []Noble, claims []Claim) error {
	if deck == nil {
		return nil
	}
	cards := make(map[NobleCardID]NobleCard, len(deck.Cards))
	cardCodes := make(map[string]bool, len(deck.Cards))
	for _, card := range deck.Cards {
		if card.ID == "" || !card.Kind.IsValid() {
			return fmt.Errorf("models: noble card %q: invalid id or kind", card.ID)
		}
		if _, exists := cards[card.ID]; exists {
			return fmt.Errorf("models: noble card %q: duplicate id", card.ID)
		}
		switch card.Kind {
		case NobleCardKindNoble:
			if !isCode(card.Code, 3) || card.Name == "" || !card.Sex.IsValid() || card.Dignity != "" {
				return fmt.Errorf("models: noble card %q: invalid noble identity", card.ID)
			}
			if cardCodes[card.Code] {
				return fmt.Errorf("models: noble card %q: duplicate code %q", card.ID, card.Code)
			}
			cardCodes[card.Code] = true
		case NobleCardKindClaim:
			if card.Code != ClaimCardCode || card.Name != "" || card.Sex != "" || card.Dignity != "" {
				return fmt.Errorf("models: noble card %q: invalid claim card", card.ID)
			}
		case NobleCardKindDignity:
			if !card.Dignity.IsValid() || card.Code != card.Dignity.Effect().CardCode || card.Name != "" || card.Sex != "" {
				return fmt.Errorf("models: noble card %q: invalid dignity", card.ID)
			}
		}
		cards[card.ID] = card
	}
	locations := make(map[NobleCardID]string, len(cards))
	checkCard := func(id NobleCardID, location string) error {
		card, exists := cards[id]
		if !exists {
			return fmt.Errorf("models: noble card %q: %s references unknown card", id, location)
		}
		if previous, exists := locations[id]; exists {
			return fmt.Errorf("models: noble card %q: located in %s and %s", id, previous, location)
		}
		locations[id] = location
		// A noble card that is still in the deck reserves its code: no noble
		// in play may already carry it.
		if card.Kind == NobleCardKindNoble && location != "played" {
			if owner, taken := nobleCodes[card.Code]; taken {
				return fmt.Errorf("models: noble card %q: code %q is already used by noble %q", id, card.Code, owner)
			}
		}
		return nil
	}
	for index, id := range deck.DrawPile {
		if err := checkCard(id, fmt.Sprintf("draw pile index %d", index)); err != nil {
			return err
		}
	}
	for index, id := range deck.Discard {
		if err := checkCard(id, fmt.Sprintf("discard index %d", index)); err != nil {
			return err
		}
	}
	livingNobles := make(map[NobleID]Noble, len(nobles))
	for _, noble := range nobles {
		livingNobles[noble.ID] = noble
	}
	for _, play := range deck.Played {
		if err := checkCard(play.Card, "played"); err != nil {
			return err
		}
		card := cards[play.Card]
		noble, alive := livingNobles[play.Noble]
		switch {
		case !alive:
			return fmt.Errorf("models: noble card %q: played on unknown noble %q", play.Card, play.Noble)
		case card.Kind == NobleCardKindNoble && noble.Code != card.Code && noble.SecretCode != card.Code:
			return fmt.Errorf("models: noble card %q: noble %q has code %q", play.Card, noble.ID, noble.Code)
		case card.Kind == NobleCardKindClaim:
			if !hasClaim(claims, noble.ID) {
				return fmt.Errorf("models: noble card %q: noble %q holds no claim", play.Card, noble.ID)
			}
		case card.Kind == NobleCardKindDignity && !noble.Has(card.Dignity):
			return fmt.Errorf("models: noble card %q: noble %q does not carry dignity %q", play.Card, noble.ID, card.Dignity)
		}
	}
	for playerID, hand := range deck.Hands {
		if !players[playerID] {
			return fmt.Errorf("models: noble deck: hand has unknown player %q", playerID)
		}
		for index, id := range hand {
			if err := checkCard(id, fmt.Sprintf("hand of player %q index %d", playerID, index)); err != nil {
				return err
			}
		}
	}
	if deck.Reshuffles < 0 {
		return fmt.Errorf("models: noble deck: negative reshuffle count %d", deck.Reshuffles)
	}
	poolCodes := make(map[string]bool, len(deck.NamePool))
	for _, name := range deck.NamePool {
		if !isCode(name.Code, 3) || name.Name == "" || !name.Sex.IsValid() {
			return fmt.Errorf("models: noble deck: invalid pool name %+v", name)
		}
		if poolCodes[name.Code] || cardCodes[name.Code] {
			return fmt.Errorf("models: noble deck: pool code %q already used by a card", name.Code)
		}
		if owner, taken := nobleCodes[name.Code]; taken {
			return fmt.Errorf("models: noble deck: pool code %q is already used by noble %q", name.Code, owner)
		}
		poolCodes[name.Code] = true
	}
	if len(locations) != len(cards) {
		return fmt.Errorf("models: noble deck: %d of %d cards have no location", len(cards)-len(locations), len(cards))
	}
	return nil
}

func validateNobleDignities(noble Noble) error {
	seen := make(map[Dignity]bool, len(noble.Dignities))
	for _, dignity := range noble.Dignities {
		if !dignity.IsValid() {
			return fmt.Errorf("models: noble %q: invalid dignity %q", noble.ID, dignity)
		}
		if seen[dignity] {
			return fmt.Errorf("models: noble %q: duplicate dignity %q", noble.ID, dignity)
		}
		seen[dignity] = true
	}
	return nil
}
