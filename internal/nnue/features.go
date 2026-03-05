package nnue

import "checkmatego/internal/board"

// Network architecture constants.
const (
	// InputSize is the number of input features per perspective.
	// 2 (relative color) × 6 (piece types Pawn..King) × 64 (squares) = 768
	InputSize = 768

	// HiddenSize is the number of neurons in the feature transformer output.
	HiddenSize = 256

	// L2Size is the number of neurons in the hidden layer.
	L2Size = 32

	// QA is the quantization/clamp range for the feature transformer output (ClippedReLU).
	QA = 255

	// QB is the quantization/clamp range for the hidden layer output (ClippedReLU).
	QB = 64

	// OutputScale converts the raw network output to centipawns.
	OutputScale = 400
)

// FeatureIndex computes the input feature index for a piece on a given square,
// from a given perspective.
//
// Index = relativeColor * 384 + (pieceType - 1) * 64 + mappedSquare
//
// relativeColor: 0 = friendly, 1 = enemy (pieceColor XOR perspective)
// mappedSquare: square from perspective's viewpoint (flipped for Black)
func FeatureIndex(perspective, pieceColor board.Color, pieceType board.Piece, sq board.Square) int {
	relColor := int(pieceColor ^ perspective)
	mappedSq := int(sq)
	if perspective == board.Black {
		mappedSq = int(sq) ^ 56
	}
	return relColor*384 + (int(pieceType)-1)*64 + mappedSq
}
