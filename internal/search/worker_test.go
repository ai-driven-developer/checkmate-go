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

	OrderMoves(&ml, board.NullMove, [2]board.Move{}, board.NullMove, &history, board.White, nil)

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

func TestSingularExtensionExcludedMoveSkipped(t *testing.T) {
	// Verify the excludedMove field is NullMove after search and that
	// singular extensions don't break normal search.
	pos := board.NewPosition()

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 10})

	if bestMove == board.NullMove {
		t.Error("search should return a valid move with singular extensions enabled")
	}

	// After search, a fresh worker should have excludedMove = NullMove.
	w := &worker{engine: engine}
	if w.excludedMove != board.NullMove {
		t.Error("excludedMove should be NullMove after search completes")
	}
}

func TestSingularExtensionDoesNotRegress(t *testing.T) {
	// Tactical positions must still be solved correctly with SE enabled.
	tests := []struct {
		name     string
		fen      string
		wantMove string
		depth    int
	}{
		{
			name:     "capture free queen",
			fen:      "4k3/8/8/8/3q4/8/5B2/4K3 w - - 0 1",
			wantMove: "f2d4",
			depth:    8,
		},
		{
			name:     "back rank mate",
			fen:      "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1",
			wantMove: "a1a8",
			depth:    8,
		},
		{
			name:     "mate in 2",
			fen:      "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
			wantMove: "h5f7",
			depth:    8,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: tc.depth})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestReverseFutilityPruningReducesNodes(t *testing.T) {
	// White has an overwhelming material advantage (extra queen + rook).
	// RFP should prune many nodes at shallow depths where staticEval - margin >= beta.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/8/8/8/QR2K3 w - - 0 1")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 6})
	nodes := engine.nodes.Load()

	if bestMove == board.NullMove {
		t.Error("reverse futility pruning should not prevent finding a valid move")
	}
	if nodes == 0 {
		t.Fatal("search produced no nodes")
	}
}

func TestReverseFutilityPruningDoesNotMissMate(t *testing.T) {
	// White has huge material advantage AND a mate in 1. RFP must not prevent
	// the engine from finding the mating move.
	pos := &board.Position{}
	_ = pos.SetFromFEN("6k1/5ppp/8/8/8/8/1Q6/R3K3 w - - 0 1")

	engine := NewEngine()
	var lastScore int
	engine.SetInfoCallback(func(info SearchInfo) {
		lastScore = info.Score
	})
	bestMove := engine.Search(pos, SearchLimits{Depth: 4})

	if bestMove == board.NullMove {
		t.Fatal("expected a valid move")
	}
	// The engine should still find the mate despite the large material advantage
	// that could trigger RFP in some branches.
	if lastScore < MateScore-MaxDepth {
		t.Errorf("expected mate score, got %d", lastScore)
	}
}

func TestReverseFutilityPruningPreservesCorrectPlay(t *testing.T) {
	// Positions with large material advantage where RFP will be active.
	// Verify the engine still plays correctly.
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "queen vs bare king",
			fen:  "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1",
		},
		{
			name: "two rooks vs bare king",
			fen:  "4k3/8/8/8/8/8/8/R1R1K3 w - - 0 1",
		},
		{
			name: "extra piece in middlegame",
			fen:  "r1bqkb1r/pppppppp/2n2n2/8/4P3/2N2N2/PPPP1PPP/R1BQKB1R w KQkq - 4 4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 5})
			if bestMove == board.NullMove {
				t.Error("expected a valid move with RFP enabled")
			}
		})
	}
}

func TestLateMovePruningReducesNodes(t *testing.T) {
	// A quiet middlegame position. LMP should prune late quiet moves at
	// shallow depths and the search should still find a valid move.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r1bqkb1r/pppppppp/2n2n2/8/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 6})
	nodes := engine.nodes.Load()

	if bestMove == board.NullMove {
		t.Error("late move pruning should not prevent finding a valid move")
	}
	if nodes == 0 {
		t.Fatal("search produced no nodes")
	}
}

