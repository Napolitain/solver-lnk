# Lords and Knights Build Order Optimizer

A Python-based optimization solver for the game **Lords and Knights** using greedy heuristics and OR-Tools.

## Features

- **Resource-Aware CP-SAT Solver**: Intelligent build order with resource balance
- **Multiple Solver Options**: CP-SAT (resource-aware), CP-SAT (basic), Greedy
- **CLI with argparse**: Full command-line interface
- **Rich formatting**: Beautiful colored output with tables
- **JSON Export**: Save build plans to file
- **Building Prerequisites**: Handles dependency chains
- **Type-Safe**: Full type hints with ty (Astral)
- **Code Quality**: Enforced with ruff (Astral)
- **Data-Driven**: All building data loaded from JSON files

## Project Structure

```
solver-lnk/
├── solver_lnk/
│   ├── models/          # Data models (buildings, resources, game state)
│   ├── solvers/         # Optimization solvers (greedy, future: CP-SAT)
│   └── utils/           # Helper functions and data loaders
├── data/                # Game data files (to be added)
├── main.py              # Example usage
├── pyproject.toml       # Project configuration
└── ruff.toml            # Linter configuration
```

## Installation

Requires Python 3.13+

```bash
# Using uv (recommended)
uv sync

# Or with pip
pip install -e .
```

## Usage

```bash
# Run with default (resource-aware CP-SAT - RECOMMENDED)
uv run python main.py

# Use basic CP-SAT (faster but arbitrary build order)
uv run python main.py --solver cpsat

# Use greedy solver (will fail on complex problems)
uv run python main.py --solver greedy

# Quiet mode (only output completion time)
uv run python main.py --quiet

# Export solution to JSON
uv run python main.py --export build_plan.json

# Load custom configuration
uv run python main.py --config my_castle.json

# See all options
uv run python main.py --help
```

### Solver Comparison

| Solver | Build Order | Time | Status |
|--------|-------------|------|--------|
| `cpsat-resource` | Balanced (Lumberjack→Quarry→Ore Mine→Farm) | 709h | ✅ **RECOMMENDED** |
| `cpsat` | Arbitrary (dict order) | 675h | ⚠️ Works but poor resource balance |
| `greedy` | Priority-based | N/A | ❌ Fails on complex problems |

**Problem Types:**
- `castle-levelup` (default): Optimize building upgrade order for castle development

**Configuration JSON format:**
```json
{
  "initial_buildings": {
    "lumberjack": 0
  },
  "initial_resources": {
    "wood": 1000,
    "stone": 1000,
    "iron": 500,
    "food": 100
  },
  "target_levels": {
    "lumberjack": 15
  }
}
```

## Development

```bash
# Lint code
uv run ruff check .

# Format code
uv run ruff format .

# Type check
uv run ty check .
```

## How It Works

### Greedy Solver Strategy

1. **Priority-based**: Buildings are ranked by priority (Lumberjack → Quarry → Mine → Farm → Storage → Core → Military)
2. **Resource Simulation**: Tracks resource production and accumulation over time
3. **Wait-and-Build**: If can't afford next priority upgrade, simulates waiting for resources
4. **Sequential Execution**: One build at a time (configurable for multiple queues later)

### Priority Order

```python
LUMBERJACK:     Priority 1  # Wood production first
QUARRY:         Priority 2  # Stone second
MINE:           Priority 3  # Iron third
FARM:           Priority 4  # Food fourth
WOOD_STORE:     Priority 10 # Storage as needed
...
```

## Roadmap

- [ ] Add more building types (Quarry, Mine, Farm, Castle, etc.)
- [ ] Implement storage overflow checks
- [ ] Add prerequisite constraints (e.g., Castle level required)
- [ ] Support multiple build queues
- [ ] JSON data loader for building stats
- [ ] OR-Tools CP-SAT solver for optimal solutions
- [ ] Web interface for visualization
- [ ] Export build plans to CSV/JSON

## License

MIT
