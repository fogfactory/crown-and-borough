package orders

import (
	"fmt"
	"strconv"
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
		order.Line = lineNumber + 1
		chain.Orders = append(chain.Orders, order)
	}
	if !headerSeen {
		errors = append(errors, parseMessage(0, ParseCodeNoHeader, "error.parse.no_header"))
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
		error := parseMessage(lineNumber, ParseCodeBadHeader, "error.parse.bad_header")
		return "", &error
	}
	nobleID, exists := indexes.noblesByCode[line]
	if !exists {
		error := parseMessage(lineNumber, ParseCodeNobleNotFound, "error.parse.noble_not_found", line)
		return "", &error
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
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeUnknownSymbol, "error.parse.order_symbol_missing")}
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
		error := parseMessage(lineNumber, ParseCodeUnclosedParenthesis, "error.parse.unclosed_parenthesis")
		return "", "", &error
	}
	return strings.TrimSpace(line[1 : len(line)-1]), models.LiaisonModeLoop, nil
}

func parsePrefixOrder(fields []string, lineNumber int, liaison models.LiaisonMode, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 1 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeMissingTarget, "error.parse.position_required", fields[0])}
	}
	if len(fields) > 2 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeTooManyTargets, "error.parse.position_only_one", fields[0])}
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
		Type:       orderType,
		PositionID: positionID,
		TargetIDs:  []models.TerritoryID{},
		Liaison:    liaison,
	}, nil
}

func parsePositionOrder(fields []string, lineNumber int, liaison models.LiaisonMode, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) < 2 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeUnknownSymbol, "error.parse.order_position_required")}
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
	case "M":
		return parseManeuver(fields, lineNumber, liaison, positionID, indexes)
	case "D":
		return parseDisperse(fields, lineNumber, liaison, positionID, indexes)
	case "T":
		return parseTransfer(fields, lineNumber, liaison, positionID, indexes)
	default:
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeUnknownSymbol, "error.parse.unsupported_order_symbol", fields[1])}
	}
}

func parseTransfer(fields []string, lineNumber int, liaison models.LiaisonMode, positionID models.TerritoryID, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeMissingTarget, "error.parse.destination_required", "T")}
	}
	if len(fields) != 4 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeTooManyTargets, "error.parse.transfer_shape")}
	}
	targetID, codeErrors := parseTerritory(fields[2], lineNumber, "transfer destination", indexes)
	if len(codeErrors) != 0 {
		return models.Order{}, codeErrors
	}
	amount, amountError := parsePositiveAmount(fields[3], lineNumber, "error.parse.transfer_amount")
	if amountError != nil {
		return models.Order{}, []ParseError{*amountError}
	}
	return models.Order{
		Type:       models.OrderTypeTransfer,
		PositionID: positionID,
		TargetIDs:  []models.TerritoryID{targetID},
		Amount:     amount,
		Liaison:    liaison,
	}, nil
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
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeMissingTarget, "error.parse.destination_required", fields[1])}
	}
	if len(fields) > 3 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeTooManyTargets, "error.parse.destination_only_one", fields[1])}
	}
	targetID, codeErrors := parseTerritory(fields[2], lineNumber, "target", indexes)
	if len(codeErrors) != 0 {
		return models.Order{}, codeErrors
	}
	return models.Order{
		Type:       orderType,
		PositionID: positionID,
		TargetIDs:  []models.TerritoryID{targetID},
		Liaison:    liaison,
	}, nil
}

func parseSupport(fields []string, lineNumber int, liaison models.LiaisonMode, positionID models.TerritoryID, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeMissingTarget, "error.parse.support_position_required")}
	}
	if len(fields) == 4 && fields[3] == "-" {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeMissingTarget, "error.parse.offensive_support_destination")}
	}
	if len(fields) != 3 && len(fields) != 5 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeTooManyTargets, "error.parse.support_shape")}
	}
	if len(fields) == 5 && fields[3] != "-" {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeUnknownSymbol, "error.parse.offensive_support_dash")}
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
		Type:       models.OrderTypeSupport,
		PositionID: positionID,
		TargetIDs:  targets,
		Liaison:    liaison,
	}, nil
}

func parseDisperse(fields []string, lineNumber int, liaison models.LiaisonMode, positionID models.TerritoryID, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeMissingTarget, "error.parse.disperse_destination_required")}
	}

	order := models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       positionID,
		TargetIDs:        make([]models.TerritoryID, 0, len(fields)-2),
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{},
		Liaison:          liaison,
	}
	errors := []ParseError{}
	for _, token := range fields[2:] {
		territoryID, assignedCodes, tokenErrors := parseDisperseTarget(token, lineNumber, indexes)
		if len(tokenErrors) != 0 {
			errors = append(errors, tokenErrors...)
			continue
		}
		order.TargetIDs = append(order.TargetIDs, territoryID)
		if len(assignedCodes) != 0 {
			order.NobleAssignments[territoryID] = append(order.NobleAssignments[territoryID], assignedCodes...)
		}
	}
	if len(errors) != 0 {
		return models.Order{}, errors
	}
	return order, nil
}

