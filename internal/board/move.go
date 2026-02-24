package board

import "fmt"

// Move packs move information into a uint32.
// Bit layout:
//
//	bits  0-5:  from square (0-63)
//	bits  6-11: to square (0-63)
//	bits 12-15: flags
//	bits 16-19: moving piece type
//	bits 20-23: captured piece type
type Move uint32

// MoveFlag constants.
const (
	FlagQuiet       uint32 = 0
	FlagDoublePawn  uint32 = 1
	FlagKingCastle  uint32 = 2
	FlagQueenCastle uint32 = 3
	FlagCapture     uint32 = 4
	FlagEnPassant   uint32 = 5
	FlagPromoKnight uint32 = 8
	FlagPromoBishop uint32 = 9
	FlagPromoRook   uint32 = 10
	FlagPromoQueen  uint32 = 11
	FlagPromoCaptureKnight uint32 = 12
	FlagPromoCaptureBishop uint32 = 13
	FlagPromoCaptureRook   uint32 = 14
	FlagPromoCaptureQueen  uint32 = 15
)

const NullMove Move = 0

func NewMove(from, to Square, flags uint32, piece, captured Piece) Move {
	return Move(uint32(from) | uint32(to)<<6 | flags<<12 | uint32(piece)<<16 | uint32(captured)<<20)
}

func (m Move) From() Square      { return Square(m & 0x3F) }
func (m Move) To() Square        { return Square((m >> 6) & 0x3F) }
func (m Move) Flags() uint32     { return uint32((m >> 12) & 0xF) }
func (m Move) Piece() Piece      { return Piece((m >> 16) & 0xF) }
func (m Move) CapturedPiece() Piece { return Piece((m >> 20) & 0xF) }

func (m Move) IsCapture() bool {
	f := m.Flags()
	return f == FlagCapture || f == FlagEnPassant || f >= FlagPromoCaptureKnight
}

func (m Move) IsPromotion() bool {
	return m.Flags() >= FlagPromoKnight
}

func (m Move) PromotionPiece() Piece {
	switch m.Flags() {
	case FlagPromoKnight, FlagPromoCaptureKnight:
		return Knight
	case FlagPromoBishop, FlagPromoCaptureBishop:
		return Bishop
	case FlagPromoRook, FlagPromoCaptureRook:
		return Rook
	case FlagPromoQueen, FlagPromoCaptureQueen:
		return Queen
	}
	return NoPiece
}

func (m Move) IsCastle() bool {
	f := m.Flags()
	return f == FlagKingCastle || f == FlagQueenCastle
}

// String returns UCI long algebraic notation (e.g. "e2e4", "e7e8q").
func (m Move) String() string {
	if m == NullMove {
		return "0000"
	}
	s := fmt.Sprintf("%s%s", m.From(), m.To())
	if m.IsPromotion() {
		promoChar := map[Piece]byte{Knight: 'n', Bishop: 'b', Rook: 'r', Queen: 'q'}
		s += string(promoChar[m.PromotionPiece()])
	}
	return s
}

// MoveList is a stack-allocated move buffer.
type MoveList struct {
	Moves [256]Move
	Count int
}

func (ml *MoveList) Add(m Move) {
	ml.Moves[ml.Count] = m
	ml.Count++
}

func (ml *MoveList) Clear() { ml.Count = 0 }
