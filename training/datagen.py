"""Generate NNUE training data via engine self-play.

Launches the CheckmateGo engine, plays self-play games with randomized
openings, and records (position features, eval, game result) into the
packed binary format defined in dataset.py.

Iteration 2: uses NNUE evaluation by default, adds draw adjudication,
filters noisy positions, supports opening book FENs, and tracks stats.

Usage:
    python datagen.py --games 10000 --depth 9 --output data/training_v2.bin
    python datagen.py --games 5000 --openings openings.fen --no-use-nnue
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
    DRAW_ADJUDICATION_CP,
    DRAW_ADJUDICATION_COUNT,
    SCORE_FILTER_CP,
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


def is_capture(board, uci_move):
    """Check if a UCI move string is a capture on the given board."""
    move = chess.Move.from_uci(uci_move)
    target = board.piece_at(move.to_sq)
    if target is not None and target.color != board.turn:
        return True
    # En passant.
    piece = board.piece_at(move.from_sq)
    if piece is not None and piece.piece_type == chess.PAWN:
        if move.to_sq == board.ep_square:
            return True
    return False


def load_openings(path):
    """Load opening FENs from a text file (one FEN per line)."""
    fens = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#"):
                fens.append(line)
    if not fens:
        raise ValueError(f"No openings found in {path}")
    return fens


class UCIEngine:
    """Minimal UCI engine interface for data generation."""

    def __init__(self, path, use_nnue=True, eval_file=None):
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
        if use_nnue:
            self._send("setoption name UseNNUE value true")
            if eval_file:
                self._send(f"setoption name EvalFile value {eval_file}")
        else:
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


def play_game(engine, depth, random_ply, game_id, openings=None):
    """Play one self-play game and collect training samples.

    Returns (samples, result, stats) where:
      samples: list of (white_features, black_features, stm, score)
      result: 1=White, 0=Draw, -1=Black
      stats: dict with game statistics
    """
    board = chess.Board()
    stats = {"plies": 0, "filtered": 0}

    if openings:
        # Start from a random opening position.
        fen = random.choice(openings)
        board = chess.Board(fen)
    else:
        # Random opening moves.
        for _ in range(random_ply):
            moves = list(board.legal_moves)
            if not moves:
                break
            board.push(random.choice(moves))

    samples = []
    win_adj_count = 0
    draw_adj_count = 0

    # Play game with engine evaluation.
    while not board.is_game_over(claim_draw=True) and board.ply() < MAX_GAME_PLY:
        score_cp, bestmove_uci = engine.search(board, depth)

        if score_cp is None or bestmove_uci is None:
            break

        # Record position with quality filters.
        skip = False
        if board.is_check():
            skip = True
        elif abs(score_cp) > SCORE_FILTER_CP:
            skip = True
            stats["filtered"] += 1
        elif is_capture(board, bestmove_uci):
            skip = True
            stats["filtered"] += 1

        if not skip:
            white_feats, black_feats = board_features(board)
            stm = 0 if board.turn == chess.WHITE else 1
            # Score from White's perspective.
            score_white = score_cp if board.turn == chess.WHITE else -score_cp
            samples.append((white_feats, black_feats, stm, score_white))

        # Win adjudication.
        if abs(score_cp) >= ADJUDICATION_CP:
            win_adj_count += 1
            draw_adj_count = 0
            if win_adj_count >= ADJUDICATION_COUNT:
                stm_color = chess.WHITE if board.turn == chess.WHITE else chess.BLACK
                if score_cp > 0:
                    result = 1 if stm_color == chess.WHITE else -1
                else:
                    result = -1 if stm_color == chess.WHITE else 1
                stats["plies"] = board.ply()
                return samples, result, stats
        # Draw adjudication.
        elif abs(score_cp) <= DRAW_ADJUDICATION_CP:
            draw_adj_count += 1
            win_adj_count = 0
            if draw_adj_count >= DRAW_ADJUDICATION_COUNT:
                stats["plies"] = board.ply()
                return samples, 0, stats
        else:
            win_adj_count = 0
            draw_adj_count = 0

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

    stats["plies"] = board.ply()
    return samples, result, stats


def generate_chunk(args):
    """Generate training data for a chunk of games (for multiprocessing)."""
    (engine_path, num_games, depth, random_ply, chunk_id, output_path,
     use_nnue, eval_file, openings, append) = args

    engine = UCIEngine(engine_path, use_nnue=use_nnue, eval_file=eval_file)
    total_positions = 0
    wins, draws, losses = 0, 0, 0
    total_plies = 0
    total_filtered = 0

    mode = "ab" if append else "wb"
    with open(output_path, mode) as f:
        for i in range(num_games):
            samples, result, stats = play_game(
                engine, depth, random_ply, i, openings=openings,
            )

            # Write all positions from this game with the final result.
            for white_feats, black_feats, stm, score in samples:
                write_record(f, white_feats, black_feats, stm, score, result)
                total_positions += 1

            # Track stats.
            if result == 1:
                wins += 1
            elif result == -1:
                losses += 1
            else:
                draws += 1
            total_plies += stats["plies"]
            total_filtered += stats["filtered"]

            if (i + 1) % 100 == 0:
                print(f"  [Worker {chunk_id}] {i + 1}/{num_games} games, "
                      f"{total_positions} positions", flush=True)

    engine.quit()
    return {
        "positions": total_positions,
        "wins": wins,
        "draws": draws,
        "losses": losses,
        "plies": total_plies,
        "filtered": total_filtered,
        "games": num_games,
    }


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
    parser.add_argument("--use-nnue", action="store_true", default=True,
                        help="Use NNUE evaluation (default: true)")
    parser.add_argument("--no-use-nnue", dest="use_nnue", action="store_false",
                        help="Use HCE evaluation instead of NNUE")
    parser.add_argument("--eval-file", default=None,
                        help="Path to .nnue network file (default: embedded)")
    parser.add_argument("--openings", default=None,
                        help="Path to opening FENs file (one per line)")
    parser.add_argument("--append", action="store_true",
                        help="Append to output file instead of overwriting")
    args = parser.parse_args()

    # Verify engine exists.
    if not os.path.isfile(args.engine):
        print(f"Error: engine not found at {args.engine}", file=sys.stderr)
        print(f"Build it first: cd .. && make build", file=sys.stderr)
        sys.exit(1)

    # Load openings if provided.
    openings = None
    if args.openings:
        openings = load_openings(args.openings)
        print(f"Loaded {len(openings)} opening positions")

    os.makedirs(os.path.dirname(args.output) or ".", exist_ok=True)

    eval_mode = "NNUE" if args.use_nnue else "HCE"
    print(f"Generating {args.games} games at depth {args.depth} "
          f"with {args.workers} worker(s) [{eval_mode}]")
    print(f"Engine: {args.engine}")
    if args.eval_file:
        print(f"Network: {args.eval_file}")
    print(f"Output: {args.output}{' (append)' if args.append else ''}")
    start = time.time()

    if args.workers <= 1:
        # Single-process mode.
        result = generate_chunk((
            args.engine, args.games, args.depth, args.random_ply,
            0, args.output, args.use_nnue, args.eval_file,
            openings, args.append,
        ))
        results = [result]
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
                args.use_nnue, args.eval_file, openings, False,
            ))

        with Pool(args.workers) as pool:
            results = pool.map(generate_chunk, chunk_args)

        # Merge part files.
        mode = "ab" if args.append else "wb"
        with open(args.output, mode) as out:
            for w in range(args.workers):
                chunk_path = f"{args.output}.part{w}"
                with open(chunk_path, "rb") as part:
                    while True:
                        data = part.read(1024 * 1024)
                        if not data:
                            break
                        out.write(data)
                os.remove(chunk_path)

    # Aggregate and print statistics.
    total_pos = sum(r["positions"] for r in results)
    total_wins = sum(r["wins"] for r in results)
    total_draws = sum(r["draws"] for r in results)
    total_losses = sum(r["losses"] for r in results)
    total_plies = sum(r["plies"] for r in results)
    total_filtered = sum(r["filtered"] for r in results)
    total_games = sum(r["games"] for r in results)

    elapsed = time.time() - start
    avg_ply = total_plies / max(total_games, 1)
    avg_pos = total_pos / max(total_games, 1)

    print(f"\n{'=' * 60}")
    print(f"Done: {total_pos} positions from {total_games} games "
          f"in {elapsed:.1f}s ({total_pos / max(elapsed, 1):.0f} pos/s)")
    print(f"Results: +{total_wins} ={total_draws} -{total_losses} "
          f"({total_wins / max(total_games, 1) * 100:.1f}% / "
          f"{total_draws / max(total_games, 1) * 100:.1f}% / "
          f"{total_losses / max(total_games, 1) * 100:.1f}%)")
    print(f"Avg game length: {avg_ply:.1f} plies, "
          f"avg positions/game: {avg_pos:.1f}")
    if total_filtered > 0:
        print(f"Filtered positions: {total_filtered} "
              f"(captures + extreme evals)")


if __name__ == "__main__":
    main()
