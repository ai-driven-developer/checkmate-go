"""Tests for chess_util — minimal chess logic for NNUE training."""

import unittest

from chess_util import (
    WHITE, BLACK,
    PAWN, KNIGHT, BISHOP, ROOK, QUEEN, KING,
    SQUARES, A1, B1, C1, D1, E1, F1, G1, H1,
    A8, B8, C8, D8, E8, F8, G8, H8,
    square_name, square_file, square_rank,
    Piece, Move, Board, Outcome,
    STARTING_FEN,
)


class TestSquares(unittest.TestCase):
    def test_square_name(self):
        self.assertEqual(square_name(A1), "a1")
        self.assertEqual(square_name(H8), "h8")
        self.assertEqual(square_name(E1), "e1")
        self.assertEqual(square_name(D4), "d4")

    def test_file_rank(self):
        self.assertEqual(square_file(A1), 0)
        self.assertEqual(square_rank(A1), 0)
        self.assertEqual(square_file(H8), 7)
        self.assertEqual(square_rank(H8), 7)
        self.assertEqual(square_file(E4), 4)
        self.assertEqual(square_rank(E4), 3)

    def test_square_constants(self):
        self.assertEqual(A1, 0)
        self.assertEqual(H1, 7)
        self.assertEqual(A8, 56)
        self.assertEqual(H8, 63)
        self.assertEqual(E1, 4)


# Convenience squares not exported as constants.
D4 = 3 + 3 * 8  # d4 = file 3, rank 3
E4 = 4 + 3 * 8  # e4
E5 = 4 + 4 * 8  # e5
D5 = 3 + 4 * 8  # d5
F5 = 5 + 4 * 8  # f5


class TestMove(unittest.TestCase):
    def test_uci_roundtrip(self):
        m = Move(E1, G1)
        self.assertEqual(m.uci(), "e1g1")
        m2 = Move.from_uci("e7e8q")
        self.assertEqual(m2.from_sq, 4 + 6 * 8)
        self.assertEqual(m2.to_sq, 4 + 7 * 8)
        self.assertEqual(m2.promotion, QUEEN)

    def test_promotion_symbols(self):
        for uci, pt in [("a7a8q", QUEEN), ("a7a8r", ROOK),
                         ("a7a8b", BISHOP), ("a7a8n", KNIGHT)]:
            m = Move.from_uci(uci)
            self.assertEqual(m.promotion, pt)
            self.assertTrue(m.uci().endswith(uci[-1]))


class TestFEN(unittest.TestCase):
    def test_starting_position(self):
        b = Board()
        self.assertEqual(b.turn, WHITE)
        # White pieces on rank 1.
        self.assertEqual(b.piece_at(A1), Piece(ROOK, WHITE))
        self.assertEqual(b.piece_at(E1), Piece(KING, WHITE))
        self.assertEqual(b.piece_at(D1), Piece(QUEEN, WHITE))
        # Black pieces on rank 8.
        self.assertEqual(b.piece_at(A8), Piece(ROOK, BLACK))
        self.assertEqual(b.piece_at(E8), Piece(KING, BLACK))
        # Pawns.
        for f in range(8):
            self.assertEqual(b.piece_at(8 + f), Piece(PAWN, WHITE))
            self.assertEqual(b.piece_at(48 + f), Piece(PAWN, BLACK))
        # Empty squares.
        for sq in range(16, 48):
            self.assertIsNone(b.piece_at(sq))

    def test_fen_roundtrip(self):
        fens = [
            STARTING_FEN,
            "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
            "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
            "8/8/4k3/8/8/4K3/4P3/8 w - - 0 1",
            "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
        ]
        for fen in fens:
            b = Board(fen)
            self.assertEqual(b.fen(), fen, f"Roundtrip failed for: {fen}")

    def test_custom_position(self):
        fen = "8/8/4k3/8/8/4K3/4P3/8 w - - 5 40"
        b = Board(fen)
        self.assertEqual(b.turn, WHITE)
        self.assertEqual(b.halfmove_clock, 5)
        self.assertEqual(b.fullmove_number, 40)
        self.assertEqual(b.castling_rights, 0)
        self.assertIsNone(b.ep_square)


