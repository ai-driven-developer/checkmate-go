package search

import (
	"checkmatego/internal/board"
	"checkmatego/internal/eval"
	"checkmatego/internal/movegen"
	"time"
)

// worker is a per-thread search context for Lazy SMP.
type worker struct {
	engine       *Engine        // shared state (nodes, stopFlag, deadline, limits, start, tt)
	pos          board.Position // per-worker position copy
	id           int
	killers      [MaxDepth][2]board.Move // killer moves per ply
	history      [2][64][64]int32        // history heuristic [color][from][to]
	excludedMove board.Move              // move to skip during singular extension search
}

// workerResult holds the outcome of a worker's search.
type workerResult struct {
	move  board.Move
	score int
	depth int
}

func (w *worker) search(maxDepth int) workerResult {
	result := workerResult{move: board.NullMove}

	w.history = [2][64][64]int32{}

	prevScore := 0

	for depth := 1; depth <= maxDepth; depth++ {
		alpha, beta := -Infinity, Infinity

		// Aspiration windows: use a narrow window around previous score.
		delta := 25
		if depth >= 4 {
			alpha = prevScore - delta
			beta = prevScore + delta
		}

		var score int
		var pv []board.Move

		for {
			score, pv = w.negamax(depth, alpha, beta, 0, true)
			if w.shouldStop() && depth > 1 {
				break
			}
			if score <= alpha {
				// Fail low: widen lower bound.
				alpha -= delta
				if alpha < -Infinity {
					alpha = -Infinity
				}
				delta *= 2
			} else if score >= beta {
				// Fail high: widen upper bound.
				beta += delta
				if beta > Infinity {
					beta = Infinity
				}
				delta *= 2
			} else {
				break
			}
		}

		if w.shouldStop() && depth > 1 {
			break
		}

		prevScore = score

		if len(pv) > 0 {
			result.move = pv[0]
			result.score = score
			result.depth = depth
		}
		// Only main thread reports info.
		if w.id == 0 && w.engine.onInfo != nil {
			w.engine.onInfo(SearchInfo{
				Depth:    depth,
				Score:    score,
				Nodes:    w.engine.nodes.Load(),
				Time:     time.Since(w.engine.start),
				PV:       pv,
				Hashfull: w.engine.tt.Hashfull(),
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

func (w *worker) negamax(depth, alpha, beta, ply int, nullAllowed bool) (int, []board.Move) {
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

	// Repetition detection (2-fold: draw if position seen before).
	if ply > 0 && w.pos.IsRepetition() {
		return 0, nil
	}

	// TT probe.
	tt := w.engine.tt
	hash := w.pos.Hash
	// Modify hash when inside a singular extension search to avoid TT pollution.
	if w.excludedMove != board.NullMove {
		hash ^= uint64(w.excludedMove) * 0x5a3e7f1b2c4d6e8f
	}
	var hashMove board.Move
	var ttHit bool
	var ttScoreRaw int16
	var ttDepthRaw int8
	var ttBoundRaw Bound
	isPV := beta-alpha > 1

	if hit, ttMove, ttScore, ttDepth, ttBound := tt.Probe(hash); hit {
		ttHit = true
		hashMove = ttMove
		ttScoreRaw = ttScore
		ttDepthRaw = ttDepth
		ttBoundRaw = ttBound
		if !isPV && int8(depth) <= ttDepth && w.excludedMove == board.NullMove {
			score := scoreFromTT(ttScore, ply)
			switch ttBound {
			case BoundExact:
				return score, []board.Move{ttMove}
			case BoundLower:
				if score >= beta {
					return score, []board.Move{ttMove}
				}
			case BoundUpper:
				if score <= alpha {
					return score, nil
				}
			}
		}
	}

	// Internal iterative reductions: if we have no hash move to guide the
	// search, reduce depth by 1 so we fill the TT faster.
	if depth >= 4 && hashMove == board.NullMove {
		depth--
	}

	inCheck := movegen.IsSquareAttacked(&w.pos, w.pos.KingSquare(w.pos.SideToMove), w.pos.SideToMove.Other())

	// Null-move pruning (skip during singular extension search).
	if nullAllowed && !isPV && !inCheck && depth > 3 && w.excludedMove == board.NullMove {
		w.pos.MakeNullMove()
		nullScore, _ := w.negamax(depth-1-3, -beta, -beta+1, ply+1, false)
		nullScore = -nullScore
		w.pos.UnmakeNullMove()
		if nullScore >= beta {
			return beta, nil
		}
	}

	// Static eval for pruning decisions.
	staticEval := eval.Evaluate(&w.pos)

	// Reverse futility pruning: at shallow depths, if static eval is far
	// above beta, the position is so good that we can prune immediately.
	if !isPV && !inCheck && depth <= 7 && w.excludedMove == board.NullMove {
		margin := depth * 80
		if staticEval-margin >= beta {
			return staticEval, nil
		}
	}

	// Futility pruning: at shallow depths, if static eval + margin is far
	// below alpha, quiet moves are unlikely to raise it, so we can skip them.
	futile := false
	if !isPV && !inCheck && depth <= 2 {
		margin := depth * 150 // 150 cp per depth ply
		if staticEval+margin <= alpha {
			futile = true
		}
	}

	var ml board.MoveList
	movegen.GenerateLegalMoves(&w.pos, &ml)

	if ml.Count == 0 {
		if inCheck {
			return -MateScore + ply, nil // checkmate
		}
		return 0, nil // stalemate
	}

	OrderMoves(&ml, hashMove, w.killers[ply], &w.history, w.pos.SideToMove, &w.pos)

	// Late move pruning thresholds: maximum quiet move index per depth.
	var lmpThresholds = [4]int{0, 5, 8, 12}

	origAlpha := alpha
	bestScore := -Infinity
	bestMove := board.NullMove
	var bestPV []board.Move

	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]

		// Skip excluded move during singular extension verification search.
		if m == w.excludedMove {
			continue
		}

		// Futility pruning: skip quiet moves that are unlikely to raise alpha.
		if futile && bestScore > -MateScore+MaxDepth && !m.IsCapture() && !m.IsPromotion() {
			continue
		}

		// Late move pruning: at shallow depths, skip quiet moves that appear
		// late in the move ordering, as they are unlikely to improve alpha.
		if !isPV && !inCheck && depth <= 3 && bestScore > -MateScore+MaxDepth &&
			!m.IsCapture() && !m.IsPromotion() && i >= lmpThresholds[depth] {
			continue
		}

		w.pos.MakeMove(m)

		// Check extension: search deeper when the move gives check.
		extension := 0
		if movegen.IsSquareAttacked(&w.pos, w.pos.KingSquare(w.pos.SideToMove), w.pos.SideToMove.Other()) {
			extension = 1
		}

		// Singular extension: if the TT move is significantly better than
		// all alternatives, extend its search by 1 ply.
		if extension == 0 && depth >= 8 && m == hashMove && ttHit &&
			ttDepthRaw >= int8(depth-3) && ttBoundRaw != BoundUpper &&
			w.excludedMove == board.NullMove {
			sBeta := int(scoreFromTT(ttScoreRaw, ply)) - depth*2
			sDepth := (depth - 1) / 2

			w.pos.UnmakeMove(m)
			w.excludedMove = m
			seScore, _ := w.negamax(sDepth, sBeta-1, sBeta, ply, false)
			w.excludedMove = board.NullMove
			w.pos.MakeMove(m)

			if seScore < sBeta {
				extension = 1
			}
		}

		newDepth := depth - 1 + extension

		var score int
		var childPV []board.Move

		if i == 0 {
			// First move: search with full window.
			score, childPV = w.negamax(newDepth, -beta, -alpha, ply+1, true)
			score = -score
		} else {
			// Late Move Reduction: reduce depth for quiet late moves.
			reduction := 0
			if i >= 3 && depth >= 3 && !inCheck && !m.IsCapture() && !m.IsPromotion() {
				mi := i
				if mi > 63 {
					mi = 63
				}
				reduction = lmrReductions[depth][mi]
				if reduction < 1 {
					reduction = 1
				}
				// Don't reduce into negative depth.
				if reduction > newDepth {
					reduction = newDepth
				}
			}

			// PVS: zero-window search (possibly with LMR reduction).
			score, _ = w.negamax(newDepth-reduction, -alpha-1, -alpha, ply+1, true)
			score = -score

			// Re-search at full depth if reduced search beats alpha.
			if score > alpha && reduction > 0 {
				score, _ = w.negamax(newDepth, -alpha-1, -alpha, ply+1, true)
				score = -score
			}

			// Full window re-search if zero-window search found a better move.
			if score > alpha && score < beta {
				score, childPV = w.negamax(newDepth, -beta, -alpha, ply+1, true)
				score = -score
			}
		}

		w.pos.UnmakeMove(m)

		if w.shouldStop() && ply > 0 {
			return 0, nil
		}

		if score > bestScore {
			bestScore = score
			bestMove = m
		}

		if score > alpha {
			alpha = score
			bestPV = make([]board.Move, 1+len(childPV))
			bestPV[0] = m
			copy(bestPV[1:], childPV)
			if alpha >= beta {
				if !m.IsCapture() {
					w.storeKiller(m, ply)
					w.history[w.pos.SideToMove][m.From()][m.To()] += int32(depth * depth)
				}
				break
			}
		}
	}

	// TT store (skip during singular extension search — hash is modified).
	if !w.shouldStop() && w.excludedMove == board.NullMove {
		var bound Bound
		if bestScore >= beta {
			bound = BoundLower
		} else if bestScore > origAlpha {
			bound = BoundExact
		} else {
			bound = BoundUpper
		}
		tt.Store(hash, bestMove, scoreToTT(bestScore, ply), int8(depth), bound)
	}

	return alpha, bestPV
}

// storeKiller saves a quiet move that caused a beta cutoff.
func (w *worker) storeKiller(m board.Move, ply int) {
	if m != w.killers[ply][0] {
		w.killers[ply][1] = w.killers[ply][0]
		w.killers[ply][0] = m
	}
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
	OrderMoves(&ml, board.NullMove, [2]board.Move{}, nil, 0, nil)

	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]

		// SEE pruning: skip captures that lose material.
		if SEE(&w.pos, m) < 0 {
			continue
		}

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
