# Agent Rules and Guidelines

## Quick Reference

```bash
# Setup (first time)
git clone --recursive git@github.com:Napolitain/solver-lnk.git
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Daily workflow
go generate ./...          # Regenerate protos (if changed)
go build ./cmd/server      # Build server
go test ./...              # Run tests
./server                   # Start gRPC server
```

## Project Status (2025-12-28)

### What Works ✅
- ✅ Go greedy simulation solver with accurate resource tracking
- ✅ Continuous resource production/accumulation over time
- ✅ Storage capacity constraints enforced dynamically
- ✅ Waits for resources when insufficient
- ✅ Interleaved resource building upgrades (LJ→Q→OM per level)
- ✅ Food is ABSOLUTE capacity from Farm (not produced!)
- ✅ Dual queue system (building + research)
- ✅ Technology prerequisites (Farm 15/25/30)
- ✅ Full castle build in ~45 days (realistic)
- ✅ gRPC server for bot integration
- ✅ Deterministic output (fuzz-tested)

### Implementation Details
- **Castle Solver**: `internal/solver/castle/solver.go`
- **Units Solver**: `internal/solver/units/solver.go`
- **gRPC Server**: `cmd/server/main.go`
- **Approach**: Simulation-based with event-driven resource accumulation
- **Language**: Go 1.23+ (uses 1.25 features)

## Project Structure

```
solver-lnk/
├── cmd/
│   ├── castle/              # Castle CLI entry point
│   ├── server/              # gRPC server entry point
│   └── units/               # Units CLI entry point
├── internal/
│   ├── converter/           # Proto <-> internal models
│   ├── loader/              # JSON data loaders
│   ├── models/              # Domain models
│   └── solver/
│       ├── castle/          # Castle solver + tests
│       └── units/           # Units solver + tests
├── proto/                   # Submodule → proto-lnk
├── data/                    # Game data (JSON)
└── go.mod
```

## Technology Stack

- **Language**: Go 1.23+
- **CLI**: spf13/cobra
- **Tables**: olekukonko/tablewriter
- **Colors**: fatih/color
- **gRPC**: google.golang.org/grpc
- **Protobuf**: google.golang.org/protobuf
- **Testing**: Go standard library + fuzz tests

## Key Game Mechanics

For more details refer to RULES.md file.

1. **Resources**: Wood, Stone, Iron produced by buildings
2. **Food**: ABSOLUTE capacity from Farm - consumed by upgrades
3. **Storage**: Wood/Stone/Ore stores limit accumulation
4. **Queues**: 
   - Building queue (one at a time)
   - Research queue (Library + tech research, parallel)
5. **Priority**: Resource buildings → Farm → Storage → Core → Military

## Technology Prerequisites

- Farm Level 15 requires "Crop Rotation" research
- Farm Level 25 requires "Yoke" research
- Farm Level 30 requires "Cellar Storeroom" research

## gRPC API

The server exposes `CastleSolverService`:

```protobuf
service CastleSolverService {
  rpc Solve(SolveRequest) returns (SolveResponse);
}
```

- **Input**: Current castle state (buildings, resources, levels)
- **Output**: Recommended next action + full build plan

## Development Commands

```bash
# Proto generation (after proto-lnk changes)
go generate ./...

# Build
go build ./cmd/castle
go build ./cmd/server
go build ./cmd/units

# Test
go test ./...                    # All tests
go test -race ./...              # With race detection
go test -cover ./...             # With coverage

# Fuzz testing
go test -fuzz=FuzzSolverDeterminism -fuzztime=30s ./internal/solver/castle

# Run
./castle -d data                 # CLI solver
./server                         # gRPC server (port 50051)
```

## CI/CD

GitHub Actions runs on push/PR:
1. Checkout with submodules
2. Install protoc + plugins
3. `go generate ./...`
4. `go test -race ./...`
5. `go test -cover ./...`
6. Fuzz tests (20s each)
