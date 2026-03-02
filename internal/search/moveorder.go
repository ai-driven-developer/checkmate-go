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

// ScoreMoves assigns ordering scores to each move in the move list without sorting.
// Used with PickBest for lazy move ordering: only the next-best move is selected
// on each iteration, avoiding a full O(n²) sort when beta cutoff happens early.
func ScoreMoves(ml *board.MoveList, scores *[256]int32, hashMove board.Move, killers [2]board.Move, countermove board.Move, history *[2][64][64]int32, side board.Color, pos *board.Position) {
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		if m == hashMove {
			scores[i] = 2_000_000
			continue
		}
		if m.IsCapture() {
			mvvScore := int32(mvvLva[m.CapturedPiece()][m.Piece()])
			if pos != nil && SEE(pos, m) < 0 {
				// Bad capture: demote below killers and quiet moves.
				scores[i] = mvvScore - 1_000_000
			} else {
				scores[i] = mvvScore + 1_000_000
			}
		} else if m == killers[0] || m == killers[1] {
			scores[i] = 500_000
		} else if m == countermove {
			scores[i] = 400_000
		} else if history != nil {
			scores[i] = history[side][m.From()][m.To()]
		}
		if m.IsPromotion() {
			scores[i] += 900_000
		}
	}
}

// PickBest finds the highest-scored move in [from, count) and swaps it into
// position 'from'. This implements lazy selection sort: O(n) per call, but
// only invoked for the moves actually searched before a cutoff.
func PickBest(ml *board.MoveList, scores *[256]int32, from int) {
	best := from
	for j := from + 1; j < ml.Count; j++ {
		if scores[j] > scores[best] {
			best = j
		}
	}
	if best != from {
		ml.Moves[from], ml.Moves[best] = ml.Moves[best], ml.Moves[from]
		scores[from], scores[best] = scores[best], scores[from]
	}
}

// OrderMoves scores and fully sorts the move list. Used in contexts where
// all moves will be examined (tests, simple callers). The search loop uses
// ScoreMoves + PickBest instead for lazy evaluation.
func OrderMoves(ml *board.MoveList, hashMove board.Move, killers [2]board.Move, countermove board.Move, history *[2][64][64]int32, side board.Color, pos *board.Position) {
	var scores [256]int32
	ScoreMoves(ml, &scores, hashMove, killers, countermove, history, side, pos)
	for i := 0; i < ml.Count; i++ {
		PickBest(ml, &scores, i)
	}
}
