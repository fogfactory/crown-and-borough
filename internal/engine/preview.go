package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// OrdersPreview is a dry run of one player's draft against the current state.
// It never changes the game. Unlike ResolveTurn, it keeps going past invalid
// lines so a client can show the valid orders next to the errors, and it
// reports the outcome the winter resolver would give each winter line if the
// player's sheet were resolved alone.
type OrdersPreview struct {
	// Errors lists every problem that would reject the submission, followed by
	// the chains the engine would refuse to receive.
	Errors []InputError
	// Chains holds, per submitted chain, the order lines that parse.
	Chains []ChainPreview
	// Winter holds one entry per non-empty winter line, in source order.
	Winter []WinterLinePreview
	// WinterCost is set during winter only.
	WinterCost *WinterCostPreview
}

// ChainPreview is the parsed content of one submitted chain.
type ChainPreview struct {
	Noble  models.NobleCode
	Orders []models.Order
}

// WinterLinePreview is one winter line: exactly one of Order, Discard or
// ParseError is set. Applied, Reason, Cost, Level and Territory describe the
// simulated outcome of Order.
type WinterLinePreview struct {
	Line       int
	Order      *models.WinterOrder
	Discard    *models.DeckOrder
	ParseError *InputError
	Applied    bool
	Reason     string
	Cost       int
	Level      int
	Territory  models.TerritoryID
}

// WinterCostPreview compares the resources the simulated winter sheet spends
// with the player's payment reserves (controlled castles and villages).
type WinterCostPreview struct {
	Spent     int
	Available int
}

// PreviewOrders dry-runs input, which must already belong to playerID.
func PreviewOrders(game *models.GameState, balance assetgen.Balance, playerID models.PlayerID, input OrdersInput) (OrdersPreview, error) {
	preview := OrdersPreview{Errors: []InputError{}, Chains: []ChainPreview{}, Winter: []WinterLinePreview{}}
	if game == nil {
		return preview, fmt.Errorf("engine: preview orders: nil game state")
	}

	report, err := ResolveTurn(game, balance, input)
	var inputErrors *InputErrors
	switch {
	case errors.As(err, &inputErrors):
		preview.Errors = append(preview.Errors, inputErrors.Errors...)
	case err != nil:
		return preview, err
	default:
		for _, reception := range report.Receptions {
			if reception.Received {
				continue
			}
			preview.Errors = append(preview.Errors, InputError{
				Player:      reception.Player,
				Noble:       reception.Noble,
				Code:        "not_received",
				Message:     reception.Reason,
				MessageKey:  reception.ReasonKey,
				MessageArgs: reception.ReasonArgs,
			})
		}
	}

	if game.Season != models.SeasonWinter {
		for _, submission := range input.Chains {
			chain, _ := orders.ParseChain(submission.Text, game)
			preview.Chains = append(preview.Chains, ChainPreview{Noble: submission.Noble, Orders: chain.Orders})
		}
		return preview, nil
	}

	lines := make([]string, 0, len(input.Winter))
	for _, submission := range input.Winter {
		lines = append(lines, submission.Lines)
	}
	if err := previewWinter(&preview, game, balance, playerID, strings.Join(lines, "\n")); err != nil {
		return preview, err
	}
	return preview, nil
}

func previewWinter(preview *OrdersPreview, game *models.GameState, balance assetgen.Balance, playerID models.PlayerID, text string) error {
	winterOrders := []models.WinterOrder{}
	deckOrders := []models.DeckOrder{}
	for _, line := range orders.ParseWinterSheetLines(text, game) {
		entry := WinterLinePreview{Line: line.Line}
		switch {
		case line.Error != nil:
			parseError := newInputError(playerID, "", line.Line, "parse_"+line.Error.Code, line.Error.MessageKey, line.Error.MessageArgs...)
			entry.ParseError = &parseError
		case line.Deck != nil:
			deck := *line.Deck
			deck.ID = models.OrderID(fmt.Sprintf("D%d", line.Line))
			deckOrders = append(deckOrders, deck)
			entry.Discard = &deck
		default:
			order := *line.Winter
			order.ID = models.OrderID(fmt.Sprintf("W%d", line.Line))
			winterOrders = append(winterOrders, order)
			entry.Order = &order
		}
		preview.Winter = append(preview.Winter, entry)
	}

	cost := &WinterCostPreview{Available: winterPaymentReserves(game, playerID)}
	preview.WinterCost = cost
	if len(winterOrders) == 0 && len(deckOrders) == 0 {
		return nil
	}

	resolution, err := ResolveWinterWithDeckOrders(game, balance,
		map[models.PlayerID][]models.WinterOrder{playerID: winterOrders},
		map[models.PlayerID][]models.DeckOrder{playerID: deckOrders},
	)
	if err != nil {
		return fmt.Errorf("engine: preview winter: %w", err)
	}
	outcomes := make(map[models.OrderID]Event)
	for _, event := range resolution.Events {
		if event.OrderID == "" {
			continue
		}
		if _, seen := outcomes[event.OrderID]; !seen {
			outcomes[event.OrderID] = event
		}
	}
	for index := range preview.Winter {
		entry := &preview.Winter[index]
		if entry.Order == nil {
			continue
		}
		event, ok := outcomes[entry.Order.ID]
		if !ok {
			continue
		}
		entry.Applied = event.Type != EventTypeRejected
		entry.Reason = event.Reason
		entry.Cost = event.ResourceSpent
		entry.Level = event.Level
		entry.Territory = event.TerritoryID
		if entry.Territory == "" {
			entry.Territory = winterOrderTerritory(game, *entry.Order)
		}
		if entry.Applied {
			cost.Spent += entry.Cost
		}
	}
	return nil
}

// winterPaymentReserves is the stock a player can spend on winter orders:
// the resources of every controlled castle and village.
func winterPaymentReserves(game *models.GameState, playerID models.PlayerID) int {
	settlements := make(map[models.InfraID]bool)
	for _, infrastructure := range game.Infrastructures {
		if infrastructure.Type == models.InfraTypeCastle || infrastructure.Type == models.InfraTypeVillage {
			settlements[infrastructure.ID] = true
		}
	}
	total := 0
	for _, territoryState := range game.TerritoryStates {
		if territoryState.OwnerID == nil || *territoryState.OwnerID != playerID || territoryState.Resources <= 0 {
			continue
		}
		for _, infrastructureID := range territoryState.Infrastructures {
			if settlements[infrastructureID] {
				total += territoryState.Resources
				break
			}
		}
	}
	return total
}

// winterOrderTerritory locates an order that names a noble or a transfer
// source rather than a territory.
func winterOrderTerritory(game *models.GameState, order models.WinterOrder) models.TerritoryID {
	if order.TerritoryID != "" {
		return order.TerritoryID
	}
	if order.SourceID != "" {
		return order.SourceID
	}
	for _, noble := range game.Nobles {
		if models.NobleCode(noble.Code) == order.NobleCode {
			return noble.LocationID
		}
	}
	return ""
}
