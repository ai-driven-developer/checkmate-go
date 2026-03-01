# Changelog

## 1.4.0

### Search
- **Late move pruning (LMP):** skip quiet moves beyond a move count threshold at shallow depths (depth <= 3)
- **Reverse futility pruning:** prune nodes where static eval minus a margin already exceeds beta (depth <= 5)
- **Internal iterative reductions:** reduce depth by 1 when no hash move is available at higher depths (depth >= 4)
- **Improved time management:** better time allocation for classical and increment time controls

### Move Ordering
- **Countermove heuristic:** track the move that refuted the opponent's last move, used as an additional quiet move ordering signal

## 1.3.0

### Search
- **Singular extensions:** extend search by 1 ply when the TT move is significantly better than all alternatives (depth >= 8)
- **Static Exchange Evaluation (SEE):** full exchange sequence analysis with x-ray attack discovery, en passant and promotion support
- **SEE pruning in quiescence:** skip captures that lose material

### Move Ordering
- **SEE-aware capture ordering:** good captures (SEE >= 0) before quiet moves, losing captures last

### Evaluation
- **Pawn structure:** doubled pawn penalty, isolated pawn penalty, backward pawn penalty (separate MG/EG values)
- **King safety:** pawn shield bonus, open file penalty near king, king zone attacker pressure (quadratic scaling)
- **Rook mobility:** 3cp per available square
- **Queen mobility:** 2cp per available square
- **Bishop pair bonus:** 30cp for having both bishops

## 1.2.0

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

### Search
- **Null-move pruning:** skip a move and search with reduced depth to get a lower bound on the score
- **Killer moves:** store quiet moves that caused beta cutoffs for improved move ordering

## 1.0.0

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
