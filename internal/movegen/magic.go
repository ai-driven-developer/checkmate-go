package movegen

import (
	"checkmatego/internal/board"
	"math/bits"
)

// Magic entry for a single square.
type Magic struct {
	Mask    board.Bitboard
	Magic   uint64
	Shift   uint
	Attacks []board.Bitboard
}

var (
	rookMagics   [64]Magic
	bishopMagics [64]Magic

	// Backing storage — sized for worst case with variable shift.
	rookAttackTable   [64 * 4096]board.Bitboard
	bishopAttackTable [64 * 512]board.Bitboard
)

// Simple PRNG for magic number search (xorshift64).
type rng uint64

func (r *rng) next() uint64 {
	*r ^= *r << 13
	*r ^= *r >> 7
	*r ^= *r << 17
	return uint64(*r)
}

// sparseRandom returns a random number with few bits set (good magic candidate).
func (r *rng) sparseRandom() uint64 {
	return r.next() & r.next() & r.next()
}

func init() {
	initMagicBitboards()
}

func initMagicBitboards() {
	seed := rng(0x12345678ABCDEF01)

	// Initialize rook magics.
	var rookOffset int
	for sq := board.A1; sq <= board.H8; sq++ {
		mask := rookOccupancyMask(sq)
		numBits := uint(bits.OnesCount64(uint64(mask)))
		shift := 64 - numBits
		size := 1 << numBits

		// Enumerate all subsets and compute reference attacks.
		subsets := make([]board.Bitboard, size)
		reference := make([]board.Bitboard, size)
		occ := board.Bitboard(0)
		for i := 0; i < size; i++ {
			subsets[i] = occ
			reference[i] = rookAttacksSlow(sq, occ)
			occ = (occ - mask) & mask
		}

		// Find a working magic number.
		magic := findMagic(&seed, subsets, reference, shift, size)

		rookMagics[sq] = Magic{
			Mask:    mask,
			Magic:   magic,
			Shift:   shift,
			Attacks: rookAttackTable[rookOffset : rookOffset+size],
		}

		// Fill the attack table.
		for i := 0; i < size; i++ {
			idx := (uint64(subsets[i]) * magic) >> shift
			rookMagics[sq].Attacks[idx] = reference[i]
		}

		rookOffset += size
	}

	// Initialize bishop magics.
	var bishopOffset int
	for sq := board.A1; sq <= board.H8; sq++ {
		mask := bishopOccupancyMask(sq)
		numBits := uint(bits.OnesCount64(uint64(mask)))
		shift := 64 - numBits
		size := 1 << numBits

		subsets := make([]board.Bitboard, size)
		reference := make([]board.Bitboard, size)
		occ := board.Bitboard(0)
		for i := 0; i < size; i++ {
			subsets[i] = occ
			reference[i] = bishopAttacksSlow(sq, occ)
			occ = (occ - mask) & mask
		}

		magic := findMagic(&seed, subsets, reference, shift, size)

		bishopMagics[sq] = Magic{
			Mask:    mask,
			Magic:   magic,
			Shift:   shift,
			Attacks: bishopAttackTable[bishopOffset : bishopOffset+size],
		}

		for i := 0; i < size; i++ {
			idx := (uint64(subsets[i]) * magic) >> shift
			bishopMagics[sq].Attacks[idx] = reference[i]
		}

		bishopOffset += size
	}
}

// findMagic finds a magic number that maps all subsets to unique (or constructively colliding) indices.
func findMagic(seed *rng, subsets, reference []board.Bitboard, shift uint, size int) uint64 {
	used := make([]board.Bitboard, size)
	epoch := make([]int, size) // tracks which attempt wrote each slot

	for attempt := 1; ; attempt++ {
		magic := seed.sparseRandom()
		if bits.OnesCount64((uint64(subsets[0]|subsets[len(subsets)-1])*magic)>>56) < 6 {
			continue // heuristic: skip unlikely candidates
		}

		ok := true
		for i, occ := range subsets {
			idx := (uint64(occ) * magic) >> shift
			if epoch[idx] < attempt {
				// First write this attempt — claim the slot.
				epoch[idx] = attempt
				used[idx] = reference[i]
			} else if used[idx] != reference[i] {
				// Destructive collision.
				ok = false
				break
			}
			// Constructive collision (same attacks) is fine.
		}
		if ok {
			return magic
		}
	}
}

