// Package engine resolves one simultaneous action turn without performing I/O.
//
// Resolve runs five ordered phases: enumerate intentions, calculate explicit
// supports, settle contested territories to a fixed point, execute movements
// and retreats, then progress chains and update territorial control. P1.5 can
// insert supply before these phases, while P2.3 can prepare delivered chains
// before resolution.
package engine
