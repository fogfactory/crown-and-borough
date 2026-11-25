package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// MovementKind is how an army's move reads in a turn report.
type MovementKind string

const (
	// MovementVictory: the army won a contested territory.
	MovementVictory MovementKind = "victory"
	// MovementMove: nobody opposed the army on its destination.
	MovementMove MovementKind = "move"
	// MovementFailure: the attack was repelled or the order could not be carried out.
	MovementFailure MovementKind = "failure"
)

// MovementLeg is one step of an army's move; From is empty when unknown.
type MovementLeg struct {
	From models.TerritoryID `json:"from,omitempty"`
	To   models.TerritoryID `json:"to"`
}

// ArmyMovement summarizes what one army did this turn when it attacked or
// maneuvered: the report reads it instead of joining combats and orders.
type ArmyMovement struct {
	Kind  MovementKind    `json:"kind"`
	Army  models.ArmyID   `json:"army"`
	Owner models.PlayerID `json:"owner"`
	Legs  []MovementLeg   `json:"legs"`
	// Combat is the territory of the contested fight behind the move, empty
	// for a move that nobody opposed.
	Combat models.TerritoryID `json:"combat,omitempty"`
	// Chain is the chain of the order behind the move, empty when the army
	// only appears as a contender.
	Chain  models.ChainID `json:"chain,omitempty"`
	Reason string         `json:"reason,omitempty"`
}

// buildMovements lists the armies that moved or tried to: every attacker of a
// contested combat, then the attack and maneuver orders no combat covered. An
// attack on an empty, unfortified territory is a plain move, never a combat.
func buildMovements(combats []CombatReport, orders []OrderReport) []ArmyMovement {
	movements := []ArmyMovement{}
	orderOf := make(map[models.ArmyID]OrderReport, len(orders))
	for _, order := range orders {
		if _, seen := orderOf[order.Army]; !seen && order.Army != "" && order.Source != "" {
			orderOf[order.Army] = order
		}
	}
	covered := map[models.ArmyID]bool{}
	for _, combat := range combats {
		if !combat.Contested {
			continue
		}
		for _, contender := range combat.Contenders {
			if contender.Defender || contender.ArmyID == "" {
				continue
			}
			order := orderOf[contender.ArmyID]
			movement := ArmyMovement{
				Kind: MovementVictory, Army: contender.ArmyID, Owner: contender.OwnerID,
				Legs:   []MovementLeg{{From: order.Source, To: combat.Territory}},
				Combat: combat.Territory, Chain: order.Chain,
			}
			if combat.Winner != contender.ArmyID {
				movement.Kind = MovementFailure
				movement.Reason = combat.Reason
			}
			covered[contender.ArmyID] = true
			movements = append(movements, movement)
		}
	}
	for _, order := range orders {
		if order.Army == "" || covered[order.Army] {
			continue
		}
		if order.Type != models.OrderTypeAttack && order.Type != models.OrderTypeManeuver {
			continue
		}
		targets := order.Targets
		if len(targets) == 0 && order.Target != "" {
			targets = []models.TerritoryID{order.Target}
		}
		if len(targets) == 0 {
			continue
		}
		movement := ArmyMovement{Kind: MovementMove, Army: order.Army, Owner: order.Owner, Chain: order.Chain}
		for _, target := range targets {
			movement.Legs = append(movement.Legs, MovementLeg{From: order.Source, To: target})
		}
		if order.Outcome != OutcomeSuccess {
			movement.Kind = MovementFailure
			movement.Reason = order.Reason
		}
		covered[order.Army] = true
		movements = append(movements, movement)
	}
	return movements
}
