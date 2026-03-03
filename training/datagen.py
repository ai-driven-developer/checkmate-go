"""Generate NNUE training data via engine self-play.

Launches the CheckmateGo engine, plays self-play games with randomized
openings, and records (position features, eval, game result) into the
packed binary format defined in dataset.py.

Usage:
    python datagen.py --games 10000 --depth 8 --output data/training.bin
"""

import argparse
import os
import random
import subprocess
import sys
import time
from multiprocessing import Pool, cpu_count

import chess_util as chess

from config import (
    ENGINE_PATH,
    DATAGEN_DEPTH,
    DATAGEN_GAMES,
    DATAGEN_RANDOM_PLY,
    ADJUDICATION_CP,
    ADJUDICATION_COUNT,
    MAX_GAME_PLY,
)
from dataset import write_record


def feature_index(perspective, piece_color, piece_type, sq):
    """Compute NNUE feature index (must match Go engine exactly).

    perspective: 0=White, 1=Black
    piece_color: 0=White, 1=Black
    piece_type: 1=Pawn .. 6=King
    sq: 0=A1 .. 63=H8 (LERF)
    """
    rel_color = piece_color ^ perspective
    mapped_sq = sq ^ 56 if perspective == 1 else sq
    return rel_color * 384 + (piece_type - 1) * 64 + mapped_sq


def board_features(board):
    """Extract NNUE feature indices for both perspectives.

    Returns (white_features, black_features) as lists of ints.
    """
    white_feats = []
    black_feats = []

    for sq in chess.SQUARES:
        piece = board.piece_at(sq)
        if piece is None:
            continue
        pt = piece.piece_type  # 1-6
        pc = 0 if piece.color == chess.WHITE else 1

        white_feats.append(feature_index(0, pc, pt, sq))
        black_feats.append(feature_index(1, pc, pt, sq))

    return white_feats, black_feats


class UCIEngine:
    """Minimal UCI engine interface for data generation."""

    def __init__(self, path):
        self.proc = subprocess.Popen(
            [path],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            bufsize=1,
        )
        self._send("uci")
        self._wait_for("uciok")
        # Use HCE for initial training data.
        self._send("setoption name UseNNUE value false")
        self._send("isready")
        self._wait_for("readyok")

    def _send(self, cmd):
        self.proc.stdin.write(cmd + "\n")
        self.proc.stdin.flush()

    def _wait_for(self, token):
        while True:
            line = self.proc.stdout.readline().strip()
            if line.startswith(token):
                return line

    def search(self, board, depth):
        """Search position and return (score_cp, bestmove_uci).

        Returns None for score if mate is found.
        """
        fen = board.fen()
        self._send(f"position fen {fen}")
        self._send(f"go depth {depth}")

        score_cp = None
        bestmove = None

        while True:
            line = self.proc.stdout.readline().strip()
            if not line:
                continue

            if line.startswith("info") and f"depth {depth} " in line:
                # Parse score.
                parts = line.split()
                if "score cp" in line:
                    idx = parts.index("cp")
                    score_cp = int(parts[idx + 1])
                elif "score mate" in line:
                    idx = parts.index("mate")
                    mate_in = int(parts[idx + 1])
                    # Convert mate score to large cp value.
                    score_cp = (30000 - abs(mate_in) * 10) * (1 if mate_in > 0 else -1)

            if line.startswith("bestmove"):
                bestmove = line.split()[1]
                break

        return score_cp, bestmove

    def quit(self):
        try:
            self._send("quit")
            self.proc.wait(timeout=5)
        except Exception:
            self.proc.kill()


