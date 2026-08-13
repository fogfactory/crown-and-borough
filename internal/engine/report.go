package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TurnReport is the public, typed summary of one resolved season. State is an
// internal handoff for the hotseat session and is intentionally excluded from
// JSON: the API returns the projected state as a separate document.
type TurnReport struct {
	Header     ReportHeader      `json:"header"`
	Players    []PlayerReport    `json:"players"`
	Receptions []ReceptionReport `json:"receptions"`
	Supply     []SupplyReport    `json:"supply"`
	Famines    []FamineReport    `json:"famines"`
	Combats    []CombatReport    `json:"combats"`
	Orders     []OrderReport     `json:"orders"`
	Moves      []MoveReport      `json:"moves"`
	Nobles     []NobleReport     `json:"nobles"`
	Winter     *WinterReport     `json:"winter,omitempty"`
	State      *models.GameState `json:"-"`
}

// ReportHeader identifies the season described by the report, before the
// calendar advances to the next season.
type ReportHeader struct {
	Year   int           `json:"year"`
	Season models.Season `json:"season"`
	Turn   int           `json:"turn"`
}

// ReceptionReport records whether a syntactically valid chain was received by
// an army. A failed reception is a lost chain and does not cancel the turn.
type ReceptionReport struct {
	Player   models.PlayerID  `json:"player"`
	Noble    models.NobleCode `json:"noble"`
	Received bool             `json:"received"`
	Reason   string           `json:"reason,omitempty"`
}

type PlayerReport struct {
	ID               models.PlayerID        `json:"id"`
	Name             string                 `json:"name"`
	ResourcesBefore  int                    `json:"resourcesBefore"`
	ResourcesAfter   int                    `json:"resourcesAfter"`
	ControlledBefore int                    `json:"controlledBefore"`
	ControlledAfter  int                    `json:"controlledAfter"`
	Armies           []ArmyReport           `json:"armies"`
	Nobles           []NobleReport          `json:"nobles"`
	Infrastructures  []InfrastructureReport `json:"infrastructures"`
}

type ArmyReport struct {
	ID        models.ArmyID      `json:"id"`
	Owner     models.PlayerID    `json:"owner"`
	Territory models.TerritoryID `json:"territory"`
	Size      int                `json:"size"`
}

type InfrastructureReport struct {
	ID        models.InfraID     `json:"id"`
	Type      models.InfraType   `json:"type"`
	Level     int                `json:"level"`
	Territory models.TerritoryID `json:"territory"`
}

type SupplyReport struct {
	Source        models.TerritoryID         `json:"source"`
	Owner         models.PlayerID            `json:"owner"`
	Production    int                        `json:"production"`
	Demand        int                        `json:"demand"`
	Rations       map[models.TerritoryID]int `json:"rations"`
	StockConsumed int                        `json:"stockConsumed"`
	StockAfter    int                        `json:"stockAfter"`
}

type FamineReport struct {
	Army               models.ArmyID      `json:"army"`
	Owner              models.PlayerID    `json:"owner"`
	Territory          models.TerritoryID `json:"territory"`
	Source             models.TerritoryID `json:"source"`
	Troops             int                `json:"troops"`
	TroopsLost         int                `json:"troopsLost,omitempty"`
	SavedByPillage     bool               `json:"savedByPillage"`
	Infrastructure     models.InfraID     `json:"infrastructure,omitempty"`
	InfrastructureType models.InfraType   `json:"infrastructureType,omitempty"`
	ResourceCredit     int                `json:"resourceCredit,omitempty"`
	CreditTerritory    models.TerritoryID `json:"creditTerritory,omitempty"`
}

type CombatReport struct {
	Territory     models.TerritoryID `json:"territory"`
	BaseDefense   int                `json:"baseDefense"`
	Defense       int                `json:"defense"`
	CastleBonus   int                `json:"castleBonus"`
	Contenders    []CombatContender  `json:"contenders"`
	Winner        models.ArmyID      `json:"winner,omitempty"`
	Dislodged     models.ArmyID      `json:"dislodged,omitempty"`
	CutSupporters []models.ArmyID    `json:"cutSupporters"`
	Reason        string             `json:"reason"`
	Standoff      bool               `json:"standoff"`
}

