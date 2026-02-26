package eval

import (
	"checkmatego/internal/board"
	"testing"
)

func TestKingSafetyStartPosition(t *testing.T) {
	// Start position is symmetric — king safety should be 0.
	pos := board.NewPosition()
	mg, eg := kingSafetyScore(pos)
	if mg != 0 {
		t.Errorf("start position king safety should be 0, got mg=%d", mg)
	}
	if eg != 0 {
		t.Errorf("endgame king safety should always be 0, got eg=%d", eg)
	}
}

func TestKingSafetyEndgameAlwaysZero(t *testing.T) {
	// Even an asymmetric position should return 0 for endgame component.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/8/8/PPPPPPPP/4K3 w - - 0 1")
	_, eg := kingSafetyScore(pos)
	if eg != 0 {
		t.Errorf("endgame king safety should always be 0, got eg=%d", eg)
	}
}

func TestPawnShieldBonus(t *testing.T) {
	// White king on g1 with full pawn shield (f2, g2, h2) vs
	// Black king on e8 with no pawn shield.
	posShielded := &board.Position{}
	_ = posShielded.SetFromFEN("4k3/8/8/8/8/8/5PPP/6K1 w - - 0 1")
	mgShielded, _ := kingSafetyScore(posShielded)

	posExposed := &board.Position{}
	_ = posExposed.SetFromFEN("4k3/8/8/8/8/8/8/6K1 w - - 0 1")
	mgExposed, _ := kingSafetyScore(posExposed)

	// Shielded king should score better than exposed king.
	if mgShielded <= mgExposed {
		t.Errorf("shielded king should score better: shielded=%d, exposed=%d",
			mgShielded, mgExposed)
	}
}

func TestPawnShieldAdvancedPawnLessBonus(t *testing.T) {
	// Pawns on rank 2 (ideal) vs pawns on rank 3 (less bonus).
	posRank2 := &board.Position{}
	_ = posRank2.SetFromFEN("4k3/8/8/8/8/8/5PPP/6K1 w - - 0 1")
	mgRank2, _ := kingSafetyScore(posRank2)

	posRank3 := &board.Position{}
	_ = posRank3.SetFromFEN("4k3/8/8/8/8/5PPP/8/6K1 w - - 0 1")
	mgRank3, _ := kingSafetyScore(posRank3)

	if mgRank2 <= mgRank3 {
		t.Errorf("rank 2 shield should be better than rank 3: rank2=%d, rank3=%d",
			mgRank2, mgRank3)
	}
}

func TestOpenFileNearKingPenalty(t *testing.T) {
	// White king on g1, no f-pawn, black rook on f8.
	// Semi-open f-file facing enemy rook should be penalized.
	posOpen := &board.Position{}
	_ = posOpen.SetFromFEN("4kr2/8/8/8/8/8/6PP/6K1 w - - 0 1")
	mgOpen, _ := kingSafetyScore(posOpen)

	// Same but with f-pawn intact (not semi-open).
	posClosed := &board.Position{}
	_ = posClosed.SetFromFEN("4kr2/8/8/8/8/8/5PPP/6K1 w - - 0 1")
	mgClosed, _ := kingSafetyScore(posClosed)

	if mgOpen >= mgClosed {
		t.Errorf("open file near king should be worse: open=%d, closed=%d",
			mgOpen, mgClosed)
	}
}

func TestKingZoneAttackers(t *testing.T) {
	// Black queen attacking white king zone should give a big penalty.
	posAttacked := &board.Position{}
	_ = posAttacked.SetFromFEN("4k3/8/8/8/8/5q2/5PPP/6K1 w - - 0 1")
	mgAttacked, _ := kingSafetyScore(posAttacked)

	posSafe := &board.Position{}
	_ = posSafe.SetFromFEN("4k3/8/8/8/8/8/5PPP/6K1 w - - 0 1")
	mgSafe, _ := kingSafetyScore(posSafe)

	// Attacked position should score much worse for white.
	if mgAttacked >= mgSafe {
		t.Errorf("king zone attack should penalize: attacked=%d, safe=%d",
			mgAttacked, mgSafe)
	}
}

func TestMultipleAttackersWorseThanOne(t *testing.T) {
	// Queen + rook attacking king zone should be worse than just queen.
	posQueenOnly := &board.Position{}
	_ = posQueenOnly.SetFromFEN("4k3/8/8/8/8/5q2/5PPP/6K1 w - - 0 1")
	mgQueen, _ := kingSafetyScore(posQueenOnly)

	posQueenRook := &board.Position{}
	_ = posQueenRook.SetFromFEN("4k3/8/8/8/8/5q2/5PPP/4r1K1 w - - 0 1")
	mgQueenRook, _ := kingSafetyScore(posQueenRook)

	if mgQueenRook >= mgQueen {
		t.Errorf("queen+rook attack should be worse than queen alone: qr=%d, q=%d",
			mgQueenRook, mgQueen)
	}
}

func TestKingSafetySymmetry(t *testing.T) {
	// Mirrored positions should give opposite scores.
	posW := &board.Position{}
	_ = posW.SetFromFEN("4k3/8/8/8/8/8/5PPP/6K1 w - - 0 1")
	mgW, _ := kingSafetyScore(posW)

	posB := &board.Position{}
	_ = posB.SetFromFEN("6k1/5ppp/8/8/8/8/8/4K3 w - - 0 1")
	mgB, _ := kingSafetyScore(posB)

	if mgW != -mgB {
		t.Errorf("king safety should be symmetric: white=%d, black=%d", mgW, mgB)
	}
}

func TestExposedKingPenalized(t *testing.T) {
	// White king on e4 (exposed, no shield) with enemy queen nearby
	// should be much worse than king on g1 with pawn shield.
	posExposed := &board.Position{}
	_ = posExposed.SetFromFEN("4k3/8/8/3q4/4K3/8/5PPP/8 w - - 0 1")
	mgExposed := kingSafety(posExposed, board.White)

	posShielded := &board.Position{}
	_ = posShielded.SetFromFEN("4k3/8/8/3q4/8/8/5PPP/6K1 w - - 0 1")
	mgShielded := kingSafety(posShielded, board.White)

	if mgExposed >= mgShielded {
		t.Errorf("exposed king should be worse: exposed=%d, shielded=%d",
			mgExposed, mgShielded)
	}
}
