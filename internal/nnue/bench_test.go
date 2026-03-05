package nnue

import (
	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
	"testing"
)

func BenchmarkEvaluate(b *testing.B) {
	net, err := LoadEmbeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		net.Evaluate(as.Current(), board.White)
	}
}

func BenchmarkAccumulatorRefresh(b *testing.B) {
	net, err := LoadEmbeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		as.Refresh(pos)
	}
}

func BenchmarkAccumulatorMakeUnmake(b *testing.B) {
	net, err := LoadEmbeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e2e4")
	if m == board.NullMove {
		b.Fatal("e2e4 not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		as.MakeMove(pos, m)
		as.UnmakeMove()
	}
}

func BenchmarkSearchLikeWorkload(b *testing.B) {
	net, err := LoadEmbeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	// Get all legal moves from starting position.
	var ml board.MoveList
	movegen.GenerateLegalMoves(pos, &ml)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate: for each legal move, make, evaluate, unmake.
		for j := 0; j < ml.Count; j++ {
			m := ml.Moves[j]
			as.MakeMove(pos, m)
			pos.MakeMove(m)
			net.Evaluate(as.Current(), pos.SideToMove)
			pos.UnmakeMove(m)
			as.UnmakeMove()
		}
	}
}
