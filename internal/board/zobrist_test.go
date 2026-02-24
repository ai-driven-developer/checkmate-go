package board

import "testing"

func TestZobristTablesInitialized(t *testing.T) {
	// All Zobrist values should be non-zero (with overwhelming probability).
	zeroCount := 0
	for c := 0; c < 2; c++ {
		for p := 0; p < 7; p++ {
			for sq := 0; sq < 64; sq++ {
				if ZobristPiece[c][p][sq] == 0 {
					zeroCount++
				}
			}
		}
	}
	if zeroCount > 2 {
		t.Errorf("too many zero Zobrist piece keys: %d", zeroCount)
	}
	if ZobristSideToMove == 0 {
		t.Error("ZobristSideToMove should not be zero")
	}
}

func TestZobristUniqueness(t *testing.T) {
	// Spot check that a few specific keys are distinct.
	keys := []uint64{
		ZobristPiece[White][Pawn][E2],
		ZobristPiece[White][Pawn][E4],
		ZobristPiece[Black][Pawn][E7],
		ZobristPiece[White][King][E1],
		ZobristCastling[AllCastling],
		ZobristSideToMove,
	}
	seen := make(map[uint64]bool)
	for _, k := range keys {
		if seen[k] {
			t.Errorf("duplicate Zobrist key: %x", k)
		}
		seen[k] = true
	}
}

func TestZobristDifferentPositions(t *testing.T) {
	p1 := NewPosition()
	p2 := &Position{}
	_ = p2.SetFromFEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")

	if p1.Hash == p2.Hash {
		t.Error("different positions should have different hashes")
	}
}
