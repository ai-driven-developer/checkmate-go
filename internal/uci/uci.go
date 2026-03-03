package uci

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
	"checkmatego/internal/nnue"
	"checkmatego/internal/search"
)

const (
	EngineName   = "CheckmateGo"
	EngineAuthor = "ai-driven-developer"
	Version      = "1.4.0"
)

// Handler manages the UCI protocol.
type Handler struct {
	pos       *board.Position
	engine    *search.Engine
	options   Options
	input     io.Reader
	output    io.Writer
	searchDone chan struct{}
}

// NewHandler creates a UCI handler with default settings.
func NewHandler() *Handler {
	h := &Handler{
		pos:     board.NewPosition(),
		engine:  search.NewEngine(),
		options: DefaultOptions(),
		input:   os.Stdin,
		output:  os.Stdout,
	}
	h.applyOptions()
	return h
}

// NewHandlerWithIO creates a handler with custom I/O (for testing).
func NewHandlerWithIO(in io.Reader, out io.Writer) *Handler {
	h := NewHandler()
	h.input = in
	h.output = out
	return h
}

func (h *Handler) printf(format string, a ...interface{}) {
	fmt.Fprintf(h.output, format, a...)
}

// applyOptions propagates current option values to the engine.
func (h *Handler) applyOptions() {
	h.engine.SetMoveOverhead(time.Duration(h.options.MoveOverhead) * time.Millisecond)
	h.engine.SetThreads(h.options.Threads)
	h.engine.SetHash(h.options.Hash)
	h.loadNetwork()
}

// loadNetwork loads or clears the NNUE network based on current options.
func (h *Handler) loadNetwork() {
	if !h.options.UseNNUE || h.options.EvalFile == "" {
		h.engine.SetNetwork(nil)
		return
	}
	net, err := nnue.LoadNetwork(h.options.EvalFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "info string Failed to load NNUE: %v\n", err)
		h.engine.SetNetwork(nil)
		return
	}
	h.engine.SetNetwork(net)
}

// Run reads and processes UCI commands until "quit".
func (h *Handler) Run() {
	scanner := bufio.NewScanner(h.input)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		quit := h.ProcessCommand(line)
		if quit {
			return
		}
	}
}

// ProcessCommand handles a single UCI command. Returns true on "quit".
func (h *Handler) ProcessCommand(line string) bool {
	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return false
	}

	switch tokens[0] {
	case "uci":
		h.cmdUCI()
	case "isready":
		h.cmdIsReady()
	case "ucinewgame":
		h.cmdNewGame()
	case "position":
		h.cmdPosition(tokens[1:])
	case "go":
		h.cmdGo(tokens[1:])
	case "stop":
		h.cmdStop()
	case "quit":
		h.cmdStop()
		return true
	case "setoption":
		h.cmdSetOption(tokens[1:])
	case "d", "display":
		h.cmdDisplay()
	case "perft":
		h.cmdPerft(tokens[1:])
	}
	return false
}

func (h *Handler) cmdUCI() {
	h.printf("id name %s %s\n", EngineName, Version)
	h.printf("id author %s\n", EngineAuthor)
	h.options.PrintOptions(h.printf)
	h.printf("uciok\n")
}

func (h *Handler) cmdIsReady() {
	h.printf("readyok\n")
}

func (h *Handler) cmdNewGame() {
	h.pos = board.NewPosition()
	h.engine = search.NewEngine()
	h.applyOptions()
}

