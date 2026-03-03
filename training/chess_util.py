"""Minimal chess logic for NNUE training data generation.

Provides board representation, FEN parsing, legal move generation,
and game-over detection — no external dependencies.

Square mapping: LERF (Little-Endian Rank-File), same as python-chess
and the Go engine.  A1=0, B1=1, ..., H1=7, A2=8, ..., H8=63.
"""

from collections import namedtuple

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
WHITE = True
BLACK = False

# ---------------------------------------------------------------------------
# Piece types (match Go engine's board.Piece enum)
# ---------------------------------------------------------------------------
PAWN = 1
KNIGHT = 2
BISHOP = 3
ROOK = 4
QUEEN = 5
KING = 6

_PIECE_SYMBOLS = {PAWN: "p", KNIGHT: "n", BISHOP: "b",
                  ROOK: "r", QUEEN: "q", KING: "k"}
_SYMBOL_TO_PT = {v: k for k, v in _PIECE_SYMBOLS.items()}

# ---------------------------------------------------------------------------
# Squares
# ---------------------------------------------------------------------------
SQUARES = list(range(64))

A1, B1, C1, D1, E1, F1, G1, H1 = range(0, 8)
A2, B2, C2, D2, E2, F2, G2, H2 = range(8, 16)
A7, B7, C7, D7, E7, F7, G7, H7 = range(48, 56)
A8, B8, C8, D8, E8, F8, G8, H8 = range(56, 64)

_FILE_NAMES = "abcdefgh"
_RANK_NAMES = "12345678"

STARTING_FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"


def square_file(sq):
    return sq & 7


def square_rank(sq):
    return sq >> 3


def square_name(sq):
    return _FILE_NAMES[square_file(sq)] + _RANK_NAMES[square_rank(sq)]


def _square_from_name(name):
    return _FILE_NAMES.index(name[0]) + _RANK_NAMES.index(name[1]) * 8


# ---------------------------------------------------------------------------
# Piece (namedtuple with .piece_type and .color)
# ---------------------------------------------------------------------------
Piece = namedtuple("Piece", ["piece_type", "color"])

# ---------------------------------------------------------------------------
# Move
# ---------------------------------------------------------------------------
_KNIGHT_DELTAS = [(-2, -1), (-2, 1), (-1, -2), (-1, 2),
                  (1, -2), (1, 2), (2, -1), (2, 1)]
_KING_DELTAS = [(-1, -1), (-1, 0), (-1, 1), (0, -1),
                (0, 1), (1, -1), (1, 0), (1, 1)]
_BISHOP_DIRS = [(-1, -1), (-1, 1), (1, -1), (1, 1)]
_ROOK_DIRS = [(-1, 0), (1, 0), (0, -1), (0, 1)]


class Move:
    """A chess move (from_sq, to_sq, optional promotion)."""

    __slots__ = ("from_sq", "to_sq", "promotion")

    def __init__(self, from_sq, to_sq, promotion=None):
        self.from_sq = from_sq
        self.to_sq = to_sq
        self.promotion = promotion

    def uci(self):
        s = square_name(self.from_sq) + square_name(self.to_sq)
        if self.promotion:
            s += _PIECE_SYMBOLS[self.promotion]
        return s

    @staticmethod
    def from_uci(uci_str):
        from_sq = _square_from_name(uci_str[0:2])
        to_sq = _square_from_name(uci_str[2:4])
        promotion = _SYMBOL_TO_PT.get(uci_str[4]) if len(uci_str) > 4 else None
        return Move(from_sq, to_sq, promotion)

    def __repr__(self):
        return f"Move({self.uci()})"


# ---------------------------------------------------------------------------
# Outcome
# ---------------------------------------------------------------------------
class Outcome:
    """Game result.  winner is WHITE, BLACK, or None (draw)."""

    __slots__ = ("winner",)

    def __init__(self, winner):
        self.winner = winner


