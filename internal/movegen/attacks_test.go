package movegen

import (
	"checkmatego/internal/board"
	"testing"
)

func TestKnightAttacks(t *testing.T) {
	// Knight on e4 attacks 8 squares: d2, f2, c3, g3, c5, g5, d6, f6.
	att := KnightAttacks[board.E4]
	if att.Count() != 8 {
		t.Errorf("knight on E4: expected 8 attacks, got %d", att.Count())
	}
	expected := []board.Square{board.D2, board.F2, board.C3, board.G3, board.C5, board.G5, board.D6, board.F6}
	for _, sq := range expected {
		if !att.Has(sq) {
			t.Errorf("knight on E4: expected attack on %s", sq)
		}
	}

	// Knight on a1 attacks 2 squares: b3, c2.
	att = KnightAttacks[board.A1]
	if att.Count() != 2 {
		t.Errorf("knight on A1: expected 2 attacks, got %d", att.Count())
	}
	if !att.Has(board.B3) || !att.Has(board.C2) {
		t.Error("knight on A1: expected attacks on B3 and C2")
	}

	// Knight on h8 attacks 2 squares: g6, f7.
	att = KnightAttacks[board.H8]
	if att.Count() != 2 {
		t.Errorf("knight on H8: expected 2 attacks, got %d", att.Count())
	}
}

func TestKingAttacks(t *testing.T) {
	// King on e4 attacks 8 squares.
	att := KingAttacks[board.E4]
	if att.Count() != 8 {
		t.Errorf("king on E4: expected 8 attacks, got %d", att.Count())
	}

	// King on a1 attacks 3 squares.
	att = KingAttacks[board.A1]
	if att.Count() != 3 {
		t.Errorf("king on A1: expected 3 attacks, got %d", att.Count())
	}
}

func TestPawnAttacks(t *testing.T) {
	// White pawn on e4 attacks d5 and f5.
	att := PawnAttacks[board.White][board.E4]
	if att.Count() != 2 {
		t.Errorf("white pawn on E4: expected 2 attacks, got %d", att.Count())
	}
	if !att.Has(board.D5) || !att.Has(board.F5) {
		t.Error("white pawn on E4: expected attacks on D5 and F5")
	}

	// White pawn on a4 attacks only b5.
	att = PawnAttacks[board.White][board.A4]
	if att.Count() != 1 {
		t.Errorf("white pawn on A4: expected 1 attack, got %d", att.Count())
	}

	// Black pawn on e5 attacks d4 and f4.
	att = PawnAttacks[board.Black][board.E5]
	if !att.Has(board.D4) || !att.Has(board.F4) {
		t.Error("black pawn on E5: expected attacks on D4 and F4")
	}
}

func TestRookAttacksEmptyBoard(t *testing.T) {
	// Rook on e4, empty board: should attack 14 squares.
	att := RookAttacks(board.E4, 0)
	if att.Count() != 14 {
		t.Errorf("rook on E4 empty: expected 14 attacks, got %d", att.Count())
	}
}

func TestRookAttacksWithBlockers(t *testing.T) {
	// Rook on e4 with blocker on e7: can see e5, e6, e7 but not e8.
	occ := board.SquareBB(board.E7)
	att := RookAttacks(board.E4, occ)
	if !att.Has(board.E5) || !att.Has(board.E6) || !att.Has(board.E7) {
		t.Error("rook should see through to blocker")
	}
	if att.Has(board.E8) {
		t.Error("rook should not see past blocker")
	}
}

func TestBishopAttacksEmptyBoard(t *testing.T) {
	// Bishop on e4, empty board: should attack 13 squares.
	att := BishopAttacks(board.E4, 0)
	if att.Count() != 13 {
		t.Errorf("bishop on E4 empty: expected 13 attacks, got %d", att.Count())
	}
}

func TestQueenAttacks(t *testing.T) {
	// Queen on e4, empty board: rook (14) + bishop (13) = 27.
	att := QueenAttacks(board.E4, 0)
	if att.Count() != 27 {
		t.Errorf("queen on E4 empty: expected 27 attacks, got %d", att.Count())
	}
}

func TestIsSquareAttackedStartPos(t *testing.T) {
	pos := board.NewPosition()

	// e2 is defended by white (pawn on d1? no, by pawns, knights, etc.)
	// d1 queen attacks e2? No. But Ke1 attacks e2. Pawns d2,f2 don't attack e2.
	// Actually: Bf1 attacks e2, Ke1 attacks e2.
	if !IsSquareAttacked(pos, board.E2, board.White) {
		t.Error("e2 should be attacked by white in start position")
	}

	// e4 should not be attacked by black in start position.
	if IsSquareAttacked(pos, board.E4, board.Black) {
		t.Error("e4 should not be attacked by black in start position")
	}

	// d3 should be attacked by white (pawns c2, e2 attack d3).
	if !IsSquareAttacked(pos, board.D3, board.White) {
		t.Error("d3 should be attacked by white pawns")
	}
}
