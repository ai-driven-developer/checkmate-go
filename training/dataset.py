"""Training data format and DataLoader for NNUE training.

Binary record format (136 bytes, little-endian):
  [0]:       u8   n_white  (number of active white-perspective features)
  [1]:       u8   n_black  (number of active black-perspective features)
  [2]:       u8   stm      (0=White, 1=Black)
  [3]:       i8   result   (1=White wins, 0=Draw, -1=Black wins)
  [4:6]:     i16  score    (centipawns, from White's perspective)
  [6:8]:     u16  padding
  [8:72]:    u16[32]  white feature indices (0xFFFF = unused)
  [72:136]:  u16[32]  black feature indices (0xFFFF = unused)
"""

import struct

import numpy as np
import torch
from torch.utils.data import Dataset

from config import RECORD_SIZE, MAX_FEATURES, UNUSED_FEATURE


# Struct for the fixed header (6 bytes + 2 padding).
_HEADER_FMT = "<BBBbhH"
_HEADER_SIZE = struct.calcsize(_HEADER_FMT)  # 8 bytes

# Feature indices: 32 * uint16 per perspective.
_FEATURES_FMT = f"<{MAX_FEATURES}H"
_FEATURES_SIZE = struct.calcsize(_FEATURES_FMT)  # 64 bytes


class NNUEDataset(Dataset):
    """Memory-mapped dataset for packed binary training data."""

    def __init__(self, path):
        self.data = np.memmap(path, dtype=np.uint8, mode="r")
        if len(self.data) % RECORD_SIZE != 0:
            raise ValueError(
                f"File size {len(self.data)} is not a multiple of "
                f"record size {RECORD_SIZE}"
            )
        self.num_samples = len(self.data) // RECORD_SIZE

    def __len__(self):
        return self.num_samples

    def __getitem__(self, idx):
        offset = idx * RECORD_SIZE
        record = bytes(self.data[offset : offset + RECORD_SIZE])

        # Parse header.
        n_white, n_black, stm, result, score, _ = struct.unpack_from(
            _HEADER_FMT, record, 0
        )

        # Parse feature indices.
        white_raw = struct.unpack_from(_FEATURES_FMT, record, _HEADER_SIZE)
        black_raw = struct.unpack_from(
            _FEATURES_FMT, record, _HEADER_SIZE + _FEATURES_SIZE
        )

        white_indices = torch.tensor(
            [w for w in white_raw[:n_white] if w != UNUSED_FEATURE],
            dtype=torch.long,
        )
        black_indices = torch.tensor(
            [b for b in black_raw[:n_black] if b != UNUSED_FEATURE],
            dtype=torch.long,
        )

        return (
            white_indices,
            black_indices,
            bool(stm),
            float(score),
            float(result),
        )


def collate_fn(batch):
    """Custom collate for variable-length feature index lists.

    Produces flat index tensors + offsets for EmbeddingBag.
    """
    white_indices_list = []
    white_offsets = [0]
    black_indices_list = []
    black_offsets = [0]
    stm_list = []
    score_list = []
    result_list = []

    for w_idx, b_idx, stm, score, result in batch:
        white_indices_list.append(w_idx)
        white_offsets.append(white_offsets[-1] + len(w_idx))
        black_indices_list.append(b_idx)
        black_offsets.append(black_offsets[-1] + len(b_idx))
        stm_list.append(stm)
        score_list.append(score)
        result_list.append(result)

    return (
        torch.cat(white_indices_list) if white_indices_list else torch.zeros(0, dtype=torch.long),
        torch.tensor(white_offsets[:-1], dtype=torch.long),
        torch.cat(black_indices_list) if black_indices_list else torch.zeros(0, dtype=torch.long),
        torch.tensor(black_offsets[:-1], dtype=torch.long),
        torch.tensor(stm_list, dtype=torch.bool),
        torch.tensor(score_list, dtype=torch.float32),
        torch.tensor(result_list, dtype=torch.float32),
    )


