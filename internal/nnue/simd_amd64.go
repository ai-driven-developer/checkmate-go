//go:build amd64 && simd

package nnue

// useAVX2 is set once at init time via CPUID.
var useAVX2 bool

func init() {
	useAVX2 = asmHasAVX2()
}

// Assembly stubs — require AVX2.

//go:noescape
func asmHasAVX2() bool

//go:noescape
func asmVecAddSub16(dst, add, sub *int16)

//go:noescape
func asmVecAdd16(dst, src *int16)

//go:noescape
func asmVecSub16(dst, src *int16)

//go:noescape
func asmVecSubAddSub16(dst, add, sub1, sub2 *int16)

//go:noescape
func asmVecEvalPerspective(hidden *int32, acc *int16, weights *int16)

// Dispatch: AVX2 assembly when available, pure Go otherwise.

func vecSubAddSub16(dst, add, sub1, sub2 *int16) {
	if useAVX2 {
		asmVecSubAddSub16(dst, add, sub1, sub2)
	} else {
		goVecSubAddSub16(dst, add, sub1, sub2)
	}
}

func vecAddSub16(dst, add, sub *int16) {
	if useAVX2 {
		asmVecAddSub16(dst, add, sub)
	} else {
		goVecAddSub16(dst, add, sub)
	}
}

func vecAdd16(dst, src *int16) {
	if useAVX2 {
		asmVecAdd16(dst, src)
	} else {
		goVecAdd16(dst, src)
	}
}

func vecSub16(dst, src *int16) {
	if useAVX2 {
		asmVecSub16(dst, src)
	} else {
		goVecSub16(dst, src)
	}
}

func vecEvalPerspective(hidden *int32, acc *int16, weights *int16) {
	if useAVX2 {
		asmVecEvalPerspective(hidden, acc, weights)
	} else {
		goVecEvalPerspective(hidden, acc, weights)
	}
}
