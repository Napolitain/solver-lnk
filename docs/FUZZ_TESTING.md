# Fuzz Testing and Assertion Strategy

## Overview

This document describes the fuzz testing strategy for the solver-lnk project, specifically addressing the question raised in issue "Explore fuzz test with assertion": **Where should assertions live - in the solver or in tests?**

## Decision: Assertions in Tests, Not in Solver

After careful consideration, we've decided that **assertions should be in fuzz tests, not in the production solver code**.

### Rationale

1. **Performance**: The solver is performance-critical. Adding runtime assertions would slow it down in production.

2. **Separation of Concerns**: The solver should focus on finding optimal solutions. Validation is the test's responsibility.

3. **Flexibility**: Test assertions can be more comprehensive and verbose than runtime checks.

4. **Edge Case Discovery**: Fuzz tests with assertions are excellent at finding edge cases that violate game rules.

## Fuzz Test Categories

We have organized fuzz tests into several categories, each enforcing specific game rules:

### Phase 1: ROI and Scarcity Tests
- `FuzzROICalculation` - ROI calculations must be non-negative and finite
- `FuzzDynamicScarcity` - Scarcity multipliers must be bounded [0.5, 2.0]
- `FuzzROINonNegative` - ROI for production buildings must never be negative

### Phase 2: Resource Constraint Tests
- `FuzzResourcesNeverNegative` - Resource costs must be non-negative
- `FuzzSolverResourceConstraints` - Resources never go negative, costs are reasonable

### Phase 3: Queue Constraint Tests
- `FuzzBuildingQueueSingleItem` - Only one building upgrade at a time
- `FuzzResearchQueueSingleItem` - Only one research at a time

### Phase 4: Production & Storage Tests
- `FuzzStorageNeverExceeded` - Resources never exceed storage caps

### Phase 5: Prerequisite Tests
- `FuzzFarmResearchPrerequisites` - Farm upgrades respect tech requirements

### Phase 6: Mission Tests
- `FuzzMissionSelection` - Missions respect tavern level and unit requirements
- `FuzzMissionNoSameTypeOverlap` - Same mission never runs in parallel
- `FuzzSolverMissionNoOverlap` - No duplicate missions running

### Phase 7: End-State Tests
- `FuzzAllTargetsReached` - All building targets are reached

### Phase 8: Determinism Tests
- `FuzzDeterministicOutput` - Same inputs produce same outputs

### Phase 9: Comprehensive End-to-End Test
- `FuzzSolverEndToEnd` - **UNIFIED** Complete end-to-end validation with all invariants

## Comprehensive End-to-End Fuzz Test

The `FuzzSolverEndToEnd` test is a unified, comprehensive fuzz test that validates the entire solver flow from initial state to full targets reached. It consolidates all invariants and property assertions into a single test:

**The test validates 21 categories of game rules:**

**Correctness Assertions (Game Rules):**

1. **Ending Conditions**
   - All building targets that were set must be reached
   
2-5. **Time Progression**
   - Building queue: only one at a time, no overlaps
   - Research queue: only one at a time, no overlaps
   - Training actions: no negative durations
   - Total time must be positive and reasonable (< 200 days for most configs)
   
6. **Food Capacity**
   - Food usage never exceeds capacity at any point in time
   - All food events validated chronologically
   
7. **Library Prerequisites**
   - All researched techs must have library level requirements met when research started
   
8-10. **Resource Management**
   - Resource costs must be non-negative for all actions
   - Costs must be reasonable (not astronomically high)
   - Final resources must be non-negative
   
11-13. **Building Level Progression**
   - Building levels increase by exactly 1 per upgrade
   - FromLevel matches current state
   - ToLevel = FromLevel + 1 (no skipping levels)
   - Buildings don't exceed maximum level (30)
   - Final levels meet or exceed targets
   - All building levels are >= 1 (game default)

14. **No Duplicate Research**
   - Same technology should never be researched twice

15. **Technology Prerequisite Chains**
   - Unit training respects tech prerequisites (e.g., Longbow before Archer)

16. **Action Durations Match Data**
   - Build/research/training times match data files for each level

17. **Production Rate Consistency**
   - Production buildings provide expected production rates (non-negative)

