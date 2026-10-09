package models

// ChainID identifies a stored order chain. IDs are allocated by reception and
// remain stable while the chain exists.
type ChainID string

// OrderID identifies an order within one chain. The parser assigns local IDs
// in source order.
type OrderID string

// NobleCode preserves a noble trigram in syntax-derived dispersion assignments.
// The value "*" represents every remaining noble.
type NobleCode string

// OrderType identifies the operation parsed from an order symbol and consumed
// by static validation and P1.4 resolution.
type OrderType string

const (
	OrderTypeAttack   OrderType = "attack"
	OrderTypeSupport  OrderType = "support"
	OrderTypeHold     OrderType = "hold"
	OrderTypeJoin     OrderType = "join"
	OrderTypePillage  OrderType = "pillage"
	OrderTypeDisperse OrderType = "disperse"
	OrderTypeTransfer OrderType = "transfer"
)

// IsValid reports whether the order type is known to the command model.
func (t OrderType) IsValid() bool {
	switch t {
	case OrderTypeAttack, OrderTypeSupport, OrderTypeHold, OrderTypeJoin,
		OrderTypePillage, OrderTypeDisperse, OrderTypeTransfer:
		return true
	}
	return false
}

// WinterOrderType identifies an immediate winter management instruction.
type WinterOrderType string

const (
	WinterOrderTypeRecruitNoble        WinterOrderType = "recruit_noble"
	WinterOrderTypeRecruitTroop        WinterOrderType = "recruit_troop"
	WinterOrderTypeBuild               WinterOrderType = "build"
	WinterOrderTypeElectCapital        WinterOrderType = "elect_capital"
	WinterOrderTypeTransferNoble       WinterOrderType = "transfer_noble"
	WinterOrderTypeHostage             WinterOrderType = "hostage"
	WinterOrderTypeDungeon             WinterOrderType = "dungeon"
	WinterOrderTypeTransfer            WinterOrderType = "transfer"
	WinterOrderTypeFoundFief           WinterOrderType = "found_fief"
	WinterOrderTypeAssignFief          WinterOrderType = "assign_fief"
	WinterOrderTypeMarriage            WinterOrderType = "marriage"
	WinterOrderTypeDrawNoble           WinterOrderType = "draw_noble"
	WinterOrderTypeDignity             WinterOrderType = "play_dignity"
	WinterOrderTypeDiscardNoble        WinterOrderType = "discard_noble_card"
	WinterOrderTypeClaim               WinterOrderType = "claim"
	WinterOrderTypeCalamityVeto        WinterOrderType = "calamity_veto"
	WinterOrderTypeCandidacy           WinterOrderType = "candidacy"
	WinterOrderTypeVote                WinterOrderType = "vote"
	WinterOrderTypeExcommunicate       WinterOrderType = "excommunicate"
	WinterOrderTypeLiftExcommunication WinterOrderType = "lift_excommunication"
)

// IsValid reports whether a winter order type is known to the winter resolver.
func (t WinterOrderType) IsValid() bool {
	switch t {
	case WinterOrderTypeRecruitNoble, WinterOrderTypeRecruitTroop, WinterOrderTypeBuild,
		WinterOrderTypeElectCapital, WinterOrderTypeTransferNoble,
		WinterOrderTypeHostage, WinterOrderTypeDungeon, WinterOrderTypeTransfer,
		WinterOrderTypeFoundFief, WinterOrderTypeAssignFief, WinterOrderTypeMarriage,
		WinterOrderTypeDrawNoble, WinterOrderTypeDignity, WinterOrderTypeDiscardNoble,
		WinterOrderTypeClaim, WinterOrderTypeCalamityVeto,
		WinterOrderTypeCandidacy, WinterOrderTypeVote,
		WinterOrderTypeExcommunicate, WinterOrderTypeLiftExcommunication:
		return true
	}
	return false
}

// LiaisonMode controls P1.4 progression after an order succeeds or fails.
type LiaisonMode string

const (
	LiaisonModeSingle LiaisonMode = "single"
	LiaisonModeLoop   LiaisonMode = "loop"
)

// IsValid reports whether the liaison mode is known to the command model.
func (m LiaisonMode) IsValid() bool {
	return m == LiaisonModeSingle || m == LiaisonModeLoop
}

// Order is one syntax-derived instruction in an order chain.
type Order struct {
	// ID is assigned by the parser in source order and is consumed by validation,
	// state projection, and P1.4 resolution.
	ID OrderID `json:"id"`
	// Type is parsed from the order symbol and is consumed by validation and P1.4
	// resolution.
	Type OrderType `json:"type"`
	// ArmyID is empty after parsing and set by reception for the receiving army.
	// It lets GameState validation and P1.4 resolution address the applied army.
	ArmyID ArmyID `json:"army,omitempty"`
	// PositionID comes from the explicit territory code in the order line and is
	// required by validation and P1.4 resolution.
	PositionID TerritoryID `json:"position"`
	// TargetIDs come from territory codes in A, S, J, and D orders. Validation
	// checks them and P1.4 resolution consumes them.
	TargetIDs []TerritoryID `json:"targets"`
	// NobleAssignments comes from D-order asterisks and is consumed by reception
	// coverage checks and P1.4 dispersion resolution.
	NobleAssignments map[TerritoryID][]NobleCode `json:"nobleAssignments"`
	// Liaison comes from parentheses around an order line and is consumed by P1.4
	// progression after the order outcome is known.
	Liaison LiaisonMode `json:"liaison"`
	// Amount is used only by transfer orders and is expressed in resources.
	Amount int `json:"amount,omitempty"`
	// Line is the source line of a freshly parsed order, used to locate
	// submission errors. It is not persisted.
	Line int `json:"-"`
}

