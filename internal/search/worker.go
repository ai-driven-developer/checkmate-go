package search

import (
	"checkmatego/internal/board"
	"checkmatego/internal/eval"
	"checkmatego/internal/movegen"
	"checkmatego/internal/nnue"
	"time"
)

// worker is a per-thread search context for Lazy SMP.
type worker struct {
	engine       *Engine        // shared state (nodes, stopFlag, deadline, limits, start, tt)
	pos          board.Position // per-worker position copy
	id           int
	killers      [MaxDepth][2]board.Move // killer moves per ply
	history      [2][64][64]int32        // history heuristic [color][from][to]
	countermoves [7][64]board.Move       // countermove heuristic [prevPiece][prevTo]
	contHist     [2][7][64][7][64]int32  // continuation history [depth_offset][prevPc][prevTo][currPc][currTo]
	moveStack    [MaxDepth]board.Move    // move that brought us to each ply (for cont hist lookups)
	excludedMove board.Move              // move to skip during singular extension search
	staticEvals  [MaxDepth]int           // static eval per ply for improving detection
	pawnCache    *eval.PawnCache         // per-worker pawn structure cache
	net          *nnue.Network           // NNUE network (nil = use HCE)
	accStack     *nnue.AccumulatorStack  // NNUE accumulator stack

	// Triangular PV table: pvTable[ply] holds the PV starting at that ply.
	// Eliminates all heap allocations for PV construction in the search loop.
	pvTable  [MaxDepth][MaxDepth]board.Move
	pvLength [MaxDepth]int
}

// workerResult holds the outcome of a worker's search.
type workerResult struct {
	move  board.Move
	score int
	depth int
}

// getPV returns the root PV as a slice (allocates once for info reporting).
func (w *worker) getPV() []board.Move {
	n := w.pvLength[0]
	if n <= 0 {
		return nil
	}
	pv := make([]board.Move, n)
	copy(pv, w.pvTable[0][:n])
	return pv
}

// makeMove wraps pos.MakeMove with NNUE accumulator updates.
func (w *worker) makeMove(m board.Move) {
	if w.accStack != nil {
		w.accStack.MakeMove(&w.pos, m)
	}
	w.pos.MakeMove(m)
}

// unmakeMove wraps pos.UnmakeMove with NNUE accumulator pop.
func (w *worker) unmakeMove(m board.Move) {
	w.pos.UnmakeMove(m)
	if w.accStack != nil {
		w.accStack.UnmakeMove()
	}
}

// makeNullMove wraps pos.MakeNullMove with NNUE accumulator push.
func (w *worker) makeNullMove() {
	if w.accStack != nil {
		w.accStack.MakeNullMove()
	}
	w.pos.MakeNullMove()
}

// unmakeNullMove wraps pos.UnmakeNullMove with NNUE accumulator pop.
func (w *worker) unmakeNullMove() {
	w.pos.UnmakeNullMove()
	if w.accStack != nil {
		w.accStack.UnmakeNullMove()
	}
}

// evaluate returns the position score using NNUE or HCE.
func (w *worker) evaluate() int {
	if w.net != nil {
		return w.net.Evaluate(w.accStack.Current(), w.pos.SideToMove)
	}
	return eval.EvaluateWithCache(&w.pos, w.pawnCache)
}

func (w *worker) search(maxDepth int) workerResult {
	result := workerResult{move: board.NullMove}

	w.history = [2][64][64]int32{}
	w.countermoves = [7][64]board.Move{}
	w.contHist = [2][7][64][7][64]int32{}

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

		for {
			score = w.negamax(depth, alpha, beta, 0, true, board.NullMove)
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

		if w.pvLength[0] > 0 {
			result.move = w.pvTable[0][0]
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
				PV:       w.getPV(),
				Hashfull: w.engine.tt.Hashfull(),
			})
		}
		// Stop if we found a forced mate.
		if score > MateScore-MaxDepth || score < -MateScore+MaxDepth {
			break
		}
		// Main thread: check soft time limit with stability/score-drop adjustments.
		if w.id == 0 && !w.engine.limits.Infinite && w.engine.limits.Depth == 0 &&
			result.move != board.NullMove {
			if w.engine.tm.shouldStopSoft(result.move, score, depth) {
				w.engine.stopFlag.Store(true)
				break
			}
		}
	}
	return result
}

