//go:build !amd64 && !arm64

package nnue

// On non-amd64 platforms, dispatch directly to pure Go implementations.

func vecAddSub16(dst, add, sub *int16) {
	goVecAddSub16(dst, add, sub)
}

func vecAdd16(dst, src *int16) {
	goVecAdd16(dst, src)
}

func vecSub16(dst, src *int16) {
	goVecSub16(dst, src)
}

func vecEvalPerspective(hidden *int32, acc *int16, weights *int16) {
	goVecEvalPerspective(hidden, acc, weights)
}
