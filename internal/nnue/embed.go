//go:build embed_nnue

package nnue

import (
	"bytes"
	_ "embed"
)

//go:embed best.nnue
var embeddedNet []byte

// LoadEmbeddedNetwork returns the built-in NNUE network embedded at compile time.
func LoadEmbeddedNetwork() (*Network, error) {
	return ReadNetwork(bytes.NewReader(embeddedNet))
}
