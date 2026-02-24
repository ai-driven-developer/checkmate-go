package search

import (
	"checkmatego/internal/board"
	"checkmatego/internal/eval"
	"checkmatego/internal/movegen"
	"time"
)

// worker is a per-thread search context for Lazy SMP.
type worker struct {
	engine *Engine        // shared state (nodes, stopFlag, deadline, limits, start)
	pos    board.Position // per-worker position copy
	id     int
}

// workerResult holds the outcome of a worker's search.
type workerResult struct {
	move  board.Move
	score int
	depth int
}

func (w *worker) search(maxDepth int) workerResult {
	result := workerResult{move: board.NullMove}

	for depth := 1; depth <= maxDepth; depth++ {
		score, pv := w.negamax(depth, -Infinity, Infinity, 0)
		if w.shouldStop() && depth > 1 {
			break
		}
		if len(pv) > 0 {
			result.move = pv[0]
			result.score = score
			result.depth = depth
		}
		// Only main thread reports info.
		if w.id == 0 && w.engine.onInfo != nil {
			w.engine.onInfo(SearchInfo{
				Depth: depth,
				Score: score,
				Nodes: w.engine.nodes.Load(),
				Time:  time.Since(w.engine.start),
				PV:    pv,
			})
		}
		// Stop if we found a forced mate.
		if score > MateScore-MaxDepth || score < -MateScore+MaxDepth {
			break
		}
	}
	return result
}

func (w *worker) shouldStop() bool {
	if w.engine.stopFlag.Load() {
		return true
	}
	if !w.engine.limits.Infinite && w.engine.limits.Depth == 0 {
		return time.Now().After(w.engine.deadline)
	}
	return false
}

func (w *worker) negamax(depth, alpha, beta, ply int) (int, []board.Move) {
	// Check time every 4096 nodes.
	if w.engine.nodes.Load()&4095 == 0 && ply > 0 {
		if w.shouldStop() {
			return 0, nil
		}
	}

	if depth <= 0 {
		return w.quiesce(alpha, beta, ply), nil
	}

	w.engine.nodes.Add(1)

	// 50-move rule.
	if w.pos.HalfMoveClock >= 100 {
		return 0, nil
	}

	var ml board.MoveList
	movegen.GenerateLegalMoves(&w.pos, &ml)

	if ml.Count == 0 {
		if movegen.IsSquareAttacked(&w.pos, w.pos.KingSquare(w.pos.SideToMove), w.pos.SideToMove.Other()) {
			return -MateScore + ply, nil // checkmate
		}
		return 0, nil // stalemate
	}

	OrderMoves(&ml)

	var bestPV []board.Move
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		w.pos.MakeMove(m)
		score, childPV := w.negamax(depth-1, -beta, -alpha, ply+1)
		score = -score
		w.pos.UnmakeMove(m)

		if w.shouldStop() && ply > 0 {
			return 0, nil
		}

		if score > alpha {
			alpha = score
			bestPV = make([]board.Move, 1+len(childPV))
			bestPV[0] = m
			copy(bestPV[1:], childPV)
			if alpha >= beta {
				break
			}
		}
	}
	return alpha, bestPV
}

func (w *worker) quiesce(alpha, beta, ply int) int {
	w.engine.nodes.Add(1)

	standPat := eval.Evaluate(&w.pos)
	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	var ml board.MoveList
	movegen.GenerateCaptures(&w.pos, &ml)
	OrderMoves(&ml)

	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		w.pos.MakeMove(m)
		score := -w.quiesce(-beta, -alpha, ply+1)
		w.pos.UnmakeMove(m)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}
	return alpha
}
