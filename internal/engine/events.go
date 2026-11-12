package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// Resolution is the deterministic result of resolving one game state. State is
// a deep clone of the input after all resolution phases have completed.
type Resolution struct {
	State  *models.GameState `json:"state"`
	Events []Event           `json:"events"`
}

// PillageEffect is what a pillage did to an infrastructure: it degrades it
// instead of always destroying it.
type PillageEffect string

const (
	PillageEffectDestroyed   PillageEffect = "destroyed"
	PillageEffectDowngraded  PillageEffect = "downgraded"
	PillageEffectUnfortified PillageEffect = "unfortified"
	PillageEffectRuined      PillageEffect = "ruined"
)

// EventType identifies the kind of a resolution event.
type EventType string

const (
	EventTypeOrderOutcome       EventType = "order_outcome"
	EventTypeCombat             EventType = "combat"
	EventTypeMovement           EventType = "movement"
	EventTypeFusion             EventType = "fusion"
	EventTypeDispersion         EventType = "dispersion"
	EventTypePillage            EventType = "pillage"
	EventTypeRetreat            EventType = "retreat"
	EventTypeArmyDestroyed      EventType = "army_destroyed"
	EventTypeNobleMovement      EventType = "noble_movement"
	EventTypeCapture            EventType = "capture"
	EventTypeControlChanged     EventType = "control_changed"
	EventTypeChainProgression   EventType = "chain_progression"
	EventTypeSupply             EventType = "supply"
	EventTypeFamine             EventType = "famine"
	EventTypeTransfer           EventType = "transfer"
	EventTypeWinterStock        EventType = "winter_stock"
	EventTypeRecruit            EventType = "recruit"
	EventTypeBuild              EventType = "build"
	EventTypeUpgrade            EventType = "upgrade"
	EventTypeFortify            EventType = "fortify"
	EventTypeRejected           EventType = "rejected"
	EventTypeCapitalElected     EventType = "capital_elected"
	EventTypeNobleTransfer      EventType = "noble_transfer"
	EventTypeDeckDraw           EventType = "deck_draw"
	EventTypeDeckDiscard        EventType = "deck_discard"
	EventTypeDeckRestore        EventType = "deck_restore"
	EventTypeCalamityScheduled  EventType = "calamity_scheduled"
	EventTypeAuguryRevealed     EventType = "augury_revealed"
	EventTypeDeckOrderPlayed    EventType = "deck_order_played"
	EventTypeCalamityApplied    EventType = "calamity_applied"
	EventTypeCalamityCanceled   EventType = "calamity_canceled"
	EventTypeBonusEffect        EventType = "bonus_effect"
	EventTypeNeutralArmy        EventType = "neutral_army_created"
	EventTypeRevoltAppeased     EventType = "revolt_appeased"
	EventTypePlagueDeath        EventType = "plague_noble_death"
	EventTypePlagueSurvived     EventType = "plague_noble_survived"
	EventTypeBadWeatherBlocked  EventType = "bad_weather_blocked"
	EventTypeFamineLoss         EventType = "famine_loss"
	EventTypeBadWeatherLoss     EventType = "bad_weather_loss"
	EventTypeProduction         EventType = "production"
	EventTypeMillProduction     EventType = "mill_production"
	EventTypeIncome             EventType = "income"
	EventTypeConsumption        EventType = "consumption"
	EventTypeCardCanceled       EventType = "card_canceled"
	EventTypeRumor              EventType = "rumor"
	EventTypeFiefFounded        EventType = "fief_founded"
	EventTypeFiefAssigned       EventType = "fief_assigned"
	EventTypeFiefAnnexed        EventType = "fief_annexed"
	EventTypeFiefRankChanged    EventType = "fief_rank_changed"
	EventTypeRoyalAccord        EventType = "royal_accord"
	EventTypeFiefGranted        EventType = "fief_granted"
	EventTypeFiefConquered      EventType = "fief_conquered"
	EventTypeFiefVacated        EventType = "fief_vacated"
	EventTypeFiefDissolved      EventType = "fief_dissolved"
	EventTypeFiefMemberOccupied EventType = "fief_member_occupied"
	// EventTypeFiefTitleSettled reports a usurped fief whose exile died while
	// the usurper lives: Reason "usurper_legitimised" (the usurper is now the
	// official titulaire) or "exile_replaced" (NobleID, a claim heir, is the new
	// exile).
	EventTypeFiefTitleSettled EventType = "fief_title_settled"
	// EventTypeTitleRenounced reports a free winter order giving up a claim
	// (Reason "claim") or an exiled or usurped title (Reason "exiled" or
	// "usurped").
	EventTypeTitleRenounced    EventType = "title_renounced"
	EventTypeFiefAutoAssigned  EventType = "fief_auto_assigned"
	EventTypeProsperityFounded EventType = "prosperity_founded"
	EventTypeMarriage          EventType = "marriage"
	EventTypeMarriageRefused   EventType = "marriage_refused"
	EventTypeMarriageDissolved EventType = "marriage_dissolved"
	EventTypeNobleDraw         EventType = "noble_draw"
	EventTypeDignity           EventType = "dignity"
	EventTypeTrial             EventType = "trial"
	EventTypeTunnel            EventType = "tunnel"
	EventTypeAmbush            EventType = "ambush"
	EventTypeAssassination     EventType = "assassination"
	EventTypeJustice           EventType = "justice"
	// EventTypeAssassinationRevealed is private to its owner, the inquirer: an
	// inquiry taught them who commissioned an assassination (NobleCode: the
	// victim, CommissionerCode: the commissioner).
	EventTypeAssassinationRevealed EventType = "assassination_revealed"
	// EventTypeCalamityVeto is private: no report section lists it.
	EventTypeCalamityVeto EventType = "calamity_veto"
	EventTypeNobleDiscard EventType = "noble_discard"
	EventTypeClaim        EventType = "claim"
	EventTypeEonUnmasked  EventType = "eon_unmasked"
	// EventTypeInquiry is the paid Q order (ResourceSpent), emitted whether or
	// not the target hid a dignity.
	EventTypeInquiry EventType = "inquiry"
	// EventTypeTrialFiled is the private acknowledgement of an accepted J order
	// (NobleID is the target); the verdict, if a second cardinal backs it, is a
	// public EventTypeTrial at the end of the winter.
	EventTypeTrialFiled EventType = "trial_filed"
	// EventTypeDignityRevealed is the public reveal of a hidden dignity by an
	// inquiry (Dignity names it); a chevalier d'Éon emits EventTypeEonUnmasked.
	EventTypeDignityRevealed EventType = "dignity_revealed"
	// EventTypeElectionOpened, EventTypeElectionResult and
	// EventTypeInvestiture carry only public totals, never a ballot.
	EventTypeElectionOpened EventType = "election_opened"
	EventTypeElectionResult EventType = "election_result"
	EventTypeInvestiture    EventType = "investiture"
	// EventTypeRitual is private to the Witch's owner: the declaration of a
	// ritual, then its outcome (Reason "succeeded" or "failed").
	EventTypeRitual EventType = "ritual"
	// EventTypeExcommunication and EventTypeExcommunicationLifted are public
	// papal sanctions (winter stage 1).
	EventTypeExcommunication       EventType = "excommunication"
	EventTypeExcommunicationLifted EventType = "excommunication_lifted"
	// EventTypeCardinalPurchase is the paid N C order; the title itself is
	// conferred at the investiture.
	EventTypeCardinalPurchase EventType = "cardinal_purchase"
	// EventTypeArmyRevolt is the outcome of the revolt cards played on one
	// starving army: Troops is their summed roll, SizeBefore the army's size.
	EventTypeArmyRevolt EventType = "army_revolt"
)

