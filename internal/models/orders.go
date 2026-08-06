package models

// ChainID identifies a stored order chain. IDs are allocated by reception and
// remain stable while the chain exists.
type ChainID string

// OrderID identifies an order within one chain. The parser assigns local IDs
// in source order.
type OrderID string

// TerritoryCode preserves a territory trigram in syntax-derived data such as
// dispersion noble assignments.
type TerritoryCode string

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
	OrderTypeHostage  OrderType = "hostage"
	OrderTypeDungeon  OrderType = "dungeon"
)

// IsValid reports whether the order type is known to the command model.
func (t OrderType) IsValid() bool {
	switch t {
	case OrderTypeAttack, OrderTypeSupport, OrderTypeHold, OrderTypeJoin,
		OrderTypePillage, OrderTypeDisperse, OrderTypeHostage, OrderTypeDungeon:
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
	// NobleTargetIDs come from the noble code in O or K orders. Validation and
	// P1.4 resolution use this separate target domain.
	NobleTargetIDs []NobleID `json:"nobleTargets"`
	// NobleAssignments comes from D-order asterisks and is consumed by reception
	// coverage checks and P1.4 dispersion resolution.
	NobleAssignments map[TerritoryCode][]NobleCode `json:"nobleAssignments"`
	// Liaison comes from parentheses around an order line and is consumed by P1.4
	// progression after the order outcome is known.
	Liaison LiaisonMode `json:"liaison"`
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
}