18. **Storage Upgrades Triggered Appropriately**
   - Storage upgraded before costs would exceed capacity

**Optimality Assertions (Efficiency Checks):**

19. **Resource Balance**
   - Production buildings upgraded in reasonable ratios (logs if > 10 level difference)

20. **Parallel Queue Utilization**
   - Both building and research queues should have minimal idle time
   - Logs if building queue idle > 10% of total time
   - Logs if research queue idle > 20% of total time (more lenient)

21. **Farm Upgrades On-Demand**
   - Farm only upgraded when food capacity is actually needed
   - Logs if farm upgraded when < 50% capacity used

## Game Rules Enforced

The fuzz tests enforce these core game rules from RULES.md:

1. **Resources**:
   - Wood, Stone, Iron: Never negative, never exceed storage
   - Food: ABSOLUTE capacity (not produced), used by buildings/units
   - Costs must be non-negative and reasonable
   - Storage must be upgraded before costs exceed capacity

2. **Queues**:
   - Only one building upgrade at a time
   - Only one research at a time
   - Training and missions run in parallel
   - Minimal idle time for efficient parallel execution

3. **Prerequisites**:
   - Technology requirements must be met before research
   - Library level requirements must be met
   - Farm upgrades require specific technologies (Crop Rotation @ L15, Yoke @ L25, Cellar Storeroom @ L30)
   - Unit training requires enabling technologies

4. **Building Levels**:
   - Levels progress sequentially (no skipping)
   - Buildings default to level 1 in the game
   - Maximum level is typically 30
   - No duplicate upgrades

5. **Time**:
   - Time always progresses forward
   - No time travel or negative durations
   - Action durations match data files
   - Total time should be reasonable for the configuration

6. **Research**:
   - No duplicate research of same tech
   - Tech prerequisite chains respected

7. **Production**:
   - Production rates match building levels
   - Production buildings balanced (no extreme ratios)

8. **Ending Condition**:
   - All building targets reached
   - All possible techs researched (or blocked by food capacity)
   - All possible units recruited (or blocked by food capacity)

9. **Optimality** (informational checks):
   - Farm upgrades on-demand when capacity needed
   - Queue utilization high (minimal idle time)
   - Resource production balanced

## Running Fuzz Tests

### Seed Corpus Testing (Regression)
```bash
# Run all fuzz tests with seed corpus (fast)
go test ./internal/solver/castle -v

# Run the comprehensive end-to-end fuzz test
go test ./internal/solver/castle -run FuzzSolverEndToEnd -v
```

### Active Fuzzing (Discovery)
```bash
# Fuzz the comprehensive end-to-end test for 30 seconds
go test -fuzz=FuzzSolverEndToEnd -fuzztime=30s ./internal/solver/castle

# Fuzz for longer to discover more edge cases
go test -fuzz=FuzzSolverEndToEnd -fuzztime=5m ./internal/solver/castle
```

### Run All Quality Checks
```bash
# Comprehensive check (lint + race + fuzz corpus)
golangci-lint run && go test -race ./...

# Full test suite with fuzz discovery
./scripts/test-all.sh 60s  # Runs fuzzing for 60s each test
```

## Expected Failures

Some configurations intentionally fail to test edge cases:

1. **Tavern target not reached**: May occur if the solver determines the tavern isn't needed
2. **Very long build times**: Some extreme fuzz configurations may take 100+ days (acceptable)
3. **Tech not researched**: Optional techs may not be researched if not needed for targets

These are logged as INFO messages or acceptable test failures, not bugs.

## Benefits of This Approach

1. **Bug Detection**: Found real issues like library prerequisite violations
2. **Edge Case Discovery**: Fuzzing explores configurations we wouldn't manually test
3. **Regression Prevention**: Seed corpus ensures found issues stay fixed
4. **Documentation**: Assertions serve as executable documentation of game rules
5. **Performance**: No impact on production solver performance

## Future Work

- Extend fuzzing to cover more complex scenarios (multiple missions, large armies)
- Add fuzzing for unit training constraints
- Add fuzzing for storage capacity edge cases
- Continuous fuzzing in CI with longer durations
