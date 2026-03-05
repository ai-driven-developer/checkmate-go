//go:build !embed_nnue

package nnue

import "errors"

// LoadEmbeddedNetwork returns an error when built without the embed_nnue tag.
func LoadEmbeddedNetwork() (*Network, error) {
	return nil, errors.New("no embedded NNUE network (build with -tags embed_nnue)")
}
