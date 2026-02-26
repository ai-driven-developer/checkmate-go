# CheckmateGo

A UCI-compatible chess engine written in Go from scratch, with no external dependencies.

## Features

- **Board representation:** bitboard + mailbox hybrid for fast move generation and piece lookups
- **Move encoding:** compact 32-bit representation (from/to/flags/piece/captured)
- **Move generation:** magic bitboards generated at runtime; full support for castling, en passant, and promotions
- **Search:** iterative deepening with principal variation search (PVS), aspiration windows, null-move pruning, futility pruning, late move reductions, check extensions, and quiescence search with SEE pruning
- **Transposition table:** lockless hash table with depth-preferred replacement and generation aging
- **Lazy SMP:** multi-threaded search via the `Threads` UCI option
- **Move ordering:** hash move, SEE-aware capture ordering (good captures first, losing captures last), MVV-LVA, killer moves, history heuristic, promotion bonus
- **Static Exchange Evaluation (SEE):** full exchange sequence analysis with x-ray attack discovery, en passant and promotion support
- **Draw detection:** repetition detection (2-fold) and 50-move rule
- **Evaluation:** tapered evaluation (middlegame/endgame interpolation), material balance, piece-square tables, mobility, passed pawn bonus
- **Time management:** supports classical, increment, and fixed move time controls
- **UCI protocol:** full implementation including `position`, `go`, `stop`, `setoption`, `perft`, and more

## Estimated Rating

Approximate Elo ratings estimated by playing against Stockfish at various skill levels:

| Version | Elo  | Lichess Bullet | Lichess Blitz |
|---------|------|----------------|---------------|
| 1.0.0   | 1429 |                |               |
| 1.1.0   | 1443 | 1823           | 1748          |
| 1.2.0   | 1592 |                |               |

## Building

Requires Go 1.22 or later.

```
make build
```

This produces a `checkmatego` binary in the project root.

## Usage

Run the binary and communicate via the UCI protocol (stdin/stdout):

```
./checkmatego
```

Example session:

```
uci
id name CheckmateGo 1.2.0
id author ai-driven-developer
option name Hash type spin default 64 min 1 max 4096
option name Threads type spin default 1 min 1 max 128
option name Move Overhead type spin default 10 min 0 max 5000
option name SyzygyPath type string default
option name UCI_ShowWDL type check default false
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
| UCI_ShowWDL | check | false | -- | Show Win/Draw/Loss probabilities in search info |

## Testing

```
make test
```

The test suite includes 165+ tests covering:

- **board:** bitboard operations, FEN parsing, move encoding, Zobrist hashing
- **movegen:** legal move generation, capture generation, magic bitboards, perft validation (starting position through depth 5, Kiwi Pete, and other standard positions)
- **eval:** evaluation symmetry, material balance, piece-square tables, tapered evaluation, game phase, king endgame centralization, passed pawn detection and scoring
- **search:** mate-in-1, mate-in-2, stalemate avoidance, capture detection, move ordering, history heuristic, killer moves, 50-move rule, null-move pruning, futility pruning, aspiration windows, PVS, check extensions, multi-threaded search, repetition avoidance, transposition table, time control allocation, SEE (undefended captures, defended captures, equal exchanges, complex exchanges, x-ray discovery, en passant, promotions)
- **uci:** all protocol commands, option parsing (Hash, Threads, Move Overhead, SyzygyPath, UCI_ShowWDL), time control modes, move parsing with promotions and castling, WDL output

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
cmd/checkmatego/       Entry point
internal/
  board/               Position, bitboards, moves, FEN, Zobrist hashing
  movegen/             Legal move generation, magic bitboards, perft
  eval/                Tapered evaluation (material + PST + mobility + passed pawns, MG/EG interpolation)
  search/              PVS, quiescence, TT, move ordering, SEE, killer moves, history heuristic, LMR, null-move pruning, futility pruning, check extensions, aspiration windows, time control, Lazy SMP
  uci/                 UCI protocol handler and engine options
```
