package movegen

import (
	"checkmatego/internal/board"
	"testing"
)

func TestMagicRookConsistency(t *testing.T) {
	// For every square, verify that the magic table matches the slow reference
	// for a few occupancy configurations.
	testOcc := []board.Bitboard{
		0,
		board.SquareBB(board.E4),
		board.SquareBB(board.A1) | board.SquareBB(board.H8),
		board.Rank4BB,
		board.FileEBB,
	}

	for sq := board.A1; sq <= board.H8; sq++ {
		for _, occ := range testOcc {
			got := RookAttacks(sq, occ)
			want := rookAttacksSlow(sq, occ)
			if got != want {
				t.Errorf("rook attacks mismatch on %s with occ %x: got %x, want %x", sq, occ, got, want)
			}
		}
	}
}

func TestMagicBishopConsistency(t *testing.T) {
	testOcc := []board.Bitboard{
		0,
		board.SquareBB(board.E4),
		board.SquareBB(board.C3) | board.SquareBB(board.F6),
		board.Rank4BB | board.FileDBB,
	}

	for sq := board.A1; sq <= board.H8; sq++ {
		for _, occ := range testOcc {
			got := BishopAttacks(sq, occ)
			want := bishopAttacksSlow(sq, occ)
			if got != want {
				t.Errorf("bishop attacks mismatch on %s with occ %x: got %x, want %x", sq, occ, got, want)
			}
		}
	}
}
