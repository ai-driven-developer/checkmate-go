package board

import "fmt"

// Square represents a board square 0-63 using LERF mapping (a1=0, h8=63).
type Square uint8

const (
	A1 Square = iota
	B1
	C1
	D1
	E1
	F1
	G1
	H1
	A2
	B2
	C2
	D2
	E2
	F2
	G2
	H2
	A3
	B3
	C3
	D3
	E3
	F3
	G3
	H3
	A4
	B4
	C4
	D4
	E4
	F4
	G4
	H4
	A5
	B5
	C5
	D5
	E5
	F5
	G5
	H5
	A6
	B6
	C6
	D6
	E6
	F6
	G6
	H6
	A7
	B7
	C7
	D7
	E7
	F7
	G7
	H7
	A8
	B8
	C8
	D8
	E8
	F8
	G8
	H8
	NoSquare Square = 64
)

func (s Square) Rank() int         { return int(s) / 8 }
func (s Square) File() int         { return int(s) % 8 }
func NewSquare(file, rank int) Square { return Square(rank*8 + file) }

func (s Square) String() string {
	if s >= NoSquare {
		return "-"
	}
	return fmt.Sprintf("%c%c", 'a'+rune(s.File()), '1'+rune(s.Rank()))
}

// SquareFromString parses a square name like "e4" into a Square.
func SquareFromString(s string) Square {
	if len(s) != 2 {
		return NoSquare
	}
	file := int(s[0] - 'a')
	rank := int(s[1] - '1')
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return NoSquare
	}
	return NewSquare(file, rank)
}

// Piece represents a piece type (color-independent).
type Piece uint8

const (
	NoPiece Piece = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

var pieceChars = [7]byte{'.', 'P', 'N', 'B', 'R', 'Q', 'K'}

func (p Piece) Char() byte { return pieceChars[p] }

// Color represents White or Black.
type Color uint8

const (
	White Color = 0
	Black Color = 1
)

func (c Color) Other() Color { return c ^ 1 }

// CastlingRights is a bitmask for castling availability.
type CastlingRights uint8

const (
	WhiteKingSide  CastlingRights = 1 << iota // 0001
	WhiteQueenSide                             // 0010
	BlackKingSide                              // 0100
	BlackQueenSide                             // 1000
	NoCastling     CastlingRights = 0
	AllCastling    CastlingRights = WhiteKingSide | WhiteQueenSide | BlackKingSide | BlackQueenSide
)

// castlingMask is used to update castling rights when a piece moves from/to a square.
// Moving from or to the relevant corner/king square clears the corresponding right.
var CastlingMask [64]CastlingRights

func init() {
	for i := range CastlingMask {
		CastlingMask[i] = AllCastling
	}
	CastlingMask[A1] = AllCastling ^ WhiteQueenSide
	CastlingMask[E1] = AllCastling ^ (WhiteKingSide | WhiteQueenSide)
	CastlingMask[H1] = AllCastling ^ WhiteKingSide
	CastlingMask[A8] = AllCastling ^ BlackQueenSide
	CastlingMask[E8] = AllCastling ^ (BlackKingSide | BlackQueenSide)
	CastlingMask[H8] = AllCastling ^ BlackKingSide
}
