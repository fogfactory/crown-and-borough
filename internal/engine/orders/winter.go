package orders

import (
	"fmt"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ParseWinterOrders parses direct winter orders. It preserves source line
// numbers and returns every line error. Any parsing error makes the returned
// order batch empty, so malformed input cannot be partially resolved.
func ParseWinterOrders(text string, game *models.GameState) ([]models.WinterOrder, []ParseError) {
	indexes := indexGame(game)
	orders := []models.WinterOrder{}
	parseErrors := []ParseError{}
	for lineNumber, sourceLine := range strings.Split(text, "\n") {
		line := normalizeLine(sourceLine)
		if line == "" {
			continue
		}
		order, parseError := parseWinterOrderLine(line, lineNumber+1, indexes)
		if parseError != nil {
			parseErrors = append(parseErrors, *parseError)
			continue
		}
		order.ID = models.OrderID(fmt.Sprintf("O%d", len(orders)+1))
		orders = append(orders, order)
	}
	if len(parseErrors) != 0 {
		return nil, parseErrors
	}
	return orders, nil
}

func parseWinterOrderLine(line string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return models.WinterOrder{}, &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeMissingTarget,
			Message: "a winter order requires a symbol, a subtype, and one target code",
		}
	}
	if len(fields) > 3 {
		return models.WinterOrder{}, &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeTooManyTargets,
			Message: "a winter order accepts exactly one target code",
		}
	}

	switch fields[0] {
	case "R":
		territoryID, parseError := winterTerritoryID(fields[2], lineNumber, indexes)
		if parseError != nil {
			return models.WinterOrder{}, parseError
		}
		switch fields[1] {
		case "N":
			return models.WinterOrder{Type: models.WinterOrderTypeRecruitNoble, TerritoryID: territoryID}, nil
		case "T":
			return models.WinterOrder{Type: models.WinterOrderTypeRecruitTroop, TerritoryID: territoryID}, nil
		default:
			return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
		}
	case "C":
		territoryID, parseError := winterTerritoryID(fields[2], lineNumber, indexes)
		if parseError != nil {
			return models.WinterOrder{}, parseError
		}
		infrastructureType, exists := map[string]models.InfraType{
			"M": models.InfraTypeMill,
			"C": models.InfraTypeCastle,
			"R": models.InfraTypePostRelay,
			"T": models.InfraTypeWatchtower,
			"D": models.InfraTypeSupplyDepot,
		}[fields[1]]
		if !exists {
			return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
		}
		return models.WinterOrder{Type: models.WinterOrderTypeBuild, TerritoryID: territoryID, InfraType: infrastructureType}, nil
	case "E":
		if fields[1] != "C" {
			return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
		}
		territoryID, parseError := winterTerritoryID(fields[2], lineNumber, indexes)
		if parseError != nil {
			return models.WinterOrder{}, parseError
		}
		return models.WinterOrder{Type: models.WinterOrderTypeElectCapital, TerritoryID: territoryID}, nil
	case "L":
		if fields[1] != "N" {
			return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
		}
		if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
		return models.WinterOrder{Type: models.WinterOrderTypeLiberateNoble, NobleCode: models.NobleCode(fields[2])}, nil
	default:
		return models.WinterOrder{}, &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeUnknownSymbol,
			Message: fmt.Sprintf("unknown winter order symbol %q", fields[0]),
		}
	}
}

func winterTerritoryID(code string, lineNumber int, indexes gameIndexes) (models.TerritoryID, *ParseError) {
	if !isCode(code) {
		return "", &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("territory code %q must contain exactly three uppercase letters", code),
		}
	}
	territoryID, exists := indexes.territoriesByCode[code]
	if !exists {
		return "", &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("territory code %q does not exist", code),
		}
	}
	return territoryID, nil
}

func winterNobleCode(code string, lineNumber int, indexes gameIndexes) *ParseError {
	if !isCode(code) {
		return &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("noble code %q must contain exactly three uppercase letters", code),
		}
	}
	if _, exists := indexes.noblesByCode[code]; !exists {
		return &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("noble code %q does not exist", code),
		}
	}
	return nil
}

func unknownWinterSubtype(lineNumber int, symbol, subtype string) *ParseError {
	return &ParseError{
		Line:    lineNumber,
		Code:    ParseCodeUnknownSymbol,
		Message: fmt.Sprintf("unknown winter order %s %s", symbol, subtype),
	}
}