class BatchedNNUEDataset(Dataset):
    """Batch-level dataset with vectorized numpy parsing.

    Each __getitem__ returns a full pre-collated batch, eliminating
    per-sample struct.unpack and Python-level collation overhead.
    The DataLoader should use batch_size=None with this dataset.
    """

    def __init__(self, path, batch_size):
        data = np.memmap(path, dtype=np.uint8, mode="r")
        if len(data) % RECORD_SIZE != 0:
            raise ValueError(
                f"File size {len(data)} is not a multiple of "
                f"record size {RECORD_SIZE}"
            )
        self.num_samples = len(data) // RECORD_SIZE
        self.batch_size = batch_size
        self.num_batches = self.num_samples // batch_size
        # 2D view: [num_samples, 136] for efficient fancy indexing.
        self.records = data.reshape(self.num_samples, RECORD_SIZE)
        # Column range for vectorized mask building.
        self._col_range = np.arange(MAX_FEATURES, dtype=np.int32)

    def __len__(self):
        return self.num_batches

    def __getitem__(self, batch_idx):
        # Random sampling (with replacement) — simple, no epoch-level shuffle.
        indices = np.random.randint(0, self.num_samples, size=self.batch_size)
        records = self.records[indices]  # [batch_size, 136] uint8, contiguous copy

        # --- Parse headers (vectorized) ---
        n_white = records[:, 0].astype(np.int32)
        n_black = records[:, 1].astype(np.int32)
        stm = records[:, 2].astype(np.bool_)
        result = records[:, 3].view(np.int8).astype(np.float32)
        score = records[:, 4:6].copy().view(np.dtype("<i2"))[:, 0].astype(np.float32)

        # --- Parse feature grids (vectorized) ---
        white_raw = records[:, 8:72].copy().view(np.dtype("<u2")).reshape(
            self.batch_size, MAX_FEATURES
        )
        black_raw = records[:, 72:136].copy().view(np.dtype("<u2")).reshape(
            self.batch_size, MAX_FEATURES
        )

        # --- Build valid-feature masks (vectorized) ---
        white_mask = self._col_range[np.newaxis, :] < n_white[:, np.newaxis]
        black_mask = self._col_range[np.newaxis, :] < n_black[:, np.newaxis]
        white_mask &= white_raw != UNUSED_FEATURE
        black_mask &= black_raw != UNUSED_FEATURE

        # --- Flat indices + offsets for EmbeddingBag (vectorized) ---
        white_counts = white_mask.sum(axis=1)
        black_counts = black_mask.sum(axis=1)

        white_offsets = np.empty(self.batch_size, dtype=np.int64)
        white_offsets[0] = 0
        np.cumsum(white_counts[:-1], out=white_offsets[1:])

        black_offsets = np.empty(self.batch_size, dtype=np.int64)
        black_offsets[0] = 0
        np.cumsum(black_counts[:-1], out=black_offsets[1:])

        white_flat = white_raw[white_mask].astype(np.int64)
        black_flat = black_raw[black_mask].astype(np.int64)

        return (
            torch.from_numpy(white_flat),
            torch.from_numpy(white_offsets),
            torch.from_numpy(black_flat),
            torch.from_numpy(black_offsets),
            torch.from_numpy(stm),
            torch.from_numpy(score),
            torch.from_numpy(result),
        )


def write_record(f, white_features, black_features, stm, score, result):
    """Write a single training record to a binary file.

    Args:
        f: file object opened in binary write mode
        white_features: list of int, active white-perspective feature indices
        black_features: list of int, active black-perspective feature indices
        stm: int, 0=White, 1=Black
        score: int, centipawns from White's perspective
        result: int, 1=White wins, 0=Draw, -1=Black wins
    """
    n_white = len(white_features)
    n_black = len(black_features)

    # Clamp score to int16 range.
    score = max(-32768, min(32767, score))

    # Header.
    f.write(struct.pack(_HEADER_FMT, n_white, n_black, stm, result, score, 0))

    # White features (padded to 32 with 0xFFFF).
    padded_w = list(white_features) + [UNUSED_FEATURE] * (MAX_FEATURES - n_white)
    f.write(struct.pack(_FEATURES_FMT, *padded_w))

    # Black features (padded to 32 with 0xFFFF).
    padded_b = list(black_features) + [UNUSED_FEATURE] * (MAX_FEATURES - n_black)
    f.write(struct.pack(_FEATURES_FMT, *padded_b))
