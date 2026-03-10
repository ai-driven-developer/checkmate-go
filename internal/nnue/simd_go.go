package nnue

import "unsafe"

// Pure Go implementations of SIMD operations.
// Used as fallback on non-amd64 platforms or when AVX2 is unavailable.

func goVecSubAddSub16(dst, add, sub1, sub2 *int16) {
	d := unsafe.Slice(dst, HiddenSize)
	a := unsafe.Slice(add, HiddenSize)
	s1 := unsafe.Slice(sub1, HiddenSize)
	s2 := unsafe.Slice(sub2, HiddenSize)
	for i := 0; i < HiddenSize; i++ {
		d[i] += a[i] - s1[i] - s2[i]
	}
}

func goVecAddSub16(dst, add, sub *int16) {
	d := unsafe.Slice(dst, HiddenSize)
	a := unsafe.Slice(add, HiddenSize)
	s := unsafe.Slice(sub, HiddenSize)
	for i := 0; i < HiddenSize; i++ {
		d[i] += a[i] - s[i]
	}
}

func goVecAdd16(dst, src *int16) {
	d := unsafe.Slice(dst, HiddenSize)
	s := unsafe.Slice(src, HiddenSize)
	for i := 0; i < HiddenSize; i++ {
		d[i] += s[i]
	}
}

func goVecSub16(dst, src *int16) {
	d := unsafe.Slice(dst, HiddenSize)
	s := unsafe.Slice(src, HiddenSize)
	for i := 0; i < HiddenSize; i++ {
		d[i] -= s[i]
	}
}

func goVecEvalPerspective(hidden *int32, acc *int16, weights *int16) {
	h := unsafe.Slice(hidden, L2Size)
	a := unsafe.Slice(acc, HiddenSize)
	// weights is [HiddenSize][L2Size]int16 laid out contiguously.
	w := unsafe.Slice(weights, HiddenSize*L2Size)

	for i := 0; i < HiddenSize; i++ {
		v := int32(a[i])
		if v <= 0 {
			continue
		}
		if v > QA {
			v = QA
		}
		row := w[i*L2Size : i*L2Size+L2Size]
		for j := 0; j < L2Size; j++ {
			h[j] += v * int32(row[j])
		}
	}
}
