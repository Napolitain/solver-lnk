# Lords and Knights Build Order Optimizer

A Go-based optimization solver for the game **Lords and Knights** using greedy simulation heuristics.

## Features

- **Greedy Simulation Solver**: Smart build order with resource accumulation over time
- **Dual Queue System**: Separate building queue and research queue (parallel execution)
- **Technology Prerequisites**: Library research unlocks higher building levels (e.g., Farm 15/25/30)
- **CLI with Cobra**: Full command-line interface with flags
- **Pretty Tables**: Beautiful colored output with tablewriter
- **Data-Driven**: All building and technology data loaded from JSON files
- **Tested**: Includes unit tests for constraint validation

## Project Structure

```
solver-lnk/
├── cmd/solver/          # Main entry point
├── pkg/
│   ├── models/          # Data models (buildings, resources, technologies)
│   ├── solver/          # Greedy simulation solver
│   └── loader/          # JSON data loaders
├── data/                # Game data files
│   ├── buildings/       # Building upgrade costs and times
│   └── technologies/    # Technology research data
├── go.mod               # Go module configuration
└── go.sum               # Dependency checksums
```

## Installation

Requires Go 1.21+

```bash
# Build the solver
go build -o solver ./cmd/solver/

# Or run directly
go run ./cmd/solver/
```

## Usage

```bash
# Run with default settings
./solver -d data

# Quiet mode (minimal output)
./solver -d data --quiet

# See all options
./solver --help
```

### Example Output

```
╭───────────────────────────╮
│  Lords and Knights        │
│  Build Order Optimizer    │
│  (Go Version)             │
╰───────────────────────────╯

📦 Loaded 13 buildings, 3 technologies

🔄 Solving...

✓ Found solution with 255 building upgrades and 3 research tasks!

┌─────┬─────────────┬──────────────────┬─────────┬────────────┬────────────┬──────────┬─────────────────────────────┐
│  #  │    QUEUE    │      ACTION      │ UPGRADE │   START    │    END     │ DURATION │            COSTS            │
├─────┼─────────────┼──────────────────┼─────────┼────────────┼────────────┼──────────┼─────────────────────────────┤
│ 1   │ 🏗️ Building │ Lumberjack       │ 1 → 2   │ 00:00:00   │ 00:06:00   │ 00:06:00 │ W:   31 S:   26 I:   7 F: 2 │
│ 2   │ 🏗️ Building │ Quarry           │ 1 → 2   │ 00:06:00   │ 00:11:00   │ 00:05:00 │ W:   20 S:   25 I:  12 F: 1 │
...
```

## How It Works

### Greedy Simulation Strategy

1. **Priority-based**: Buildings are ranked by priority:
   - Lumberjack (wood production) → Quarry (stone) → Ore Mine (iron)
   - Storage buildings when capacity needed
   - Core buildings (Keep, Library) and military last

2. **Resource Simulation**: Tracks resource production and accumulation over real time
   - Production rates based on building levels
   - Storage capacity limits enforced

3. **Wait-and-Build**: If can't afford next priority upgrade, waits for resources to accumulate

4. **Technology Prerequisites**: 
   - Farm Level 15 requires "Crop Rotation" research
   - Farm Level 25 requires "Yoke" research  
   - Farm Level 30 requires "Cellar Storeroom" research

### Dual Queue System

The game has two parallel construction queues:
- **Building Queue**: All regular buildings (can only build one at a time)
- **Research Queue**: Library upgrades + Technology research (independent from buildings)

## Development

```bash
# Run tests
go test ./...

# Build
go build -o solver ./cmd/solver/

# Run
./solver -d data
```

## Roadmap

- [x] Greedy simulation solver with resource accumulation
- [x] Storage capacity constraints
- [x] Farm-only-when-needed logic
- [x] Technology prerequisites (Library research)
- [x] Dual queue system (building + research)
- [ ] Export build plans to JSON
- [ ] Custom target configurations
- [ ] Web interface for visualization
- [ ] CP-SAT solver for optimal solutions

## License

MIT
