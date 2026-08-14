package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type receivedChain struct {
	chain           models.Chain
	previousChainID *models.ChainID
}

// trackTurnPrivacy applies the privacy effects of a resolved turn to the
// cloned result returned by the engine. The engine remains unaware of viewers.
func trackTurnPrivacy(before, after *models.GameState, input engine.OrdersInput, report engine.TurnReport) {
	privacy := ensurePrivacy(after)
	for _, received := range receivedChains(before, input, report.Receptions) {
		recordChainKnowledge(before, after, received.chain, received.previousChainID)
	}
	trackCombatParticipation(before, after, report.Combats, privacy)
	reconcileChainKnowledge(after)
}

func ensurePrivacy(state *models.GameState) *models.PrivacyMeta {
	if state.Privacy == nil {
		state.Privacy = &models.PrivacyMeta{}
	}
	if state.Privacy.ChainKnowledge == nil {
		state.Privacy.ChainKnowledge = make(map[models.PlayerID]map[models.ChainID]models.ChainSnapshot)
	}
	if state.Privacy.CombatParticipation == nil {
		state.Privacy.CombatParticipation = make(map[models.PlayerID]map[string]bool)
	}
	return state.Privacy
}

func receivedChains(before *models.GameState, input engine.OrdersInput, receptions []engine.ReceptionReport) []receivedChain {
	if before == nil {
		return nil
	}
	nextChainID := before.NextChainID
	received := make([]receivedChain, 0, len(input.Chains))
	usedReceptions := make([]bool, len(receptions))
	for _, submission := range input.Chains {
		parsed, parseErrors := orders.ParseChain(submission.Text, before)
		if len(parseErrors) != 0 || len(parsed.Orders) == 0 {
			continue
		}
		noble, exists := nobleByID(before, parsed.NobleID)
		receptionIndex := findReceivedReception(receptions, usedReceptions, submission.Player, models.NobleCode(noble.Code))
		if !exists || receptionIndex < 0 {
			continue
		}
		usedReceptions[receptionIndex] = true
		army := armyAtTerritory(before, parsed.Orders[0].PositionID)
		if army == nil {
			continue
		}

		chainID := models.ChainID(fmt.Sprintf("C%d", nextChainID))
		nextChainID++
		parsed.ID = chainID
		parsed.ArmyID = army.ID
		parsed.CurrentIndex = 0
		for index := range parsed.Orders {
			parsed.Orders[index].ArmyID = army.ID
		}
		var previousChainID *models.ChainID
		if army.ChainID != nil {
			previous := *army.ChainID
			previousChainID = &previous
		}
		received = append(received, receivedChain{chain: parsed, previousChainID: previousChainID})
	}
	return received
}

func findReceivedReception(receptions []engine.ReceptionReport, used []bool, playerID models.PlayerID, noble models.NobleCode) int {
	for index, reception := range receptions {
		if used[index] || reception.Player != playerID || reception.Noble != noble {
			continue
		}
		if reception.Received {
			return index
		}
		return -1
	}
	return -1
}

func recordChainKnowledge(before, after *models.GameState, chain models.Chain, previousChainID *models.ChainID) {
	if before == nil || after == nil {
		return
	}
	noble, nobleExists := nobleByID(before, chain.NobleID)
	army, armyExists := armyByID(before, chain.ArmyID)
	if !nobleExists || !armyExists {
		return
	}
	privacy := ensurePrivacy(after)
	snapshot := makeChainSnapshot(chain, before.Turn)

	if previousChainID != nil {
		// The army owner sees the replacement through their own submission. A
		// third party must keep its prior knowledge until an action contradicts it.
		deleteChainSnapshot(privacy, army.OwnerID, *previousChainID)
	}
	putChainSnapshot(privacy, noble.OwnerID, snapshot)

	if noble.Status != models.NobleStatusHostage {
		return
	}
	holder := armyAtTerritory(before, noble.LocationID)
	if holder != nil && holder.OwnerID != noble.OwnerID {
		putChainSnapshot(privacy, holder.OwnerID, snapshot)
	}
}

func makeChainSnapshot(chain models.Chain, turn int) models.ChainSnapshot {
	snapshot := models.ChainSnapshot{
		ID:           chain.ID,
		NobleID:      chain.NobleID,
		ArmyID:       chain.ArmyID,
		Orders:       cloneOrders(chain.Orders),
		CurrentIndex: chain.CurrentIndex,
		CapturedTurn: turn,
	}
	return snapshot
}

