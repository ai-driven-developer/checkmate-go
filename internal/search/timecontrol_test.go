package search

import (
	"checkmatego/internal/board"
	"testing"
	"time"
)

func TestComputeDeadlineMoveTime(t *testing.T) {
	e := NewEngine()
	e.limits = SearchLimits{MoveTime: 500 * time.Millisecond}

	before := time.Now()
	deadline := e.computeDeadline()
	after := time.Now()

	// Deadline should be ~500ms from now (minus overhead of 0 by default).
	minExpected := before.Add(490 * time.Millisecond)
	maxExpected := after.Add(510 * time.Millisecond)
	if deadline.Before(minExpected) || deadline.After(maxExpected) {
		t.Errorf("movetime deadline out of range: got %v, expected between %v and %v",
			deadline, minExpected, maxExpected)
	}
}

func TestComputeDeadlineMoveTimeMinimum(t *testing.T) {
	e := NewEngine()
	// Very short movetime with overhead should clamp to at least 1ms.
	e.moveOverhead = 100 * time.Millisecond
	e.limits = SearchLimits{MoveTime: 50 * time.Millisecond}

	before := time.Now()
	deadline := e.computeDeadline()

	// movetime(50) - overhead(100) = -50ms, clamped to 1ms.
	expected := before.Add(1 * time.Millisecond)
	if deadline.Before(expected.Add(-5 * time.Millisecond)) {
		t.Errorf("movetime deadline should be at least 1ms from now, got %v", deadline.Sub(before))
	}
}

func TestComputeDeadlineInfinite(t *testing.T) {
	e := NewEngine()
	e.limits = SearchLimits{Infinite: true}

	before := time.Now()
	deadline := e.computeDeadline()

	// Infinite search: deadline should be very far in the future (24h).
	if deadline.Before(before.Add(23 * time.Hour)) {
		t.Errorf("infinite search deadline should be ~24h away, got %v", deadline.Sub(before))
	}
}

func TestComputeDeadlineDepthOnly(t *testing.T) {
	e := NewEngine()
	e.limits = SearchLimits{Depth: 10}

	before := time.Now()
	deadline := e.computeDeadline()

	// Depth-only search: deadline should be very far in the future.
	if deadline.Before(before.Add(23 * time.Hour)) {
		t.Errorf("depth-only search deadline should be ~24h away, got %v", deadline.Sub(before))
	}
}

func TestComputeDeadlineWhiteTime(t *testing.T) {
	e := NewEngine()
	e.color = board.White
	e.limits = SearchLimits{
		WTime: 60 * time.Second,
		WInc:  2 * time.Second,
	}

	before := time.Now()
	deadline := e.computeDeadline()

	// With 60s remaining and default movesLeft=30: allocated = 60/30 + 2 - 0.05 = ~4s.
	allocated := deadline.Sub(before)
	if allocated < 1*time.Second || allocated > 10*time.Second {
		t.Errorf("white time allocation out of range: %v", allocated)
	}
}

func TestComputeDeadlineBlackTime(t *testing.T) {
	e := NewEngine()
	e.color = board.Black
	e.limits = SearchLimits{
		BTime: 30 * time.Second,
		BInc:  1 * time.Second,
	}

	before := time.Now()
	deadline := e.computeDeadline()

	// With 30s remaining, movesLeft=30: allocated = 30/30 + 1 - 0.05 = ~1.95s.
	allocated := deadline.Sub(before)
	if allocated < 500*time.Millisecond || allocated > 5*time.Second {
		t.Errorf("black time allocation out of range: %v", allocated)
	}
}

func TestComputeDeadlineMovesToGo(t *testing.T) {
	e := NewEngine()
	e.color = board.White
	e.limits = SearchLimits{
		WTime:     30 * time.Second,
		MovesToGo: 10,
	}

	before := time.Now()
	deadline := e.computeDeadline()

	// With 30s remaining, movesToGo=10: allocated = 30/10 + 0 - 0.05 = ~2.95s.
	allocated := deadline.Sub(before)
	if allocated < 1*time.Second || allocated > 5*time.Second {
		t.Errorf("movestogo allocation out of range: %v", allocated)
	}
}

func TestComputeDeadlineLowTime(t *testing.T) {
	e := NewEngine()
	e.color = board.White
	e.limits = SearchLimits{
		WTime: 200 * time.Millisecond,
	}

	before := time.Now()
	deadline := e.computeDeadline()

	// Very low time: allocation should be clamped to minimum (50-100ms).
	allocated := deadline.Sub(before)
	if allocated < 40*time.Millisecond || allocated > 200*time.Millisecond {
		t.Errorf("low-time allocation out of range: %v", allocated)
	}
}

func TestComputeDeadlineWithOverhead(t *testing.T) {
	e := NewEngine()
	e.color = board.White
	e.moveOverhead = 50 * time.Millisecond
	e.limits = SearchLimits{
		WTime: 60 * time.Second,
		WInc:  2 * time.Second,
	}

	before := time.Now()
	deadline := e.computeDeadline()

	// Same as white time test but with 50ms overhead subtracted from remaining.
	allocated := deadline.Sub(before)
	if allocated < 1*time.Second || allocated > 10*time.Second {
		t.Errorf("allocation with overhead out of range: %v", allocated)
	}
}

func TestSetHashResizesTT(t *testing.T) {
	e := NewEngine()
	e.SetHash(32)
	// Should not panic and should be usable.
	pos := board.NewPosition()
	bestMove := e.Search(pos, SearchLimits{Depth: 2})
	if bestMove == board.NullMove {
		t.Error("search with resized hash table returned null move")
	}
}

func TestClearHash(t *testing.T) {
	e := NewEngine()
	// Do a search to populate TT.
	pos := board.NewPosition()
	e.Search(pos, SearchLimits{Depth: 3})
	// Clear and verify it doesn't crash.
	e.ClearHash()
	// Hashfull should be 0 after clear.
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
