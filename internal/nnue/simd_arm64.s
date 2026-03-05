#include "textflag.h"

// ARM64 NEON implementations. NEON is mandatory on all ARMv8-A processors.

// func asmVecAdd16(dst, src *int16)
// dst[i] += src[i] for 256 int16 values.
// NEON: 16 int16 per iteration (2 × V.H8), 16 iterations.
TEXT ·asmVecAdd16(SB), NOSPLIT, $0-16
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD $16, R2

loop_add:
	VLD1 (R1), [V0.H8, V1.H8]     // load 16 int16 from src
	VLD1 (R0), [V2.H8, V3.H8]     // load 16 int16 from dst
	VADD V0.H8, V2.H8, V2.H8
	VADD V1.H8, V3.H8, V3.H8
	VST1 [V2.H8, V3.H8], (R0)
	ADD  $32, R0, R0
	ADD  $32, R1, R1
	SUB  $1, R2, R2
	CBNZ R2, loop_add
	RET

// func asmVecSub16(dst, src *int16)
// dst[i] -= src[i] for 256 int16 values.
TEXT ·asmVecSub16(SB), NOSPLIT, $0-16
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD $16, R2

loop_sub:
	VLD1 (R0), [V0.H8, V1.H8]     // load 16 int16 from dst
	VLD1 (R1), [V2.H8, V3.H8]     // load 16 int16 from src
	VSUB V2.H8, V0.H8, V0.H8
	VSUB V3.H8, V1.H8, V1.H8
	VST1 [V0.H8, V1.H8], (R0)
	ADD  $32, R0, R0
	ADD  $32, R1, R1
	SUB  $1, R2, R2
	CBNZ R2, loop_sub
	RET

// func asmVecAddSub16(dst, add, sub *int16)
// dst[i] += add[i] - sub[i] for 256 int16 values.
TEXT ·asmVecAddSub16(SB), NOSPLIT, $0-24
	MOVD dst+0(FP), R0
	MOVD add+8(FP), R1
	MOVD sub+16(FP), R2
	MOVD $16, R3

loop_addsub:
	VLD1 (R1), [V0.H8, V1.H8]     // add[16]
	VLD1 (R2), [V2.H8, V3.H8]     // sub[16]
	VSUB V2.H8, V0.H8, V0.H8      // add - sub
	VSUB V3.H8, V1.H8, V1.H8
	VLD1 (R0), [V4.H8, V5.H8]     // dst[16]
	VADD V0.H8, V4.H8, V4.H8      // dst + (add - sub)
	VADD V1.H8, V5.H8, V5.H8
	VST1 [V4.H8, V5.H8], (R0)
	ADD  $32, R0, R0
	ADD  $32, R1, R1
	ADD  $32, R2, R2
	SUB  $1, R3, R3
	CBNZ R3, loop_addsub
	RET

// func asmVecEvalPerspective(hidden *int32, acc *int16, weights *int16)
// Processes one perspective (256 neurons) of the NNUE hidden layer.
// For each accumulator value: ClippedReLU(0, 255), then if non-zero,
// hidden[j] += clamped * weights[i][j] for j=0..31.
//
// Uses SMLAL/SMLAL2 (signed multiply-accumulate long, int16→int32)
// via WORD encoding — Go 1.22's assembler lacks these mnemonics.
//
// Hidden[32] is kept entirely in V16-V23 (8 × S4) across all 256
// iterations, eliminating per-neuron memory traffic for the output.
//
// weights layout: [256][32]int16, stride = 64 bytes per row.
TEXT ·asmVecEvalPerspective(SB), NOSPLIT, $0-24
	MOVD hidden+0(FP), R0      // R0 = &hidden[0]
	MOVD acc+8(FP), R1         // R1 = &acc[0] (256 int16)
	MOVD weights+16(FP), R2    // R2 = &weights[0][0]

	// Load hidden[0..31] into V16-V23 (8 × 4 int32 = 128 bytes).
	VLD1 (R0), [V16.S4, V17.S4, V18.S4, V19.S4]
	ADD  $64, R0, R5
	VLD1 (R5), [V20.S4, V21.S4, V22.S4, V23.S4]

	MOVD $256, R3               // 256 neurons

eval_loop:
	// Load int16 accumulator value, sign-extend to 64-bit.
	MOVH (R1), R4

	// ClippedReLU: skip if <= 0.
	CMPW $0, R4
	BLE  eval_skip

	// Clamp to QA = 255.
	CMPW $255, R4
	BLE  eval_no_clamp
	MOVW $255, R4
eval_no_clamp:

	// Broadcast clamped value to all 8 int16 lanes.
	VDUP R4, V0.H8

	// Load 32 int16 weights (4 × V.H8 = 64 bytes).
	VLD1 (R2), [V1.H8, V2.H8, V3.H8, V4.H8]

	// SMLAL/SMLAL2: hidden[j] += clamped * weights[j]
	// Go 1.22 lacks SMLAL/SMLAL2 mnemonics — raw WORD encoding.
	//
	// SMLAL  Vd.4S, Vn.4H, Vm.4H  = 0x0E608000 | Rm<<16 | Rn<<5 | Rd
	// SMLAL2 Vd.4S, Vn.8H, Vm.8H  = 0x4E608000 | Rm<<16 | Rn<<5 | Rd
	WORD $0x0E618010  // SMLAL  V16.4S, V0.4H, V1.4H  → hidden[0..3]
	WORD $0x4E618011  // SMLAL2 V17.4S, V0.8H, V1.8H  → hidden[4..7]
	WORD $0x0E628012  // SMLAL  V18.4S, V0.4H, V2.4H  → hidden[8..11]
	WORD $0x4E628013  // SMLAL2 V19.4S, V0.8H, V2.8H  → hidden[12..15]
	WORD $0x0E638014  // SMLAL  V20.4S, V0.4H, V3.4H  → hidden[16..19]
	WORD $0x4E638015  // SMLAL2 V21.4S, V0.8H, V3.8H  → hidden[20..23]
	WORD $0x0E648016  // SMLAL  V22.4S, V0.4H, V4.4H  → hidden[24..27]
	WORD $0x4E648017  // SMLAL2 V23.4S, V0.8H, V4.8H  → hidden[28..31]

eval_skip:
	ADD  $2, R1, R1             // next int16 in accumulator
	ADD  $64, R2, R2            // next weight row (32 × 2 bytes)
	SUB  $1, R3, R3
	CBNZ R3, eval_loop

	// Store hidden[0..31] back from V16-V23.
	VST1 [V16.S4, V17.S4, V18.S4, V19.S4], (R0)
	ADD  $64, R0, R5
	VST1 [V20.S4, V21.S4, V22.S4, V23.S4], (R5)

	RET
