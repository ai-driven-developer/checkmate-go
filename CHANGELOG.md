# Changelog

## 1.2.0

**Estimated Elo: 1592** (+149 over 1.1.0)

### Search
- **Principal Variation Search (PVS):** first move searched with full window, remaining moves with zero-window and re-search on fail high
- **Aspiration windows:** narrow search window around previous iteration's score (delta 25, exponential widening on fail)
- **Futility pruning:** skip quiet moves at depth <= 2 when static eval + margin is below alpha
- **Check extensions:** extend search by 1 ply when a move gives check
- **Late move reductions (LMR):** reduce depth for quiet late moves at depth >= 3

### Move Ordering
- **History heuristic:** track quiet moves that cause beta cutoffs, use accumulated scores to order quiet moves

### Evaluation
- **Tapered evaluation:** interpolate between middlegame and endgame scores based on remaining non-pawn material (phase 0-24)
- **King endgame PST:** separate piece-square table for the king in endgames, rewarding centralization


## 1.1.0

**Estimated Elo: 1443** (+14 over 1.0.0)

### Search
- **Null-move pruning:** skip a move and search with reduced depth to get a lower bound on the score
- **Killer moves:** store quiet moves that caused beta cutoffs for improved move ordering

## 1.0.0

**Estimated Elo: 1429**

Initial release.

### Board
- Bitboard + mailbox hybrid board representation
- Compact 32-bit move encoding
- Zobrist hashing for positions

### Move Generation
- Magic bitboards generated at runtime
- Full support for castling, en passant, and promotions

### Search
- Iterative deepening with alpha-beta pruning
- Quiescence search for tactical stability
- Transposition table with lockless concurrent access
- Repetition detection (2-fold) and 50-move rule
- Lazy SMP multi-threaded search

### Move Ordering
- Hash move priority
- MVV-LVA for captures
- Promotion bonus

### Evaluation
- Material balance
- Piece-square tables
- Knight and bishop mobility

### UCI Protocol
- Full UCI implementation
- Configurable hash size, threads, and move overhead
- Time management for classical, increment, and fixed time controls
