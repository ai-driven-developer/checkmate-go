"""Constants and hyperparameters for NNUE training.

Architecture constants must match the Go engine exactly
(internal/nnue/features.go, internal/nnue/network.go).
"""

# ---------------------------------------------------------------------------
# Architecture (must match Go engine)
# ---------------------------------------------------------------------------
INPUT_SIZE = 768        # 2 colors * 6 piece types * 64 squares
HIDDEN_SIZE = 256       # feature transformer output per perspective
L2_SIZE = 32            # hidden layer size
QA = 255                # ClippedReLU clamp for feature transformer
QB = 64                 # ClippedReLU clamp for hidden layer
OUTPUT_SCALE = 400      # centipawn scaling factor

# ---------------------------------------------------------------------------
# Binary network format
# ---------------------------------------------------------------------------
MAGIC = b"NNUE"
VERSION = 1

# ---------------------------------------------------------------------------
# Training hyperparameters
# ---------------------------------------------------------------------------
BATCH_SIZE = 16384
LEARNING_RATE = 1e-3
LR_DROP_EPOCH = 20
LR_DROP_FACTOR = 0.1
NUM_EPOCHS = 40
WEIGHT_DECAY = 1e-6
GRAD_CLIP = 1.0
LAMBDA = 0.75           # blend: lambda*eval + (1-lambda)*wdl
EVAL_SCALE = 400.0      # sigmoid scaling for eval targets

# ---------------------------------------------------------------------------
# Data generation
# ---------------------------------------------------------------------------
ENGINE_PATH = "../checkmatego"
DATAGEN_DEPTH = 8
DATAGEN_GAMES = 10000
DATAGEN_RANDOM_PLY = 8          # random moves at game start for variety
ADJUDICATION_CP = 1000          # adjudicate when |eval| exceeds this
ADJUDICATION_COUNT = 5          # for this many consecutive moves
MAX_GAME_PLY = 300              # max plies per game

# ---------------------------------------------------------------------------
# Record format (136 bytes per sample)
# ---------------------------------------------------------------------------
RECORD_SIZE = 136
MAX_FEATURES = 32               # max active features per perspective
UNUSED_FEATURE = 0xFFFF         # padding value for unused feature slots
