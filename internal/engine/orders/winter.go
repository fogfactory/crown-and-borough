package orders

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/i18n"
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

// ParseWinterOrdersWithDeckOrders parses one winter sheet while keeping card
// discards in the same submission as the other winter orders.
func ParseWinterOrdersWithDeckOrders(text string, game *models.GameState) ([]models.WinterOrder, []models.DeckOrder, []ParseError) {
	indexes := indexGame(game)
	winterOrders := []models.WinterOrder{}
	deckOrders := []models.DeckOrder{}
	parseErrors := []ParseError{}
	for lineNumber, sourceLine := range strings.Split(text, "\n") {
		line := normalizeLine(sourceLine)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if isSpecialDiscardLine(fields) {
			order, parseError := parseDeckOrderLine(line, lineNumber+1, game)
			if parseError != nil {
				parseErrors = append(parseErrors, *parseError)
				continue
			}
			order.ID = models.OrderID(fmt.Sprintf("O%d", len(winterOrders)+len(deckOrders)+1))
			deckOrders = append(deckOrders, order)
			continue
		}
		order, parseError := parseWinterOrderLine(line, lineNumber+1, indexes)
		if parseError != nil {
			parseErrors = append(parseErrors, *parseError)
			continue
		}
		order.ID = models.OrderID(fmt.Sprintf("O%d", len(winterOrders)+len(deckOrders)+1))
		winterOrders = append(winterOrders, order)
	}
	if len(parseErrors) != 0 {
		return nil, nil, parseErrors
	}
	return winterOrders, deckOrders, nil
}

// WinterSheetLine is one non-empty line of a winter sheet parsed on its own:
// exactly one of Winter, Deck or Error is set.
type WinterSheetLine struct {
	Line   int
	Winter *models.WinterOrder
	Deck   *models.DeckOrder
	Error  *ParseError
}

// ParseWinterSheetLines parses every line of a winter sheet independently, so
// a draft preview can report valid lines next to malformed ones. Order IDs
// are left empty for the caller to assign.
func ParseWinterSheetLines(text string, game *models.GameState) []WinterSheetLine {
	indexes := indexGame(game)
	lines := []WinterSheetLine{}
	for lineNumber, sourceLine := range strings.Split(text, "\n") {
		line := normalizeLine(sourceLine)
		if line == "" {
			continue
		}
		parsed := WinterSheetLine{Line: lineNumber + 1}
		fields := strings.Fields(line)
		if isSpecialDiscardLine(fields) {
			order, parseError := parseDeckOrderLine(line, lineNumber+1, game)
			if parseError != nil {
				parsed.Error = parseError
			} else {
				parsed.Deck = &order
			}
		} else {
			order, parseError := parseWinterOrderLine(line, lineNumber+1, indexes)
			if parseError != nil {
				parsed.Error = parseError
			} else {
				parsed.Winter = &order
			}
		}
		lines = append(lines, parsed)
	}
	return lines
}

// isSpecialDiscardLine tells a special-card discard (D C KIND, a two-letter
// kind such as BT or TX) from a noble-hand discard (D C CCC, a three-letter
// noble trigram or dignity code): the code length decides, and no special
// kind code has three letters, so the two never collide. A malformed D C line
// is reported by the special-card parser unless its code has three letters.
func isSpecialDiscardLine(fields []string) bool {
	if len(fields) < 2 || fields[0] != "D" || fields[1] != "C" {
		return false
	}
	return len(fields) != 3 || !isCode(fields[2])
}

