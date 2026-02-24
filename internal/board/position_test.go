package board

import "testing"

func TestKingSquare(t *testing.T) {
	p := NewPosition()
	if p.KingSquare(White) != E1 {
		t.Errorf("expected white king on E1, got %s", p.KingSquare(White))
	}
	if p.KingSquare(Black) != E8 {
		t.Errorf("expected black king on E8, got %s", p.KingSquare(Black))
	}
}

func TestMakeUnmakeQuietMove(t *testing.T) {
	p := NewPosition()
	origFEN := p.FEN()
	origHash := p.Hash

	m := NewMove(E2, E4, FlagDoublePawn, Pawn, NoPiece)
	p.MakeMove(m)

	// After e2e4: pawn should be on e4, not e2.
	piece, _ := p.PieceAt(E4)
	if piece != Pawn {
		t.Errorf("expected pawn on E4 after move, got %d", piece)
	}
	piece, _ = p.PieceAt(E2)
	if piece != NoPiece {
		t.Errorf("expected empty E2 after move, got %d", piece)
	}
	if p.SideToMove != Black {
		t.Error("expected black to move after e2e4")
	}
	if p.EnPassant != E3 {
		t.Errorf("expected en passant on E3, got %s", p.EnPassant)
	}

	p.UnmakeMove(m)

	// Position should be fully restored.
	if p.FEN() != origFEN {
		t.Errorf("FEN not restored after unmake:\n  got:  %s\n  want: %s", p.FEN(), origFEN)
	}
	if p.Hash != origHash {
		t.Errorf("hash not restored after unmake: got %x, want %x", p.Hash, origHash)
	}
}

func TestMakeUnmakeCapture(t *testing.T) {
	p := &Position{}
	_ = p.SetFromFEN("rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2")
	origFEN := p.FEN()
	origHash := p.Hash

	// e4xd5
	m := NewMove(E4, D5, FlagCapture, Pawn, Pawn)
	p.MakeMove(m)

	piece, color := p.PieceAt(D5)
	if piece != Pawn || color != White {
		t.Errorf("expected white pawn on D5, got piece=%d color=%d", piece, color)
	}
	piece, _ = p.PieceAt(E4)
	if piece != NoPiece {
		t.Errorf("expected empty E4 after capture, got %d", piece)
	}
	// Black should have one fewer pawn.
	if p.Pieces[Black][Pawn].Count() != 7 {
		t.Errorf("expected 7 black pawns, got %d", p.Pieces[Black][Pawn].Count())
	}

	p.UnmakeMove(m)

	if p.FEN() != origFEN {
		t.Errorf("FEN not restored after unmake capture:\n  got:  %s\n  want: %s", p.FEN(), origFEN)
	}
	if p.Hash != origHash {
		t.Error("hash not restored after unmake capture")
	}
}

func TestMakeUnmakeCastling(t *testing.T) {
	p := &Position{}
	_ = p.SetFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
	origFEN := p.FEN()
	origHash := p.Hash

	// White kingside castle.
	m := NewMove(E1, G1, FlagKingCastle, King, NoPiece)
	p.MakeMove(m)

	piece, color := p.PieceAt(G1)
	if piece != King || color != White {
		t.Error("expected white king on G1")
	}
	piece, color = p.PieceAt(F1)
	if piece != Rook || color != White {
		t.Error("expected white rook on F1")
	}
	piece, _ = p.PieceAt(E1)
	if piece != NoPiece {
		t.Error("expected empty E1")
	}
	piece, _ = p.PieceAt(H1)
	if piece != NoPiece {
		t.Error("expected empty H1")
	}

	p.UnmakeMove(m)
	if p.FEN() != origFEN {
		t.Errorf("FEN not restored after unmake castle:\n  got:  %s\n  want: %s", p.FEN(), origFEN)
	}
	if p.Hash != origHash {
		t.Error("hash not restored after unmake castle")
	}
}

func TestMakeUnmakeEnPassant(t *testing.T) {
	p := &Position{}
	_ = p.SetFromFEN("rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3")
	origFEN := p.FEN()
	origHash := p.Hash

	// f5xe6 en passant.
	m := NewMove(F5, E6, FlagEnPassant, Pawn, Pawn)
	p.MakeMove(m)

	piece, color := p.PieceAt(E6)
	if piece != Pawn || color != White {
		t.Error("expected white pawn on E6")
	}
	// The captured pawn was on E5.
	piece, _ = p.PieceAt(E5)
	if piece != NoPiece {
		t.Errorf("expected empty E5 after en passant, got %d", piece)
	}

	p.UnmakeMove(m)
	if p.FEN() != origFEN {
		t.Errorf("FEN not restored after unmake en passant:\n  got:  %s\n  want: %s", p.FEN(), origFEN)
	}
	if p.Hash != origHash {
		t.Error("hash not restored after unmake en passant")
	}
}

func TestMakeUnmakePromotion(t *testing.T) {
	p := &Position{}
	_ = p.SetFromFEN("8/4P3/8/8/8/8/8/4K2k w - - 0 1")
	origFEN := p.FEN()
	origHash := p.Hash

	m := NewMove(E7, E8, FlagPromoQueen, Pawn, NoPiece)
	p.MakeMove(m)

	piece, color := p.PieceAt(E8)
	if piece != Queen || color != White {
		t.Errorf("expected white queen on E8, got piece=%d color=%d", piece, color)
	}
	if p.Pieces[White][Pawn].Has(E7) {
		t.Error("expected pawn removed from E7")
	}

	p.UnmakeMove(m)
	if p.FEN() != origFEN {
		t.Errorf("FEN not restored after unmake promotion:\n  got:  %s\n  want: %s", p.FEN(), origFEN)
	}
	if p.Hash != origHash {
		t.Error("hash not restored after unmake promotion")
	}
}

func TestZobristHashConsistency(t *testing.T) {
	p := NewPosition()

	// After a sequence of moves and unmakes, the hash should be incrementally consistent.
	m1 := NewMove(E2, E4, FlagDoublePawn, Pawn, NoPiece)
	p.MakeMove(m1)
	h1 := p.Hash

	// Hash should match recomputed hash.
	if h1 != p.computeHash() {
		t.Error("incremental hash doesn't match recomputed hash after e2e4")
	}

	m2 := NewMove(E7, E5, FlagDoublePawn, Pawn, NoPiece)
	p.MakeMove(m2)
	if p.Hash != p.computeHash() {
		t.Error("incremental hash doesn't match recomputed hash after e7e5")
	}

	p.UnmakeMove(m2)
	if p.Hash != h1 {
		t.Error("hash not restored after unmake e7e5")
	}
}
