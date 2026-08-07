// Package orders parses, validates, and receives order chains. Parsing turns
// source text into an unassigned chain; static validation checks references and
// intrinsic order rules; reception checks the current game state and atomically
// attaches a valid chain to an army. Static adjacency diagnostics are
// deferrable: reception keeps the chain and engine.Resolve invalidates the
// offending order at execution, breaking the chain there. Execution-time
// conditions, simultaneous resolution, and chain progression belong to
// engine.Resolve.
package orders