// WinterOrder is one direct winter management instruction. Fields irrelevant
// to Type are left at their zero value. For WinterOrderTypeFoundFief,
// TerritoryID holds the capital (also TerritoryIDs[0]) and TerritoryIDs holds
// the whole group in source order; NobleCode is the titleholder. For
// WinterOrderTypeAssignFief, TerritoryID is the fief's capital and NobleCode
// is the noble it is attributed to. For WinterOrderTypeMarriage, NobleCode is
// the player's own noble and SpouseCode the other player's noble it asks to
// marry (specs/succession.md § Conclusion d'un mariage). For
// WinterOrderTypeRecruitNoble, CardCode is the noble card played from the
// hand and TerritoryID the castle or village where the noble appears. For
// WinterOrderTypeDignity, NobleCode is the player's own noble and CardCode
// the dignity card played on it. For WinterOrderTypeDiscardNoble, CardCode is
// the code (noble trigram or dignity code) of the noble-hand card discarded
// unplayed. WinterOrderTypeDrawNoble carries no field. For
// WinterOrderTypeClaim, NobleCode is the player's own noble, the heir, and
// SpouseCode the noble of another player whose titles it claims. For
// WinterOrderTypeCandidacy and WinterOrderTypeVote, Election names the
// election, NobleCode the candidate and TerritoryID the seed village of the
// bishopric (bishop elections only). For WinterOrderTypeExcommunicate and
// WinterOrderTypeLiftExcommunication, NobleCode is the targeted noble (papal
// orders, resolved first: specs/religieux.md § Excommunication).
type WinterOrder struct {
	ID           OrderID         `json:"id"`
	Type         WinterOrderType `json:"type"`
	TerritoryID  TerritoryID     `json:"territory,omitempty"`
	SourceID     TerritoryID     `json:"source,omitempty"`
	TargetID     TerritoryID     `json:"target,omitempty"`
	TerritoryIDs []TerritoryID   `json:"territories,omitempty"`
	Amount       int             `json:"amount,omitempty"`
	InfraType    InfraType       `json:"infrastructureType,omitempty"`
	NobleCode    NobleCode       `json:"nobleCode,omitempty"`
	SpouseCode   NobleCode       `json:"spouseCode,omitempty"`
	CardCode     string          `json:"cardCode,omitempty"`
	// Election is the election a candidacy or a vote belongs to.
	Election ElectionKind `json:"election,omitempty"`
	// Status is the optional status a noble transfer gives a noble handed to
	// another owner's army (hostage or dungeon); empty keeps its status.
	Status NobleStatus `json:"status,omitempty"`
	// Indices are the 1-based positions in the astrologer forecast a
	// calamity veto removes.
	Indices []int `json:"indices,omitempty"`
}

type DeckOrderType string

const (
	DeckOrderTypeDiscard DeckOrderType = "discard_card"
	DeckOrderTypePlay    DeckOrderType = "play_card"
)

type DeckOrder struct {
	ID                OrderID       `json:"id"`
	Type              DeckOrderType `json:"type"`
	Kind              CardKind      `json:"kind,omitempty"`
	RegionSeed        TerritoryID   `json:"regionSeed,omitempty"`
	TargetTerritoryID TerritoryID   `json:"targetTerritory,omitempty"`
	// TargetNobleID is the noble a trial card puts on trial.
	TargetNobleID NobleID `json:"targetNoble,omitempty"`
}

func (t DeckOrderType) IsValid() bool {
	return t == DeckOrderTypeDiscard || t == DeckOrderTypePlay
}

// PendingDisperse records unresolved branches of a looped dispersion after
// completed branches have already left the source army. The command chain stays
// with its original carrier while the residual army retries these branches.
type PendingDisperse struct {
	ArmyID           ArmyID                      `json:"army"`
	SourceID         TerritoryID                 `json:"source"`
	TargetIDs        []TerritoryID               `json:"targets"`
	NobleAssignments map[TerritoryID][]NobleCode `json:"nobleAssignments"`
}

// Chain is a received sequence of orders carried by one army.
type Chain struct {
	// ID is allocated by reception and remains stable while the chain exists.
	ID ChainID `json:"id,omitempty"`
	// NobleID comes from the parsed header and identifies the noble that emitted
	// this chain.
	NobleID NobleID `json:"noble"`
	// ArmyID is empty after parsing and set by reception for the army carrying
	// this chain.
	ArmyID ArmyID `json:"army,omitempty"`
	// Orders are parsed in source order and progressed by CurrentIndex in P1.4.
	Orders []Order `json:"orders"`
	// CurrentIndex is set to zero by reception and advanced by P1.4 resolution.
	CurrentIndex int `json:"currentIndex"`
	// PendingDisperse is set only while a looped D retries its unresolved
	// residual branches. It is internal resolution state, not source syntax.
	PendingDisperse *PendingDisperse `json:"pendingDisperse,omitempty"`
}
