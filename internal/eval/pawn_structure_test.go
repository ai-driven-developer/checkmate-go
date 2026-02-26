package eval

import (
	"checkmatego/internal/board"
	"testing"
)

func TestDoubledPawns(t *testing.T) {
	// White has doubled pawns on e-file.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/4P3/4P3/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	if mg >= 0 || eg >= 0 {
		t.Errorf("doubled white pawns should penalize white, got mg=%d eg=%d", mg, eg)
	}
}

func TestDoubledPawnsBlack(t *testing.T) {
	// Black has doubled pawns on d-file.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/3p4/3p4/8/8/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	if mg <= 0 || eg <= 0 {
		t.Errorf("doubled black pawns should favor white, got mg=%d eg=%d", mg, eg)
	}
}

func TestTriplePawnsPenalizedMore(t *testing.T) {
	// Triple pawns should be penalized more than doubled.
	posDouble := &board.Position{}
	_ = posDouble.SetFromFEN("4k3/8/8/4P3/4P3/8/8/4K3 w - - 0 1")
	mgD, egD := pawnStructureScore(posDouble)

	posTriple := &board.Position{}
	_ = posTriple.SetFromFEN("4k3/8/4P3/4P3/4P3/8/8/4K3 w - - 0 1")
	mgT, egT := pawnStructureScore(posTriple)

	if mgT >= mgD {
		t.Errorf("triple pawns should be worse than doubled: triple mg=%d, double mg=%d", mgT, mgD)
	}
	if egT >= egD {
		t.Errorf("triple pawns should be worse than doubled: triple eg=%d, double eg=%d", egT, egD)
	}
}

func TestIsolatedPawn(t *testing.T) {
	// White pawn on a-file with no pawn on b-file — isolated.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/P7/8/4P3/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	// a-pawn is isolated, e-pawn is also isolated → both penalized.
	if mg >= 0 || eg >= 0 {
		t.Errorf("isolated white pawns should penalize white, got mg=%d eg=%d", mg, eg)
	}
}

func TestNotIsolatedPawn(t *testing.T) {
	// Pawns on d and e files — neither is isolated.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/3PP3/8/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("adjacent pawns should not be penalized, got mg=%d eg=%d", mg, eg)
	}
}

func TestIsolatedPawnBlack(t *testing.T) {
	// Black pawn on h-file isolated.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/4p2p/8/8/8/8/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	// h-pawn is isolated (no g-pawn), e-pawn is also isolated → favors white.
	if mg <= 0 || eg <= 0 {
		t.Errorf("isolated black pawns should favor white, got mg=%d eg=%d", mg, eg)
	}
}

func TestBackwardPawn(t *testing.T) {
	// White: d3, e4. Black: c5, b6.
	// d3 stop sq = d4, attacked by c5 (SouthEast). d3 has no support
	// on adj files at or below rank 2 (e4 is rank 3). → backward.
	// Black pawns b6+c5 support each other, so no isolated penalties.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/1p6/2p5/4P3/3P4/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	if mg >= 0 || eg >= 0 {
		t.Errorf("backward white pawn should penalize white, got mg=%d eg=%d", mg, eg)
	}
}

func TestBackwardPawnBlack(t *testing.T) {
	// Mirror: Black d6, e5. White c4, b3.
	// d6 stop sq = d5, attacked by c4 (NorthEast). d6 has no support
	// on adj files at or above rank 5 (e5 is rank 4). → backward.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/3p4/4p3/2P5/1P6/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	if mg <= 0 || eg <= 0 {
		t.Errorf("backward black pawn should favor white, got mg=%d eg=%d", mg, eg)
	}
}

func TestNoBackwardWhenSupported(t *testing.T) {
	// White pawns on c3 and d3 — d3 has support from c3, not backward.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/8/2PP4/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	// Neither is backward or isolated, no doubled pawns.
	if mg != 0 || eg != 0 {
		t.Errorf("supported pawns should not be penalized, got mg=%d eg=%d", mg, eg)
	}
}

func TestPawnStructureSymmetry(t *testing.T) {
	// White doubled on e-file vs Black doubled on e-file should be symmetric.
	posW := &board.Position{}
	_ = posW.SetFromFEN("4k3/8/8/4P3/4P3/8/8/4K3 w - - 0 1")
	mgW, egW := pawnStructureScore(posW)

	posB := &board.Position{}
	_ = posB.SetFromFEN("4k3/8/4p3/4p3/8/8/8/4K3 w - - 0 1")
	mgB, egB := pawnStructureScore(posB)

	if mgW != -mgB {
		t.Errorf("pawn structure should be symmetric: white mg=%d, black mg=%d", mgW, mgB)
	}
	if egW != -egB {
		t.Errorf("pawn structure should be symmetric: white eg=%d, black eg=%d", egW, egB)
	}
}

func TestPawnStructureStartPosition(t *testing.T) {
	// Start position: no doubled, no isolated, no backward pawns.
	pos := board.NewPosition()
	mg, eg := pawnStructureScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("start position should have 0 pawn structure score, got mg=%d eg=%d", mg, eg)
	}
}

func TestDoubledAndIsolatedPawn(t *testing.T) {
	// White has doubled isolated pawns on a-file — both penalties apply.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/P7/P7/8/8/4K3 w - - 0 1")
	mg, eg := pawnStructureScore(pos)
	// Doubled penalty + 2× isolated penalty.
	expectedMG := -doubledPenaltyMG - 2*isolatedPenaltyMG
	expectedEG := -doubledPenaltyEG - 2*isolatedPenaltyEG
	if mg != expectedMG {
		t.Errorf("doubled isolated a-pawns: expected mg=%d, got mg=%d", expectedMG, mg)
	}
	if eg != expectedEG {
		t.Errorf("doubled isolated a-pawns: expected eg=%d, got eg=%d", expectedEG, eg)
	}
}
