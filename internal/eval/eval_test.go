package eval

import (
	"checkmatego/internal/board"
	"testing"
)

func TestEvalStartPosition(t *testing.T) {
	pos := board.NewPosition()
	score := Evaluate(pos)
	// Start position should be roughly equal (within a small margin for PST asymmetry).
	if score < -30 || score > 30 {
		t.Errorf("start position eval should be ~0, got %d", score)
	}
}

func TestEvalSymmetry(t *testing.T) {
	// Start position is symmetric — eval from White's perspective should equal eval from Black's.
	pos := board.NewPosition()
	whiteScore := Evaluate(pos)

	// Flip: set side to move to black.
	pos.SideToMove = board.Black
	pos.Hash = 0 // hash doesn't matter for eval
	blackScore := Evaluate(pos)

	// Should be opposite signs (or both zero).
	if whiteScore != -blackScore {
		t.Errorf("symmetric position: white=%d, black=%d, expected opposite", whiteScore, blackScore)
	}
}

func TestEvalMaterialAdvantage(t *testing.T) {
	// White up a queen.
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnb1kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	score := Evaluate(pos)
	if score < 800 {
		t.Errorf("white up a queen should score > 800cp, got %d", score)
	}
}

func TestEvalMaterialBalance(t *testing.T) {
	pos := board.NewPosition()
	mat := materialBalance(pos)
	if mat != 0 {
		t.Errorf("start position material should be 0, got %d", mat)
	}
}

func TestGamePhaseStartPosition(t *testing.T) {
	pos := board.NewPosition()
	if pos.Phase != totalPhase {
		t.Errorf("start position should have full phase (%d), got %d", totalPhase, pos.Phase)
	}
}

func TestGamePhaseEndgame(t *testing.T) {
	// Kings and pawns only — phase should be 0.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/pppppppp/8/8/8/8/PPPPPPPP/4K3 w - - 0 1")
	if pos.Phase != 0 {
		t.Errorf("K+P endgame should have phase 0, got %d", pos.Phase)
	}
}

func TestTaperedEvalKingCentralizesInEndgame(t *testing.T) {
	// In a K+P endgame, a centralized king should be valued higher than
	// a king on the back rank, because the endgame king PST rewards center.
	posCentered := &board.Position{}
	_ = posCentered.SetFromFEN("4k3/8/8/8/4K3/8/8/8 w - - 0 1")
	scoreCentered := Evaluate(posCentered)

	posEdge := &board.Position{}
	_ = posEdge.SetFromFEN("4k3/8/8/8/8/8/8/4K3 w - - 0 1")
	scoreEdge := Evaluate(posEdge)

	if scoreCentered <= scoreEdge {
		t.Errorf("centralized king should score better in endgame: center=%d, edge=%d",
			scoreCentered, scoreEdge)
	}
}

