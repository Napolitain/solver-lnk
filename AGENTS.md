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
golangci-lint run          # Lint code
go test ./...              # Run tests
./server                   # Start gRPC server
```

## Project Status (2025-12-29)

### What Works ✅
- ✅ Go greedy simulation solver with accurate resource tracking
- ✅ Continuous resource production/accumulation over time
- ✅ Storage capacity constraints enforced dynamically
- ✅ Waits for resources when insufficient
- ✅ Interleaved resource building upgrades (LJ→Q→OM per level)
- ✅ Food is ABSOLUTE capacity from Farm (not produced!)
- ✅ Dual queue system (building + research)
- ✅ Technology prerequisites (Farm 15/25/30)
- ✅ Full castle build in ~60 days (with all techs)
- ✅ Unit training with resource costs
- ✅ Unified build order (buildings + research + units)
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

# Lint (required before commit)
golangci-lint run

# Test
go test ./...                    # All tests (includes fuzz seed corpus)
go test -race ./...              # With race detection
go test -cover ./...             # With coverage

# Run
./castle -d data                 # CLI solver
./server                         # gRPC server (port 50051)
```

## Testing

### Standard Tests

```bash
go test ./...                    # Run all tests
go test -race ./...              # With race detection
go test -coverprofile=coverage.out ./...  # With coverage
go tool cover -func=coverage.out # Show coverage report
```

### Fuzz Tests

`go test ./...` runs fuzz tests against their seed corpus (regression tests), but does NOT run the fuzzing engine with random generation. To actually fuzz:

**Castle solver fuzz tests (`internal/solver/castle/`):**
```bash
go test -fuzz=FuzzSolverDeterminism -fuzztime=30s ./internal/solver/castle
go test -fuzz=FuzzSolverResources -fuzztime=30s ./internal/solver/castle
go test -fuzz=FuzzSolverBuildingLevels -fuzztime=30s ./internal/solver/castle
```

**Units solver fuzz tests (`internal/solver/units/`):**
```bash
go test -fuzz=FuzzSolverConstraints -fuzztime=30s ./internal/solver/units
go test -fuzz=FuzzUnitThroughput -fuzztime=30s ./internal/solver/units
go test -fuzz=FuzzUnitResourceCosts -fuzztime=30s ./internal/solver/units
```

### Run All Quality Checks (before commit)

```bash
golangci-lint run && go test -race ./...
```

## CI/CD

GitHub Actions runs on push/PR:
1. Checkout with submodules
2. Install protoc + plugins
3. `go generate ./...`
4. `golangci-lint run`
5. `go test -race ./...`
6. `go test -coverprofile=coverage.out ./...`
7. Fuzz tests (20s each):
   - `FuzzSolverDeterminism` (castle)
   - `FuzzSolverResources` (castle)
   - `FuzzSolverBuildingLevels` (castle)
   - `FuzzSolverConstraints` (units)
   - `FuzzUnitThroughput` (units)
8. Upload coverage to Codecov

## Philosophy

**Always favor state-of-the-art, edge tooling and practices.** This project aims to use the latest stable versions of all tools and follow modern idioms for 2025+:

- Go 1.25+ with latest language features
- Fuzz testing for robustness
- `go generate` for code generation
- `internal/` package layout
- gRPC for service communication
