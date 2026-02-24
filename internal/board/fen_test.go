package board

import "testing"

func TestFENStartPosition(t *testing.T) {
	p := NewPosition()

	// Check piece counts.
	if p.Pieces[White][Pawn].Count() != 8 {
		t.Errorf("expected 8 white pawns, got %d", p.Pieces[White][Pawn].Count())
	}
	if p.Pieces[Black][Pawn].Count() != 8 {
		t.Errorf("expected 8 black pawns, got %d", p.Pieces[Black][Pawn].Count())
	}
	if p.Pieces[White][Rook].Count() != 2 {
		t.Errorf("expected 2 white rooks, got %d", p.Pieces[White][Rook].Count())
	}
	if p.Pieces[White][King].Count() != 1 {
		t.Errorf("expected 1 white king, got %d", p.Pieces[White][King].Count())
	}
	if p.Occupied[White].Count() != 16 {
		t.Errorf("expected 16 white pieces, got %d", p.Occupied[White].Count())
	}
	if p.AllOccupied.Count() != 32 {
		t.Errorf("expected 32 total pieces, got %d", p.AllOccupied.Count())
	}

	// Check specific squares.
	piece, color := p.PieceAt(E1)
	if piece != King || color != White {
		t.Errorf("expected white king on e1, got piece=%d color=%d", piece, color)
	}
	piece, color = p.PieceAt(D8)
	if piece != Queen || color != Black {
		t.Errorf("expected black queen on d8, got piece=%d color=%d", piece, color)
	}
	piece, _ = p.PieceAt(E4)
	if piece != NoPiece {
		t.Errorf("expected empty square e4, got piece=%d", piece)
	}

	// Game state.
	if p.SideToMove != White {
		t.Error("expected white to move")
	}
	if p.Castling != AllCastling {
		t.Errorf("expected all castling rights, got %d", p.Castling)
	}
	if p.EnPassant != NoSquare {
		t.Errorf("expected no en passant, got %s", p.EnPassant)
	}
	if p.HalfMoveClock != 0 {
		t.Errorf("expected half-move clock 0, got %d", p.HalfMoveClock)
	}
	if p.FullMoveNumber != 1 {
		t.Errorf("expected full-move 1, got %d", p.FullMoveNumber)
	}
}

func TestFENRoundTrip(t *testing.T) {
	fens := []string{
		StartFEN,
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 2",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/8/8/8/8/8/8/4K2k w - - 0 1",
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	}

	for _, fen := range fens {
		p := &Position{}
		if err := p.SetFromFEN(fen); err != nil {
			t.Errorf("failed to parse FEN '%s': %v", fen, err)
			continue
		}
		got := p.FEN()
		if got != fen {
			t.Errorf("FEN round-trip failed:\n  input:  %s\n  output: %s", fen, got)
		}
	}
}

func TestFENInvalid(t *testing.T) {
	invalid := []string{
		"",
		"invalid",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP w KQkq - 0 1", // 7 ranks
	}
	for _, fen := range invalid {
		p := &Position{}
		if err := p.SetFromFEN(fen); err == nil {
			t.Errorf("expected error for FEN '%s'", fen)
		}
	}
}

func TestFENMailboxConsistency(t *testing.T) {
	p := NewPosition()

	for sq := A1; sq <= H8; sq++ {
		piece := p.PieceOn[sq]
		if piece != NoPiece {
			// Mailbox says there's a piece — check bitboards agree.
			if !p.AllOccupied.Has(sq) {
				t.Errorf("mailbox has piece on %s but AllOccupied doesn't", sq)
			}
			_, color := p.PieceAt(sq)
			if !p.Pieces[color][piece].Has(sq) {
				t.Errorf("mailbox has %d on %s but Pieces[%d][%d] doesn't", piece, sq, color, piece)
			}
		} else {
			if p.AllOccupied.Has(sq) {
				t.Errorf("mailbox has no piece on %s but AllOccupied does", sq)
			}
		}
	}
}
