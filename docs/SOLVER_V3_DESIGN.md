# Solver V3: ROI-Based Multi-Queue Scheduler

## Overview

The solver optimizes castle development by making ROI-driven decisions across all available actions. At any decision point, it evaluates ALL possible actions and picks the one with the highest return on investment.

## Core Principles

### 1. ROI-First Decision Making

Every action has an ROI measured in **effective resource gain per hour**:

```
ROI = (Future Resource Gain per Hour) / (Time Investment + Resource Cost Opportunity)
```

**For Production Buildings (Lumberjack, Quarry, Ore Mine):**
```
ROI = (NewRate - CurrentRate) / BuildTime
Example: Lumberjack 1→2 gives +2 wood/hour, takes 383 seconds
ROI = 2 / (383/3600) = 18.8 wood/hour gained per hour invested
```

**For Missions:**
```
ROI = AverageReward / DurationMinutes * 60
Example: Overtime Wood gives 7.5 avg wood in 5 minutes
ROI = 7.5 / 5 * 60 = 90 wood/hour
```

**For Tavern Upgrades (unlocks new missions):**
```
ROI = (Sum of new mission ROIs * expected utilization) / BuildTime
Must consider: unit requirements, mission cooldowns, parallel slots
```

### 2. Queue Model

**4 Independent Queues:**
| Queue | Behavior | Constraint |
|-------|----------|------------|
| Building | Serial | One upgrade at a time |
| Research | Serial | One tech at a time, parallel to building |
| Units | Serial | One batch at a time, parallel to others |
| Missions | Parallel | Each mission type runs AT MOST ONCE at a time |

**Key Mission Constraint:**
- "Overtime Wood" can run once
- "Overtime Stone" can run once (at same time as Overtime Wood)
- You CANNOT run 4x "Overtime Wood" simultaneously
- Different missions CAN run in parallel if you have enough units

### 3. State Model

```go
type SimState struct {
    // Time
    Now int  // Current simulation time in seconds
    
    // Resources
    Resources map[ResourceType]float64  // Current amounts
    StorageCaps map[ResourceType]int    // Maximum storage
    ProductionRates map[ResourceType]float64  // Per-hour rates
    
    // Buildings
    BuildingLevels map[BuildingType]int  // Current levels (start at 1)
    BuildingQueueFreeAt int              // When building queue is free
    
    // Research
    ResearchedTechs map[string]bool
    ResearchQueueFreeAt int
    
    // Units
    Units map[UnitType]int           // Total owned
    AvailableUnits map[UnitType]int  // Not on missions
    UnitQueueFreeAt int
    
    // Missions
    RunningMissions map[string]*MissionState  // Key = mission name, only ONE per type
    
    // Food
    FoodCapacity int
    FoodUsed int
}
```

### 4. Decision Loop

At each tick, the solver:

```
1. ADVANCE to next decision point (earliest of: queue free, mission complete, resource ready)

2. UPDATE state (process completed events, accumulate resources)

3. EVALUATE all possible actions:
   - Next building upgrade (if queue free)
   - Next research (if queue free)  
   - Next unit training (if queue free)
   - Each available mission (if not already running AND units available)

4. CALCULATE ROI for each action:
   - Buildings: production gain / time
   - Research: enables future actions (high priority if blocking)
   - Units: enables missions (ROI = mission ROI they unlock)
   - Missions: reward / duration

5. SELECT action with highest ROI
   - Special rules:
     a. If building queue is blocked on resources, prefer missions that provide those resources
     b. If building queue is free AND affordable, building ROI gets 2x multiplier (always build if possible)
     c. Reactive upgrades (Storage, Farm) inserted only when actually needed

6. EXECUTE selected action:
   - Deduct costs
   - Update queue availability
   - Schedule completion event

7. REPEAT until all targets reached
```

### 5. Building Priority

**Proactive Buildings (upgrade aggressively):**
- Lumberjack, Quarry, Ore Mine - core income
- Tavern - unlocks missions (evaluate ROI of new missions)
- Arsenal - required for units (evaluate if units needed for missions)

**Reactive Buildings (upgrade only when needed):**
- Storage (Wood/Stone/Ore Store) - only when cost exceeds capacity
- Farm - only when food capacity insufficient
- Library - only when research requires it
- Keep, Market, Fortifications - only if in target list

### 6. Mission Scheduling

**Tavern Level → Available Missions:**
| Level | Missions Available |
|-------|-------------------|
| 1 | Overtime Wood |
| 2 | + Overtime Stone, Hunting |
| 3 | + Overtime Ore, Chop Wood |
| 4 | + Mandatory Overtime, Help Stone Cutters |
| 5 | + Market Day |
| 6 | + Forging Tools, Feed Miners |
| ... | ... |

