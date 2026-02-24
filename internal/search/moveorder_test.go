package search

import (
	"checkmatego/internal/board"
	"testing"
)

func TestMoveOrderCapturesFirst(t *testing.T) {
	var ml board.MoveList
	// Add a quiet move.
	ml.Add(board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece))
	// Add a capture (pawn takes queen).
	ml.Add(board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Queen))
	// Add another quiet move.
	ml.Add(board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece))

	OrderMoves(&ml)

	// The capture should be first.
	if !ml.Moves[0].IsCapture() {
		t.Error("expected capture to be ordered first")
	}
}

func TestMoveOrderMVVLVA(t *testing.T) {
	var ml board.MoveList
	// Pawn captures pawn (low priority).
	ml.Add(board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Pawn))
	// Pawn captures queen (high priority).
	ml.Add(board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Queen))
	// Knight captures rook (medium priority).
	ml.Add(board.NewMove(board.C3, board.D5, board.FlagCapture, board.Knight, board.Rook))

	OrderMoves(&ml)

	// PxQ should be first, NxR second, PxP last.
	if ml.Moves[0].CapturedPiece() != board.Queen {
		t.Error("PxQ should be first")
	}
	if ml.Moves[1].CapturedPiece() != board.Rook {
		t.Error("NxR should be second")
	}
	if ml.Moves[2].CapturedPiece() != board.Pawn {
		t.Error("PxP should be last")
	}
}
