package orders

import (
	"fmt"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func ParseDeckOrders(text string, game *models.GameState) ([]models.DeckOrder, []ParseError) {
	parsed := []models.DeckOrder{}
	parseErrors := []ParseError{}
	for lineNumber, sourceLine := range strings.Split(text, "\n") {
		line := normalizeLine(sourceLine)
		if line == "" {
			continue
		}
		order, parseError := parseDeckOrderLine(line, lineNumber+1, game)
		if parseError != nil {
			parseErrors = append(parseErrors, *parseError)
			continue
		}
		order.ID = models.OrderID(fmt.Sprintf("O%d", len(parsed)+1))
		parsed = append(parsed, order)
	}
	if len(parseErrors) != 0 {
		return nil, parseErrors
	}
	return parsed, nil
}

func parseDeckOrderLine(line string, lineNumber int, game *models.GameState) (models.DeckOrder, *ParseError) {
	fields := strings.Fields(line)
	if order, parseError, handled := parseAppeasementLine(fields, lineNumber, game); handled {
		return order, parseError
	}
	if len(fields) < 3 {
		error := parseMessage(lineNumber, ParseCodeMissingTarget, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error
	}
	if len(fields) == 3 && isTitheCode(fields[1]) {
		error := parseMessage(lineNumber, ParseCodeMissingTarget, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error
	}
	if len(fields) > 4 || (len(fields) == 4 && !isTaxCode(fields[1]) && !isTitheCode(fields[1])) {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error
	}
	if len(fields) == 4 {
		return parseIssuedTaxLine(fields, lineNumber, game)
	}
	if fields[0] == "D" && fields[1] == "C" {
		kind, parseError := parseSpecialKind(fields[2], lineNumber)
		if parseError != nil {
			return models.DeckOrder{}, parseError
		}
		if !kind.IsBonus() {
			error := parseMessage(lineNumber, ParseCodeSpecialKind, i18n.DeckOrderKindNotPlayable, fields[2])
			return models.DeckOrder{}, &error
		}
		return models.DeckOrder{Type: models.DeckOrderTypeDiscard, Kind: kind}, nil
	}
	if fields[0] != "P" {
		error := parseMessage(lineNumber, ParseCodeUnknownSymbol, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error
	}
	kind, parseError := parseSpecialKind(fields[1], lineNumber)
	if parseError != nil {
		return models.DeckOrder{}, parseError
	}
	if !kind.IsBonus() {
		error := parseMessage(lineNumber, ParseCodeSpecialKind, i18n.DeckOrderKindNotPlayable, fields[1])
		return models.DeckOrder{}, &error
	}
	target := models.TerritoryID(fields[2])
	if kind == models.CardKindRevolt {
		if !isTerritory(game, target) {
			error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderRegionUnknown, fields[2])
			return models.DeckOrder{}, &error
		}
		return models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: kind, TargetTerritoryID: target}, nil
	}
	if kind == models.CardKindSeigneurialTax {
		// TER is a fief capital here, by exception to the usual "TER is a
		// region's seed village" rule (titres.md "Taxe seigneuriale"):
		// ownership of the targeted fief is checked later by CanPlay.
		if !isFiefCapital(game, target) {
			error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderRegionUnknown, fields[2])
			return models.DeckOrder{}, &error
		}
		return models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: kind, TargetTerritoryID: target}, nil
	}
	if kind == models.CardKindTrial {
		// HHH is a noble code here, by exception to the usual "TER is a
		// region's seed village" rule (specs/dames.md § Carte de procès).
		nobleID, found := nobleIDByCode(game, fields[2])
		if !found {
			error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderNobleUnknown, fields[2])
			return models.DeckOrder{}, &error
		}
		return models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: kind, TargetNobleID: nobleID}, nil
	}
	if !isSpecialRegionSeed(game, target) {
		error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderRegionUnknown, fields[2])
		return models.DeckOrder{}, &error
	}
	return models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: kind, RegionSeed: target}, nil
}

func isTaxCode(code string) bool { return code == "TX" || code == "ST" }

// isTitheCode reports the explicit tithe code ("P DI HHH XXX"), which keeps a
// tithe distinct from a seigneurial tax when the target is both a fief capital
// and a bishopric seed.
func isTitheCode(code string) bool { return code == "DI" || code == "TT" }

