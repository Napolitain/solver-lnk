# Lords and Knights Build Order Optimizer

A Go-based optimization solver for the game **Lords and Knights** using greedy simulation heuristics. Provides both CLI tools and a gRPC server for integration with automation bots.

## Features

- **Greedy Simulation Solver**: Smart build order with resource accumulation over time
- **Dual Queue System**: Separate building queue and research queue (parallel execution)
- **Technology Prerequisites**: Library research unlocks higher building levels (e.g., Farm 15/25/30)
- **gRPC Server**: Exposes solver as a service for bot integration
- **CLI with Cobra**: Full command-line interface with flags
- **Pretty Tables**: Beautiful colored output with tablewriter
- **Data-Driven**: All building and technology data loaded from JSON files
- **Deterministic**: Same input always produces same output (fuzz-tested)

## Quick Start

### Prerequisites

- Go 1.23+ (uses Go 1.25 features)
- Protocol Buffers compiler (`protoc`)
- protoc-gen-go and protoc-gen-go-grpc plugins

### Installation

```bash
# Clone with submodules
git clone --recursive git@github.com:Napolitain/solver-lnk.git
cd solver-lnk

# Install protoc plugins (one-time)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protobuf code
go generate ./...

# Install dependencies
go mod download

# Build all binaries
go build -o castle ./cmd/castle/
go build -o server ./cmd/server/
go build -o units ./cmd/units/
```

### Running

```bash
# Run castle solver CLI
./castle -d data

# Run gRPC server (for bot-lnk integration)
./server

# Run units solver CLI  
./units -d data
```

## Project Structure

```
solver-lnk/
├── cmd/
│   ├── castle/          # Castle build order CLI
│   ├── server/          # gRPC server for bot integration
│   └── units/           # Units solver CLI
├── internal/
│   ├── converter/       # Proto <-> internal model conversion
│   ├── loader/          # JSON data loaders
│   ├── models/          # Data models (buildings, resources, tech)
│   └── solver/
│       ├── castle/      # Castle build order solver
│       └── units/       # Units recruitment solver
├── proto/               # Protobuf definitions (submodule → proto-lnk)
├── data/                # Game data files (JSON)
│   ├── buildings/       # Building upgrade costs and times
│   └── technologies/    # Technology research data
├── go.mod
└── go.sum
```

## Usage

### Castle Solver CLI

```bash
# Run with default settings
./castle -d data

# Quiet mode (minimal output)
./castle -d data --quiet

# See all options
./castle --help
```

### gRPC Server

```bash
# Start server (default port 50051)
./server

# Server listens for Solve requests from bot-lnk
```

### Units Solver CLI

```bash
./units -d data
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

### Commands Reference

```bash
# Generate protobuf (after proto changes)
go generate ./...

# Build all binaries
go build ./cmd/castle && go build ./cmd/server && go build ./cmd/units

# Lint (required before commit)
golangci-lint run

# Run tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run fuzz tests
go test -fuzz=FuzzSolverDeterminism -fuzztime=30s ./internal/solver/castle

# Format code
go fmt ./...
```

### Code Quality

This project uses `golangci-lint` for linting. Install it via:

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Always run before committing:**
```bash
golangci-lint run
go test ./...
```

### Proto Submodule

The `proto/` folder is a git submodule pointing to `proto-lnk`. To update:

```bash
cd proto
git pull origin master
cd ..
git add proto
git commit -m "chore: update proto submodule"
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
