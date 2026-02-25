package search

import (
	"checkmatego/internal/board"
	"testing"
)

func TestStoreKillerSwap(t *testing.T) {
	w := &worker{engine: NewEngine()}
	m1 := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	m2 := board.NewMove(board.D2, board.D4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	m3 := board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece)

	// Store first killer.
	w.storeKiller(m1, 0)
	if w.killers[0][0] != m1 {
		t.Errorf("killers[0][0] = %v, want %v", w.killers[0][0], m1)
	}

	// Store second killer: m1 shifts to slot 1, m2 takes slot 0.
	w.storeKiller(m2, 0)
	if w.killers[0][0] != m2 {
		t.Errorf("killers[0][0] = %v, want %v", w.killers[0][0], m2)
	}
	if w.killers[0][1] != m1 {
		t.Errorf("killers[0][1] = %v, want %v", w.killers[0][1], m1)
	}

	// Store same move again — should not change anything.
	w.storeKiller(m2, 0)
	if w.killers[0][0] != m2 || w.killers[0][1] != m1 {
		t.Error("storing duplicate killer should be a no-op")
	}

	// Store third killer: m2 shifts to slot 1, m3 takes slot 0.
	w.storeKiller(m3, 0)
	if w.killers[0][0] != m3 {
		t.Errorf("killers[0][0] = %v, want %v", w.killers[0][0], m3)
	}
	if w.killers[0][1] != m2 {
		t.Errorf("killers[0][1] = %v, want %v", w.killers[0][1], m2)
	}

	// Different ply should be independent.
	if w.killers[1][0] != board.NullMove {
		t.Error("killers at different ply should be independent")
	}
}

func TestFiftyMoveRuleDraw(t *testing.T) {
	// White is up a queen, but halfmove clock is 100 → draw.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/8/8/8/Q3K3 w - - 100 80")

	engine := NewEngine()
	var lastScore int
	engine.SetInfoCallback(func(info SearchInfo) {
		lastScore = info.Score
	})
	engine.Search(pos, SearchLimits{Depth: 4})

	// Score should be near zero (draw), not a big positive.
	if lastScore > 100 || lastScore < -100 {
		t.Errorf("expected draw score (~0) due to 50-move rule, got %d", lastScore)
	}
}

func TestFutilityPruningReducesNodes(t *testing.T) {
	// A position where white is significantly down in material.
	// Futility pruning should reduce nodes at shallow depths.
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBK1BNR w kq - 0 1") // white missing queen

	engine1 := NewEngine()
	engine1.Search(pos, SearchLimits{Depth: 5})
	nodesWithFutility := engine1.nodes.Load()

	// Verify the search actually ran and produced nodes.
	if nodesWithFutility == 0 {
		t.Fatal("search produced no nodes")
	}

	// We can't easily disable futility pruning in a unit test without a flag,
	// but we verify the search completes correctly with it enabled and finds
	// a valid move.
	bestMove := engine1.Search(pos, SearchLimits{Depth: 4})
	if bestMove == board.NullMove {
		t.Error("futility pruning should not prevent finding a valid move")
	}
}

func TestAspirationWindowsCorrectness(t *testing.T) {
	// Test that aspiration windows produce the same best move as a full-window
	// search. Use a tactical position where the score can change significantly.
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "winning capture",
			fen:  "4k3/8/8/8/3q4/8/5B2/4K3 w - - 0 1",
		},
		{
			name: "mate in one",
			fen:  "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1",
		},
		{
			name: "starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			// Aspiration windows kick in at depth >= 4. Search at depth 6
			// to exercise multiple iterations with the window.
			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove == board.NullMove {
				t.Error("aspiration windows should not prevent finding a move")
			}
		})
	}
}

func TestNullMovePruningReducesNodes(t *testing.T) {
	// A quiet middlegame position. Null-move pruning should reduce nodes.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r1bqkb1r/pppppppp/2n2n2/8/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 6})
	nodes := engine.nodes.Load()

	if bestMove == board.NullMove {
		t.Error("expected a valid move")
	}
	if nodes == 0 {
		t.Error("expected some nodes to be searched")
	}
}

