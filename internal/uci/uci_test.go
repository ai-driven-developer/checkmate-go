package uci

import (
	"bytes"
	"checkmatego/internal/board"
	"strings"
	"testing"
	"time"
)

func newTestHandler() (*Handler, *bytes.Buffer) {
	var buf bytes.Buffer
	h := NewHandlerWithIO(strings.NewReader(""), &buf)
	return h, &buf
}

func TestUCICommand(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("uci")
	out := buf.String()

	if !strings.Contains(out, "id name CheckmateGo") {
		t.Error("uci response should contain engine name")
	}
	if !strings.Contains(out, "id author") {
		t.Error("uci response should contain author")
	}
	if !strings.Contains(out, "uciok") {
		t.Error("uci response should end with uciok")
	}
}

func TestIsReadyCommand(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("isready")
	if !strings.Contains(buf.String(), "readyok") {
		t.Error("isready should respond with readyok")
	}
}

func TestPositionStartpos(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("position startpos")

	fen := h.pos.FEN()
	if fen != "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" {
		t.Errorf("unexpected FEN after startpos: %s", fen)
	}
}

func TestPositionStartposWithMoves(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("position startpos moves e2e4 e7e5")

	fen := h.pos.FEN()
	expected := "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2"
	if fen != expected {
		t.Errorf("unexpected FEN:\n  got:  %s\n  want: %s", fen, expected)
	}
}

