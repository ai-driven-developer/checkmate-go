package movegen

import (
	"checkmatego/internal/board"
	"fmt"
)

// Perft counts leaf nodes at the given depth.
func Perft(pos *board.Position, depth int) uint64 {
	if depth == 0 {
		return 1
	}
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)
	if depth == 1 {
		return uint64(ml.Count)
	}
	var nodes uint64
	for i := 0; i < ml.Count; i++ {
		pos.MakeMove(ml.Moves[i])
		nodes += Perft(pos, depth-1)
		pos.UnmakeMove(ml.Moves[i])
	}
	return nodes
}

// Divide runs perft and prints per-move node counts.
func Divide(pos *board.Position, depth int) uint64 {
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)
	var total uint64
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		pos.MakeMove(m)
		nodes := Perft(pos, depth-1)
		pos.UnmakeMove(m)
		fmt.Printf("%s: %d\n", m, nodes)
		total += nodes
	}
	fmt.Printf("\nTotal: %d\n", total)
	return total
}
