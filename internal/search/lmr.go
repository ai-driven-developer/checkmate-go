package search

import "math"

// lmrReductions is a precomputed table of depth reductions for Late Move Reduction.
// Indexed by [depth][moveIndex].
var lmrReductions [MaxDepth + 1][64]int

func init() {
	for depth := 1; depth <= MaxDepth; depth++ {
		for moves := 1; moves < 64; moves++ {
			lmrReductions[depth][moves] = int(math.Log(float64(depth)) * math.Log(float64(moves)) / 2.25)
		}
	}
}
