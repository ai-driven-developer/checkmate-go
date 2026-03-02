package eval

// PawnEntry stores cached pawn structure and passed pawn evaluation.
type PawnEntry struct {
	key uint64
	mg  int16
	eg  int16
}

// PawnCache is a fixed-size hash table for pawn evaluation results.
// Keyed by Position.PawnHash, it avoids recomputing pawn structure and
// passed pawn scores when the pawn configuration hasn't changed (>95%
// hit rate in typical games). Each worker owns its own PawnCache.
type PawnCache struct {
	entries []PawnEntry
	mask    uint64
}

// NewPawnCache creates a pawn cache with the given number of entries.
// Size is rounded down to a power of two for fast modulo via mask.
func NewPawnCache(size int) *PawnCache {
	if size < 1024 {
		size = 1024
	}
	n := roundDownPow2(uint64(size))
	return &PawnCache{
		entries: make([]PawnEntry, n),
		mask:    n - 1,
	}
}

// Probe looks up a pawn hash in the cache.
func (pc *PawnCache) Probe(key uint64) (hit bool, mg, eg int16) {
	e := &pc.entries[key&pc.mask]
	if e.key == key {
		return true, e.mg, e.eg
	}
	return false, 0, 0
}

// Store saves a pawn evaluation result.
func (pc *PawnCache) Store(key uint64, mg, eg int16) {
	e := &pc.entries[key&pc.mask]
	e.key = key
	e.mg = mg
	e.eg = eg
}

func roundDownPow2(n uint64) uint64 {
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return (n >> 1) + 1
}