func (h *Handler) cmdPosition(tokens []string) {
	if len(tokens) == 0 {
		return
	}

	idx := 0
	if tokens[0] == "startpos" {
		h.pos = board.NewPosition()
		idx = 1
	} else if tokens[0] == "fen" {
		if len(tokens) < 5 {
			return
		}
		// Collect FEN fields (up to 6).
		fenEnd := 5
		if len(tokens) > 5 && tokens[5] != "moves" {
			fenEnd = 6
			if len(tokens) > 6 && tokens[6] != "moves" {
				fenEnd = 7
			}
		}
		if fenEnd > len(tokens) {
			fenEnd = len(tokens)
		}
		fen := strings.Join(tokens[1:fenEnd], " ")
		h.pos = &board.Position{}
		if err := h.pos.SetFromFEN(fen); err != nil {
			return
		}
		idx = fenEnd
	} else {
		return
	}

	// Apply moves.
	if idx < len(tokens) && tokens[idx] == "moves" {
		for _, moveStr := range tokens[idx+1:] {
			m := parseMoveUCI(h.pos, moveStr)
			if m != board.NullMove {
				h.pos.MakeMove(m)
			}
		}
	}
}

func (h *Handler) cmdGo(tokens []string) {
	limits := search.SearchLimits{}
	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "depth":
			if i+1 < len(tokens) {
				limits.Depth, _ = strconv.Atoi(tokens[i+1])
				i++
			}
		case "movetime":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				limits.MoveTime = time.Duration(ms) * time.Millisecond
				i++
			}
		case "wtime":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				limits.WTime = time.Duration(ms) * time.Millisecond
				i++
			}
		case "btime":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				limits.BTime = time.Duration(ms) * time.Millisecond
				i++
			}
		case "winc":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				limits.WInc = time.Duration(ms) * time.Millisecond
				i++
			}
		case "binc":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				limits.BInc = time.Duration(ms) * time.Millisecond
				i++
			}
		case "movestogo":
			if i+1 < len(tokens) {
				limits.MovesToGo, _ = strconv.Atoi(tokens[i+1])
				i++
			}
		case "nodes":
			if i+1 < len(tokens) {
				limits.Nodes, _ = strconv.ParseUint(tokens[i+1], 10, 64)
				i++
			}
		case "infinite":
			limits.Infinite = true
		case "perft":
			if i+1 < len(tokens) {
				depth, _ := strconv.Atoi(tokens[i+1])
				h.runPerft(depth)
				return
			}
		}
	}

	// Set up info callback.
	h.engine.SetInfoCallback(func(info search.SearchInfo) {
		h.printInfo(info)
	})

	// Run search in goroutine.
	done := make(chan struct{})
	h.searchDone = done
	go func() {
		posCopy := h.pos.Copy()
		bestMove := h.engine.Search(posCopy, limits)
		h.printf("bestmove %s\n", bestMove)
		close(done)
	}()
}

func (h *Handler) cmdStop() {
	h.engine.Stop()
	if h.searchDone != nil {
		<-h.searchDone
		h.searchDone = nil
	}
}

func (h *Handler) cmdSetOption(tokens []string) {
	// Format: name <name> [value <value>]
	nameIdx := -1
	valueIdx := -1
	for i, t := range tokens {
		if t == "name" {
			nameIdx = i + 1
		}
		if t == "value" {
			valueIdx = i + 1
		}
	}
	if nameIdx < 0 || nameIdx >= len(tokens) {
		return
	}
	// Name is everything between "name" and "value" (or end).
	nameEnd := len(tokens)
	if valueIdx > 0 {
		nameEnd = valueIdx - 1
	}
	name := strings.Join(tokens[nameIdx:nameEnd], " ")
	value := ""
	if valueIdx >= 0 && valueIdx < len(tokens) {
		value = strings.Join(tokens[valueIdx:], " ")
	}
	_ = h.options.SetOption(name, value)
	h.applyOptions()
}

func (h *Handler) cmdDisplay() {
	h.printf("%s", h.pos.String())
	h.printf("FEN: %s\n", h.pos.FEN())
	h.printf("Hash: %016x\n", h.pos.Hash)
}

func (h *Handler) cmdPerft(tokens []string) {
	if len(tokens) == 0 {
		return
	}
	depth, err := strconv.Atoi(tokens[0])
	if err != nil || depth < 1 {
		return
	}
	h.runPerft(depth)
}