type OrderReport struct {
	Army             models.ArmyID                               `json:"army"`
	Chain            models.ChainID                              `json:"chain"`
	Order            models.OrderID                              `json:"order"`
	Owner            models.PlayerID                             `json:"owner"`
	Noble            models.NobleCode                            `json:"noble"`
	Type             models.OrderType                            `json:"type"`
	Source           models.TerritoryID                          `json:"source"`
	Target           models.TerritoryID                          `json:"target,omitempty"`
	Targets          []models.TerritoryID                        `json:"targets,omitempty"`
	NobleAssignments map[models.TerritoryCode][]models.NobleCode `json:"nobleAssignments,omitempty"`
	Liaison          models.LiaisonMode                          `json:"liaison"`
	Outcome          Outcome                                     `json:"outcome"`
	Reason           string                                      `json:"reason,omitempty"`
	Progression      Progression                                 `json:"progression"`
	IndexBefore      int                                         `json:"indexBefore"`
	IndexAfter       int                                         `json:"indexAfter"`
}

// MoveReport is a discriminated event projection. Kind is one of movement,
// fusion, dispersion, pillage, retreat, army_destroyed, or control_changed.
type MoveReport struct {
	Kind               EventType          `json:"kind"`
	Army               models.ArmyID      `json:"army,omitempty"`
	OtherArmy          models.ArmyID      `json:"otherArmy,omitempty"`
	Armies             []models.ArmyID    `json:"armies,omitempty"`
	Territory          models.TerritoryID `json:"territory,omitempty"`
	Source             models.TerritoryID `json:"source,omitempty"`
	Target             models.TerritoryID `json:"target,omitempty"`
	Destination        models.TerritoryID `json:"destination,omitempty"`
	AttackerOrigin     models.TerritoryID `json:"attackerOrigin,omitempty"`
	OrderType          models.OrderType   `json:"orderType,omitempty"`
	Outcome            Outcome            `json:"outcome,omitempty"`
	Reason             string             `json:"reason,omitempty"`
	Resolved           bool               `json:"resolved,omitempty"`
	Infrastructure     models.InfraID     `json:"infrastructure,omitempty"`
	InfrastructureType models.InfraType   `json:"infrastructureType,omitempty"`
	ResourceCredit     int                `json:"resourceCredit,omitempty"`
	CreditTerritory    models.TerritoryID `json:"creditTerritory,omitempty"`
	PreviousOwner      models.PlayerID    `json:"previousOwner,omitempty"`
	Owner              models.PlayerID    `json:"owner,omitempty"`
}

type NobleReport struct {
	Kind           EventType          `json:"kind,omitempty"`
	Noble          models.NobleID     `json:"noble"`
	Code           models.NobleCode   `json:"code,omitempty"`
	Name           string             `json:"name,omitempty"`
	Owner          models.PlayerID    `json:"owner,omitempty"`
	Army           models.ArmyID      `json:"army,omitempty"`
	Territory      models.TerritoryID `json:"territory,omitempty"`
	Source         models.TerritoryID `json:"source,omitempty"`
	Destination    models.TerritoryID `json:"destination,omitempty"`
	PreviousStatus models.NobleStatus `json:"previousStatus,omitempty"`
	Status         models.NobleStatus `json:"status,omitempty"`
	Captor         models.PlayerID    `json:"captor,omitempty"`
}

type WinterReport struct {
	Investments []WinterInvestmentReport `json:"investments"`
	Stocks      []WinterStockReport      `json:"stocks"`
}

type WinterInvestmentReport struct {
	Kind           EventType           `json:"kind"`
	Player         models.PlayerID     `json:"player"`
	Outcome        Outcome             `json:"outcome"`
	Cost           int                 `json:"cost"`
	Territory      models.TerritoryID  `json:"territory,omitempty"`
	Infrastructure models.InfraID      `json:"infrastructure,omitempty"`
	Type           models.InfraType    `json:"type,omitempty"`
	Level          int                 `json:"level,omitempty"`
	Noble          models.NobleID      `json:"noble,omitempty"`
	NobleCode      models.NobleCode    `json:"nobleCode,omitempty"`
	NobleName      string              `json:"nobleName,omitempty"`
	Reason         string              `json:"reason,omitempty"`
	Order          *models.WinterOrder `json:"order,omitempty"`
}

