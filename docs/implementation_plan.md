# Implementation Plan: Unified Castle + Units + Missions Solver

## Overview

Integrate unit training and tavern missions into the castle solver using an **event-driven architecture** for correctness and clean state management.

---

## Architecture

### Event-Driven Solver

```
┌─────────────────────────────────────────────────────────────┐
│                      Event Queue                            │
│  (sorted by Time, then Priority)                            │
├─────────────────────────────────────────────────────────────┤
│ Time=0    StateChanged (bootstrap)                          │
│ Time=100  MissionComplete (priority=0)                      │
│ Time=100  BuildingComplete (priority=1)                     │
│ Time=100  StateChanged (priority=10) ← re-evaluate          │
│ Time=250  TrainingComplete                                  │
│ ...                                                         │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Main Loop                                │
│                                                             │
│  1. Pop next event                                          │
│  2. Advance time (accumulate resources)                     │
│  3. Process event (update state)                            │
│  4. If completion → push StateChanged event                 │
│  5. If StateChanged → try start actions on free queues      │
│  6. Repeat until targets reached                            │
└─────────────────────────────────────────────────────────────┘
```

### Event Types & Priority

| Priority | Event Type | Description |
|----------|------------|-------------|
| 0 | MissionComplete | Adds resources, returns units |
| 1 | BuildingComplete | Updates production/storage |
| 2 | ResearchComplete | Unlocks techs/units |
| 3 | TrainingComplete | Adds unit to army |
| 10 | StateChanged | Re-evaluate all queues (dummy event) |

**Key Invariant**: Completions always process before StateChanged at same time.

### Four Parallel Queues

| Queue | Location | Constraint |
|-------|----------|------------|
| Building | Keep | 1 at a time |
| Research | Library | 1 at a time |
| Training | Arsenal | 1 unit at a time |
| Missions | Tavern | Limited by available units |

---

## Data Structures

### Strict Typing (No Maps for Counts)

```go
// internal/models/army.go

type Army struct {
    Spearman    int
    Swordsman   int
    Archer      int
    Crossbowman int
    Horseman    int
    Lancer      int
    Handcart    int
    Oxcart      int
}

func (a *Army) Get(ut UnitType) int { ... }
func (a *Army) Add(ut UnitType, count int) { ... }
func (a *Army) Remove(ut UnitType, count int) { ... }
func (a *Army) CanSatisfy(reqs []UnitRequirement) bool { ... }
func (a *Army) TotalFood() int { ... }
```

### Event Queue

```go
// internal/solver/v4/events.go

type EventType int

const (
    EventMissionComplete EventType = iota
    EventBuildingComplete
    EventResearchComplete
    EventTrainingComplete
    EventStateChanged
)

type Event struct {
    Time    int
    Type    EventType
    Payload any
}

func (e Event) Priority() int {
    switch e.Type {
    case EventMissionComplete:  return 0
    case EventBuildingComplete: return 1
    case EventResearchComplete: return 2
    case EventTrainingComplete: return 3
    case EventStateChanged:     return 10
    }
    return 99
}

type EventQueue struct {
    events []Event
}

func (q *EventQueue) Push(e Event) { ... }      // Insert sorted
func (q *EventQueue) Pop() Event { ... }         // Remove first
func (q *EventQueue) Empty() bool { ... }
func (q *EventQueue) PushIfNotExists(e Event) { ... }  // Avoid duplicate StateChanged
```

### Solver State

```go
// internal/solver/v4/state.go

type State struct {
    // Time
    Now int
    
    // Queue busy-until times
    BuildingQueueFreeAt int
    ResearchQueueFreeAt int
    TrainingQueueFreeAt int
    
    // Pending actions (for completion handling)
    PendingBuilding *BuildingAction
    PendingResearch *ResearchAction
    PendingTraining *TrainUnitAction
    
    // Buildings
    BuildingLevels map[models.BuildingType]int
    
    // Resources (indexed: 0=Wood, 1=Stone, 2=Iron)
    Resources       [3]float64
    ProductionRates [3]float64
    StorageCaps     [3]int
    ProductionBonus float64  // From Beer tester, Wheelbarrow
    
    // Food
    FoodUsed     int
    FoodCapacity int
    
    // Research
    ResearchedTechs map[string]bool
    
    // Army (strict typing)
    Army           models.Army  // Available units
    UnitsOnMission models.Army  // Busy on missions
    
    // Missions
    RunningMissions []*models.MissionState
}
```

---

## Unit Technology Prerequisites

```go
// internal/models/units.go

var UnitTechRequirements = map[UnitType]string{
    Spearman:    "",             // No tech needed
    Swordsman:   "Swordsmith",   // Library 4
    Archer:      "Longbow",      // Library 1
    Crossbowman: "Crossbow",     // Library 5
    Horseman:    "",             // No tech needed
    Lancer:      "Horse armour", // Library 7
    Handcart:    "",             // No tech needed
    Oxcart:      "",             // No tech needed
}
```

