package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func validateDeckOrders(game *models.GameState, balance assetgen.Balance, deckOrders map[models.PlayerID][]models.DeckOrder) error {
	players := make(map[models.PlayerID]bool, len(game.Players))
	for _, player := range game.Players {
		players[player.ID] = true
	}
	hands := make(map[models.PlayerID][]models.SpecialCardID)
	if game.SpecialDeck != nil {
		for playerID, hand := range game.SpecialDeck.Hands {
			hands[playerID] = append([]models.SpecialCardID(nil), hand...)
		}
	}
	playerIDs := make([]models.PlayerID, 0, len(deckOrders))
	for playerID := range deckOrders {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Slice(playerIDs, func(i, j int) bool { return playerIDs[i] < playerIDs[j] })
	drawLimit := balance.SpecialOrders.DrawOrdersLimit
	for _, playerID := range playerIDs {
		if !players[playerID] {
			return fmt.Errorf("engine: resolve winter: unknown player %q", playerID)
		}
		draws := 0
		for _, order := range deckOrders[playerID] {
			switch order.Type {
			case models.DeckOrderTypeDraw:
				draws++
				if draws > drawLimit {
					return fmt.Errorf("engine: resolve winter: player %q exceeds deck draw limit", playerID)
				}
			case models.DeckOrderTypeDiscard:
				if !order.Kind.IsBonus() {
					return fmt.Errorf("engine: resolve winter: invalid discard kind %q", order.Kind)
				}
				index := -1
				for handIndex, cardID := range hands[playerID] {
					if cardKind(game.SpecialDeck, cardID) == order.Kind {
						index = handIndex
						break
					}
				}
				if index < 0 {
					return fmt.Errorf("engine: resolve winter: player %q has no card of kind %q", playerID, order.Kind)
				}
				hands[playerID] = append(hands[playerID][:index], hands[playerID][index+1:]...)
			case models.DeckOrderTypePlay:
				return fmt.Errorf("engine: resolve winter: deck order %q is not playable in winter", order.Kind)
			default:
				return fmt.Errorf("engine: resolve winter: invalid deck order %q", order.Type)
			}
		}
	}
	return nil
}

func cardKind(deck *models.SpecialDeck, cardID models.SpecialCardID) models.CardKind {
	if deck == nil {
		return ""
	}
	for _, card := range deck.Cards {
		if card.ID == cardID {
			return card.Kind
		}
	}
	return ""
}

func resolveWinterDeckOrders(ctx *resolutionContext, deckOrders map[models.PlayerID][]models.DeckOrder) {
	for _, playerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, order := range deckOrders[playerID] {
			switch order.Type {
			case models.DeckOrderTypeDiscard:
				ctx.consumeDeckCard(playerID, order.Kind)
			case models.DeckOrderTypeDraw:
				ctx.drawUsefulDeckCard(playerID)
			}
		}
	}
}

func (ctx *resolutionContext) drawUsefulDeckCard(playerID models.PlayerID) {
	if ctx.state.SpecialDeck == nil || len(ctx.state.SpecialDeck.Hands[playerID]) >= ctx.balance.SpecialOrders.HandLimit {
		return
	}
	cycle := len(ctx.state.SpecialDeck.DrawPile) + len(ctx.state.SpecialDeck.Discard)
	for processed := 0; processed < cycle; processed++ {
		cardID, ok := ctx.drawDeckCard()
		if !ok {
			return
		}
		kind := cardKind(ctx.state.SpecialDeck, cardID)
		if kind.IsBonus() {
			ctx.state.SpecialDeck.Hands[playerID] = append(ctx.state.SpecialDeck.Hands[playerID], cardID)
			return
		}
		if kind.IsCalamity() {
			if !ctx.programCalamity(cardID, kind) {
				ctx.state.SpecialDeck.Discard = append(ctx.state.SpecialDeck.Discard, cardID)
			}
		}
	}
}

func (ctx *resolutionContext) drawDeckCard() (models.SpecialCardID, bool) {
	if ctx.state.SpecialDeck == nil {
		return "", false
	}
	if len(ctx.state.SpecialDeck.DrawPile) == 0 {
		if len(ctx.state.SpecialDeck.Discard) == 0 {
			return "", false
		}
		ctx.state.SpecialDeck.DrawPile = append([]models.SpecialCardID(nil), ctx.state.SpecialDeck.Discard...)
		ctx.state.SpecialDeck.Discard = []models.SpecialCardID{}
		ctx.specialReshuffles++
		shuffleSpecialIDs(newReshuffleRNG(ctx.state.Seed, ctx.state.Turn, ctx.specialReshuffles), ctx.state.SpecialDeck.DrawPile)
	}
	cardID := ctx.state.SpecialDeck.DrawPile[0]
	ctx.state.SpecialDeck.DrawPile = ctx.state.SpecialDeck.DrawPile[1:]
	return cardID, true
}

func (ctx *resolutionContext) programCalamity(cardID models.SpecialCardID, kind models.CardKind) bool {
	if ctx.state.SpecialDeck == nil || len(ctx.state.Regions) == 0 {
		return false
	}
	year := ctx.state.Year() + 1
	augury := ctx.state.Auguries[year]
	if augury.Year == 0 {
		augury.Year = year
		augury.Capacities = map[models.Season]int{
			models.SeasonSpring: ctx.balance.SpecialOrders.CalamitySlots[models.SeasonSpring],
			models.SeasonSummer: ctx.balance.SpecialOrders.CalamitySlots[models.SeasonSummer],
			models.SeasonWinter: ctx.balance.SpecialOrders.CalamitySlots[models.SeasonWinter],
		}
		augury.Calamities = []models.Calamity{}
	}
	counts := map[models.Season]int{}
	for _, calamity := range augury.Calamities {
		counts[calamity.Season]++
	}
	season := models.Season("")
	for _, candidate := range []models.Season{models.SeasonSpring, models.SeasonSummer, models.SeasonWinter} {
		if counts[candidate] < augury.Capacities[candidate] {
			season = candidate
			break
		}
	}
	if season == "" {
		return false
	}
	seeds := make([]models.TerritoryID, len(ctx.state.Regions))
	for index, region := range ctx.state.Regions {
		seeds[index] = region.Seed
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i] < seeds[j] })
	regionRNG := newCalamityRegionRNG(ctx.state.Seed, ctx.state.Turn, cardID)
	regionSeed := seeds[regionRNG.IntN(len(seeds))]
	augury.Calamities = append(augury.Calamities, models.Calamity{CardID: cardID, Kind: kind, Year: year, Season: season, RegionSeed: regionSeed})
	ctx.state.Auguries[year] = augury
	return true
}

func newReshuffleRNG(seed string, turn, count int) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|special-reshuffle|%d|%d", seed, turn, count)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func newCalamityRegionRNG(seed string, turn int, cardID models.SpecialCardID) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|calamity-region|%d|%s", seed, turn, cardID)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func shuffleSpecialIDs(rng *rand.Rand, values []models.SpecialCardID) {
	for index := len(values) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		values[index], values[swap] = values[swap], values[index]
	}
}
