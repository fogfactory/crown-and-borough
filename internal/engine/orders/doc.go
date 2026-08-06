// Package orders parses, validates, and receives order chains. Parsing turns
// source text into an unassigned chain; static validation checks references and
// intrinsic order rules; reception checks the current game state and atomically
// attaches a valid chain to an army. P1.4 owns execution-time conditions.
//
// P1.4 will document and implement the following simultaneous resolution
// contract:
//
//	Resolve(game *models.GameState) (Resolution, error)
//
// It will consume every active chain in one pass, report success, failure, or
// invalid outcomes per order, and then apply single or loop progression.
package orders