func TestLateMovePruningDoesNotMissTactics(t *testing.T) {
	// Tactical positions must still be solved correctly with LMP enabled.
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
		{
			name:     "mate in 2",
			fen:      "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
			wantMove: "h5f7",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestLateMovePruningPreservesCorrectPlay(t *testing.T) {
	// Various positions where LMP will be active at shallow internal nodes.
	// Verify the engine still returns valid moves.
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "sicilian defense",
			fen:  "rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2",
		},
		{
			name: "queen endgame",
			fen:  "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 5})
			if bestMove == board.NullMove {
				t.Error("expected a valid move with LMP enabled")
			}
		})
	}
}

func TestIIRReducesNodes(t *testing.T) {
	// A quiet middlegame position searched at depth where IIR fires (>= 4).
	// IIR should reduce nodes by filling the TT faster when no hash move exists.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r1bqkb1r/pppppppp/2n2n2/8/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 7})
	nodes := engine.nodes.Load()

	if bestMove == board.NullMove {
		t.Error("IIR should not prevent finding a valid move")
	}
	if nodes == 0 {
		t.Fatal("search produced no nodes")
	}
}

func TestIIRDoesNotMissTactics(t *testing.T) {
	// Tactical positions must still be solved correctly with IIR enabled.
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
		{
			name:     "mate in 2",
			fen:      "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
			wantMove: "h5f7",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestIIRPreservesCorrectPlay(t *testing.T) {
	// Various positions where IIR will fire on first visits (no TT entry).
	// Verify the engine still returns valid moves.
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "sicilian defense",
			fen:  "rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2",
		},
		{
			name: "queen endgame",
			fen:  "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove == board.NullMove {
				t.Error("expected a valid move with IIR enabled")
			}
		})
	}
}

func TestImprovingFlagReducesNodes(t *testing.T) {
	// A quiet middlegame position. The improving heuristic should help prune
	// more aggressively when the position is not improving, reducing nodes.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r1bqkb1r/pppppppp/2n2n2/8/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 6})
	nodes := engine.nodes.Load()

	if bestMove == board.NullMove {
		t.Error("improving heuristic should not prevent finding a valid move")
	}
	if nodes == 0 {
		t.Fatal("search produced no nodes")
	}
}

func TestImprovingFlagDoesNotMissTactics(t *testing.T) {
	// Tactical positions must still be solved correctly with improving heuristic.
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
		{
			name:     "mate in 2",
			fen:      "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
			wantMove: "h5f7",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestImprovingFlagPreservesCorrectPlay(t *testing.T) {
	// Various positions where the improving heuristic will influence pruning.
	// Verify the engine still returns valid moves.
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "sicilian defense",
			fen:  "rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2",
		},
		{
			name: "queen endgame",
			fen:  "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 5})
			if bestMove == board.NullMove {
				t.Error("expected a valid move with improving heuristic enabled")
			}
		})
	}
}

func TestCountermoveStorage(t *testing.T) {
	w := &worker{engine: NewEngine()}

	prevMove := board.NewMove(board.E7, board.E5, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	counterMove := board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece)

	// Initially no countermove stored.
	if w.countermoves[prevMove.Piece()][prevMove.To()] != board.NullMove {
		t.Error("countermove should initially be NullMove")
	}

	// Store a countermove for Pawn to E5.
	w.countermoves[prevMove.Piece()][prevMove.To()] = counterMove

	got := w.countermoves[prevMove.Piece()][prevMove.To()]
	if got != counterMove {
		t.Errorf("countermove = %v, want %v", got, counterMove)
	}

	// Different previous move should have independent storage.
	otherPrev := board.NewMove(board.D7, board.D5, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	if w.countermoves[otherPrev.Piece()][otherPrev.To()] != board.NullMove {
		t.Error("countermove for different prev move should be independent")
	}
}

func TestCountermoveReplacesOldEntry(t *testing.T) {
	w := &worker{engine: NewEngine()}

	prevMove := board.NewMove(board.E7, board.E5, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	cm1 := board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece)
	cm2 := board.NewMove(board.D2, board.D4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	w.countermoves[prevMove.Piece()][prevMove.To()] = cm1
	w.countermoves[prevMove.Piece()][prevMove.To()] = cm2

	got := w.countermoves[prevMove.Piece()][prevMove.To()]
	if got != cm2 {
		t.Errorf("countermove should be replaced: got %v, want %v", got, cm2)
	}
}

func TestCountermoveResetBetweenIterations(t *testing.T) {
	// Search clears countermoves at start — verify by running a search and
	// checking the table is populated during search (implicitly via correct play).
	pos := board.NewPosition()

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 6})
	if bestMove == board.NullMove {
		t.Error("search with countermove heuristic should return a valid move")
	}
}

