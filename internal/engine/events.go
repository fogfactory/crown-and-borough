package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// Resolution is the deterministic result of resolving one game state. State is
// a deep clone of the input after all resolution phases have completed.
type Resolution struct {
	State  *models.GameState `json:"state"`
	Events []Event           `json:"events"`
}

// EventType identifies the kind of a resolution event.
type EventType string

const (
	EventTypeOrderOutcome     EventType = "order_outcome"
	EventTypeCombat           EventType = "combat"
	EventTypeMovement         EventType = "movement"
	EventTypeFusion           EventType = "fusion"
	EventTypeDispersion       EventType = "dispersion"
	EventTypePillage          EventType = "pillage"
	EventTypeRetreat          EventType = "retreat"
	EventTypeArmyDestroyed    EventType = "army_destroyed"
	EventTypeNobleMovement    EventType = "noble_movement"
	EventTypeCapture          EventType = "capture"
	EventTypeControlChanged   EventType = "control_changed"
	EventTypeChainProgression EventType = "chain_progression"
	EventTypeSupply           EventType = "supply"
	EventTypeFamine           EventType = "famine"
	EventTypeWinterStock      EventType = "winter_stock"
	EventTypeRecruit          EventType = "recruit"
	EventTypeBuild            EventType = "build"
	EventTypeUpgrade          EventType = "upgrade"
	EventTypeRejected         EventType = "rejected"
	EventTypeCapitalElected   EventType = "capital_elected"
	EventTypeLiberation       EventType = "liberation"
)

// Outcome is the execution result of one current order.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeInvalid Outcome = "invalid"
)

// Progression records how the order outcome changed its chain.
type Progression string

const (
	ProgressionAdvanced Progression = "advanced"
	ProgressionRetried  Progression = "retried"
	ProgressionBroken   Progression = "broken"
	ProgressionConsumed Progression = "consumed"
)

// CombatContender is one independent attacking or defending force in a combat
// event. An empty ArmyID denotes castle-only defense.
type CombatContender struct {
	ArmyID   models.ArmyID   `json:"army,omitempty"`
	OwnerID  models.PlayerID `json:"owner,omitempty"`
	Force    int             `json:"force"`
	Defender bool            `json:"defender"`
}

// Event is a value-only report of one resolution decision. Fields irrelevant
// to Type are left at their zero value. IDs and forces are copied so events do
// not retain mutable engine state.
type Event struct {
	Type  EventType `json:"type"`
	Phase int       `json:"phase"`

	ArmyID      models.ArmyID    `json:"army,omitempty"`
	OtherArmyID models.ArmyID    `json:"otherArmy,omitempty"`
	ArmyIDs     []models.ArmyID  `json:"armies,omitempty"`
	ChainID     models.ChainID   `json:"chain,omitempty"`
	OrderID     models.OrderID   `json:"order,omitempty"`
	OrderType   models.OrderType `json:"orderType,omitempty"`
	Outcome     Outcome          `json:"outcome,omitempty"`
	Reason      string           `json:"reason,omitempty"`
	Progression Progression      `json:"progression,omitempty"`

	TerritoryID       models.TerritoryID `json:"territory,omitempty"`
	SourceID          models.TerritoryID `json:"source,omitempty"`
	TargetID          models.TerritoryID `json:"target,omitempty"`
	DestinationID     models.TerritoryID `json:"destination,omitempty"`
	AttackerOriginID  models.TerritoryID `json:"attackerOrigin,omitempty"`
	BaseDefense       int                `json:"baseDefense,omitempty"`
	Defense           int                `json:"defense,omitempty"`
	CastleBonus       int                `json:"castleBonus,omitempty"`
	Contenders        []CombatContender  `json:"contenders,omitempty"`
	WinnerArmyID      models.ArmyID      `json:"winnerArmy,omitempty"`
	DislodgedArmyID   models.ArmyID      `json:"dislodgedArmy,omitempty"`
	CutSupporterIDs   []models.ArmyID    `json:"cutSupporters,omitempty"`
	Resolved          bool               `json:"resolved,omitempty"`
	RemainingStrength int                `json:"remainingStrength,omitempty"`

	InfrastructureID   models.InfraID             `json:"infrastructure,omitempty"`
	InfrastructureType models.InfraType           `json:"infrastructureType,omitempty"`
	Level              int                        `json:"level,omitempty"`
	ResourceCredit     int                        `json:"resourceCredit,omitempty"`
	CreditTerritoryID  models.TerritoryID         `json:"creditTerritory,omitempty"`
	Production         int                        `json:"production,omitempty"`
	Demand             int                        `json:"demand,omitempty"`
	Rations            map[models.TerritoryID]int `json:"rations,omitempty"`
	StockConsumed      int                        `json:"stockConsumed,omitempty"`
	StockBefore        int                        `json:"stockBefore,omitempty"`
	StockAfter         int                        `json:"stockAfter,omitempty"`
	Troops             int                        `json:"troops,omitempty"`
	SavedByPillage     bool                       `json:"savedByPillage,omitempty"`

	NobleID         models.NobleID      `json:"noble,omitempty"`
	NobleCode       models.NobleCode    `json:"nobleCode,omitempty"`
	NobleName       string              `json:"nobleName,omitempty"`
	PreviousStatus  models.NobleStatus  `json:"previousStatus,omitempty"`
	Status          models.NobleStatus  `json:"status,omitempty"`
	CaptorPlayerID  models.PlayerID     `json:"captorPlayer,omitempty"`
	PreviousOwnerID models.PlayerID     `json:"previousOwner,omitempty"`
	OwnerID         models.PlayerID     `json:"owner,omitempty"`
	IndexBefore     int                 `json:"indexBefore,omitempty"`
	IndexAfter      int                 `json:"indexAfter,omitempty"`
	WinterOrder     *models.WinterOrder `json:"winterOrder,omitempty"`
}
