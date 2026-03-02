package eval

import "checkmatego/internal/board"

// MirrorSquare flips the rank for Black PST lookups.
func MirrorSquare(sq board.Square) board.Square {
	return sq ^ 56
}

// pstBalanceTapered computes separate middlegame and endgame PST scores
// from White's perspective by iterating all pieces.
// Kept for testing/validation; the search uses the incremental Position.PSTMG/PSTEG.
func pstBalanceTapered(pos *board.Position) (mg, eg int) {
	for piece := board.Pawn; piece <= board.King; piece++ {
		// White pieces — use tables directly.
		bb := pos.ColorPieces(board.White, piece)
		for bb != 0 {
			sq := bb.PopLSB()
			mg += board.PiecePSTMG[piece][sq]
			eg += board.PiecePSTEG[piece][sq]
		}
		// Black pieces — mirror square.
		bb = pos.ColorPieces(board.Black, piece)
		for bb != 0 {
			sq := bb.PopLSB()
			mg -= board.PiecePSTMG[piece][MirrorSquare(sq)]
			eg -= board.PiecePSTEG[piece][MirrorSquare(sq)]
		}
	}
	return mg, eg
}
