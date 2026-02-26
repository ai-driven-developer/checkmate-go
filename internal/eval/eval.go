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

// gamePhase returns a value from 0 (endgame) to totalPhase (opening).
func gamePhase(pos *board.Position) int {
	phase := 0
	phase += (pos.ColorPieces(board.White, board.Knight).Count() +
		pos.ColorPieces(board.Black, board.Knight).Count()) * knightPhase
	phase += (pos.ColorPieces(board.White, board.Bishop).Count() +
		pos.ColorPieces(board.Black, board.Bishop).Count()) * bishopPhase
	phase += (pos.ColorPieces(board.White, board.Rook).Count() +
		pos.ColorPieces(board.Black, board.Rook).Count()) * rookPhase
	phase += (pos.ColorPieces(board.White, board.Queen).Count() +
		pos.ColorPieces(board.Black, board.Queen).Count()) * queenPhase
	return phase
}

// Evaluate returns a score in centipawns from the perspective of the side to move.
// Positive = good for side to move. Uses tapered evaluation to interpolate
// between middlegame and endgame scores based on remaining material.
func Evaluate(pos *board.Position) int {
	mat := materialBalance(pos)
	mob := mobilityScore(pos)
	mgPST, egPST := pstBalanceTapered(pos)
	mgPP, egPP := passedPawnScore(pos)
	mgPS, egPS := pawnStructureScore(pos)
	mgKS, egKS := kingSafetyScore(pos)

	phase := gamePhase(pos)
	// Tapered score: interpolate between MG and EG.
	mg := mat + mgPST + mob + mgPP + mgPS + mgKS
	eg := mat + egPST + mob + egPP + egPS + egKS
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
