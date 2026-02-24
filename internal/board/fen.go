package board

import (
	"fmt"
	"strconv"
	"strings"
)

const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

var fenPieceMap = map[byte](struct {
	piece Piece
	color Color
}){
	'P': {Pawn, White}, 'N': {Knight, White}, 'B': {Bishop, White},
	'R': {Rook, White}, 'Q': {Queen, White}, 'K': {King, White},
	'p': {Pawn, Black}, 'n': {Knight, Black}, 'b': {Bishop, Black},
	'r': {Rook, Black}, 'q': {Queen, Black}, 'k': {King, Black},
}

// SetFromFEN parses a FEN string and sets the position.
func (p *Position) SetFromFEN(fen string) error {
	// Reset.
	*p = Position{EnPassant: NoSquare}

	parts := strings.Fields(fen)
	if len(parts) < 4 {
		return fmt.Errorf("invalid FEN: expected at least 4 fields, got %d", len(parts))
	}

	// 1. Piece placement.
	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return fmt.Errorf("invalid FEN: expected 8 ranks, got %d", len(ranks))
	}
	for i, rankStr := range ranks {
		rank := 7 - i // FEN starts from rank 8.
		file := 0
		for _, ch := range rankStr {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
			} else if info, ok := fenPieceMap[byte(ch)]; ok {
				sq := NewSquare(file, rank)
				p.putPiece(info.color, info.piece, sq)
				file++
			} else {
				return fmt.Errorf("invalid FEN: unexpected character '%c'", ch)
			}
		}
		if file != 8 {
			return fmt.Errorf("invalid FEN: rank %d has %d files", rank+1, file)
		}
	}

	// 2. Side to move.
	switch parts[1] {
	case "w":
		p.SideToMove = White
	case "b":
		p.SideToMove = Black
	default:
		return fmt.Errorf("invalid FEN: side to move '%s'", parts[1])
	}

	// 3. Castling rights.
	if parts[2] != "-" {
		for _, ch := range parts[2] {
			switch ch {
			case 'K':
				p.Castling |= WhiteKingSide
			case 'Q':
				p.Castling |= WhiteQueenSide
			case 'k':
				p.Castling |= BlackKingSide
			case 'q':
				p.Castling |= BlackQueenSide
			}
		}
	}

	// 4. En passant.
	if parts[3] != "-" {
		p.EnPassant = SquareFromString(parts[3])
		if p.EnPassant == NoSquare {
			return fmt.Errorf("invalid FEN: en passant square '%s'", parts[3])
		}
	}

	// 5. Half-move clock (optional).
	if len(parts) > 4 {
		hmc, err := strconv.Atoi(parts[4])
		if err != nil {
			return fmt.Errorf("invalid FEN: half-move clock '%s'", parts[4])
		}
		p.HalfMoveClock = uint8(hmc)
	}

	// 6. Full-move number (optional).
	if len(parts) > 5 {
		fmn, err := strconv.Atoi(parts[5])
		if err != nil {
			return fmt.Errorf("invalid FEN: full-move number '%s'", parts[5])
		}
		p.FullMoveNumber = uint16(fmn)
	} else {
		p.FullMoveNumber = 1
	}

	p.Hash = p.computeHash()
	return nil
}

// FEN returns the FEN string for the current position.
func (p *Position) FEN() string {
	var sb strings.Builder

	// 1. Piece placement.
	for rank := 7; rank >= 0; rank-- {
		empty := 0
		for file := 0; file < 8; file++ {
			sq := NewSquare(file, rank)
			piece, color := p.PieceAt(sq)
			if piece == NoPiece {
				empty++
			} else {
				if empty > 0 {
					sb.WriteByte(byte('0' + empty))
					empty = 0
				}
				ch := piece.Char()
				if color == Black {
					ch = ch - 'A' + 'a'
				}
				sb.WriteByte(ch)
			}
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if rank > 0 {
			sb.WriteByte('/')
		}
	}

	// 2. Side to move.
	if p.SideToMove == White {
		sb.WriteString(" w ")
	} else {
		sb.WriteString(" b ")
	}

	// 3. Castling rights.
	if p.Castling == NoCastling {
		sb.WriteByte('-')
	} else {
		if p.Castling&WhiteKingSide != 0 {
			sb.WriteByte('K')
		}
		if p.Castling&WhiteQueenSide != 0 {
			sb.WriteByte('Q')
		}
		if p.Castling&BlackKingSide != 0 {
			sb.WriteByte('k')
		}
		if p.Castling&BlackQueenSide != 0 {
			sb.WriteByte('q')
		}
	}

	// 4. En passant.
	sb.WriteByte(' ')
	sb.WriteString(p.EnPassant.String())

	// 5-6. Clocks.
	sb.WriteString(fmt.Sprintf(" %d %d", p.HalfMoveClock, p.FullMoveNumber))

	return sb.String()
}
