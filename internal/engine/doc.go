// Package engine resolves one simultaneous action turn without performing I/O.
//
// Resolve runs phase 0 supply from start-of-turn positions, then five ordered
// action phases: enumerate intentions, calculate explicit supports, adjudicate
// attacks, joins and dispersions as a graph of decisions (see adjudicator),
// execute movements and retreats, then progress chains and update territorial
// control. Chains are already attached to armies before resolution.
package engine
