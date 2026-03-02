package eval

import (
	"checkmatego/internal/board"
	"testing"
)

// --- Knight outpost tests ---

func TestOutpostKnightOnProtectedSquare(t *testing.T) {
	// White knight on d5 supported by pawn on e4, no black pawns on c/e files above.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/3N4/4P3/8/8/4K3 w - - 0 1")
	mg, eg := outpostScore(pos)
	if mg <= 0 || eg <= 0 {
		t.Errorf("knight on d5 supported by e4 pawn should get outpost bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestOutpostKnightNotSupportedByPawn(t *testing.T) {
	// White knight on d5 but no friendly pawn supports it.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/3N4/8/8/8/4K3 w - - 0 1")
	mg, eg := outpostScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("unsupported knight should get no outpost bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestOutpostKnightAttackableByEnemyPawn(t *testing.T) {
	// White knight on d5 supported by e4, but black pawn on c6 can attack.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/2p5/3N4/4P3/8/8/4K3 w - - 0 1")
	mg, eg := outpostScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("knight attackable by enemy pawn should get no outpost bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestOutpostKnightOnLowRank(t *testing.T) {
	// White knight on d3 (rank 3, too low for outpost) supported by e2.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/8/3N4/4P3/4K3 w - - 0 1")
	mg, eg := outpostScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("knight on rank 3 should not get outpost bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestOutpostSymmetry(t *testing.T) {
	// White knight outpost on d5 vs black knight outpost on d4.
	posW := &board.Position{}
	_ = posW.SetFromFEN("4k3/8/8/3N4/4P3/8/8/4K3 w - - 0 1")
	mgW, egW := outpostScore(posW)

	posB := &board.Position{}
	_ = posB.SetFromFEN("4k3/8/4p3/3n4/8/8/8/4K3 w - - 0 1")
	mgB, egB := outpostScore(posB)

	if mgW != -mgB {
		t.Errorf("outpost score should be symmetric: white mg=%d, black mg=%d", mgW, mgB)
	}
	if egW != -egB {
		t.Errorf("outpost score should be symmetric: white eg=%d, black eg=%d", egW, egB)
	}
}

func TestOutpostStartPosition(t *testing.T) {
	pos := board.NewPosition()
	mg, eg := outpostScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("start position should have no outpost bonus, got mg=%d eg=%d", mg, eg)
	}
}

// --- Rook evaluation tests ---

func TestRookOnOpenFile(t *testing.T) {
	// White rook on e-file with no pawns on that file.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/pppp1ppp/8/8/8/8/PPPP1PPP/4RK2 w - - 0 1")
	mg, eg := rookScore(pos)
	if mg < rookOpenFileMG || eg < rookOpenFileEG {
		t.Errorf("rook on open file should get open file bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestRookOnSemiOpenFile(t *testing.T) {
	// White rook on e-file, no white pawn but black pawn on e7.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/pppppppp/8/8/8/8/PPPP1PPP/4RK2 w - - 0 1")
	mg, eg := rookScore(pos)
	if mg < rookSemiOpenFileMG || eg < rookSemiOpenFileEG {
		t.Errorf("rook on semi-open file should get semi-open bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestRookOnClosedFile(t *testing.T) {
	// White rook on e-file with white pawn on e2.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/pppppppp/8/8/8/8/PPPPPPPP/4RK2 w - - 0 1")
	mg, eg := rookScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("rook on closed file should get no file bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestRookOnSeventhRank(t *testing.T) {
	// White rook on e7 (7th rank).
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/4R3/8/8/8/8/8/4K3 w - - 0 1")
	mg, eg := rookScore(pos)
	// Should include 7th rank bonus + open file bonus.
	if mg < rookSeventhRankMG {
		t.Errorf("rook on 7th rank should get bonus, got mg=%d", mg)
	}
	if eg < rookSeventhRankEG {
		t.Errorf("rook on 7th rank should get bonus, got eg=%d", eg)
	}
}

func TestRookScoreSymmetry(t *testing.T) {
	// White rook on open e-file vs black rook on open e-file.
	posW := &board.Position{}
	_ = posW.SetFromFEN("4k3/8/8/8/8/8/8/4RK2 w - - 0 1")
	mgW, egW := rookScore(posW)

	posB := &board.Position{}
	_ = posB.SetFromFEN("2k1r3/8/8/8/8/8/8/4K3 w - - 0 1")
	mgB, egB := rookScore(posB)

	if mgW != -mgB {
		t.Errorf("rook score should be symmetric: white mg=%d, black mg=%d", mgW, mgB)
	}
	if egW != -egB {
		t.Errorf("rook score should be symmetric: white eg=%d, black eg=%d", egW, egB)
	}
}

func TestRookStartPosition(t *testing.T) {
	pos := board.NewPosition()
	mg, eg := rookScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("start position should have no rook bonus (both sides equal), got mg=%d eg=%d", mg, eg)
	}
}

// --- King-passer distance tests ---

func TestKingPasserFriendlyKingClose(t *testing.T) {
	// White passed pawn on e5, white king on e4 (distance 1) = good.
	pos := &board.Position{}
	_ = pos.SetFromFEN("7k/8/8/4P3/4K3/8/8/8 w - - 0 1")
	_, eg := kingPasserDistanceScore(pos)
	if eg <= 0 {
		t.Errorf("friendly king close to passed pawn should give EG bonus, got eg=%d", eg)
	}
}

func TestKingPasserFriendlyKingFar(t *testing.T) {
	// White passed pawn on e5, white king on a1 (far away).
	posFar := &board.Position{}
	_ = posFar.SetFromFEN("7k/8/8/4P3/8/8/8/K7 w - - 0 1")
	_, egFar := kingPasserDistanceScore(posFar)

	posClose := &board.Position{}
	_ = posClose.SetFromFEN("7k/8/8/4P3/4K3/8/8/8 w - - 0 1")
	_, egClose := kingPasserDistanceScore(posClose)

	if egClose <= egFar {
		t.Errorf("close king should score better: close=%d, far=%d", egClose, egFar)
	}
}

func TestKingPasserEnemyKingFar(t *testing.T) {
	// White passed pawn on e5, enemy king on h8 (far) vs enemy king on f6 (close).
	posFar := &board.Position{}
	_ = posFar.SetFromFEN("7k/8/8/4P3/4K3/8/8/8 w - - 0 1")
	_, egFar := kingPasserDistanceScore(posFar)

	posClose := &board.Position{}
	_ = posClose.SetFromFEN("8/8/5k2/4P3/4K3/8/8/8 w - - 0 1")
	_, egClose := kingPasserDistanceScore(posClose)

	if egFar <= egClose {
		t.Errorf("far enemy king should score better: far=%d, close=%d", egFar, egClose)
	}
}

func TestKingPasserSymmetry(t *testing.T) {
	// Mirrored positions: white pawn e5 + wK e4 + bK e8
	// vs black pawn e4 + bK e5 + wK e1.
	posW := &board.Position{}
	_ = posW.SetFromFEN("4k3/8/8/4P3/4K3/8/8/8 w - - 0 1")
	_, egW := kingPasserDistanceScore(posW)

	posB := &board.Position{}
	_ = posB.SetFromFEN("8/8/8/4k3/4p3/8/8/4K3 w - - 0 1")
	_, egB := kingPasserDistanceScore(posB)

	if egW != -egB {
		t.Errorf("king passer distance should be symmetric: white eg=%d, black eg=%d", egW, egB)
	}
}

func TestKingPasserNoPassedPawns(t *testing.T) {
	// No passed pawns — should return 0.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/pppppppp/8/8/8/8/PPPPPPPP/4K3 w - - 0 1")
	mg, eg := kingPasserDistanceScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("no passed pawns should give 0, got mg=%d eg=%d", mg, eg)
	}
}

func TestKingPasserStartPosition(t *testing.T) {
	pos := board.NewPosition()
	mg, eg := kingPasserDistanceScore(pos)
	if mg != 0 || eg != 0 {
		t.Errorf("start position should have no king-passer bonus, got mg=%d eg=%d", mg, eg)
	}
}

func TestKingPasserEndgameOnlyTerm(t *testing.T) {
	// The MG component should always be 0.
	pos := &board.Position{}
	_ = pos.SetFromFEN("7k/8/8/4P3/4K3/8/8/8 w - - 0 1")
	mg, _ := kingPasserDistanceScore(pos)
	if mg != 0 {
		t.Errorf("king-passer distance should have no MG component, got mg=%d", mg)
	}
}
