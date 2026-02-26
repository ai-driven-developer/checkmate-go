package eval

import (
	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
)

// Pawn shield bonus by relative rank distance from the back rank.
// Index 0 = pawn on rank 2 (relative), 1 = rank 3, etc.
var shieldBonus = [8]int{0, 10, 5, 0, 0, 0, 0, 0}

// Penalty for a semi-open file near the king facing an enemy rook or queen.
const openFilePenalty = 15

// Attacker weights by piece type.
var attackerWeight = [7]int{
	0, // NoPiece
	0, // Pawn (handled via pawn shield)
	2, // Knight
	2, // Bishop
	3, // Rook
	5, // Queen
	0, // King
}

// safetyTable maps total attacker weight to centipawn penalty.
// Non-linear: multiple attackers are disproportionately dangerous.
var safetyTable [100]int

func init() {
	for i := range safetyTable {
		// Quadratic scaling: penalty = w^2 where w is total weight.
		// Capped at a reasonable maximum.
		penalty := i * i
		if penalty > 500 {
			penalty = 500
		}
		safetyTable[i] = penalty
	}
}

// kingSafetyScore evaluates king safety for both sides.
// Returns middlegame and endgame scores from White's perspective.
// The endgame component is always 0 (king PST handles endgame).
func kingSafetyScore(pos *board.Position) (mg, eg int) {
	mg += kingSafety(pos, board.White)
	mg -= kingSafety(pos, board.Black)
	return mg, 0
}

// kingSafety returns a positive score (bonus) for the given side's king safety.
// Higher = safer king.
func kingSafety(pos *board.Position, us board.Color) int {
	them := us.Other()
	kingSq := pos.KingSquare(us)
	if kingSq >= 64 {
		return 0
	}
	kingFile := kingSq.File()
	score := 0

	ownPawns := pos.ColorPieces(us, board.Pawn)
	enemyRooksQueens := pos.ColorPieces(them, board.Rook) | pos.ColorPieces(them, board.Queen)

	// --- Pawn shield ---
	for df := -1; df <= 1; df++ {
		f := kingFile + df
		if f < 0 || f > 7 {
			continue
		}
		pawnsOnFile := ownPawns & fileMask[f]
		if pawnsOnFile == 0 {
			// No pawn shield on this file.
			// Penalize if enemy has rook/queen on this file.
			if enemyRooksQueens&fileMask[f] != 0 {
				score -= openFilePenalty
			}
			continue
		}
		// Find the closest pawn to the back rank.
		var relRank int
		if us == board.White {
			// White: lowest rank pawn is closest to back rank.
			closest := pawnsOnFile.LSB()
			relRank = closest.Rank() // rank 1 = index 1
		} else {
			// Black: highest rank pawn is closest to back rank (rank 8).
			// We need the MSB. Iterate to find it.
			bb := pawnsOnFile
			var closest board.Square
			for bb != 0 {
				closest = bb.PopLSB()
			}
			relRank = 7 - closest.Rank() // mirror for relative rank
		}
		if relRank < len(shieldBonus) {
			score += shieldBonus[relRank]
		}
	}

	// --- King zone attackers ---
	kingZone := movegen.KingAttacks[kingSq] | board.SquareBB(kingSq)
	occ := pos.AllOccupied
	totalWeight := 0

	// Knights attacking king zone.
	bb := pos.ColorPieces(them, board.Knight)
	for bb != 0 {
		sq := bb.PopLSB()
		if movegen.KnightAttacks[sq]&kingZone != 0 {
			totalWeight += attackerWeight[board.Knight]
		}
	}

	// Bishops attacking king zone.
	bb = pos.ColorPieces(them, board.Bishop)
	for bb != 0 {
		sq := bb.PopLSB()
		if movegen.BishopAttacks(sq, occ)&kingZone != 0 {
			totalWeight += attackerWeight[board.Bishop]
		}
	}

	// Rooks attacking king zone.
	bb = pos.ColorPieces(them, board.Rook)
	for bb != 0 {
		sq := bb.PopLSB()
		if movegen.RookAttacks(sq, occ)&kingZone != 0 {
			totalWeight += attackerWeight[board.Rook]
		}
	}

	// Queens attacking king zone.
	bb = pos.ColorPieces(them, board.Queen)
	for bb != 0 {
		sq := bb.PopLSB()
		if movegen.QueenAttacks(sq, occ)&kingZone != 0 {
			totalWeight += attackerWeight[board.Queen]
		}
	}

	if totalWeight >= len(safetyTable) {
		totalWeight = len(safetyTable) - 1
	}
	score -= safetyTable[totalWeight]

	return score
}
