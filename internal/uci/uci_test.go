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