func TestPositionFEN(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("position fen r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	fen := h.pos.FEN()
	expected := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	if fen != expected {
		t.Errorf("unexpected FEN:\n  got:  %s\n  want: %s", fen, expected)
	}
}

func TestGoDepth(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("position startpos")
	h.ProcessCommand("go depth 2")

	// Wait for search to complete.
	h.ProcessCommand("stop")

	out := buf.String()
	if !strings.Contains(out, "bestmove") {
		t.Error("go depth should produce bestmove output")
	}
	if !strings.Contains(out, "info depth") {
		t.Error("go depth should produce info output")
	}
}

func TestGoMoveTime(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("position startpos")
	h.ProcessCommand("go movetime 500")

	// Wait for search to complete.
	time.Sleep(600 * time.Millisecond)
	h.ProcessCommand("stop")

	out := buf.String()
	if !strings.Contains(out, "bestmove") {
		t.Error("go movetime should produce bestmove")
	}
}

func TestQuitCommand(t *testing.T) {
	h, _ := newTestHandler()
	quit := h.ProcessCommand("quit")
	if !quit {
		t.Error("quit should return true")
	}
}

func TestDisplayCommand(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("d")
	out := buf.String()
	if !strings.Contains(out, "FEN:") {
		t.Error("display should show FEN")
	}
}

func TestSetOption(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("setoption name Hash value 128")
	if h.options.Hash != 128 {
		t.Errorf("expected Hash=128, got %d", h.options.Hash)
	}
}

func TestNewGame(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("position startpos moves e2e4")
	h.ProcessCommand("ucinewgame")
	fen := h.pos.FEN()
	expected := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	if fen != expected {
		t.Errorf("ucinewgame should reset position: got %s", fen)
	}
}

func TestParseMoveUCI(t *testing.T) {
	pos := newTestPosition()
	tests := []struct {
		move string
		from string
		to   string
	}{
		{"e2e4", "e2", "e4"},
		{"g1f3", "g1", "f3"},
		{"b1c3", "b1", "c3"},
	}
	for _, tt := range tests {
		m := parseMoveUCI(pos, tt.move)
		if m.From().String() != tt.from || m.To().String() != tt.to {
			t.Errorf("parseMoveUCI(%s): got %s%s, want %s%s",
				tt.move, m.From(), m.To(), tt.from, tt.to)
		}
	}
}

func newTestPosition() *board.Position {
	return board.NewPosition()
}

// --- SetOption tests ---

func TestSetOptionThreads(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("setoption name Threads value 4")
	if h.options.Threads != 4 {
		t.Errorf("expected Threads=4, got %d", h.options.Threads)
	}
}

func TestSetOptionMoveOverhead(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("setoption name Move Overhead value 100")
	if h.options.MoveOverhead != 100 {
		t.Errorf("expected MoveOverhead=100, got %d", h.options.MoveOverhead)
	}
}

func TestSetOptionSyzygyPath(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("setoption name SyzygyPath value /path/to/syzygy")
	if h.options.SyzygyPath != "/path/to/syzygy" {
		t.Errorf("expected SyzygyPath=/path/to/syzygy, got %s", h.options.SyzygyPath)
	}
}

func TestSetOptionShowWDL(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("setoption name UCI_ShowWDL value true")
	if !h.options.ShowWDL {
		t.Error("expected ShowWDL=true")
	}
}

func TestSetOptionHashOutOfRange(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("setoption name Hash value 9999")
	// Should remain at default because 9999 > 4096.
	if h.options.Hash != 64 {
		t.Errorf("expected Hash to remain 64 (out of range), got %d", h.options.Hash)
	}
}

func TestSetOptionThreadsOutOfRange(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("setoption name Threads value 0")
	// Should remain at default because 0 < 1.
	if h.options.Threads != 1 {
		t.Errorf("expected Threads to remain 1 (out of range), got %d", h.options.Threads)
	}
}

func TestSetOptionUnknown(t *testing.T) {
	h, _ := newTestHandler()
	// Unknown options should be silently ignored.
	h.ProcessCommand("setoption name FooBar value 42")
	// No crash is the test.
}

// --- cmdGo parsing tests ---

func TestGoInfinite(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("position startpos")
	h.ProcessCommand("go infinite")
	// Let it run briefly, then stop.
	time.Sleep(100 * time.Millisecond)
	h.ProcessCommand("stop")

	out := buf.String()
	if !strings.Contains(out, "bestmove") {
		t.Error("go infinite + stop should produce bestmove")
	}
}

func TestGoWithTimeControl(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("position startpos")
	h.ProcessCommand("go wtime 10000 btime 10000 winc 100 binc 100")
	// Wait for search to complete (should be quick with time control).
	time.Sleep(2 * time.Second)
	h.ProcessCommand("stop")

	out := buf.String()
	if !strings.Contains(out, "bestmove") {
		t.Error("go with time control should produce bestmove")
	}
}

func TestGoMovesToGo(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("position startpos")
	h.ProcessCommand("go wtime 30000 btime 30000 movestogo 10")
	time.Sleep(2 * time.Second)
	h.ProcessCommand("stop")

	out := buf.String()
	if !strings.Contains(out, "bestmove") {
		t.Error("go with movestogo should produce bestmove")
	}
}

// --- scoreToWDL tests ---

func TestScoreToWDLMateWhite(t *testing.T) {
	w, d, l := scoreToWDL(29000)
	if w != 1000 || d != 0 || l != 0 {
		t.Errorf("mate score WDL: got w=%d d=%d l=%d, want 1000/0/0", w, d, l)
	}
}

func TestScoreToWDLMateBlack(t *testing.T) {
	w, d, l := scoreToWDL(-29000)
	if w != 0 || d != 0 || l != 1000 {
		t.Errorf("losing mate WDL: got w=%d d=%d l=%d, want 0/0/1000", w, d, l)
	}
}

func TestScoreToWDLEqual(t *testing.T) {
	w, d, l := scoreToWDL(0)
	// At score 0, draw probability should be dominant.
	if d < 400 {
		t.Errorf("equal position should have high draw prob, got d=%d", d)
	}
	if w+d+l != 1000 {
		t.Errorf("WDL should sum to 1000, got %d", w+d+l)
	}
}

func TestScoreToWDLWinning(t *testing.T) {
	w, d, l := scoreToWDL(500)
	// +5 pawns: win probability should be very high.
	if w < 800 {
		t.Errorf("+500cp should give high win prob, got w=%d", w)
	}
	if w+d+l != 1000 {
		t.Errorf("WDL should sum to 1000, got %d", w+d+l)
	}
	_ = l
}

func TestScoreToWDLLosing(t *testing.T) {
	w, d, l := scoreToWDL(-500)
	// -5 pawns: loss probability should be very high.
	if l < 800 {
		t.Errorf("-500cp should give high loss prob, got l=%d", l)
	}
	if w+d+l != 1000 {
		t.Errorf("WDL should sum to 1000, got %d", w+d+l)
	}
	_ = w
}

func TestScoreToWDLSymmetry(t *testing.T) {
	w1, d1, l1 := scoreToWDL(200)
	w2, d2, l2 := scoreToWDL(-200)
	if w1 != l2 || l1 != w2 || d1 != d2 {
		t.Errorf("WDL should be symmetric: +200 = (%d,%d,%d), -200 = (%d,%d,%d)",
			w1, d1, l1, w2, d2, l2)
	}
}

// --- parseMoveUCI tests ---

func TestParseMoveUCIPromotion(t *testing.T) {
	// Pawn on e7 promotes to queen on e8.
	pos := &board.Position{}
	_ = pos.SetFromFEN("2k5/4P3/8/8/8/8/8/4K3 w - - 0 1")

	m := parseMoveUCI(pos, "e7e8q")
	if m == board.NullMove {
		t.Fatal("expected valid promotion move")
	}
	if !m.IsPromotion() {
		t.Error("expected promotion flag")
	}
	if m.PromotionPiece() != board.Queen {
		t.Errorf("expected queen promotion, got %d", m.PromotionPiece())
	}
}

func TestParseMoveUCIPromotionKnight(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("2k5/4P3/8/8/8/8/8/4K3 w - - 0 1")

	m := parseMoveUCI(pos, "e7e8n")
	if m == board.NullMove {
		t.Fatal("expected valid knight promotion move")
	}
	if m.PromotionPiece() != board.Knight {
		t.Errorf("expected knight promotion, got %d", m.PromotionPiece())
	}
}

func TestParseMoveUCIInvalid(t *testing.T) {
	pos := board.NewPosition()

	tests := []string{"", "xx", "zz99", "e2e9"}
	for _, s := range tests {
		m := parseMoveUCI(pos, s)
		if m != board.NullMove {
			t.Errorf("parseMoveUCI(%q) should return NullMove, got %v", s, m)
		}
	}
}

func TestParseMoveUCICastle(t *testing.T) {
	pos := &board.Position{}
	_ = pos.SetFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")

	// Kingside castle.
	m := parseMoveUCI(pos, "e1g1")
	if m == board.NullMove {
		t.Fatal("expected valid castle move")
	}
	if !m.IsCastle() {
		t.Error("expected castle flag")
	}
}

// --- ShowWDL integration test ---

func TestGoWithWDLOutput(t *testing.T) {
	h, buf := newTestHandler()
	h.ProcessCommand("setoption name UCI_ShowWDL value true")
	h.ProcessCommand("position startpos")
	h.ProcessCommand("go depth 2")
	h.ProcessCommand("stop")

	out := buf.String()
	if !strings.Contains(out, "wdl") {
		t.Error("with UCI_ShowWDL enabled, info output should contain wdl")
	}
}

// --- Position FEN with moves ---

func TestPositionFENWithMoves(t *testing.T) {
	h, _ := newTestHandler()
	h.ProcessCommand("position fen rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1 moves e7e5")

	fen := h.pos.FEN()
	expected := "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2"
	if fen != expected {
		t.Errorf("unexpected FEN:\n  got:  %s\n  want: %s", fen, expected)
	}
}

func TestPositionEmpty(t *testing.T) {
	h, _ := newTestHandler()
	// Empty position command should not crash.
	h.ProcessCommand("position")
}

func TestEmptyCommand(t *testing.T) {
	h, _ := newTestHandler()
	quit := h.ProcessCommand("")
	if quit {
		t.Error("empty command should not quit")
	}
}
