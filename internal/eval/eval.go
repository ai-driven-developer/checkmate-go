package eval

import (
	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
)

// Phase weights for non-pawn pieces.
const (
	knightPhase = 1
	bishopPhase = 1
	rookPhase   = 2
	queenPhase  = 4
	totalPhase  = 4*knightPhase + 4*bishopPhase + 4*rookPhase + 2*queenPhase // 24
)

// Evaluate returns a score in centipawns from the perspective of the side to move.
// Positive = good for side to move. Uses tapered evaluation to interpolate
// between middlegame and endgame scores based on remaining material.
func Evaluate(pos *board.Position) int {
	return EvaluateWithCache(pos, nil)
}

// EvaluateWithCache is like Evaluate but uses a pawn cache to avoid
// recomputing pawn structure and passed pawn scores when the pawn
// configuration hasn't changed.
func EvaluateWithCache(pos *board.Position, pc *PawnCache) int {
	mat := materialBalance(pos)
	mob := mobilityScore(pos)
	mgPST, egPST := pos.PSTMG, pos.PSTEG
	mgKS, egKS := kingSafetyScore(pos)
	mgOp, egOp := outpostScore(pos)
	mgRk, egRk := rookScore(pos)
	mgKP, egKP := kingPasserDistanceScore(pos)

	var mgPP, egPP, mgPS, egPS int
	if pc != nil {
		if hit, cmg, ceg := pc.Probe(pos.PawnHash); hit {
			// Cache hit: combined pawn structure + passed pawn scores.
			mgPP = int(cmg) // packed: pawnStruct.mg + passedPawn.mg
			egPP = int(ceg)
			mgPS, egPS = 0, 0
		} else {
			mgPP, egPP = passedPawnScore(pos)
			mgPS, egPS = pawnStructureScore(pos)
			pc.Store(pos.PawnHash, int16(mgPP+mgPS), int16(egPP+egPS))
			mgPP, egPP = mgPP+mgPS, egPP+egPS
			mgPS, egPS = 0, 0
		}
	} else {
		mgPP, egPP = passedPawnScore(pos)
		mgPS, egPS = pawnStructureScore(pos)
	}

	phase := pos.Phase
	// Tapered score: interpolate between MG and EG.
	mg := mat + mgPST + mob + mgPP + mgPS + mgKS + mgOp + mgRk + mgKP
	eg := mat + egPST + mob + egPP + egPS + egKS + egOp + egRk + egKP
	score := (mg*phase + eg*(totalPhase-phase)) / totalPhase

	if pos.SideToMove == board.Black {
		score = -score
	}
	return score
}

// mobilityScore computes a mobility bonus for minor and major pieces.
func mobilityScore(pos *board.Position) int {
	score := 0
	occ := pos.AllOccupied

	// Knight mobility.
	bb := pos.ColorPieces(board.White, board.Knight)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.KnightAttacks[sq] &^ pos.Occupied[board.White]
		score += attacks.Count() * 4
	}
	bb = pos.ColorPieces(board.Black, board.Knight)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.KnightAttacks[sq] &^ pos.Occupied[board.Black]
		score -= attacks.Count() * 4
	}

	// Bishop mobility.
	bb = pos.ColorPieces(board.White, board.Bishop)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.BishopAttacks(sq, occ) &^ pos.Occupied[board.White]
		score += attacks.Count() * 5
	}
	bb = pos.ColorPieces(board.Black, board.Bishop)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.BishopAttacks(sq, occ) &^ pos.Occupied[board.Black]
		score -= attacks.Count() * 5
	}

	// Rook mobility.
	bb = pos.ColorPieces(board.White, board.Rook)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.RookAttacks(sq, occ) &^ pos.Occupied[board.White]
		score += attacks.Count() * 3
	}
	bb = pos.ColorPieces(board.Black, board.Rook)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.RookAttacks(sq, occ) &^ pos.Occupied[board.Black]
		score -= attacks.Count() * 3
	}

	// Queen mobility.
	bb = pos.ColorPieces(board.White, board.Queen)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.QueenAttacks(sq, occ) &^ pos.Occupied[board.White]
		score += attacks.Count() * 2
	}
	bb = pos.ColorPieces(board.Black, board.Queen)
	for bb != 0 {
		sq := bb.PopLSB()
		attacks := movegen.QueenAttacks(sq, occ) &^ pos.Occupied[board.Black]
		score -= attacks.Count() * 2
	}

	return score
}
