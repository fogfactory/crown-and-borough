package orders

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ValidateChain checks references, adjacency, order shape, and other rules
// intrinsic to a chain. It does not mutate its inputs and intentionally leaves
// execution-time world conditions to P1.4.
func ValidateChain(game *models.GameState, chain models.Chain) []ValidationError {
	indexes := indexGame(game)
	errors := []ValidationError{}
	if _, exists := indexes.noblesByID[chain.NobleID]; !exists {
		errors = append(errors, chainValidationError("unknown_noble", fmt.Sprintf("noble %q does not exist", nobleReference(indexes, chain.NobleID))))
	}
	if len(chain.Orders) == 0 {
		errors = append(errors, chainValidationError("empty_chain", "a chain must contain at least one order"))
		return errors
	}

	orderIDs := make(map[models.OrderID]bool, len(chain.Orders))
	for index, order := range chain.Orders {
		if order.ID != "" && orderIDs[order.ID] {
			errors = append(errors, orderValidationError(order, "duplicate_order_id", fmt.Sprintf("order id %q is duplicated", order.ID)))
		}
		orderIDs[order.ID] = true
		errors = append(errors, validateOrder(indexes, order, index == len(chain.Orders)-1)...)
	}
	return errors
}

func chainValidationError(code, message string) ValidationError {
	return ValidationError{Code: code, Message: message}
}

func orderValidationError(order models.Order, code, message string) ValidationError {
	return ValidationError{OrderID: order.ID, Code: code, Message: message}
}

func validateOrder(indexes gameIndexes, order models.Order, last bool) []ValidationError {
	errors := []ValidationError{}
	if order.ID == "" {
		errors = append(errors, orderValidationError(order, "invalid_order_id", "an order must have an id"))
	}
	if !order.Type.IsValid() {
		errors = append(errors, orderValidationError(order, "invalid_order_type", fmt.Sprintf("order type %q is invalid", order.Type)))
	}
	if !order.Liaison.IsValid() {
		errors = append(errors, orderValidationError(order, "invalid_liaison", fmt.Sprintf("liaison %q is invalid", order.Liaison)))
	}
	if order.Type != models.OrderTypeDisperse && len(order.NobleAssignments) != 0 {
		errors = append(errors, orderValidationError(order, "unexpected_noble_assignments", fmt.Sprintf("%s does not accept noble assignments", order.Type)))
	}
	positionExists := indexes.territoriesByID[order.PositionID] != nil
	if !positionExists {
		errors = append(errors, orderValidationError(order, "unknown_position", fmt.Sprintf("position %q does not exist", territoryReference(indexes, order.PositionID))))
	}
	for _, targetID := range order.TargetIDs {
		if indexes.territoriesByID[targetID] == nil {
			errors = append(errors, orderValidationError(order, "unknown_target", fmt.Sprintf("territory target %q does not exist", territoryReference(indexes, targetID))))
		}
	}
	for _, nobleTargetID := range order.NobleTargetIDs {
		if _, exists := indexes.noblesByID[nobleTargetID]; !exists {
			errors = append(errors, orderValidationError(order, "unknown_noble_target", fmt.Sprintf("noble target %q does not exist", nobleReference(indexes, nobleTargetID))))
		}
	}

	switch order.Type {
	case models.OrderTypeAttack:
		errors = append(errors, validateSingleTerritoryOrder(indexes, order, positionExists, "A")...)
	case models.OrderTypeJoin:
		errors = append(errors, validateSingleTerritoryOrder(indexes, order, positionExists, "J")...)
		if !last {
			errors = append(errors, orderValidationError(order, "join_not_last", "J must be the last order in a chain"))
		}
	case models.OrderTypeSupport:
		errors = append(errors, validateSupport(indexes, order, positionExists)...)
	case models.OrderTypeHold, models.OrderTypePillage:
		if len(order.TargetIDs) != 0 {
			errors = append(errors, orderValidationError(order, "unexpected_target", fmt.Sprintf("%s does not accept territory targets", order.Type)))
		}
		if len(order.NobleTargetIDs) != 0 {
			errors = append(errors, orderValidationError(order, "unexpected_noble_target", fmt.Sprintf("%s does not accept noble targets", order.Type)))
		}
	case models.OrderTypeDisperse:
		errors = append(errors, validateDisperse(indexes, order, positionExists)...)
	case models.OrderTypeHostage, models.OrderTypeDungeon:
		errors = append(errors, validateNobleTargetOrder(order)...)
	}
	return errors
}

func validateSingleTerritoryOrder(indexes gameIndexes, order models.Order, positionExists bool, symbol string) []ValidationError {
	errors := []ValidationError{}
	if len(order.TargetIDs) == 0 {
		errors = append(errors, orderValidationError(order, "missing_target", fmt.Sprintf("%s requires one territory target", symbol)))
		return errors
	}
	if len(order.TargetIDs) > 1 {
		errors = append(errors, orderValidationError(order, "too_many_targets", fmt.Sprintf("%s accepts one territory target", symbol)))
	}
	if len(order.NobleTargetIDs) != 0 {
		errors = append(errors, orderValidationError(order, "unexpected_noble_target", fmt.Sprintf("%s does not accept noble targets", symbol)))
	}
	targetID := order.TargetIDs[0]
	if positionExists && indexes.territoriesByID[targetID] != nil && !adjacent(indexes, order.PositionID, targetID) {
		errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, fmt.Sprintf("%s target %q is not adjacent to %q", symbol, territoryReference(indexes, targetID), territoryReference(indexes, order.PositionID))))
	}
	return errors
}

