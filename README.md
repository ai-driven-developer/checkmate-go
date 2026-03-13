# CheckmateGo

A UCI-compatible chess engine written in Go from scratch, with no external dependencies.

## Features

- **Board representation:** bitboard + mailbox hybrid for fast move generation and piece lookups, incremental pawn Zobrist hash, incremental piece-square table scores
- **Move encoding:** compact 32-bit representation (from/to/flags/piece/captured)
- **Move generation:** magic bitboards generated at runtime; full support for castling, en passant, and promotions
- **Search:** iterative deepening with principal variation search (PVS), aspiration windows, null-move pruning, ProbCut, internal iterative reductions (IIR), reverse futility pruning, razoring, futility pruning, late move pruning, late move reductions, SEE pruning (quiet moves and captures), improving heuristic, check extensions, singular extensions, and quiescence search with SEE pruning and delta pruning
- **Transposition table:** lockless hash table with depth-preferred replacement and generation aging
- **Lazy SMP:** multi-threaded search via the `Threads` UCI option
- **Move ordering:** hash move, SEE-aware capture ordering (good captures first, losing captures last), MVV-LVA, killer moves, countermove heuristic, history heuristic with gravity (bonus/malus), continuation history (1-ply and 2-ply), promotion bonus
- **Static Exchange Evaluation (SEE):** full exchange sequence analysis with x-ray attack discovery, en passant and promotion support
- **Draw detection:** repetition detection (2-fold) and 50-move rule
- **NNUE evaluation:** efficiently updatable neural network (768→256)×2→32→1 with int16/int8 quantized weights, incremental accumulator updates on make/unmake, per-worker accumulator stacks, ClippedReLU activations; supports all move types (quiet, captures, en passant, castling, promotions); toggled via `UseNNUE` UCI option with HCE fallback
- **Hand-crafted evaluation (HCE):** tapered evaluation (middlegame/endgame interpolation), material balance, bishop pair bonus, piece-square tables, mobility (knight/bishop/rook/queen), knight outposts (pawn-supported, unassailable), rook bonuses (open/semi-open files, 7th rank), passed pawn bonus with king-passer distance (friendly king proximity, enemy king distance), pawn structure (doubled/isolated/backward pawn penalties), king safety (pawn shield, open file penalty, king zone attacker pressure), per-worker pawn hash table for caching pawn evaluation
- **Ponder:** thinks on the opponent's time; the engine predicts the opponent's reply and searches ahead, transitioning seamlessly to normal search on `ponderhit`; toggled via `Ponder` option
- **Time management:** adaptive soft/hard time limits with move stability detection, score-drop extension, and time-aware move allocation (bullet-optimized); supports classical, increment, and fixed move time controls
- **UCI protocol:** full implementation including `position`, `go`, `stop`, `ponderhit`, `setoption`, `perft`, and more

## Lichess Rating

