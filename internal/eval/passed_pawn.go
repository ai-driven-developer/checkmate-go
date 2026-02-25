package eval

import "checkmatego/internal/board"

// Passed pawn bonuses by rank, indexed from White's perspective (rank 0-7).
// Rank 0 and 1 are never relevant (pawn starts on rank 2 at earliest).
// Bonuses increase sharply as the pawn advances closer to promotion.
var passedPawnBonus = [8]int{
	0, 0, 10, 20, 40, 60, 100, 0, // rank 8 = promotion, never a pawn there
}

// Endgame bonuses are higher because passed pawns are more dangerous in endgames.
var passedPawnBonusEG = [8]int{
	0, 0, 15, 30, 60, 100, 150, 0,
}

// fileMask contains a bitboard for each file (0-7).
var fileMask [8]board.Bitboard

// adjacentFileMask contains the union of file masks for adjacent files.
var adjacentFileMask [8]board.Bitboard

// whitePassedMask[sq] contains the squares where an enemy pawn would block
// or be able to capture a White pawn on sq. It spans the same file and
// adjacent files, from the rank above sq up to rank 8.
var whitePassedMask [64]board.Bitboard

// blackPassedMask[sq] is the same for Black (looking downward).
var blackPassedMask [64]board.Bitboard

func init() {
	// Build file masks.
	for f := 0; f < 8; f++ {
		fileMask[f] = board.FileABB << f
	}

	// Build adjacent file masks.
	for f := 0; f < 8; f++ {
		if f > 0 {
			adjacentFileMask[f] |= fileMask[f-1]
		}
		if f < 7 {
			adjacentFileMask[f] |= fileMask[f+1]
		}
	}

	// Build passed pawn masks.
	for sq := board.Square(0); sq < 64; sq++ {
		file := sq.File()
		rank := sq.Rank()
		files := fileMask[file] | adjacentFileMask[file]

		// White: mask all ranks above this square.
		var wMask board.Bitboard
		for r := rank + 1; r < 8; r++ {
			wMask |= board.Bitboard(0xFF) << (r * 8)
		}
		whitePassedMask[sq] = files & wMask

		// Black: mask all ranks below this square.
		var bMask board.Bitboard
		for r := rank - 1; r >= 0; r-- {
			bMask |= board.Bitboard(0xFF) << (r * 8)
		}
		blackPassedMask[sq] = files & bMask
	}
}

// passedPawnScore returns middlegame and endgame passed pawn bonuses
// from White's perspective.
func passedPawnScore(pos *board.Position) (mg, eg int) {
	whitePawns := pos.ColorPieces(board.White, board.Pawn)
	blackPawns := pos.ColorPieces(board.Black, board.Pawn)

	// White passed pawns.
	bb := whitePawns
	for bb != 0 {
		sq := bb.PopLSB()
		if whitePassedMask[sq]&blackPawns == 0 {
			rank := sq.Rank()
			mg += passedPawnBonus[rank]
			eg += passedPawnBonusEG[rank]
		}
	}

	// Black passed pawns.
	bb = blackPawns
	for bb != 0 {
		sq := bb.PopLSB()
		if blackPassedMask[sq]&whitePawns == 0 {
			rank := 7 - sq.Rank() // mirror rank for bonus lookup
			mg -= passedPawnBonus[rank]
			eg -= passedPawnBonusEG[rank]
		}
	}

	return mg, eg
}
