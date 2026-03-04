package nnue

import "checkmatego/internal/board"

// MakeMove pushes the accumulator and incrementally updates features
// based on the move. Must be called BEFORE pos.MakeMove since this
// method reads the side-to-move from the position.
//
// The move itself encodes all needed info (from, to, piece, captured, flags),
// so the position is only used for SideToMove.
func (as *AccumulatorStack) MakeMove(pos *board.Position, m board.Move) {
	as.Push()

	from := m.From()
	to := m.To()
	us := pos.SideToMove
	them := us.Other()
	piece := m.Piece()
	captured := m.CapturedPiece()
	flags := m.Flags()

	switch flags {
	case board.FlagQuiet, board.FlagDoublePawn:
		// Simple move: remove piece from 'from', add to 'to'.
		as.movePiece(us, piece, from, to)

	case board.FlagCapture:
		// Combined remove captured + move our piece in one pass.
		as.capturePiece(us, piece, from, to, them, captured, to)

	case board.FlagEnPassant:
		// Captured pawn is behind the target square.
		var capturedSq board.Square
		if us == board.White {
			capturedSq = to - 8
		} else {
			capturedSq = to + 8
		}
		as.capturePiece(us, board.Pawn, from, to, them, board.Pawn, capturedSq)

	case board.FlagKingCastle:
		// Move king.
		as.movePiece(us, board.King, from, to)
		// Move rook.
		var rookFrom, rookTo board.Square
		if us == board.White {
			rookFrom, rookTo = board.H1, board.F1
		} else {
			rookFrom, rookTo = board.H8, board.F8
		}
		as.movePiece(us, board.Rook, rookFrom, rookTo)

	case board.FlagQueenCastle:
		// Move king.
		as.movePiece(us, board.King, from, to)
		// Move rook.
		var rookFrom, rookTo board.Square
		if us == board.White {
			rookFrom, rookTo = board.A1, board.D1
		} else {
			rookFrom, rookTo = board.A8, board.D8
		}
		as.movePiece(us, board.Rook, rookFrom, rookTo)

	default:
		// Promotions (with or without capture).
		promoPiece := m.PromotionPiece()
		if m.IsCapture() {
			as.removePiece(them, captured, to)
		}
		// Remove pawn from source, add promoted piece at destination.
		as.removePiece(us, board.Pawn, from)
		as.addPiece(us, promoPiece, to)
	}
}

// MakeNullMove pushes the accumulator without changing any features.
func (as *AccumulatorStack) MakeNullMove() {
	as.Push()
}

// UnmakeMove pops the accumulator, restoring the previous state.
func (as *AccumulatorStack) UnmakeMove() {
	as.Pop()
}

// UnmakeNullMove pops the accumulator.
func (as *AccumulatorStack) UnmakeNullMove() {
	as.Pop()
}

// capturePiece combines removePiece + movePiece into a single pass for captures.
func (as *AccumulatorStack) capturePiece(
	movColor board.Color, movPiece board.Piece, from, to board.Square,
	capColor board.Color, capPiece board.Piece, capSq board.Square,
) {
	wCapIdx := FeatureIndex(board.White, capColor, capPiece, capSq)
	bCapIdx := FeatureIndex(board.Black, capColor, capPiece, capSq)
	wFrom := FeatureIndex(board.White, movColor, movPiece, from)
	wTo := FeatureIndex(board.White, movColor, movPiece, to)
	bFrom := FeatureIndex(board.Black, movColor, movPiece, from)
	bTo := FeatureIndex(board.Black, movColor, movPiece, to)
	as.subAddSubBoth(wCapIdx, bCapIdx, wTo, wFrom, bTo, bFrom)
}

// movePiece updates both perspectives for a piece moving from one square to another.
func (as *AccumulatorStack) movePiece(color board.Color, piece board.Piece, from, to board.Square) {
	wFrom := FeatureIndex(board.White, color, piece, from)
	wTo := FeatureIndex(board.White, color, piece, to)
	bFrom := FeatureIndex(board.Black, color, piece, from)
	bTo := FeatureIndex(board.Black, color, piece, to)
	as.addSubBoth(wTo, wFrom, bTo, bFrom)
}

// addPiece updates both perspectives for a piece being placed on a square.
func (as *AccumulatorStack) addPiece(color board.Color, piece board.Piece, sq board.Square) {
	wIdx := FeatureIndex(board.White, color, piece, sq)
	bIdx := FeatureIndex(board.Black, color, piece, sq)
	as.addBoth(wIdx, bIdx)
}

// removePiece updates both perspectives for a piece being removed from a square.
func (as *AccumulatorStack) removePiece(color board.Color, piece board.Piece, sq board.Square) {
	wIdx := FeatureIndex(board.White, color, piece, sq)
	bIdx := FeatureIndex(board.Black, color, piece, sq)
	as.subBoth(wIdx, bIdx)
}