---

## ROI Calculations

### Mission Investment ROI

A mission may require a chain of prerequisites:

```
Mission "Castle Festival"
  └── Tavern Level 5
  └── 100 Infantry + 100 Cavalry + 100 Artillery
        └── Units may need tech → tech needs Library level
```

```go
// internal/solver/v4/roi.go

type MissionInvestment struct {
    Mission *models.Mission
    
    // Prerequisites needed
    TavernUpgradesNeeded  int
    LibraryUpgradesNeeded int
    TechsToResearch       []string
    UnitsToTrain          models.Army
    
    // Calculated costs
    TotalBuildSeconds    int  // Building queue blocked
    TotalResearchSeconds int  // Research queue blocked
    TotalTrainSeconds    int  // Training queue blocked
}

func (mi *MissionInvestment) ROI(state *State) float64 {
    // Benefit: repeatable mission rewards over remaining game time
    rewardPerHour := mi.Mission.NetAverageRewardPerHour()
    
    // Estimate how many times we can run this mission
    // (simplified: assume 100 hours remaining playtime)
    hoursRemaining := 100.0
    missionDurationHours := float64(mi.Mission.DurationMinutes) / 60.0
    estimatedRuns := hoursRemaining / missionDurationHours
    totalBenefit := rewardPerHour * missionDurationHours * estimatedRuns
    
    // Cost: opportunity cost of blocking queues
    // Building queue is most expensive
    buildCostWeight := 1.0
    researchCostWeight := 0.3
    trainCostWeight := 0.1
    
    totalCostHours := float64(mi.TotalBuildSeconds)/3600*buildCostWeight +
                      float64(mi.TotalResearchSeconds)/3600*researchCostWeight +
                      float64(mi.TotalTrainSeconds)/3600*trainCostWeight
    
    if totalCostHours <= 0 {
        return totalBenefit * 1000  // Already ready
    }
    
    return totalBenefit / totalCostHours
}
```

### Reactive Prerequisites

When a mission has high ROI, we work backwards through its prerequisites:

```go
func (s *Solver) getNextPrerequisiteAction(state *State, mi *MissionInvestment) Action {
    // 1. Tavern level
    if state.GetBuildingLevel(models.Tavern) < mi.Mission.TavernLevel {
        return s.createTavernUpgrade(state)
    }
    
    // 2. For each required unit type
    for _, req := range mi.Mission.UnitsRequired {
        have := state.Army.Get(req.Type)
        if have >= req.Count {
            continue
        }
        
        // 2a. Check tech requirement
        techName := models.UnitTechRequirements[req.Type]
        if techName != "" && !state.ResearchedTechs[techName] {
            tech := s.Technologies[techName]
            
            // 2b. Check Library level for tech
            if state.GetBuildingLevel(models.Library) < tech.RequiredLibraryLevel {
                return s.createLibraryUpgrade(state, tech.RequiredLibraryLevel)
            }
            
            // 2c. Research the tech (on research queue)
            return &ResearchAction{Technology: tech}
        }
        
        // 2d. Train the unit (on training queue)
        return &TrainUnitAction{UnitType: req.Type, Quantity: 1}
    }
    
    // 3. All prerequisites met
    return nil
}
```

---

## Main Solver Loop

```go
// internal/solver/v4/solver.go

func (s *Solver) Solve(initialState *models.GameState) *models.Solution {
    state := NewState(initialState)
    events := NewEventQueue()
    
    // Bootstrap: evaluate initial state
    events.Push(Event{Time: 0, Type: EventStateChanged})
    
    maxIterations := 1000000
    iterations := 0
    
    for !events.Empty() && !s.allTargetsReached(state) && iterations < maxIterations {
        iterations++
        
        event := events.Pop()
        
        // Advance time (accumulate resources)
        if event.Time > state.Now {
            s.advanceTime(state, event.Time - state.Now)
        }
        
        s.processEvent(state, event, events)
    }
    
    return s.buildSolution(state)
}

func (s *Solver) processEvent(state *State, event Event, events *EventQueue) {
    switch event.Type {
    case EventMissionComplete:
        s.handleMissionComplete(state, event, events)
        
    case EventBuildingComplete:
        s.handleBuildingComplete(state, event, events)
        
    case EventResearchComplete:
        s.handleResearchComplete(state, event, events)
        
    case EventTrainingComplete:
        s.handleTrainingComplete(state, event, events)
        
    case EventStateChanged:
        s.handleStateChanged(state, events)
    }
}

func (s *Solver) handleStateChanged(state *State, events *EventQueue) {
    // Try to start actions on all free queues
    
    // Building queue
    if state.Now >= state.BuildingQueueFreeAt {
        if action := s.pickBestBuildingAction(state); action != nil {
            if s.canAfford(state, action) {
                s.executeBuilding(state, action, events)
            }
        }
    }
    
    // Research queue
    if state.Now >= state.ResearchQueueFreeAt {
        if action := s.pickBestResearchAction(state); action != nil {
            if s.canAfford(state, action) {
                s.executeResearch(state, action, events)
            }
        }
    }
    
    // Training queue
    if state.Now >= state.TrainingQueueFreeAt {
        if action := s.pickBestTrainingAction(state); action != nil {
            if s.canAfford(state, action) {
                s.executeTraining(state, action, events)
            }
        }
    }
    
    // Missions (can start multiple)
    for {
        if mission := s.pickBestMissionToStart(state); mission != nil {
            s.startMission(state, mission, events)
        } else {
            break
        }
    }
    
    // If any queue is waiting for resources, schedule wake-up
    s.scheduleResourceWakeup(state, events)
}
```