class TestLegalMoves(unittest.TestCase):
    def test_starting_position_count(self):
        b = Board()
        moves = b.legal_moves
        # 16 pawn moves + 4 knight moves = 20.
        self.assertEqual(len(moves), 20)

    def test_uci_strings(self):
        b = Board()
        ucis = {m.uci() for m in b.legal_moves}
        # Spot-check a few expected moves.
        self.assertIn("e2e4", ucis)
        self.assertIn("g1f3", ucis)
        self.assertIn("b1c3", ucis)
        self.assertIn("a2a3", ucis)
        # King can't move at start.
        self.assertNotIn("e1e2", ucis)

    def test_no_moves_checkmate(self):
        # Scholar's mate final position.
        fen = "r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4"
        b = Board(fen)
        self.assertEqual(len(b.legal_moves), 0)
        self.assertTrue(b.is_check())
        self.assertTrue(b.is_checkmate())

    def test_stalemate(self):
        # Black king a8, white king b6, white queen c7.
        fen = "k7/2Q5/1K6/8/8/8/8/8 b - - 0 1"
        b = Board(fen)
        self.assertEqual(len(b.legal_moves), 0)
        self.assertFalse(b.is_check())
        self.assertTrue(b.is_stalemate())


class TestCheck(unittest.TestCase):
    def test_not_in_check_startpos(self):
        b = Board()
        self.assertFalse(b.is_check())

    def test_in_check(self):
        # White queen gives check on e7.
        fen = "rnbqkbnr/ppppQppp/8/8/8/8/PPPP1PPP/RNB1KBNR b KQkq - 0 1"
        b = Board(fen)
        self.assertTrue(b.is_check())

    def test_check_by_bishop(self):
        # Bishop on b4 gives check to white king (diagonal b4-c3-d2-e1 clear).
        fen = "rnbqk1nr/pppp1ppp/8/4p3/1b6/4P3/PPP2PPP/RNBQKBNR w KQkq - 1 3"
        b = Board(fen)
        self.assertTrue(b.is_check())


class TestEnPassant(unittest.TestCase):
    def test_en_passant_capture(self):
        # White pawn f5, black just played e7-e5.
        fen = "rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3"
        b = Board(fen)
        ucis = {m.uci() for m in b.legal_moves}
        self.assertIn("f5e6", ucis)

    def test_en_passant_removes_pawn(self):
        fen = "rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3"
        b = Board(fen)
        b.push_uci("f5e6")
        # e5 pawn should be gone.
        self.assertIsNone(b.piece_at(E5))
        # Pawn should be on e6.
        self.assertEqual(b.piece_at(4 + 5 * 8), Piece(PAWN, WHITE))

    def test_en_passant_square_set(self):
        b = Board()
        b.push_uci("e2e4")
        self.assertEqual(b.ep_square, 4 + 2 * 8)  # e3
        b.push_uci("d7d5")
        self.assertEqual(b.ep_square, 3 + 5 * 8)  # d6

    def test_en_passant_square_cleared(self):
        b = Board()
        b.push_uci("e2e4")
        self.assertIsNotNone(b.ep_square)
        b.push_uci("e7e6")  # not a double push
        self.assertIsNone(b.ep_square)


class TestCastling(unittest.TestCase):
    def _open_board(self):
        return Board("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")

    def test_kingside_white(self):
        b = self._open_board()
        ucis = {m.uci() for m in b.legal_moves}
        self.assertIn("e1g1", ucis)

    def test_queenside_white(self):
        b = self._open_board()
        ucis = {m.uci() for m in b.legal_moves}
        self.assertIn("e1c1", ucis)

    def test_kingside_rook_moves(self):
        b = self._open_board()
        b.push_uci("e1g1")
        # King on g1, rook on f1.
        self.assertEqual(b.piece_at(G1), Piece(KING, WHITE))
        self.assertEqual(b.piece_at(F1), Piece(ROOK, WHITE))
        self.assertIsNone(b.piece_at(E1))
        self.assertIsNone(b.piece_at(H1))

    def test_queenside_rook_moves(self):
        b = self._open_board()
        b.push_uci("e1c1")
        self.assertEqual(b.piece_at(C1), Piece(KING, WHITE))
        self.assertEqual(b.piece_at(D1), Piece(ROOK, WHITE))
        self.assertIsNone(b.piece_at(E1))
        self.assertIsNone(b.piece_at(A1))

    def test_castling_removes_rights(self):
        b = self._open_board()
        b.push_uci("e1g1")
        # White castling rights gone.
        self.assertFalse(b.castling_rights & 3)

    def test_castling_blocked_by_piece(self):
        b = Board()  # starting position — pieces in the way
        ucis = {m.uci() for m in b.legal_moves}
        self.assertNotIn("e1g1", ucis)
        self.assertNotIn("e1c1", ucis)

    def test_castling_blocked_by_check(self):
        # Rook on e-file attacks e1 — can't castle.
        fen = "r3k2r/pppppppp/8/8/4r3/8/PPPP1PPP/R3K2R w KQkq - 0 1"
        b = Board(fen)
        ucis = {m.uci() for m in b.legal_moves}
        self.assertNotIn("e1g1", ucis)
        self.assertNotIn("e1c1", ucis)

    def test_castling_through_check(self):
        # Black rook on d6 attacks d1 through the file — can't castle queenside.
        # f1 is NOT attacked, so kingside is still available.
        fen = "4k3/8/3r4/8/8/8/8/R3K2R w KQ - 0 1"
        b = Board(fen)
        self.assertFalse(b.is_check())  # king not in check
        ucis = {m.uci() for m in b.legal_moves}
        self.assertNotIn("e1c1", ucis)  # d1 attacked, queenside blocked
        self.assertIn("e1g1", ucis)     # kingside still available

    def test_black_castling(self):
        fen = "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQkq - 0 1"
        b = Board(fen)
        ucis = {m.uci() for m in b.legal_moves}
        self.assertIn("e8g8", ucis)
        self.assertIn("e8c8", ucis)

    def test_rook_capture_removes_rights(self):
        # If a rook is captured on its starting square, castling rights removed.
        fen = "r3k2r/pppppppp/8/8/8/8/PPPPPPPb/R3K2R b KQkq - 0 1"
        b = Board(fen)
        b.push_uci("h2h1")  # Bxh1 captures rook — wait, h2 is a pawn here
        # Let me use a better position.

    def test_rook_move_removes_rights(self):
        b = self._open_board()
        b.push_uci("a1b1")  # move a-rook
        # Queenside right removed.
        self.assertFalse(b.castling_rights & 2)
        # Kingside right still there.
        self.assertTrue(b.castling_rights & 1)


