package movegen

import (
	"checkmatego/internal/board"
	"testing"
)

func TestStartPositionMoveCount(t *testing.T) {
	pos := board.NewPosition()
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)
	if ml.Count != 20 {
		t.Errorf("start position: expected 20 legal moves, got %d", ml.Count)
	}
}

func TestCastlingMoves(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)

	hasCastle := func(flag uint32) bool {
		for i := 0; i < ml.Count; i++ {
			if ml.Moves[i].Flags() == flag {
				return true
			}
		}
		return false
	}
	if !hasCastle(board.FlagKingCastle) {
		t.Error("expected kingside castling available")
	}
	if !hasCastle(board.FlagQueenCastle) {
		t.Error("expected queenside castling available")
	}
}

func TestCastlingBlockedByCheck(t *testing.T) {
	// King is in check — no castling allowed.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r3k2r/pppp1ppp/8/4q3/8/8/PPPP1PPP/R3K2R w KQkq - 0 1")
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)

	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].IsCastle() {
			t.Error("castling should not be available when in check")
		}
	}
}

func TestEnPassant(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3")
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)

	hasEP := false
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].Flags() == board.FlagEnPassant {
			hasEP = true
			if ml.Moves[i].To() != board.E6 {
				t.Errorf("en passant target should be E6, got %s", ml.Moves[i].To())
			}
		}
	}
	if !hasEP {
		t.Error("expected en passant move available")
	}
}

func TestPromotion(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("8/4P3/8/8/8/8/8/4K2k w - - 0 1")
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)

	promoCount := 0
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].IsPromotion() {
			promoCount++
		}
	}
	// 4 promotions for e7-e8 (Q, R, B, N).
	if promoCount != 4 {
		t.Errorf("expected 4 promotion moves, got %d", promoCount)
	}
}

func TestStalemateNoMoves(t *testing.T) {
	// Stalemate position.
	pos := &board.Position{}
	_ = pos.SetFromFEN("k7/8/1K6/8/8/8/8/8 b - - 0 1")
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)
	// Black king on a8, white king on b6 — black has limited moves but not 0 in this position.
	// Let's use a true stalemate: Ka8, Qb6 white.
	_ = pos.SetFromFEN("k7/8/1Q6/8/8/8/8/4K3 b - - 0 1")
	GenerateLegalMoves(pos, &ml)
	if ml.Count != 0 {
		t.Errorf("expected 0 legal moves in stalemate, got %d", ml.Count)
	}
}

func TestCheckmateNoMoves(t *testing.T) {
	// Scholar's mate position.
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 0 1")
	var ml board.MoveList
	GenerateLegalMoves(pos, &ml)
	if ml.Count != 0 {
		t.Errorf("expected 0 legal moves in checkmate, got %d", ml.Count)
	}
}