func parseWinterOrderLine(line string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	fields := strings.Fields(line)
	if len(fields) >= 2 && fields[0] == "T" && fields[1] == "N" {
		if len(fields) != 2 {
			error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterNobleDrawShape)
			return models.WinterOrder{}, &error
		}
		return models.WinterOrder{Type: models.WinterOrderTypeDrawNoble}, nil
	}
	if len(fields) >= 2 && fields[0] == "D" && fields[1] == "C" && len(fields) != 3 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterDiscardNobleShape)
		return models.WinterOrder{}, &error
	}
	if fields[0] == "Q" {
		return parseInquiryLine(fields, lineNumber, indexes)
	}
	if fields[0] == "J" {
		return parseTrialLine(fields, lineNumber, indexes)
	}
	if len(fields) < 3 {
		error := parseMessage(lineNumber, ParseCodeMissingTarget, "error.winter.order_shape")
		return models.WinterOrder{}, &error
	}
	if fields[0] == "G" {
		if len(fields) != 4 {
			error := parseMessage(lineNumber, ParseCodeTooManyTargets, "error.winter.transfer_shape")
			return models.WinterOrder{}, &error
		}
		sourceID, sourceError := winterTerritoryID(fields[1], lineNumber, indexes)
		if sourceError != nil {
			return models.WinterOrder{}, sourceError
		}
		targetID, targetError := winterTerritoryID(fields[2], lineNumber, indexes)
		if targetError != nil {
			return models.WinterOrder{}, targetError
		}
		amount, amountError := parsePositiveAmount(fields[3], lineNumber, "error.winter.transfer_amount")
		if amountError != nil {
			return models.WinterOrder{}, amountError
		}
		return models.WinterOrder{
			Type:     models.WinterOrderTypeTransfer,
			SourceID: sourceID,
			TargetID: targetID,
			Amount:   amount,
		}, nil
	}
	if fields[0] == "T" {
		return parseFiefOrderLine(fields, lineNumber, indexes)
	}
	if fields[0] == "H" {
		return parseTransferNobleLine(fields, lineNumber, indexes)
	}
	if fields[0] == "M" {
		return parseMarriageOrderLine(fields, lineNumber, indexes)
	}
	if fields[0] == "R" && fields[1] == "N" {
		order, parseError := parseRecruitNobleLine(fields, lineNumber, indexes)
		if parseError == nil {
			// The recruited noble can be targeted by the later orders of the
			// same sheet (a dignity, a claim): its code is its card code.
			if _, exists := indexes.noblesByCode[fields[2]]; !exists {
				indexes.noblesByCode[fields[2]] = ""
			}
		}
		return order, parseError
	}
	if fields[0] == "D" && fields[1] == "C" {
		if !isCode(fields[2]) {
			error := parseMessage(lineNumber, ParseCodeInvalidCode, "error.winter.noble_code_format", fields[2])
			return models.WinterOrder{}, &error
		}
		return models.WinterOrder{Type: models.WinterOrderTypeDiscardNoble, CardCode: fields[2]}, nil
	}
	if fields[0] == "D" && fields[1] == "N" {
		return parseDignityOrderLine(fields, lineNumber, indexes)
	}
	if fields[0] == "C" && fields[1] == "N" {
		return parseClaimOrderLine(fields, lineNumber, indexes)
	}
	if fields[0] == "V" && fields[1] == "C" {
		return parseCalamityVetoLine(fields, lineNumber, indexes)
	}
	if fields[0] == "S" {
		return parseRitualLine(fields, lineNumber, indexes)
	}
	if fields[0] == "X" {
		return parseExcommunicationLine(fields, lineNumber, indexes)
	}
	if fields[0] == "N" {
		return parseBuyCardinalLine(fields, lineNumber, indexes)
	}
	if fields[0] == "K" || fields[0] == "V" {
		return parseElectionOrderLine(fields, lineNumber, indexes)
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

// parseFiefOrderLine handles the two T subtypes: F (found, T F NNN XXX YYY
// ZZZ …, the titleholder noble then the group with the capital first) and A
// (assign, T A NNN XXX, a free noble then the vacant fief's capital). It only
// checks syntax: group size, duplicate territories, and contiguity are
// explicit engine rejects (see winter_found_fief_order.go).
func parseFiefOrderLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	switch fields[1] {
	case "F":
		if len(fields) < 3+models.FiefMinTerritories {
			error := parseMessage(lineNumber, ParseCodeMissingTarget, i18n.WinterFiefFoundShape)
			return models.WinterOrder{}, &error
		}
		if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
		territoryIDs := make([]models.TerritoryID, 0, len(fields)-3)
		for _, code := range fields[3:] {
			territoryID, parseError := winterTerritoryID(code, lineNumber, indexes)
			if parseError != nil {
				return models.WinterOrder{}, parseError
			}
			territoryIDs = append(territoryIDs, territoryID)
		}
		return models.WinterOrder{
			Type:         models.WinterOrderTypeFoundFief,
			NobleCode:    models.NobleCode(fields[2]),
			TerritoryID:  territoryIDs[0],
			TerritoryIDs: territoryIDs,
		}, nil
	case "A":
		if len(fields) != 4 {
			error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterFiefAssignShape)
			return models.WinterOrder{}, &error
		}
		if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
		territoryID, parseError := winterTerritoryID(fields[3], lineNumber, indexes)
		if parseError != nil {
			return models.WinterOrder{}, parseError
		}
		return models.WinterOrder{
			Type:        models.WinterOrderTypeAssignFief,
			NobleCode:   models.NobleCode(fields[2]),
			TerritoryID: territoryID,
		}, nil
	default:
		error := parseMessage(lineNumber, ParseCodeUnknownSymbol, i18n.WinterFiefShape)
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

// parseMarriageOrderLine handles M N XXX YYY: XXX is the player's own noble,
// YYY the other player's noble it asks to marry. Ownership, sex and the other
// conditions are engine rejects (see winter_marriage_order.go).
// parseTransferNobleLine handles H N NNN XXX [O|P]: NNN is a noble under the
// player's control, XXX the territory of the army it is handed to, O or P an optional new
// status (hostage or dungeon). Whether the
// noble is controlled and XXX holds a recipient army are engine rejects.
func parseTransferNobleLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if fields[1] != "N" {
		return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
	}
	if len(fields) != 4 && len(fields) != 5 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterTransferNobleShape)
		return models.WinterOrder{}, &error
	}
	if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
		return models.WinterOrder{}, parseError
	}
	territoryID, parseError := winterTerritoryID(fields[3], lineNumber, indexes)
	if parseError != nil {
		return models.WinterOrder{}, parseError
	}
	var status models.NobleStatus
	if len(fields) == 5 {
		switch fields[4] {
		case "O":
			status = models.NobleStatusHostage
		case "P":
			status = models.NobleStatusDungeon
		default:
			error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterTransferNobleShape)
			return models.WinterOrder{}, &error
		}
	}
	return models.WinterOrder{
		Type:        models.WinterOrderTypeTransferNoble,
		NobleCode:   models.NobleCode(fields[2]),
		TerritoryID: territoryID,
		Status:      status,
	}, nil
}

func parseMarriageOrderLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if fields[1] != "N" {
		return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
	}
	if len(fields) != 4 || fields[2] == fields[3] {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterMarriageShape)
		return models.WinterOrder{}, &error
	}
	for _, code := range fields[2:] {
		if parseError := winterNobleCode(code, lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
	}
	return models.WinterOrder{
		Type:       models.WinterOrderTypeMarriage,
		NobleCode:  models.NobleCode(fields[2]),
		SpouseCode: models.NobleCode(fields[3]),
	}, nil
}

// parseRecruitNobleLine handles R N XXX YYY: XXX is the code of a noble card
// played from the player's hand, YYY the castle or village where the noble
// appears. Whether the card is in hand is an engine reject (the card codes of
// the deck are not part of the public state).
func parseRecruitNobleLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if len(fields) != 4 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterRecruitNobleShape)
		return models.WinterOrder{}, &error
	}
	if !isCode(fields[2]) {
		error := parseMessage(lineNumber, ParseCodeInvalidCode, "error.winter.noble_code_format", fields[2])
		return models.WinterOrder{}, &error
	}
	territoryID, parseError := winterTerritoryID(fields[3], lineNumber, indexes)
	if parseError != nil {
		return models.WinterOrder{}, parseError
	}
	return models.WinterOrder{
		Type:        models.WinterOrderTypeRecruitNoble,
		CardCode:    fields[2],
		TerritoryID: territoryID,
	}, nil
}

