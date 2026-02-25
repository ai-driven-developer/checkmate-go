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

// --- GenerateCaptures tests ---

func TestCapturesOnlyReturnsCaptures(t *testing.T) {
	// A position with both captures and quiet moves available.
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2")

	var ml board.MoveList
	GenerateCaptures(pos, &ml)

	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		if !m.IsCapture() && !m.IsPromotion() {
			t.Errorf("GenerateCaptures returned non-capture/non-promotion move: %s (flags=%d)",
				m, m.Flags())
		}
	}
}

func TestCapturesStartPosition(t *testing.T) {
	// Starting position has no captures.
	pos := board.NewPosition()
	var ml board.MoveList
	GenerateCaptures(pos, &ml)

	if ml.Count != 0 {
		t.Errorf("starting position should have 0 captures, got %d", ml.Count)
	}
}

func TestCapturesIncludesEnPassant(t *testing.T) {
	// En passant should be included in captures.
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3")

	var ml board.MoveList
	GenerateCaptures(pos, &ml)

	hasEP := false
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].Flags() == board.FlagEnPassant {
			hasEP = true
		}
	}
	if !hasEP {
		t.Error("GenerateCaptures should include en passant")
	}
}

func TestCapturesIncludesPromotions(t *testing.T) {
	// A pawn about to promote should generate promotion moves.
	pos := &board.Position{}
	_ = pos.SetFromFEN("2k5/4P3/8/8/8/8/8/4K3 w - - 0 1")

	var ml board.MoveList
	GenerateCaptures(pos, &ml)

	promoCount := 0
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].IsPromotion() {
			promoCount++
		}
	}
	if promoCount != 4 {
		t.Errorf("expected 4 promotion moves in captures, got %d", promoCount)
	}
}

func TestCapturesSubsetOfLegalMoves(t *testing.T) {
	// All captures returned by GenerateCaptures should also be in GenerateLegalMoves.
	positions := []string{
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", // Kiwi Pete
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
	}

	for _, fen := range positions {
		pos := &board.Position{}
		_ = pos.SetFromFEN(fen)

		var captures board.MoveList
		GenerateCaptures(pos, &captures)

		var allMoves board.MoveList
		GenerateLegalMoves(pos, &allMoves)

		for i := 0; i < captures.Count; i++ {
			found := false
			for j := 0; j < allMoves.Count; j++ {
				if captures.Moves[i] == allMoves.Moves[j] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("capture %s not found in legal moves for FEN %s",
					captures.Moves[i], fen)
			}
		}
	}
}

func TestCapturesCountMatchesLegalCaptures(t *testing.T) {
	// The number of captures from GenerateCaptures should equal the number
	// of capture/promotion moves in GenerateLegalMoves.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	var captures board.MoveList
	GenerateCaptures(pos, &captures)

	var allMoves board.MoveList
	GenerateLegalMoves(pos, &allMoves)

	legalCaptures := 0
	for i := 0; i < allMoves.Count; i++ {
		m := allMoves.Moves[i]
		if m.IsCapture() || m.IsPromotion() {
			legalCaptures++
		}
	}

	if captures.Count != legalCaptures {
		t.Errorf("GenerateCaptures count %d != legal captures count %d",
			captures.Count, legalCaptures)
	}
}
