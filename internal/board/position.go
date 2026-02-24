package board

import "fmt"

// Position holds the complete game state.
type Position struct {
	Pieces      [2][7]Bitboard // [color][piece]
	Occupied    [2]Bitboard    // all pieces per color
	AllOccupied Bitboard       // union

	PieceOn [64]Piece // mailbox: what piece type is on each square

	SideToMove     Color
	Castling       CastlingRights
	EnPassant      Square // NoSquare if none
	HalfMoveClock  uint8
	FullMoveNumber uint16

	Hash uint64

	stateHistory []stateInfo
}

// stateInfo stores irreversible state for UnmakeMove.
type stateInfo struct {
	Castling      CastlingRights
	EnPassant     Square
	HalfMoveClock uint8
	CapturedPiece Piece
	Hash          uint64
}

// NewPosition returns the starting chess position.
func NewPosition() *Position {
	p := &Position{}
	_ = p.SetFromFEN(StartFEN)
	return p
}

// putPiece places a piece on the board (no validation).
func (p *Position) putPiece(color Color, piece Piece, sq Square) {
	p.Pieces[color][piece].Set(sq)
	p.Occupied[color].Set(sq)
	p.AllOccupied.Set(sq)
	p.PieceOn[sq] = piece
}

// removePiece removes a piece from the board (no validation).
func (p *Position) removePiece(color Color, piece Piece, sq Square) {
	p.Pieces[color][piece].Clear(sq)
	p.Occupied[color].Clear(sq)
	p.AllOccupied.Clear(sq)
	p.PieceOn[sq] = NoPiece
}

// movePiece moves a piece (assumes no capture).
func (p *Position) movePiece(color Color, piece Piece, from, to Square) {
	bb := SquareBB(from) | SquareBB(to)
	p.Pieces[color][piece] ^= bb
	p.Occupied[color] ^= bb
	p.AllOccupied ^= bb
	p.PieceOn[from] = NoPiece
	p.PieceOn[to] = piece
}

// PieceAt returns the piece type and color on a square.
// Returns (NoPiece, White) if the square is empty.
func (p *Position) PieceAt(sq Square) (Piece, Color) {
	piece := p.PieceOn[sq]
	if piece == NoPiece {
		return NoPiece, White
	}
	if p.Occupied[Black].Has(sq) {
		return piece, Black
	}
	return piece, White
}

// ColorPieces returns the bitboard of a specific piece of a specific color.
func (p *Position) ColorPieces(c Color, piece Piece) Bitboard {
	return p.Pieces[c][piece]
}

// KingSquare returns the square of the king for a given color.
func (p *Position) KingSquare(c Color) Square {
	return p.Pieces[c][King].LSB()
}

// computeHash computes the Zobrist hash from scratch.
func (p *Position) computeHash() uint64 {
	var h uint64
	for color := Color(0); color <= 1; color++ {
		for piece := Pawn; piece <= King; piece++ {
			bb := p.Pieces[color][piece]
			for bb != 0 {
				sq := bb.PopLSB()
				h ^= ZobristPiece[color][piece][sq]
			}
		}
	}
	h ^= ZobristCastling[p.Castling]
	h ^= ZobristEnPassant[p.EnPassant]
	if p.SideToMove == Black {
		h ^= ZobristSideToMove
	}
	return h
}

// recomputeOccupied rebuilds aggregate bitboards from piece bitboards.
func (p *Position) recomputeOccupied() {
	p.Occupied[White] = 0
	p.Occupied[Black] = 0
	for piece := Pawn; piece <= King; piece++ {
		p.Occupied[White] |= p.Pieces[White][piece]
		p.Occupied[Black] |= p.Pieces[Black][piece]
	}
	p.AllOccupied = p.Occupied[White] | p.Occupied[Black]
}