func (h *Handler) runPerft(depth int) {
	start := time.Now()
	var ml board.MoveList
	movegen.GenerateLegalMoves(h.pos, &ml)
	var total uint64
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		h.pos.MakeMove(m)
		nodes := movegen.Perft(h.pos, depth-1)
		h.pos.UnmakeMove(m)
		h.printf("%s: %d\n", m, nodes)
		total += nodes
	}
	elapsed := time.Since(start)
	nps := uint64(0)
	if elapsed.Milliseconds() > 0 {
		nps = total * 1000 / uint64(elapsed.Milliseconds())
	}
	h.printf("\nTotal: %d nodes in %v (%d nps)\n", total, elapsed, nps)
}

func (h *Handler) printInfo(info search.SearchInfo) {
	ms := info.Time.Milliseconds()
	if ms == 0 {
		ms = 1
	}
	nps := info.Nodes * 1000 / uint64(ms)

	pvStr := ""
	for i, m := range info.PV {
		if i > 0 {
			pvStr += " "
		}
		pvStr += m.String()
	}

	scoreStr := fmt.Sprintf("cp %d", info.Score)
	if info.Score > search.MateScore-search.MaxDepth {
		mateMoves := (search.MateScore - info.Score + 1) / 2
		scoreStr = fmt.Sprintf("mate %d", mateMoves)
	} else if info.Score < -search.MateScore+search.MaxDepth {
		mateMoves := -(search.MateScore + info.Score + 1) / 2
		scoreStr = fmt.Sprintf("mate %d", mateMoves)
	}

	wdlStr := ""
	if h.options.ShowWDL {
		w, d, l := scoreToWDL(info.Score)
		wdlStr = fmt.Sprintf(" wdl %d %d %d", w, d, l)
	}

	h.printf("info depth %d score %s%s nodes %d nps %d time %d hashfull %d pv %s\n",
		info.Depth, scoreStr, wdlStr, info.Nodes, nps, info.Time.Milliseconds(), info.Hashfull, pvStr)
}

// scoreToWDL converts a centipawn score to Win/Draw/Loss per mille values.
// Uses a logistic model with a draw margin so positions near equality
// have a meaningful draw probability.
func scoreToWDL(score int) (w, d, l int) {
	// Mate scores.
	if score > search.MateScore-search.MaxDepth {
		return 1000, 0, 0
	}
	if score < -search.MateScore+search.MaxDepth {
		return 0, 0, 1000
	}

	const (
		scale     = 100.0 // steepness of the logistic curve
		drawWidth = 100.0 // centipawn half-width of the draw region
	)
	cp := float64(score)
	pWin := 1.0 / (1.0 + math.Exp(-(cp-drawWidth)/scale))
	pLoss := 1.0 / (1.0 + math.Exp((cp+drawWidth)/scale))
	pDraw := 1.0 - pWin - pLoss

	w = int(math.Round(pWin * 1000))
	d = int(math.Round(pDraw * 1000))
	l = 1000 - w - d // ensure sum is exactly 1000
	return
}

// parseMoveUCI converts a UCI move string (e.g. "e2e4", "e7e8q") to a Move.
func parseMoveUCI(pos *board.Position, s string) board.Move {
	if len(s) < 4 {
		return board.NullMove
	}
	from := board.SquareFromString(s[0:2])
	to := board.SquareFromString(s[2:4])
	if from == board.NoSquare || to == board.NoSquare {
		return board.NullMove
	}

	var promoChar byte
	if len(s) == 5 {
		promoChar = s[4]
	}

	// Find the matching legal move.
	var ml board.MoveList
	movegen.GenerateLegalMoves(pos, &ml)
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		if m.From() == from && m.To() == to {
			if promoChar == 0 && !m.IsPromotion() {
				return m
			}
			if promoChar != 0 && m.IsPromotion() {
				pp := m.PromotionPiece()
				match := (promoChar == 'q' && pp == board.Queen) ||
					(promoChar == 'r' && pp == board.Rook) ||
					(promoChar == 'b' && pp == board.Bishop) ||
					(promoChar == 'n' && pp == board.Knight)
				if match {
					return m
				}
			}
		}
	}
	return board.NullMove
}
