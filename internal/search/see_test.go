package search

import (
	"checkmatego/internal/board"
	"testing"
)

func TestSEEUndefendedCapture(t *testing.T) {
	// White pawn takes undefended black queen.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/4q3/3P4/8/8/4K3 w - - 0 1")

	m := board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Queen)
	score := SEE(pos, m)

	if score != 900 {
		t.Errorf("PxQ undefended: expected 900, got %d", score)
	}
}

func TestSEEQueenTakesDefendedPawn(t *testing.T) {
	// White queen takes black pawn defended by pawn. No white recapture.
	// QxP (+100), PxQ (-900). Net = -800.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/4p3/3p4/8/8/8/3QK3 w - - 0 1")

	m := board.NewMove(board.D1, board.D5, board.FlagCapture, board.Queen, board.Pawn)
	score := SEE(pos, m)

	if score != -800 {
		t.Errorf("QxP defended by pawn: expected -800, got %d", score)
	}
}

func TestSEEPawnTakesDefendedKnight(t *testing.T) {
	// White pawn takes black knight defended by black pawn.
	// PxN (+320), PxP (-100). Net = 220.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/5p2/4n3/3P4/8/8/4K3 w - - 0 1")

	m := board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Knight)
	score := SEE(pos, m)

	if score != 220 {
		t.Errorf("PxN defended by pawn: expected 220, got %d", score)
	}
}

func TestSEEKnightTakesDefendedPawn(t *testing.T) {
	// White knight takes black pawn defended by pawn.
	// NxP (+100), PxN (-320). Net = -220.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/4p3/3p4/8/2N5/8/4K3 w - - 0 1")

	m := board.NewMove(board.C3, board.D5, board.FlagCapture, board.Knight, board.Pawn)
	score := SEE(pos, m)

	if score != -220 {
		t.Errorf("NxP defended by pawn: expected -220, got %d", score)
	}
}

func TestSEERookExchangeEqual(t *testing.T) {
	// White rook takes black rook, defended by another black rook.
	// RxR (+500), RxR (-500). Net = 0.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4r2k/8/8/4r3/8/8/8/4R2K w - - 0 1")

	m := board.NewMove(board.E1, board.E5, board.FlagCapture, board.Rook, board.Rook)
	score := SEE(pos, m)

	if score != 0 {
		t.Errorf("RxR defended by rook: expected 0, got %d", score)
	}
}

func TestSEEBishopExchangeForKnight(t *testing.T) {
	// White bishop takes black knight defended by black pawn.
	// BxN (+320), PxB (-330). Net = -10.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/5p2/4n3/3B4/8/8/4K3 w - - 0 1")

	m := board.NewMove(board.D4, board.E5, board.FlagCapture, board.Bishop, board.Knight)

	// Verify: e5 is attacked by f6 pawn (black pawn attacks diagonally toward lower ranks).
	score := SEE(pos, m)

	// BxN has a 10cp cost because bishop (330) > knight (320).
	if score != -10 {
		t.Errorf("BxN defended by pawn: expected -10, got %d", score)
	}
}

func TestSEEComplexExchange(t *testing.T) {
	// White rook takes black knight defended by black pawn, but white has bishop backup.
	// RxN (+320), PxR (-500), BxP (+100). Net = -80.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/4p3/3n4/8/1B6/8/3RK3 w - - 0 1")

	m := board.NewMove(board.D1, board.D5, board.FlagCapture, board.Rook, board.Knight)
	score := SEE(pos, m)

	if score != -80 {
		t.Errorf("RxN (defended by pawn, bishop backup): expected -80, got %d", score)
	}
}

func TestSEENonCapture(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")

	m := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	score := SEE(pos, m)

	if score != 0 {
		t.Errorf("non-capture: expected 0, got %d", score)
	}
}

func TestSEEEnPassant(t *testing.T) {
	// White pawn captures en passant.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1")

	m := board.NewMove(board.E5, board.D6, board.FlagEnPassant, board.Pawn, board.Pawn)
	score := SEE(pos, m)

	// Undefended en passant: just pawn value.
	if score != 100 {
		t.Errorf("en passant undefended: expected 100, got %d", score)
	}
}

func TestSEEXRayDiscovery(t *testing.T) {
	// White rook on e1, white rook on e2 (x-ray behind e2).
	// Black rook on e5, defended by black rook on e8.
	// RxR(e2xe5, +500), RxR(e8xe5, -500), RxR(e1xe5, +500), no more. Net = 500.
	pos := &board.Position{}
	_ = pos.SetFromFEN("4r2k/8/8/4r3/8/8/4R3/4R2K w - - 0 1")

	m := board.NewMove(board.E2, board.E5, board.FlagCapture, board.Rook, board.Rook)
	score := SEE(pos, m)

	// RxR, RxR, RxR — white has 2 rooks, black has 2 rooks.
	// +500 -500 +500 -500 = 0? Wait, let me think.
	// White e2 takes black e5 (x-ray: white e1 behind).
	// Black e8 takes white rook on e5 (x-ray: none for black).
	// White e1 takes black rook on e5.
	// No more black attackers.
	// Net: +500 -500 +500 = 500.
	if score != 500 {
		t.Errorf("x-ray rook discovery: expected 500, got %d", score)
	}
}

func TestSEEPromotionCapture(t *testing.T) {
	// White pawn promotes and captures black rook.
	pos := &board.Position{}
	_ = pos.SetFromFEN("3rk3/4P3/8/8/8/8/8/4K3 w - - 0 1")

	m := board.NewMove(board.E7, board.D8, board.FlagPromoCaptureQueen, board.Pawn, board.Rook)
	score := SEE(pos, m)

	// Captures rook (500) + gains queen (900) - loses pawn (100) = 1300.
	// But if black king takes back: loses queen (900).
	// Black king on e8 attacks d8, so KxQ is possible.
	// Net: 500 + (900 - 100) - 900 = 400.
	// gain[0] = 500 + 800 = 1300. attacker = Queen.
	// gain[1] = 900 - 1300 = -400. Black king takes.
	// No white attacker. Minimax: gain[0] = -max(-1300, -400) = -(-400) = 400.
	if score != 400 {
		t.Errorf("promotion capture: expected 400, got %d", score)
	}
}

func TestSEEKnightExchangeEqual(t *testing.T) {
	// NxN where the captured knight is defended by another knight.
	// NxN (+320), NxN (-320). Net = 0.
	// Black knight on d7 defends e5 (d7→e5 is a valid knight move).
	pos := &board.Position{}
	_ = pos.SetFromFEN("4k3/3n4/8/4n3/3N4/8/8/4K3 w - - 0 1")

	m := board.NewMove(board.D4, board.E5, board.FlagCapture, board.Knight, board.Knight)
	score := SEE(pos, m)

	if score != 0 {
		t.Errorf("NxN defended by knight: expected 0, got %d", score)
	}
}