def play_game(engine, depth, random_ply, game_id):
    """Play one self-play game and collect training samples.

    Returns list of (white_features, black_features, stm, score) tuples
    and the game result (1=White, 0=Draw, -1=Black).
    """
    board = chess.Board()
    samples = []
    adj_count = 0

    # Random opening moves.
    for _ in range(random_ply):
        moves = list(board.legal_moves)
        if not moves:
            break
        board.push(random.choice(moves))

    # Play game with engine evaluation.
    while not board.is_game_over(claim_draw=True) and board.ply() < MAX_GAME_PLY:
        score_cp, bestmove_uci = engine.search(board, depth)

        if score_cp is None or bestmove_uci is None:
            break

        # Record position (skip positions in check for cleaner data).
        if not board.is_check():
            white_feats, black_feats = board_features(board)
            stm = 0 if board.turn == chess.WHITE else 1
            # Score from White's perspective.
            score_white = score_cp if board.turn == chess.WHITE else -score_cp
            samples.append((white_feats, black_feats, stm, score_white))

        # Check adjudication.
        if abs(score_cp) >= ADJUDICATION_CP:
            adj_count += 1
            if adj_count >= ADJUDICATION_COUNT:
                # Adjudicate: winner is the side with positive eval.
                stm_color = chess.WHITE if board.turn == chess.WHITE else chess.BLACK
                if score_cp > 0:
                    result = 1 if stm_color == chess.WHITE else -1
                else:
                    result = -1 if stm_color == chess.WHITE else 1
                return samples, result
        else:
            adj_count = 0

        # Play the best move.
        try:
            board.push_uci(bestmove_uci)
        except ValueError:
            break

    # Determine game result.
    outcome = board.outcome(claim_draw=True)
    if outcome is None:
        result = 0  # max ply reached -> draw
    elif outcome.winner == chess.WHITE:
        result = 1
    elif outcome.winner == chess.BLACK:
        result = -1
    else:
        result = 0

    return samples, result


def generate_chunk(args):
    """Generate training data for a chunk of games (for multiprocessing)."""
    engine_path, num_games, depth, random_ply, chunk_id, output_path = args

    engine = UCIEngine(engine_path)
    total_positions = 0

    with open(output_path, "wb") as f:
        for i in range(num_games):
            samples, result = play_game(engine, depth, random_ply, i)

            # Write all positions from this game with the final result.
            for white_feats, black_feats, stm, score in samples:
                write_record(f, white_feats, black_feats, stm, score, result)
                total_positions += 1

            if (i + 1) % 100 == 0:
                print(f"  [Worker {chunk_id}] {i + 1}/{num_games} games, "
                      f"{total_positions} positions", flush=True)

    engine.quit()
    return total_positions


def main():
    parser = argparse.ArgumentParser(description="Generate NNUE training data")
    parser.add_argument("--engine", default=ENGINE_PATH,
                        help="Path to engine binary")
    parser.add_argument("--games", type=int, default=DATAGEN_GAMES,
                        help="Number of self-play games")
    parser.add_argument("--depth", type=int, default=DATAGEN_DEPTH,
                        help="Search depth for evaluation")
    parser.add_argument("--random-ply", type=int, default=DATAGEN_RANDOM_PLY,
                        help="Random moves at game start")
    parser.add_argument("--output", default="data/training.bin",
                        help="Output binary file path")
    parser.add_argument("--workers", type=int, default=1,
                        help="Number of parallel workers")
    args = parser.parse_args()

    # Verify engine exists.
    if not os.path.isfile(args.engine):
        print(f"Error: engine not found at {args.engine}", file=sys.stderr)
        print(f"Build it first: cd .. && make build", file=sys.stderr)
        sys.exit(1)

    os.makedirs(os.path.dirname(args.output) or ".", exist_ok=True)

    print(f"Generating {args.games} games at depth {args.depth} "
          f"with {args.workers} worker(s)")
    print(f"Engine: {args.engine}")
    print(f"Output: {args.output}")
    start = time.time()

    if args.workers <= 1:
        # Single-process mode.
        total = generate_chunk((
            args.engine, args.games, args.depth, args.random_ply,
            0, args.output,
        ))
    else:
        # Multi-process: each worker writes to a temp file, then merge.
        games_per_worker = args.games // args.workers
        remainder = args.games % args.workers

        chunk_args = []
        for w in range(args.workers):
            n = games_per_worker + (1 if w < remainder else 0)
            chunk_path = f"{args.output}.part{w}"
            chunk_args.append((
                args.engine, n, args.depth, args.random_ply, w, chunk_path,
            ))

        with Pool(args.workers) as pool:
            results = pool.map(generate_chunk, chunk_args)

        total = sum(results)

        # Merge part files.
        with open(args.output, "wb") as out:
            for w in range(args.workers):
                chunk_path = f"{args.output}.part{w}"
                with open(chunk_path, "rb") as part:
                    while True:
                        data = part.read(1024 * 1024)
                        if not data:
                            break
                        out.write(data)
                os.remove(chunk_path)

    elapsed = time.time() - start
    print(f"\nDone: {total} positions from {args.games} games "
          f"in {elapsed:.1f}s ({total / max(elapsed, 1):.0f} pos/s)")


if __name__ == "__main__":
    main()
