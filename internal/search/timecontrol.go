package search

import (
	"checkmatego/internal/board"
	"time"
)

// TimeManager handles adaptive time allocation with soft and hard limits.
// The soft limit (optimumTime) is the target duration per move; the hard
// limit (maximumTime) is the absolute ceiling. After each completed
// iteration the main thread checks the soft limit, which may be adjusted
// by move stability and score-drop heuristics. During the search all
// threads check the hard limit periodically to guarantee we never exceed
// the maximum allowed time.
type TimeManager struct {
	optimumTime time.Duration
	maximumTime time.Duration
	startTime   time.Time

	// Stability tracking (main thread only).
	bestMove       board.Move
	stabilityCount int
	prevScore      int
}

func (tm *TimeManager) init(limits SearchLimits, color board.Color, overhead time.Duration) {
	tm.startTime = time.Now()
	tm.bestMove = board.NullMove
	tm.stabilityCount = 0
	tm.prevScore = 0

	if limits.MoveTime > 0 {
		t := limits.MoveTime - overhead
		if t < time.Millisecond {
			t = time.Millisecond
		}
		tm.optimumTime = t
		tm.maximumTime = t
		return
	}

	if limits.Infinite || limits.Depth > 0 {
		tm.optimumTime = 24 * time.Hour
		tm.maximumTime = 24 * time.Hour
		return
	}

	var remaining, inc time.Duration
	if color == board.White {
		remaining = limits.WTime
		inc = limits.WInc
	} else {
		remaining = limits.BTime
		inc = limits.BInc
	}

	remaining -= overhead

	movesLeft := limits.MovesToGo
	if movesLeft == 0 {
		// Adaptive estimate: be more conservative when time is low
		// (typical of bullet) and spend more when comfortable.
		switch {
		case remaining < 5*time.Second:
			movesLeft = 40
		case remaining < 15*time.Second:
			movesLeft = 30
		case remaining < 60*time.Second:
			movesLeft = 25
		default:
			movesLeft = 22
		}
	}

	// Optimum: base time per move + 3/4 of increment.
	optimum := remaining/time.Duration(movesLeft) + inc*3/4

	// Maximum: up to 30% of remaining time, capped at 5x optimum.
	maximum := remaining * 3 / 10
	if maximum > optimum*5 {
		maximum = optimum * 5
	}

	// Safety: never exceed the remaining clock time after overhead.
	// Note: overhead has already been subtracted from remaining above.
	safeMax := remaining
	if safeMax < 0 {
		safeMax = 0
	}
	if optimum > safeMax {
		optimum = safeMax
	}
	if maximum > safeMax {
		maximum = safeMax
	}

	// Minimum clamps.
	if optimum < 50*time.Millisecond {
		optimum = 50 * time.Millisecond
	}
	if maximum < optimum {
		maximum = optimum
	}
	if optimum <= 0 {
		optimum = 50 * time.Millisecond
	}
	if maximum <= 0 {
		maximum = 50 * time.Millisecond
	}

	tm.optimumTime = optimum
	tm.maximumTime = maximum
}

// elapsed returns the time spent since the search started.
func (tm *TimeManager) elapsed() time.Duration {
	return time.Since(tm.startTime)
}

// shouldStopHard returns true when the hard time limit has been exceeded.
// Called from every worker thread (read-only, no side effects).
func (tm *TimeManager) shouldStopHard() bool {
	return tm.elapsed() >= tm.maximumTime
}

// shouldStopSoft is called by the main thread after each completed
// iteration. It updates stability tracking and decides whether to
// continue searching based on the adjusted optimum time.
func (tm *TimeManager) shouldStopSoft(bestMove board.Move, score int, depth int) bool {
	if depth <= 1 {
		tm.prevScore = score
		tm.bestMove = bestMove
		return false
	}

	// Update stability tracking.
	if bestMove == tm.bestMove {
		tm.stabilityCount++
	} else {
		tm.bestMove = bestMove
		tm.stabilityCount = 0
	}

	// Start with the base optimum time.
	adjusted := tm.optimumTime

	// Stability: scale down for stable moves, scale up for unstable ones.
	// More granular tiers help bullet play fast on obvious moves.
	switch {
	case tm.stabilityCount >= 8:
		adjusted = adjusted * 40 / 100 // 40%
	case tm.stabilityCount >= 6:
		adjusted = adjusted * 50 / 100 // 50%
	case tm.stabilityCount >= 4:
		adjusted = adjusted * 65 / 100 // 65%
	case tm.stabilityCount >= 2:
		adjusted = adjusted * 85 / 100 // 85%
	case tm.stabilityCount == 0:
		adjusted = adjusted * 130 / 100 // 130%
	}

	// Score drop: extend thinking time when the score drops significantly.
	drop := tm.prevScore - score
	if depth >= 5 && drop > 30 {
		// Scale: 30 cp drop → 1.15x, 100+ cp drop → 1.5x.
		factor := 100 + min(drop, 100)*50/100
		adjusted = adjusted * time.Duration(factor) / 100
	}

	tm.prevScore = score

	// Never exceed the hard limit.
	if adjusted > tm.maximumTime {
		adjusted = tm.maximumTime
	}

	return tm.elapsed() >= adjusted
}
