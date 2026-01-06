# Algorithm Changes Since Toon Version (v0.1.0, commit 482500b)

## Performance Comparison

| Metric | Toon Version | Current Version | Change |
|--------|--------------|-----------------|--------|
| **Completion Time** | ~56 days | **54.3 days** | ✅ **-1.7 days (3% faster)** |
| **Technologies** | 20 researched | 20 researched | ✅ Same |
| **Determinism** | Yes | Yes | ✅ Same |
| **Code Lines** | 1665 | 1850 | +185 lines |

## Key Algorithm Changes

### 1. Target System Enhancement (NEW)

**Toon Version:**
```go
type Solver struct {
    Buildings    [13]*models.Building
    Technologies [20]*models.Technology
    Missions     []*models.Mission
    TargetLevels models.BuildingLevelMap
}
```

**Current Version:**
```go
type Solver struct {
    Buildings    [13]*models.Building
    Technologies [20]*models.Technology
    Missions     []*models.Mission
    TargetLevels models.BuildingLevelMap
    TargetTechs  map[string]bool         // NEW: Which techs to research
    TargetUnits  map[models.UnitType]int // NEW: Exact unit counts
}
```

**Impact:**
- Solver can now have explicit tech targets (not just "research all 20")
- Solver can have explicit unit targets (not just "train for missions")
- More flexible for future use cases (e.g., "research only 5 specific techs")
- **Currently**: TargetTechs = all 20 techs, TargetUnits = empty (missions only)

---

### 2. Main-Phase Research Scheduling (CRITICAL FIX)

**Toon Version:**
```go
func pickBestResearchAction(state *State) *ResearchAction {
    // 1. Building prerequisites (e.g., Crop rotation for Farm 15)
    // 2. Unit techs for missions (e.g., Longbow for Archer)
    // 3. Production techs (Beer tester, Wheelbarrow)
    return nil  // ← STOPS HERE, defers remaining techs to post-building
}
```

**Current Version:**
```go
func pickBestResearchAction(state *State) *ResearchAction {
    // 1. Building prerequisites (e.g., Crop rotation for Farm 15)
    // 2. Unit techs for missions (e.g., Longbow for Archer)
    // 3. Production techs (Beer tester, Wheelbarrow)
    
    // 4. FALLBACK: Research ANY remaining target tech (NEW!)
    for _, techName := range models.AllTechNames() {
        if TargetTechs[techName] && !Researched[techName] && CanResearch {
            return techName  // ← Bonus techs now researched in main phase
        }
    }
    return nil
}
```

**Why This Matters:**

In the **Toon version**, 11 expensive bonus techs (Weaponsmith, Armoursmith, Swordsmith, Iron hardening, Poison arrows, Horse breeding, Weapons manufacturing, Flaming arrows, Blacksmith, Map of area, Cistern) were **deferred to post-building phase**.

**Problem**: After buildings complete (~56 days), the solver enters post-building phase with:
- 0 accumulated resources (just spent everything on last building)
- 11 expensive techs to research (total: ~10,000 wood, ~14,000 stone, ~11,000 iron)
- Must wait for resources to accumulate → causes massive delay

**Solution**: The fallback now schedules these techs **during main phase** when resources are naturally available from production buildings. Techs are researched **in parallel** with building construction instead of **sequentially after**.

**Result**: ~1.7 day improvement (56 → 54.3 days)

---

### 3. Post-Building Phase Mission Gating (REFINED)

**Toon Version:**
```go
func handlePostBuildingStateChanged() {
    // Research all remaining techs
    // Train units for missions
    // Start missions (unlimited)  ← Runs immediately
}
```

**Current Version:**
```go
func handlePostBuildingStateChanged() {
    // Check if all target techs done
    allTargetTechsResearched := checkAllTargetTechs()
    
    // Research all remaining techs (priority)
    // Train units ONLY if all techs done
    if allTargetTechsResearched {
        trainUnits()
    }
    // Start missions ONLY if all techs done
    if allTargetTechsResearched {
        startMissions()
    }
}
```