// parseIssuedTaxLine parses "P TX HHH XXX": HHH is the issuing noble and XXX
// the target, a fief capital (seigneurial tax) or a bishopric's seed village
// (tithe, religieux.md "Dîme"). Whether the issuer may tax the target is
// checked later by CanPlay.
func parseIssuedTaxLine(fields []string, lineNumber int, game *models.GameState) (models.DeckOrder, *ParseError) {
	if fields[0] != "P" {
		error := parseMessage(lineNumber, ParseCodeUnknownSymbol, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error
	}
	nobleID, found := nobleIDByCode(game, fields[2])
	if !found {
		error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderNobleUnknown, fields[2])
		return models.DeckOrder{}, &error
	}
	target := models.TerritoryID(fields[3])
	tithe := isTitheCode(fields[1])
	known := isFiefCapital(game, target) || isSpecialRegionSeed(game, target)
	if tithe {
		known = isSpecialRegionSeed(game, target)
	}
	if !known {
		error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderRegionUnknown, fields[3])
		return models.DeckOrder{}, &error
	}
	// "P TX" on a bishopric seed that is no fief capital can only be a tithe.
	tithe = tithe || !isFiefCapital(game, target)
	return models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: nobleID, TargetTerritoryID: target, Tithe: tithe}, nil
}

func isTerritory(game *models.GameState, territoryID models.TerritoryID) bool {
	if game == nil {
		return false
	}
	for _, territory := range game.Territories {
		if territory.ID == territoryID {
			return true
		}
	}
	return false
}

func nobleIDByCode(game *models.GameState, code string) (models.NobleID, bool) {
	if game == nil {
		return "", false
	}
	for _, noble := range game.Nobles {
		if noble.Code == code {
			return noble.ID, true
		}
	}
	return "", false
}

func isFiefCapital(game *models.GameState, territoryID models.TerritoryID) bool {
	if game == nil {
		return false
	}
	for _, fief := range game.Fiefs {
		if fief.CapitalTerritoryID == territoryID {
			return true
		}
	}
	return false
}

func parseSpecialKind(value string, lineNumber int) (models.CardKind, *ParseError) {
	kind, exists := map[string]models.CardKind{
		"BT": models.CardKindFairWeather,
		"FW": models.CardKindFairWeather,
		"RA": models.CardKindAbundantHarvest,
		"AH": models.CardKindAbundantHarvest,
		"PE": models.CardKindPlague,
		"PL": models.CardKindPlague,
		"MT": models.CardKindBadWeather,
		"BW": models.CardKindBadWeather,
		"RE": models.CardKindRevolt,
		"RV": models.CardKindRevolt,
		"FA": models.CardKindFamine,
		"FN": models.CardKindFamine,
		"TX": models.CardKindSeigneurialTax,
		"ST": models.CardKindSeigneurialTax,
		"DI": models.CardKindSeigneurialTax,
		"TT": models.CardKindSeigneurialTax,
		"PR": models.CardKindTrial,
		"TR": models.CardKindTrial,
	}[value]
	if !exists {
		error := parseMessage(lineNumber, ParseCodeSpecialKind, i18n.DeckOrderKindUnknown, value)
		return "", &error
	}
	return kind, nil
}

func isSpecialRegionSeed(game *models.GameState, seed models.TerritoryID) bool {
	if game == nil {
		return false
	}
	for _, region := range game.Regions {
		if region.Seed == seed {
			return true
		}
	}
	return false
}

// parseAppeasementLine reads "P AG HHH TER" (free rite) and "P AP HHH TER"
// (paid appeasement): HHH is the cleric's noble code and TER a territory (EN
// aliases FQ and PQ). handled is false for any other line.
func parseAppeasementLine(fields []string, lineNumber int, game *models.GameState) (models.DeckOrder, *ParseError, bool) {
	if len(fields) < 2 || fields[0] != "P" {
		return models.DeckOrder{}, nil, false
	}
	var orderType models.DeckOrderType
	switch fields[1] {
	case "AG", "FQ":
		orderType = models.DeckOrderTypeAppeaseRite
	case "AP", "PQ":
		orderType = models.DeckOrderTypeAppeasePaid
	default:
		return models.DeckOrder{}, nil, false
	}
	if len(fields) < 4 {
		error := parseMessage(lineNumber, ParseCodeMissingTarget, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error, true
	}
	if len(fields) > 4 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error, true
	}
	nobleID, found := nobleIDByCode(game, fields[2])
	if !found {
		error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderNobleUnknown, fields[2])
		return models.DeckOrder{}, &error, true
	}
	target := models.TerritoryID(fields[3])
	if !isTerritory(game, target) {
		error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderRegionUnknown, fields[3])
		return models.DeckOrder{}, &error, true
	}
	return models.DeckOrder{Type: orderType, TargetNobleID: nobleID, TargetTerritoryID: target}, nil, true
}