**Mission Execution Rules:**
1. Each mission type can only run ONCE at a time
2. Different mission types run in parallel
3. Units are locked during mission, returned on completion
4. Rewards added to resources on completion (capped by storage)

**Mission Selection Priority:**
1. If building blocked on resource X → prefer missions that give X
2. Otherwise → pick highest ROI mission with available units
3. Never run a mission if it would delay a building start

### 7. Unit Training

Units are trained to enable missions. Training decision:

```
Should I train units?
1. What missions could I run with more units?
2. What's the ROI of those missions?
3. Is training cost + time worth the mission ROI?

Train if: MissionROI * ExpectedMissionRuns > TrainingCost + OpportunityCost
```

### 8. Example Decision Flow

**State:** T=0, Tavern=1, 20 Spearmen available, Building queue free
**Options:**
1. Build Lumberjack 1→2: ROI = 18.8 wood/hour, cost 31W/26S/7I
2. Run Overtime Wood: ROI = 90 wood/hour, needs 5 spearmen, 5 min

**Decision:** 
- We CAN afford Lumberjack (have 120 of each resource)
- Building queue is free
- Rule: "If building queue free AND affordable → build"
- **Action: Start Lumberjack 1→2**

**Simultaneously:**
- Mission queue is separate
- We have 20 spearmen, need 5 for Overtime Wood
- Only ONE Overtime Wood can run (Tavern 1 only has this mission)
- **Action: Start Overtime Wood** (uses 5 spearmen)

**Result at T=0:**
- Lumberjack upgrading (finishes T=383s)
- Overtime Wood running (finishes T=300s)
- 15 spearmen idle (no other missions available at Tavern 1)

### 9. Tavern Upgrade ROI Calculation

When to upgrade Tavern?

```
TavernUpgradeROI = (NewMissionsValuePerHour - CurrentMissionsValuePerHour) / TavernBuildTime

NewMissionsValuePerHour = Sum of: MissionROI * (CanRunInParallel ? 1 : 0.5) for each new mission

Consider:
- Do we have units to run new missions?
- If not, add unit training time to the equation
```

**Example: Tavern 1→2**
- Current: Only Overtime Wood (90/hour, but only runs 1 at a time)
- New: + Overtime Stone (90/hour) + Hunting (if have archers)
- If we have spearmen: We can now run Wood AND Stone simultaneously
- Effective gain: +90/hour (Stone mission)
- Build time: 672 seconds = 0.187 hours
- ROI = 90 / 0.187 = 481 equivalent resource gain per build-hour

Compare to Lumberjack 1→2: ROI = 18.8
**Tavern upgrade is MUCH better ROI if we have units!**

### 10. Implementation Structure

```go
// Main solver loop
func (s *Solver) Solve() *Solution {
    state := s.initState()
    
    for !s.allTargetsReached(state) {
        // 1. Advance to next decision point
        nextTime := s.nextDecisionPoint(state)
        s.advanceTime(state, nextTime)
        
        // 2. Get all possible actions
        actions := s.getPossibleActions(state)
        
        // 3. Calculate ROI for each
        for _, a := range actions {
            a.ROI = s.calculateROI(state, a)
        }
        
        // 4. Select best action (may select multiple if different queues)
        selected := s.selectBestActions(state, actions)
        
        // 5. Execute
        for _, a := range selected {
            s.executeAction(state, a)
        }
    }
    
    return s.buildSolution(state)
}

// Action types
type Action interface {
    Queue() QueueType  // Building, Research, Unit, Mission
    CanExecute(state) bool
    Costs() Costs
    Duration() int
    ROI(state) float64
    Execute(state)
}
```

### 11. Output Format

The solution is a chronological list of action START times

```
T=0:      Start Lumberjack 1→2
T=0:      Start Overtime Wood (Mission)
T=300:    Overtime Wood complete, Start Overtime Wood again
T=383:    Lumberjack 1→2 complete, Start Quarry 1→2
T=600:    Overtime Wood complete, Start Overtime Wood again
...
```

Actions are sorted by start time. Multiple actions can have the same start time if they're on different queues.

## Success Criteria

1. **Buildings start at correct level** (1→2, not 0→2)
2. **Missions run at most once per type** at any time
3. **ROI drives all decisions** - no hardcoded priority order
4. **~45-60 day completion** for full castle with missions
5. **Deterministic output** - same input always produces same output


## Integration

Castle cli must output the entire build (=action order). chronological list in the end.
Server must be the protobuf grpc endpoint always returning the immediate next action.
The code must make use ideally of protobuf defined data structures rather than Go defined data structures, to allow easy integration with grpc server.
maps are discouraged, i want for example STONE_STORAGE: int rather than resourcestorage<map:int>. Avoid dynamic allocation.
Prefer cache locality, extremely cpu cache friendly design for hotpath.
