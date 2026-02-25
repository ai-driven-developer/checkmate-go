package search

import "checkmatego/internal/board"

// MVV-LVA scores: [victim][attacker]. Higher = try first.
var mvvLva = [7][7]int{
	{0, 0, 0, 0, 0, 0, 0},       // NoPiece victim
	{0, 15, 14, 13, 12, 11, 10},  // Pawn victim
	{0, 25, 24, 23, 22, 21, 20},  // Knight victim
	{0, 35, 34, 33, 32, 31, 30},  // Bishop victim
	{0, 45, 44, 43, 42, 41, 40},  // Rook victim
	{0, 55, 54, 53, 52, 51, 50},  // Queen victim
	{0, 0, 0, 0, 0, 0, 0},       // King victim
}

// OrderMoves sorts the move list. If hashMove is not NullMove, it gets
// highest priority. Captures are ordered by MVV-LVA. Killer moves are
// ordered between captures and plain quiet moves. Remaining quiet moves
// are ordered by history heuristic scores.
// Uses insertion sort (optimal for ~30-50 moves).
func OrderMoves(ml *board.MoveList, hashMove board.Move, killers [2]board.Move, history *[2][64][64]int32, side board.Color) {
	var scores [256]int32
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		if m == hashMove {
			scores[i] = 2_000_000
			continue
		}
		if m.IsCapture() {
			scores[i] = int32(mvvLva[m.CapturedPiece()][m.Piece()]) + 1_000_000
		} else if m == killers[0] || m == killers[1] {
			scores[i] = 500_000
		} else if history != nil {
			scores[i] = history[side][m.From()][m.To()]
		}
		if m.IsPromotion() {
			scores[i] += 900_000
		}
	}
	// Insertion sort descending.
	for i := 1; i < ml.Count; i++ {
		for j := i; j > 0 && scores[j] > scores[j-1]; j-- {
			ml.Moves[j], ml.Moves[j-1] = ml.Moves[j-1], ml.Moves[j]
			scores[j], scores[j-1] = scores[j-1], scores[j]
		}
	}
}