func putChainSnapshot(privacy *models.PrivacyMeta, playerID models.PlayerID, snapshot models.ChainSnapshot) {
	if playerID == "" {
		return
	}
	if privacy.ChainKnowledge[playerID] == nil {
		privacy.ChainKnowledge[playerID] = make(map[models.ChainID]models.ChainSnapshot)
	}
	privacy.ChainKnowledge[playerID][snapshot.ID] = cloneChainSnapshot(snapshot)
}

func deleteChainSnapshot(privacy *models.PrivacyMeta, playerID models.PlayerID, chainID models.ChainID) {
	if privacy == nil || privacy.ChainKnowledge == nil {
		return
	}
	delete(privacy.ChainKnowledge[playerID], chainID)
}

func cloneChainSnapshot(source models.ChainSnapshot) models.ChainSnapshot {
	clone := source
	clone.Orders = cloneOrders(source.Orders)
	return clone
}

func cloneOrders(source []models.Order) []models.Order {
	if source == nil {
		return nil
	}
	clone := make([]models.Order, len(source))
	for index, order := range source {
		clone[index] = order
		clone[index].TargetIDs = append([]models.TerritoryID(nil), order.TargetIDs...)
		if order.NobleAssignments != nil {
			clone[index].NobleAssignments = make(map[models.TerritoryID][]models.NobleCode, len(order.NobleAssignments))
			for destination, codes := range order.NobleAssignments {
				clone[index].NobleAssignments[destination] = append([]models.NobleCode(nil), codes...)
			}
		}
	}
	return clone
}

func reconcileChainKnowledge(state *models.GameState) {
	if state == nil || state.Privacy == nil {
		return
	}
	armies := make(map[models.ArmyID]models.Army, len(state.Armies))
	activeChains := make(map[models.ChainID]bool, len(state.Chains))
	for _, army := range state.Armies {
		armies[army.ID] = army
	}
	for _, chain := range state.Chains {
		activeChains[chain.ID] = true
	}
	for viewer, snapshots := range state.Privacy.ChainKnowledge {
		for chainID, snapshot := range snapshots {
			army, exists := armies[snapshot.ArmyID]
			if !exists {
				delete(snapshots, chainID)
				continue
			}
			if activeChains[snapshot.ID] || army.OwnerID == viewer {
				continue
			}
			if !trajectoryContains(snapshot, army.TerritoryID) {
				delete(snapshots, chainID)
			}
		}
	}
}

// trajectoryContains is intentionally conservative: a known chain remains
// plausible when the army is at a source or destination named by any order in
// its known history. Knowledge is discarded only once the public position is
// outside that set, avoiding invalidation merely because a replacement was
// received.
func trajectoryContains(snapshot models.ChainSnapshot, position models.TerritoryID) bool {
	if len(snapshot.Orders) == 0 {
		return false
	}
	// Include the complete known chain rather than only the current suffix. A
	// snapshot can be created after a chain has already progressed, and the
	// earlier positions remain part of the observer's compatible history.
	for _, order := range snapshot.Orders {
		if order.PositionID == position {
			return true
		}
		switch order.Type {
		case models.OrderTypeAttack, models.OrderTypeJoin:
			if len(order.TargetIDs) > 0 && order.TargetIDs[0] == position {
				return true
			}
		case models.OrderTypeDisperse:
			for _, targetID := range order.TargetIDs {
				if targetID == position {
					return true
				}
			}
		}
	}
	return false
}

