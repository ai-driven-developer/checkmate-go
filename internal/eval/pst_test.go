package eval

import (
	"checkmatego/internal/board"
	"testing"
)

func TestPSTBalanceStartPosition(t *testing.T) {
	pos := board.NewPosition()
	mg, eg := pstBalanceTapered(pos)
	// Start position is symmetric — both MG and EG PST should be 0.
	if mg != 0 {
		t.Errorf("start position MG PST balance should be 0, got %d", mg)
	}
	if eg != 0 {
		t.Errorf("start position EG PST balance should be 0, got %d", eg)
	}
}

func TestPSTCentralPawnsPreferred(t *testing.T) {
	// A position where white has pawns advanced to the center should score better.
	pos1 := &board.Position{}
	_ = pos1.SetFromFEN("4k3/8/8/8/3PP3/8/8/4K3 w - - 0 1")
	mg1, _ := pstBalanceTapered(pos1)

	pos2 := &board.Position{}
	_ = pos2.SetFromFEN("4k3/8/8/8/P6P/8/8/4K3 w - - 0 1")
	mg2, _ := pstBalanceTapered(pos2)

	if mg1 <= mg2 {
		t.Errorf("central pawns should score higher: center=%d, edge=%d", mg1, mg2)
	}
}

func TestPSTKnightCenter(t *testing.T) {
	// Knight on e4 should have a better PST score than knight on a1.
	if knightPST[board.E4] <= knightPST[board.A1] {
		t.Error("knight PST should prefer center over corner")
	}
}

func TestKingEndgamePSTPreferCenter(t *testing.T) {
	// In the endgame table, king on e4 should score better than on e1.
	if kingEndgamePST[board.E4] <= kingEndgamePST[board.E1] {
		t.Errorf("endgame king should prefer center: e4=%d, e1=%d",
			kingEndgamePST[board.E4], kingEndgamePST[board.E1])
	}
}

func TestKingMiddlegamePSTPreferEdge(t *testing.T) {
	// In the middlegame table, king on g1 should score better than on e4.
	if kingMiddlegamePST[board.G1] <= kingMiddlegamePST[board.E4] {
		t.Errorf("middlegame king should prefer castled position: g1=%d, e4=%d",
			kingMiddlegamePST[board.G1], kingMiddlegamePST[board.E4])
	}
}