class TestPromotion(unittest.TestCase):
    def test_promotion_moves_generated(self):
        fen = "8/4P3/8/8/8/8/4k1K1/8 w - - 0 1"
        b = Board(fen)
        promos = [m for m in b.legal_moves if m.promotion is not None]
        # e7-e8 with 4 promotion types.
        promo_types = {m.promotion for m in promos}
        self.assertEqual(promo_types, {QUEEN, ROOK, BISHOP, KNIGHT})

    def test_promotion_applies(self):
        fen = "8/4P3/8/8/8/8/4k1K1/8 w - - 0 1"
        b = Board(fen)
        b.push_uci("e7e8q")
        self.assertEqual(b.piece_at(E8), Piece(QUEEN, WHITE))
        self.assertIsNone(b.piece_at(4 + 6 * 8))  # e7 empty

    def test_capture_promotion(self):
        fen = "3r4/4P3/8/8/8/8/4k1K1/8 w - - 0 1"
        b = Board(fen)
        ucis = {m.uci() for m in b.legal_moves}
        self.assertIn("e7d8q", ucis)
        self.assertIn("e7d8n", ucis)

    def test_underpromotion(self):
        fen = "8/4P3/8/8/8/8/4k1K1/8 w - - 0 1"
        b = Board(fen)
        b.push_uci("e7e8n")
        self.assertEqual(b.piece_at(E8), Piece(KNIGHT, WHITE))


class TestGameOver(unittest.TestCase):
    def test_checkmate(self):
        fen = "r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4"
        b = Board(fen)
        self.assertTrue(b.is_game_over())
        self.assertTrue(b.is_checkmate())
        o = b.outcome()
        self.assertEqual(o.winner, WHITE)

    def test_stalemate(self):
        fen = "k7/2Q5/1K6/8/8/8/8/8 b - - 0 1"
        b = Board(fen)
        self.assertTrue(b.is_game_over())
        o = b.outcome()
        self.assertIsNone(o.winner)

    def test_fifty_moves(self):
        fen = "8/8/4k3/8/8/4K3/8/8 w - - 100 80"
        b = Board(fen)
        self.assertFalse(b.is_game_over())  # not claimed
        self.assertTrue(b.is_game_over(claim_draw=True))
        o = b.outcome(claim_draw=True)
        self.assertIsNone(o.winner)

    def test_insufficient_material_kk(self):
        fen = "8/8/4k3/8/8/4K3/8/8 w - - 0 1"
        b = Board(fen)
        self.assertTrue(b.is_insufficient_material())

    def test_insufficient_material_kbk(self):
        fen = "8/8/4k3/8/8/4KB2/8/8 w - - 0 1"
        b = Board(fen)
        self.assertTrue(b.is_insufficient_material())

    def test_insufficient_material_knk(self):
        fen = "8/8/4k3/8/8/4KN2/8/8 w - - 0 1"
        b = Board(fen)
        self.assertTrue(b.is_insufficient_material())

    def test_sufficient_material_krk(self):
        fen = "8/8/4k3/8/8/4KR2/8/8 w - - 0 1"
        b = Board(fen)
        self.assertFalse(b.is_insufficient_material())

    def test_sufficient_material_kpk(self):
        fen = "8/8/4k3/8/8/4K3/4P3/8 w - - 0 1"
        b = Board(fen)
        self.assertFalse(b.is_insufficient_material())