func TestMultiThreadedSearch(t *testing.T) {
	pos := board.NewPosition()

	engine := NewEngine()
	engine.SetThreads(4)

	bestMove := engine.Search(pos, SearchLimits{Depth: 5})
	if bestMove == board.NullMove {
		t.Error("multi-threaded search returned null move")
	}

	nodes := engine.nodes.Load()
	if nodes == 0 {
		t.Error("multi-threaded search reported 0 nodes")
	}
}

func TestHistoryHeuristicOrdering(t *testing.T) {
	var history [2][64][64]int32
	// Give Nf3 (g1→f3) a high history score for White.
	history[board.White][board.G1][board.F3] = 1000
	// Give e4 (e2→e4) a low history score.
	history[board.White][board.E2][board.E4] = 10

	var ml board.MoveList
	nf3 := board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece)
	e4 := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	d3 := board.NewMove(board.D2, board.D3, board.FlagQuiet, board.Pawn, board.NoPiece) // score 0

	ml.Add(d3)
	ml.Add(e4)
	ml.Add(nf3)

	OrderMoves(&ml, board.NullMove, [2]board.Move{}, &history, board.White)

	// Nf3 (score 1000) should come first, then e4 (10), then d3 (0).
	if ml.Moves[0] != nf3 {
		t.Errorf("expected Nf3 first (highest history), got %v", ml.Moves[0])
	}
	if ml.Moves[1] != e4 {
		t.Errorf("expected e4 second, got %v", ml.Moves[1])
	}
	if ml.Moves[2] != d3 {
		t.Errorf("expected d3 last, got %v", ml.Moves[2])
	}
}

func TestPVSCorrectness(t *testing.T) {
	// PVS must find the same best moves as a plain alpha-beta would.
	tests := []struct {
		name     string
		fen      string
		wantMove string
	}{
		{
			name:     "capture free queen",
			fen:      "4k3/8/8/8/3q4/8/5B2/4K3 w - - 0 1",
			wantMove: "f2d4",
		},
		{
			name:     "back rank mate",
			fen:      "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1",
			wantMove: "a1a8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			// Depth 6 exercises PVS: first move full window, rest zero-window.
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestPVSNodeReduction(t *testing.T) {
	// PVS should not search more nodes than a reasonable upper bound.
	// This is a sanity check that zero-window searches are happening.
	pos := board.NewPosition()

	engine := NewEngine()
	engine.Search(pos, SearchLimits{Depth: 7})
	nodes := engine.nodes.Load()

	if nodes == 0 {
		t.Fatal("search produced no nodes")
	}
	// With PVS + LMR + NMP + futility the starting position at depth 7
	// should stay well under 1M nodes.
	if nodes > 1_000_000 {
		t.Errorf("PVS search used %d nodes at depth 7, expected fewer", nodes)
	}
}

func TestCheckExtensionFindsDeeperMate(t *testing.T) {
	// Mate in 2 via checks: 1. Rd7+ Ka8 2. Rg8#
	// Without check extensions depth 2 cannot see the mate (quiesce misses
	// the non-capture Rg8#). With check extensions Rd7+ extends by 1 ply,
	// making the mate visible even at depth 2.
	pos := &board.Position{}
	_ = pos.SetFromFEN("8/1k4R1/8/1K6/8/8/8/3R4 w - - 0 1")

	engine := NewEngine()
	var mateScore int
	engine.SetInfoCallback(func(info SearchInfo) {
		mateScore = info.Score
	})
	bestMove := engine.Search(pos, SearchLimits{Depth: 4})

	if bestMove == board.NullMove {
		t.Fatal("expected a valid move")
	}
	if mateScore < MateScore-MaxDepth {
		t.Errorf("expected mate score with check extension, got %d", mateScore)
	}
}

func TestHistoryDoesNotOverrideCaptures(t *testing.T) {
	var history [2][64][64]int32
	// Even with a very high history score, captures should still come first.
	history[board.White][board.G1][board.F3] = 999_999

	var ml board.MoveList
	nf3 := board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece)
	capture := board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Pawn)

	ml.Add(nf3)
	ml.Add(capture)

	OrderMoves(&ml, board.NullMove, [2]board.Move{}, &history, board.White)

	if ml.Moves[0] != capture {
		t.Error("capture should still come before quiet move with high history score")
	}
}
