package board

import "testing"

func TestMoveEncoding(t *testing.T) {
	m := NewMove(E2, E4, FlagDoublePawn, Pawn, NoPiece)
	if m.From() != E2 {
		t.Errorf("expected from E2, got %s", m.From())
	}
	if m.To() != E4 {
		t.Errorf("expected to E4, got %s", m.To())
	}
	if m.Flags() != FlagDoublePawn {
		t.Errorf("expected flag DoublePawn, got %d", m.Flags())
	}
	if m.Piece() != Pawn {
		t.Errorf("expected piece Pawn, got %d", m.Piece())
	}
	if m.CapturedPiece() != NoPiece {
		t.Errorf("expected no captured piece, got %d", m.CapturedPiece())
	}
	if m.String() != "e2e4" {
		t.Errorf("expected 'e2e4', got '%s'", m.String())
	}
}

func TestMoveCapture(t *testing.T) {
	m := NewMove(D4, E5, FlagCapture, Knight, Pawn)
	if !m.IsCapture() {
		t.Error("expected capture")
	}
	if m.CapturedPiece() != Pawn {
		t.Errorf("expected captured pawn, got %d", m.CapturedPiece())
	}
}

func TestMovePromotion(t *testing.T) {
	m := NewMove(E7, E8, FlagPromoQueen, Pawn, NoPiece)
	if !m.IsPromotion() {
		t.Error("expected promotion")
	}
	if m.PromotionPiece() != Queen {
		t.Errorf("expected queen promotion, got %d", m.PromotionPiece())
	}
	if m.String() != "e7e8q" {
		t.Errorf("expected 'e7e8q', got '%s'", m.String())
	}
}

func TestMovePromotionCapture(t *testing.T) {
	m := NewMove(D7, C8, FlagPromoCaptureKnight, Pawn, Rook)
	if !m.IsCapture() {
		t.Error("expected capture")
	}
	if !m.IsPromotion() {
		t.Error("expected promotion")
	}
	if m.PromotionPiece() != Knight {
		t.Errorf("expected knight promotion, got %d", m.PromotionPiece())
	}
	if m.String() != "d7c8n" {
		t.Errorf("expected 'd7c8n', got '%s'", m.String())
	}
}

func TestMoveCastle(t *testing.T) {
	m := NewMove(E1, G1, FlagKingCastle, King, NoPiece)
	if !m.IsCastle() {
		t.Error("expected castle")
	}
	if m.String() != "e1g1" {
		t.Errorf("expected 'e1g1', got '%s'", m.String())
	}
}

func TestNullMove(t *testing.T) {
	if NullMove.String() != "0000" {
		t.Errorf("expected '0000', got '%s'", NullMove.String())
	}
}

func TestMoveList(t *testing.T) {
	var ml MoveList
	ml.Add(NewMove(E2, E4, FlagDoublePawn, Pawn, NoPiece))
	ml.Add(NewMove(D2, D4, FlagDoublePawn, Pawn, NoPiece))
	if ml.Count != 2 {
		t.Errorf("expected count 2, got %d", ml.Count)
	}
	ml.Clear()
	if ml.Count != 0 {
		t.Errorf("expected count 0 after clear, got %d", ml.Count)
	}
}
