"""NNUE training loop.

Trains the NNUE model using sigmoid-MSE loss with WDL blending,
the standard approach from Stockfish NNUE training.

Usage:
    python train.py --data data/training.bin --epochs 40 --output models/best.nnue
"""

import argparse
import os
import time

import torch
from torch.utils.data import DataLoader

from config import (
    BATCH_SIZE,
    LEARNING_RATE,
    LR_DROP_EPOCH,
    LR_DROP_FACTOR,
    NUM_EPOCHS,
    WEIGHT_DECAY,
    GRAD_CLIP,
    LAMBDA,
    EVAL_SCALE,
)
from model import NNUE
from dataset import NNUEDataset, collate_fn
from export import export_network


def loss_fn(model_output, score_target, result_target, lambda_):
    """Sigmoid-MSE loss with eval/WDL blending.

    Args:
        model_output: [batch] raw model output (centipawn-scale, STM POV)
        score_target: [batch] target eval (centipawns, STM POV)
        result_target: [batch] game result (-1, 0, 1; STM POV)
    """
    # Sigmoid of model output and target score.
    model_sig = torch.sigmoid(model_output / EVAL_SCALE)
    target_sig = torch.sigmoid(score_target / EVAL_SCALE)

    # Game result mapped to [0, 1]: -1 -> 0, 0 -> 0.5, 1 -> 1.
    result_01 = (result_target + 1.0) / 2.0

    # Blended target.
    blended = lambda_ * target_sig + (1.0 - lambda_) * result_01

    # MSE loss.
    return torch.mean((model_sig - blended) ** 2)


def train_epoch(model, loader, optimizer, device, lambda_):
    """Train for one epoch. Returns average loss."""
    model.train()
    total_loss = 0.0
    num_batches = 0

    for batch in loader:
        w_idx, w_off, b_idx, b_off, stm, score, result = [
            x.to(device) for x in batch
        ]

        # Forward pass.
        output = model(w_idx, w_off, b_idx, b_off, stm).squeeze(1)

        # Flip score and result to STM perspective.
        # Data stores scores from White's POV; when Black is STM, flip sign.
        stm_float = stm.float()
        score_stm = score * (1.0 - 2.0 * stm_float)
        result_stm = result * (1.0 - 2.0 * stm_float)

        loss = loss_fn(output, score_stm, result_stm, lambda_)

        optimizer.zero_grad()
        loss.backward()
        torch.nn.utils.clip_grad_norm_(model.parameters(), GRAD_CLIP)
        optimizer.step()

        total_loss += loss.item()
        num_batches += 1

    return total_loss / max(num_batches, 1)


def main():
    parser = argparse.ArgumentParser(description="Train NNUE network")
    parser.add_argument("--data", required=True, help="Training data .bin file")
    parser.add_argument("--epochs", type=int, default=NUM_EPOCHS)
    parser.add_argument("--batch-size", type=int, default=BATCH_SIZE)
    parser.add_argument("--lr", type=float, default=LEARNING_RATE)
    parser.add_argument("--lambda", dest="lambda_", type=float, default=LAMBDA,
                        help="Eval/WDL blend factor (1=pure eval, 0=pure WDL)")
    parser.add_argument("--output", default="models/best.nnue",
                        help="Output .nnue path")
    parser.add_argument("--resume", default=None,
                        help="Resume from .pt checkpoint")
    parser.add_argument("--device", default=None,
                        help="Device (auto-detected if not set)")
    args = parser.parse_args()

    # Device selection.
    if args.device:
        device = torch.device(args.device)
    elif torch.cuda.is_available():
        device = torch.device("cuda")
    else:
        device = torch.device("cpu")
    print(f"Device: {device}")

    # Dataset.
    dataset = NNUEDataset(args.data)
    print(f"Training data: {len(dataset)} positions")

    loader = DataLoader(
        dataset,
        batch_size=args.batch_size,
        shuffle=True,
        num_workers=min(4, os.cpu_count() or 1),
        collate_fn=collate_fn,
        pin_memory=(device.type == "cuda"),
        drop_last=True,
    )

    # Model.
    model = NNUE().to(device)
    if args.resume:
        model.load_state_dict(
            torch.load(args.resume, map_location=device, weights_only=True)
        )
        print(f"Resumed from {args.resume}")

    total_params = sum(p.numel() for p in model.parameters())
    print(f"Model parameters: {total_params:,}")

    # Optimizer and scheduler.
    optimizer = torch.optim.Adam(
        model.parameters(), lr=args.lr, weight_decay=WEIGHT_DECAY
    )
    scheduler = torch.optim.lr_scheduler.StepLR(
        optimizer, step_size=LR_DROP_EPOCH, gamma=LR_DROP_FACTOR
    )

    # Directories.
    os.makedirs("models", exist_ok=True)

    # Training loop.
    print(f"\nTraining for {args.epochs} epochs, batch size {args.batch_size}, "
          f"lambda={args.lambda_}")
    print("-" * 60)

    best_loss = float("inf")

    for epoch in range(args.epochs):
        t0 = time.time()
        avg_loss = train_epoch(model, loader, optimizer, device, args.lambda_)
        scheduler.step()
        elapsed = time.time() - t0

        lr = scheduler.get_last_lr()[0]
        print(f"Epoch {epoch + 1:3d}/{args.epochs}  "
              f"loss={avg_loss:.6f}  lr={lr:.2e}  time={elapsed:.1f}s")

        # Checkpoint every 5 epochs.
        if (epoch + 1) % 5 == 0:
            path = f"models/epoch_{epoch + 1}.pt"
            torch.save(model.state_dict(), path)

        # Track best.
        if avg_loss < best_loss:
            best_loss = avg_loss
            torch.save(model.state_dict(), "models/best.pt")

    # Final save and export.
    torch.save(model.state_dict(), "models/final.pt")
    print(f"\nBest loss: {best_loss:.6f}")

    model.eval()
    export_network(model, args.output)
    print(f"Exported quantized network to {args.output}")


if __name__ == "__main__":
    main()
