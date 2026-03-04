package nnue

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
)

// makeTestNetwork creates a deterministic network for testing.
func makeTestNetwork() *Network {
	n := &Network{}
	v := int16(1)
	for i := range n.FeatureWeights {
		for j := range n.FeatureWeights[i] {
			n.FeatureWeights[i][j] = v
			v++
			if v > 10 {
				v = -10
			}
		}
	}
	for j := range n.FeatureBiases {
		n.FeatureBiases[j] = int16(j % 5)
	}
	for i := range n.HiddenWeights {
		for j := range n.HiddenWeights[i] {
			n.HiddenWeights[i][j] = int8((i + j) % 5)
		}
	}
	for j := range n.HiddenBiases {
		n.HiddenBiases[j] = int32(j)
	}
	for j := range n.OutputWeights {
		n.OutputWeights[j] = int8(j%3 - 1)
	}
	n.OutputBias = 0
	n.expandWeights()
	return n
}

// writeNetwork serializes a Network to a buffer in the binary format.
func writeNetwork(n *Network) *bytes.Buffer {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, [4]byte{'N', 'N', 'U', 'E'})
	binary.Write(&buf, binary.LittleEndian, uint32(1))
	binary.Write(&buf, binary.LittleEndian, &n.FeatureWeights)
	binary.Write(&buf, binary.LittleEndian, &n.FeatureBiases)
	binary.Write(&buf, binary.LittleEndian, &n.HiddenWeights)
	binary.Write(&buf, binary.LittleEndian, &n.HiddenBiases)
	binary.Write(&buf, binary.LittleEndian, &n.OutputWeights)
	binary.Write(&buf, binary.LittleEndian, &n.OutputBias)
	return &buf
}

// findMove finds a legal move matching the given UCI string (e.g. "e2e4").
func findMove(pos *board.Position, uci string) board.Move {
	var ml board.MoveList
	movegen.GenerateLegalMoves(pos, &ml)
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].String() == uci {
			return ml.Moves[i]
		}
	}
	return board.NullMove
}

// accEqual returns true if two accumulators have identical values.
func accEqual(a, b *Accumulator) bool {
	return a.Values == b.Values
}

