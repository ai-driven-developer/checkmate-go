// Command gennet generates a random NNUE network file for testing.
//
// Usage: go run ./cmd/gennet -o network.nnue
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math/rand"
	"os"

	"checkmatego/internal/nnue"
)

func main() {
	output := flag.String("o", "network.nnue", "output file path")
	seed := flag.Int64("seed", 42, "random seed")
	flag.Parse()

	rng := rand.New(rand.NewSource(*seed))

	f, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Write header.
	binary.Write(f, binary.LittleEndian, [4]byte{'N', 'N', 'U', 'E'})
	binary.Write(f, binary.LittleEndian, uint32(1)) // version

	// Feature transformer weights [768][256] int16.
	// Small random values to avoid overflow in accumulator.
	for i := 0; i < nnue.InputSize; i++ {
		for j := 0; j < nnue.HiddenSize; j++ {
			v := int16(rng.Intn(21) - 10) // [-10, 10]
			binary.Write(f, binary.LittleEndian, v)
		}
	}

	// Feature transformer biases [256] int16.
	for j := 0; j < nnue.HiddenSize; j++ {
		v := int16(rng.Intn(21) - 10)
		binary.Write(f, binary.LittleEndian, v)
	}

	// Hidden layer weights [512][32] int8.
	for i := 0; i < 2*nnue.HiddenSize; i++ {
		for j := 0; j < nnue.L2Size; j++ {
			v := int8(rng.Intn(11) - 5) // [-5, 5]
			binary.Write(f, binary.LittleEndian, v)
		}
	}

	// Hidden layer biases [32] int32.
	for j := 0; j < nnue.L2Size; j++ {
		v := int32(rng.Intn(201) - 100) // [-100, 100]
		binary.Write(f, binary.LittleEndian, v)
	}

	// Output layer weights [32] int8.
	for j := 0; j < nnue.L2Size; j++ {
		v := int8(rng.Intn(11) - 5)
		binary.Write(f, binary.LittleEndian, v)
	}

	// Output layer bias int32.
	binary.Write(f, binary.LittleEndian, int32(0))

	fmt.Printf("Generated %s (seed=%d)\n", *output, *seed)
}
