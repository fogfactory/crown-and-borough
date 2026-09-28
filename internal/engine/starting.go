package engine

import "github.com/fogfactory/crown-and-borough/internal/engine/mapgen"

// MinimumStartingDistance is the minimum graph distance between two starting
// territories. Adjacent territories have distance 1. Starting-position
// selection itself lives in mapgen, alongside the rest of map generation;
// this re-export lets engine code and tests reason about the constraint
// without importing mapgen directly.
const MinimumStartingDistance = mapgen.MinimumStartingDistance
