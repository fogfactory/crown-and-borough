package orders

import (
	"fmt"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ParseChain parses order text without assigning it to an army. It keeps
// original line numbers, returns every syntactically valid order, and collects
// source errors so callers can report all faults in one submission.
func ParseChain(text string, game *models.GameState) (models.Chain, []ParseError) {
	chain := models.Chain{Orders: []models.Order{}, CurrentIndex: 0}
	indexes := indexGame(game)
	errors := []ParseError{}

	headerSeen := false
	for lineNumber, sourceLine := range strings.Split(text, "\n") {
		line := normalizeLine(sourceLine)
		if line == "" {
			continue
		}
		if !headerSeen {
			headerSeen = true
			if nobleID, parseError := parseHeader(line, lineNumber+1, indexes); parseError != nil {
				errors = append(errors, *parseError)
			} else {
				chain.NobleID = nobleID
			}
			continue
		}

		order, parseErrors := parseOrderLine(line, lineNumber+1, indexes)
		if len(parseErrors) != 0 {
			errors = append(errors, parseErrors...)
			continue
		}
		order.ID = models.OrderID(fmt.Sprintf("O%d", len(chain.Orders)+1))
		chain.Orders = append(chain.Orders, order)
	}
	if !headerSeen {
		errors = append(errors, ParseError{
			Line:    0,
			Code:    ParseCodeNoHeader,
			Message: "an order chain requires a noble header",
		})
	}
	return chain, errors
}

func normalizeLine(line string) string {
	if comment := strings.IndexByte(line, '#'); comment >= 0 {
		line = line[:comment]
	}
	return strings.ToUpper(strings.TrimSpace(line))
}

func parseHeader(line string, lineNumber int, indexes gameIndexes) (models.NobleID, *ParseError) {
	fields := strings.Fields(line)
	if len(fields) != 1 || !isCode(line) {
		return "", &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeBadHeader,
			Message: "the first content line must contain exactly one noble code",
		}
	}
	nobleID, exists := indexes.noblesByCode[line]
	if !exists {
		return "", &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeNobleNotFound,
			Message: fmt.Sprintf("noble code %q does not exist", line),
		}
	}
	return nobleID, nil
}

func parseOrderLine(line string, lineNumber int, indexes gameIndexes) (models.Order, []ParseError) {
	content, liaison, parenthesisError := parseLiaison(line, lineNumber)
	if parenthesisError != nil {
		return models.Order{}, []ParseError{*parenthesisError}
	}
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeUnknownSymbol,
			Message: "an order line must contain an order symbol",
		}}
	}

	if fields[0] == "H" || fields[0] == "P" {
		return parsePrefixOrder(fields, lineNumber, liaison, indexes)
	}
	return parsePositionOrder(fields, lineNumber, liaison, indexes)
}

func parseLiaison(line string, lineNumber int) (string, models.LiaisonMode, *ParseError) {
	if !strings.ContainsAny(line, "()") {
		return line, models.LiaisonModeSingle, nil
	}
	if !strings.HasPrefix(line, "(") || !strings.HasSuffix(line, ")") || strings.Count(line, "(") != 1 || strings.Count(line, ")") != 1 {
		return "", "", &ParseError{
			Line:    lineNumber,
			Code:    ParseCodeUnclosedParenthesis,
			Message: "parentheses must enclose one complete order line",
		}
	}
	return strings.TrimSpace(line[1 : len(line)-1]), models.LiaisonModeLoop, nil
}

func parsePrefixOrder(fields []string, lineNumber int, liaison models.LiaisonMode, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 1 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeMissingTarget,
			Message: fmt.Sprintf("%s requires a position code", fields[0]),
		}}
	}
	if len(fields) > 2 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeTooManyTargets,
			Message: fmt.Sprintf("%s accepts only one position code", fields[0]),
		}}
	}
	positionID, codeErrors := parseTerritory(fields[1], lineNumber, "position", indexes)
	if len(codeErrors) != 0 {
		return models.Order{}, codeErrors
	}
	orderType := models.OrderTypeHold
	if fields[0] == "P" {
		orderType = models.OrderTypePillage
	}
	return models.Order{
		Type:           orderType,
		PositionID:     positionID,
		TargetIDs:      []models.TerritoryID{},
		NobleTargetIDs: []models.NobleID{},
		Liaison:        liaison,
	}, nil
}