func validateSupport(indexes gameIndexes, order models.Order, positionExists bool) []ValidationError {
	errors := []ValidationError{}
	if len(order.TargetIDs) == 0 {
		return []ValidationError{orderValidationError(order, "missing_target", "S requires a supported position")}
	}
	if len(order.TargetIDs) > 2 {
		errors = append(errors, orderValidationError(order, "too_many_targets", "S accepts one supported position and one optional attack destination"))
	}
	if len(order.NobleTargetIDs) != 0 {
		errors = append(errors, orderValidationError(order, "unexpected_noble_target", "S does not accept noble targets"))
	}
	supportedID := order.TargetIDs[0]
	if len(order.TargetIDs) == 1 {
		if supportedID == order.PositionID {
			errors = append(errors, orderValidationError(order, "support_same_position", "defensive S cannot support its own position"))
		}
		if positionExists && indexes.territoriesByID[supportedID] != nil && !adjacent(indexes, order.PositionID, supportedID) {
			errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, fmt.Sprintf("defensive S target %q is not adjacent to %q", territoryReference(indexes, supportedID), territoryReference(indexes, order.PositionID))))
		}
		return errors
	}
	destinationID := order.TargetIDs[1]
	if positionExists && indexes.territoriesByID[destinationID] != nil && !adjacent(indexes, order.PositionID, destinationID) {
		errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, fmt.Sprintf("offensive S destination %q is not adjacent to %q", territoryReference(indexes, destinationID), territoryReference(indexes, order.PositionID))))
	}
	if indexes.territoriesByID[supportedID] != nil && indexes.territoriesByID[destinationID] != nil && !adjacent(indexes, supportedID, destinationID) {
		errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, fmt.Sprintf("offensive S supported position %q is not adjacent to %q", territoryReference(indexes, supportedID), territoryReference(indexes, destinationID))))
	}
	return errors
}

func validateDisperse(indexes gameIndexes, order models.Order, positionExists bool) []ValidationError {
	errors := []ValidationError{}
	if len(order.TargetIDs) == 0 {
		errors = append(errors, orderValidationError(order, "missing_target", "D requires at least one destination"))
	}
	if len(order.NobleTargetIDs) != 0 {
		errors = append(errors, orderValidationError(order, "unexpected_noble_target", "D does not accept noble targets"))
	}
	for _, targetID := range order.TargetIDs {
		if positionExists && indexes.territoriesByID[targetID] != nil && targetID != order.PositionID && !adjacent(indexes, order.PositionID, targetID) {
			errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, fmt.Sprintf("D destination %q is not adjacent to %q", territoryReference(indexes, targetID), territoryReference(indexes, order.PositionID))))
		}
	}

	targets := make(map[models.TerritoryID]bool, len(order.TargetIDs))
	for _, targetID := range order.TargetIDs {
		targets[targetID] = true
	}
	assignedNobles := map[models.NobleCode]bool{}
	wildcards := 0
	for _, destination := range sortedDestinations(order.NobleAssignments) {
		destinationID, exists := indexes.territoriesByCode[string(destination)]
		if !exists {
			errors = append(errors, orderValidationError(order, "unknown_assignment_destination", fmt.Sprintf("assignment destination %q does not exist", destination)))
			continue
		}
		if !targets[destinationID] {
			errors = append(errors, orderValidationError(order, "assignment_destination_not_declared", fmt.Sprintf("assignment destination %q is not listed in D targets", destination)))
		}
		for _, nobleCode := range order.NobleAssignments[destination] {
			if nobleCode == "*" {
				wildcards++
				continue
			}
			if _, exists := indexes.noblesByCode[string(nobleCode)]; !exists {
				errors = append(errors, orderValidationError(order, "unknown_assignment_noble", fmt.Sprintf("assigned noble %q does not exist", nobleCode)))
				continue
			}
			if assignedNobles[nobleCode] {
				errors = append(errors, orderValidationError(order, "duplicate_assignment_noble", fmt.Sprintf("noble %q is assigned more than once", nobleCode)))
			}
			assignedNobles[nobleCode] = true
		}
	}
	if wildcards > 1 {
		errors = append(errors, orderValidationError(order, "multiple_wildcards", "D accepts at most one remaining-nobles wildcard"))
	}
	return errors
}

func validateNobleTargetOrder(order models.Order) []ValidationError {
	errors := []ValidationError{}
	if len(order.TargetIDs) != 0 {
		errors = append(errors, orderValidationError(order, "unexpected_target", fmt.Sprintf("%s does not accept territory targets", order.Type)))
	}
	if len(order.NobleTargetIDs) == 0 {
		errors = append(errors, orderValidationError(order, "missing_target", fmt.Sprintf("%s requires one noble target", order.Type)))
	}
	if len(order.NobleTargetIDs) > 1 {
		errors = append(errors, orderValidationError(order, "too_many_targets", fmt.Sprintf("%s accepts one noble target", order.Type)))
	}
	return errors
}

func sortedDestinations(assignments map[models.TerritoryCode][]models.NobleCode) []models.TerritoryCode {
	destinations := make([]models.TerritoryCode, 0, len(assignments))
	for destination := range assignments {
		destinations = append(destinations, destination)
	}
	sort.Slice(destinations, func(i, j int) bool { return destinations[i] < destinations[j] })
	return destinations
}