// parseManeuver parses "XXX M DEST [DEST ...]" where DEST is
// TER[/N|+][*NOB...|*]. A bare TER means TER/1.
func parseManeuver(fields []string, lineNumber int, liaison models.LiaisonMode, positionID models.TerritoryID, indexes gameIndexes) (models.Order, []ParseError) {
	if len(fields) == 2 {
		return models.Order{}, []ParseError{parseMessage(lineNumber, ParseCodeMissingTarget, "error.parse.maneuver_destination_required")}
	}
	order := models.Order{
		Type:             models.OrderTypeManeuver,
		PositionID:       positionID,
		TargetIDs:        make([]models.TerritoryID, 0, len(fields)-2),
		Counts:           make([]int, 0, len(fields)-2),
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{},
		Liaison:          liaison,
	}
	errors := []ParseError{}
	for _, token := range fields[2:] {
		head, nobles := token, ""
		if star := strings.IndexByte(token, '*'); star >= 0 {
			head, nobles = token[:star], token[star:]
		}
		code, count, countOK := splitManeuverCount(head)
		if !countOK {
			errors = append(errors, parseMessage(lineNumber, ParseCodeInvalidAmount, "error.parse.maneuver_count", token))
			continue
		}
		territoryID, territoryErrors := parseTerritory(code, lineNumber, "maneuver destination", indexes)
		if len(territoryErrors) != 0 {
			errors = append(errors, territoryErrors...)
			continue
		}
		assignedCodes, nobleErrors := parseAssignedNobles(strings.Split(nobles, "*")[1:], lineNumber, indexes)
		if len(nobleErrors) != 0 {
			errors = append(errors, nobleErrors...)
			continue
		}
		order.TargetIDs = append(order.TargetIDs, territoryID)
		order.Counts = append(order.Counts, count)
		if len(assignedCodes) != 0 {
			order.NobleAssignments[territoryID] = append(order.NobleAssignments[territoryID], assignedCodes...)
		}
	}
	if len(errors) != 0 {
		return models.Order{}, errors
	}
	return order, nil
}

func parseDisperseTarget(token string, lineNumber int, indexes gameIndexes) (models.TerritoryID, []models.NobleCode, []ParseError) {
	parts := strings.Split(token, "*")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil, []ParseError{parseMessage(lineNumber, ParseCodeInvalidCode, "error.parse.assignment_destination")}
	}
	territoryID, territoryErrors := parseTerritory(parts[0], lineNumber, "dispersion destination", indexes)
	if len(territoryErrors) != 0 {
		return "", nil, territoryErrors
	}
	if len(parts) == 1 {
		return territoryID, nil, nil
	}

	assignedCodes, nobleErrors := parseAssignedNobles(parts[1:], lineNumber, indexes)
	if len(nobleErrors) != 0 {
		return "", nil, nobleErrors
	}
	return territoryID, assignedCodes, nil
}

// parseAssignedNobles parses the noble codes after the first "*" of a
// destination: a lone "*" is the wildcard for every remaining noble.
func parseAssignedNobles(parts []string, lineNumber int, indexes gameIndexes) ([]models.NobleCode, []ParseError) {
	assignedCodes := make([]models.NobleCode, 0, len(parts))
	for index, nobleCode := range parts {
		if nobleCode == "" {
			if len(parts) == 1 && index == 0 {
				assignedCodes = append(assignedCodes, models.NobleCode("*"))
				continue
			}
			return nil, []ParseError{parseMessage(lineNumber, ParseCodeInvalidCode, "error.parse.assignment_empty_noble")}
		}
		if !isCode(nobleCode) {
			return nil, []ParseError{parseMessage(lineNumber, ParseCodeInvalidCode, "error.parse.assignment_invalid_noble", nobleCode)}
		}
		if _, exists := indexes.noblesByCode[nobleCode]; !exists {
			return nil, []ParseError{parseMessage(lineNumber, ParseCodeInvalidCode, "error.parse.assignment_unknown_noble", nobleCode)}
		}
		assignedCodes = append(assignedCodes, models.NobleCode(nobleCode))
	}
	return assignedCodes, nil
}

// splitManeuverCount splits "TER", "TER/N" or "TER+" into the territory code
// and its troop count (models.ManeuverRest for "+"). N is a plain positive
// number without sign or leading zero.
func splitManeuverCount(head string) (string, int, bool) {
	if code, isRest := strings.CutSuffix(head, "+"); isRest {
		return code, models.ManeuverRest, !strings.ContainsAny(code, "+/")
	}
	code, digits, hasCount := strings.Cut(head, "/")
	if !hasCount {
		return head, 1, !strings.Contains(head, "+")
	}
	if digits == "" || digits[0] == '0' || strings.Trim(digits, "0123456789") != "" {
		return "", 0, false
	}
	count, err := strconv.Atoi(digits)
	return code, count, err == nil && !strings.ContainsAny(code, "+/")
}

func parseTerritory(code string, lineNumber int, role string, indexes gameIndexes) (models.TerritoryID, []ParseError) {
	if !isCode(code) {
		return "", []ParseError{parseMessage(lineNumber, ParseCodeInvalidCode, "error.parse.code_format", role, code)}
	}
	id := models.TerritoryID(code)
	if indexes.territoriesByID[id] == nil {
		return "", []ParseError{parseMessage(lineNumber, ParseCodeInvalidCode, "error.parse.territory_unknown", code)}
	}
	return id, nil
}

func parsePositiveAmount(value string, lineNumber int, key string) (int, *ParseError) {
	amount, err := strconv.Atoi(value)
	if err != nil || amount < 1 {
		error := parseMessage(lineNumber, ParseCodeInvalidAmount, key, value)
		return 0, &error
	}
	return amount, nil
}
