package board

import (
	"math/bits"
	"strings"
)

// Bitboard is a 64-bit set of squares.
type Bitboard uint64

// File masks.
const (
	FileABB Bitboard = 0x0101010101010101
	FileBBB Bitboard = FileABB << 1
	FileCBB Bitboard = FileABB << 2
	FileDBB Bitboard = FileABB << 3
	FileEBB Bitboard = FileABB << 4
	FileFBB Bitboard = FileABB << 5
	FileGBB Bitboard = FileABB << 6
	FileHBB Bitboard = FileABB << 7
)

// Rank masks.
const (
	Rank1BB Bitboard = 0x00000000000000FF
	Rank2BB Bitboard = Rank1BB << 8
	Rank3BB Bitboard = Rank1BB << 16
	Rank4BB Bitboard = Rank1BB << 24
	Rank5BB Bitboard = Rank1BB << 32
	Rank6BB Bitboard = Rank1BB << 40
	Rank7BB Bitboard = Rank1BB << 48
	Rank8BB Bitboard = Rank1BB << 56
)

func (b Bitboard) Has(sq Square) bool { return b&(1<<sq) != 0 }
func (b *Bitboard) Set(sq Square)     { *b |= 1 << sq }
func (b *Bitboard) Clear(sq Square)   { *b &^= 1 << sq }
func (b Bitboard) Count() int         { return bits.OnesCount64(uint64(b)) }

func (b Bitboard) LSB() Square {
	return Square(bits.TrailingZeros64(uint64(b)))
}

func (b *Bitboard) PopLSB() Square {
	sq := b.LSB()
	*b &= *b - 1
	return sq
}

func SquareBB(sq Square) Bitboard { return 1 << sq }

// Shift operations (edge-aware).
func (b Bitboard) North() Bitboard     { return b << 8 }
func (b Bitboard) South() Bitboard     { return b >> 8 }
func (b Bitboard) East() Bitboard      { return (b &^ FileHBB) << 1 }
func (b Bitboard) West() Bitboard      { return (b &^ FileABB) >> 1 }
func (b Bitboard) NorthEast() Bitboard { return (b &^ FileHBB) << 9 }
func (b Bitboard) NorthWest() Bitboard { return (b &^ FileABB) << 7 }
func (b Bitboard) SouthEast() Bitboard { return (b &^ FileHBB) >> 7 }
func (b Bitboard) SouthWest() Bitboard { return (b &^ FileABB) >> 9 }

// String returns a visual representation of the bitboard (for debugging).
func (b Bitboard) String() string {
	var sb strings.Builder
	for rank := 7; rank >= 0; rank-- {
		for file := 0; file < 8; file++ {
			sq := NewSquare(file, rank)
			if b.Has(sq) {
				sb.WriteByte('1')
			} else {
				sb.WriteByte('.')
			}
			if file < 7 {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