type WinterStockReport struct {
	Territory   models.TerritoryID `json:"territory"`
	Owner       models.PlayerID    `json:"owner,omitempty"`
	StockBefore int                `json:"stockBefore"`
	StockAfter  int                `json:"stockAfter"`
}

// BuildTurnReport converts value-only engine events into the typed report
// consumed by the API and frontend. It does not inspect resolution internals.
func BuildTurnReport(before, after *models.GameState, events []Event, receptions []ReceptionReport) TurnReport {
	report := TurnReport{
		Players:    []PlayerReport{},
		Receptions: []ReceptionReport{},
		Supply:     []SupplyReport{},
		Famines:    []FamineReport{},
		Combats:    []CombatReport{},
		Orders:     []OrderReport{},
		Moves:      []MoveReport{},
		Nobles:     []NobleReport{},
	}
	report.Receptions = append(report.Receptions, receptions...)
	if before != nil {
		report.Header = ReportHeader{Year: before.Year(), Season: before.Season, Turn: before.Turn}
		if before.Season == models.SeasonWinter {
			report.Winter = &WinterReport{Investments: []WinterInvestmentReport{}, Stocks: []WinterStockReport{}}
		}
	}
	report.Players = buildPlayerReports(before, after)

	armiesByID := make(map[models.ArmyID]models.Army)
	chainsByID := make(map[models.ChainID]models.Chain)
	noblesByID := make(map[models.NobleID]models.Noble)
	if before != nil {
		for _, army := range before.Armies {
			armiesByID[army.ID] = army
		}
		for _, chain := range before.Chains {
			chainsByID[chain.ID] = chain
		}
		for _, noble := range before.Nobles {
			noblesByID[noble.ID] = noble
		}
	}

	afterNobles := make(map[models.NobleID]models.Noble)
	if after != nil {
		for _, noble := range after.Nobles {
			afterNobles[noble.ID] = noble
		}
	}

	progressions := make(map[string]Event)
	orderIndexes := make(map[string]int)
	for _, event := range events {
		if event.Type == EventTypeChainProgression {
			progressions[eventKey(event.ChainID, event.OrderID)] = event
		}
	}
	for _, event := range events {
		switch event.Type {
		case EventTypeSupply:
			report.Supply = append(report.Supply, SupplyReport{
				Source: event.SourceID, Owner: event.OwnerID, Production: event.Production,
				Demand: event.Demand, Rations: cloneRationMap(event.Rations), StockConsumed: event.StockConsumed,
				StockAfter: event.StockAfter,
			})
		case EventTypeFamine:
			report.Famines = append(report.Famines, FamineReport{
				Army: event.ArmyID, Owner: event.OwnerID, Territory: event.TerritoryID, Source: event.SourceID,
				Troops: event.Troops, TroopsLost: event.TroopsLost, SavedByPillage: event.SavedByPillage,
				Infrastructure: event.InfrastructureID, InfrastructureType: event.InfrastructureType,
				ResourceCredit: event.ResourceCredit, CreditTerritory: event.CreditTerritoryID,
			})
		case EventTypeCombat:
			contenders := append([]CombatContender{}, event.Contenders...)
			cutSupporters := append([]models.ArmyID{}, event.CutSupporterIDs...)
			report.Combats = append(report.Combats, CombatReport{
				Territory: event.TerritoryID, BaseDefense: event.BaseDefense, Defense: event.Defense,
				CastleBonus: event.CastleBonus, Contenders: contenders,
				Winner: event.WinnerArmyID, Dislodged: event.DislodgedArmyID,
				CutSupporters: cutSupporters, Reason: event.Reason,
				Standoff: event.Reason == "standoff",
			})
		case EventTypeOrderOutcome:
			entry := OrderReport{
				Army: event.ArmyID, Chain: event.ChainID, Order: event.OrderID, Type: event.OrderType,
				Source: event.SourceID, Target: event.TargetID, Outcome: event.Outcome,
				Reason: event.Reason, Progression: event.Progression,
			}
			if army, exists := armiesByID[event.ArmyID]; exists {
				entry.Owner = army.OwnerID
			}
			if chain, exists := chainsByID[event.ChainID]; exists {
				if noble, nobleExists := noblesByID[chain.NobleID]; nobleExists {
					entry.Noble = models.NobleCode(noble.Code)
				}
				for _, order := range chain.Orders {
					if order.ID != event.OrderID {
						continue
					}
					entry.Type = order.Type
					entry.Source = order.PositionID
					entry.Targets = append([]models.TerritoryID(nil), order.TargetIDs...)
					if len(entry.Targets) > 0 {
						entry.Target = entry.Targets[0]
					}
					entry.NobleAssignments = cloneNobleAssignments(order.NobleAssignments)
					entry.Liaison = order.Liaison
					break
				}
			}
			if progression, exists := progressions[eventKey(event.ChainID, event.OrderID)]; exists {
				entry.IndexBefore = progression.IndexBefore
				entry.IndexAfter = progression.IndexAfter
				entry.Progression = progression.Progression
			}
			orderIndexes[eventKey(event.ChainID, event.OrderID)] = len(report.Orders)
			report.Orders = append(report.Orders, entry)
		case EventTypeMovement, EventTypeFusion, EventTypeDispersion, EventTypePillage,
			EventTypeRetreat, EventTypeArmyDestroyed, EventTypeControlChanged:
			report.Moves = append(report.Moves, MoveReport{
				Kind: event.Type, Army: event.ArmyID, OtherArmy: event.OtherArmyID,
				Armies: append([]models.ArmyID(nil), event.ArmyIDs...), Territory: event.TerritoryID,
				Source: event.SourceID, Target: event.TargetID, Destination: event.DestinationID,
				AttackerOrigin: event.AttackerOriginID, OrderType: event.OrderType, Outcome: event.Outcome,
				Reason: event.Reason, Resolved: event.Resolved, Infrastructure: event.InfrastructureID,
				InfrastructureType: event.InfrastructureType, ResourceCredit: event.ResourceCredit,
				CreditTerritory: event.CreditTerritoryID, PreviousOwner: event.PreviousOwnerID, Owner: event.OwnerID,
			})
		case EventTypeNobleMovement, EventTypeCapture, EventTypeLiberation:
			noble := afterNobles[event.NobleID]
			report.Nobles = append(report.Nobles, NobleReport{
				Kind: event.Type, Noble: event.NobleID, Code: models.NobleCode(noble.Code), Name: noble.Name,
				Owner: noble.OwnerID, Army: event.ArmyID, Territory: event.TerritoryID,
				Source: event.SourceID, Destination: event.DestinationID, PreviousStatus: event.PreviousStatus,
				Status: event.Status, Captor: event.CaptorPlayerID,
			})
			if event.Type == EventTypeCapture && event.Phase == winterPhase && report.Winter != nil && event.WinterOrder != nil {
				orderCopy := *event.WinterOrder
				report.Winter.Investments = append(report.Winter.Investments, WinterInvestmentReport{
					Kind: event.Type, Player: event.OwnerID, Outcome: OutcomeSuccess,
					Cost: event.ResourceSpent, Territory: event.TerritoryID,
					Noble: event.NobleID, NobleCode: event.NobleCode, NobleName: event.NobleName,
					Order: &orderCopy,
				})
			}
			if event.Type == EventTypeLiberation && report.Winter != nil {
				report.Winter.Investments = append(report.Winter.Investments, WinterInvestmentReport{
					Kind: event.Type, Player: event.OwnerID, Outcome: OutcomeSuccess, Territory: event.TerritoryID,
					Noble: event.NobleID, NobleCode: event.NobleCode, NobleName: event.NobleName,
					Cost: event.ResourceSpent,
				})
			}
		case EventTypeWinterStock, EventTypeRecruit, EventTypeBuild, EventTypeUpgrade,
			EventTypeRejected, EventTypeCapitalElected:
			if report.Winter == nil {
				report.Winter = &WinterReport{Investments: []WinterInvestmentReport{}, Stocks: []WinterStockReport{}}
			}
			if event.Type == EventTypeWinterStock {
				report.Winter.Stocks = append(report.Winter.Stocks, WinterStockReport{
					Territory: event.TerritoryID, Owner: event.OwnerID,
					StockBefore: event.StockBefore, StockAfter: event.StockAfter,
				})
				continue
			}
			if event.Type == EventTypeCapitalElected && event.Automatic {
				continue
			}
			investment := WinterInvestmentReport{
				Kind: event.Type, Player: event.OwnerID, Territory: event.TerritoryID,
				Outcome:        OutcomeSuccess,
				Cost:           event.ResourceSpent,
				Infrastructure: event.InfrastructureID, Type: event.InfrastructureType,
				Level: event.Level, Noble: event.NobleID, NobleCode: event.NobleCode,
				NobleName: event.NobleName, Reason: event.Reason,
			}
			if event.Type == EventTypeRejected {
				investment.Outcome = OutcomeFailure
				investment.Cost = 0
			}
			if event.WinterOrder != nil {
				order := *event.WinterOrder
				investment.Order = &order
			}
			report.Winter.Investments = append(report.Winter.Investments, investment)
		case EventTypeChainProgression:
			if index, exists := orderIndexes[eventKey(event.ChainID, event.OrderID)]; exists {
				report.Orders[index].Progression = event.Progression
				report.Orders[index].IndexBefore = event.IndexBefore
				report.Orders[index].IndexAfter = event.IndexAfter
			}
		}
	}
	return report
}

