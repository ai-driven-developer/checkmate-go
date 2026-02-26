package search

import (
	"checkmatego/internal/board"
	"checkmatego/internal/movegen"
)

// seeValue holds piece values for Static Exchange Evaluation.
var seeValue = [7]int{
	0,     // NoPiece
	100,   // Pawn
	320,   // Knight
	330,   // Bishop
	500,   // Rook
	900,   // Queen
	20000, // King
}

// attackersTo returns a bitboard of all pieces (both colors) that attack sq
// with the given occupancy.
func attackersTo(pos *board.Position, sq board.Square, occ board.Bitboard) board.Bitboard {
	bishops := pos.Pieces[board.White][board.Bishop] | pos.Pieces[board.Black][board.Bishop]
	rooks := pos.Pieces[board.White][board.Rook] | pos.Pieces[board.Black][board.Rook]
	queens := pos.Pieces[board.White][board.Queen] | pos.Pieces[board.Black][board.Queen]

	return (movegen.PawnAttacks[board.Black][sq] & pos.Pieces[board.White][board.Pawn]) |
		(movegen.PawnAttacks[board.White][sq] & pos.Pieces[board.Black][board.Pawn]) |
		(movegen.KnightAttacks[sq] & (pos.Pieces[board.White][board.Knight] | pos.Pieces[board.Black][board.Knight])) |
		(movegen.BishopAttacks(sq, occ) & (bishops | queens)) |
		(movegen.RookAttacks(sq, occ) & (rooks | queens)) |
		(movegen.KingAttacks[sq] & (pos.Pieces[board.White][board.King] | pos.Pieces[board.Black][board.King]))
}

// SEE performs Static Exchange Evaluation for a capture move.
// Returns the expected material gain (positive) or loss (negative) from
// the full exchange sequence on the target square.
func SEE(pos *board.Position, m board.Move) int {
	to := m.To()
	target := m.CapturedPiece()

	if target == board.NoPiece {
		return 0
	}

	attacker := m.Piece()

	var gain [32]int
	d := 0
	gain[d] = seeValue[target]

	occ := pos.AllOccupied
	occ ^= board.SquareBB(m.From())

	// En passant: remove the actual captured pawn from occupancy.
	if m.Flags() == board.FlagEnPassant {
		var epSq board.Square
		if pos.SideToMove == board.White {
			epSq = to - 8
		} else {
			epSq = to + 8
		}
		occ ^= board.SquareBB(epSq)
	}

	// Promotion: the piece on the target square becomes the promoted piece.
	if m.IsPromotion() {
		gain[d] += seeValue[m.PromotionPiece()] - seeValue[board.Pawn]
		attacker = m.PromotionPiece()
	}

	// Precompute sliding piece bitboards for x-ray discovery.
	diagSliders := pos.Pieces[board.White][board.Bishop] | pos.Pieces[board.Black][board.Bishop] |
		pos.Pieces[board.White][board.Queen] | pos.Pieces[board.Black][board.Queen]
	straightSliders := pos.Pieces[board.White][board.Rook] | pos.Pieces[board.Black][board.Rook] |
		pos.Pieces[board.White][board.Queen] | pos.Pieces[board.Black][board.Queen]

	attackers := attackersTo(pos, to, occ) & occ
	color := pos.SideToMove.Other()

	for {
		d++
		gain[d] = seeValue[attacker] - gain[d-1]

		// Find the least valuable attacker for the current side.
		found := false
		for piece := board.Pawn; piece <= board.King; piece++ {
			subset := attackers & pos.Pieces[color][piece]
			if subset != 0 {
				sq := subset.LSB()
				occ ^= board.SquareBB(sq)

				// Discover sliding piece x-rays through the removed piece.
				attackers |= movegen.BishopAttacks(to, occ) & diagSliders
				attackers |= movegen.RookAttacks(to, occ) & straightSliders
				attackers &= occ

				attacker = piece
				found = true
				break
			}
		}

		if !found {
			break
		}

		color = color.Other()
	}

	// Minimax the gain array from back to front.
	for d--; d > 0; d-- {
		gain[d-1] = -max(-gain[d-1], gain[d])
	}

	return gain[0]
}
