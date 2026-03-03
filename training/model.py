"""PyTorch NNUE model: (768->256)x2 -> 32 -> 1.

Uses EmbeddingBag for efficient sparse feature transformer computation.
Only ~32 of 768 features are active per position (~4% density).
"""

import torch
import torch.nn as nn

from config import INPUT_SIZE, HIDDEN_SIZE, L2_SIZE


class NNUE(nn.Module):
    """NNUE network with perspective-based feature transformer."""

    def __init__(self):
        super().__init__()
        # Feature transformer (shared weights, applied per perspective).
        # EmbeddingBag sums the weight rows for active feature indices.
        self.ft = nn.EmbeddingBag(INPUT_SIZE, HIDDEN_SIZE, mode="sum",
                                  sparse=False)
        self.ft_bias = nn.Parameter(torch.zeros(HIDDEN_SIZE))

        # Hidden layer: 512 (2*256 concatenated) -> 32
        self.l1 = nn.Linear(2 * HIDDEN_SIZE, L2_SIZE)

        # Output layer: 32 -> 1
        self.l2 = nn.Linear(L2_SIZE, 1)

        self._init_weights()

    def _init_weights(self):
        # Small init for feature transformer (accumulates ~32 rows).
        nn.init.uniform_(self.ft.weight, -0.05, 0.05)
        nn.init.zeros_(self.ft_bias)
        nn.init.kaiming_normal_(self.l1.weight, nonlinearity="relu")
        nn.init.zeros_(self.l1.bias)
        nn.init.kaiming_normal_(self.l2.weight, nonlinearity="linear")
        nn.init.zeros_(self.l2.bias)

    def forward(self, white_indices, white_offsets, black_indices,
                black_offsets, stm):
        """Forward pass.

        Args:
            white_indices: flat LongTensor of active white-perspective features
            white_offsets: LongTensor of per-sample start offsets
            black_indices: flat LongTensor of active black-perspective features
            black_offsets: LongTensor of per-sample start offsets
            stm: BoolTensor [batch], True if Black to move

        Returns:
            Tensor [batch, 1] — raw score (centipawn-scale)
        """
        # Feature transformer: sum weight rows for active features, add bias.
        white_acc = self.ft(white_indices, white_offsets) + self.ft_bias
        black_acc = self.ft(black_indices, black_offsets) + self.ft_bias

        # ClippedReLU [0, 1] (maps to [0, QA] in quantized domain).
        white_acc = torch.clamp(white_acc, 0.0, 1.0)
        black_acc = torch.clamp(black_acc, 0.0, 1.0)

        # Concatenate: side-to-move perspective first.
        stm_col = stm.unsqueeze(1)  # [batch, 1]
        us = torch.where(stm_col, black_acc, white_acc)
        them = torch.where(stm_col, white_acc, black_acc)
        combined = torch.cat([us, them], dim=1)  # [batch, 512]

        # Hidden layer with ClippedReLU [0, 1].
        hidden = torch.clamp(self.l1(combined), 0.0, 1.0)

        # Output layer (no activation).
        return self.l2(hidden)
