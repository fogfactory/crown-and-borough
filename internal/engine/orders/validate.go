package orders

import (
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
		errors = append(errors, chainValidationError("unknown_noble", "error.validation.unknown_noble", nobleReference(indexes, chain.NobleID)))
	}
	if len(chain.Orders) == 0 {
		errors = append(errors, chainValidationError("empty_chain", "error.validation.empty_chain"))
		return errors
	}

	orderIDs := make(map[models.OrderID]bool, len(chain.Orders))
	for index, order := range chain.Orders {
		if order.ID != "" && orderIDs[order.ID] {
			errors = append(errors, orderValidationError(order, "duplicate_order_id", "error.validation.duplicate_order", order.ID))
		}
		orderIDs[order.ID] = true
		errors = append(errors, validateOrder(indexes, order, index == len(chain.Orders)-1)...)
	}
	return errors
}

func chainValidationError(code, key string, args ...any) ValidationError {
	return validationMessage("", code, key, args...)
}

func orderValidationError(order models.Order, code, key string, args ...any) ValidationError {
	return validationMessage(order.ID, code, key, args...)
}

func validateOrder(indexes gameIndexes, order models.Order, last bool) []ValidationError {
	errors := []ValidationError{}
	if order.ID == "" {
		errors = append(errors, orderValidationError(order, "invalid_order_id", "error.validation.invalid_order_id"))
	}
	if !order.Type.IsValid() {
		errors = append(errors, orderValidationError(order, "invalid_order_type", "error.validation.invalid_order_type", order.Type))
	}
	if !order.Liaison.IsValid() {
		errors = append(errors, orderValidationError(order, "invalid_liaison", "error.validation.invalid_liaison", order.Liaison))
	}
	if order.Type != models.OrderTypeDisperse && len(order.NobleAssignments) != 0 {
		errors = append(errors, orderValidationError(order, "unexpected_noble_assignments", "error.validation.unexpected_noble_assignments", order.Type))
	}
	positionExists := indexes.territoriesByID[order.PositionID] != nil
	if !positionExists {
		errors = append(errors, orderValidationError(order, "unknown_position", "error.validation.unknown_position", territoryReference(indexes, order.PositionID)))
	}
	for _, targetID := range order.TargetIDs {
		if indexes.territoriesByID[targetID] == nil {
			errors = append(errors, orderValidationError(order, "unknown_target", "error.validation.unknown_target", territoryReference(indexes, targetID)))
		}
	}
	switch order.Type {
	case models.OrderTypeAttack:
		errors = append(errors, validateSingleTerritoryOrder(indexes, order, positionExists, "A")...)
	case models.OrderTypeJoin:
		errors = append(errors, validateSingleTerritoryOrder(indexes, order, positionExists, "J")...)
		if !last {
			errors = append(errors, orderValidationError(order, "join_not_last", "error.validation.join_not_last"))
		}
	case models.OrderTypeSupport:
		errors = append(errors, validateSupport(indexes, order, positionExists)...)
	case models.OrderTypeHold, models.OrderTypePillage:
		if len(order.TargetIDs) != 0 {
			errors = append(errors, orderValidationError(order, "unexpected_target", "error.validation.unexpected_target", order.Type))
		}
	case models.OrderTypeDisperse:
		errors = append(errors, validateDisperse(indexes, order, positionExists)...)
	}
	return errors
}

func validateSingleTerritoryOrder(indexes gameIndexes, order models.Order, positionExists bool, symbol string) []ValidationError {
	errors := []ValidationError{}
	if len(order.TargetIDs) == 0 {
		errors = append(errors, orderValidationError(order, "missing_target", "error.validation.missing_target", symbol))
		return errors
	}
	if len(order.TargetIDs) > 1 {
		errors = append(errors, orderValidationError(order, "too_many_targets", "error.validation.too_many_targets", symbol))
	}
	targetID := order.TargetIDs[0]
	if positionExists && indexes.territoriesByID[targetID] != nil && !adjacent(indexes, order.PositionID, targetID) {
		errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, "error.validation.not_adjacent", symbol, territoryReference(indexes, targetID), territoryReference(indexes, order.PositionID)))
	}
	return errors
}

func validateSupport(indexes gameIndexes, order models.Order, positionExists bool) []ValidationError {
	errors := []ValidationError{}
	if len(order.TargetIDs) == 0 {
		return []ValidationError{orderValidationError(order, "missing_target", "error.parse.support_position_required")}
	}
	if len(order.TargetIDs) > 2 {
		errors = append(errors, orderValidationError(order, "too_many_targets", "error.parse.support_shape"))
	}
	supportedID := order.TargetIDs[0]
	if len(order.TargetIDs) == 1 {
		if supportedID == order.PositionID {
			errors = append(errors, orderValidationError(order, "support_same_position", "error.validation.support_same_position"))
		}
		if positionExists && indexes.territoriesByID[supportedID] != nil && !adjacent(indexes, order.PositionID, supportedID) {
			errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, "error.validation.not_adjacent", "S", territoryReference(indexes, supportedID), territoryReference(indexes, order.PositionID)))
		}
		return errors
	}
	destinationID := order.TargetIDs[1]
	if positionExists && indexes.territoriesByID[destinationID] != nil && !adjacent(indexes, order.PositionID, destinationID) {
		errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, "error.validation.not_adjacent", "S", territoryReference(indexes, destinationID), territoryReference(indexes, order.PositionID)))
	}
	if indexes.territoriesByID[supportedID] != nil && indexes.territoriesByID[destinationID] != nil && !adjacent(indexes, supportedID, destinationID) {
		errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, "error.validation.not_adjacent", "S", territoryReference(indexes, supportedID), territoryReference(indexes, destinationID)))
	}
	return errors
}

func validateDisperse(indexes gameIndexes, order models.Order, positionExists bool) []ValidationError {
	errors := []ValidationError{}
	if len(order.TargetIDs) == 0 {
		errors = append(errors, orderValidationError(order, "missing_target", "error.parse.disperse_destination_required"))
	}
	for _, targetID := range order.TargetIDs {
		if positionExists && indexes.territoriesByID[targetID] != nil && targetID != order.PositionID && !adjacent(indexes, order.PositionID, targetID) {
			errors = append(errors, orderValidationError(order, ValidationCodeNotAdjacent, "error.validation.not_adjacent", "D", territoryReference(indexes, targetID), territoryReference(indexes, order.PositionID)))
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
			errors = append(errors, orderValidationError(order, "unknown_assignment_destination", "error.validation.unknown_assignment_destination", destination))
			continue
		}
		if !targets[destinationID] {
			errors = append(errors, orderValidationError(order, "assignment_destination_not_declared", "error.validation.assignment_destination_missing", destination))
		}
		for _, nobleCode := range order.NobleAssignments[destination] {
			if nobleCode == "*" {
				wildcards++
				continue
			}
			if _, exists := indexes.noblesByCode[string(nobleCode)]; !exists {
				errors = append(errors, orderValidationError(order, "unknown_assignment_noble", "error.validation.unknown_assignment_noble", nobleCode))
				continue
			}
			if assignedNobles[nobleCode] {
				errors = append(errors, orderValidationError(order, "duplicate_assignment_noble", "error.validation.duplicate_assignment_noble", nobleCode))
			}
			assignedNobles[nobleCode] = true
		}
	}
	if wildcards > 1 {
		errors = append(errors, orderValidationError(order, "multiple_wildcards", "error.validation.multiple_wildcards"))
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
