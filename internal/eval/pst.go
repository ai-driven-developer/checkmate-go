package eval

import "checkmatego/internal/board"

// Piece-square tables from the Simplified Evaluation Function.
// Values are from White's perspective; rank 1 at index 0 (LERF layout).
// For Black, we mirror the square vertically via sq^56.

var pawnPST = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0, // rank 1 (never occupied by pawn)
	5, 10, 10, -20, -20, 10, 10, 5, // rank 2
	5, -5, -10, 0, 0, -10, -5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, 5, 10, 25, 25, 10, 5, 5,
	10, 10, 20, 30, 30, 20, 10, 10,
	50, 50, 50, 50, 50, 50, 50, 50, // rank 7
	0, 0, 0, 0, 0, 0, 0, 0, // rank 8 (never occupied — promotion)
}

var knightPST = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var bishopPST = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var rookPST = [64]int{
	0, 0, 0, 5, 5, 0, 0, 0,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	5, 10, 10, 10, 10, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var queenPST = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-5, 0, 5, 5, 5, 5, 0, -5,
	0, 0, 5, 5, 5, 5, 0, -5,
	-10, 5, 5, 5, 5, 5, 0, -10,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var kingMiddlegamePST = [64]int{
	20, 30, 10, 0, 0, 10, 30, 20,
	20, 20, 0, 0, 0, 0, 20, 20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
}

var kingEndgamePST = [64]int{
	-50, -30, -30, -30, -30, -30, -30, -50,
	-30, -30, 0, 0, 0, 0, -30, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -20, -10, 0, 0, -10, -20, -30,
	-50, -40, -30, -20, -20, -30, -40, -50,
}

// pstMG and pstEG hold middlegame and endgame PST tables.
// All pieces except the king share the same tables in both phases.
var pstMG = [7]*[64]int{
	nil,                 // NoPiece
	&pawnPST,            // Pawn
	&knightPST,          // Knight
	&bishopPST,          // Bishop
	&rookPST,            // Rook
	&queenPST,           // Queen
	&kingMiddlegamePST,  // King
}

var pstEG = [7]*[64]int{
	nil,              // NoPiece
	&pawnPST,         // Pawn
	&knightPST,       // Knight
	&bishopPST,       // Bishop
	&rookPST,         // Rook
	&queenPST,        // Queen
	&kingEndgamePST,  // King
}

// pstTable kept as alias for backward compatibility.
var pstTable = pstMG

// MirrorSquare flips the rank for Black PST lookups.
func MirrorSquare(sq board.Square) board.Square {
	return sq ^ 56
}

// pstBalanceTapered computes separate middlegame and endgame PST scores
// from White's perspective.
func pstBalanceTapered(pos *board.Position) (mg, eg int) {
	for piece := board.Pawn; piece <= board.King; piece++ {
		mgTable := pstMG[piece]
		egTable := pstEG[piece]
		if mgTable == nil {
			continue
		}
		// White pieces — use tables directly.
		bb := pos.ColorPieces(board.White, piece)
		for bb != 0 {
			sq := bb.PopLSB()
			mg += mgTable[sq]
			eg += egTable[sq]
		}
		// Black pieces — mirror square.
		bb = pos.ColorPieces(board.Black, piece)
		for bb != 0 {
			sq := bb.PopLSB()
			mg -= mgTable[MirrorSquare(sq)]
			eg -= egTable[MirrorSquare(sq)]
		}
	}
	return mg, eg
}
