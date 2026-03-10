//go:build arm64 && !nosimd

package nnue

// NEON is mandatory on ARMv8-A — no runtime detection needed.

//go:noescape
func asmVecAddSub16(dst, add, sub *int16)

//go:noescape
func asmVecAdd16(dst, src *int16)

//go:noescape
func asmVecSub16(dst, src *int16)

//go:noescape
func asmVecEvalPerspective(hidden *int32, acc *int16, weights *int16)

// vecSubAddSub16 falls back to Go on ARM64 (NEON version not yet implemented).
func vecSubAddSub16(dst, add, sub1, sub2 *int16) {
	goVecSubAddSub16(dst, add, sub1, sub2)
}

func vecAddSub16(dst, add, sub *int16) {
	asmVecAddSub16(dst, add, sub)
}

func vecAdd16(dst, src *int16) {
	asmVecAdd16(dst, src)
}

func vecSub16(dst, src *int16) {
	asmVecSub16(dst, src)
}

func vecEvalPerspective(hidden *int32, acc *int16, weights *int16) {
	asmVecEvalPerspective(hidden, acc, weights)
}
