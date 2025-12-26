# Lords and Knights - Game Rules & Solver Design

## Game Overview

**Lords and Knights** is a medieval strategy game where players build and upgrade castles. The goal is to develop your castle efficiently by managing resources, constructing buildings, and researching technologies.

---

## Resources

The game has **4 resource types**:

| Resource | Production Building | Storage Building | Behavior |
|----------|---------------------|------------------|----------|
| **Wood** | Lumberjack | Wood Store | Continuously produced per hour |
| **Stone** | Quarry | Stone Store | Continuously produced per hour |
| **Iron** | Ore Mine | Ore Store | Continuously produced per hour |
| **Food** | Farm | Farm | **ABSOLUTE capacity** (not produced!) |

### Resource Mechanics

1. **Wood, Stone, Iron**: Produced continuously over time at rates determined by production building levels
2. **Food (Workers)**: Represents available workforce capacity from the Farm
   - Food is **NOT** produced - it's an absolute limit
   - Upgrading the Farm increases total worker capacity
   - Each building upgrade consumes workers permanently
   - Workers are "used" when buildings are upgraded (FoodUsed tracks this)
   - Workers can also be used to train troops, important later.

### Storage Limits

- Resources accumulate up to storage building capacity
- Exceeding capacity wastes production
- Must upgrade storage before large purchases

---

## Buildings

### Building Categories

| Category | Buildings | Purpose |
|----------|-----------|---------|
| **Production** | Lumberjack, Quarry, Ore Mine | Generate resources per hour |
| **Storage** | Wood Store, Stone Store, Ore Store | Increase resource capacity |
| **Population** | Farm | Provides worker (food) capacity |
| **Core** | Keep, Library | Central buildings, unlock features |
| **Military** | Arsenal, Fortifications | Train units, defend castle |
| **Economy** | Market, Tavern | Trade, recruit heroes |

### Building Mechanics

- All buildings start at **Level 1**
- Maximum level is typically **30**
- Each level requires:
  - **Resources**: Wood, Stone, Iron
  - **Food (Workers)**: Consumed permanently
  - **Build Time**: Real-time construction duration
- Only **one building** can be upgraded at a time (single building queue)

---

## Technologies (Research)

### Research System

- Researched at the **Library**
- Each technology requires a minimum Library level
- Research runs on a **separate queue** (parallel to building queue)
- Technologies can:
  - Unlock unit types (Longbow → Archer)
  - Enable building levels (Crop Rotation → Farm Level 15)
  - Provide bonuses (Weaponsmith → 5% Attack)

### Key Technology Prerequisites

| Technology | Enables | Required Library Level |
|------------|---------|------------------------|
| Crop Rotation | Farm Level 15 | 1 |
| Yoke | Farm Level 25 | 1 |
| Cellar Storeroom | Farm Level 30 | 1 |
| Longbow | Archer | 1 |
| Stirrup | Armoured Horseman | 2 |

---

## Queues

The game has **two parallel queues**:

1. **Building Queue**: One building upgrade at a time
2. **Research Queue**: One technology research at a time

Both queues can run simultaneously, allowing efficient parallel development.

---

## Solver Design

### Approach: Greedy Simulation

Our solver uses a **simulation-based greedy algorithm** that:

1. **Simulates time progression** with continuous resource accumulation
2. **Maintains dual queues** for buildings and research
3. **Dynamically inserts prerequisites** when constraints are hit

### Priority Order

The solver prioritizes upgrades in this order:

1. **Resource Production** (Lumberjack, Quarry, Ore Mine) - interleaved per level
2. **Storage Buildings** (Wood/Stone/Ore Store) - as needed
3. **Core Buildings** (Keep, Library)
4. **Military/Economy** (Arsenal, Tavern, Market, Fortifications)
5. **Farm** - upgraded on-demand when worker capacity is insufficient

### Key Solver Features

```
✅ Continuous resource production over time
✅ Storage capacity constraints enforced
✅ Waits for resources when insufficient (no cheating)
✅ Interleaved resource building upgrades (LJ→Q→OM per level)
✅ Food treated as ABSOLUTE capacity (not produced)
✅ Dual queue system (building + research in parallel)
✅ Technology prerequisites auto-scheduled
✅ Farm upgrades inserted when worker capacity is needed
```
b
### Algorithm Flow

```
1. Initialize state (resources, building levels, production rates)
2. Create prioritized upgrade queue
3. For each upgrade in queue:
   a. Wait for building queue if busy
   b. Check technology prerequisites → schedule research if needed
   c. Check food capacity → insert Farm upgrade if needed
   d. Check storage capacity → insert storage upgrade if needed
   e. Wait for resources to accumulate
   f. Start upgrade, deduct resources and food
   g. Advance time, complete upgrade
   h. Update production rates and storage caps
4. Return solution with all actions and total time
```

### Constraint Handling

| Constraint | Solver Response |
|------------|-----------------|
| Insufficient resources | Wait for production |
| Storage too small | Insert storage upgrade at queue front |
| Not enough workers (food) | Insert Farm upgrade at queue front |
| Technology required | Schedule research, wait for completion |
| Library level too low | Insert Library upgrade at queue front |

When we have not enough resources, it means that resource is a bottleneck. Theorically we need then to upgrade that resource production for optimizing build order.
But we must never fall into local minimum, it may happen that ore is a bottleneck locally but globally wood and stone are likely more of a bottleneck.
This is why we may run multiple times and do trial and errors.

---

## Data Files

Game data is stored in JSON format:

```
data/
├── buildings.json          # All building definitions
├── technologies.json       # Technology definitions
├── tech_library_requirements.json
├── technology_prerequisites.json
├── lumberjack/            # Per-level data for each building
├── quarry/
├── ore_mine/
├── farm/
├── wood_store/
├── stone_store/
├── ore_store/
├── keep/
├── arsenal/
├── library/
├── tavern/
├── market/
└── fortifications/
```

---

## Example: Typical Build Order

A well-optimized castle build follows roughly this pattern:

1. **Early Game**: Interleave Lumberjack/Quarry/Ore Mine upgrades. Maybe prioritize lumberjack and stone by a few levels over ore mine.
2. **Mid Game**: Upgrade storages as needed, build Library for research
3. **Late Game**: Push production to 30, research Farm techs, complete targets

Expected completion time for full castle (all targets at 30): **~45 days**

---

## Solver Usage

```bash
# Build the solver
go build -o solver ./cmd/solver/

# Run with default data directory
./solver -d data

# Quiet mode (minimal output)
./solver -d data -q
```

Output includes:
- Complete build order with timestamps
- Resource costs per upgrade
- Total completion time
- Target verification
