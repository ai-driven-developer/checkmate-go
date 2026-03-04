package nnue

import "checkmatego/internal/board"

const maxPly = 512

// Accumulator stores the feature transformer output for both perspectives.
type Accumulator struct {
	Values [2][HiddenSize]int16
}

// AccumulatorStack maintains a stack of accumulators for incremental
// updates during search (push on MakeMove, pop on UnmakeMove).
type AccumulatorStack struct {
	stack [maxPly]Accumulator
	idx   int
	Net   *Network
}

// NewAccumulatorStack creates a new accumulator stack bound to a network.
func NewAccumulatorStack(net *Network) *AccumulatorStack {
	return &AccumulatorStack{Net: net}
}

// Current returns a pointer to the current accumulator.
func (as *AccumulatorStack) Current() *Accumulator {
	return &as.stack[as.idx]
}

// Push copies the current accumulator to the next slot and advances the index.
func (as *AccumulatorStack) Push() {
	copy(as.stack[as.idx+1].Values[0][:], as.stack[as.idx].Values[0][:])
	copy(as.stack[as.idx+1].Values[1][:], as.stack[as.idx].Values[1][:])
	as.idx++
}

// Pop restores the previous accumulator.
func (as *AccumulatorStack) Pop() {
	as.idx--
}

// Refresh recomputes the accumulator from scratch by scanning all pieces
// on the board. Must be called at the root before search begins.
func (as *AccumulatorStack) Refresh(pos *board.Position) {
	as.idx = 0
	acc := &as.stack[0]

	// Start from biases.
	copy(acc.Values[0][:], as.Net.FeatureBiases[:])
	copy(acc.Values[1][:], as.Net.FeatureBiases[:])

	// Add active features for every piece on the board.
	for sq := board.Square(0); sq < 64; sq++ {
		piece, color := pos.PieceAt(sq)
		if piece == board.NoPiece {
			continue
		}
		wIdx := FeatureIndex(board.White, color, piece, sq)
		bIdx := FeatureIndex(board.Black, color, piece, sq)
		vecAdd16(&acc.Values[0][0], &as.Net.FeatureWeights[wIdx][0])
		vecAdd16(&acc.Values[1][0], &as.Net.FeatureWeights[bIdx][0])
	}
}

// addBoth adds the weights for a feature to both perspectives.
func (as *AccumulatorStack) addBoth(wIdx, bIdx int) {
	acc := &as.stack[as.idx]
	vecAdd16(&acc.Values[0][0], &as.Net.FeatureWeights[wIdx][0])
	vecAdd16(&acc.Values[1][0], &as.Net.FeatureWeights[bIdx][0])
}

// subBoth subtracts the weights for a feature from both perspectives.
func (as *AccumulatorStack) subBoth(wIdx, bIdx int) {
	acc := &as.stack[as.idx]
	vecSub16(&acc.Values[0][0], &as.Net.FeatureWeights[wIdx][0])
	vecSub16(&acc.Values[1][0], &as.Net.FeatureWeights[bIdx][0])
}

// addSubBoth performs combined add+subtract for both perspectives
// (moving a piece: subtract old square, add new square).
func (as *AccumulatorStack) addSubBoth(wAddIdx, wSubIdx, bAddIdx, bSubIdx int) {
	acc := &as.stack[as.idx]
	vecAddSub16(&acc.Values[0][0], &as.Net.FeatureWeights[wAddIdx][0], &as.Net.FeatureWeights[wSubIdx][0])
	vecAddSub16(&acc.Values[1][0], &as.Net.FeatureWeights[bAddIdx][0], &as.Net.FeatureWeights[bSubIdx][0])
}

// subAddSubBoth combines removePiece + movePiece into a single loop for captures.
// Subtracts captured piece and moves our piece in one pass over 256 elements.
func (as *AccumulatorStack) subAddSubBoth(
	wCapIdx, bCapIdx int, // captured piece feature indices
	wAddIdx, wSubIdx, bAddIdx, bSubIdx int, // moving piece add/sub indices
) {
	acc := &as.stack[as.idx]
	wc := &as.Net.FeatureWeights[wCapIdx]
	bc := &as.Net.FeatureWeights[bCapIdx]
	wa := &as.Net.FeatureWeights[wAddIdx]
	ws := &as.Net.FeatureWeights[wSubIdx]
	ba := &as.Net.FeatureWeights[bAddIdx]
	bs := &as.Net.FeatureWeights[bSubIdx]
	for j := 0; j < HiddenSize; j++ {
		acc.Values[0][j] += wa[j] - ws[j] - wc[j]
		acc.Values[1][j] += ba[j] - bs[j] - bc[j]
	}
}