Plays as a bot on Lichess: [checkmatego-bot](https://lichess.org/@/checkmatego-bot)

| Version | Bullet | Blitz |
|---------|--------|-------|
| 1.1.0   | 1823   | 1748  |
| 1.2.0   | 2105   | 2108  |
| 1.3.0   | 2212   | 2252  |
| 1.4.0   | 2327   | 2344  |
| 1.5.0   | 2412   | 2418  |

## Building

Requires Go 1.22 or later.

```
make build
```

This produces a `checkmatego` binary in the project root.

### Build with embedded NNUE

To embed the NNUE network into the binary (no external file needed at runtime):

```
make build-nnue
```

Without this flag, NNUE can still be used by providing a network file via the `EvalFile` UCI option.

### Build with SIMD

To build with architecture-specific SIMD optimizations (AVX2 on AMD64, NEON on ARM64):

```
make build-simd
```

Or with embedded NNUE and SIMD together:

```
make nnue-simd
```

By default, the engine uses pure Go implementations. The `simd` build tag enables optimized assembly routines.

## Usage

Run the binary and communicate via the UCI protocol (stdin/stdout):

```
./checkmatego
```

Example session:

```
uci
id name CheckmateGo 1.5.0
id author ai-driven-developer
option name Hash type spin default 64 min 1 max 4096
option name Threads type spin default 1 min 1 max 128
option name Move Overhead type spin default 10 min 0 max 5000
option name SyzygyPath type string default
option name Ponder type check default false
option name UCI_ShowWDL type check default false
option name UseNNUE type check default true
option name EvalFile type string default <embedded>
uciok

isready
readyok

position startpos
go depth 10
info depth 1 score cp 44 nodes 31 nps 31000 time 1 pv e2e4
...
bestmove e2e4

quit
```

## UCI Options

| Option | Type | Default | Range | Description |
|---|---|---|---|---|
| Hash | spin | 64 | 1 -- 4096 | Hash table size in MB |
| Threads | spin | 1 | 1 -- 128 | Number of search threads (Lazy SMP) |
| Move Overhead | spin | 10 | 0 -- 5000 | Time reserved for communication overhead (ms) |
| SyzygyPath | string | *(empty)* | -- | Path to Syzygy endgame tablebases (not yet implemented) |
| Ponder | check | false | -- | Enable pondering (thinking on opponent's time) |
| UCI_ShowWDL | check | false | -- | Show Win/Draw/Loss probabilities in search info |
| UseNNUE | check | true | -- | Use NNUE evaluation (false = hand-crafted evaluation) |
| EvalFile | string | `<embedded>` | -- | Path to NNUE network file (`.nnue`); `<embedded>` uses the built-in network (requires `build-nnue`) |

## Testing

```
make test
```

The test suite includes 350+ tests covering:

- **board:** bitboard operations, FEN parsing, move encoding, Zobrist hashing, pawn hash incremental consistency, incremental PST consistency (quiet moves, captures, castling, promotions, en passant)
- **movegen:** legal move generation, capture generation, magic bitboards, perft validation (starting position through depth 5, Kiwi Pete, and other standard positions)
- **eval:** evaluation symmetry, material balance, piece-square tables, tapered evaluation, game phase, king endgame centralization, knight outposts (protected, unsupported, attackable, rank filtering, symmetry), rook evaluation (open file, semi-open file, closed file, 7th rank, symmetry), king-passer distance (friendly king close, enemy king far, symmetry, endgame-only), passed pawn detection and scoring, pawn structure (doubled, isolated, backward pawns), king safety (pawn shield, open files, attacker pressure), pawn cache (probe/store, overwrite, cache-vs-no-cache consistency, hit verification), incremental PST vs from-scratch consistency
- **search:** mate-in-1, mate-in-2, stalemate avoidance, capture detection, move ordering, history heuristic (gravity), killer moves, countermove heuristic, continuation history (update/lookup, malus, gravity bounds, null-move safety, independent entries, move scoring integration, LMR integration), 50-move rule, null-move pruning, ProbCut (node reduction, tactics safety, mate safety, depth gating), razoring (node reduction, tactics safety, mate safety, depth gating), IIR, reverse futility pruning, futility pruning, late move pruning, SEE pruning in main search (quiet moves, captures, promotion safety, mate safety), delta pruning, improving heuristic, aspiration windows, PVS, check extensions, history-aware LMR, multi-threaded search, repetition avoidance, transposition table, node limit, ponder (indefinite search, ponderhit transition, stop during ponder, depth/node limits, multi-threaded), time management (soft/hard limits, move stability, score-drop extension, adaptive allocation, classical/increment/movetime, ponder mode), SEE (undefended captures, defended captures, equal exchanges, complex exchanges, x-ray discovery, en passant, promotions)
- **nnue:** feature index computation (bounds, symmetry, uniqueness, friendly/enemy), network loading (roundtrip, bad magic/version, truncated), forward pass (determinism, perspective, zero network, ClippedReLU bounds), accumulator refresh (starting position, idempotent, FEN), incremental updates for all move types (quiet, capture, en passant, kingside/queenside castling, promotion, promotion-capture, underpromotion), make/unmake consistency, null move handling, full game sequences (Italian Game, 40-ply stress test), stack operations
- **uci:** all protocol commands, option parsing (Hash, Threads, Move Overhead, SyzygyPath, Ponder, UCI_ShowWDL, UseNNUE, EvalFile), time control modes, move parsing with promotions and castling, WDL output, ponder (go ponder, ponderhit, bestmove with ponder move, full game sequence)

## Benchmarks

Run perft benchmarks:

```
make bench
```

Run perft from the engine itself:

```
make perft
```

## Project Structure

```
cmd/
  checkmatego/         Entry point
  gennet/              NNUE test network generator
internal/
  board/               Position, bitboards, moves, FEN, Zobrist hashing, incremental PST
  movegen/             Legal move generation, magic bitboards, perft
  eval/                HCE: tapered evaluation (material + PST + mobility + outposts + rooks + passed pawns + king-passer distance + pawn structure + king safety), per-worker pawn hash table
  nnue/                NNUE: network loading, forward pass, incremental accumulator, feature indexing
  search/              PVS, quiescence, TT, move ordering, SEE, SEE pruning (main search), killer moves, countermove heuristic, history heuristic (gravity), continuation history (1-ply/2-ply), LMR (history-aware), null-move pruning, ProbCut, IIR, reverse futility pruning, razoring, futility pruning, late move pruning, delta pruning, improving heuristic, check extensions, aspiration windows, time control, ponder, Lazy SMP
  uci/                 UCI protocol handler and engine options
```