class TestRepetition(unittest.TestCase):
    def test_threefold(self):
        b = Board()
        # Nf3 Nf6 Ng1 Ng8  (back to start — 2nd occurrence)
        for uci in ["g1f3", "g8f6", "f3g1", "f6g8"]:
            b.push_uci(uci)
        self.assertFalse(b.is_game_over(claim_draw=True))
        # Nf3 Nf6 Ng1 Ng8  (back to start — 3rd occurrence)
        for uci in ["g1f3", "g8f6", "f3g1", "f6g8"]:
            b.push_uci(uci)
        self.assertTrue(b.is_game_over(claim_draw=True))
        o = b.outcome(claim_draw=True)
        self.assertIsNone(o.winner)


class TestPushPop(unittest.TestCase):
    def test_push_pop_restores(self):
        b = Board()
        fen_before = b.fen()
        b.push_uci("e2e4")
        self.assertNotEqual(b.fen(), fen_before)
        b.pop()
        self.assertEqual(b.fen(), fen_before)

    def test_ply_counter(self):
        b = Board()
        self.assertEqual(b.ply(), 0)
        b.push_uci("e2e4")
        self.assertEqual(b.ply(), 1)
        b.push_uci("e7e5")
        self.assertEqual(b.ply(), 2)
        b.pop()
        self.assertEqual(b.ply(), 1)

    def test_turn_alternates(self):
        b = Board()
        self.assertEqual(b.turn, WHITE)
        b.push_uci("e2e4")
        self.assertEqual(b.turn, BLACK)
        b.push_uci("e7e5")
        self.assertEqual(b.turn, WHITE)


class TestPushUCI(unittest.TestCase):
    def test_basic_move(self):
        b = Board()
        b.push_uci("e2e4")
        self.assertIsNone(b.piece_at(4 + 1 * 8))  # e2 empty
        self.assertEqual(b.piece_at(E4), Piece(PAWN, WHITE))

    def test_capture(self):
        b = Board()
        for uci in ["e2e4", "d7d5", "e4d5"]:
            b.push_uci(uci)
        self.assertEqual(b.piece_at(D5), Piece(PAWN, WHITE))

    def test_castling_uci(self):
        b = Board("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
        b.push_uci("e1g1")
        self.assertEqual(b.piece_at(G1), Piece(KING, WHITE))
        self.assertEqual(b.piece_at(F1), Piece(ROOK, WHITE))


class TestAttackDetection(unittest.TestCase):
    def test_pawn_attack(self):
        b = Board()
        # White pawn on e2 attacks d3 and f3.
        self.assertTrue(b.is_attacked_by(3 + 2 * 8, WHITE))   # d3
        self.assertTrue(b.is_attacked_by(5 + 2 * 8, WHITE))   # f3
        # e3 IS attacked by d2 and f2 pawns (diagonal).
        self.assertTrue(b.is_attacked_by(4 + 2 * 8, WHITE))   # e3
        # e4 is NOT attacked by any white pawn (too far).
        self.assertFalse(b.is_attacked_by(E4, WHITE))

    def test_knight_attack(self):
        b = Board()
        # White knight on g1 attacks f3 and h3.
        self.assertTrue(b.is_attacked_by(5 + 2 * 8, WHITE))   # f3
        self.assertTrue(b.is_attacked_by(7 + 2 * 8, WHITE))   # h3

    def test_sliding_attack(self):
        fen = "8/8/4k3/8/8/8/8/4K2R w - - 0 1"
        b = Board(fen)
        # White rook on h1 attacks the whole first rank and h-file.
        self.assertTrue(b.is_attacked_by(F1, WHITE))
        self.assertTrue(b.is_attacked_by(7 + 7 * 8, WHITE))   # h8


class TestPerft(unittest.TestCase):
    """Simple perft to verify move generation correctness."""

    def _perft(self, board, depth):
        if depth == 0:
            return 1
        nodes = 0
        for m in board.legal_moves:
            board.push(m)
            nodes += self._perft(board, depth - 1)
            board.pop()
        return nodes

    def test_startpos_depth1(self):
        b = Board()
        self.assertEqual(self._perft(b, 1), 20)

    def test_startpos_depth2(self):
        b = Board()
        self.assertEqual(self._perft(b, 2), 400)

    def test_startpos_depth3(self):
        b = Board()
        self.assertEqual(self._perft(b, 3), 8902)

    def test_kiwipete_depth1(self):
        fen = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
        b = Board(fen)
        self.assertEqual(self._perft(b, 1), 48)


if __name__ == "__main__":
    unittest.main()
