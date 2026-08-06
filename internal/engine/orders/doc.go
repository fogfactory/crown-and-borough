// Package orders parses, validates, and receives order chains. Parsing turns
// source text into an unassigned chain; static validation checks references and
// intrinsic order rules; reception checks the current game state and atomically
// attaches a valid chain to an army. Execution-time conditions, simultaneous
// resolution, and chain progression belong to engine.Resolve.
package orders