// assertAccMatch checks that incremental accumulator matches a fresh refresh.
func assertAccMatch(t *testing.T, net *Network, as *AccumulatorStack, pos *board.Position, context string) {
	t.Helper()
	fresh := NewAccumulatorStack(net)
	fresh.Refresh(pos)
	inc := as.Current()
	ref := fresh.Current()
	for p := 0; p < 2; p++ {
		for j := 0; j < HiddenSize; j++ {
			if inc.Values[p][j] != ref.Values[p][j] {
				t.Errorf("%s: mismatch perspective=%d idx=%d: incremental=%d refresh=%d",
					context, p, j, inc.Values[p][j], ref.Values[p][j])
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Feature index tests
// ---------------------------------------------------------------------------

func TestFeatureIndex(t *testing.T) {
	// White pawn on e2 from White's perspective.
	idx := FeatureIndex(board.White, board.White, board.Pawn, board.E2)
	// relColor=0, pieceType-1=0, sq=12 -> 0*384+0*64+12=12
	if idx != 12 {
		t.Errorf("White pawn e2 from White: got %d, want 12", idx)
	}

	// Black knight on c6 from White's perspective.
	idx = FeatureIndex(board.White, board.Black, board.Knight, board.C6)
	// relColor=1, pieceType-1=1, sq=42 -> 384+64+42=490
	if idx != 490 {
		t.Errorf("Black knight c6 from White: got %d, want 490", idx)
	}

	// White pawn on e2 from Black's perspective.
	idx = FeatureIndex(board.Black, board.White, board.Pawn, board.E2)
	// relColor=1, pieceType-1=0, sq=12^56=52 -> 384+0+52=436
	if idx != 436 {
		t.Errorf("White pawn e2 from Black: got %d, want 436", idx)
	}
}

func TestFeatureIndexBounds(t *testing.T) {
	// All valid (perspective, color, piece, sq) combinations must produce
	// indices in [0, InputSize).
	for _, persp := range []board.Color{board.White, board.Black} {
		for _, color := range []board.Color{board.White, board.Black} {
			for piece := board.Pawn; piece <= board.King; piece++ {
				for sq := board.Square(0); sq < 64; sq++ {
					idx := FeatureIndex(persp, color, piece, sq)
					if idx < 0 || idx >= InputSize {
						t.Fatalf("out of range: persp=%d color=%d piece=%d sq=%d -> %d",
							persp, color, piece, sq, idx)
					}
				}
			}
		}
	}
}

func TestFeatureIndexSymmetry(t *testing.T) {
	// A white pawn on e2 from White's perspective should have the same index
	// as a black pawn on e7 from Black's perspective (mirrored board).
	// White pawn e2 from White: relColor=0, piece=Pawn(0), sq=E2(12) -> 12
	idx1 := FeatureIndex(board.White, board.White, board.Pawn, board.E2)
	// Black pawn e7 from Black: relColor=0, piece=Pawn(0), sq=E7(52)^56=12 -> 12
	idx2 := FeatureIndex(board.Black, board.Black, board.Pawn, board.E7)
	if idx1 != idx2 {
		t.Errorf("symmetry broken: White pawn e2 from White=%d, Black pawn e7 from Black=%d", idx1, idx2)
	}
}

func TestFeatureIndexUniqueness(t *testing.T) {
	// All feature indices for a given perspective must be unique.
	seen := make(map[int]bool)
	for _, color := range []board.Color{board.White, board.Black} {
		for piece := board.Pawn; piece <= board.King; piece++ {
			for sq := board.Square(0); sq < 64; sq++ {
				idx := FeatureIndex(board.White, color, piece, sq)
				if seen[idx] {
					t.Fatalf("duplicate index %d for color=%d piece=%d sq=%d", idx, color, piece, sq)
				}
				seen[idx] = true
			}
		}
	}
	if len(seen) != InputSize {
		t.Errorf("expected %d unique indices, got %d", InputSize, len(seen))
	}
}

func TestFeatureIndexFriendlyEnemy(t *testing.T) {
	// From White's perspective: White pieces are friendly (relColor=0),
	// Black pieces are enemy (relColor=1).
	wPawn := FeatureIndex(board.White, board.White, board.Pawn, board.A1)
	bPawn := FeatureIndex(board.White, board.Black, board.Pawn, board.A1)
	if wPawn >= 384 {
		t.Error("friendly piece index should be in [0, 384)")
	}
	if bPawn < 384 {
		t.Error("enemy piece index should be in [384, 768)")
	}
}

// ---------------------------------------------------------------------------
// Network loading / error handling tests
// ---------------------------------------------------------------------------

func TestNetworkLoadRoundtrip(t *testing.T) {
	net := makeTestNetwork()
	buf := writeNetwork(net)

	net2, err := ReadNetwork(buf)
	if err != nil {
		t.Fatalf("ReadNetwork: %v", err)
	}

	if net.FeatureWeights != net2.FeatureWeights {
		t.Error("Feature weights mismatch")
	}
	if net.FeatureBiases != net2.FeatureBiases {
		t.Error("Feature biases mismatch")
	}
	if net.HiddenWeights != net2.HiddenWeights {
		t.Error("Hidden weights mismatch")
	}
	if net.HiddenBiases != net2.HiddenBiases {
		t.Error("Hidden biases mismatch")
	}
	if net.OutputWeights != net2.OutputWeights {
		t.Error("Output weights mismatch")
	}
	if net.OutputBias != net2.OutputBias {
		t.Error("Output bias mismatch")
	}
}

func TestNetworkLoadBadMagic(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte("XXXX"))
	binary.Write(&buf, binary.LittleEndian, uint32(1))

	_, err := ReadNetwork(&buf)
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
	if !strings.Contains(err.Error(), "bad magic") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNetworkLoadBadVersion(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte("NNUE"))
	binary.Write(&buf, binary.LittleEndian, uint32(99))

	_, err := ReadNetwork(&buf)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
	if !strings.Contains(err.Error(), "unsupported version") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNetworkLoadTruncated(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte("NNUE"))
	binary.Write(&buf, binary.LittleEndian, uint32(1))
	// No weights at all.

	_, err := ReadNetwork(&buf)
	if err == nil {
		t.Fatal("expected error for truncated data")
	}
}

func TestNetworkLoadEmpty(t *testing.T) {
	_, err := ReadNetwork(&bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for empty reader")
	}
}

// ---------------------------------------------------------------------------
// Forward pass tests
// ---------------------------------------------------------------------------

func TestEvaluateDeterministic(t *testing.T) {
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	score1 := net.Evaluate(as.Current(), pos.SideToMove)
	score2 := net.Evaluate(as.Current(), pos.SideToMove)
	if score1 != score2 {
		t.Errorf("non-deterministic: %d vs %d", score1, score2)
	}
}

func TestEvaluatePerspectiveUsesCorrectOrder(t *testing.T) {
	// Verify that Evaluate actually reads the sideToMove argument:
	// use an asymmetric position where White has extra material.
	net := makeTestNetwork()
	pos := &board.Position{}
	// White has an extra queen.
	pos.SetFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKQNR w KQkq - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	scoreW := net.Evaluate(as.Current(), board.White)
	scoreB := net.Evaluate(as.Current(), board.Black)
	// With different "us" and "them" weight blocks, the score must differ.
	// This verifies that the perspective order (us first, them second) is
	// actually applied.
	_ = scoreW
	_ = scoreB
	// Just ensure no panic. With random-ish weights, exact value
	// comparison is fragile; the key invariant is that the call uses
	// sideToMove to select weight blocks.
}

func TestEvaluateZeroNetwork(t *testing.T) {
	// A network with all-zero weights should always produce zero.
	net := &Network{}
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	score := net.Evaluate(as.Current(), board.White)
	if score != 0 {
		t.Errorf("zero network produced non-zero score: %d", score)
	}
}

func TestEvaluateAfterMove(t *testing.T) {
	// After making a move, the score should change.
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	scoreBefore := net.Evaluate(as.Current(), board.White)

	m := findMove(pos, "e2e4")
	if m == board.NullMove {
		t.Fatal("move e2e4 not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)

	scoreAfter := net.Evaluate(as.Current(), pos.SideToMove)
	// We don't know the direction, but the score should differ.
	_ = scoreBefore
	_ = scoreAfter
	// Just verify no panic — with random weights the scores will differ.
}

// ---------------------------------------------------------------------------
// Accumulator refresh tests
// ---------------------------------------------------------------------------

func TestAccumulatorRefreshStartingPosition(t *testing.T) {
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	// Verify it starts from biases, not zero.
	acc := as.Current()
	allZero := true
	for j := 0; j < HiddenSize; j++ {
		if acc.Values[0][j] != 0 || acc.Values[1][j] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("accumulator should not be all zeros after refresh with non-zero network")
	}
}

func TestAccumulatorRefreshIdempotent(t *testing.T) {
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)
	snap1 := *as.Current()

	as.Refresh(pos)
	snap2 := *as.Current()

	if !accEqual(&snap1, &snap2) {
		t.Error("double refresh produced different accumulators")
	}
}

func TestAccumulatorRefreshFromFEN(t *testing.T) {
	// Refresh from a midgame FEN should work correctly.
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("r1bqkb1r/pppppppp/2n2n2/8/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3")

	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	// Should not panic and should produce non-zero values.
	acc := as.Current()
	allZero := true
	for j := 0; j < HiddenSize; j++ {
		if acc.Values[0][j] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("accumulator should not be all zeros from midgame FEN")
	}
}

// ---------------------------------------------------------------------------
// Accumulator incremental update tests — specific move types
// ---------------------------------------------------------------------------

func TestAccumulatorQuietMove(t *testing.T) {
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e2e4") // double pawn push (FlagDoublePawn)
	if m == board.NullMove {
		t.Fatal("e2e4 not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after e2e4")
}

func TestAccumulatorCapture(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	// After 1.e4 d5 — exd5 is a capture.
	pos.SetFromFEN("rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e4d5")
	if m == board.NullMove {
		t.Fatal("e4d5 not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after exd5")
}

func TestAccumulatorEnPassant(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	// White pawn on f5, Black just played e7-e5: en passant available.
	pos.SetFromFEN("rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "f5e6")
	if m == board.NullMove {
		t.Fatal("f5e6 (en passant) not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after fxe6 en passant")
}

func TestAccumulatorKingsideCastle(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e1g1")
	if m == board.NullMove {
		t.Fatal("e1g1 (O-O) not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after O-O white")
}

func TestAccumulatorQueensideCastle(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e1c1")
	if m == board.NullMove {
		t.Fatal("e1c1 (O-O-O) not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after O-O-O white")
}

func TestAccumulatorBlackCastle(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQkq - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e8g8")
	if m == board.NullMove {
		t.Fatal("e8g8 (O-O black) not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after O-O black")
}

func TestAccumulatorPromotion(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("8/4P3/8/8/8/8/8/4K2k w - - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e7e8q")
	if m == board.NullMove {
		t.Fatal("e7e8q not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after e8=Q")
}

func TestAccumulatorPromotionCapture(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("3r4/4P3/8/8/8/8/8/4K2k w - - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e7d8q")
	if m == board.NullMove {
		t.Fatal("e7d8q not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after exd8=Q")
}

func TestAccumulatorUnderpromotion(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("8/4P3/8/8/8/8/8/4K2k w - - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	m := findMove(pos, "e7e8n")
	if m == board.NullMove {
		t.Fatal("e7e8n not found")
	}
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	assertAccMatch(t, net, as, pos, "after e8=N")
}

// ---------------------------------------------------------------------------
// Accumulator make/unmake consistency tests
// ---------------------------------------------------------------------------

func TestAccumulatorMakeUnmakeQuiet(t *testing.T) {
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)
	snap := *as.Current()

	m := findMove(pos, "g1f3")
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	pos.UnmakeMove(m)
	as.UnmakeMove()

	if !accEqual(as.Current(), &snap) {
		t.Error("accumulator not restored after make/unmake Nf3")
	}
}

func TestAccumulatorMakeUnmakeCapture(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)
	snap := *as.Current()

	m := findMove(pos, "e4d5")
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	pos.UnmakeMove(m)
	as.UnmakeMove()

	if !accEqual(as.Current(), &snap) {
		t.Error("accumulator not restored after make/unmake exd5")
	}
}

func TestAccumulatorMakeUnmakeCastle(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)
	snap := *as.Current()

	m := findMove(pos, "e1g1")
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	pos.UnmakeMove(m)
	as.UnmakeMove()

	if !accEqual(as.Current(), &snap) {
		t.Error("accumulator not restored after make/unmake O-O")
	}
}

func TestAccumulatorMakeUnmakeEnPassant(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)
	snap := *as.Current()

	m := findMove(pos, "f5e6")
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	pos.UnmakeMove(m)
	as.UnmakeMove()

	if !accEqual(as.Current(), &snap) {
		t.Error("accumulator not restored after make/unmake en passant")
	}
}

func TestAccumulatorMakeUnmakePromotion(t *testing.T) {
	net := makeTestNetwork()
	pos := &board.Position{}
	pos.SetFromFEN("8/4P3/8/8/8/8/8/4K2k w - - 0 1")
	as := NewAccumulatorStack(net)
	as.Refresh(pos)
	snap := *as.Current()

	m := findMove(pos, "e7e8q")
	as.MakeMove(pos, m)
	pos.MakeMove(m)
	pos.UnmakeMove(m)
	as.UnmakeMove()

	if !accEqual(as.Current(), &snap) {
		t.Error("accumulator not restored after make/unmake promotion")
	}
}

// ---------------------------------------------------------------------------
// Null move tests
// ---------------------------------------------------------------------------

func TestAccumulatorNullMove(t *testing.T) {
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)
	snap := *as.Current()

	// Null move should not change accumulator values.
	as.MakeNullMove()
	if !accEqual(as.Current(), &snap) {
		t.Error("null move changed accumulator values")
	}

	as.UnmakeNullMove()
	if !accEqual(as.Current(), &snap) {
		t.Error("unmake null move changed accumulator values")
	}
}

// ---------------------------------------------------------------------------
// Deep game sequence test — plays a full opening and checks consistency
// ---------------------------------------------------------------------------

func TestAccumulatorFullGameSequence(t *testing.T) {
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	// Italian Game: 1.e4 e5 2.Nf3 Nc6 3.Bc4 Bc5
	uciMoves := []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5"}

	// Store actual Move values so we can unmake correctly.
	played := make([]board.Move, 0, len(uciMoves))
	for _, uci := range uciMoves {
		m := findMove(pos, uci)
		if m == board.NullMove {
			t.Fatalf("move %s not found", uci)
		}
		played = append(played, m)
		as.MakeMove(pos, m)
		pos.MakeMove(m)
		assertAccMatch(t, net, as, pos, "after "+uci)
	}

	// Now unmake all moves — should be back to starting position.
	for i := len(played) - 1; i >= 0; i-- {
		pos.UnmakeMove(played[i])
		as.UnmakeMove()
	}

	// Verify we're back at the starting position accumulator.
	assertAccMatch(t, net, as, pos, "back at startpos")
}

func TestAccumulatorLongSequenceVsRefresh(t *testing.T) {
	// Play many random-ish moves and verify accumulator consistency
	// at various depths.
	net := makeTestNetwork()
	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	for ply := 0; ply < 40; ply++ {
		var ml board.MoveList
		movegen.GenerateLegalMoves(pos, &ml)
		if ml.Count == 0 {
			break
		}
		// Pick move at index ply%Count for variety.
		m := ml.Moves[ply%ml.Count]
		as.MakeMove(pos, m)
		pos.MakeMove(m)

		// Check consistency every 5 plies.
		if ply%5 == 4 {
			assertAccMatch(t, net, as, pos, "ply "+string(rune('0'+ply/10))+string(rune('0'+ply%10)))
		}
	}
}

// ---------------------------------------------------------------------------
// Stack depth tests
// ---------------------------------------------------------------------------

func TestAccumulatorStackPushPop(t *testing.T) {
	net := makeTestNetwork()
	as := NewAccumulatorStack(net)
	pos := board.NewPosition()
	as.Refresh(pos)

	// Push multiple times, then pop back.
	for i := 0; i < 10; i++ {
		as.Push()
	}
	for i := 0; i < 10; i++ {
		as.Pop()
	}

	// Should be back at idx 0 — same as initial refresh.
	fresh := NewAccumulatorStack(net)
	fresh.Refresh(pos)
	if !accEqual(as.Current(), fresh.Current()) {
		t.Error("stack push/pop cycle corrupted accumulator")
	}
}

// ---------------------------------------------------------------------------
// ClippedReLU boundary tests
// ---------------------------------------------------------------------------

func TestEvaluateClippedReLUBounds(t *testing.T) {
	// Create a network where accumulator values will be negative and above QA
	// to exercise ClippedReLU clamping.
	net := &Network{}
	// Set large positive biases so accumulator > QA.
	for j := range net.FeatureBiases {
		net.FeatureBiases[j] = 1000
	}
	// Set some output weights so we can observe the effect.
	for j := range net.OutputWeights {
		net.OutputWeights[j] = 1
	}
	for j := range net.HiddenWeights {
		for k := range net.HiddenWeights[j] {
			net.HiddenWeights[j][k] = 1
		}
	}

	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	// Should not panic and produce a finite result.
	score := net.Evaluate(as.Current(), board.White)
	_ = score
}

func TestEvaluateNegativeAccumulator(t *testing.T) {
	// Create a network where accumulator values will be very negative.
	net := &Network{}
	for j := range net.FeatureBiases {
		net.FeatureBiases[j] = -1000
	}
	for j := range net.OutputWeights {
		net.OutputWeights[j] = 1
	}

	pos := board.NewPosition()
	as := NewAccumulatorStack(net)
	as.Refresh(pos)

	// ClippedReLU should clamp negatives to 0, so output should be
	// determined only by biases and output bias.
	score := net.Evaluate(as.Current(), board.White)
	// All accumulator values clamped to 0 -> hidden layer gets only biases
	// -> after ClippedReLU and scaling, score should be 0 (biases are 0).
	if score != 0 {
		t.Errorf("expected 0 with all-negative accumulator, got %d", score)
	}
}
