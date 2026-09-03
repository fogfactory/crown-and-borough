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
	if len(fields) == 2 && fields[0] == "T" && fields[1] == "C" {
		return models.DeckOrder{Type: models.DeckOrderTypeDraw}, nil
	}
	if len(fields) < 3 {
		error := parseMessage(lineNumber, ParseCodeMissingTarget, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error
	}
	if len(fields) > 3 {
		error := parseMessage(lineNumber, ParseCodeTooManyTargets, i18n.DeckOrderShape)
		return models.DeckOrder{}, &error
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
	if fields[0] != "P" && fields[0] != "J" {
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
	regionSeed := models.TerritoryID(fields[2])
	if !isSpecialRegionSeed(game, regionSeed) {
		error := parseMessage(lineNumber, ParseCodeSpecialRegion, i18n.DeckOrderRegionUnknown, fields[2])
		return models.DeckOrder{}, &error
	}
	return models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: kind, RegionSeed: regionSeed}, nil
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