// BuildReport is kept as a concise alias for callers that use the report
// package terminology directly.
func BuildReport(before, after *models.GameState, events []Event, receptions []ReceptionReport) TurnReport {
	return BuildTurnReport(before, after, events, receptions)
}

func buildPlayerReports(before, after *models.GameState) []PlayerReport {
	players := []models.Player{}
	if after != nil {
		players = append(players, after.Players...)
	} else if before != nil {
		players = append(players, before.Players...)
	}
	reports := make([]PlayerReport, 0, len(players))
	for _, player := range players {
		report := PlayerReport{ID: player.ID, Name: player.Name, Armies: []ArmyReport{}, Nobles: []NobleReport{}, Infrastructures: []InfrastructureReport{}}
		report.ResourcesBefore, report.ControlledBefore = playerTerritoryTotals(before, player.ID)
		report.ResourcesAfter, report.ControlledAfter = playerTerritoryTotals(after, player.ID)
		if after != nil {
			for _, army := range after.Armies {
				if army.OwnerID == player.ID {
					report.Armies = append(report.Armies, ArmyReport{ID: army.ID, Owner: army.OwnerID, Territory: army.TerritoryID, Size: army.Size})
				}
			}
			for _, noble := range after.Nobles {
				if noble.OwnerID == player.ID {
					report.Nobles = append(report.Nobles, NobleReport{Noble: noble.ID, Code: models.NobleCode(noble.Code), Name: noble.Name, Owner: noble.OwnerID, Territory: noble.LocationID, Status: noble.Status})
				}
			}
			for _, infrastructure := range after.Infrastructures {
				if state := after.TerritoryStates[infrastructure.TerritoryID]; state.OwnerID != nil && *state.OwnerID == player.ID {
					report.Infrastructures = append(report.Infrastructures, InfrastructureReport{ID: infrastructure.ID, Type: infrastructure.Type, Level: infrastructure.Level, Territory: infrastructure.TerritoryID})
				}
			}
		}
		sort.Slice(report.Armies, func(i, j int) bool { return report.Armies[i].ID < report.Armies[j].ID })
		sort.Slice(report.Nobles, func(i, j int) bool { return report.Nobles[i].Noble < report.Nobles[j].Noble })
		sort.Slice(report.Infrastructures, func(i, j int) bool { return report.Infrastructures[i].ID < report.Infrastructures[j].ID })
		reports = append(reports, report)
	}
	return reports
}

func playerTerritoryTotals(state *models.GameState, playerID models.PlayerID) (int, int) {
	if state == nil {
		return 0, 0
	}
	resources := 0
	controlled := 0
	for _, territory := range state.Territories {
		territoryState := state.TerritoryStates[territory.ID]
		if territoryState.OwnerID == nil || *territoryState.OwnerID != playerID {
			continue
		}
		controlled++
		resources += territoryState.Resources
	}
	return resources, controlled
}

func eventKey(chainID models.ChainID, orderID models.OrderID) string {
	return fmt.Sprintf("%s/%s", chainID, orderID)
}

func cloneRationMap(source map[models.TerritoryID]int) map[models.TerritoryID]int {
	if source == nil {
		return map[models.TerritoryID]int{}
	}
	clone := make(map[models.TerritoryID]int, len(source))
	for territoryID, amount := range source {
		clone[territoryID] = amount
	}
	return clone
}
