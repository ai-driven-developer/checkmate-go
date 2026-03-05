# NNUE (Efficiently Updatable Neural Network)

Neural network-based evaluation for the CheckmateGo chess engine.

## Architecture

```
Per perspective (White & Black):
  768 input features --> Feature Transformer (768x256, int16) --> 256 neurons --> ClippedReLU

Combined (side-to-move perspective first):
  512 neurons --> Hidden Layer (512x32, int8) --> 32 neurons --> ClippedReLU
              --> Output Layer (32x1, int8)   --> 1 score (centipawns)
```

### Input features (768 per perspective)

Each feature is a binary indicator: "is there a piece of this type and color on this square?"

```
index = relativeColor * 384 + (pieceType - 1) * 64 + mappedSquare
```

- `relativeColor`: 0 = friendly, 1 = enemy (pieceColor XOR perspective)
- `pieceType`: Pawn=1, Knight=2, Bishop=3, Rook=4, Queen=5, King=6
- `mappedSquare`: square index (flipped via `sq ^ 56` for Black perspective)
- Total: 2 colors * 6 piece types * 64 squares = 768

### Quantization

| Layer               | Weights | Biases | Activations   |
|---------------------|---------|--------|---------------|
| Feature transformer | int16   | int16  | int16 (QA=255)|
| Hidden layer        | int8    | int32  | int32 (QB=64) |
| Output layer        | int8    | int32  | int32         |

Final score: `output * 400 / (QA * QB)` centipawns.

## Package structure

| File              | Purpose                                          |
|-------------------|--------------------------------------------------|
| `features.go`     | Constants (`InputSize`, `HiddenSize`, etc.) and `FeatureIndex()` |
| `network.go`      | `Network` struct, `LoadNetwork()`, `ReadNetwork()`, `Evaluate()` (forward pass) |
| `accumulator.go`  | `Accumulator`, `AccumulatorStack` with `Push`/`Pop`/`Refresh`/`AddFeature`/`SubFeature` |
| `update.go`       | `MakeMove`/`UnmakeMove`/`MakeNullMove`/`UnmakeNullMove` (incremental accumulator updates) |
| `nnue_test.go`    | 35 unit tests covering features, loading, evaluation, all move types, consistency |

## Incremental updates

The key NNUE optimization: the feature transformer layer (768x256) is not recomputed from scratch on every move. Instead:

1. At the root position, `Refresh()` computes the full accumulator by iterating all pieces.
2. On each `MakeMove`, only the changed features are updated (add/subtract weight columns).
3. On `UnmakeMove`, the previous accumulator is restored via stack pop (O(1)).

This makes evaluation O(HiddenSize) per move instead of O(InputSize * HiddenSize).

### Move type handling

| Move type          | Accumulator update                                         |
|--------------------|------------------------------------------------------------|
| Quiet / Double pawn| Sub(piece, from) + Add(piece, to)                          |
| Capture            | Sub(captured, to) + Sub(piece, from) + Add(piece, to)      |
| En passant         | Sub(pawn, capturedSq) + Sub(pawn, from) + Add(pawn, to)    |
| Kingside castle    | Move(king) + Move(rook H->F)                               |
| Queenside castle   | Move(king) + Move(rook A->D)                               |
| Promotion          | Sub(pawn, from) + Add(promoted, to) [+ Sub(captured, to)]  |
| Null move          | Push only (no feature changes)                              |

All updates are applied to both White and Black perspective accumulators simultaneously.

## Network file format

Binary, little-endian:

| Offset | Size      | Content                          |
|--------|-----------|----------------------------------|
| 0      | 4 bytes   | Magic: `NNUE`                    |
| 4      | 4 bytes   | Version: `uint32(1)`             |
| 8      | 393216 B  | Feature weights `[768][256]int16` |
| 393224 | 512 B     | Feature biases `[256]int16`       |
| 393736 | 16384 B   | Hidden weights `[512][32]int8`    |
| 410120 | 128 B     | Hidden biases `[32]int32`         |
| 410248 | 32 B      | Output weights `[32]int8`         |
| 410280 | 4 B       | Output bias `int32`               |

Total: ~400 KB.

### Generating a test network

```bash
go run ./cmd/gennet -o network.nnue -seed 42
```

This creates a network with random weights for testing the inference pipeline. A trained network is required for actual play.

## UCI integration

| Option    | Type   | Default | Description                                    |
|-----------|--------|---------|------------------------------------------------|
| `UseNNUE` | check  | true    | Enable NNUE evaluation (false = use HCE)       |
| `EvalFile`| string | (empty) | Path to `.nnue` network file                   |

If `UseNNUE` is true but no `EvalFile` is set, the engine falls back to hand-crafted evaluation (HCE).

## Search integration

The NNUE accumulator is managed per search worker (thread-local), following the same pattern as the pawn cache:

```
Engine.Search()
  for each worker thread:
    create AccumulatorStack
    Refresh() at root position
    search loop:
      w.makeMove(m)   -->  accStack.MakeMove() + pos.MakeMove()
      w.evaluate()     -->  net.Evaluate(accStack.Current(), sideToMove)
      w.unmakeMove(m)  -->  pos.UnmakeMove() + accStack.Pop()
```

## Training (not yet implemented)

To train a network, you need:

1. Generate training data: millions of positions with evaluations (from self-play with the HCE)
2. Train with a framework like PyTorch using the same architecture
3. Quantize weights to int16/int8
4. Export to the binary format described above

## Tests

```bash
go test ./internal/nnue/ -v
```

35 tests covering:
- **Feature indexing**: bounds, symmetry, uniqueness, friendly/enemy separation
- **Network loading**: roundtrip, bad magic, bad version, truncated, empty
- **Forward pass**: determinism, perspective, zero network, ClippedReLU bounds
- **Accumulator refresh**: starting position, idempotent, FEN positions
- **Incremental updates**: quiet, capture, en passant, kingside castle, queenside castle, black castle, promotion, promotion-capture, underpromotion
- **Make/unmake consistency**: quiet, capture, castle, en passant, promotion
- **Null move**: values unchanged after push/pop
- **Game sequences**: Italian Game (6 plies), 40-ply stress test
- **Stack operations**: push/pop cycle integrity
