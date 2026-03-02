package board

// Zobrist hashing keys for incremental position hashing.
var (
	ZobristPiece      [2][7][64]uint64
	ZobristCastling   [16]uint64
	ZobristEnPassant  [65]uint64 // index 64 = NoSquare (no en passant)
	ZobristSideToMove uint64
	ZobristPawn       [2][64]uint64 // separate keys for pawn-only hash
)

func init() {
	// Use a simple xorshift64 PRNG with a fixed seed for reproducibility.
	state := uint64(0x12345678DEADBEEF)
	next := func() uint64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return state
	}

	for color := 0; color < 2; color++ {
		for piece := 0; piece < 7; piece++ {
			for sq := 0; sq < 64; sq++ {
				ZobristPiece[color][piece][sq] = next()
			}
		}
	}
	for i := 0; i < 16; i++ {
		ZobristCastling[i] = next()
	}
	for i := 0; i < 65; i++ {
		ZobristEnPassant[i] = next()
	}
	ZobristSideToMove = next()
	for color := 0; color < 2; color++ {
		for sq := 0; sq < 64; sq++ {
			ZobristPawn[color][sq] = next()
		}
	}
}
