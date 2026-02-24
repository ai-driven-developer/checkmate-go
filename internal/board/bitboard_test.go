package board

import "testing"

func TestBitboardSetHasClear(t *testing.T) {
	var bb Bitboard
	bb.Set(E4)
	if !bb.Has(E4) {
		t.Error("expected E4 to be set")
	}
	if bb.Has(E5) {
		t.Error("expected E5 to not be set")
	}
	bb.Clear(E4)
	if bb.Has(E4) {
		t.Error("expected E4 to be cleared")
	}
}

func TestBitboardCount(t *testing.T) {
	bb := SquareBB(A1) | SquareBB(H8) | SquareBB(D4)
	if bb.Count() != 3 {
		t.Errorf("expected count 3, got %d", bb.Count())
	}
	if Bitboard(0).Count() != 0 {
		t.Error("empty bitboard should have count 0")
	}
}

func TestBitboardLSBPopLSB(t *testing.T) {
	bb := SquareBB(C3) | SquareBB(F6)
	lsb := bb.LSB()
	if lsb != C3 {
		t.Errorf("expected LSB C3, got %s", lsb)
	}
	sq := bb.PopLSB()
	if sq != C3 {
		t.Errorf("expected PopLSB C3, got %s", sq)
	}
	if bb.Count() != 1 {
		t.Errorf("expected count 1 after pop, got %d", bb.Count())
	}
	sq = bb.PopLSB()
	if sq != F6 {
		t.Errorf("expected PopLSB F6, got %s", sq)
	}
	if bb != 0 {
		t.Error("expected empty bitboard after popping all")
	}
}

func TestBitboardShifts(t *testing.T) {
	bb := SquareBB(E4)

	if !bb.North().Has(E5) {
		t.Error("North of E4 should be E5")
	}
	if !bb.South().Has(E3) {
		t.Error("South of E4 should be E3")
	}
	if !bb.East().Has(F4) {
		t.Error("East of E4 should be F4")
	}
	if !bb.West().Has(D4) {
		t.Error("West of E4 should be D4")
	}
	if !bb.NorthEast().Has(F5) {
		t.Error("NorthEast of E4 should be F5")
	}
	if !bb.NorthWest().Has(D5) {
		t.Error("NorthWest of E4 should be D5")
	}
	if !bb.SouthEast().Has(F3) {
		t.Error("SouthEast of E4 should be F3")
	}
	if !bb.SouthWest().Has(D3) {
		t.Error("SouthWest of E4 should be D3")
	}
}

func TestBitboardEdgeShifts(t *testing.T) {
	// East from H-file should not wrap to A-file.
	bb := SquareBB(H4)
	if bb.East() != 0 {
		t.Error("East from H-file should be empty")
	}

	// West from A-file should not wrap to H-file.
	bb = SquareBB(A4)
	if bb.West() != 0 {
		t.Error("West from A-file should be empty")
	}

	// North from rank 8 should be empty.
	bb = SquareBB(E8)
	if bb.North() != 0 {
		t.Error("North from rank 8 should be empty")
	}

	// South from rank 1 should be empty.
	bb = SquareBB(E1)
	if bb.South() != 0 {
		t.Error("South from rank 1 should be empty")
	}
}

func TestFileMasks(t *testing.T) {
	if FileABB.Count() != 8 {
		t.Errorf("FileA should have 8 bits, got %d", FileABB.Count())
	}
	if !FileABB.Has(A1) || !FileABB.Has(A8) {
		t.Error("FileA should contain A1 and A8")
	}
	if FileABB.Has(B1) {
		t.Error("FileA should not contain B1")
	}
}

func TestRankMasks(t *testing.T) {
	if Rank1BB.Count() != 8 {
		t.Errorf("Rank1 should have 8 bits, got %d", Rank1BB.Count())
	}
	if !Rank1BB.Has(A1) || !Rank1BB.Has(H1) {
		t.Error("Rank1 should contain A1 and H1")
	}
	if Rank1BB.Has(A2) {
		t.Error("Rank1 should not contain A2")
	}
}
