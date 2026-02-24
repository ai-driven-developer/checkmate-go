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