func parsePositionOrder(fields []string, lineNumber int, liaison models.LiaisonMode, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) < 2 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeUnknownSymbol,
			Message: "an order line requires a position code and order symbol",
		}}
	}
	positionID, codeErrors := parseTerritory(fields[0], lineNumber, "position", indexes)
	if len(codeErrors) != 0 {
		return models.Order{}, codeErrors
	}

	switch fields[1] {
	case "A":
		return parseSingleTerritoryTarget(models.OrderTypeAttack, fields, lineNumber, liaison, positionID, indexes)
	case "J":
		return parseSingleTerritoryTarget(models.OrderTypeJoin, fields, lineNumber, liaison, positionID, indexes)
	case "S":
		return parseSupport(fields, lineNumber, liaison, positionID, indexes)
	case "D":
		return parseDisperse(fields, lineNumber, liaison, positionID, indexes)
	case "O":
		return parseSingleNobleTarget(models.OrderTypeHostage, fields, lineNumber, liaison, positionID, indexes)
	case "K":
		return parseSingleNobleTarget(models.OrderTypeDungeon, fields, lineNumber, liaison, positionID, indexes)
	default:
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeUnknownSymbol,
			Message: fmt.Sprintf("%q is not a supported order symbol", fields[1]),
		}}
	}
}

func parseSingleTerritoryTarget(
	orderType models.OrderType,
	fields []string,
	lineNumber int,
	liaison models.LiaisonMode,
	positionID models.TerritoryID,
	indexes gameIndexes,
) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeMissingTarget,
			Message: fmt.Sprintf("%s requires one destination code", fields[1]),
		}}
	}
	if len(fields) > 3 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeTooManyTargets,
			Message: fmt.Sprintf("%s accepts exactly one destination code", fields[1]),
		}}
	}
	targetID, codeErrors := parseTerritory(fields[2], lineNumber, "target", indexes)
	if len(codeErrors) != 0 {
		return models.Order{}, codeErrors
	}
	return models.Order{
		Type:           orderType,
		PositionID:     positionID,
		TargetIDs:      []models.TerritoryID{targetID},
		NobleTargetIDs: []models.NobleID{},
		Liaison:        liaison,
	}, nil
}

func parseSupport(fields []string, lineNumber int, liaison models.LiaisonMode, positionID models.TerritoryID, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeMissingTarget,
			Message: "S requires a supported army position",
		}}
	}
	if len(fields) == 4 && fields[3] == "-" {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeMissingTarget,
			Message: "offensive S requires an attack destination after -",
		}}
	}
	if len(fields) != 3 && len(fields) != 5 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeTooManyTargets,
			Message: "S accepts either one supported position or one position and attack destination",
		}}
	}
	if len(fields) == 5 && fields[3] != "-" {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeUnknownSymbol,
			Message: "offensive S requires - before its attack destination",
		}}
	}
	supportedID, codeErrors := parseTerritory(fields[2], lineNumber, "supported position", indexes)
	if len(codeErrors) != 0 {
		return models.Order{}, codeErrors
	}
	targets := []models.TerritoryID{supportedID}
	if len(fields) == 5 {
		destinationID, destinationErrors := parseTerritory(fields[4], lineNumber, "attack destination", indexes)
		if len(destinationErrors) != 0 {
			return models.Order{}, destinationErrors
		}
		targets = append(targets, destinationID)
	}
	return models.Order{
		Type:           models.OrderTypeSupport,
		PositionID:     positionID,
		TargetIDs:      targets,
		NobleTargetIDs: []models.NobleID{},
		Liaison:        liaison,
	}, nil
}

func parseSingleNobleTarget(
	orderType models.OrderType,
	fields []string,
	lineNumber int,
	liaison models.LiaisonMode,
	positionID models.TerritoryID,
	indexes gameIndexes,
) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeMissingTarget,
			Message: fmt.Sprintf("%s requires one noble code", fields[1]),
		}}
	}
	if len(fields) > 3 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeTooManyTargets,
			Message: fmt.Sprintf("%s accepts exactly one noble code", fields[1]),
		}}
	}
	nobleID, codeErrors := parseNoble(fields[2], lineNumber, "noble target", indexes)
	if len(codeErrors) != 0 {
		return models.Order{}, codeErrors
	}
	return models.Order{
		Type:           orderType,
		PositionID:     positionID,
		TargetIDs:      []models.TerritoryID{},
		NobleTargetIDs: []models.NobleID{nobleID},
		Liaison:        liaison,
	}, nil
}

