package eval

import "checkmatego/internal/board"

// Piece values in centipawns.
var PieceValue = [7]int{
	0,     // NoPiece
	100,   // Pawn
	320,   // Knight
	330,   // Bishop
	500,   // Rook
	900,   // Queen
	20000, // King
}

// Bishop pair bonus in centipawns.
const bishopPairBonus = 30

// materialBalance returns material score from White's perspective.
func materialBalance(pos *board.Position) int {
	score := 0
	for piece := board.Pawn; piece <= board.Queen; piece++ {
		white := pos.ColorPieces(board.White, piece).Count()
		black := pos.ColorPieces(board.Black, piece).Count()
		score += PieceValue[piece] * (white - black)
	}

	// Bishop pair bonus.
	if pos.ColorPieces(board.White, board.Bishop).Count() >= 2 {
		score += bishopPairBonus
	}
	if pos.ColorPieces(board.Black, board.Bishop).Count() >= 2 {
		score -= bishopPairBonus
	}

	return score
}
