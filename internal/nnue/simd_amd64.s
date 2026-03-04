#include "textflag.h"

// All functions require AVX2 (GOAMD64=v3).

// func vecAddSub16(dst, add, sub *int16)
// dst[i] += add[i] - sub[i] for 256 int16 values.
// AVX2: 16 int16 per YMM register, 16 iterations.
TEXT ·vecAddSub16(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), AX
	MOVQ add+8(FP), BX
	MOVQ sub+16(FP), CX
	MOVQ $16, DX

loop_addsub:
	VMOVDQU (BX), Y0
	VMOVDQU (CX), Y1
	VPSUBW  Y1, Y0, Y0       // Y0 = add - sub
	VMOVDQU (AX), Y2
	VPADDW  Y0, Y2, Y2       // Y2 = dst + (add - sub)
	VMOVDQU Y2, (AX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	ADDQ    $32, CX
	DECQ    DX
	JNZ     loop_addsub
	VZEROUPPER
	RET

// func vecAdd16(dst, src *int16)
// dst[i] += src[i] for 256 int16 values.
TEXT ·vecAdd16(SB), NOSPLIT, $0-16
	MOVQ dst+0(FP), AX
	MOVQ src+8(FP), BX
	MOVQ $16, CX

loop_add:
	VMOVDQU (BX), Y0
	VMOVDQU (AX), Y1
	VPADDW  Y0, Y1, Y1
	VMOVDQU Y1, (AX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	DECQ    CX
	JNZ     loop_add
	VZEROUPPER
	RET

// func vecSub16(dst, src *int16)
// dst[i] -= src[i] for 256 int16 values.
TEXT ·vecSub16(SB), NOSPLIT, $0-16
	MOVQ dst+0(FP), AX
	MOVQ src+8(FP), BX
	MOVQ $16, CX

loop_sub:
	VMOVDQU (AX), Y0
	VMOVDQU (BX), Y1
	VPSUBW  Y1, Y0, Y0
	VMOVDQU Y0, (AX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	DECQ    CX
	JNZ     loop_sub
	VZEROUPPER
	RET

// func vecEvalPerspective(hidden *int32, acc *int16, weights *int32)
// Processes one perspective (256 neurons) of the NNUE hidden layer.
// For each accumulator value: ClippedReLU(0, 255), then if non-zero,
// hidden[j] += clamped * weights[i][j] for j=0..31 using AVX2 VPMULLD.
// weights layout: [256][32]int32, stride = 128 bytes per row.
TEXT ·vecEvalPerspective(SB), NOSPLIT, $0-24
	MOVQ hidden+0(FP), DI      // DI = &hidden[0]
	MOVQ acc+8(FP), SI         // SI = &acc[0] (256 int16)
	MOVQ weights+16(FP), R8    // R8 = &weights[0][0]
	MOVQ $256, CX              // 256 neurons

eval_loop:
	// Load int16 accumulator value, sign-extend to int32.
	MOVWLSX (SI), AX

	// ClippedReLU: skip if <= 0.
	TESTL AX, AX
	JLE   eval_skip

	// Clamp to QA = 255.
	CMPL  AX, $255
	JLE   eval_no_clamp
	MOVL  $255, AX
eval_no_clamp:

	// Broadcast clamped value to all 8 int32 slots.
	MOVD         AX, X0
	VPBROADCASTD X0, Y0

	// hidden[0..7] += v * w[0..7]
	VMOVDQU 0(R8), Y1
	VPMULLD Y0, Y1, Y1
	VMOVDQU 0(DI), Y2
	VPADDD  Y1, Y2, Y2
	VMOVDQU Y2, 0(DI)

	// hidden[8..15] += v * w[8..15]
	VMOVDQU 32(R8), Y1
	VPMULLD Y0, Y1, Y1
	VMOVDQU 32(DI), Y2
	VPADDD  Y1, Y2, Y2
	VMOVDQU Y2, 32(DI)

	// hidden[16..23] += v * w[16..23]
	VMOVDQU 64(R8), Y1
	VPMULLD Y0, Y1, Y1
	VMOVDQU 64(DI), Y2
	VPADDD  Y1, Y2, Y2
	VMOVDQU Y2, 64(DI)

	// hidden[24..31] += v * w[24..31]
	VMOVDQU 96(R8), Y1
	VPMULLD Y0, Y1, Y1
	VMOVDQU 96(DI), Y2
	VPADDD  Y1, Y2, Y2
	VMOVDQU Y2, 96(DI)

eval_skip:
	ADDQ $2, SI                // next int16 in accumulator
	ADDQ $128, R8              // next weight row (32 * 4 bytes)
	DECQ CX
	JNZ  eval_loop

	VZEROUPPER
	RET