func trackCombatParticipation(before, after *models.GameState, combats []engine.CombatReport, privacy *models.PrivacyMeta) {
	if privacy == nil {
		return
	}
	// Combat IDs are intentionally scoped to the current report as
	// combat-<territory>. Historical reports will carry their own audience
	// metadata when report persistence is introduced.
	privacy.CombatParticipation = make(map[models.PlayerID]map[string]bool)
	beforeArmies := armyOwners(before)
	afterArmies := armyOwners(after)
	for _, combat := range combats {
		combatID := fmt.Sprintf("combat-%s", combat.Territory)
		participants := make(map[models.PlayerID]bool)
		for _, contender := range combat.Contenders {
			if contender.OwnerID != "" {
				participants[contender.OwnerID] = true
			}
			if owner, ok := beforeArmies[contender.ArmyID]; ok {
				participants[owner] = true
			}
			if owner, ok := afterArmies[contender.ArmyID]; ok {
				participants[owner] = true
			}
		}
		for _, armyID := range append(append([]models.ArmyID{}, combat.Supporters...), combat.CutSupporters...) {
			if owner, ok := beforeArmies[armyID]; ok {
				participants[owner] = true
			}
			if owner, ok := afterArmies[armyID]; ok {
				participants[owner] = true
			}
		}
		if owner := territoryOwner(before, after, combat.Territory); owner != "" {
			for _, contender := range combat.Contenders {
				if contender.ArmyID == "" && contender.Defender {
					participants[owner] = true
					break
				}
			}
		}
		for playerID := range participants {
			if privacy.CombatParticipation[playerID] == nil {
				privacy.CombatParticipation[playerID] = make(map[string]bool)
			}
			privacy.CombatParticipation[playerID][combatID] = true
		}
	}
}

func armyOwners(state *models.GameState) map[models.ArmyID]models.PlayerID {
	owners := make(map[models.ArmyID]models.PlayerID)
	if state == nil {
		return owners
	}
	for _, army := range state.Armies {
		owners[army.ID] = army.OwnerID
	}
	return owners
}

func territoryOwner(before, after *models.GameState, territoryID models.TerritoryID) models.PlayerID {
	for _, state := range []*models.GameState{before, after} {
		if state == nil {
			continue
		}
		territoryState, exists := state.TerritoryStates[territoryID]
		if exists && territoryState.OwnerID != nil {
			return *territoryState.OwnerID
		}
	}
	return ""
}

func armyByID(state *models.GameState, armyID models.ArmyID) (models.Army, bool) {
	if state == nil {
		return models.Army{}, false
	}
	for _, army := range state.Armies {
		if army.ID == armyID {
			return army, true
		}
	}
	return models.Army{}, false
}

func nobleByID(state *models.GameState, nobleID models.NobleID) (models.Noble, bool) {
	if state == nil {
		return models.Noble{}, false
	}
	for _, noble := range state.Nobles {
		if noble.ID == nobleID {
			return noble, true
		}
	}
	return models.Noble{}, false
}

func armyAtTerritory(state *models.GameState, territoryID models.TerritoryID) *models.Army {
	if state == nil {
		return nil
	}
	territoryState, exists := state.TerritoryStates[territoryID]
	if !exists || territoryState.Army == nil {
		return nil
	}
	army, exists := armyByID(state, *territoryState.Army)
	if !exists || army.TerritoryID != territoryID {
		return nil
	}
	return &army
}

func requestedViewer(r *http.Request) (models.PlayerID, bool, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", false, err
	}
	players, exists := values["player"]
	if !exists {
		return "", false, nil
	}
	if len(players) != 1 || players[0] == "" {
		return "", false, fmt.Errorf("player must be provided exactly once")
	}
	return models.PlayerID(players[0]), true, nil
}

// TurnReportView is the player-filtered JSON form of a turn report. Combat
// details and order details from unknown chains are redacted server-side.
type TurnReportView struct {
	Header     engine.ReportHeader      `json:"header"`
	Players    []engine.PlayerReport    `json:"players"`
	Receptions []engine.ReceptionReport `json:"receptions"`
	Supply     []engine.SupplyReport    `json:"supply"`
	Famines    []engine.FamineReport    `json:"famines"`
	Combats    []CombatView             `json:"combats"`
	Orders     []OrderReportView        `json:"orders"`
	Moves      []engine.MoveReport      `json:"moves"`
	Nobles     []engine.NobleReport     `json:"nobles"`
	Winter     *engine.WinterReport     `json:"winter,omitempty"`
}

// OrderReportView keeps order outcomes useful to spectators without returning
// the source, targets, noble, or internal identifiers of an unknown chain.
type OrderReportView struct {
	Visibility       string
	Army             models.ArmyID
	Chain            models.ChainID
	Order            models.OrderID
	Owner            models.PlayerID
	Noble            models.NobleCode
	Type             models.OrderType
	Source           models.TerritoryID
	Target           models.TerritoryID
	Targets          []models.TerritoryID
	NobleAssignments map[models.TerritoryID][]models.NobleCode
	Liaison          models.LiaisonMode
	Outcome          engine.Outcome
	Reason           string
	Progression      engine.Progression
	IndexBefore      int
	IndexAfter       int
}

