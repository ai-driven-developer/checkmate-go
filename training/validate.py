"""Validate that an exported .nnue file matches the PyTorch model.

Re-implements the Go engine's quantized forward pass in Python and
compares outputs against the float model on test positions.

Usage:
    python validate.py --checkpoint models/best.pt --nnue models/best.nnue
"""

import argparse
import struct

import chess_util as chess
import numpy as np
import torch

from config import (
    INPUT_SIZE, HIDDEN_SIZE, L2_SIZE,
    QA, QB, OUTPUT_SCALE,
    MAGIC, VERSION,
    RECORD_SIZE, MAX_FEATURES, UNUSED_FEATURE,
)
from model import NNUE
from datagen import board_features


class QuantizedNetwork:
    """Quantized NNUE network loaded from .nnue binary file."""

    def __init__(self, path):
        with open(path, "rb") as f:
            magic = f.read(4)
            if magic != MAGIC:
                raise ValueError(f"Bad magic: {magic}")
            version = struct.unpack("<I", f.read(4))[0]
            if version != VERSION:
                raise ValueError(f"Bad version: {version}")

            self.ft_weight = np.frombuffer(
                f.read(INPUT_SIZE * HIDDEN_SIZE * 2), dtype=np.int16
            ).reshape(INPUT_SIZE, HIDDEN_SIZE).copy()
            self.ft_bias = np.frombuffer(
                f.read(HIDDEN_SIZE * 2), dtype=np.int16
            ).copy()
            self.l1_weight = np.frombuffer(
                f.read(2 * HIDDEN_SIZE * L2_SIZE), dtype=np.int8
            ).reshape(2 * HIDDEN_SIZE, L2_SIZE).copy()
            self.l1_bias = np.frombuffer(
                f.read(L2_SIZE * 4), dtype=np.int32
            ).copy()
            self.l2_weight = np.frombuffer(
                f.read(L2_SIZE), dtype=np.int8
            ).copy()
            self.l2_bias = struct.unpack("<i", f.read(4))[0]

    def evaluate(self, white_features, black_features, stm):
        """Quantized forward pass matching Go engine's Evaluate().

        Args:
            white_features: list of int (active white-perspective feature indices)
            black_features: list of int (active black-perspective feature indices)
            stm: 0=White, 1=Black

        Returns:
            int: evaluation in centipawns from STM perspective
        """
        # Build accumulators.
        acc_white = self.ft_bias.astype(np.int32).copy()
        acc_black = self.ft_bias.astype(np.int32).copy()

        for idx in white_features:
            acc_white += self.ft_weight[idx].astype(np.int32)
        for idx in black_features:
            acc_black += self.ft_weight[idx].astype(np.int32)

        # Perspective selection.
        if stm == 0:  # White
            us_acc, them_acc = acc_white, acc_black
        else:  # Black
            us_acc, them_acc = acc_black, acc_white

        # Hidden layer.
        hidden = self.l1_bias.astype(np.int64).copy()

        # "Us" perspective (first 256 inputs).
        for i in range(HIDDEN_SIZE):
            v = int(us_acc[i])
            v = max(0, min(v, QA))
            for j in range(L2_SIZE):
                hidden[j] += v * int(self.l1_weight[i][j])

        # "Them" perspective (next 256 inputs).
        for i in range(HIDDEN_SIZE):
            v = int(them_acc[i])
            v = max(0, min(v, QA))
            for j in range(L2_SIZE):
                hidden[j] += v * int(self.l1_weight[HIDDEN_SIZE + i][j])

        # Output layer.
        output = int(self.l2_bias)
        for j in range(L2_SIZE):
            v = int(hidden[j]) // QA
            v = max(0, min(v, QB))
            output += v * int(self.l2_weight[j])

        return output * OUTPUT_SCALE // (QB * QA)


def float_evaluate(model, white_features, black_features, stm, device):
    """Run the float PyTorch model on a single position."""
    w_idx = torch.tensor(white_features, dtype=torch.long, device=device)
    w_off = torch.tensor([0], dtype=torch.long, device=device)
    b_idx = torch.tensor(black_features, dtype=torch.long, device=device)
    b_off = torch.tensor([0], dtype=torch.long, device=device)
    stm_t = torch.tensor([stm], dtype=torch.bool, device=device)

    with torch.no_grad():
        output = model(w_idx, w_off, b_idx, b_off, stm_t)
    return output.item()


# Test positions covering various game phases and piece configurations.
TEST_FENS = [
    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
    "r1bqkbnr/pppppppp/2n5/8/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 2 2",
    "r1bqkb1r/pppppppp/2n2n2/8/2B1P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 4 3",
    "r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
    "rnbq1rk1/pppp1ppp/4pn2/8/1bPP4/2N2N2/PP2PPPP/R1BQKB1R w KQ - 4 5",
    "r1bq1rk1/ppp1bppp/2n2n2/3pp3/2PP4/2N1PN2/PP2BPPP/R1BQK2R w KQ - 2 6",
    "8/8/4k3/8/8/4K3/4P3/8 w - - 0 1",  # KPK endgame
    "8/5k2/8/8/8/8/3RK3/8 w - - 0 1",  # KRK endgame
    "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",  # Kiwi Pete
    "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10",
    "2r3k1/pp3pp1/4p2p/8/1PPn4/P4NP1/4PPKP/R7 b - - 0 20",
    "5rk1/5ppp/p7/1p6/2p5/P1P2P2/1P4PP/4R1K1 w - - 0 30",
]


def main():
    parser = argparse.ArgumentParser(description="Validate .nnue against PyTorch model")
    parser.add_argument("--checkpoint", required=True, help="Path to .pt checkpoint")
    parser.add_argument("--nnue", required=True, help="Path to .nnue binary")
    parser.add_argument("--device", default="cpu")
    args = parser.parse_args()

    device = torch.device(args.device)

    # Load float model.
    model = NNUE().to(device)
    model.load_state_dict(
        torch.load(args.checkpoint, map_location=device, weights_only=True)
    )
    model.eval()
    print(f"Loaded PyTorch model from {args.checkpoint}")

    # Load quantized network.
    qnet = QuantizedNetwork(args.nnue)
    print(f"Loaded quantized network from {args.nnue}")

    # Compare on test positions.
    errors = []
    print(f"\n{'FEN':<70s} {'Float':>8s} {'Quant':>8s} {'Diff':>8s}")
    print("-" * 98)

    for fen in TEST_FENS:
        board = chess.Board(fen)
        white_feats, black_feats = board_features(board)
        stm = 0 if board.turn == chess.WHITE else 1

        float_score = float_evaluate(model, white_feats, black_feats, stm, device)
        quant_score = qnet.evaluate(white_feats, black_feats, stm)
        diff = abs(float_score - quant_score)
        errors.append(diff)

        fen_short = fen[:68] + ".." if len(fen) > 70 else fen
        print(f"{fen_short:<70s} {float_score:8.1f} {quant_score:8d} {diff:8.1f}")

    print("-" * 98)
    print(f"Max error:  {max(errors):.1f} cp")
    print(f"Mean error: {sum(errors) / len(errors):.1f} cp")

    if max(errors) > 10:
        print("\nWARNING: Max error exceeds 10 cp — check quantization!")
    else:
        print("\nQuantization looks good.")


if __name__ == "__main__":
    main()
