#!/bin/bash
# Run all tests including fuzz tests with limited duration
# Usage: ./scripts/test-all.sh [fuzz_duration]
# Default fuzz duration: 20s

set -e

FUZZ_DURATION="${1:-20s}"

echo "=== Building ===" 
go build ./...

echo ""
echo "=== Running unit tests with race detection ==="
go test -race ./...

echo ""
echo "=== Running tests with coverage ==="
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1

echo ""
echo "=== Running fuzz tests (${FUZZ_DURATION} each) ==="

# Castle solver fuzz tests
echo "  FuzzSolverDeterminism..."
go test -fuzz=FuzzSolverDeterminism -fuzztime="$FUZZ_DURATION" ./internal/solver/castle 2>&1 | tail -1

echo "  FuzzSolverResources..."
go test -fuzz=FuzzSolverResources -fuzztime="$FUZZ_DURATION" ./internal/solver/castle 2>&1 | tail -1

echo "  FuzzSolverBuildingLevels..."
go test -fuzz=FuzzSolverBuildingLevels -fuzztime="$FUZZ_DURATION" ./internal/solver/castle 2>&1 | tail -1

# Units solver fuzz tests
echo "  FuzzSolverConstraints..."
go test -fuzz=FuzzSolverConstraints -fuzztime="$FUZZ_DURATION" ./internal/solver/units 2>&1 | tail -1

echo "  FuzzUnitThroughput..."
go test -fuzz=FuzzUnitThroughput -fuzztime="$FUZZ_DURATION" ./internal/solver/units 2>&1 | tail -1

echo "  FuzzUnitResourceCosts..."
go test -fuzz=FuzzUnitResourceCosts -fuzztime="$FUZZ_DURATION" ./internal/solver/units 2>&1 | tail -1

echo ""
echo "=== All tests passed! ==="
