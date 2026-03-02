package eval

import "checkmatego/internal/board"

// --- Knight outpost ---

// outpostBonusMG/EG: bonus for a knight on an outpost square
// (rank 4-6 for White, rank 3-5 for Black), supported by a friendly pawn
// and not attackable by enemy pawns on adjacent files.
const (
	outpostBonusMG = 20
	outpostBonusEG = 15
)

// outpostRanks is the set of ranks where an outpost is relevant.
// For White: ranks 4-6 (indices 3-5). For Black: ranks 3-5 (indices 2-4).
var outpostRanksWhite = board.Rank4BB | board.Rank5BB | board.Rank6BB
var outpostRanksBlack = board.Rank3BB | board.Rank4BB | board.Rank5BB

// outpostScore evaluates knight outposts for both sides.
// Returns middlegame and endgame scores from White's perspective.
func outpostScore(pos *board.Position) (mg, eg int) {
	whitePawns := pos.ColorPieces(board.White, board.Pawn)
	blackPawns := pos.ColorPieces(board.Black, board.Pawn)

	// White pawn attacks: squares controlled by white pawns.
	whitePawnAttacks := whitePawns.NorthWest() | whitePawns.NorthEast()

	// White knights on outpost squares.
	bb := pos.ColorPieces(board.White, board.Knight)
	for bb != 0 {
		sq := bb.PopLSB()
		sqBB := board.SquareBB(sq)
		// Must be on an outpost rank.
		if sqBB&outpostRanksWhite == 0 {
			continue
		}
		// Must be supported by a friendly pawn.
		if sqBB&whitePawnAttacks == 0 {
			continue
		}
		// Must not be attackable by enemy pawns on adjacent files ahead.
		file := sq.File()
		if adjacentFileMask[file]&whitePassedMask[sq]&blackPawns != 0 {
			// Enemy pawn can advance to attack this square.
			// Use a stricter check: only squares ahead on adjacent files.
			// whitePassedMask already covers ranks above, so filter to
			// just the enemy pawns that could attack this square.
			// Actually, we want: no enemy pawns on adjacent files at
			// ranks >= this rank (they could move down to attack).
			continue
		}
		mg += outpostBonusMG
		eg += outpostBonusEG
	}

	// Black pawn attacks.
	blackPawnAttacks := blackPawns.SouthWest() | blackPawns.SouthEast()

	// Black knights on outpost squares.
	bb = pos.ColorPieces(board.Black, board.Knight)
	for bb != 0 {
		sq := bb.PopLSB()
		sqBB := board.SquareBB(sq)
		if sqBB&outpostRanksBlack == 0 {
			continue
		}
		if sqBB&blackPawnAttacks == 0 {
			continue
		}
		file := sq.File()
		if adjacentFileMask[file]&blackPassedMask[sq]&whitePawns != 0 {
			continue
		}
		mg -= outpostBonusMG
		eg -= outpostBonusEG
	}

	return mg, eg
}

// --- Rook evaluation ---

const (
	rookOpenFileMG     = 20
	rookOpenFileEG     = 10
	rookSemiOpenFileMG = 10
	rookSemiOpenFileEG = 5
	rookSeventhRankMG  = 20
	rookSeventhRankEG  = 30
)

// rookScore evaluates rook placement bonuses.
// Returns middlegame and endgame scores from White's perspective.
func rookScore(pos *board.Position) (mg, eg int) {
	whitePawns := pos.ColorPieces(board.White, board.Pawn)
	blackPawns := pos.ColorPieces(board.Black, board.Pawn)

	// White rooks.
	bb := pos.ColorPieces(board.White, board.Rook)
	for bb != 0 {
		sq := bb.PopLSB()
		file := sq.File()
		fMask := fileMask[file]

		// Open file: no pawns of either color.
		if fMask&(whitePawns|blackPawns) == 0 {
			mg += rookOpenFileMG
			eg += rookOpenFileEG
		} else if fMask&whitePawns == 0 {
			// Semi-open file: no friendly pawns.
			mg += rookSemiOpenFileMG
			eg += rookSemiOpenFileEG
		}

		// Rook on 7th rank (rank index 6 for White).
		if sq.Rank() == 6 {
			mg += rookSeventhRankMG
			eg += rookSeventhRankEG
		}
	}

	// Black rooks.
	bb = pos.ColorPieces(board.Black, board.Rook)
	for bb != 0 {
		sq := bb.PopLSB()
		file := sq.File()
		fMask := fileMask[file]

		if fMask&(whitePawns|blackPawns) == 0 {
			mg -= rookOpenFileMG
			eg -= rookOpenFileEG
		} else if fMask&blackPawns == 0 {
			mg -= rookSemiOpenFileMG
			eg -= rookSemiOpenFileEG
		}

		// Rook on 2nd rank (rank index 1 for Black = their 7th).
		if sq.Rank() == 1 {
			mg -= rookSeventhRankMG
			eg -= rookSeventhRankEG
		}
	}

	return mg, eg
}

// --- King-passer distance ---

// In the endgame, king proximity to passed pawns matters:
// - Friendly king close to own passed pawn = bonus (can escort)
// - Enemy king far from passed pawn = bonus (can't stop it)
const (
	friendlyKingPasserBonus = 5 // per square closer than 4
	enemyKingPasserBonus    = 3 // per square farther than 2
)

// chebyshevDistance returns the Chebyshev (king-move) distance between two squares.
func chebyshevDistance(a, b board.Square) int {
	dr := a.Rank() - b.Rank()
	if dr < 0 {
		dr = -dr
	}
	df := a.File() - b.File()
	if df < 0 {
		df = -df
	}
	if dr > df {
		return dr
	}
	return df
}

// kingPasserDistanceScore evaluates king proximity to passed pawns.
// This is primarily an endgame term.
// Returns middlegame and endgame scores from White's perspective.
func kingPasserDistanceScore(pos *board.Position) (mg, eg int) {
	whitePawns := pos.ColorPieces(board.White, board.Pawn)
	blackPawns := pos.ColorPieces(board.Black, board.Pawn)
	wKing := pos.KingSquare(board.White)
	bKing := pos.KingSquare(board.Black)

	// White passed pawns.
	bb := whitePawns
	for bb != 0 {
		sq := bb.PopLSB()
		if whitePassedMask[sq]&blackPawns != 0 {
			continue // not passed
		}
		// Friendly king close = good.
		friendlyDist := chebyshevDistance(wKing, sq)
		if friendlyDist < 4 {
			eg += (4 - friendlyDist) * friendlyKingPasserBonus
		}
		// Enemy king far = good.
		enemyDist := chebyshevDistance(bKing, sq)
		if enemyDist > 2 {
			eg += (enemyDist - 2) * enemyKingPasserBonus
		}
	}

	// Black passed pawns.
	bb = blackPawns
	for bb != 0 {
		sq := bb.PopLSB()
		if blackPassedMask[sq]&whitePawns != 0 {
			continue // not passed
		}
		friendlyDist := chebyshevDistance(bKing, sq)
		if friendlyDist < 4 {
			eg -= (4 - friendlyDist) * friendlyKingPasserBonus
		}
		enemyDist := chebyshevDistance(wKing, sq)
		if enemyDist > 2 {
			eg -= (enemyDist - 2) * enemyKingPasserBonus
		}
	}

	return mg, eg
}

