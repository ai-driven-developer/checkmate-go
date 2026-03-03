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
	as.stack[as.idx+1] = as.stack[as.idx]
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
	for i := 0; i < HiddenSize; i++ {
		acc.Values[0][i] = as.Net.FeatureBiases[i]
		acc.Values[1][i] = as.Net.FeatureBiases[i]
	}

	// Add active features for every piece on the board.
	for sq := board.Square(0); sq < 64; sq++ {
		piece, color := pos.PieceAt(sq)
		if piece == board.NoPiece {
			continue
		}
		wIdx := FeatureIndex(board.White, color, piece, sq)
		bIdx := FeatureIndex(board.Black, color, piece, sq)
		for j := 0; j < HiddenSize; j++ {
			acc.Values[0][j] += as.Net.FeatureWeights[wIdx][j]
			acc.Values[1][j] += as.Net.FeatureWeights[bIdx][j]
		}
	}
}

// AddFeature adds the weights for a single feature to the current accumulator
// for the given perspective.
func (as *AccumulatorStack) AddFeature(perspective board.Color, index int) {
	acc := &as.stack[as.idx]
	p := int(perspective)
	for j := 0; j < HiddenSize; j++ {
		acc.Values[p][j] += as.Net.FeatureWeights[index][j]
	}
}

// SubFeature subtracts the weights for a single feature from the current
// accumulator for the given perspective.
func (as *AccumulatorStack) SubFeature(perspective board.Color, index int) {
	acc := &as.stack[as.idx]
	p := int(perspective)
	for j := 0; j < HiddenSize; j++ {
		acc.Values[p][j] -= as.Net.FeatureWeights[index][j]
	}
}

// addSubFeature performs a combined add+subtract for efficiency when
// moving a piece (subtract old square, add new square).
func (as *AccumulatorStack) addSubFeature(perspective board.Color, addIdx, subIdx int) {
	acc := &as.stack[as.idx]
	p := int(perspective)
	for j := 0; j < HiddenSize; j++ {
		acc.Values[p][j] += as.Net.FeatureWeights[addIdx][j] - as.Net.FeatureWeights[subIdx][j]
	}
}
