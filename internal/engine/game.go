package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	minimumGamePlayers  = 2
	maximumGamePlayers  = 16
	mapReferencePlayers = 4
	mapReferenceWidth   = 1000
	mapReferenceHeight  = 700
)

var defaultPlayerColors = [...]string{
	"#a84632",
	"#2d5f9e",
	"#7052a1",
	"#34775c",
	"#ad7a25",
	"#b3546e",
	"#1f7a8c",
	"#7a6b2d",
	"#c05621",
	"#4262c0",
	"#8f3b8f",
	"#5c8a3a",
	"#96663d",
	"#3d8fae",
	"#a64d79",
	"#6e7f9e",
}

// PlayerInit describes one player at game creation. IDs are optional; when an
// ID is omitted, CreateGame allocates P1, P2, and so on in input order.
type PlayerInit struct {
	ID    models.PlayerID `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Color string          `json:"color,omitempty"`
}

// ChainSubmission is one complete chain submitted by a player. Text contains
// the noble header followed by the order lines, exactly as described in the
// text command format.
type ChainSubmission struct {
	Player models.PlayerID  `json:"player"`
	Noble  models.NobleCode `json:"noble"`
	Text   string           `json:"text"`
}

// WinterSubmission contains the direct winter orders for one player. Lines is
// a newline-separated text block without a noble header.
type WinterSubmission struct {
	Player models.PlayerID `json:"player"`
	Lines  string          `json:"lines"`
}

// OrdersInput is one complete hotseat turn. All players submit at once.
type OrdersInput struct {
	Chains []ChainSubmission  `json:"chains"`
	Winter []WinterSubmission `json:"winter"`
}

// InputError identifies a client-side submission error. Line is the source
// line in the submitted chain or winter text, and is zero for batch errors.
type InputError struct {
	Player      models.PlayerID  `json:"player,omitempty"`
	Noble       models.NobleCode `json:"noble,omitempty"`
	Line        int              `json:"line,omitempty"`
	Code        string           `json:"code"`
	Message     string           `json:"message"`
	MessageKey  string           `json:"-"`
	MessageArgs []any            `json:"-"`
}

// InputErrors groups all errors found before a turn can be resolved. Parsing
// is deliberately atomic: callers receive every source error and the input
// game remains untouched.
type InputErrors struct {
	Errors []InputError `json:"errors"`
}

func newInputError(player models.PlayerID, noble models.NobleCode, line int, code, key string, args ...any) InputError {
	message := i18n.Message{Key: key, Args: args}
	return InputError{
		Player:      player,
		Noble:       noble,
		Line:        line,
		Code:        code,
		Message:     i18n.EnglishText(message),
		MessageKey:  key,
		MessageArgs: append([]any(nil), args...),
	}
}

func (e *InputErrors) Error() string {
	if e == nil || len(e.Errors) == 0 {
		return "invalid order submission"
	}
	return e.Errors[0].Message
}

// GameMapConfig returns the deterministic map-generation settings used by a
// game with playerCount players. Keeping this in the engine prevents the API
// from having a second, subtly different map configuration. The viewport is
// 1000x700 at four players and scales linearly with the player count.
func GameMapConfig(playerCount int) mapgen.Config {
	baseTerritories := mapgen.TerritoriesPerPlayer * playerCount
	return mapgen.Config{
		Width:            mapReferenceWidth * playerCount / mapReferencePlayers,
		Height:           mapReferenceHeight * playerCount / mapReferencePlayers,
		SiteCount:        baseTerritories + mapgen.TerritoriesPerVillage*(playerCount+1),
		VillageCount:     playerCount + 1,
		VillageSitesFrom: baseTerritories,
	}
}

// GenerateMap generates the static map used by CreateGame and the hotseat API.
func GenerateMap(seed string, playerCount int, assets assetgen.Assets) (mapgen.MapData, error) {
	if playerCount < minimumGamePlayers || playerCount > maximumGamePlayers {
		return mapgen.MapData{}, fmt.Errorf("engine: player count must be between %d and %d, got %d", minimumGamePlayers, maximumGamePlayers, playerCount)
	}
	return mapgen.Generate(seed, assets, GameMapConfig(playerCount))
}

// CreateGame creates the deterministic initial state for one game. Each player
// receives a distinct non-village starting territory, ordered by territory ID
// and sampled at regular intervals through the sorted eligible territory list.
func CreateGame(seed string, players []PlayerInit, balance assetgen.Balance, assets assetgen.Assets) (*models.GameState, error) {
	if len(players) < minimumGamePlayers || len(players) > maximumGamePlayers {
		return nil, fmt.Errorf("engine: player count must be between %d and %d, got %d", minimumGamePlayers, maximumGamePlayers, len(players))
	}
	if balance.StartingNobles < 0 || balance.StartingTroops < 0 || balance.StartingResources < 0 {
		return nil, fmt.Errorf("engine: starting balance values must be non-negative")
	}

	mapData, err := GenerateMap(seed, len(players), assets)
	if err != nil {
		return nil, err
	}

	state := models.NewGameState()
	state.ID = "game"
	state.Seed = seed
	state.Territories = make([]models.Territory, 0, len(mapData.Territories))
	state.TerritoryStates = make(map[models.TerritoryID]models.TerritoryState, len(mapData.Territories))

	villageIDs := make([]models.TerritoryID, 0)
	nonVillageIDs := make([]models.TerritoryID, 0)
	for _, generated := range mapData.Territories {
		territoryID := models.TerritoryID(generated.ID)
		adjacencies := make([]models.TerritoryID, len(generated.Adjacencies))
		for index, adjacentID := range generated.Adjacencies {
			adjacencies[index] = models.TerritoryID(adjacentID)
		}
		state.Territories = append(state.Territories, models.Territory{
			ID:          territoryID,
			Code:        generated.Code,
			Name:        generated.Name,
			Terrain:     generated.Terrain,
			Adjacencies: adjacencies,
		})
		state.TerritoryStates[territoryID] = models.TerritoryState{
			Infrastructures: []models.InfraID{},
		}
		if generated.Village {
			villageIDs = append(villageIDs, territoryID)
		} else {
			nonVillageIDs = append(nonVillageIDs, territoryID)
		}
	}
	if len(villageIDs) != len(players)+1 {
		return nil, fmt.Errorf("engine: map generated %d villages for %d players, want %d", len(villageIDs), len(players), len(players)+1)
	}
	if len(nonVillageIDs) < len(players) {
		return nil, fmt.Errorf("engine: map generated %d non-village territories for %d players", len(nonVillageIDs), len(players))
	}
	sort.Slice(nonVillageIDs, func(i, j int) bool { return nonVillageIDs[i] < nonVillageIDs[j] })
	adjacencies := make(map[models.TerritoryID][]models.TerritoryID, len(state.Territories))
	for _, territory := range state.Territories {
		adjacencies[territory.ID] = territory.Adjacencies
	}

	state.Players = make([]models.Player, 0, len(players))
	for index, init := range players {
		playerID := init.ID
		if playerID == "" {
			playerID = models.PlayerID(fmt.Sprintf("P%d", index+1))
		}
		for _, existing := range state.Players {
			if existing.ID == playerID {
				return nil, fmt.Errorf("engine: duplicate player id %q", playerID)
			}
		}
		name := init.Name
		if name == "" {
			name = string(playerID)
		}
		color := init.Color
		if color == "" {
			color = defaultPlayerColors[index%len(defaultPlayerColors)]
		}
		state.Players = append(state.Players, models.Player{ID: playerID, Name: name, Color: color})
	}

	starts, err := SelectStartingTerritories(nonVillageIDs, adjacencies, len(players))
	if err != nil {
		return nil, fmt.Errorf("engine: select starting territories: %w", err)
	}

	nextInfrastructureID := 1
	for index, startID := range starts {
		player := &state.Players[index]
		ownerID := player.ID
		territoryState := state.TerritoryStates[startID]
		territoryState.OwnerID = &ownerID
		territoryState.Resources = balance.StartingResources
		state.TerritoryStates[startID] = territoryState

		infrastructure := models.Infrastructure{
			ID:          models.InfraID(fmt.Sprintf("I%d", nextInfrastructureID)),
			Type:        models.InfraTypeCastle,
			Level:       1,
			TerritoryID: startID,
		}
		nextInfrastructureID++
		state.Infrastructures = append(state.Infrastructures, infrastructure)
		territoryState = state.TerritoryStates[startID]
		territoryState.Infrastructures = append(territoryState.Infrastructures, infrastructure.ID)
		state.TerritoryStates[startID] = territoryState
		capitalID := infrastructure.ID
		player.CapitalCastleID = &capitalID

		if balance.StartingTroops > 0 {
			armyID := models.ArmyID(fmt.Sprintf("A%d", state.NextArmyID))
			state.NextArmyID++
			state.Armies = append(state.Armies, models.Army{
				ID:          armyID,
				OwnerID:     player.ID,
				TerritoryID: startID,
				Size:        balance.StartingTroops,
			})
			territoryState = state.TerritoryStates[startID]
			territoryState.Army = &armyID
			state.TerritoryStates[startID] = territoryState
		}
	}

	for _, generated := range mapData.Territories {
		territoryID := models.TerritoryID(generated.ID)
		if !generated.Village {
			continue
		}
		infrastructure := models.Infrastructure{
			ID:          models.InfraID(fmt.Sprintf("I%d", nextInfrastructureID)),
			Type:        models.InfraTypeVillage,
			Level:       1,
			TerritoryID: territoryID,
		}
		nextInfrastructureID++
		state.Infrastructures = append(state.Infrastructures, infrastructure)
		territoryState := state.TerritoryStates[territoryID]
		territoryState.Infrastructures = append(territoryState.Infrastructures, infrastructure.ID)
		state.TerritoryStates[territoryID] = territoryState
	}

	if balance.StartingNobles > 0 {
		firstNames := append([]assetgen.Asset(nil), assets.Prenoms...)
		shuffleSetupNames(newSetupRNG(seed), firstNames)
		usedCodes := make(map[string]bool)
		nameIndex := 0
		for playerIndex, startID := range starts {
			for nobleIndex := 0; nobleIndex < balance.StartingNobles; nobleIndex++ {
				for nameIndex < len(firstNames) && usedCodes[firstNames[nameIndex].Code] {
					nameIndex++
				}
				if nameIndex == len(firstNames) {
					return nil, fmt.Errorf("engine: need %d unique first names, got %d", len(players)*balance.StartingNobles, len(assets.Prenoms))
				}
				firstName := firstNames[nameIndex]
				nameIndex++
				usedCodes[firstName.Code] = true
				territory := territoryByID(state.Territories, startID)
				state.Nobles = append(state.Nobles, models.Noble{
					ID:               models.NobleID(fmt.Sprintf("N%d", len(state.Nobles)+1)),
					Code:             firstName.Code,
					Name:             fmt.Sprintf("%s de %s", firstName.Name, territory.Name),
					OwnerID:          state.Players[playerIndex].ID,
					LocationID:       startID,
					Status:           models.NobleStatusFree,
					LastEmissionTurn: 0,
				})
			}
		}
	}

	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("engine: create game: invalid generated state: %w", err)
	}
	return state, nil
}

// ResolveTurn parses and resolves one complete season without mutating game.
// Calendar advancement is deliberately kept here because Resolve and
// ResolveWinter are reusable season resolvers that do not own the calendar.
func ResolveTurn(game *models.GameState, balance assetgen.Balance, input OrdersInput) (TurnReport, error) {
	if game == nil {
		return TurnReport{}, fmt.Errorf("engine: resolve turn: nil game state")
	}
	if err := game.Validate(); err != nil {
		return TurnReport{}, fmt.Errorf("engine: resolve turn: invalid game state: %w", err)
	}

	inputErrors := &InputErrors{Errors: []InputError{}}
	if game.Season == models.SeasonWinter && len(input.Chains) != 0 {
		inputErrors.Errors = append(inputErrors.Errors, newInputError("", "", 0, "chains_in_winter", i18n.ErrorChainsInWinter))
	}
	if game.Season != models.SeasonWinter && len(input.Winter) != 0 {
		inputErrors.Errors = append(inputErrors.Errors, newInputError("", "", 0, "winter_out_of_season", i18n.ErrorWinterOutOfSeason))
	}

	players := make(map[models.PlayerID]bool, len(game.Players))
	for _, player := range game.Players {
		players[player.ID] = true
	}

	type parsedSubmission struct {
		input          ChainSubmission
		chain          models.Chain
		noble          models.Noble
		receptionError error
	}
	parsedChains := make([]parsedSubmission, 0, len(input.Chains))
	seenNobles := make(map[models.NobleID]bool)
	for _, submission := range input.Chains {
		if !players[submission.Player] {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, submission.Noble, 0, "unknown_player", i18n.ErrorUnknownPlayer, submission.Player))
			continue
		}
		chain, parseErrors := orders.ParseChain(submission.Text, game)
		for _, parseError := range parseErrors {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, submission.Noble, parseError.Line, "parse_"+parseError.Code, parseError.MessageKey, parseError.MessageArgs...))
		}
		if len(parseErrors) != 0 {
			continue
		}

		noble, exists := findNoble(game.Nobles, chain.NobleID)
		if !exists {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, submission.Noble, 1, "unknown_noble", i18n.ErrorNobleUnknown))
			continue
		}
		if submission.Noble != "" && submission.Noble != models.NobleCode(noble.Code) {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, submission.Noble, 1, "noble_mismatch", i18n.ErrorNobleMismatch, submission.Noble, noble.Code))
			continue
		}
		if noble.OwnerID != submission.Player {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, models.NobleCode(noble.Code), 1, "noble_not_owned", i18n.ErrorNobleNotOwned, noble.Code, noble.OwnerID))
			continue
		}
		if seenNobles[noble.ID] {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, models.NobleCode(noble.Code), 1, "duplicate_emission", i18n.ErrorDuplicateEmission, noble.Code))
			continue
		}
		seenNobles[noble.ID] = true
		receptionError := orders.ValidateReceivingArmyOwnership(game, chain, noble.OwnerID)
		parsedChains = append(parsedChains, parsedSubmission{
			input:          submission,
			chain:          chain,
			noble:          noble,
			receptionError: receptionError,
		})
	}

	winterOrders := make(map[models.PlayerID][]models.WinterOrder)
	for _, submission := range input.Winter {
		if !players[submission.Player] {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, "", 0, "unknown_player", i18n.ErrorUnknownPlayer, submission.Player))
			continue
		}
		parsed, parseErrors := orders.ParseWinterOrders(submission.Lines, game)
		for _, parseError := range parseErrors {
			inputErrors.Errors = append(inputErrors.Errors, newInputError(submission.Player, "", parseError.Line, "parse_"+parseError.Code, parseError.MessageKey, parseError.MessageArgs...))
		}
		if len(parseErrors) == 0 {
			winterOrders[submission.Player] = append(winterOrders[submission.Player], parsed...)
		}
	}

	if len(inputErrors.Errors) != 0 {
		return TurnReport{}, inputErrors
	}

	working := cloneGameState(game)
	receptions := make([]ReceptionReport, 0, len(parsedChains))
	// Find target collisions before AssignChain can replace a previous chain or
	// consume an emitter's capacity.
	receivingArmies := make([]*models.Army, len(parsedChains))
	receptionCounts := make(map[models.ArmyID]int, len(parsedChains))
	for index, submission := range parsedChains {
		if submission.receptionError != nil {
			continue
		}
		army := orders.ReceivingArmy(working, submission.chain)
		receivingArmies[index] = army
		if army != nil {
			receptionCounts[army.ID]++
		}
	}
	for index, submission := range parsedChains {
		if submission.receptionError != nil {
			reception := ReceptionReport{
				Player:   submission.input.Player,
				Noble:    models.NobleCode(submission.noble.Code),
				Received: false,
				Reason:   submission.receptionError.Error(),
			}
			if message, ok := orders.CatalogMessage(submission.receptionError); ok {
				reception.ReasonKey = message.Key
				reception.ReasonArgs = message.Args
			}
			receptions = append(receptions, reception)
			continue
		}
		if army := receivingArmies[index]; army != nil && receptionCounts[army.ID] > 1 {
			position := territoryByID(working.Territories, army.TerritoryID)
			positionReference := position.Code
			if positionReference == "" {
				positionReference = string(position.ID)
			}
			receptions = append(receptions, ReceptionReport{
				Player:   submission.input.Player,
				Noble:    models.NobleCode(submission.noble.Code),
				Received: false,
				Reason: fmt.Sprintf(
					"concurrent_reception: army at %q was targeted by %d chains in turn %d",
					positionReference,
					receptionCounts[army.ID],
					working.Turn,
				),
				ReasonKey:  i18n.ReceptionConcurrent,
				ReasonArgs: []any{positionReference, receptionCounts[army.ID], working.Turn},
			})
			continue
		}
		if err := orders.AssignChain(working, submission.chain); err != nil {
			reception := ReceptionReport{
				Player:   submission.input.Player,
				Noble:    models.NobleCode(submission.noble.Code),
				Received: false,
				Reason:   err.Error(),
			}
			if message, ok := orders.CatalogMessage(err); ok {
				reception.ReasonKey = message.Key
				reception.ReasonArgs = message.Args
			}
			receptions = append(receptions, reception)
			continue
		}
		receptions = append(receptions, ReceptionReport{
			Player:   submission.input.Player,
			Noble:    models.NobleCode(submission.noble.Code),
			Received: true,
		})
	}

	var resolution Resolution
	var err error
	if game.Season == models.SeasonWinter {
		resolution, err = ResolveWinter(working, balance, winterOrders)
	} else {
		resolution, err = Resolve(working, balance)
	}
	if err != nil {
		return TurnReport{}, err
	}

	result := resolution.State
	result.Turn++
	result.Season = models.SeasonForTurn(result.Turn)
	if err := result.Validate(); err != nil {
		return TurnReport{}, fmt.Errorf("engine: resolve turn: invalid advanced result: %w", err)
	}

	report := BuildTurnReport(working, result, resolution.Events, receptions)
	report.State = result
	return report, nil
}

func territoryByID(territories []models.Territory, territoryID models.TerritoryID) models.Territory {
	for _, territory := range territories {
		if territory.ID == territoryID {
			return territory
		}
	}
	return models.Territory{ID: territoryID}
}

func findNoble(nobles []models.Noble, nobleID models.NobleID) (models.Noble, bool) {
	for _, noble := range nobles {
		if noble.ID == nobleID {
			return noble, true
		}
	}
	return models.Noble{}, false
}

func newSetupRNG(seed string) *rand.Rand {
	digest := sha256.Sum256([]byte(seed + "|setup-nobles"))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func shuffleSetupNames[T any](rng *rand.Rand, values []T) {
	for index := len(values) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		values[index], values[swap] = values[swap], values[index]
	}
}
