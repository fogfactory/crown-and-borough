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
		error := parseMessage(lineNumber, ParseCodeMissingTarget, "error.winter.order_shape")
		return models.WinterOrder{}, &error
	}
	if len(fields) > 3 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, "error.winter.target_only_one")
		return models.WinterOrder{}, &error
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
	case "O", "P":
		if fields[1] != "N" {
			return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
		}
		if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
		orderType := models.WinterOrderTypeHostage
		if fields[0] == "P" {
			orderType = models.WinterOrderTypeDungeon
		}
		return models.WinterOrder{Type: orderType, NobleCode: models.NobleCode(fields[2])}, nil
	default:
		error := parseMessage(lineNumber, ParseCodeUnknownSymbol, "error.winter.unknown_symbol", fields[0])
		return models.WinterOrder{}, &error
	}
}

func winterTerritoryID(code string, lineNumber int, indexes gameIndexes) (models.TerritoryID, *ParseError) {
	if !isCode(code) {
		error := parseMessage(lineNumber, ParseCodeInvalidCode, "error.winter.territory_code_format", code)
		return "", &error
	}
	territoryID := models.TerritoryID(code)
	if indexes.territoriesByID[territoryID] == nil {
		error := parseMessage(lineNumber, ParseCodeInvalidCode, "error.winter.territory_unknown", code)
		return "", &error
	}
	return territoryID, nil
}

func winterNobleCode(code string, lineNumber int, indexes gameIndexes) *ParseError {
	if !isCode(code) {
		error := parseMessage(lineNumber, ParseCodeInvalidCode, "error.winter.noble_code_format", code)
		return &error
	}
	if _, exists := indexes.noblesByCode[code]; !exists {
		error := parseMessage(lineNumber, ParseCodeInvalidCode, "error.winter.noble_unknown", code)
		return &error
	}
	return nil
}

func unknownWinterSubtype(lineNumber int, symbol, subtype string) *ParseError {
	error := parseMessage(lineNumber, ParseCodeUnknownSymbol, "error.winter.unknown_subtype", symbol, subtype)
	return &error
}