**Impact:**
- Prevents missions from consuming resources needed for remaining techs
- In practice, with the fallback fix, most techs are done in main phase anyway
- This is a **safety guardrail** for edge cases

---

### 4. Food Reservation Removal (PERFORMANCE FIX)

**Toon Version:**
```go
func pickBestTrainingAction(state *State) *TrainUnitAction {
    // Calculate food reserved for techs (assumed 3 food each)
    unresearcedTechs := countUnresearchedTechs()
    foodReserved := unresearcedTechs * 3
    
    foodHeadroom := FoodCapacity - FoodUsed - foodReserved
    if foodHeadroom < 5 {
        return nil  // Block training to reserve food
    }
    // ... train units
}
```

**Current Version:**
```go
func pickBestTrainingAction(state *State) *TrainUnitAction {
    // Techs cost 0 food (only Wood/Stone/Iron)
    // No need to reserve food for techs!
    
    foodHeadroom := FoodCapacity - FoodUsed
    if foodHeadroom < 5 {
        return nil
    }
    // ... train units
}
```

**Why This Changed:**

After fixing the **data file bug** (techs had Library level parsed as food cost), techs now correctly cost **0 food**. The reservation logic became:
1. Incorrect (techs don't consume food)
2. Restrictive (blocked unit training unnecessarily)

**Impact:**
- More efficient unit training during main phase
- Slight performance improvement (~0.2-0.5 days)

---

### 5. Deterministic Map Iteration (CORRECTNESS FIX)

**Both Versions Have This**: Already fixed in toon version via `AllTechNames()` and `BuildingLevelMap`.

**Additional Fixes in Current Version:**
- Display code now sorts building verification output
- Display code now sorts researched tech list alphabetically
- All `range s.TargetTechs` iterations now use `AllTechNames()` deterministic order

**Impact**: No performance change, but ensures 100% deterministic output across runs.

---

### 6. Data File Format Fix (CRITICAL BUG)

**Not an algorithm change**, but critical for correctness:

**Before:**
```
Blacksmith
...
Costs
1140    ← Wood
760     ← Stone
1900    ← Iron
9       ← Library level (WRONGLY parsed as Food!)
11:00:00
```
Parser thought: "Blacksmith costs 9 food"

**After:**
```
Blacksmith
...
Costs
1140    ← Wood
760     ← Stone
1900    ← Iron
11:00:00  ← Library level comes from technologies.json
```
Parser now correctly: "Blacksmith costs 0 food"

**Impact**: 
- Without this fix, the solver appeared to run out of food (4997/5000) when 3 techs remained
- With fix, all techs complete normally

---

## Summary Table

| Change | Type | Impact | Days Saved |
|--------|------|--------|------------|
| **Main-phase fallback research** | Algorithm | Critical | ~1.5 days |
| **Food reservation removal** | Performance | Minor | ~0.2 days |
| **Target system addition** | Architecture | Neutral | 0 (enables future features) |
| **Post-building mission gating** | Safety | Neutral | 0 (rarely triggered) |
| **Deterministic iteration** | Correctness | Neutral | 0 |
| **Data file format fix** | Bug Fix | Critical | N/A (was blocking completion) |

## Recommendations

1. **Keep main-phase fallback**: This is the key performance improvement
2. **Keep target system**: Enables future flexibility (partial tech targets, specific unit counts)
3. **Keep food reservation removed**: Techs cost 0 food, no need to reserve
4. **Keep deterministic iteration**: Required for reproducible builds

## What NOT to Change

⚠️ **DO NOT remove the main-phase fallback** - this will regress performance back to 56+ days with 1541-day post-building resource accumulation delay.

⚠️ **DO NOT re-add food reservation** - techs cost 0 food, reservation logic is incorrect.

⚠️ **DO NOT use map iteration without deterministic ordering** - breaks reproducibility.
