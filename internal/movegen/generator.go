package movegen

import "checkmatego/internal/board"

// GenerateLegalMoves generates all legal moves for the position.
func GenerateLegalMoves(pos *board.Position, ml *board.MoveList) {
	ml.Clear()
	generateAllPseudoLegal(pos, ml)
	filterIllegal(pos, ml)
}

// GenerateCaptures generates all legal capture moves (for quiescence).
func GenerateCaptures(pos *board.Position, ml *board.MoveList) {
	ml.Clear()
	generateCapturePseudoLegal(pos, ml)
	filterIllegal(pos, ml)
}

// filterIllegal removes pseudo-legal moves that leave the king in check.
func filterIllegal(pos *board.Position, ml *board.MoveList) {
	n := 0
	us := pos.SideToMove
	kingSq := pos.KingSquare(us)
	if kingSq >= board.NoSquare {
		ml.Count = 0
		return
	}
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		pos.MakeMove(m)
		kSq := pos.KingSquare(us)
		if kSq < board.NoSquare && !IsSquareAttacked(pos, kSq, us.Other()) {
			ml.Moves[n] = m
			n++
		}
		pos.UnmakeMove(m)
	}
	ml.Count = n
}

func generateAllPseudoLegal(pos *board.Position, ml *board.MoveList) {
	us := pos.SideToMove
	them := us.Other()
	ours := pos.Occupied[us]
	theirs := pos.Occupied[them]
	all := pos.AllOccupied

	generatePawnMoves(pos, ml, us, them, ours, theirs, all, false)
	generateKnightMoves(pos, ml, us, ours)
	generateBishopMoves(pos, ml, us, ours, all)
	generateRookMoves(pos, ml, us, ours, all)
	generateQueenMoves(pos, ml, us, ours, all)
	generateKingMoves(pos, ml, us, ours, theirs, all)
	generateCastling(pos, ml, us, all)
}

func generateCapturePseudoLegal(pos *board.Position, ml *board.MoveList) {
	us := pos.SideToMove
	them := us.Other()
	ours := pos.Occupied[us]
	theirs := pos.Occupied[them]
	all := pos.AllOccupied

	generatePawnMoves(pos, ml, us, them, ours, theirs, all, true)
	generateKnightCaptures(pos, ml, us, ours, theirs)
	generateBishopCaptures(pos, ml, us, ours, theirs, all)
	generateRookCaptures(pos, ml, us, ours, theirs, all)
	generateQueenCaptures(pos, ml, us, ours, theirs, all)
	generateKingCaptures(pos, ml, us, ours, theirs)
}

