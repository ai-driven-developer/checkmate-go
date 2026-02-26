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

	OrderMoves(&ml, board.NullMove, [2]board.Move{}, nil, 0, nil)

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

	OrderMoves(&ml, board.NullMove, [2]board.Move{}, nil, 0, nil)

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

func TestMoveOrderHashMoveFirst(t *testing.T) {
	var ml board.MoveList
	quiet1 := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	capture := board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Queen)
	quiet2 := board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece)

	ml.Add(quiet1)
	ml.Add(capture)
	ml.Add(quiet2)

	// quiet2 is the hash move — it should come first despite being quiet.
	OrderMoves(&ml, quiet2, [2]board.Move{}, nil, 0, nil)

	if ml.Moves[0] != quiet2 {
		t.Errorf("hash move should be first, got %v", ml.Moves[0])
	}
	if ml.Moves[1] != capture {
		t.Errorf("capture should be second, got %v", ml.Moves[1])
	}
}

func TestMoveOrderKillerMovePriority(t *testing.T) {
	var ml board.MoveList
	quiet1 := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	capture := board.NewMove(board.D4, board.E5, board.FlagCapture, board.Pawn, board.Queen)
	quiet2 := board.NewMove(board.G1, board.F3, board.FlagQuiet, board.Knight, board.NoPiece)
	quiet3 := board.NewMove(board.B1, board.C3, board.FlagQuiet, board.Knight, board.NoPiece)

	ml.Add(quiet1)
	ml.Add(capture)
	ml.Add(quiet2)
	ml.Add(quiet3)

	// quiet2 is a killer move — should come after captures but before other quiets.
	killers := [2]board.Move{quiet2, board.NullMove}
	OrderMoves(&ml, board.NullMove, killers, nil, 0, nil)

	if ml.Moves[0] != capture {
		t.Errorf("capture should be first, got %v", ml.Moves[0])
	}
	if ml.Moves[1] != quiet2 {
		t.Errorf("killer move should be second, got %v", ml.Moves[1])
	}
}
