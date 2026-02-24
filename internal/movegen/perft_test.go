package movegen

import (
	"checkmatego/internal/board"
	"testing"
)

func TestPerftStartPosition(t *testing.T) {
	pos := board.NewPosition()

	tests := []struct {
		depth int
		nodes uint64
	}{
		{1, 20},
		{2, 400},
		{3, 8902},
		{4, 197281},
		{5, 4865609},
	}

	for _, tt := range tests {
		got := Perft(pos, tt.depth)
		if got != tt.nodes {
			t.Errorf("perft(%d) start position: got %d, want %d", tt.depth, got, tt.nodes)
		}
	}
}

func TestPerftKiwiPete(t *testing.T) {
	// "Kiwi Pete" — exercises castling, en passant, promotions, and pins.
	pos := &board.Position{}
	_ = pos.SetFromFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	tests := []struct {
		depth int
		nodes uint64
	}{
		{1, 48},
		{2, 2039},
		{3, 97862},
		{4, 4085603},
	}

	for _, tt := range tests {
		got := Perft(pos, tt.depth)
		if got != tt.nodes {
			t.Errorf("perft(%d) Kiwi Pete: got %d, want %d", tt.depth, got, tt.nodes)
		}
	}
}

func TestPerftPosition3(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1")

	tests := []struct {
		depth int
		nodes uint64
	}{
		{1, 14},
		{2, 191},
		{3, 2812},
		{4, 43238},
		{5, 674624},
	}

	for _, tt := range tests {
		got := Perft(pos, tt.depth)
		if got != tt.nodes {
			t.Errorf("perft(%d) position 3: got %d, want %d", tt.depth, got, tt.nodes)
		}
	}
}

func TestPerftPosition4(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1")

	tests := []struct {
		depth int
		nodes uint64
	}{
		{1, 6},
		{2, 264},
		{3, 9467},
		{4, 422333},
	}

	for _, tt := range tests {
		got := Perft(pos, tt.depth)
		if got != tt.nodes {
			t.Errorf("perft(%d) position 4: got %d, want %d", tt.depth, got, tt.nodes)
		}
	}
}

func TestPerftPosition5(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8")

	tests := []struct {
		depth int
		nodes uint64
	}{
		{1, 44},
		{2, 1486},
		{3, 62379},
		{4, 2103487},
	}

	for _, tt := range tests {
		got := Perft(pos, tt.depth)
		if got != tt.nodes {
			t.Errorf("perft(%d) position 5: got %d, want %d", tt.depth, got, tt.nodes)
		}
	}
}

func BenchmarkPerftDepth5(b *testing.B) {
	pos := board.NewPosition()
	for i := 0; i < b.N; i++ {
		Perft(pos, 5)
	}
}
