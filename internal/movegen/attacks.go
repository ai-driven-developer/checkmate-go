package movegen

import "checkmatego/internal/board"

// Precomputed attack tables, initialized in init().
var (
	KnightAttacks [64]board.Bitboard
	KingAttacks   [64]board.Bitboard
	PawnAttacks   [2][64]board.Bitboard // [color][square]
)

func init() {
	initKnightAttacks()
	initKingAttacks()
	initPawnAttacks()
}

func initKnightAttacks() {
	for sq := board.A1; sq <= board.H8; sq++ {
		bb := board.SquareBB(sq)
		var attacks board.Bitboard
		attacks |= (bb &^ board.FileABB &^ board.FileBBB) >> 6  // SSW→ left 2, down 1 — wrong, let me recalculate
		// Knight moves: (+1,+2), (+2,+1), (+2,-1), (+1,-2), (-1,-2), (-2,-1), (-2,+1), (-1,+2)
		attacks = 0
		attacks |= (bb &^ board.FileHBB) << 17          // up 2, right 1
		attacks |= (bb &^ board.FileABB) << 15          // up 2, left 1
		attacks |= (bb &^ board.FileGBB &^ board.FileHBB) << 10 // up 1, right 2
		attacks |= (bb &^ board.FileABB &^ board.FileBBB) << 6  // up 1, left 2
		attacks |= (bb &^ board.FileHBB) >> 15          // down 2, right 1
		attacks |= (bb &^ board.FileABB) >> 17          // down 2, left 1
		attacks |= (bb &^ board.FileGBB &^ board.FileHBB) >> 6  // down 1, right 2
		attacks |= (bb &^ board.FileABB &^ board.FileBBB) >> 10 // down 1, left 2
		KnightAttacks[sq] = attacks
	}
}

func initKingAttacks() {
	for sq := board.A1; sq <= board.H8; sq++ {
		bb := board.SquareBB(sq)
		var attacks board.Bitboard
		attacks |= bb.North()
		attacks |= bb.South()
		attacks |= bb.East()
		attacks |= bb.West()
		attacks |= bb.NorthEast()
		attacks |= bb.NorthWest()
		attacks |= bb.SouthEast()
		attacks |= bb.SouthWest()
		KingAttacks[sq] = attacks
	}
}

func initPawnAttacks() {
	for sq := board.A1; sq <= board.H8; sq++ {
		bb := board.SquareBB(sq)
		PawnAttacks[board.White][sq] = bb.NorthWest() | bb.NorthEast()
		PawnAttacks[board.Black][sq] = bb.SouthWest() | bb.SouthEast()
	}
}

// IsSquareAttacked returns true if 'sq' is attacked by any piece of 'attacker' color.
func IsSquareAttacked(pos *board.Position, sq board.Square, attacker board.Color) bool {
	if sq >= board.NoSquare {
		return false
	}
	// Pawn attacks.
	if PawnAttacks[attacker.Other()][sq]&pos.Pieces[attacker][board.Pawn] != 0 {
		return true
	}
	// Knight attacks.
	if KnightAttacks[sq]&pos.Pieces[attacker][board.Knight] != 0 {
		return true
	}
	// King attacks.
	if KingAttacks[sq]&pos.Pieces[attacker][board.King] != 0 {
		return true
	}
	// Bishop/Queen (diagonal).
	occ := pos.AllOccupied
	if BishopAttacks(sq, occ)&(pos.Pieces[attacker][board.Bishop]|pos.Pieces[attacker][board.Queen]) != 0 {
		return true
	}
	// Rook/Queen (straight).
	if RookAttacks(sq, occ)&(pos.Pieces[attacker][board.Rook]|pos.Pieces[attacker][board.Queen]) != 0 {
		return true
	}
	return false
}