func TestTaperedEvalMiddlegamePrefersKingSafety(t *testing.T) {
	// In a full middlegame with all pieces, king safety (back rank)
	// should be preferred over centralizing.
	posBack := &board.Position{}
	_ = posBack.SetFromFEN("rnbq1bnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	scoreBack := Evaluate(posBack)

	// Same material but white king on e4 (very exposed).
	posExposed := &board.Position{}
	_ = posExposed.SetFromFEN("rnbq1bnr/pppppppp/8/8/4K3/8/PPPPPPPP/RNBQ1BNR w kq - 0 1")
	scoreExposed := Evaluate(posExposed)

	if scoreBack <= scoreExposed {
		t.Errorf("middlegame king on back rank should score better: back=%d, exposed=%d",
			scoreBack, scoreExposed)
	}
}

func TestRookMobility(t *testing.T) {
	// Rook on open file (e4, no pawns) should have better mobility
	// than rook boxed in on a1.
	posOpen := &board.Position{}
	_ = posOpen.SetFromFEN("4k3/8/8/8/4R3/8/8/4K3 w - - 0 1")
	scoreOpen := mobilityScore(posOpen)

	posBoxed := &board.Position{}
	_ = posBoxed.SetFromFEN("4k3/8/8/8/8/8/1P6/RP2K3 w - - 0 1")
	scoreBoxed := mobilityScore(posBoxed)

	if scoreOpen <= scoreBoxed {
		t.Errorf("open rook should have more mobility: open=%d, boxed=%d",
			scoreOpen, scoreBoxed)
	}
}

func TestQueenMobility(t *testing.T) {
	// Queen in the center should have more mobility than on a1.
	posCenter := &board.Position{}
	_ = posCenter.SetFromFEN("4k3/8/8/8/4Q3/8/8/4K3 w - - 0 1")
	scoreCenter := mobilityScore(posCenter)

	posCorner := &board.Position{}
	_ = posCorner.SetFromFEN("4k3/8/8/8/8/8/8/Q3K3 w - - 0 1")
	scoreCorner := mobilityScore(posCorner)

	if scoreCenter <= scoreCorner {
		t.Errorf("central queen should have more mobility: center=%d, corner=%d",
			scoreCenter, scoreCorner)
	}
}

func TestBishopPairBonus(t *testing.T) {
	// White with two bishops vs White with one bishop — pair should score higher.
	posPair := &board.Position{}
	_ = posPair.SetFromFEN("4k3/8/8/8/8/8/8/2B1KB2 w - - 0 1")
	matPair := materialBalance(posPair)

	posSingle := &board.Position{}
	_ = posSingle.SetFromFEN("4k3/8/8/8/8/8/8/2B1K3 w - - 0 1")
	matSingle := materialBalance(posSingle)

	// Pair should be worth more than just an extra bishop's value.
	diff := matPair - matSingle
	if diff <= PieceValue[board.Bishop] {
		t.Errorf("bishop pair should add bonus beyond piece value: diff=%d, bishop=%d",
			diff, PieceValue[board.Bishop])
	}
}

func TestBishopPairSymmetry(t *testing.T) {
	// Both sides with bishop pair — bonus should cancel.
	pos := &board.Position{}
	_ = pos.SetFromFEN("2b1kb2/8/8/8/8/8/8/2B1KB2 w - - 0 1")
	mat := materialBalance(pos)
	if mat != 0 {
		t.Errorf("symmetric bishop pairs should give 0, got %d", mat)
	}
}

func TestNoBishopPairWithOneBishop(t *testing.T) {
	// One bishop each — no pair bonus.
	pos := &board.Position{}
	_ = pos.SetFromFEN("2b1k3/8/8/8/8/8/8/2B1K3 w - - 0 1")
	mat := materialBalance(pos)
	if mat != 0 {
		t.Errorf("one bishop each should give 0, got %d", mat)
	}
}

func TestPSTMirror(t *testing.T) {
	if MirrorSquare(board.A1) != board.A8 {
		t.Errorf("mirror A1 should be A8, got %s", MirrorSquare(board.A1))
	}
	if MirrorSquare(board.E4) != board.E5 {
		t.Errorf("mirror E4 should be E5, got %s", MirrorSquare(board.E4))
	}
	if MirrorSquare(board.H8) != board.H1 {
		t.Errorf("mirror H8 should be H1, got %s", MirrorSquare(board.H8))
	}
}

func TestPawnCacheProbeStore(t *testing.T) {
	pc := NewPawnCache(1024)

	// Miss on empty cache.
	hit, _, _ := pc.Probe(0x12345678)
	if hit {
		t.Error("expected cache miss on empty cache")
	}

	// Store and probe back.
	pc.Store(0x12345678, 42, -15)
	hit, mg, eg := pc.Probe(0x12345678)
	if !hit {
		t.Error("expected cache hit after store")
	}
	if mg != 42 || eg != -15 {
		t.Errorf("expected mg=42, eg=-15, got mg=%d, eg=%d", mg, eg)
	}

	// Different key should miss.
	hit, _, _ = pc.Probe(0xAAAABBBB)
	if hit {
		t.Error("expected cache miss for different key")
	}
}

func TestPawnCacheOverwrite(t *testing.T) {
	pc := NewPawnCache(1024)

	pc.Store(0xABC, 10, 20)
	pc.Store(0xABC, 30, 40)

	hit, mg, eg := pc.Probe(0xABC)
	if !hit {
		t.Error("expected cache hit")
	}
	if mg != 30 || eg != 40 {
		t.Errorf("expected overwritten values mg=30, eg=40, got mg=%d, eg=%d", mg, eg)
	}
}

func TestEvaluateWithCacheConsistency(t *testing.T) {
	// EvaluateWithCache should return the same score as Evaluate (no cache).
	positions := []string{
		board.StartFEN,
		"rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2",
		"r1bqkbnr/pppppppp/2n5/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 1 2",
		"2r3k1/pp3ppp/2n1bn2/8/2B1P3/2N2N2/PPP2PPP/R3K2R w KQ - 4 12",
	}

	pc := NewPawnCache(4096)

	for _, fen := range positions {
		pos := &board.Position{}
		if err := pos.SetFromFEN(fen); err != nil {
			t.Fatalf("bad FEN %q: %v", fen, err)
		}
		scoreNil := Evaluate(pos)
		scoreCache := EvaluateWithCache(pos, pc)

		if scoreNil != scoreCache {
			t.Errorf("FEN %q: Evaluate=%d, EvaluateWithCache=%d", fen, scoreNil, scoreCache)
		}
	}
}

func TestIncrementalPSTMatchesFromScratch(t *testing.T) {
	// The incremental PST in Position must match pstBalanceTapered for various positions.
	positions := []string{
		board.StartFEN,
		"rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2",
		"r1bqkbnr/pppppppp/2n5/8/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 2 2",
		"r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
		"8/4P3/8/8/8/8/8/4K2k w - - 0 1",
		"4k3/pppppppp/8/8/8/8/PPPPPPPP/4K3 w - - 0 1",
	}
	for _, fen := range positions {
		pos := &board.Position{}
		if err := pos.SetFromFEN(fen); err != nil {
			t.Fatalf("bad FEN %q: %v", fen, err)
		}
		wantMG, wantEG := pstBalanceTapered(pos)
		if pos.PSTMG != wantMG || pos.PSTEG != wantEG {
			t.Errorf("FEN %q: incremental mg=%d eg=%d, from-scratch mg=%d eg=%d",
				fen, pos.PSTMG, pos.PSTEG, wantMG, wantEG)
		}
	}
}

func TestPawnCacheHitOnSecondCall(t *testing.T) {
	// Calling EvaluateWithCache twice on the same position should hit the cache.
	pos := board.NewPosition()
	pc := NewPawnCache(4096)

	score1 := EvaluateWithCache(pos, pc)
	// The first call should have populated the cache.
	hit, _, _ := pc.Probe(pos.PawnHash)
	if !hit {
		t.Error("expected pawn cache hit after first evaluation")
	}

	score2 := EvaluateWithCache(pos, pc)
	if score1 != score2 {
		t.Errorf("expected same score on repeated eval: %d vs %d", score1, score2)
	}
}
