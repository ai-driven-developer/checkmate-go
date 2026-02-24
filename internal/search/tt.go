package search

import (
	"checkmatego/internal/board"
	"sync/atomic"
)

// Bound represents the type of score stored in a TT entry.
type Bound uint8

const (
	BoundNone  Bound = 0
	BoundUpper Bound = 1 // score is an upper bound (failed low)
	BoundLower Bound = 2 // score is a lower bound (failed high)
	BoundExact Bound = 3 // score is exact
)

// ttEntry stores a TT entry as two atomic uint64 values for lockless access.
// key is stored as hash XOR data — a torn read from a concurrent write will
// produce a key mismatch and be treated as a miss (safe).
//
// data layout (64 bits):
//
//	bits  0-31: best move (board.Move = uint32)
//	bits 32-47: score (int16, stored as uint16)
//	bits 48-55: depth (int8, stored as uint8)
//	bits 56-57: bound type (2 bits)
//	bits 58-63: generation (6 bits, 0-63)
type ttEntry struct {
	key  atomic.Uint64
	data atomic.Uint64
}

func packData(move board.Move, score int16, depth int8, bound Bound, gen uint8) uint64 {
	return uint64(move) |
		uint64(uint16(score))<<32 |
		uint64(uint8(depth))<<48 |
		uint64(bound&0x3)<<56 |
		uint64(gen&0x3F)<<58
}

func unpackMove(data uint64) board.Move { return board.Move(data & 0xFFFFFFFF) }
func unpackScore(data uint64) int16     { return int16(uint16(data >> 32)) }
func unpackDepth(data uint64) int8      { return int8(uint8(data >> 48)) }
func unpackBound(data uint64) Bound     { return Bound((data >> 56) & 0x3) }
func unpackGen(data uint64) uint8       { return uint8((data >> 58) & 0x3F) }

// TransTable is a lockless transposition table shared across search threads.
type TransTable struct {
	entries []ttEntry
	mask    uint64
	gen     uint8
}

// NewTransTable creates a transposition table of the given size in megabytes.
func NewTransTable(sizeMB int) *TransTable {
	if sizeMB < 1 {
		sizeMB = 1
	}
	const entrySize = 16 // two uint64 fields
	numEntries := (uint64(sizeMB) * 1024 * 1024) / entrySize
	numEntries = roundDownPow2(numEntries)
	if numEntries < 1024 {
		numEntries = 1024
	}
	return &TransTable{
		entries: make([]ttEntry, numEntries),
		mask:    numEntries - 1,
	}
}

func roundDownPow2(n uint64) uint64 {
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return (n >> 1) + 1
}

// Probe looks up a position in the TT.
func (tt *TransTable) Probe(hash uint64) (hit bool, move board.Move, score int16, depth int8, bound Bound) {
	idx := hash & tt.mask
	entry := &tt.entries[idx]

	data := entry.data.Load()
	storedKey := entry.key.Load()

	// Lockless verification: key was stored as hash XOR data.
	if storedKey^data != hash {
		return false, board.NullMove, 0, 0, BoundNone
	}

	return true, unpackMove(data), unpackScore(data), unpackDepth(data), unpackBound(data)
}

// Store writes a position into the TT.
// Replacement policy: always replace if different position, exact bound,
// newer generation, or deeper/equal depth.
func (tt *TransTable) Store(hash uint64, move board.Move, score int16, depth int8, bound Bound) {
	idx := hash & tt.mask
	entry := &tt.entries[idx]

	data := packData(move, score, depth, bound, tt.gen)

	oldData := entry.data.Load()
	oldKey := entry.key.Load()

	// If same position and new entry has no move, preserve existing move.
	if oldKey^oldData == hash && move == board.NullMove {
		move = unpackMove(oldData)
		data = packData(move, score, depth, bound, tt.gen)
	}

	// Replacement decision for same-position entries.
	if oldKey^oldData == hash {
		oldGen := unpackGen(oldData)
		oldDepth := unpackDepth(oldData)
		if bound != BoundExact && oldGen == tt.gen && depth < oldDepth {
			return // keep existing deeper entry from this generation
		}
	}

	entry.key.Store(hash ^ data)
	entry.data.Store(data)
}

// Clear zeroes all entries.
func (tt *TransTable) Clear() {
	for i := range tt.entries {
		tt.entries[i].key.Store(0)
		tt.entries[i].data.Store(0)
	}
}

// NewSearch increments the generation counter.
func (tt *TransTable) NewSearch() {
	tt.gen = (tt.gen + 1) & 0x3F
}

// Hashfull returns table usage in per mille (0-1000).
func (tt *TransTable) Hashfull() int {
	sample := 1000
	total := int(tt.mask + 1)
	if total < sample {
		sample = total
	}
	count := 0
	for i := 0; i < sample; i++ {
		data := tt.entries[i].data.Load()
		if data != 0 && unpackGen(data) == tt.gen {
			count++
		}
	}
	return count * 1000 / sample
}

// scoreToTT adjusts a score for TT storage.
// Mate scores are converted from root-relative to node-absolute.
func scoreToTT(score, ply int) int16 {
	if score > MateScore-MaxDepth {
		return int16(score + ply)
	}
	if score < -MateScore+MaxDepth {
		return int16(score - ply)
	}
	return int16(score)
}

// scoreFromTT reverses the TT adjustment.
func scoreFromTT(score int16, ply int) int {
	s := int(score)
	if s > MateScore-MaxDepth {
		return s - ply
	}
	if s < -MateScore+MaxDepth {
		return s + ply
	}
	return s
}
