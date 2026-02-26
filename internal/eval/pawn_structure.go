package eval

import "checkmatego/internal/board"

// Pawn structure penalties (centipawns), indexed as [mg, eg].
var (
	doubledPenaltyMG  = 10
	doubledPenaltyEG  = 20
	isolatedPenaltyMG = 12
	isolatedPenaltyEG = 18
	backwardPenaltyMG = 8
	backwardPenaltyEG = 12
)

// whiteBackwardMask[sq] holds squares on adjacent files at the same rank or
// below where a friendly pawn could support the pawn on sq.
var whiteBackwardMask [64]board.Bitboard

// blackBackwardMask[sq] is the same for Black (looking upward).
var blackBackwardMask [64]board.Bitboard

func init() {
	for sq := board.Square(0); sq < 64; sq++ {
		file := sq.File()
		rank := sq.Rank()
		adj := adjacentFileMask[file]

		// White: adjacent files, same rank and all ranks below.
		var wMask board.Bitboard
		for r := 0; r <= rank; r++ {
			wMask |= board.Bitboard(0xFF) << (r * 8)
		}
		whiteBackwardMask[sq] = adj & wMask

		// Black: adjacent files, same rank and all ranks above.
		var bMask board.Bitboard
		for r := rank; r < 8; r++ {
			bMask |= board.Bitboard(0xFF) << (r * 8)
		}
		blackBackwardMask[sq] = adj & bMask
	}
}

// pawnStructureScore evaluates doubled, isolated, and backward pawns.
// Returns middlegame and endgame scores from White's perspective.
func pawnStructureScore(pos *board.Position) (mg, eg int) {
	whitePawns := pos.ColorPieces(board.White, board.Pawn)
	blackPawns := pos.ColorPieces(board.Black, board.Pawn)

	// --- Doubled pawns ---
	// Penalize each extra pawn on the same file.
	for f := 0; f < 8; f++ {
		wCount := (whitePawns & fileMask[f]).Count()
		if wCount > 1 {
			mg -= (wCount - 1) * doubledPenaltyMG
			eg -= (wCount - 1) * doubledPenaltyEG
		}
		bCount := (blackPawns & fileMask[f]).Count()
		if bCount > 1 {
			mg += (bCount - 1) * doubledPenaltyMG
			eg += (bCount - 1) * doubledPenaltyEG
		}
	}

	// --- Isolated pawns ---
	// A pawn with no friendly pawns on adjacent files.
	bb := whitePawns
	for bb != 0 {
		sq := bb.PopLSB()
		file := sq.File()
		if adjacentFileMask[file]&whitePawns == 0 {
			mg -= isolatedPenaltyMG
			eg -= isolatedPenaltyEG
		}
	}

	bb = blackPawns
	for bb != 0 {
		sq := bb.PopLSB()
		file := sq.File()
		if adjacentFileMask[file]&blackPawns == 0 {
			mg += isolatedPenaltyMG
			eg += isolatedPenaltyEG
		}
	}

	// --- Backward pawns ---
	// A pawn that is not isolated, has no friendly pawn support behind it on
	// adjacent files, and whose stop square is controlled by an enemy pawn.
	blackPawnAttacks := blackPawns.SouthWest() | blackPawns.SouthEast()

	bb = whitePawns
	for bb != 0 {
		sq := bb.PopLSB()
		file := sq.File()
		// Skip isolated pawns (already penalized).
		if adjacentFileMask[file]&whitePawns == 0 {
			continue
		}
		// No friendly support on adjacent files at or behind this rank.
		if whiteBackwardMask[sq]&whitePawns == 0 {
			stopSq := sq + 8
			if stopSq < 64 && board.SquareBB(stopSq)&blackPawnAttacks != 0 {
				mg -= backwardPenaltyMG
				eg -= backwardPenaltyEG
			}
		}
	}

	whitePawnAttacks := whitePawns.NorthWest() | whitePawns.NorthEast()

	bb = blackPawns
	for bb != 0 {
		sq := bb.PopLSB()
		file := sq.File()
		if adjacentFileMask[file]&blackPawns == 0 {
			continue
		}
		if blackBackwardMask[sq]&blackPawns == 0 {
			stopSq := sq - 8
			if stopSq < 64 && board.SquareBB(stopSq)&whitePawnAttacks != 0 {
				mg += backwardPenaltyMG
				eg += backwardPenaltyEG
			}
		}
	}

	return mg, eg
}
