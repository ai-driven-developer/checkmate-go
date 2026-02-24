package search

import (
	"checkmatego/internal/board"
	"time"
)

func (e *Engine) computeDeadline() time.Time {
	overhead := e.moveOverhead

	if e.limits.MoveTime > 0 {
		t := e.limits.MoveTime - overhead
		if t < time.Millisecond {
			t = time.Millisecond
		}
		return time.Now().Add(t)
	}
	if e.limits.Infinite || e.limits.Depth > 0 {
		return time.Now().Add(24 * time.Hour)
	}

	var remaining, inc time.Duration
	if e.color == board.White {
		remaining = e.limits.WTime
		inc = e.limits.WInc
	} else {
		remaining = e.limits.BTime
		inc = e.limits.BInc
	}

	// Subtract overhead from available time.
	remaining -= overhead

	movesLeft := e.limits.MovesToGo
	if movesLeft == 0 {
		movesLeft = 30
	}

	allocated := remaining/time.Duration(movesLeft) + inc - 50*time.Millisecond
	if allocated < 100*time.Millisecond {
		allocated = 100 * time.Millisecond
	}
	if allocated > remaining-100*time.Millisecond {
		allocated = remaining - 100*time.Millisecond
	}
	if allocated <= 0 {
		allocated = 50 * time.Millisecond
	}

	return time.Now().Add(allocated)
}
