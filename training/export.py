"""Quantize a trained PyTorch NNUE model and export to binary .nnue format.

The binary format must exactly match the Go engine's ReadNetwork()
in internal/nnue/network.go.

Quantization mapping (float -> quantized):
  ft_weight:  round(float * QA)           -> int16  [768][256]
  ft_bias:    round(float * QA)           -> int16  [256]
  l1_weight:  round(float * QB)           -> int8   [512][32]
  l1_bias:    round(float * QA * QB)      -> int32  [32]
  l2_weight:  round(float * QA / OS)      -> int8   [32]
  l2_bias:    round(float * QA * QB / OS) -> int32  [1]
"""

import argparse
import struct

import numpy as np
import torch

from config import (
    INPUT_SIZE, HIDDEN_SIZE, L2_SIZE,
    QA, QB, OUTPUT_SCALE,
    MAGIC, VERSION,
)
from model import NNUE


def quantize(model):
    """Extract and quantize weights from a float PyTorch model.

    Returns a dict of numpy arrays in the quantized types.
    """
    with torch.no_grad():
        ft_weight = model.ft.weight.data.cpu().numpy()       # [768, 256]
        ft_bias = model.ft_bias.data.cpu().numpy()            # [256]
        l1_weight = model.l1.weight.data.cpu().numpy().T      # Linear stores [out, in] -> transpose to [512, 32]
        l1_bias = model.l1.bias.data.cpu().numpy()            # [32]
        l2_weight = model.l2.weight.data.cpu().numpy().T      # [32, 1] -> flatten to [32]
        l2_bias = model.l2.bias.data.cpu().numpy()            # [1]

    # Feature transformer: float [0,1] range -> int16 scaled by QA.
    ft_weight_q = np.clip(
        np.round(ft_weight * QA), -32768, 32767
    ).astype(np.int16)
    ft_bias_q = np.clip(
        np.round(ft_bias * QA), -32768, 32767
    ).astype(np.int16)

    # Hidden layer weights: float -> int8 scaled by QB.
    l1_weight_q = np.clip(
        np.round(l1_weight * QB), -128, 127
    ).astype(np.int8)

    # Hidden layer biases: float -> int32 scaled by QA * QB.
    l1_bias_q = np.clip(
        np.round(l1_bias * QA * QB),
        np.iinfo(np.int32).min, np.iinfo(np.int32).max,
    ).astype(np.int32)

    # Output weights: float -> int8 scaled by QA / OutputScale.
    l2_weight_flat = l2_weight.flatten()
    l2_weight_q = np.clip(
        np.round(l2_weight_flat * QA / OUTPUT_SCALE), -128, 127
    ).astype(np.int8)

    # Output bias: float -> int32 scaled by QA * QB / OutputScale.
    l2_bias_q = int(np.clip(
        np.round(l2_bias[0] * QA * QB / OUTPUT_SCALE),
        np.iinfo(np.int32).min, np.iinfo(np.int32).max,
    ))

    return {
        "ft_weight": ft_weight_q,   # [768, 256] int16
        "ft_bias": ft_bias_q,       # [256] int16
        "l1_weight": l1_weight_q,   # [512, 32] int8
        "l1_bias": l1_bias_q,       # [32] int32
        "l2_weight": l2_weight_q,   # [32] int8
        "l2_bias": l2_bias_q,       # int32 scalar
    }


def write_network(path, q):
    """Write quantized weights to the binary .nnue format.

    Binary layout (little-endian):
      magic "NNUE"  (4 bytes)
      version 1     (uint32)
      ft_weight     [768][256] int16  (393216 bytes)
      ft_bias       [256] int16      (512 bytes)
      l1_weight     [512][32] int8   (16384 bytes)
      l1_bias       [32] int32       (128 bytes)
      l2_weight     [32] int8        (32 bytes)
      l2_bias       int32            (4 bytes)
    """
    with open(path, "wb") as f:
        # Header.
        f.write(MAGIC)
        f.write(struct.pack("<I", VERSION))

        # Weights in C-order (row-major), matching Go's array layout.
        f.write(q["ft_weight"].tobytes())
        f.write(q["ft_bias"].tobytes())
        f.write(q["l1_weight"].tobytes())
        f.write(q["l1_bias"].tobytes())
        f.write(q["l2_weight"].tobytes())
        f.write(struct.pack("<i", q["l2_bias"]))

    total = 4 + 4 + (INPUT_SIZE * HIDDEN_SIZE * 2) + (HIDDEN_SIZE * 2) + \
            (2 * HIDDEN_SIZE * L2_SIZE) + (L2_SIZE * 4) + L2_SIZE + 4
    print(f"Wrote {path} ({total} bytes)")


def export_network(model, path):
    """Quantize model and write to .nnue file."""
    q = quantize(model)
    write_network(path, q)


def main():
    parser = argparse.ArgumentParser(description="Export PyTorch NNUE to binary .nnue")
    parser.add_argument("--checkpoint", required=True, help="Path to .pt checkpoint")
    parser.add_argument("--output", default="network.nnue", help="Output .nnue path")
    args = parser.parse_args()

    model = NNUE()
    model.load_state_dict(torch.load(args.checkpoint, map_location="cpu", weights_only=True))
    model.eval()

    export_network(model, args.output)


if __name__ == "__main__":
    main()