func (w *worker) shouldStop() bool {
	if w.engine.stopFlag.Load() {
		return true
	}
	if w.engine.limits.Nodes > 0 && w.engine.nodes.Load() >= w.engine.limits.Nodes {
		return true
	}
	if !w.engine.limits.Infinite && w.engine.limits.Depth == 0 {
		return w.engine.tm.shouldStopHard()
	}
	return false
}

func (w *worker) negamax(depth, alpha, beta, ply int, nullAllowed bool, prevMove board.Move) int {
	// Track the move that brought us to this ply (for continuation history).
	w.moveStack[ply] = prevMove

	// Check time every 4096 nodes.
	if w.engine.nodes.Load()&4095 == 0 && ply > 0 {
		if w.shouldStop() {
			return 0
		}
	}

	// Initialize PV length for this ply.
	w.pvLength[ply] = 0

	if depth <= 0 {
		return w.quiesce(alpha, beta, ply)
	}

	w.engine.nodes.Add(1)

	// 50-move rule.
	if w.pos.HalfMoveClock >= 100 {
		return 0
	}

	// Repetition detection (2-fold: draw if position seen before).
	if ply > 0 && w.pos.IsRepetition() {
		return 0
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
				w.pvTable[ply][0] = ttMove
				w.pvLength[ply] = 1
				return score
			case BoundLower:
				if score >= beta {
					return score
				}
			case BoundUpper:
				if score <= alpha {
					return score
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

	// Static eval for pruning decisions.
	staticEval := w.evaluate()

	// Track static eval per ply for improving detection.
	if inCheck {
		w.staticEvals[ply] = -Infinity
	} else {
		w.staticEvals[ply] = staticEval
	}

	// Improving flag: position eval is better than 2 plies ago (same side).
	improving := !inCheck && ply >= 2 && w.staticEvals[ply-2] != -Infinity &&
		staticEval > w.staticEvals[ply-2]

	// Null-move pruning (skip during singular extension search).
	if nullAllowed && !isPV && !inCheck && depth > 3 && w.excludedMove == board.NullMove {
		R := 3
		if !improving {
			R++
		}
		w.makeNullMove()
		nullScore := -w.negamax(depth-1-R, -beta, -beta+1, ply+1, false, board.NullMove)
		w.unmakeNullMove()
		if nullScore >= beta {
			return beta
		}
	}

	// ProbCut: at moderate depths, if a shallow capture search with a
	// raised beta finds a score >= beta + margin, the full-depth search
	// is unlikely to score below beta — prune immediately.
	if !isPV && !inCheck && depth >= 5 && w.excludedMove == board.NullMove &&
		beta > -MateScore+MaxDepth && beta < MateScore-MaxDepth {
		rBeta := beta + probCutMargin

		var pcML board.MoveList
		movegen.GenerateCaptures(&w.pos, &pcML)

		var pcScores [256]int32
		ScoreMoves(&pcML, &pcScores, hashMove, [2]board.Move{}, board.NullMove, nil, [2]*[7][64]int32{}, 0, nil)

		for j := 0; j < pcML.Count; j++ {
			PickBest(&pcML, &pcScores, j)
			pcMove := pcML.Moves[j]

			if SEE(&w.pos, pcMove) < 0 {
				continue
			}

			w.makeMove(pcMove)
			// Shallow search at reduced depth with raised beta window.
			pcScore := -w.negamax(depth-4, -rBeta, -rBeta+1, ply+1, false, pcMove)
			w.unmakeMove(pcMove)

			if pcScore >= rBeta {
				return pcScore
			}
		}
	}

	// Reverse futility pruning: at shallow depths, if static eval is far
	// above beta, the position is so good that we can prune immediately.
	if !isPV && !inCheck && depth <= 7 && w.excludedMove == board.NullMove {
		margin := depth * 80
		if improving {
			margin = depth * 60
		}
		if staticEval-margin >= beta {
			return staticEval
		}
	}

	// Razoring: at shallow depths, if static eval is significantly below
	// alpha, verify with quiescence search. If qsearch confirms, prune.
	if !isPV && !inCheck && depth <= 3 && w.excludedMove == board.NullMove {
		if staticEval+depth*razoringMargin <= alpha {
			razorScore := w.quiesce(alpha, beta, ply)
			if razorScore <= alpha {
				return razorScore
			}
		}
	}

	// Futility pruning: at shallow depths, if static eval + margin is far
	// below alpha, quiet moves are unlikely to raise it, so we can skip them.
	futile := false
	if !isPV && !inCheck && depth <= 2 {
		margin := depth * 150 // 150 cp per depth ply
		if !improving {
			margin = depth * 120
		}
		if staticEval+margin <= alpha {
			futile = true
		}
	}

	var ml board.MoveList
	movegen.GenerateLegalMoves(&w.pos, &ml)

	if ml.Count == 0 {
		if inCheck {
			return -MateScore + ply // checkmate
		}
		return 0 // stalemate
	}

	// Look up countermove for the opponent's previous move.
	var countermove board.Move
	if prevMove != board.NullMove {
		countermove = w.countermoves[prevMove.Piece()][prevMove.To()]
	}

	// Compute continuation history pointers for move scoring.
	var ch [2]*[7][64]int32
	if prevMove != board.NullMove {
		ch[0] = &w.contHist[0][prevMove.Piece()][prevMove.To()]
	}
	if ply >= 1 && w.moveStack[ply-1] != board.NullMove {
		ch[1] = &w.contHist[1][w.moveStack[ply-1].Piece()][w.moveStack[ply-1].To()]
	}

	// Score moves for lazy ordering (pick-best on each iteration).
	var scores [256]int32
	ScoreMoves(&ml, &scores, hashMove, w.killers[ply], countermove, &w.history, ch, w.pos.SideToMove, &w.pos)

	// Late move pruning thresholds: maximum quiet move index per depth.
	var lmpThresholds [4]int
	if improving {
		lmpThresholds = [4]int{0, 6, 10, 15}
	} else {
		lmpThresholds = [4]int{0, 4, 6, 9}
	}

	origAlpha := alpha
	bestScore := -Infinity
	bestMove := board.NullMove

	for i := 0; i < ml.Count; i++ {
		// Lazy move ordering: select the best remaining move.
		PickBest(&ml, &scores, i)
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

		// SEE pruning: skip moves that lose material according to SEE.
		// Quiet moves use a negative threshold (scaled by depth); captures must
		// not lose material at all. Skipped for the hash move (i==0 after pick-best
		// when hash move is present) and promotions.
		if !isPV && !inCheck && bestScore > -MateScore+MaxDepth && !m.IsPromotion() {
			if !m.IsCapture() && depth <= 8 && SEE(&w.pos, m) < -depth*80 {
				continue
			}
			if m.IsCapture() && depth <= 6 && SEE(&w.pos, m) < -depth*20 {
				continue
			}
		}

		w.makeMove(m)

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

			w.unmakeMove(m)
			w.excludedMove = m
			seScore := w.negamax(sDepth, sBeta-1, sBeta, ply, false, prevMove)
			w.excludedMove = board.NullMove
			w.makeMove(m)

			if seScore < sBeta {
				extension = 1
			}
		}

		newDepth := depth - 1 + extension

		var score int

		if i == 0 {
			// First move: search with full window.
			score = -w.negamax(newDepth, -beta, -alpha, ply+1, true, m)
		} else {
			// Late Move Reduction: reduce depth for quiet late moves.
			reduction := 0
			if i >= 3 && depth >= 3 && !inCheck && !m.IsCapture() && !m.IsPromotion() {
				mi := i
				if mi > 63 {
					mi = 63
				}
				reduction = lmrReductions[depth][mi]
				if !improving {
					reduction++
				}
				// History-aware LMR: reduce more for moves with bad history,
				// reduce less for moves with good history.
				// Combine main history with continuation history scores.
				hist := w.history[w.pos.SideToMove][m.From()][m.To()]
				for _, chp := range ch {
					if chp != nil {
						hist += chp[m.Piece()][m.To()]
					}
				}
				if hist < -2048 {
					reduction++
				} else if hist > 2048 {
					reduction--
				}
				if reduction < 1 {
					reduction = 1
				}
				// Don't reduce into negative depth.
				if reduction > newDepth {
					reduction = newDepth
				}
			}

			// PVS: zero-window search (possibly with LMR reduction).
			score = -w.negamax(newDepth-reduction, -alpha-1, -alpha, ply+1, true, m)

			// Re-search at full depth if reduced search beats alpha.
			if score > alpha && reduction > 0 {
				score = -w.negamax(newDepth, -alpha-1, -alpha, ply+1, true, m)
			}

			// Full window re-search if zero-window search found a better move.
			if score > alpha && score < beta {
				score = -w.negamax(newDepth, -beta, -alpha, ply+1, true, m)
			}
		}

		w.unmakeMove(m)

		if w.shouldStop() && ply > 0 {
			return 0
		}

		if score > bestScore {
			bestScore = score
			bestMove = m
		}

		if score > alpha {
			alpha = score

			// Update PV: current move + child PV.
			w.pvTable[ply][0] = m
			childLen := w.pvLength[ply+1]
			copy(w.pvTable[ply][1:1+childLen], w.pvTable[ply+1][:childLen])
			w.pvLength[ply] = 1 + childLen

			if alpha >= beta {
				if !m.IsCapture() {
					w.storeKiller(m, ply)
					bonus := int32(depth * depth)
					// History gravity: reward the cutoff move and penalize
					// all previously searched quiet moves that failed to
					// produce a cutoff.
					w.updateHistory(w.pos.SideToMove, m.From(), m.To(), bonus)
					w.updateContHist(ply, m, bonus)
					for j := 0; j < i; j++ {
						prev := ml.Moves[j]
						if !prev.IsCapture() && prev != w.excludedMove {
							w.updateHistory(w.pos.SideToMove, prev.From(), prev.To(), -bonus)
							w.updateContHist(ply, prev, -bonus)
						}
					}
					if prevMove != board.NullMove {
						w.countermoves[prevMove.Piece()][prevMove.To()] = m
					}
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

	return alpha
}

// updateHistory applies a gravity-based update to the history table.
// The formula h += bonus - h*|bonus|/maxHistory ensures values stay bounded
// in [-maxHistory, maxHistory] and converge toward maxHistory for consistently
// good moves while decaying stale entries naturally.
func (w *worker) updateHistory(color board.Color, from, to board.Square, bonus int32) {
	entry := &w.history[color][from][to]
	abs := bonus
	if abs < 0 {
		abs = -abs
	}
	*entry += bonus - *entry*abs/maxHistory
}

// updateContHist applies a gravity-based update to continuation history tables.
// It updates both 1-ply (opponent's previous move) and 2-ply (our previous move)
// tables, correlating (prevPiece, prevTo) with (currPiece, currTo).
func (w *worker) updateContHist(ply int, m board.Move, bonus int32) {
	piece := m.Piece()
	to := m.To()
	abs := bonus
	if abs < 0 {
		abs = -abs
	}
	// 1-ply back: correlation with opponent's last move.
	if prev := w.moveStack[ply]; prev != board.NullMove {
		entry := &w.contHist[0][prev.Piece()][prev.To()][piece][to]
		*entry += bonus - *entry*abs/maxHistory
	}
	// 2-ply back: correlation with our own last move.
	if ply >= 1 {
		if prev2 := w.moveStack[ply-1]; prev2 != board.NullMove {
			entry := &w.contHist[1][prev2.Piece()][prev2.To()][piece][to]
			*entry += bonus - *entry*abs/maxHistory
		}
	}
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

	standPat := w.evaluate()
	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	// Big delta pruning: if even capturing a queen cannot raise alpha,
	// this position is hopeless — prune immediately.
	if standPat+deltaMargin < alpha {
		return alpha
	}

	var ml board.MoveList
	movegen.GenerateCaptures(&w.pos, &ml)

	// Score captures for lazy ordering.
	var scores [256]int32
	ScoreMoves(&ml, &scores, board.NullMove, [2]board.Move{}, board.NullMove, nil, [2]*[7][64]int32{}, 0, nil)

	for i := 0; i < ml.Count; i++ {
		// Lazy move ordering: select the best remaining capture.
		PickBest(&ml, &scores, i)
		m := ml.Moves[i]

		// Per-move delta pruning: if this capture's potential gain
		// cannot raise alpha, skip it without making the move.
		if !m.IsPromotion() {
			captureVal := seeValue[m.CapturedPiece()]
			if standPat+captureVal+deltaPruningMargin < alpha {
				continue
			}
		}

		// SEE pruning: skip captures that lose material.
		if SEE(&w.pos, m) < 0 {
			continue
		}

		w.makeMove(m)
		score := -w.quiesce(-beta, -alpha, ply+1)
		w.unmakeMove(m)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}
	return alpha
}
