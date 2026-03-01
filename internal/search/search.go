package search

import (
	"checkmatego/internal/board"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Infinity  = 30000
	MateScore = 29000
	MaxDepth  = 64
)

// SearchLimits defines constraints on the search.
type SearchLimits struct {
	Depth     int
	Nodes     uint64
	MoveTime  time.Duration
	Infinite  bool
	WTime     time.Duration
	BTime     time.Duration
	WInc      time.Duration
	BInc      time.Duration
	MovesToGo int
}

// SearchInfo holds real-time search statistics.
type SearchInfo struct {
	Depth    int
	Score    int
	Nodes    uint64
	Time     time.Duration
	PV       []board.Move
	Hashfull int
}

// InfoCallback is called when the search has new info to report.
type InfoCallback func(info SearchInfo)

// Engine holds the search state.
type Engine struct {
	color        board.Color
	limits       SearchLimits
	onInfo       InfoCallback
	nodes        atomic.Uint64
	stopFlag     atomic.Bool
	tm           TimeManager
	start        time.Time
	moveOverhead time.Duration
	threads      int
	tt           *TransTable
}

func NewEngine() *Engine {
	return &Engine{
		threads: 1,
		tt:      NewTransTable(64),
	}
}

// SetHash resizes the transposition table.
func (e *Engine) SetHash(sizeMB int) {
	e.tt = NewTransTable(sizeMB)
}

// ClearHash zeroes the transposition table.
func (e *Engine) ClearHash() {
	e.tt.Clear()
}

// SetThreads sets the number of search threads.
func (e *Engine) SetThreads(n int) {
	if n < 1 {
		n = 1
	}
	e.threads = n
}

// SetMoveOverhead sets the time reserved for communication overhead.
func (e *Engine) SetMoveOverhead(d time.Duration) {
	e.moveOverhead = d
}

// SetInfoCallback sets the callback for search info updates.
func (e *Engine) SetInfoCallback(cb InfoCallback) {
	e.onInfo = cb
}

// Search starts the search and returns the best move.
func (e *Engine) Search(pos *board.Position, limits SearchLimits) board.Move {
	e.color = pos.SideToMove
	e.limits = limits
	e.nodes.Store(0)
	e.stopFlag.Store(false)
	e.tm.init(limits, pos.SideToMove, e.moveOverhead)
	e.start = e.tm.startTime
	e.tt.NewSearch()

	maxDepth := MaxDepth
	if limits.Depth > 0 {
		maxDepth = limits.Depth
	}

	numThreads := e.threads
	if numThreads < 1 {
		numThreads = 1
	}

	results := make([]workerResult, numThreads)
	var wg sync.WaitGroup

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w := &worker{
				engine: e,
				pos:    *pos,
				id:     id,
			}
			results[id] = w.search(maxDepth)
		}(i)
	}

	wg.Wait()

	// Pick the best result: deepest depth wins, then highest score.
	best := results[0]
	for _, r := range results[1:] {
		if r.depth > best.depth || (r.depth == best.depth && r.score > best.score) {
			best = r
		}
	}
	return best.move
}

// Stop signals the engine to stop searching.
func (e *Engine) Stop() {
	e.stopFlag.Store(true)
}
