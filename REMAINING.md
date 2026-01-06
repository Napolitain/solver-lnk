# Remaining Tasks

## ✅ Target System - COMPLETED

**Status**: Fully implemented and tested (v0.2.0)

The solver now supports comprehensive target states across three dimensions:
- **Buildings**: Target levels (e.g., Lumberjack 30, Farm 30)
- **Technologies**: List of techs to research (defaults to all available)
- **Units**: Exact counts per unit type (defaults to missions only)

### Usage

**CLI defaults** (backward compatible):
```bash
./castle -d data  # Builds to max levels, researches all techs, trains for missions
```

**Protobuf configuration** (for gRPC clients):
```protobuf
message TargetState {
  repeated BuildingLevel building_targets = 1;
  repeated Technology technology_targets = 2;  // Empty = all
  repeated UnitCount unit_targets = 3;         // Empty = missions only
}
```

### Semantics
- **Completion**: Build order completes when ALL targets are reached
- **Technologies**: Empty list = research all available (default)
- **Units**: Empty list = train for missions only (default)
- **Priority**: Event-driven ROI-based (unchanged)

---

## Unit Training Batching Implementation

**Status:** Infrastructure complete (batching support added), batch size logic not yet implemented.

### Current State
✅ TrainUnitAction now has Count field (defaults to 1)  
✅ Costs() and Duration() multiply by Count  
✅ handleTrainingComplete adds Count units to army  
✅ Backward compatible (Count=0 treated as 1)  
✅ Golden test passes (build order unchanged)  

### Problem
Currently trains units one at a time, generating **3449 individual training events** in a full castle build. This floods the event queue and makes the simulation slower than necessary.

### Goal
Reduce training events to ~50-100 batches by intelligently grouping units:
- Mission units: Batch up to deficit or resource limits
- Defense units: Only train during dead time (no ROI actions available)
- Maximum batch size: 30 units
- Zero delay tolerance for buildings/research (ROI priority)

### Implementation Tasks

#### 1. Simple Batch Size Calculation
**File:** `internal/solver/castle/solver.go`

Add function to calculate safe batch size:
```go
func (s *Solver) calculateBatchSize(state *State, unitType models.UnitType, deficit int) int {
    const maxBatch = 30
    
    // Start with deficit or max
    size := deficit
    if size > maxBatch {
        size = maxBatch
    }
    
    // Constrain by food
    def := models.GetUnitDefinition(unitType)
    maxByFood := (state.FoodCapacity - state.FoodUsed) / def.FoodCost
    if maxByFood < size {
        size = maxByFood
    }
    
    // Constrain by resources
    costs := def.ResourceCosts
    maxByWood := int(state.GetResource(models.Wood)) / costs.Wood
    maxByStone := int(state.GetResource(models.Stone)) / costs.Stone
    maxByIron := int(state.GetResource(models.Iron)) / costs.Iron
    
    size = min(size, maxByWood, maxByStone, maxByIron)
    
    // Check storage pressure (if >80%, prefer larger batches)
    pressure := s.calculateStoragePressure(state)
    if pressure < 0.8 && size > 3 {
        size = 3 // Small batches when not near cap
    }
    
    if size < 1 {
        size = 1
    }
    
    return size
}
```

#### 2. Update pickBestTrainingAction
Update `pickBestTrainingAction` to call `calculateBatchSize` and set Count:
```go
return &TrainUnitAction{
    UnitType:   d.unitType,
    Definition: d.def,
    Count:      s.calculateBatchSize(state, d.unitType, d.deficit),
}
```

#### 3. Update pickNextTrainingAction
The post-building phase also trains units - update it similarly.

#### 4. Testing & Validation
- Run golden test - hash WILL change (batching changes order)
- Update golden hash in test
- Verify event count reduction: check len(trainingActions) in result
- Benchmark with `poop` - should be similar or faster
- Run full test suite: `go test ./...`

### Design Constraints
- **Never delay ROI actions**: Check if any building/research is affordable before training defense
- **Storage pressure threshold**: 80% utilization triggers larger batches
- **Mission units priority**: Train for missions first (up to tavern 10)
- **Defense units**: Only after missions complete AND tavern >= 10
- **Minimum batch**: Always train at least 1 unit if deficit exists

### Testing Checklist
- [ ] Implement calculateBatchSize helper
- [ ] Update pickBestTrainingAction with batching
- [ ] Update pickNextTrainingAction with batching  
- [ ] Run golden test and update hash
- [ ] Verify event count: `len(result.TrainingActions)` should be ~50-100 instead of 3449
- [ ] Benchmark: `poop -d 10s './castle -d data --silent'`
- [ ] Full test suite: `go test ./...`
- [ ] Commit with message: "feat: Implement unit batching to reduce event queue"

### Notes
- Avoid complex simulations in calculateBatchSize (caused infinite loops in previous attempt)
- Keep batch logic simple: resource constraints + storage pressure only
- Count=0 defaults to 1 in all methods (safety check already in place)
- Focus on mission units first - defense batching can be refined later
