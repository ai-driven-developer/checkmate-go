# NNUE Training Pipeline

PyTorch-based training pipeline for the CheckmateGo NNUE network.

## Architecture

```
(768 → 256) × 2 → 32 → 1
```

- **768 input features** per perspective: 2 colors × 6 piece types × 64 squares
- **Feature transformer**: EmbeddingBag (sparse, ~32 active features per position)
- **Hidden layer**: 512 → 32 with ClippedReLU
- **Output layer**: 32 → 1 (centipawn-scale)
- **Loss**: sigmoid-MSE with eval/WDL blending (λ=0.75)

## Setup

```bash
pip install -r requirements.txt
```

## Pipeline

### 1. Build the engine

```bash
cd .. && make build
```

### 2. Generate training data

Self-play with the HCE engine to produce labeled positions:

```bash
python datagen.py --games 10000 --depth 8 --output data/training.bin
```

Options:
- `--engine PATH` — path to engine binary (default: `../checkmatego`)
- `--games N` — number of self-play games
- `--depth N` — search depth for evaluation
- `--random-ply N` — random moves at game start for opening diversity
- `--workers N` — parallel engine instances

### 3. Train the network

```bash
python train.py --data data/training.bin --epochs 40 --output models/best.nnue
```

Options:
- `--batch-size N` — batch size (default: 16384)
- `--lr FLOAT` — learning rate (default: 1e-3)
- `--lambda FLOAT` — eval/WDL blend (1.0 = pure eval, 0.0 = pure WDL)
- `--resume PATH` — resume from a `.pt` checkpoint
- `--device DEVICE` — force CPU/CUDA (auto-detected by default)

Checkpoints are saved to `models/` every 5 epochs. The best model (by loss) is saved as `models/best.pt`.

### 4. Validate quantization

Verify the exported `.nnue` matches the float model:

```bash
python validate.py --checkpoint models/best.pt --nnue models/best.nnue
```

Expect max error < 10 cp between float and quantized evaluations.

### 5. Use the trained network

```bash
../checkmatego
setoption name EvalFile value models/best.nnue
setoption name UseNNUE value true
isready
position startpos
go depth 15
```

## File overview

| File | Purpose |
|---|---|
| `config.py` | Architecture constants, hyperparameters |
| `chess_util.py` | Board representation, FEN, legal move generation (no external deps) |
| `model.py` | PyTorch NNUE model (EmbeddingBag-based) |
| `dataset.py` | Binary data format (136 bytes/record), Dataset, collate |
| `datagen.py` | Self-play data generation via UCI |
| `train.py` | Training loop (Adam, sigmoid-MSE, LR scheduling) |
| `export.py` | Quantize float model → `.nnue` binary |
| `validate.py` | Verify `.nnue` matches PyTorch model |
| `test_chess_util.py` | 56 tests for chess logic (perft, castling, en passant, etc.) |

## Quantization

The export converts float32 weights to the engine's integer format:

| Layer | Float → Quantized | Type |
|---|---|---|
| Feature transformer weights | × QA (255) | int16 |
| Feature transformer biases | × QA (255) | int16 |
| Hidden layer weights | × QB (64) | int8 |
| Hidden layer biases | × QA × QB (16320) | int32 |
| Output weights | × QA / OutputScale (0.6375) | int8 |
| Output bias | × QA × QB / OutputScale (40.8) | int32 |

## Iterative improvement

For stronger networks, iterate:

1. Train initial NNUE from HCE self-play data
2. Generate new self-play data using the NNUE engine
3. Retrain NNUE from the new (stronger) data
4. Repeat until convergence