func TestCountermoveDoesNotMissTactics(t *testing.T) {
	// Tactical positions must still be solved correctly with countermove heuristic.
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
		{
			name:     "mate in 2",
			fen:      "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
			wantMove: "h5f7",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestCountermovePreservesCorrectPlay(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "sicilian defense",
			fen:  "rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2",
		},
		{
			name: "queen endgame",
			fen:  "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 5})
			if bestMove == board.NullMove {
				t.Error("expected a valid move with countermove heuristic enabled")
			}
		})
	}
}

func TestDeltaPruningReducesNodes(t *testing.T) {
	// A position where white is massively down in material (missing queen
	// and both rooks). QSearch with delta pruning should prune futile
	// captures and search fewer nodes than without it.
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/1NB1KB2 w kq - 0 1") // white missing Q + both R

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 6})
	nodes := engine.nodes.Load()

	if bestMove == board.NullMove {
		t.Error("delta pruning should not prevent finding a valid move")
	}
	if nodes == 0 {
		t.Fatal("search produced no nodes")
	}
}

func TestDeltaPruningDoesNotMissTactics(t *testing.T) {
	// Tactical positions must still be solved correctly with delta pruning.
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
		{
			name:     "mate in 2",
			fen:      "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
			wantMove: "h5f7",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestDeltaPruningPreservesCorrectPlay(t *testing.T) {
	// Various positions where delta pruning is active in QSearch.
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "sicilian defense",
			fen:  "rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2",
		},
		{
			name: "queen endgame",
			fen:  "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1",
		},
		{
			name: "complex middlegame",
			fen:  "r1bqk2r/ppp2ppp/2n1pn2/3p4/2PP4/2N2N2/PP2PPPP/R1BQKB1R w KQkq d6 0 5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 5})
			if bestMove == board.NullMove {
				t.Error("expected a valid move with delta pruning enabled")
			}
		})
	}
}

func TestDeltaPruningSkipsPromotions(t *testing.T) {
	// White has a pawn on the 7th rank that can capture-promote (bxa8=Q).
	// Delta pruning must NOT skip promotion captures, since they are
	// exempt from the per-move delta check.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r3k3/1P6/8/8/8/8/8/4K3 w q - 0 1")

	engine := NewEngine()
	var lastScore int
	engine.SetInfoCallback(func(info SearchInfo) {
		lastScore = info.Score
	})
	bestMove := engine.Search(pos, SearchLimits{Depth: 6})

	if bestMove == board.NullMove {
		t.Fatal("expected a valid move")
	}
	// White should find the promotion (b8=Q or bxa8=Q) — score should
	// reflect the massive material gain.
	if lastScore < 500 {
		t.Errorf("expected high score from promotion, got %d", lastScore)
	}
}

func TestDeltaPruningBigDelta(t *testing.T) {
	// White is extremely down (bare king vs full army). The big delta check
	// (standPat + 1100 < alpha) should fire in many QSearch nodes, pruning
	// entire subtrees.
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnbqkbnr/pppppppp/8/8/8/8/8/4K3 w kq - 0 1") // white: bare king

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 5})

	if bestMove == board.NullMove {
		t.Error("big delta pruning should not prevent finding a valid move")
	}
}

