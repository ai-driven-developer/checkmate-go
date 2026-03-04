package nnue

// vecAddSub16 computes dst[i] += add[i] - sub[i] for 256 int16 values.
// Requires AVX2.
//
//go:noescape
func vecAddSub16(dst, add, sub *int16)

// vecAdd16 computes dst[i] += src[i] for 256 int16 values.
// Requires AVX2.
//
//go:noescape
func vecAdd16(dst, src *int16)

// vecSub16 computes dst[i] -= src[i] for 256 int16 values.
// Requires AVX2.
//
//go:noescape
func vecSub16(dst, src *int16)

// vecEvalPerspective processes one perspective of the NNUE hidden layer.
// For each of 256 accumulator int16 values, applies ClippedReLU(0, 255),
// and if non-zero, accumulates: hidden[j] += clamped * weights[i][j] for j=0..31.
// Requires AVX2.
//
//go:noescape
func vecEvalPerspective(hidden *int32, acc *int16, weights *int32)
