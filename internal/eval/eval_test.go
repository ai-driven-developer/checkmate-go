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
	phase := gamePhase(pos)
	if phase != totalPhase {
		t.Errorf("start position should have full phase (%d), got %d", totalPhase, phase)
	}
}

func TestGamePhaseEndgame(t *testing.T) {
	// Kings and pawns only — phase should be 0.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/pppppppp/8/8/8/8/PPPPPPPP/4K3 w - - 0 1")
	phase := gamePhase(pos)
	if phase != 0 {
		t.Errorf("K+P endgame should have phase 0, got %d", phase)
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
