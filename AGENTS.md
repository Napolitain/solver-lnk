# Agent Rules and Guidelines

## Project Status (2025-12-25)

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

### Implementation Details
**Solver**: `pkg/solver/greedy.go`
**Approach**: Simulation-based with event-driven resource accumulation
**Language**: Go 1.21+
**Status**: ✅ WORKING AND ACCURATE

## Technology Stack

- **Language**: Go 1.21+
- **CLI**: spf13/cobra
- **Tables**: olekukonko/tablewriter
- **Colors**: fatih/color
- **Testing**: Go standard library

## Development Workflow

```bash
# Build
go build -o solver ./cmd/solver/

# Run
./solver -d data

# Test
go test ./...

# Format
go fmt ./...
```

## Key Game Mechanics

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

## Project Structure

```
solver-lnk/
├── cmd/solver/          # Main entry point
├── pkg/
│   ├── models/          # Data models
│   ├── solver/          # Greedy solver + tests
│   └── loader/          # JSON loaders
├── data/                # Game data (JSON)
└── go.mod               # Go module
```