// MakeMove applies a move to the position. Pushes state for UnmakeMove.
func (p *Position) MakeMove(m Move) {
	// Save irreversible state.
	p.stateHistory = append(p.stateHistory, stateInfo{
		Castling:      p.Castling,
		EnPassant:     p.EnPassant,
		HalfMoveClock: p.HalfMoveClock,
		CapturedPiece: m.CapturedPiece(),
		Hash:          p.Hash,
	})

	from := m.From()
	to := m.To()
	us := p.SideToMove
	them := us.Other()
	piece := m.Piece()
	captured := m.CapturedPiece()
	flags := m.Flags()

	// Update hash: remove old en passant and castling.
	p.Hash ^= ZobristEnPassant[p.EnPassant]
	p.Hash ^= ZobristCastling[p.Castling]

	// Reset en passant.
	p.EnPassant = NoSquare

	// Increment clocks.
	p.HalfMoveClock++
	if piece == Pawn || captured != NoPiece {
		p.HalfMoveClock = 0
	}
	if us == Black {
		p.FullMoveNumber++
	}

	switch flags {
	case FlagQuiet:
		p.movePiece(us, piece, from, to)
		p.Hash ^= ZobristPiece[us][piece][from] ^ ZobristPiece[us][piece][to]

	case FlagDoublePawn:
		p.movePiece(us, Pawn, from, to)
		p.Hash ^= ZobristPiece[us][Pawn][from] ^ ZobristPiece[us][Pawn][to]
		// Set en passant square (the square behind the pawn).
		if us == White {
			p.EnPassant = from + 8
		} else {
			p.EnPassant = from - 8
		}

	case FlagCapture:
		p.removePiece(them, captured, to)
		p.Hash ^= ZobristPiece[them][captured][to]
		p.movePiece(us, piece, from, to)
		p.Hash ^= ZobristPiece[us][piece][from] ^ ZobristPiece[us][piece][to]

	case FlagEnPassant:
		// The captured pawn is not on 'to' but on the square behind it.
		var capturedSq Square
		if us == White {
			capturedSq = to - 8
		} else {
			capturedSq = to + 8
		}
		p.removePiece(them, Pawn, capturedSq)
		p.Hash ^= ZobristPiece[them][Pawn][capturedSq]
		p.movePiece(us, Pawn, from, to)
		p.Hash ^= ZobristPiece[us][Pawn][from] ^ ZobristPiece[us][Pawn][to]

	case FlagKingCastle:
		p.movePiece(us, King, from, to)
		p.Hash ^= ZobristPiece[us][King][from] ^ ZobristPiece[us][King][to]
		// Move the rook.
		var rookFrom, rookTo Square
		if us == White {
			rookFrom, rookTo = H1, F1
		} else {
			rookFrom, rookTo = H8, F8
		}
		p.movePiece(us, Rook, rookFrom, rookTo)
		p.Hash ^= ZobristPiece[us][Rook][rookFrom] ^ ZobristPiece[us][Rook][rookTo]

	case FlagQueenCastle:
		p.movePiece(us, King, from, to)
		p.Hash ^= ZobristPiece[us][King][from] ^ ZobristPiece[us][King][to]
		var rookFrom, rookTo Square
		if us == White {
			rookFrom, rookTo = A1, D1
		} else {
			rookFrom, rookTo = A8, D8
		}
		p.movePiece(us, Rook, rookFrom, rookTo)
		p.Hash ^= ZobristPiece[us][Rook][rookFrom] ^ ZobristPiece[us][Rook][rookTo]

	default:
		// Promotions.
		promoPiece := m.PromotionPiece()
		if m.IsCapture() {
			p.removePiece(them, captured, to)
			p.Hash ^= ZobristPiece[them][captured][to]
		}
		// Remove pawn from origin.
		p.removePiece(us, Pawn, from)
		p.Hash ^= ZobristPiece[us][Pawn][from]
		// Place promoted piece at destination.
		p.putPiece(us, promoPiece, to)
		p.Hash ^= ZobristPiece[us][promoPiece][to]
	}

	// Update castling rights.
	p.Castling &= CastlingMask[from] & CastlingMask[to]

	// Update hash: add new en passant and castling.
	p.Hash ^= ZobristEnPassant[p.EnPassant]
	p.Hash ^= ZobristCastling[p.Castling]

	// Flip side to move.
	p.SideToMove = them
	p.Hash ^= ZobristSideToMove
}

// UnmakeMove restores the position to before the last MakeMove.
func (p *Position) UnmakeMove(m Move) {
	// Pop state.
	idx := len(p.stateHistory) - 1
	state := p.stateHistory[idx]
	p.stateHistory = p.stateHistory[:idx]

	// Flip side back.
	p.SideToMove = p.SideToMove.Other()
	us := p.SideToMove
	them := us.Other()

	from := m.From()
	to := m.To()
	piece := m.Piece()
	flags := m.Flags()

	switch flags {
	case FlagQuiet, FlagDoublePawn:
		p.movePiece(us, piece, to, from)

	case FlagCapture:
		p.movePiece(us, piece, to, from)
		p.putPiece(them, state.CapturedPiece, to)

	case FlagEnPassant:
		p.movePiece(us, Pawn, to, from)
		var capturedSq Square
		if us == White {
			capturedSq = to - 8
		} else {
			capturedSq = to + 8
		}
		p.putPiece(them, Pawn, capturedSq)

	case FlagKingCastle:
		p.movePiece(us, King, to, from)
		var rookFrom, rookTo Square
		if us == White {
			rookFrom, rookTo = H1, F1
		} else {
			rookFrom, rookTo = H8, F8
		}
		p.movePiece(us, Rook, rookTo, rookFrom)

	case FlagQueenCastle:
		p.movePiece(us, King, to, from)
		var rookFrom, rookTo Square
		if us == White {
			rookFrom, rookTo = A1, D1
		} else {
			rookFrom, rookTo = A8, D8
		}
		p.movePiece(us, Rook, rookTo, rookFrom)

	default:
		// Promotion unmake: remove promoted piece, put pawn back.
		promoPiece := m.PromotionPiece()
		p.removePiece(us, promoPiece, to)
		p.putPiece(us, Pawn, from)
		if m.IsCapture() {
			p.putPiece(them, state.CapturedPiece, to)
		}
	}

	// Restore irreversible state.
	p.Castling = state.Castling
	p.EnPassant = state.EnPassant
	p.HalfMoveClock = state.HalfMoveClock
	p.Hash = state.Hash

	if us == Black {
		p.FullMoveNumber--
	}
}

// Copy returns a deep copy of the position (for concurrent use).
func (p *Position) Copy() *Position {
	cp := *p
	cp.stateHistory = make([]stateInfo, len(p.stateHistory))
	copy(cp.stateHistory, p.stateHistory)
	return &cp
}

// String returns a human-readable board representation.
func (p *Position) String() string {
	var s string
	for rank := 7; rank >= 0; rank-- {
		s += fmt.Sprintf("%d ", rank+1)
		for file := 0; file < 8; file++ {
			sq := NewSquare(file, rank)
			piece, color := p.PieceAt(sq)
			if piece == NoPiece {
				s += ". "
			} else {
				ch := piece.Char()
				if color == Black {
					ch = ch - 'A' + 'a'
				}
				s += string(ch) + " "
			}
		}
		s += "\n"
	}
	s += "  a b c d e f g h\n"
	return s
}