// parseDignityOrderLine handles D N XXX CCC: XXX is one of the player's
// nobles, CCC the code of a dignity card played from the player's hand.
// Ownership and the card being in hand are engine rejects.
func parseDignityOrderLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if len(fields) != 4 && len(fields) != 5 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterDignityShape)
		return models.WinterOrder{}, &error
	}
	if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
		return models.WinterOrder{}, parseError
	}
	if !isCode(fields[3]) {
		error := parseMessage(lineNumber, ParseCodeInvalidCode, "error.winter.noble_code_format", fields[3])
		return models.WinterOrder{}, &error
	}
	order := models.WinterOrder{
		Type:      models.WinterOrderTypeDignity,
		NobleCode: models.NobleCode(fields[2]),
		CardCode:  fields[3],
	}
	if len(fields) == 5 {
		territoryID, parseError := winterTerritoryID(fields[4], lineNumber, indexes)
		if parseError != nil {
			return models.WinterOrder{}, parseError
		}
		order.TerritoryID = territoryID
	}
	return order, nil
}

// parseCalamityVetoLine handles V C XXX I: XXX is the astrologer noble, I the
// position (1 to 3) of the forecast calamity to strike. Ownership and the
// dignity are engine rejects.
func parseCalamityVetoLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if len(fields) != 4 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterCalamityVetoShape)
		return models.WinterOrder{}, &error
	}
	if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
		return models.WinterOrder{}, parseError
	}
	index, err := strconv.Atoi(fields[3])
	if err != nil || index < 1 || index > 3 {
		error := parseMessage(lineNumber, ParseCodeInvalidCode, i18n.WinterCalamityVetoShape)
		return models.WinterOrder{}, &error
	}
	return models.WinterOrder{
		Type:      models.WinterOrderTypeCalamityVeto,
		NobleCode: models.NobleCode(fields[2]),
		Indices:   []int{index},
	}, nil
}

// parseClaimOrderLine handles C N HHH CCC: HHH is one of the player's nobles,
// the heir, and CCC the noble of another player whose titles it claims.
// Ownership, the marriage and the other conditions are engine rejects.
func parseClaimOrderLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if len(fields) != 4 || fields[2] == fields[3] {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterClaimShape)
		return models.WinterOrder{}, &error
	}
	for _, code := range fields[2:] {
		if parseError := winterNobleCode(code, lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
	}
	return models.WinterOrder{
		Type:       models.WinterOrderTypeClaim,
		NobleCode:  models.NobleCode(fields[2]),
		SpouseCode: models.NobleCode(fields[3]),
	}, nil
}

// parseElectionOrderLine handles K E NNN BBB / K P NNN (candidacy) and
// V E NNN BBB / V P NNN (vote): NNN is the candidate, BBB the seed village of
// the bishopric. Whether the election is open, who owns the noble and whether
// it may run are engine rejects (see election.go).
func parseElectionOrderLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	orderType, shape := models.WinterOrderTypeCandidacy, i18n.WinterCandidacyShape
	if fields[0] == "V" {
		orderType, shape = models.WinterOrderTypeVote, i18n.WinterVoteShape
	}
	var election models.ElectionKind
	var wantFields int
	switch fields[1] {
	case "E":
		election, wantFields = models.ElectionBishop, 4
	case "P":
		election, wantFields = models.ElectionPope, 3
	default:
		return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
	}
	if len(fields) != wantFields {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, shape)
		return models.WinterOrder{}, &error
	}
	if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
		return models.WinterOrder{}, parseError
	}
	order := models.WinterOrder{Type: orderType, Election: election, NobleCode: models.NobleCode(fields[2])}
	if election == models.ElectionBishop {
		seat, parseError := winterTerritoryID(fields[3], lineNumber, indexes)
		if parseError != nil {
			return models.WinterOrder{}, parseError
		}
		order.TerritoryID = seat
	}
	return order, nil
}

