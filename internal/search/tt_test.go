package search

import (
	"checkmatego/internal/board"
	"sync"
	"testing"
	"unsafe"
)

func TestTTEntrySize(t *testing.T) {
	size := unsafe.Sizeof(ttEntry{})
	if size != 16 {
		t.Errorf("ttEntry size = %d, want 16", size)
	}
}

func TestTTStoreAndProbe(t *testing.T) {
	tt := NewTransTable(1)
	hash := uint64(0xDEADBEEF12345678)
	move := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	tt.Store(hash, move, 150, 10, BoundExact)

	hit, gotMove, gotScore, gotDepth, gotBound := tt.Probe(hash)
	if !hit {
		t.Fatal("expected hit")
	}
	if gotMove != move {
		t.Errorf("move: got %v, want %v", gotMove, move)
	}
	if gotScore != 150 {
		t.Errorf("score: got %d, want 150", gotScore)
	}
	if gotDepth != 10 {
		t.Errorf("depth: got %d, want 10", gotDepth)
	}
	if gotBound != BoundExact {
		t.Errorf("bound: got %d, want Exact", gotBound)
	}
}

func TestTTMiss(t *testing.T) {
	tt := NewTransTable(1)
	hit, _, _, _, _ := tt.Probe(0x1234)
	if hit {
		t.Error("expected miss on empty table")
	}
}

func TestTTNegativeScore(t *testing.T) {
	tt := NewTransTable(1)
	hash := uint64(0xAABBCCDD00112233)
	tt.Store(hash, board.NullMove, -500, 5, BoundUpper)

	hit, _, score, _, _ := tt.Probe(hash)
	if !hit {
		t.Fatal("expected hit")
	}
	if score != -500 {
		t.Errorf("score: got %d, want -500", score)
	}
}

func TestTTMateScoreAdjustment(t *testing.T) {
	// Mate in 5 plies from root.
	score := MateScore - 5
	ply := 3

	stored := scoreToTT(score, ply)
	restored := scoreFromTT(stored, ply)

	if restored != score {
		t.Errorf("mate score round-trip: stored=%d, restored=%d, want %d", stored, restored, score)
	}

	// Mated score (being mated).
	score = -MateScore + 5
	stored = scoreToTT(score, ply)
	restored = scoreFromTT(stored, ply)

	if restored != score {
		t.Errorf("mated score round-trip: stored=%d, restored=%d, want %d", stored, restored, score)
	}
}

func TestTTReplacementDepth(t *testing.T) {
	tt := NewTransTable(1)
	hash := uint64(0xFFFF0000FFFF0000)
	move1 := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	move2 := board.NewMove(board.D2, board.D4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	tt.Store(hash, move1, 100, 10, BoundLower)
	tt.Store(hash, move2, 50, 5, BoundLower) // shallower, same gen

	hit, gotMove, _, _, _ := tt.Probe(hash)
	if !hit {
		t.Fatal("expected hit")
	}
	if gotMove != move1 {
		t.Error("shallower entry should not replace deeper one in same generation")
	}
}

func TestTTReplacementGeneration(t *testing.T) {
	tt := NewTransTable(1)
	hash := uint64(0xFFFF0000FFFF0000)
	move1 := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	move2 := board.NewMove(board.D2, board.D4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	tt.Store(hash, move1, 100, 10, BoundLower)
	tt.NewSearch()
	tt.Store(hash, move2, 50, 5, BoundLower) // shallower but newer gen

	hit, gotMove, _, _, _ := tt.Probe(hash)
	if !hit {
		t.Fatal("expected hit")
	}
	if gotMove != move2 {
		t.Error("newer generation should replace older entry")
	}
}

func TestTTConcurrentAccess(t *testing.T) {
	tt := NewTransTable(1)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 10000; i++ {
				hash := uint64(id*10000+i) * 0x9E3779B97F4A7C15
				move := board.NewMove(board.Square(i%64), board.Square((i+1)%64), board.FlagQuiet, board.Pawn, board.NoPiece)
				tt.Store(hash, move, int16(i%1000), int8(i%64), BoundExact)
				tt.Probe(hash)
			}
		}(g)
	}
	wg.Wait()
}

func TestTTHashfull(t *testing.T) {
	tt := NewTransTable(1)
	if tt.Hashfull() != 0 {
		t.Error("empty table should have 0 hashfull")
	}
	for i := uint64(0); i < 500; i++ {
		tt.Store(i*0x9E3779B97F4A7C15, board.NullMove, 0, 1, BoundExact)
	}
	hf := tt.Hashfull()
	if hf == 0 {
		t.Error("hashfull should be > 0 after storing entries")
	}
}

func TestTTClear(t *testing.T) {
	tt := NewTransTable(1)
	tt.Store(0xDEAD000000000000, board.NullMove, 100, 5, BoundExact)
	tt.Clear()
	hit, _, _, _, _ := tt.Probe(0xDEAD000000000000)
	if hit {
		t.Error("clear should remove all entries")
	}
}
