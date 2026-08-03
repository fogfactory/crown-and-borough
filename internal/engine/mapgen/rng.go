package mapgen

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand/v2"
)

// newRNG derives an independent deterministic stream for one generation phase.
func newRNG(seed, phase string) *rand.Rand {
	digest := sha256.Sum256([]byte(seed + "|" + phase))
	lo := binary.BigEndian.Uint64(digest[:8])
	hi := binary.BigEndian.Uint64(digest[8:16])
	return rand.New(rand.NewPCG(lo, hi))
}

func shuffle[T any](rng *rand.Rand, values []T) {
	for i := len(values) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		values[i], values[j] = values[j], values[i]
	}
}