// parseBuyCardinalLine handles N C NNN: NNN is one of the player's bishops,
// promoted to cardinal at the investiture. Ownership, the title, the cap and
// the cost are engine rejects.
func parseBuyCardinalLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if fields[1] != "C" {
		return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
	}
	if len(fields) != 3 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterBuyCardinalShape)
		return models.WinterOrder{}, &error
	}
	if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
		return models.WinterOrder{}, parseError
	}
	return models.WinterOrder{Type: models.WinterOrderTypeBuyCardinal, NobleCode: models.NobleCode(fields[2])}, nil
}

// parseInquiryLine handles Q HHH NNN: HHH is the investigating cardinal or
// pope, NNN the investigated noble. Ownership and the title are engine rejects.
func parseInquiryLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if len(fields) != 3 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterInquiryShape)
		return models.WinterOrder{}, &error
	}
	for _, code := range fields[1:] {
		if parseError := winterNobleCode(code, lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
	}
	return models.WinterOrder{Type: models.WinterOrderTypeInquiry, NobleCode: models.NobleCode(fields[1]), TargetCode: models.NobleCode(fields[2])}, nil
}

// parseTrialLine handles J HHH NNN: HHH is the cardinal backing the trial of
// NNN. Ownership and the title are engine rejects; a second cardinal must file
// the same trial (stage 3).
func parseTrialLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if len(fields) != 3 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterTrialShape)
		return models.WinterOrder{}, &error
	}
	for _, code := range fields[1:] {
		if parseError := winterNobleCode(code, lineNumber, indexes); parseError != nil {
			return models.WinterOrder{}, parseError
		}
	}
	return models.WinterOrder{Type: models.WinterOrderTypeTrial, NobleCode: models.NobleCode(fields[1]), TargetCode: models.NobleCode(fields[2])}, nil
}

// parseExcommunicationLine handles X E NNN (excommunicate), X L NNN (lift) and
// X D NNN (dissolve the marriage of NNN): NNN is the targeted noble. Whether the player holds the papacy and the
// limits of the pope are engine rejects (see winter_papal_order.go).
func parseExcommunicationLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	var orderType models.WinterOrderType
	switch fields[1] {
	case "E":
		orderType = models.WinterOrderTypeExcommunicate
	case "L":
		orderType = models.WinterOrderTypeLiftExcommunication
	case "D":
		orderType = models.WinterOrderTypeDissolveMarriage
	default:
		return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
	}
	if len(fields) != 3 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterExcommunicationShape)
		return models.WinterOrder{}, &error
	}
	if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
		return models.WinterOrder{}, parseError
	}
	return models.WinterOrder{Type: orderType, NobleCode: models.NobleCode(fields[2])}, nil
}

// parseRitualLine handles S R NNN CAL (call the calamity CAL: PE, MT or FA)
// and S R NNN N (fix the season N of next year: 1 spring, 2 summer, 3
// autumn). NNN is the Witch; ownership and the dignity are engine rejects.
func parseRitualLine(fields []string, lineNumber int, indexes gameIndexes) (models.WinterOrder, *ParseError) {
	if fields[1] != "R" {
		return models.WinterOrder{}, unknownWinterSubtype(lineNumber, fields[0], fields[1])
	}
	if len(fields) != 4 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.WinterRitualShape)
		return models.WinterOrder{}, &error
	}
	if parseError := winterNobleCode(fields[2], lineNumber, indexes); parseError != nil {
		return models.WinterOrder{}, parseError
	}
	order := models.WinterOrder{Type: models.WinterOrderTypeRitual, NobleCode: models.NobleCode(fields[2])}
	switch fields[3] {
	case "1":
		order.Season = models.SeasonSpring
	case "2":
		order.Season = models.SeasonSummer
	case "3":
		order.Season = models.SeasonAutumn
	default:
		kind, parseError := parseSpecialKind(fields[3], lineNumber)
		if parseError != nil || !kind.IsCalamity() {
			error := parseMessage(lineNumber, ParseCodeInvalidCode, i18n.WinterRitualShape)
			return models.WinterOrder{}, &error
		}
		order.Calamity = kind
	}
	return order, nil
}
