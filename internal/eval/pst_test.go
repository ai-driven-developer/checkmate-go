package eval

import (
	"checkmatego/internal/board"
	"testing"
)

func TestPSTBalanceStartPosition(t *testing.T) {
	pos := board.NewPosition()
	pst := pstBalance(pos)
	// Start position is symmetric — PST should be 0.
	if pst != 0 {
		t.Errorf("start position PST balance should be 0, got %d", pst)
	}
}

func TestPSTCentralPawnsPreferred(t *testing.T) {
	// A position where white has pawns advanced to the center should score better.
	pos1 := &board.Position{}
	_ = pos1.SetFromFEN("4k3/8/8/8/3PP3/8/8/4K3 w - - 0 1")
	pst1 := pstBalance(pos1)

	pos2 := &board.Position{}
	_ = pos2.SetFromFEN("4k3/8/8/8/P6P/8/8/4K3 w - - 0 1")
	pst2 := pstBalance(pos2)

	if pst1 <= pst2 {
		t.Errorf("central pawns should score higher: center=%d, edge=%d", pst1, pst2)
	}
}

func TestPSTKnightCenter(t *testing.T) {
	// Knight on e4 should have a better PST score than knight on a1.
	if knightPST[board.E4] <= knightPST[board.A1] {
		t.Error("knight PST should prefer center over corner")
	}
}