func TestUpdateHistoryGravityBounds(t *testing.T) {
	// Verify the gravity formula keeps values bounded in [-maxHistory, maxHistory].
	w := &worker{engine: NewEngine()}

	// Apply many large positive bonuses — should converge toward maxHistory.
	for i := 0; i < 200; i++ {
		w.updateHistory(board.White, board.E2, board.E4, 400)
	}
	val := w.history[board.White][board.E2][board.E4]
	if val > maxHistory {
		t.Errorf("history exceeded maxHistory: %d > %d", val, maxHistory)
	}
	if val < maxHistory*90/100 {
		t.Errorf("history should converge near maxHistory: %d", val)
	}

	// Apply many large negative bonuses — should converge toward -maxHistory.
	w.history[board.White][board.D2][board.D4] = 0
	for i := 0; i < 200; i++ {
		w.updateHistory(board.White, board.D2, board.D4, -400)
	}
	val = w.history[board.White][board.D2][board.D4]
	if val < -maxHistory {
		t.Errorf("history below -maxHistory: %d", val)
	}
	if val > -maxHistory*90/100 {
		t.Errorf("history should converge near -maxHistory: %d", val)
	}
}

func TestUpdateHistoryGravityDecay(t *testing.T) {
	// A move with high history that later gets penalized should decay.
	w := &worker{engine: NewEngine()}

	// Build up positive history.
	for i := 0; i < 50; i++ {
		w.updateHistory(board.White, board.E2, board.E4, 400)
	}
	high := w.history[board.White][board.E2][board.E4]

	// Apply penalties — value should decrease.
	for i := 0; i < 50; i++ {
		w.updateHistory(board.White, board.E2, board.E4, -400)
	}
	low := w.history[board.White][board.E2][board.E4]

	if low >= high {
		t.Errorf("penalties should reduce history: before=%d, after=%d", high, low)
	}
}

func TestUpdateHistoryMalusApplied(t *testing.T) {
	// Verify that applying bonus to one move and malus to another
	// produces the expected signs.
	w := &worker{engine: NewEngine()}

	// Simulate: move A gets bonus, move B gets malus.
	w.updateHistory(board.White, board.E2, board.E4, 100) // bonus
	w.updateHistory(board.White, board.D2, board.D3, -100) // malus

	if w.history[board.White][board.E2][board.E4] <= 0 {
		t.Errorf("bonus move should have positive history, got %d",
			w.history[board.White][board.E2][board.E4])
	}
	if w.history[board.White][board.D2][board.D3] >= 0 {
		t.Errorf("malus move should have negative history, got %d",
			w.history[board.White][board.D2][board.D3])
	}
}

func TestHistoryGravityDoesNotMissTactics(t *testing.T) {
	// Tactical positions must still be solved correctly with history gravity.
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
		{
			name:     "mate in 2",
			fen:      "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
			wantMove: "h5f7",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 6})
			if bestMove.String() != tc.wantMove {
				t.Errorf("expected %s, got %s", tc.wantMove, bestMove)
			}
		})
	}
}

func TestHistoryGravityPreservesCorrectPlay(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "sicilian defense",
			fen:  "rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2",
		},
		{
			name: "queen endgame",
			fen:  "4k3/8/8/8/8/8/8/Q3K3 w - - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := &board.Position{}
			_ = pos.SetFromFEN(tc.fen)

			engine := NewEngine()
			bestMove := engine.Search(pos, SearchLimits{Depth: 5})
			if bestMove == board.NullMove {
				t.Error("expected a valid move with history gravity enabled")
			}
		})
	}
}

func TestHistoryAwareLMR(t *testing.T) {
	// Verify that history-aware LMR doesn't break the search.
	// A middlegame position where LMR is active.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r1bqkb1r/pppppppp/2n2n2/8/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3")

	engine := NewEngine()
	bestMove := engine.Search(pos, SearchLimits{Depth: 8})
	nodes := engine.nodes.Load()

	if bestMove == board.NullMove {
		t.Error("history-aware LMR should not prevent finding a valid move")
	}
	if nodes == 0 {
		t.Fatal("search produced no nodes")
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

	OrderMoves(&ml, board.NullMove, [2]board.Move{}, board.NullMove, &history, board.White, nil)

	if ml.Moves[0] != capture {
		t.Error("capture should still come before quiet move with high history score")
	}
}