func parseDisperse(fields []string, lineNumber int, liaison models.LiaisonMode, positionID models.TerritoryID, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeMissingTarget,
			Message: "D requires at least one destination code",
		}}
	}

	order := models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       positionID,
		TargetIDs:        make([]models.TerritoryID, 0, len(fields)-2),
		NobleTargetIDs:   []models.NobleID{},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{},
		Liaison:          liaison,
	}
	errors := []ParseError{}
	for _, token := range fields[2:] {
		territoryID, destinationCode, assignedCodes, tokenErrors := parseDisperseTarget(token, lineNumber, indexes)
		if len(tokenErrors) != 0 {
			errors = append(errors, tokenErrors...)
			continue
		}
		order.TargetIDs = append(order.TargetIDs, territoryID)
		if len(assignedCodes) != 0 {
			order.NobleAssignments[destinationCode] = append(order.NobleAssignments[destinationCode], assignedCodes...)
		}
	}
	if len(errors) != 0 {
		return models.Order{}, errors
	}
	return order, nil
}

func parseDisperseTarget(token string, lineNumber int, indexes gameIndexes) (models.TerritoryID, models.TerritoryCode, []models.NobleCode, []ParseError) {
	parts := strings.Split(token, "*")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", nil, []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: "a dispersion assignment requires a destination code before *",
		}}
	}
	territoryID, territoryErrors := parseTerritory(parts[0], lineNumber, "dispersion destination", indexes)
	if len(territoryErrors) != 0 {
		return "", "", nil, territoryErrors
	}
	if len(parts) == 1 {
		return territoryID, models.TerritoryCode(parts[0]), nil, nil
	}

	assignedCodes := make([]models.NobleCode, 0, len(parts)-1)
	for index, nobleCode := range parts[1:] {
		if nobleCode == "" {
			if len(parts) == 2 && index == 0 {
				assignedCodes = append(assignedCodes, models.NobleCode("*"))
				continue
			}
			return "", "", nil, []ParseError{{
				Line:    lineNumber,
				Code:    ParseCodeInvalidCode,
				Message: "a dispersion assignment contains an empty noble code",
			}}
		}
		if !isCode(nobleCode) {
			return "", "", nil, []ParseError{{
				Line:    lineNumber,
				Code:    ParseCodeInvalidCode,
				Message: fmt.Sprintf("invalid noble code %q in dispersion assignment", nobleCode),
			}}
		}
		if _, exists := indexes.noblesByCode[nobleCode]; !exists {
			return "", "", nil, []ParseError{{
				Line:    lineNumber,
				Code:    ParseCodeInvalidCode,
				Message: fmt.Sprintf("noble code %q does not exist", nobleCode),
			}}
		}
		assignedCodes = append(assignedCodes, models.NobleCode(nobleCode))
	}
	return territoryID, models.TerritoryCode(parts[0]), assignedCodes, nil
}

func parseTerritory(code string, lineNumber int, role string, indexes gameIndexes) (models.TerritoryID, []ParseError) {
	if !isCode(code) {
		return "", []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("%s code %q must contain exactly three uppercase letters", role, code),
		}}
	}
	id, exists := indexes.territoriesByCode[code]
	if !exists {
		return "", []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("territory code %q does not exist", code),
		}}
	}
	return id, nil
}

func parseNoble(code string, lineNumber int, role string, indexes gameIndexes) (models.NobleID, []ParseError) {
	if !isCode(code) {
		return "", []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("%s code %q must contain exactly three uppercase letters", role, code),
		}}
	}
	id, exists := indexes.noblesByCode[code]
	if !exists {
		return "", []ParseError{{
			Line:    lineNumber,
			Code:    ParseCodeInvalidCode,
			Message: fmt.Sprintf("noble code %q does not exist", code),
		}}
	}
	return id, nil
}
