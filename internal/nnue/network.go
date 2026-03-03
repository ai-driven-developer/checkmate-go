package nnue

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"checkmatego/internal/board"
)

// Network holds the quantized weights and biases of the NNUE.
//
// Architecture: (768→256)×2 → 32 → 1
//   - Feature transformer: [768][256] int16 weights + [256] int16 biases
//   - Hidden layer: [512][32] int8 weights + [32] int32 biases
//   - Output layer: [32] int8 weights + int32 bias
type Network struct {
	FeatureWeights [InputSize][HiddenSize]int16
	FeatureBiases  [HiddenSize]int16
	HiddenWeights  [2 * HiddenSize][L2Size]int8
	HiddenBiases   [L2Size]int32
	OutputWeights  [L2Size]int8
	OutputBias     int32
}

// Magic bytes and version for the binary network format.
var netMagic = [4]byte{'N', 'N', 'U', 'E'}

const netVersion uint32 = 1

// LoadNetwork reads a Network from a binary file.
func LoadNetwork(path string) (*Network, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("nnue: open %s: %w", path, err)
	}
	defer f.Close()
	return ReadNetwork(f)
}

// ReadNetwork reads a Network from an io.Reader.
func ReadNetwork(r io.Reader) (*Network, error) {
	var magic [4]byte
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("nnue: read magic: %w", err)
	}
	if magic != netMagic {
		return nil, fmt.Errorf("nnue: bad magic %v", magic)
	}

	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("nnue: read version: %w", err)
	}
	if version != netVersion {
		return nil, fmt.Errorf("nnue: unsupported version %d", version)
	}

	n := &Network{}

	// Feature transformer weights [768][256] int16.
	if err := binary.Read(r, binary.LittleEndian, &n.FeatureWeights); err != nil {
		return nil, fmt.Errorf("nnue: read feature weights: %w", err)
	}
	// Feature transformer biases [256] int16.
	if err := binary.Read(r, binary.LittleEndian, &n.FeatureBiases); err != nil {
		return nil, fmt.Errorf("nnue: read feature biases: %w", err)
	}
	// Hidden layer weights [512][32] int8.
	if err := binary.Read(r, binary.LittleEndian, &n.HiddenWeights); err != nil {
		return nil, fmt.Errorf("nnue: read hidden weights: %w", err)
	}
	// Hidden layer biases [32] int32.
	if err := binary.Read(r, binary.LittleEndian, &n.HiddenBiases); err != nil {
		return nil, fmt.Errorf("nnue: read hidden biases: %w", err)
	}
	// Output layer weights [32] int8.
	if err := binary.Read(r, binary.LittleEndian, &n.OutputWeights); err != nil {
		return nil, fmt.Errorf("nnue: read output weights: %w", err)
	}
	// Output layer bias int32.
	if err := binary.Read(r, binary.LittleEndian, &n.OutputBias); err != nil {
		return nil, fmt.Errorf("nnue: read output bias: %w", err)
	}

	return n, nil
}

// Evaluate performs the forward pass and returns a score in centipawns
// from the side-to-move's perspective.
func (n *Network) Evaluate(acc *Accumulator, sideToMove board.Color) int {
	// Determine perspective order: side-to-move first, then opponent.
	us := int(sideToMove)
	them := int(sideToMove ^ 1)

	// Hidden layer: ClippedReLU on accumulator, then linear transform.
	var hidden [L2Size]int32
	for j := 0; j < L2Size; j++ {
		hidden[j] = n.HiddenBiases[j]
	}

	// "Us" perspective (first 256 inputs).
	for i := 0; i < HiddenSize; i++ {
		v := int32(acc.Values[us][i])
		// ClippedReLU: clamp to [0, QA].
		if v < 0 {
			v = 0
		} else if v > QA {
			v = QA
		}
		for j := 0; j < L2Size; j++ {
			hidden[j] += v * int32(n.HiddenWeights[i][j])
		}
	}

	// "Them" perspective (next 256 inputs).
	for i := 0; i < HiddenSize; i++ {
		v := int32(acc.Values[them][i])
		if v < 0 {
			v = 0
		} else if v > QA {
			v = QA
		}
		for j := 0; j < L2Size; j++ {
			hidden[j] += v * int32(n.HiddenWeights[HiddenSize+i][j])
		}
	}

	// Output layer: ClippedReLU on hidden, then linear transform.
	output := n.OutputBias
	for j := 0; j < L2Size; j++ {
		// Divide by QA to rescale after the first multiplication.
		v := hidden[j] / QA
		if v < 0 {
			v = 0
		} else if v > QB {
			v = QB
		}
		output += v * int32(n.OutputWeights[j])
	}

	// Scale: output is in quantized units; convert to centipawns.
	// The accumulator values are already scaled by the weight quantization.
	// Final division accounts for QB scaling.
	return int(output) * OutputScale / (QB * QA)
}