# ---------------------------------------------------------------------------
# Board
# ---------------------------------------------------------------------------
class Board:
    """Chess board with legal move generation and game-over detection."""

    def __init__(self, fen=None):
        self.board = [None] * 64
        self.turn = WHITE
        self.castling_rights = 0  # bitmask: 1=K, 2=Q, 4=k, 8=q
        self.ep_square = None
        self.halfmove_clock = 0
        self.fullmove_number = 1
        self._ply = 0
        self._history = []
        self._position_counts = {}
        self._set_fen(fen or STARTING_FEN)

    # ---- FEN ----

    def _set_fen(self, fen):
        parts = fen.split()
        self.board = [None] * 64
        for rank_idx, rank_str in enumerate(reversed(parts[0].split("/"))):
            f = 0
            for ch in rank_str:
                if ch.isdigit():
                    f += int(ch)
                else:
                    color = WHITE if ch.isupper() else BLACK
                    pt = _SYMBOL_TO_PT[ch.lower()]
                    self.board[rank_idx * 8 + f] = Piece(pt, color)
                    f += 1

        self.turn = WHITE if parts[1] == "w" else BLACK

        self.castling_rights = 0
        if "K" in parts[2]:
            self.castling_rights |= 1
        if "Q" in parts[2]:
            self.castling_rights |= 2
        if "k" in parts[2]:
            self.castling_rights |= 4
        if "q" in parts[2]:
            self.castling_rights |= 8

        self.ep_square = None if parts[3] == "-" else _square_from_name(parts[3])
        self.halfmove_clock = int(parts[4]) if len(parts) > 4 else 0
        self.fullmove_number = int(parts[5]) if len(parts) > 5 else 1

        self._ply = 0
        self._history = []
        self._position_counts = {}
        self._record_position()

    def fen(self):
        rows = []
        for rank in range(7, -1, -1):
            row = ""
            empty = 0
            for file in range(8):
                p = self.board[rank * 8 + file]
                if p is None:
                    empty += 1
                else:
                    if empty:
                        row += str(empty)
                        empty = 0
                    sym = _PIECE_SYMBOLS[p.piece_type]
                    row += sym.upper() if p.color == WHITE else sym
            if empty:
                row += str(empty)
            rows.append(row)

        castling = ""
        if self.castling_rights & 1:
            castling += "K"
        if self.castling_rights & 2:
            castling += "Q"
        if self.castling_rights & 4:
            castling += "k"
        if self.castling_rights & 8:
            castling += "q"

        return "{} {} {} {} {} {}".format(
            "/".join(rows),
            "w" if self.turn == WHITE else "b",
            castling or "-",
            square_name(self.ep_square) if self.ep_square is not None else "-",
            self.halfmove_clock,
            self.fullmove_number,
        )

    # ---- Queries ----

    def piece_at(self, sq):
        return self.board[sq]

    def ply(self):
        return self._ply

    def _find_king(self, color):
        for sq in SQUARES:
            p = self.board[sq]
            if p is not None and p.piece_type == KING and p.color == color:
                return sq
        return None

    # ---- Attack detection ----

    def is_attacked_by(self, sq, color):
        """Is *sq* attacked by a piece of *color*?"""
        r, f = square_rank(sq), square_file(sq)

        # Knight
        for dr, df in _KNIGHT_DELTAS:
            nr, nf = r + dr, f + df
            if 0 <= nr < 8 and 0 <= nf < 8:
                p = self.board[nr * 8 + nf]
                if p is not None and p.color == color and p.piece_type == KNIGHT:
                    return True

        # King
        for dr, df in _KING_DELTAS:
            nr, nf = r + dr, f + df
            if 0 <= nr < 8 and 0 <= nf < 8:
                p = self.board[nr * 8 + nf]
                if p is not None and p.color == color and p.piece_type == KING:
                    return True

        # Pawn  (a white pawn on (r-1, f±1) attacks sq)
        pawn_rank = r + (-1 if color == WHITE else 1)
        if 0 <= pawn_rank < 8:
            for df in (-1, 1):
                nf = f + df
                if 0 <= nf < 8:
                    p = self.board[pawn_rank * 8 + nf]
                    if p is not None and p.color == color and p.piece_type == PAWN:
                        return True

        # Sliding — bishop / queen (diagonals)
        for dr, df in _BISHOP_DIRS:
            nr, nf = r + dr, f + df
            while 0 <= nr < 8 and 0 <= nf < 8:
                p = self.board[nr * 8 + nf]
                if p is not None:
                    if p.color == color and p.piece_type in (BISHOP, QUEEN):
                        return True
                    break
                nr += dr
                nf += df

        # Sliding — rook / queen (straights)
        for dr, df in _ROOK_DIRS:
            nr, nf = r + dr, f + df
            while 0 <= nr < 8 and 0 <= nf < 8:
                p = self.board[nr * 8 + nf]
                if p is not None:
                    if p.color == color and p.piece_type in (ROOK, QUEEN):
                        return True
                    break
                nr += dr
                nf += df

        return False

    def is_check(self):
        king_sq = self._find_king(self.turn)
        return king_sq is not None and self.is_attacked_by(king_sq, not self.turn)

    # ---- Move generation ----

    def _pseudo_legal_moves(self):
        moves = []
        for sq in SQUARES:
            p = self.board[sq]
            if p is None or p.color != self.turn:
                continue
            r, f = square_rank(sq), square_file(sq)
            pt = p.piece_type
            if pt == PAWN:
                self._gen_pawn(sq, r, f, moves)
            elif pt == KNIGHT:
                self._gen_knight(sq, r, f, moves)
            elif pt == BISHOP:
                self._gen_sliding(sq, r, f, _BISHOP_DIRS, moves)
            elif pt == ROOK:
                self._gen_sliding(sq, r, f, _ROOK_DIRS, moves)
            elif pt == QUEEN:
                self._gen_sliding(sq, r, f, _BISHOP_DIRS + _ROOK_DIRS, moves)
            elif pt == KING:
                self._gen_king(sq, r, f, moves)
        return moves

    def _gen_pawn(self, sq, r, f, moves):
        direction = 1 if self.turn == WHITE else -1
        start_rank = 1 if self.turn == WHITE else 6
        promo_rank = 7 if self.turn == WHITE else 0
        to_r = r + direction

        if not (0 <= to_r < 8):
            return

        # Single push.
        to_sq = to_r * 8 + f
        if self.board[to_sq] is None:
            if to_r == promo_rank:
                for pr in (QUEEN, ROOK, BISHOP, KNIGHT):
                    moves.append(Move(sq, to_sq, pr))
            else:
                moves.append(Move(sq, to_sq))
                # Double push.
                if r == start_rank:
                    to_sq2 = (r + 2 * direction) * 8 + f
                    if self.board[to_sq2] is None:
                        moves.append(Move(sq, to_sq2))

        # Captures (including en passant).
        for df in (-1, 1):
            nf = f + df
            if not (0 <= nf < 8):
                continue
            cap_sq = to_r * 8 + nf
            target = self.board[cap_sq]
            is_capture = target is not None and target.color != self.turn
            is_ep = cap_sq == self.ep_square
            if is_capture or is_ep:
                if to_r == promo_rank:
                    for pr in (QUEEN, ROOK, BISHOP, KNIGHT):
                        moves.append(Move(sq, cap_sq, pr))
                else:
                    moves.append(Move(sq, cap_sq))

    def _gen_knight(self, sq, r, f, moves):
        for dr, df in _KNIGHT_DELTAS:
            nr, nf = r + dr, f + df
            if 0 <= nr < 8 and 0 <= nf < 8:
                to_sq = nr * 8 + nf
                t = self.board[to_sq]
                if t is None or t.color != self.turn:
                    moves.append(Move(sq, to_sq))

    def _gen_sliding(self, sq, r, f, dirs, moves):
        for dr, df in dirs:
            nr, nf = r + dr, f + df
            while 0 <= nr < 8 and 0 <= nf < 8:
                to_sq = nr * 8 + nf
                t = self.board[to_sq]
                if t is None:
                    moves.append(Move(sq, to_sq))
                else:
                    if t.color != self.turn:
                        moves.append(Move(sq, to_sq))
                    break
                nr += dr
                nf += df

    def _gen_king(self, sq, r, f, moves):
        for dr, df in _KING_DELTAS:
            nr, nf = r + dr, f + df
            if 0 <= nr < 8 and 0 <= nf < 8:
                to_sq = nr * 8 + nf
                t = self.board[to_sq]
                if t is None or t.color != self.turn:
                    moves.append(Move(sq, to_sq))

        # Castling.
        opp = not self.turn
        if self.turn == WHITE:
            if (self.castling_rights & 1 and
                    self.board[F1] is None and self.board[G1] is None and
                    not self.is_attacked_by(E1, opp) and
                    not self.is_attacked_by(F1, opp) and
                    not self.is_attacked_by(G1, opp)):
                moves.append(Move(E1, G1))
            if (self.castling_rights & 2 and
                    self.board[D1] is None and self.board[C1] is None and
                    self.board[B1] is None and
                    not self.is_attacked_by(E1, opp) and
                    not self.is_attacked_by(D1, opp) and
                    not self.is_attacked_by(C1, opp)):
                moves.append(Move(E1, C1))
        else:
            if (self.castling_rights & 4 and
                    self.board[F8] is None and self.board[G8] is None and
                    not self.is_attacked_by(E8, opp) and
                    not self.is_attacked_by(F8, opp) and
                    not self.is_attacked_by(G8, opp)):
                moves.append(Move(E8, G8))
            if (self.castling_rights & 8 and
                    self.board[D8] is None and self.board[C8] is None and
                    self.board[B8] is None and
                    not self.is_attacked_by(E8, opp) and
                    not self.is_attacked_by(D8, opp) and
                    not self.is_attacked_by(C8, opp)):
                moves.append(Move(E8, C8))

    def _is_legal(self, move):
        """Check legality by making the move and testing for king exposure."""
        saved = (
            self.board[:], self.turn, self.castling_rights,
            self.ep_square, self.halfmove_clock, self.fullmove_number,
        )
        our_color = self.turn
        self._apply(move)
        king_sq = self._find_king(our_color)
        legal = king_sq is not None and not self.is_attacked_by(king_sq, self.turn)
        (self.board, self.turn, self.castling_rights,
         self.ep_square, self.halfmove_clock, self.fullmove_number) = saved
        return legal

    @property
    def legal_moves(self):
        return [m for m in self._pseudo_legal_moves() if self._is_legal(m)]

    # ---- Making / unmaking moves ----

    def _apply(self, move):
        """Apply move to board state (no history, used by _is_legal too)."""
        p = self.board[move.from_sq]
        captured = self.board[move.to_sq]

        # En passant capture.
        if p.piece_type == PAWN and move.to_sq == self.ep_square:
            cap_sq = move.to_sq + (-8 if self.turn == WHITE else 8)
            self.board[cap_sq] = None
            captured = Piece(PAWN, not self.turn)  # for halfmove reset

        # Move piece.
        if move.promotion:
            self.board[move.to_sq] = Piece(move.promotion, p.color)
        else:
            self.board[move.to_sq] = p
        self.board[move.from_sq] = None

        # Castling rook.
        if p.piece_type == KING:
            diff = move.to_sq - move.from_sq
            if diff == 2:  # kingside
                self.board[move.from_sq + 1] = self.board[move.from_sq + 3]
                self.board[move.from_sq + 3] = None
            elif diff == -2:  # queenside
                self.board[move.from_sq - 1] = self.board[move.from_sq - 4]
                self.board[move.from_sq - 4] = None

        # Castling rights.
        if p.piece_type == KING:
            if self.turn == WHITE:
                self.castling_rights &= ~3
            else:
                self.castling_rights &= ~12
        if move.from_sq == H1 or move.to_sq == H1:
            self.castling_rights &= ~1
        if move.from_sq == A1 or move.to_sq == A1:
            self.castling_rights &= ~2
        if move.from_sq == H8 or move.to_sq == H8:
            self.castling_rights &= ~4
        if move.from_sq == A8 or move.to_sq == A8:
            self.castling_rights &= ~8

        # En passant square.
        if p.piece_type == PAWN and abs(move.to_sq - move.from_sq) == 16:
            self.ep_square = (move.from_sq + move.to_sq) // 2
        else:
            self.ep_square = None

        # Halfmove clock.
        if p.piece_type == PAWN or captured is not None:
            self.halfmove_clock = 0
        else:
            self.halfmove_clock += 1

        # Fullmove number.
        if self.turn == BLACK:
            self.fullmove_number += 1

        self.turn = not self.turn

    def push(self, move):
        """Make a move, saving state for pop()."""
        self._history.append((
            self.board[:], self.turn, self.castling_rights,
            self.ep_square, self.halfmove_clock, self.fullmove_number,
        ))
        self._apply(move)
        self._ply += 1
        self._record_position()

    def pop(self):
        """Undo the last push()."""
        key = self._position_key()
        cnt = self._position_counts.get(key, 1) - 1
        if cnt <= 0:
            self._position_counts.pop(key, None)
        else:
            self._position_counts[key] = cnt

        state = self._history.pop()
        (self.board, self.turn, self.castling_rights,
         self.ep_square, self.halfmove_clock, self.fullmove_number) = state
        self._ply -= 1

    def push_uci(self, uci_str):
        """Parse a UCI move string and push it."""
        self.push(Move.from_uci(uci_str))

    # ---- Repetition detection ----

    def _position_key(self):
        parts = []
        for sq, p in enumerate(self.board):
            if p is not None:
                parts.append((sq, p.piece_type, p.color))
        return (tuple(parts), self.turn, self.castling_rights, self.ep_square)

    def _record_position(self):
        key = self._position_key()
        self._position_counts[key] = self._position_counts.get(key, 0) + 1

    def _is_repetition(self, count=3):
        key = self._position_key()
        return self._position_counts.get(key, 0) >= count

    # ---- Game-over detection ----

    def is_checkmate(self):
        return self.is_check() and not self.legal_moves

    def is_stalemate(self):
        return not self.is_check() and not self.legal_moves

    def is_fifty_moves(self):
        return self.halfmove_clock >= 100

    def is_insufficient_material(self):
        pieces = [(p.piece_type, p.color)
                  for p in self.board if p is not None]
        if len(pieces) <= 2:
            return True  # K vs K
        if len(pieces) == 3:
            for pt, _ in pieces:
                if pt in (KNIGHT, BISHOP):
                    return True  # KN vs K or KB vs K
        if len(pieces) == 4:
            bishops = [sq for sq, p in enumerate(self.board)
                       if p is not None and p.piece_type == BISHOP]
            if len(bishops) == 2:
                # KB vs KB on same color squares.
                c0 = (square_rank(bishops[0]) + square_file(bishops[0])) % 2
                c1 = (square_rank(bishops[1]) + square_file(bishops[1])) % 2
                p0 = self.board[bishops[0]].color
                p1 = self.board[bishops[1]].color
                if p0 != p1 and c0 == c1:
                    return True
        return False

    def is_game_over(self, claim_draw=False):
        if self.is_checkmate() or self.is_stalemate():
            return True
        if claim_draw:
            if (self.is_fifty_moves() or self._is_repetition()
                    or self.is_insufficient_material()):
                return True
        return False

    def outcome(self, claim_draw=False):
        """Return Outcome if the game is over, else None."""
        if self.is_checkmate():
            return Outcome(not self.turn)  # side that just moved wins
        if self.is_stalemate():
            return Outcome(None)
        if claim_draw:
            if (self.is_fifty_moves() or self._is_repetition()
                    or self.is_insufficient_material()):
                return Outcome(None)
        return None