---

## Implementation Order

### Phase 1: Foundation
- [ ] Create `internal/solver/v4/` package
- [ ] Implement `EventQueue` with priority sorting
- [ ] Implement `State` with all fields
- [ ] Add `Army` struct to models (strict typing)

### Phase 2: Event Handlers
- [ ] Implement `handleBuildingComplete`
- [ ] Implement `handleResearchComplete`
- [ ] Implement `handleTrainingComplete`
- [ ] Implement `handleMissionComplete`
- [ ] Implement `handleStateChanged`

### Phase 3: Action Selection
- [ ] Port `pickBestBuildingAction` from v3 (ROI-based)
- [ ] Implement `pickBestResearchAction` (reactive for unit techs)
- [ ] Implement `pickBestTrainingAction` (for mission requirements)
- [ ] Implement `pickBestMissionToStart`

### Phase 4: ROI Integration
- [ ] Implement `MissionInvestment` calculation
- [ ] Implement `getNextPrerequisiteAction` for missions
- [ ] Compare mission ROI vs building ROI in decision making

### Phase 5: Testing
- [ ] Unit tests for EventQueue ordering
- [ ] Unit tests for Army operations
- [ ] Integration test: no idle time between missions
- [ ] Integration test: mission resources available for building
- [ ] Fuzz tests for determinism
- [ ] Fuzz tests for resource correctness

### Phase 6: CLI Integration
- [ ] Update `cmd/castle` to use v4 solver
- [ ] Add mission/training output to build order table
- [ ] Add army composition summary

---

## Testing Strategy

### Correctness Tests

```go
// Event ordering
func TestEventPriority(t *testing.T) {
    q := NewEventQueue()
    q.Push(Event{Time: 100, Type: EventStateChanged})
    q.Push(Event{Time: 100, Type: EventMissionComplete})
    q.Push(Event{Time: 100, Type: EventBuildingComplete})
    
    // Mission should come first (priority 0)
    assert.Equal(t, EventMissionComplete, q.Pop().Type)
    assert.Equal(t, EventBuildingComplete, q.Pop().Type)
    assert.Equal(t, EventStateChanged, q.Pop().Type)
}

// Mission resources available for building
func TestMissionResourcesForBuilding(t *testing.T) {
    // Setup: can't afford building, mission will provide resources
    // Verify: after mission completes, building starts immediately
}

// No idle time
func TestMissionNoIdleTime(t *testing.T) {
    // Setup: units available, missions available
    // Verify: no gap between mission end and next mission start
}
```

### Determinism Tests

```go
func FuzzSolverDeterminism(f *testing.F) {
    f.Fuzz(func(t *testing.T, seed int64) {
        state := randomState(seed)
        
        solution1 := solver.Solve(state.Clone())
        solution2 := solver.Solve(state.Clone())
        
        assert.Equal(t, solution1, solution2)
    })
}
```

---

## File Structure

```
internal/solver/v4/
├── solver.go       # Main loop, event processing
├── state.go        # State struct, helpers
├── events.go       # EventQueue, Event types
├── actions.go      # BuildingAction, ResearchAction, TrainUnitAction
├── roi.go          # ROI calculations, MissionInvestment
├── handlers.go     # Event handlers (handleBuildingComplete, etc.)
├── decisions.go    # pickBestBuildingAction, pickBestMissionToStart, etc.
├── solver_test.go  # Unit tests
└── fuzz_test.go    # Fuzz tests

internal/models/
├── army.go         # Army struct (strict typing) [NEW]
├── missions.go     # Mission, MissionState (existing)
├── types.go        # Existing types
└── units.go        # UnitTechRequirements [NEW]
```

---

## Migration Path

1. Implement v4 solver alongside v3
2. Add feature flag to switch between solvers
3. Compare outputs for existing castle-only scenarios
4. Once validated, make v4 the default
5. Deprecate v3

---

## Open Questions

1. **Arsenal capacity**: Does Arsenal level affect training speed or queue size?
2. **Mission cooldowns**: Can the same mission be run immediately after completion?
3. **Transport units**: Should we train Handcart/Oxcart for trading, or only combat units for missions?
4. **Partial unit requirements**: If mission needs 100 Spearman but we have 80, can we substitute?
