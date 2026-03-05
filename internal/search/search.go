package search

import (
	"checkmatego/internal/board"
	"checkmatego/internal/eval"
	"checkmatego/internal/nnue"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Infinity  = 30000
	MateScore = 29000
	MaxDepth  = 64

	// maxHistory caps the history heuristic values. Used by the gravity
	// formula to keep entries bounded in [-maxHistory, maxHistory].
	maxHistory = 16384

	// Razoring margin per depth ply: if staticEval + depth*razoringMargin <= alpha,
	// verify with quiescence search and prune if confirmed.
	razoringMargin = 300

	// ProbCut margin: if a shallow capture search scores >= beta + probCutMargin,
	// the full-depth search is unlikely to score below beta.
	probCutMargin = 200

	// Delta pruning constants for quiescence search.
	// deltaMargin is the "big delta": if standPat + deltaMargin < alpha,
	// no capture can possibly raise alpha, so the node is pruned entirely.
	// Set to queen value (900) + a safety margin (200) for positional swings.
	deltaMargin = 1100

	// deltaPruningMargin is the per-move margin added to the captured piece
	// value. If standPat + captureValue + deltaPruningMargin < alpha,
	// that individual capture is skipped.
	deltaPruningMargin = 200
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
	net          *nnue.Network
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

// SetNetwork sets the NNUE network for evaluation. Pass nil to use HCE.
func (e *Engine) SetNetwork(net *nnue.Network) {
	e.net = net
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
				engine:    e,
				pos:       *pos,
				id:        id,
				pawnCache: eval.NewPawnCache(16384),
				net:       e.net,
			}
			if w.net != nil {
				w.accStack = nnue.NewAccumulatorStack(w.net)
				w.accStack.Refresh(&w.pos)
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