// RookAttacks returns the attack bitboard for a rook on sq given occupancy.
func RookAttacks(sq board.Square, occ board.Bitboard) board.Bitboard {
	m := &rookMagics[sq]
	idx := (uint64(occ&m.Mask) * m.Magic) >> m.Shift
	return m.Attacks[idx]
}

// BishopAttacks returns the attack bitboard for a bishop on sq given occupancy.
func BishopAttacks(sq board.Square, occ board.Bitboard) board.Bitboard {
	m := &bishopMagics[sq]
	idx := (uint64(occ&m.Mask) * m.Magic) >> m.Shift
	return m.Attacks[idx]
}

// QueenAttacks is the union of rook and bishop attacks.
func QueenAttacks(sq board.Square, occ board.Bitboard) board.Bitboard {
	return RookAttacks(sq, occ) | BishopAttacks(sq, occ)
}

// rookOccupancyMask returns the relevant occupancy mask for a rook on sq
// (excludes edge squares on the ray).
func rookOccupancyMask(sq board.Square) board.Bitboard {
	var mask board.Bitboard
	rank, file := sq.Rank(), sq.File()

	for r := rank + 1; r < 7; r++ {
		mask.Set(board.NewSquare(file, r))
	}
	for r := rank - 1; r > 0; r-- {
		mask.Set(board.NewSquare(file, r))
	}
	for f := file + 1; f < 7; f++ {
		mask.Set(board.NewSquare(f, rank))
	}
	for f := file - 1; f > 0; f-- {
		mask.Set(board.NewSquare(f, rank))
	}
	return mask
}

// bishopOccupancyMask returns the relevant occupancy mask for a bishop on sq.
func bishopOccupancyMask(sq board.Square) board.Bitboard {
	var mask board.Bitboard
	rank, file := sq.Rank(), sq.File()

	for r, f := rank+1, file+1; r < 7 && f < 7; r, f = r+1, f+1 {
		mask.Set(board.NewSquare(f, r))
	}
	for r, f := rank+1, file-1; r < 7 && f > 0; r, f = r+1, f-1 {
		mask.Set(board.NewSquare(f, r))
	}
	for r, f := rank-1, file+1; r > 0 && f < 7; r, f = r-1, f+1 {
		mask.Set(board.NewSquare(f, r))
	}
	for r, f := rank-1, file-1; r > 0 && f > 0; r, f = r-1, f-1 {
		mask.Set(board.NewSquare(f, r))
	}
	return mask
}

// rookAttacksSlow computes rook attacks by ray-tracing (reference implementation).
func rookAttacksSlow(sq board.Square, occ board.Bitboard) board.Bitboard {
	var attacks board.Bitboard
	rank, file := sq.Rank(), sq.File()

	for r := rank + 1; r <= 7; r++ {
		s := board.NewSquare(file, r)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	for r := rank - 1; r >= 0; r-- {
		s := board.NewSquare(file, r)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	for f := file + 1; f <= 7; f++ {
		s := board.NewSquare(f, rank)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	for f := file - 1; f >= 0; f-- {
		s := board.NewSquare(f, rank)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	return attacks
}

// bishopAttacksSlow computes bishop attacks by ray-tracing.
func bishopAttacksSlow(sq board.Square, occ board.Bitboard) board.Bitboard {
	var attacks board.Bitboard
	rank, file := sq.Rank(), sq.File()

	for r, f := rank+1, file+1; r <= 7 && f <= 7; r, f = r+1, f+1 {
		s := board.NewSquare(f, r)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	for r, f := rank+1, file-1; r <= 7 && f >= 0; r, f = r+1, f-1 {
		s := board.NewSquare(f, r)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	for r, f := rank-1, file+1; r >= 0 && f <= 7; r, f = r-1, f+1 {
		s := board.NewSquare(f, r)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	for r, f := rank-1, file-1; r >= 0 && f >= 0; r, f = r-1, f-1 {
		s := board.NewSquare(f, r)
		attacks.Set(s)
		if occ.Has(s) {
			break
		}
	}
	return attacks
}
