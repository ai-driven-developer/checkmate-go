package search

import (
	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
	"testing"
)

func TestMateInOne(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		mate string
	}{
		{
			name: "back rank mate",
			fen:  "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1",
			mate: "a1a8",
		},
		{
			name: "queen delivers mate",
			fen:  "2k5/8/2K5/8/8/8/8/Q7 w - - 0 1",
			mate: "a1a8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tt.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 4})
			if bestMove.String() != tt.mate {
				t.Errorf("expected %s, got %s", tt.mate, bestMove)
			}
		})
	}
}

func TestMateInTwo(t *testing.T) {
	// Scholar's mate: Qxf7#
	pos := &board.Position{}
	_ = pos.SetFromFEN("r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 5})
	if bestMove.String() != "h5f7" {
		t.Errorf("expected h5f7 (mate), got %s", bestMove)
	}
}

func TestAvoidStalemate(t *testing.T) {
	// White has king on b6, queen on c6, black king on a8.
	// Qc8 would be stalemate (or Qa6 stalemate). Engine should avoid it.
	pos := &board.Position{}
	_ = pos.SetFromFEN("k7/8/1KQ5/8/8/8/8/8 w - - 0 1")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 4})

	// Apply the move and verify it's not stalemate.
	pos.MakeMove(bestMove)
	var ml board.MoveList
	movegen.GenerateLegalMoves(pos, &ml)
	inCheck := movegen.IsSquareAttacked(pos, pos.KingSquare(pos.SideToMove), pos.SideToMove.Other())
	if ml.Count == 0 && !inCheck {
		t.Errorf("engine chose stalemate move: %s", bestMove)
	}
}

func TestSearchDepthLimit(t *testing.T) {
	pos := board.NewPosition()
	engine := NewEngine()

	var lastDepth int
	engine.SetInfoCallback(func(info SearchInfo) {
		lastDepth = info.Depth
	})

	engine.Search(pos, SearchLimits{Depth: 3})
	if lastDepth != 3 {
		t.Errorf("expected search to reach depth 3, got %d", lastDepth)
	}
}

func TestSearchReturnsValidMove(t *testing.T) {
	pos := board.NewPosition()
	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 4})

	if bestMove == board.NullMove {
		t.Error("engine returned null move for starting position")
	}
	s := bestMove.String()
	if len(s) < 4 || len(s) > 5 {
		t.Errorf("invalid move string: %s", s)
	}
}

func TestSearchFindsCapture(t *testing.T) {
	// White has a bishop that can capture a free queen.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/3q4/8/5B2/4K3 w - - 0 1")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 3})
	// Bf2 can capture d4 via the diagonal? Bf2 attacks e3,d4,e1,g3,h4,g1.
	// Yes, Bxd4 captures the queen.
	if bestMove.String() != "f2d4" {
		t.Errorf("expected f2d4 (capture queen), got %s", bestMove)
	}
}

func TestSearchMateScore(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1")

	engine := NewEngine()
	var mateScore int
	engine.SetInfoCallback(func(info SearchInfo) {
		mateScore = info.Score
	})
	engine.Search(pos, SearchLimits{Depth: 3})

	if mateScore < MateScore-10 {
		t.Errorf("expected mate score, got %d", mateScore)
	}
}
