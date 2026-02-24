package eval

import (
	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
)

// Evaluate returns a score in centipawns from the perspective of the side to move.
// Positive = good for side to move.
func Evaluate(pos *board.Position) int {
	score := materialBalance(pos) + pstBalance(pos) + mobilityScore(pos)
	if pos.SideToMove == board.Black {
		score = -score
	}
	return score
}

// mobilityScore computes a lightweight mobility bonus for knights and bishops.
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

	return score
}
