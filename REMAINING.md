# Solver V2 Migration: Remaining Tasks

The core architecture has been successfully moved to an event-driven simulation engine (`internal/engine`). The legacy `GreedySolver` has been removed, and the new `Solver` is now the standard.

## Solver Architecture & Action Ordering

### 1. N-Queue Parallel System
The solver manages multiple independent execution tracks:
- **Building Queue:** Serial (one upgrade at a time).
- **Research Queue:** Serial (one technology at a time, runs parallel to buildings).
- **Unit Queue:** Serial (one batch of units at a time, runs parallel to buildings/research).
- **Mission "Queue":** Pseudo-parallel. Any number of missions can run simultaneously as long as unit requirements are met.

### 2. Simulation Loop (Event-Driven)
The solver operates in a continuous loop until all target queues are empty:
1. **Constraint Resolution:** For the head of each queue, check for missing prerequisites (Tech, Library level, Food Capacity, Storage). If a dependency is missing, prepends the necessary fix (e.g., a Storage upgrade) to the **Building Queue**.
2. **Candidate Identification:** Identifies the next possible action from the head of every queue.
3. **Best Action Selection:** 
   - For each candidate, calculates `actualStart = Max(QueueFreeTime, ResourceReadyTime)`.
   - `ResourceReadyTime` is calculated by the engine, accounting for continuous production AND discrete future mission rewards.
   - Selects the action with the **earliest absolute start time**.
4. **Execution & Event Scheduling:** 
   - Advances simulation time to `actualStart`.
   - Deducts resources immediately.
   - Schedules a **ScheduledEvent** for the action's completion to apply delayed effects (e.g., updating production rates, increasing levels, or returning units/rewards).
   - If multiple actions share an identical start time, the internal priority is `Research > Building > Units > Missions`.

### 3. Action Order vs. Unified Timeline
- **Action Order:** The solver selects actions one by one based on their starting potential. Because queues are parallel, the "Build Order" is actually a chronological interleaving of actions from different queues.
- **Result:** The final output is a unified timeline where actions can overlap in duration but are initiated based on the optimal path to maximizing resource ROI and minimizing total build time.

## 1. Mission Selection Optimization 
**Status:** Functional but inefficient.
The current heuristic maximizes **Net Average ROI per hour**. In the full build test, this leads to a ~377-day completion time because:
- It may consume resources needed for the next building upgrade (starvation).
- It doesn't explicitly prioritize the bottleneck resource needed to unblock the building queue.

**Task:** Refine `getCandidates` mission logic to:
- Filter out missions that consume a resource currently in deficit for the `bQueue` head.
- Prioritize missions that provide the specific resource(s) required by the `bQueue` head.

## 2. Research Queue Dependency Resolution
**Status:** `FIXME` in code.
Currently, if a technology in the `rQueue` requires a higher `Library` level than current, the solver doesn't automatically insert the `Library` upgrade into `bQueue`.

**Task:** Implement `resolveConstraints` logic for the `rQueue` to detect Library level requirements and prepend the necessary upgrades to the `bQueue`.

## 3. Parallel Queue Priority Tuning
**Status:** Functional.
Actions are selected based on the earliest `actualStart` (Queue Availability + Resource Availability). If times are equal, it defaults to `Research > Building > Units`.

**Task:** Verify if `Missions` should be treated as "background" tasks that never delay a building start, even if they have a slightly earlier start time than a wait-for-resource building action.

## 4. Test Suite Restoration
**Status:** Fuzz tests are `.disabled`; legacy tests are deleted.
- **Task:** Rewrite `internal/solver/castle/determinism_test.go` to work with the new `Solver` and multiple parallel queues.
- **Task:** Port and enable fuzz tests (`FuzzSolverDeterminism`, etc.) to ensure the new engine handles edge cases without panicking.

## 5. Timeline Visualization Refinement
**Status:** Integrated into CLI and Server.
- **Task:** Update the gRPC server and CLI to correctly display mission rewards in the timeline.
- **Task:** Ensure "Food" usage tracking is accurate across all interleaved actions in the final `Solution` output.
