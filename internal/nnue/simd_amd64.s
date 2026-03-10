#include "textflag.h"

// All SIMD functions require AVX2.

// func asmHasAVX2() bool
// Checks CPUID for OSXSAVE + OS YMM save support + AVX2 bit.
TEXT ·asmHasAVX2(SB), NOSPLIT, $0-1
	// Check OSXSAVE: CPUID.1:ECX[bit 27]
	MOVL $1, AX
	XORL CX, CX
	CPUID
	TESTL $(1<<27), CX
	JZ   no_avx2

	// Check OS saves YMM state: XGETBV(XCR0) bits 1,2
	XORL CX, CX
	BYTE $0x0F; BYTE $0x01; BYTE $0xD0 // XGETBV
	ANDL $6, AX
	CMPL AX, $6
	JNE  no_avx2

	// Check AVX2: CPUID.7:EBX[bit 5]
	MOVL $7, AX
	XORL CX, CX
	CPUID
	TESTL $(1<<5), BX
	JZ   no_avx2

	MOVB $1, ret+0(FP)
	RET

no_avx2:
	MOVB $0, ret+0(FP)
	RET

// func asmVecAddSub16(dst, add, sub *int16)
// dst[i] += add[i] - sub[i] for 256 int16 values.
// AVX2: 16 int16 per YMM register, 16 iterations.
TEXT ·asmVecAddSub16(SB), NOSPLIT, $0-24
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

// func asmVecAdd16(dst, src *int16)
// dst[i] += src[i] for 256 int16 values.
TEXT ·asmVecAdd16(SB), NOSPLIT, $0-16
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

// func asmVecSub16(dst, src *int16)
// dst[i] -= src[i] for 256 int16 values.
TEXT ·asmVecSub16(SB), NOSPLIT, $0-16
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

// func asmVecSubAddSub16(dst, add, sub1, sub2 *int16)
// dst[i] += add[i] - sub1[i] - sub2[i] for 256 int16 values.
// Used for captures: move piece (add dst, sub src) and remove captured (sub2).
TEXT ·asmVecSubAddSub16(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), AX
	MOVQ add+8(FP), BX
	MOVQ sub1+16(FP), CX
	MOVQ sub2+24(FP), DX
	MOVQ $16, R8

loop_subaddsub:
	VMOVDQU (BX), Y0          // add
	VMOVDQU (CX), Y1          // sub1
	VPSUBW  Y1, Y0, Y0        // add - sub1
	VMOVDQU (DX), Y1          // sub2
	VPSUBW  Y1, Y0, Y0        // add - sub1 - sub2
	VMOVDQU (AX), Y2          // dst
	VPADDW  Y0, Y2, Y2        // dst + add - sub1 - sub2
	VMOVDQU Y2, (AX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	ADDQ    $32, CX
	ADDQ    $32, DX
	DECQ    R8
	JNZ     loop_subaddsub
	VZEROUPPER
	RET

// func asmVecEvalPerspective(hidden *int32, acc *int16, weights *int16)
// Processes one perspective (256 neurons) of the NNUE hidden layer.
// For each accumulator value: ClippedReLU(0, 255), then if non-zero,
// hidden[j] += clamped * weights[i][j] for j=0..31.
// Uses VPMULLW (int16×int16) instead of VPMULLD — faster on AMD Zen3.
// weights layout: [256][32]int16, stride = 64 bytes per row.
//
// hidden[0..31] is kept in registers Y8-Y11 across all 256 iterations,
// eliminating per-neuron memory traffic (same approach as the NEON version).
TEXT ·asmVecEvalPerspective(SB), NOSPLIT, $0-24
	MOVQ hidden+0(FP), DI      // DI = &hidden[0]
	MOVQ acc+8(FP), SI         // SI = &acc[0] (256 int16)
	MOVQ weights+16(FP), R8    // R8 = &weights[0][0]
	MOVQ $256, CX              // 256 neurons

	// Load hidden[0..31] into Y8-Y11 (4 × 8 int32 = 128 bytes).
	VMOVDQU 0(DI), Y8          // hidden[0..7]
	VMOVDQU 32(DI), Y9         // hidden[8..15]
	VMOVDQU 64(DI), Y10        // hidden[16..23]
	VMOVDQU 96(DI), Y11        // hidden[24..31]

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

	// Broadcast clamped value to all 16 int16 lanes of Y0.
	// Pack val into both halves of a 32-bit word, then broadcast.
	MOVL  AX, DX
	SHLL  $16, DX
	ORL   AX, DX
	MOVD  DX, X0
	VPBROADCASTD X0, Y0      // Y0 = [val, val, val, ...] (16 × int16)

	// --- First 16 weights → hidden[0..15] ---
	VMOVDQU (R8), Y1          // 16 int16 weights
	VPMULLW Y0, Y1, Y1        // 16 int16 products (fits: 255*127=32385)

	// Lower 8 products → sign-extend to int32 → accumulate into Y8
	VPMOVSXWD X1, Y2
	VPADDD  Y2, Y8, Y8

	// Upper 8 products → sign-extend to int32 → accumulate into Y9
	VEXTRACTI128 $1, Y1, X4
	VPMOVSXWD X4, Y2
	VPADDD  Y2, Y9, Y9

	// --- Next 16 weights → hidden[16..31] ---
	VMOVDQU 32(R8), Y1        // next 16 int16 weights
	VPMULLW Y0, Y1, Y1

	// Lower 8 → accumulate into Y10
	VPMOVSXWD X1, Y2
	VPADDD  Y2, Y10, Y10

	// Upper 8 → accumulate into Y11
	VEXTRACTI128 $1, Y1, X4
	VPMOVSXWD X4, Y2
	VPADDD  Y2, Y11, Y11

eval_skip:
	ADDQ $2, SI                // next int16 in accumulator
	ADDQ $64, R8               // next weight row (32 × 2 bytes)
	DECQ CX
	JNZ  eval_loop

	// Store hidden[0..31] back from Y8-Y11.
	VMOVDQU Y8, 0(DI)
	VMOVDQU Y9, 32(DI)
	VMOVDQU Y10, 64(DI)
	VMOVDQU Y11, 96(DI)

	VZEROUPPER
	RET