func (view OrderReportView) MarshalJSON() ([]byte, error) {
	if view.Visibility == "hidden" {
		return json.Marshal(struct {
			Visibility string         `json:"visibility"`
			Outcome    engine.Outcome `json:"outcome"`
		}{Visibility: view.Visibility, Outcome: view.Outcome})
	}
	return json.Marshal(struct {
		Visibility       string                                    `json:"visibility"`
		Army             models.ArmyID                             `json:"army"`
		Chain            models.ChainID                            `json:"chain"`
		Order            models.OrderID                            `json:"order"`
		Owner            models.PlayerID                           `json:"owner"`
		Noble            models.NobleCode                          `json:"noble"`
		Type             models.OrderType                          `json:"type"`
		Source           models.TerritoryID                        `json:"source"`
		Target           models.TerritoryID                        `json:"target,omitempty"`
		Targets          []models.TerritoryID                      `json:"targets,omitempty"`
		NobleAssignments map[models.TerritoryID][]models.NobleCode `json:"nobleAssignments,omitempty"`
		Liaison          models.LiaisonMode                        `json:"liaison"`
		Outcome          engine.Outcome                            `json:"outcome"`
		Reason           string                                    `json:"reason,omitempty"`
		Progression      engine.Progression                        `json:"progression"`
		IndexBefore      int                                       `json:"indexBefore"`
		IndexAfter       int                                       `json:"indexAfter"`
	}{
		Visibility:       view.Visibility,
		Army:             view.Army,
		Chain:            view.Chain,
		Order:            view.Order,
		Owner:            view.Owner,
		Noble:            view.Noble,
		Type:             view.Type,
		Source:           view.Source,
		Target:           view.Target,
		Targets:          view.Targets,
		NobleAssignments: view.NobleAssignments,
		Liaison:          view.Liaison,
		Outcome:          view.Outcome,
		Reason:           view.Reason,
		Progression:      view.Progression,
		IndexBefore:      view.IndexBefore,
		IndexAfter:       view.IndexAfter,
	})
}

// CombatView has two JSON variants. The general variant deliberately has no
// fields from which forces or army identities can be reconstructed.
type CombatView struct {
	Visibility    string
	Territory     models.TerritoryID
	BaseDefense   int
	Defense       int
	CastleBonus   int
	Contenders    []engine.CombatContender
	Supporters    []models.ArmyID
	Winner        models.ArmyID
	Dislodged     models.ArmyID
	CutSupporters []models.ArmyID
	Reason        string
	Standoff      bool
	Outcome       string
	Summary       string
}

func (view CombatView) MarshalJSON() ([]byte, error) {
	if view.Visibility == "general" {
		return json.Marshal(struct {
			Visibility string             `json:"visibility"`
			Territory  models.TerritoryID `json:"territory"`
			Outcome    string             `json:"outcome"`
			Summary    string             `json:"summary"`
		}{
			Visibility: view.Visibility,
			Territory:  view.Territory,
			Outcome:    view.Outcome,
			Summary:    view.Summary,
		})
	}
	return json.Marshal(struct {
		Visibility    string                   `json:"visibility"`
		Territory     models.TerritoryID       `json:"territory"`
		BaseDefense   int                      `json:"baseDefense"`
		Defense       int                      `json:"defense"`
		CastleBonus   int                      `json:"castleBonus"`
		Contenders    []engine.CombatContender `json:"contenders"`
		Supporters    []models.ArmyID          `json:"supporters,omitempty"`
		Winner        models.ArmyID            `json:"winner,omitempty"`
		Dislodged     models.ArmyID            `json:"dislodged,omitempty"`
		CutSupporters []models.ArmyID          `json:"cutSupporters"`
		Reason        string                   `json:"reason"`
		Standoff      bool                     `json:"standoff"`
	}{
		Visibility:    view.Visibility,
		Territory:     view.Territory,
		BaseDefense:   view.BaseDefense,
		Defense:       view.Defense,
		CastleBonus:   view.CastleBonus,
		Contenders:    view.Contenders,
		Supporters:    view.Supporters,
		Winner:        view.Winner,
		Dislodged:     view.Dislodged,
		CutSupporters: view.CutSupporters,
		Reason:        view.Reason,
		Standoff:      view.Standoff,
	})
}