// Outcome is the execution result of one current order.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeInvalid Outcome = "invalid"
	// OutcomeDissolved is a marriage dissolved by the pope (MarriageReport).
	OutcomeDissolved Outcome = "dissolved"
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
	ArmyID  models.ArmyID   `json:"army,omitempty"`
	OwnerID models.PlayerID `json:"owner,omitempty"`
	Force   int             `json:"force"`
	// Supported is the part of Force that comes from the supports the side
	// received; the rest is the army's own strength, fortification and noble.
	Supported  int  `json:"supported,omitempty"`
	NobleBonus int  `json:"nobleBonus,omitempty"`
	Defender   bool `json:"defender"`
	// Tunnel marks an attack carried by a tunnel card: it ignored the
	// fortifications of its target.
	Tunnel bool `json:"tunnel,omitempty"`
	// TunnelForce is, on the defender, the force a tunneled attack faced:
	// Force without the fortification bonus. Set (possibly to 0) only when a
	// tunneled attack contested a fortified territory.
	TunnelForce *int `json:"tunnelForce,omitempty"`
}

// Event is a value-only report of one resolution decision. Fields irrelevant
// to Type are left at their zero value. IDs and forces are copied so events do
// not retain mutable engine state.
type Event struct {
	Type  EventType `json:"type"`
	Phase int       `json:"phase"`

	ArmyID      models.ArmyID        `json:"army,omitempty"`
	OtherArmyID models.ArmyID        `json:"otherArmy,omitempty"`
	ArmyIDs     []models.ArmyID      `json:"armies,omitempty"`
	ChainID     models.ChainID       `json:"chain,omitempty"`
	OrderID     models.OrderID       `json:"order,omitempty"`
	OrderType   models.OrderType     `json:"orderType,omitempty"`
	CardID      models.SpecialCardID `json:"cardId,omitempty"`
	CardKind    models.CardKind      `json:"cardKind,omitempty"`
	RegionSeed  models.TerritoryID   `json:"regionSeed,omitempty"`
	Year        int                  `json:"year,omitempty"`
	Season      models.Season        `json:"season,omitempty"`
	RumorKey    string               `json:"rumorKey,omitempty"`
	RumorLevel  int                  `json:"rumorLevel,omitempty"`
	Outcome     Outcome              `json:"outcome,omitempty"`
	Automatic   bool                 `json:"automatic,omitempty"`
	Reason      string               `json:"reason,omitempty"`
	// Ritual marks a calamity scheduled by a Witch's ritual.
	Ritual      bool        `json:"ritual,omitempty"`
	Progression Progression `json:"progression,omitempty"`

	// ExiledNobleID names the noble a usurpation exiled (fief_conquered,
	// reason "usurpation"), or the fallen usurper of a restoration.
	ExiledNobleID   models.NobleID `json:"exiledNoble,omitempty"`
	ExiledNobleName string         `json:"exiledNobleName,omitempty"`

	TerritoryID       models.TerritoryID `json:"territory,omitempty"`
	SourceID          models.TerritoryID `json:"source,omitempty"`
	TargetID          models.TerritoryID `json:"target,omitempty"`
	DestinationID     models.TerritoryID `json:"destination,omitempty"`
	AttackerOriginID  models.TerritoryID `json:"attackerOrigin,omitempty"`
	BaseDefense       int                `json:"baseDefense,omitempty"`
	Defense           int                `json:"defense,omitempty"`
	CastleBonus       int                `json:"castleBonus,omitempty"`
	Contenders        []CombatContender  `json:"contenders,omitempty"`
	SupporterIDs      []models.ArmyID    `json:"supporters,omitempty"`
	WinnerArmyID      models.ArmyID      `json:"winnerArmy,omitempty"`
	DislodgedArmyID   models.ArmyID      `json:"dislodgedArmy,omitempty"`
	CutSupporterIDs   []models.ArmyID    `json:"cutSupporters,omitempty"`
	Resolved          bool               `json:"resolved,omitempty"`
	RemainingStrength int                `json:"remainingStrength,omitempty"`
	DestinationKind   string             `json:"destinationKind,omitempty"`
	HostArmyID        models.ArmyID      `json:"hostArmy,omitempty"`
	TroopsMerged      int                `json:"troopsMerged,omitempty"`
	SizeBefore        int                `json:"sizeBefore,omitempty"`
	SizeAfter         int                `json:"sizeAfter,omitempty"`

	InfrastructureID   models.InfraID   `json:"infrastructure,omitempty"`
	InfrastructureType models.InfraType `json:"infrastructureType,omitempty"`
	Level              int              `json:"level,omitempty"`
	// PillageEffect is what a pillage did to the infrastructure it reached.
	PillageEffect     PillageEffect              `json:"pillageEffect,omitempty"`
	ResourceCredit    int                        `json:"resourceCredit,omitempty"`
	ResourceAmount    int                        `json:"resourceAmount,omitempty"`
	Partial           bool                       `json:"partial,omitempty"`
	CreditTerritoryID models.TerritoryID         `json:"creditTerritory,omitempty"`
	Production        int                        `json:"production,omitempty"`
	Demand            int                        `json:"demand,omitempty"`
	Rations           map[models.TerritoryID]int `json:"rations,omitempty"`
	StockConsumed     int                        `json:"stockConsumed,omitempty"`
	StockBefore       int                        `json:"stockBefore,omitempty"`
	StockAfter        int                        `json:"stockAfter,omitempty"`
	ResourceSpent     int                        `json:"resourceSpent,omitempty"`
	Cost              int                        `json:"cost,omitempty"`
	Troops            int                        `json:"troops,omitempty"`
	TroopsLost        int                        `json:"troopsLost,omitempty"`
	RationsLost       int                        `json:"rationsLost,omitempty"`
	SavedByPillage    bool                       `json:"savedByPillage,omitempty"`
	Warned            bool                       `json:"warned,omitempty"`
	// SuppliedStrength is the size the rations an army reaches can feed.
	SuppliedStrength int `json:"suppliedStrength,omitempty"`
	// Destroyed marks an army that reached no ration and disappeared.
	Destroyed bool `json:"destroyed,omitempty"`

	TerrainRations       int                        `json:"terrainRations,omitempty"`
	BonusRations         int                        `json:"bonusRations,omitempty"`
	SuppressedRations    int                        `json:"suppressedRations,omitempty"`
	BaseProduction       int                        `json:"baseProduction,omitempty"`
	MillProduction       int                        `json:"millProduction,omitempty"`
	BonusProduction      int                        `json:"bonusProduction,omitempty"`
	SuppressedProduction int                        `json:"suppressedProduction,omitempty"`
	ReceivedLocal        int                        `json:"receivedLocal,omitempty"`
	ReceivedTransfer     int                        `json:"receivedTransfer,omitempty"`
	SentRations          map[models.TerritoryID]int `json:"sentRations,omitempty"`
	TerritoryCount       int                        `json:"territoryCount,omitempty"`
	VillageCount         int                        `json:"villageCount,omitempty"`
	Lost                 bool                       `json:"lost,omitempty"`

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

	// CommissionerCode and CommissionerName identify the noble who commissioned
	// an assassination (PreviousOwnerID is then the victim's owner).
	CommissionerCode models.NobleCode `json:"commissionerCode,omitempty"`
	CommissionerName string           `json:"commissionerName,omitempty"`

	// Dignity is the dignity a EventTypeDignity event confers.
	Dignity models.Dignity `json:"dignity,omitempty"`

	SpouseNobleID   models.NobleID   `json:"spouseNoble,omitempty"`
	SpouseNobleCode models.NobleCode `json:"spouseNobleCode,omitempty"`
	SpouseNobleName string           `json:"spouseNobleName,omitempty"`
	SpouseOwnerID   models.PlayerID  `json:"spouseOwner,omitempty"`

	// Election is the public description of an election, its result or the
	// title an investiture confers.
	Election *ElectionResult `json:"election,omitempty"`

	FiefID          models.FiefID        `json:"fiefId,omitempty"`
	FiefTitle       models.FiefTitle     `json:"fiefTitle,omitempty"`
	FiefTerritories []models.TerritoryID `json:"fiefTerritories,omitempty"`
}