func generatePawnMoves(pos *board.Position, ml *board.MoveList, us, them board.Color, ours, theirs, all board.Bitboard, capturesOnly bool) {
	pawns := pos.Pieces[us][board.Pawn]
	empty := ^all
	promoRank := board.Rank8BB
	doublePushRank := board.Rank3BB
	if us == board.Black {
		promoRank = board.Rank1BB
		doublePushRank = board.Rank6BB
	}

	// Pawn captures.
	var leftCaptures, rightCaptures board.Bitboard
	if us == board.White {
		leftCaptures = pawns.NorthWest() & theirs
		rightCaptures = pawns.NorthEast() & theirs
	} else {
		leftCaptures = pawns.SouthWest() & theirs
		rightCaptures = pawns.SouthEast() & theirs
	}

	// Left captures.
	for leftCaptures != 0 {
		to := leftCaptures.PopLSB()
		var from board.Square
		if us == board.White {
			from = to - 7
		} else {
			from = to + 9
		}
		captured := pos.PieceOn[to]
		if board.SquareBB(to)&promoRank != 0 {
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureQueen, board.Pawn, captured))
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureRook, board.Pawn, captured))
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureBishop, board.Pawn, captured))
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureKnight, board.Pawn, captured))
		} else {
			ml.Add(board.NewMove(from, to, board.FlagCapture, board.Pawn, captured))
		}
	}

	// Right captures.
	for rightCaptures != 0 {
		to := rightCaptures.PopLSB()
		var from board.Square
		if us == board.White {
			from = to - 9
		} else {
			from = to + 7
		}
		captured := pos.PieceOn[to]
		if board.SquareBB(to)&promoRank != 0 {
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureQueen, board.Pawn, captured))
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureRook, board.Pawn, captured))
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureBishop, board.Pawn, captured))
			ml.Add(board.NewMove(from, to, board.FlagPromoCaptureKnight, board.Pawn, captured))
		} else {
			ml.Add(board.NewMove(from, to, board.FlagCapture, board.Pawn, captured))
		}
	}

	// En passant.
	if pos.EnPassant != board.NoSquare {
		epBB := PawnAttacks[them][pos.EnPassant] & pawns
		for epBB != 0 {
			from := epBB.PopLSB()
			ml.Add(board.NewMove(from, pos.EnPassant, board.FlagEnPassant, board.Pawn, board.Pawn))
		}
	}

	if capturesOnly {
		// Still generate promotion pushes (they change material).
		var pushes board.Bitboard
		if us == board.White {
			pushes = pawns.North() & empty & promoRank
		} else {
			pushes = pawns.South() & empty & promoRank
		}
		for pushes != 0 {
			to := pushes.PopLSB()
			var from board.Square
			if us == board.White {
				from = to - 8
			} else {
				from = to + 8
			}
			ml.Add(board.NewMove(from, to, board.FlagPromoQueen, board.Pawn, board.NoPiece))
			ml.Add(board.NewMove(from, to, board.FlagPromoRook, board.Pawn, board.NoPiece))
			ml.Add(board.NewMove(from, to, board.FlagPromoBishop, board.Pawn, board.NoPiece))
			ml.Add(board.NewMove(from, to, board.FlagPromoKnight, board.Pawn, board.NoPiece))
		}
		return
	}

	// Single pushes.
	var singlePush board.Bitboard
	if us == board.White {
		singlePush = pawns.North() & empty
	} else {
		singlePush = pawns.South() & empty
	}

	// Promotions from single push.
	promos := singlePush & promoRank
	for promos != 0 {
		to := promos.PopLSB()
		var from board.Square
		if us == board.White {
			from = to - 8
		} else {
			from = to + 8
		}
		ml.Add(board.NewMove(from, to, board.FlagPromoQueen, board.Pawn, board.NoPiece))
		ml.Add(board.NewMove(from, to, board.FlagPromoRook, board.Pawn, board.NoPiece))
		ml.Add(board.NewMove(from, to, board.FlagPromoBishop, board.Pawn, board.NoPiece))
		ml.Add(board.NewMove(from, to, board.FlagPromoKnight, board.Pawn, board.NoPiece))
	}

	// Non-promotion single pushes.
	quietPush := singlePush &^ promoRank
	for quietPush != 0 {
		to := quietPush.PopLSB()
		var from board.Square
		if us == board.White {
			from = to - 8
		} else {
			from = to + 8
		}
		ml.Add(board.NewMove(from, to, board.FlagQuiet, board.Pawn, board.NoPiece))
	}

	// Double pushes.
	var doublePush board.Bitboard
	if us == board.White {
		doublePush = (singlePush & doublePushRank).North() & empty
	} else {
		doublePush = (singlePush & doublePushRank).South() & empty
	}
	for doublePush != 0 {
		to := doublePush.PopLSB()
		var from board.Square
		if us == board.White {
			from = to - 16
		} else {
			from = to + 16
		}
		ml.Add(board.NewMove(from, to, board.FlagDoublePawn, board.Pawn, board.NoPiece))
	}
}

func generateKnightMoves(pos *board.Position, ml *board.MoveList, us board.Color, ours board.Bitboard) {
	knights := pos.Pieces[us][board.Knight]
	for knights != 0 {
		from := knights.PopLSB()
		targets := KnightAttacks[from] &^ ours
		for targets != 0 {
			to := targets.PopLSB()
			captured := pos.PieceOn[to]
			if captured != board.NoPiece {
				ml.Add(board.NewMove(from, to, board.FlagCapture, board.Knight, captured))
			} else {
				ml.Add(board.NewMove(from, to, board.FlagQuiet, board.Knight, board.NoPiece))
			}
		}
	}
}

func generateKnightCaptures(pos *board.Position, ml *board.MoveList, us board.Color, ours, theirs board.Bitboard) {
	knights := pos.Pieces[us][board.Knight]
	for knights != 0 {
		from := knights.PopLSB()
		targets := KnightAttacks[from] & theirs
		for targets != 0 {
			to := targets.PopLSB()
			ml.Add(board.NewMove(from, to, board.FlagCapture, board.Knight, pos.PieceOn[to]))
		}
	}
}

func generateSlidingMoves(pos *board.Position, ml *board.MoveList, us board.Color, piece board.Piece, ours, all board.Bitboard, attackFn func(board.Square, board.Bitboard) board.Bitboard) {
	pieces := pos.Pieces[us][piece]
	for pieces != 0 {
		from := pieces.PopLSB()
		targets := attackFn(from, all) &^ ours
		for targets != 0 {
			to := targets.PopLSB()
			captured := pos.PieceOn[to]
			if captured != board.NoPiece {
				ml.Add(board.NewMove(from, to, board.FlagCapture, piece, captured))
			} else {
				ml.Add(board.NewMove(from, to, board.FlagQuiet, piece, board.NoPiece))
			}
		}
	}
}

func generateSlidingCaptures(pos *board.Position, ml *board.MoveList, us board.Color, piece board.Piece, ours, theirs, all board.Bitboard, attackFn func(board.Square, board.Bitboard) board.Bitboard) {
	pieces := pos.Pieces[us][piece]
	for pieces != 0 {
		from := pieces.PopLSB()
		targets := attackFn(from, all) & theirs
		for targets != 0 {
			to := targets.PopLSB()
			ml.Add(board.NewMove(from, to, board.FlagCapture, piece, pos.PieceOn[to]))
		}
	}
}

