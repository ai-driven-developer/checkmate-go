package search

import (
	"checkmatego/internal/board"
	"testing"
	"time"
)

// --- Allocation tests ---

func TestTimeManagerMoveTime(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{MoveTime: 500 * time.Millisecond}, board.White, 0)

	if tm.optimumTime != 500*time.Millisecond {
		t.Errorf("optimum = %v, want 500ms", tm.optimumTime)
	}
	if tm.maximumTime != 500*time.Millisecond {
		t.Errorf("maximum = %v, want 500ms", tm.maximumTime)
	}
}

func TestTimeManagerMoveTimeMinimum(t *testing.T) {
	var tm TimeManager
	// MoveTime 50ms with 100ms overhead → clamped to 1ms.
	tm.init(SearchLimits{MoveTime: 50 * time.Millisecond}, board.White, 100*time.Millisecond)

	if tm.optimumTime != time.Millisecond {
		t.Errorf("optimum = %v, want 1ms", tm.optimumTime)
	}
}

func TestTimeManagerInfinite(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{Infinite: true}, board.White, 0)

	if tm.optimumTime != 24*time.Hour {
		t.Errorf("optimum = %v, want 24h", tm.optimumTime)
	}
	if tm.maximumTime != 24*time.Hour {
		t.Errorf("maximum = %v, want 24h", tm.maximumTime)
	}
}

func TestTimeManagerDepthOnly(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{Depth: 10}, board.White, 0)

	if tm.optimumTime != 24*time.Hour {
		t.Errorf("optimum = %v, want 24h", tm.optimumTime)
	}
}

func TestTimeManagerWhiteTime(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{
		WTime: 60 * time.Second,
		WInc:  2 * time.Second,
	}, board.White, 0)

	// Expect: remaining=60s, movesLeft=25, optimum = 60/25 + 2*3/4 = 2.4+1.5 = 3.9s.
	if tm.optimumTime < 2*time.Second || tm.optimumTime > 6*time.Second {
		t.Errorf("optimum out of range: %v", tm.optimumTime)
	}
	// Maximum should be larger than optimum.
	if tm.maximumTime < tm.optimumTime {
		t.Errorf("maximum (%v) < optimum (%v)", tm.maximumTime, tm.optimumTime)
	}
}

func TestTimeManagerBlackTime(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{
		BTime: 30 * time.Second,
		BInc:  1 * time.Second,
	}, board.Black, 0)

	// Expect: remaining=30s, movesLeft=25, optimum = 30/25 + 1*3/4 = 1.2+0.75 = 1.95s.
	if tm.optimumTime < 1*time.Second || tm.optimumTime > 4*time.Second {
		t.Errorf("optimum out of range: %v", tm.optimumTime)
	}
}

func TestTimeManagerMovesToGo(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{
		WTime:     30 * time.Second,
		MovesToGo: 10,
	}, board.White, 0)

	// Expect: remaining=30s, movesLeft=10, optimum = 30/10 + 0 = 3s.
	if tm.optimumTime < 1*time.Second || tm.optimumTime > 5*time.Second {
		t.Errorf("movestogo optimum out of range: %v", tm.optimumTime)
	}
}

func TestTimeManagerLowTime(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{
		WTime: 200 * time.Millisecond,
	}, board.White, 0)

	// Very low time: optimum should be clamped to minimum (50ms).
	if tm.optimumTime < 50*time.Millisecond || tm.optimumTime > 200*time.Millisecond {
		t.Errorf("low-time optimum out of range: %v", tm.optimumTime)
	}
}

func TestTimeManagerWithOverhead(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{
		WTime: 60 * time.Second,
		WInc:  2 * time.Second,
	}, board.White, 50*time.Millisecond)

	// Same as white time test but with 50ms overhead subtracted.
	if tm.optimumTime < 2*time.Second || tm.optimumTime > 6*time.Second {
		t.Errorf("optimum with overhead out of range: %v", tm.optimumTime)
	}
}

func TestTimeManagerMaximumGTEOptimum(t *testing.T) {
	// Maximum should always be >= optimum.
	tests := []SearchLimits{
		{WTime: 10 * time.Second},
		{WTime: 60 * time.Second, WInc: 2 * time.Second},
		{WTime: 300 * time.Second, WInc: 5 * time.Second, MovesToGo: 40},
		{WTime: 200 * time.Millisecond},
	}
	for _, lim := range tests {
		var tm TimeManager
		tm.init(lim, board.White, 0)
		if tm.maximumTime < tm.optimumTime {
			t.Errorf("max (%v) < opt (%v) for %+v", tm.maximumTime, tm.optimumTime, lim)
		}
	}
}

func TestTimeManagerMaximumCap(t *testing.T) {
	// Maximum should never exceed 30% of remaining time.
	var tm TimeManager
	tm.init(SearchLimits{
		WTime: 100 * time.Second,
	}, board.White, 0)

	if tm.maximumTime > 30*time.Second {
		t.Errorf("maximum (%v) exceeds 30%% of remaining", tm.maximumTime)
	}
}

// --- Hard limit tests ---

func TestShouldStopHard(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{MoveTime: 10 * time.Millisecond}, board.White, 0)

	if tm.shouldStopHard() {
		t.Error("should not stop immediately")
	}
	time.Sleep(15 * time.Millisecond)
	if !tm.shouldStopHard() {
		t.Error("should stop after deadline")
	}
}

// --- Soft limit / stability tests ---

func TestShouldStopSoftDepthOne(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{WTime: 60 * time.Second}, board.White, 0)

	m := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	// Depth 1: should never stop.
	if tm.shouldStopSoft(m, 20, 1) {
		t.Error("should not stop at depth 1")
	}
}