func projectReport(report engine.TurnReport, viewer models.PlayerID, privacy *models.PrivacyMeta) TurnReportView {
	view := TurnReportView{
		Header:     report.Header,
		Players:    append([]engine.PlayerReport{}, report.Players...),
		Receptions: append([]engine.ReceptionReport{}, report.Receptions...),
		Supply:     append([]engine.SupplyReport{}, report.Supply...),
		Famines:    append([]engine.FamineReport{}, report.Famines...),
		Combats:    make([]CombatView, 0, len(report.Combats)),
		Orders:     make([]OrderReportView, 0, len(report.Orders)),
		Moves:      append([]engine.MoveReport{}, report.Moves...),
		Nobles:     append([]engine.NobleReport{}, report.Nobles...),
		Winter:     report.Winter,
	}
	for _, order := range report.Orders {
		if privacy != nil && viewerKnowsChainSnapshot(privacy, viewer, order.Chain) {
			view.Orders = append(view.Orders, knownOrderReport(order))
		} else {
			view.Orders = append(view.Orders, OrderReportView{Visibility: "hidden", Outcome: order.Outcome})
		}
	}
	for _, combat := range report.Combats {
		combatID := fmt.Sprintf("combat-%s", combat.Territory)
		exact := privacy != nil && privacy.CombatParticipation[viewer][combatID]
		if exact {
			view.Combats = append(view.Combats, CombatView{
				Visibility:    "exact",
				Territory:     combat.Territory,
				BaseDefense:   combat.BaseDefense,
				Defense:       combat.Defense,
				CastleBonus:   combat.CastleBonus,
				Contenders:    append([]engine.CombatContender(nil), combat.Contenders...),
				Supporters:    append([]models.ArmyID(nil), combat.Supporters...),
				Winner:        combat.Winner,
				Dislodged:     combat.Dislodged,
				CutSupporters: append([]models.ArmyID(nil), combat.CutSupporters...),
				Reason:        combat.Reason,
				Standoff:      combat.Standoff,
			})
			continue
		}
		outcome, summary := generalCombatSummary(combat)
		view.Combats = append(view.Combats, CombatView{
			Visibility: "general",
			Territory:  combat.Territory,
			Outcome:    outcome,
			Summary:    summary,
		})
	}
	return view
}

func viewerKnowsChainSnapshot(privacy *models.PrivacyMeta, viewer models.PlayerID, chainID models.ChainID) bool {
	if privacy == nil || chainID == "" {
		return false
	}
	_, known := privacy.ChainKnowledge[viewer][chainID]
	return known
}

func knownOrderReport(order engine.OrderReport) OrderReportView {
	return OrderReportView{
		Visibility:       "known",
		Army:             order.Army,
		Chain:            order.Chain,
		Order:            order.Order,
		Owner:            order.Owner,
		Noble:            order.Noble,
		Type:             order.Type,
		Source:           order.Source,
		Target:           order.Target,
		Targets:          append([]models.TerritoryID(nil), order.Targets...),
		NobleAssignments: cloneNobleAssignments(order.NobleAssignments),
		Liaison:          order.Liaison,
		Outcome:          order.Outcome,
		Reason:           order.Reason,
		Progression:      order.Progression,
		IndexBefore:      order.IndexBefore,
		IndexAfter:       order.IndexAfter,
	}
}

func cloneNobleAssignments(source map[models.TerritoryID][]models.NobleCode) map[models.TerritoryID][]models.NobleCode {
	if source == nil {
		return nil
	}
	clone := make(map[models.TerritoryID][]models.NobleCode, len(source))
	for destination, codes := range source {
		clone[destination] = append([]models.NobleCode(nil), codes...)
	}
	return clone
}

func generalCombatSummary(combat engine.CombatReport) (string, string) {
	if combat.Standoff || combat.Reason == "standoff" {
		return "standoff", "The combat ended without a winner."
	}
	if combat.Winner != "" || combat.Reason == "attack_wins" {
		return "attack_wins", "An attack overcame the defense."
	}
	return "defense_holds", "The defense held."
}