func generateBishopMoves(pos *board.Position, ml *board.MoveList, us board.Color, ours, all board.Bitboard) {
	generateSlidingMoves(pos, ml, us, board.Bishop, ours, all, BishopAttacks)
}

func generateBishopCaptures(pos *board.Position, ml *board.MoveList, us board.Color, ours, theirs, all board.Bitboard) {
	generateSlidingCaptures(pos, ml, us, board.Bishop, ours, theirs, all, BishopAttacks)
}

func generateRookMoves(pos *board.Position, ml *board.MoveList, us board.Color, ours, all board.Bitboard) {
	generateSlidingMoves(pos, ml, us, board.Rook, ours, all, RookAttacks)
}

func generateRookCaptures(pos *board.Position, ml *board.MoveList, us board.Color, ours, theirs, all board.Bitboard) {
	generateSlidingCaptures(pos, ml, us, board.Rook, ours, theirs, all, RookAttacks)
}

func generateQueenMoves(pos *board.Position, ml *board.MoveList, us board.Color, ours, all board.Bitboard) {
	generateSlidingMoves(pos, ml, us, board.Queen, ours, all, QueenAttacks)
}

func generateQueenCaptures(pos *board.Position, ml *board.MoveList, us board.Color, ours, theirs, all board.Bitboard) {
	generateSlidingCaptures(pos, ml, us, board.Queen, ours, theirs, all, QueenAttacks)
}

func generateKingMoves(pos *board.Position, ml *board.MoveList, us board.Color, ours, theirs, all board.Bitboard) {
	from := pos.KingSquare(us)
	if from >= board.NoSquare {
		return
	}
	targets := KingAttacks[from] &^ ours
	for targets != 0 {
		to := targets.PopLSB()
		captured := pos.PieceOn[to]
		if captured != board.NoPiece {
			ml.Add(board.NewMove(from, to, board.FlagCapture, board.King, captured))
		} else {
			ml.Add(board.NewMove(from, to, board.FlagQuiet, board.King, board.NoPiece))
		}
	}
}

func generateKingCaptures(pos *board.Position, ml *board.MoveList, us board.Color, ours, theirs board.Bitboard) {
	from := pos.KingSquare(us)
	if from >= board.NoSquare {
		return
	}
	targets := KingAttacks[from] & theirs
	for targets != 0 {
		to := targets.PopLSB()
		ml.Add(board.NewMove(from, to, board.FlagCapture, board.King, pos.PieceOn[to]))
	}
}

func generateCastling(pos *board.Position, ml *board.MoveList, us board.Color, all board.Bitboard) {
	if us == board.White {
		if pos.Castling&board.WhiteKingSide != 0 {
			// Squares between king and rook must be empty: f1, g1.
			if all&(board.SquareBB(board.F1)|board.SquareBB(board.G1)) == 0 {
				// King must not be in check, pass through check, or end in check.
				if !IsSquareAttacked(pos, board.E1, board.Black) &&
					!IsSquareAttacked(pos, board.F1, board.Black) &&
					!IsSquareAttacked(pos, board.G1, board.Black) {
					ml.Add(board.NewMove(board.E1, board.G1, board.FlagKingCastle, board.King, board.NoPiece))
				}
			}
		}
		if pos.Castling&board.WhiteQueenSide != 0 {
			// Squares between: b1, c1, d1 must be empty.
			if all&(board.SquareBB(board.B1)|board.SquareBB(board.C1)|board.SquareBB(board.D1)) == 0 {
				if !IsSquareAttacked(pos, board.E1, board.Black) &&
					!IsSquareAttacked(pos, board.D1, board.Black) &&
					!IsSquareAttacked(pos, board.C1, board.Black) {
					ml.Add(board.NewMove(board.E1, board.C1, board.FlagQueenCastle, board.King, board.NoPiece))
				}
			}
		}
	} else {
		if pos.Castling&board.BlackKingSide != 0 {
			if all&(board.SquareBB(board.F8)|board.SquareBB(board.G8)) == 0 {
				if !IsSquareAttacked(pos, board.E8, board.White) &&
					!IsSquareAttacked(pos, board.F8, board.White) &&
					!IsSquareAttacked(pos, board.G8, board.White) {
					ml.Add(board.NewMove(board.E8, board.G8, board.FlagKingCastle, board.King, board.NoPiece))
				}
			}
		}
		if pos.Castling&board.BlackQueenSide != 0 {
			if all&(board.SquareBB(board.B8)|board.SquareBB(board.C8)|board.SquareBB(board.D8)) == 0 {
				if !IsSquareAttacked(pos, board.E8, board.White) &&
					!IsSquareAttacked(pos, board.D8, board.White) &&
					!IsSquareAttacked(pos, board.C8, board.White) {
					ml.Add(board.NewMove(board.E8, board.C8, board.FlagQueenCastle, board.King, board.NoPiece))
				}
			}
		}
	}
}