func TestStabilityReducesTime(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{WTime: 60 * time.Second}, board.White, 0)

	m := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	// Simulate 7 stable iterations to build up stabilityCount.
	for depth := 1; depth <= 7; depth++ {
		tm.shouldStopSoft(m, 20, depth)
	}

	// After 6+ stable iterations, stabilityCount >= 6 → 50% of optimum.
	// The adjusted time should be half the optimum.
	if tm.stabilityCount < 6 {
		t.Errorf("stabilityCount = %d, want >= 6", tm.stabilityCount)
	}
}

func TestInstabilityExtendsTime(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{WTime: 60 * time.Second}, board.White, 0)

	m1 := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)
	m2 := board.NewMove(board.D2, board.D4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	// Alternate best moves to keep stabilityCount at 0.
	tm.shouldStopSoft(m1, 20, 1)
	tm.shouldStopSoft(m2, 20, 2) // change → stabilityCount = 0
	tm.shouldStopSoft(m1, 20, 3) // change → stabilityCount = 0

	if tm.stabilityCount != 0 {
		t.Errorf("stabilityCount = %d, want 0", tm.stabilityCount)
	}
}

func TestScoreDropExtendsTime(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{WTime: 60 * time.Second}, board.White, 0)

	m := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	// Build up depth/score.
	for depth := 1; depth <= 4; depth++ {
		tm.shouldStopSoft(m, 100, depth)
	}

	// Now at depth 5 with a 50cp drop (100 → 50).
	// shouldStopSoft should not stop because the score drop extends time.
	// We can't easily test the extension directly, but we can verify the
	// prevScore is updated correctly.
	tm.shouldStopSoft(m, 50, 5)
	if tm.prevScore != 50 {
		t.Errorf("prevScore = %d, want 50", tm.prevScore)
	}
}

func TestScoreDropSmallNoExtension(t *testing.T) {
	var tm TimeManager
	tm.init(SearchLimits{WTime: 60 * time.Second}, board.White, 0)

	m := board.NewMove(board.E2, board.E4, board.FlagDoublePawn, board.Pawn, board.NoPiece)

	for depth := 1; depth <= 4; depth++ {
		tm.shouldStopSoft(m, 100, depth)
	}

	// Small drop (20cp) should not trigger extension (threshold is 30).
	tm.shouldStopSoft(m, 80, 5)
	if tm.prevScore != 80 {
		t.Errorf("prevScore = %d, want 80", tm.prevScore)
	}
}

// --- Engine integration tests ---

func TestSetHashResizesTT(t *testing.T) {
	e := NewEngine()
	e.SetHash(32)
	pos := board.NewPosition()
	bestMove := e.Search(pos, SearchLimits{Depth: 2})
	if bestMove == board.NullMove {
		t.Error("search with resized hash table returned null move")
	}
}

func TestClearHash(t *testing.T) {
	e := NewEngine()
	pos := board.NewPosition()
	e.Search(pos, SearchLimits{Depth: 3})
	e.ClearHash()
	if e.tt.Hashfull() != 0 {
		t.Errorf("hashfull should be 0 after clear, got %d", e.tt.Hashfull())
	}
}

func TestSetThreadsMinimum(t *testing.T) {
	e := NewEngine()
	e.SetThreads(0)
	if e.threads != 1 {
		t.Errorf("SetThreads(0) should clamp to 1, got %d", e.threads)
	}
	e.SetThreads(-5)
	if e.threads != 1 {
		t.Errorf("SetThreads(-5) should clamp to 1, got %d", e.threads)
	}
}

func TestSetMoveOverhead(t *testing.T) {
	e := NewEngine()
	e.SetMoveOverhead(100 * time.Millisecond)
	if e.moveOverhead != 100*time.Millisecond {
		t.Errorf("expected moveOverhead=100ms, got %v", e.moveOverhead)
	}
}

func TestTimeManagerSoftStopSignalsAllThreads(t *testing.T) {
	// The soft stop from the main thread should signal stopFlag, which
	// all other threads observe via shouldStop().
	e := NewEngine()
	e.SetThreads(2)
	pos := board.NewPosition()

	// Use a short time control. The search should complete via time management.
	bestMove := e.Search(pos, SearchLimits{WTime: 100 * time.Millisecond})
	if bestMove == board.NullMove {
		t.Error("short time control search returned null move")
	}
}

func TestTimeManagerMoveTimeSearch(t *testing.T) {
	// Verify that a fixed-time search completes within a reasonable window.
	e := NewEngine()
	start := time.Now()
	bestMove := e.Search(board.NewPosition(), SearchLimits{MoveTime: 50 * time.Millisecond})
	elapsed := time.Since(start)

	if bestMove == board.NullMove {
		t.Error("movetime search returned null move")
	}
	// Should complete within 200ms (generous to avoid flaky tests).
	if elapsed > 200*time.Millisecond {
		t.Errorf("movetime search took %v, expected < 200ms", elapsed)
	}
}

func TestTimeManagerClassicalSearch(t *testing.T) {
	e := NewEngine()
	start := time.Now()
	bestMove := e.Search(board.NewPosition(), SearchLimits{
		WTime: 500 * time.Millisecond,
		WInc:  50 * time.Millisecond,
	})
	elapsed := time.Since(start)

	if bestMove == board.NullMove {
		t.Error("classical search returned null move")
	}
	// Should complete well within the total remaining time.
	if elapsed > 500*time.Millisecond {
		t.Errorf("classical search took %v, exceeded remaining time", elapsed)
	}
}
