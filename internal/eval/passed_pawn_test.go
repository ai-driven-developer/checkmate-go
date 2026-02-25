package eval

import (
	"checkmatego/internal/board"
	"testing"
)

func TestPassedPawnDetectionWhite(t *testing.T) {
	// White pawn on e5 with no black pawns on d/e/f files ahead — passed.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/4P3/8/8/8/4K3 w - - 0 1")
	mg, eg := passedPawnScore(pos)
	if mg <= 0 || eg <= 0 {
		t.Errorf("white passed pawn on e5 should give positive bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestPassedPawnDetectionBlack(t *testing.T) {
	// Black pawn on d4 with no white pawns on c/d/e files below — passed.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/3p4/8/8/4K3 w - - 0 1")
	mg, eg := passedPawnScore(pos)
	if mg >= 0 || eg >= 0 {
		t.Errorf("black passed pawn on d4 should give negative score, got mg=%d eg=%d", mg, eg)
	}
}

func TestNotPassedPawnBlockedByEnemy(t *testing.T) {
	// White pawn on e5, black pawn on e6 — not passed (blocked on same file).
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/4p3/4P3/8/8/8/4K3 w - - 0 1")
	mg, _ := passedPawnScore(pos)
	// White pawn is not passed, black pawn is not passed either.
	if mg != 0 {
		t.Errorf("no passed pawns expected, got mg=%d", mg)
	}
}

func TestNotPassedPawnSentryOnAdjacentFile(t *testing.T) {
	// White pawn on e5, black pawn on d6 — not passed (sentry on adjacent file).
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/3p4/4P3/8/8/8/4K3 w - - 0 1")
	mg, _ := passedPawnScore(pos)
	if mg != 0 {
		t.Errorf("pawn guarded by sentry should not be passed, got mg=%d", mg)
	}
}

func TestPassedPawnBonusIncreasesByRank(t *testing.T) {
	// A pawn on rank 6 should get a bigger bonus than one on rank 4.
	pos6 := &board.Position{}
	_ = pos6.SetFromFEN("4k3/8/4P3/8/8/8/8/4K3 w - - 0 1")
	mg6, eg6 := passedPawnScore(pos6)

	pos4 := &board.Position{}
	_ = pos4.SetFromFEN("4k3/8/8/8/4P3/8/8/4K3 w - - 0 1")
	mg4, eg4 := passedPawnScore(pos4)

	if mg6 <= mg4 {
		t.Errorf("rank 6 should score higher than rank 4: mg6=%d, mg4=%d", mg6, mg4)
	}
	if eg6 <= eg4 {
		t.Errorf("rank 6 should score higher than rank 4: eg6=%d, eg4=%d", eg6, eg4)
	}
}

func TestPassedPawnSymmetry(t *testing.T) {
	// White pawn on e5 (rank 4 from 0) vs black pawn on e4 (rank 3 from 0, mirror = rank 4).
	posW := &board.Position{}
	_ = posW.SetFromFEN("4k3/8/8/4P3/8/8/8/4K3 w - - 0 1")
	mgW, egW := passedPawnScore(posW)

	posB := &board.Position{}
	_ = posB.SetFromFEN("4k3/8/8/8/4p3/8/8/4K3 w - - 0 1")
	mgB, egB := passedPawnScore(posB)

	if mgW != -mgB {
		t.Errorf("passed pawn score should be symmetric: white mg=%d, black mg=%d", mgW, mgB)
	}
	if egW != -egB {
		t.Errorf("passed pawn score should be symmetric: white eg=%d, black eg=%d", egW, egB)
	}
}

func TestPassedPawnStartPosition(t *testing.T) {
	// Start position has no passed pawns.
	pos := board.NewPosition()
	mg, eg := passedPawnScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("start position should have no passed pawn bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestPassedPawnEndgameBonus(t *testing.T) {
	// In an endgame with a passed pawn, the eval should favor the side with the passer.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/4P3/8/8/4K3 w - - 0 1")
	score := Evaluate(pos)
	// White has a passed pawn; score should be positive.
	if score <= 0 {
		t.Errorf("white with passed pawn should have positive eval, got %d", score)
	}
}

func TestMultiplePassedPawns(t *testing.T) {
	// Two white passed pawns should score higher than one.
	pos1 := &board.Position{}
	_ = pos1.SetFromFEN("4k3/8/8/4P3/8/8/8/4K3 w - - 0 1")
	mg1, eg1 := passedPawnScore(pos1)

	pos2 := &board.Position{}
	_ = pos2.SetFromFEN("4k3/8/8/3PP3/8/8/8/4K3 w - - 0 1")
	mg2, eg2 := passedPawnScore(pos2)

	if mg2 <= mg1 {
		t.Errorf("two passed pawns should score higher: one=%d, two=%d", mg1, mg2)
	}
	if eg2 <= eg1 {
		t.Errorf("two passed pawns should score higher: one=%d, two=%d", eg1, eg2)
	}
}
