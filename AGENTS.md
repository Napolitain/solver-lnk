# Agent Rules and Guidelines

## General Behavior

### When in Doubt - Always Ask or Search

**Rule**: When you have low confidence about any decision, implementation detail, tool usage, or best practice:

1. **Ask the user first** for clarification
2. **Use Context7 MCP** for library documentation (especially OR-Tools)
3. **Search the internet** if it's a factual/technical question that can be verified
4. **Never guess or assume** when uncertain

### Examples of Low Confidence Situations

- Unfamiliar tools or libraries (e.g., "ty" vs "mypy" vs "pyright")
- Game mechanics or domain-specific rules
- API specifics or library usage patterns
- Best practices for a specific technology
- Ambiguous requirements or user intent

### Process

```
IF confidence < 80% THEN
    1. Stop current action
    2. Ask user: "I'm not sure about X. Could you clarify Y?"
    OR
    3. Search web: "What is the best way to do X with Y?"
    4. Wait for confirmation before proceeding
END IF
```

## Project-Specific Rules

### Technology Stack

- **Package Manager**: uv (Astral)
- **Linter/Formatter**: ruff (Astral)
- **Type Checker**: ty (Astral) - NOT mypy or pyright
- **Python Version**: 3.13+
- **Pretty Printing**: rich (for CLI output)
- **Optimization**: OR-Tools (Google) - Use Context7 MCP for documentation
- **Documentation Reference**: Context7 MCP (`/websites/developers_google_optimization`)

### Development Workflow

**Always follow this sequence when making changes:**

1. **Make code changes**
2. **Format code**: `uv run ruff format .`
3. **Lint code**: `uv run ruff check .` (or `uv run ruff check --fix .` for auto-fixes)
4. **Type check**: `uv run ty check .`
5. **Test run**: `uv run python main.py` (with appropriate args)
6. **Verify output**: Check that the program works as expected

**Example workflow:**
```bash
# After editing code
uv run ruff format .
uv run ruff check --fix .
uv run ty check .
uv run python main.py --target-lumberjack 10

# All must pass before committing
```

### Running the Project

**Main entry point:**
```bash
# Default run (solves castle-levelup problem)
uv run python main.py

# Quiet mode (for scripts)
uv run python main.py --quiet

# With custom config file
uv run python main.py --config my_castle.json

# Export solution
uv run python main.py --export build_plan.json

# See all options
uv run python main.py --help
```

**Problem types:**
- `castle-levelup` (default): Optimize building upgrades for castle development

**Config file format** (JSON):
```json
{
  "initial_buildings": {"lumberjack": 0},
  "initial_resources": {"wood": 1000, "stone": 1000, "iron": 500, "food": 100},
  "target_levels": {"lumberjack": 10}
}
```

### Code Quality Standards

- Full type hints on all functions
- Pass `ruff check` with no errors
- Pass `ty check` with no errors
- Keep functions focused and small
- Document complex logic
- Use `rich` for all user-facing output (not plain `print`)

### Communication Style

- Be concise and direct
- Ask clarifying questions early
- Confirm understanding before implementing
- Provide alternatives when multiple approaches exist

## Update History

- 2025-12-25: Initial creation - Added "ask or search when low confidence" rule
- 2025-12-25: Added development workflow and running instructions

## Code Style Requirements (MANDATORY)

### 1. Pretty Output - Use Rich Library ✨

- ✅ **ALWAYS use `rich`** for terminal output
- ❌ **NEVER use plain `print()`** statements
- Use `rich.console.Console` for all output
- Use `rich.table.Table` for tabular data
- Use `rich.panel.Panel` for sections
- Use `rich.progress` for progress bars

**Example:**
```python
from rich.console import Console
from rich.table import Table

console = Console()
console.print("[bold green]Success![/bold green]")

table = Table(title="Results")
table.add_column("Name")
table.add_row("Value")
console.print(table)
```

### 2. Optimization - Use OR-Tools CP-SAT 🎯

- ✅ **ALWAYS use CP-SAT** for constraint solving
- ✅ Use proper constraint programming approach
- ✅ Model with variables, constraints, and objectives
- ❌ **AVOID simple heuristics** when CP-SAT can solve optimally

**Example:**
```python
from ortools.sat.python import cp_model

model = cp_model.CpModel()
x = model.NewIntVar(0, 10, 'x')
model.Add(x >= 5)
model.Minimize(x)

solver = cp_model.CpSolver()
status = solver.Solve(model)
```

### 3. Type Safety - Full Type Hints 🔒

- ✅ **ALL functions** must have type hints
- ✅ **ALL variables** should have type annotations where ambiguous
- ✅ Use `dataclass` for data structures
- ✅ Use `Enum` for fixed sets of values
- ✅ Run `uv run ty check .` to verify

**Example:**
```python
from dataclasses import dataclass
from enum import Enum

class Status(str, Enum):
    PENDING = "pending"
    DONE = "done"

@dataclass
class Task:
    name: str
    priority: int
    status: Status

def process_task(task: Task) -> bool:
    return task.status == Status.DONE
```

### 4. Data Structures - Use Dataclasses 📦

- ✅ Use `@dataclass` for all data models
- ✅ Use `frozen=True` for immutable data
- ✅ Use `field(default_factory=...)` for mutable defaults
- ❌ **AVOID plain dicts** for structured data
- ❌ **AVOID tuple unpacking** for complex data

**Example:**
```python
from dataclasses import dataclass, field

@dataclass
class Config:
    name: str
    values: list[int] = field(default_factory=list)
    frozen: bool = False

@dataclass(frozen=True)
class ImmutablePoint:
    x: int
    y: int
```

## Anti-Patterns (AVOID THESE!) ❌

| ❌ Don't Do This | ✅ Do This Instead |
|------------------|-------------------|
| `print("Results:", results)` | `console.print("[cyan]Results:[/cyan]", results)` |
| `def func(x):` | `def func(x: int) -> str:` |
| `data = {"name": "x", "val": 1}` | `@dataclass class Data: name: str; val: int` |
| `for i in range(100): if heuristic(i): ...` | Use CP-SAT to find optimal solution |
| `pip install package` | `uv add package` |

## Summary Checklist ✓

Before finishing any task:
- [ ] All code uses type hints
- [ ] All output uses `rich` (no plain print)
- [ ] All optimization uses CP-SAT properly
- [ ] All data structures use dataclasses
- [ ] Code passes `ruff format`, `ruff check`, `ty check`
- [ ] Solution runs without errors
- [ ] Asked questions when uncertain

---

**Last Updated**: 2025-12-25 - Added mandatory style requirements

## Mandatory Code Style Requirements 🎨

### 1. Rich Library for All Output ✨

**Rule**: NEVER use plain `print()` - ALWAYS use `rich`

```python
# ❌ BAD
print("Results:", data)
print(f"Score: {score}")

# ✅ GOOD
from rich.console import Console
console = Console()
console.print("[cyan]Results:[/cyan]", data)
console.print(f"[green]Score:[/green] {score}")
```

### 2. CP-SAT for Optimization 🎯

**Rule**: Use proper constraint programming, not heuristics

```python
# ❌ BAD - greedy heuristic
while tasks:
    best = max(tasks, key=priority)
    schedule.append(best)

# ✅ GOOD - CP-SAT constraint solver
from ortools.sat.python import cp_model
model = cp_model.CpModel()
# ... define variables and constraints
solver = cp_model.CpSolver()
solver.Solve(model)
```

### 3. Full Type Hints 🔒

**Rule**: Everything must be typed

```python
# ❌ BAD
def process(data):
    return data["value"]

# ✅ GOOD
def process(data: dict[str, int]) -> int:
    return data["value"]
```

### 4. Dataclasses for Data 📦

**Rule**: Use dataclasses, not dicts

```python
# ❌ BAD
config = {"name": "test", "count": 5}

# ✅ GOOD
from dataclasses import dataclass

@dataclass
class Config:
    name: str
    count: int

config = Config(name="test", count=5)
```

## Final Checklist ✓

Before finishing:
- [ ] All output uses `rich` (zero plain print statements)
- [ ] All optimization uses CP-SAT properly
- [ ] All code has type hints
- [ ] All data uses dataclasses
- [ ] Passes: `uv run ruff format . && uv run ruff check . && uv run ty check .`

## Context7 MCP for Documentation Lookup 📚

### OR-Tools Reference (MANDATORY)

**Rule**: When working with OR-Tools or CP-SAT solver, ALWAYS use Context7 MCP for API reference.

**Context7 Library ID**: `/websites/developers_google_optimization`

### When to Use Context7

Use Context7 MCP to look up:
- OR-Tools CP-SAT API syntax
- Constraint programming methods
- Solver parameters and options
- Best practices for modeling
- Code examples and patterns

### How to Use Context7

```bash
# Resolve library ID (first time)
context7-resolve-library-id "OR-Tools"

# Get documentation
context7-get-library-docs \
  --id "/websites/developers_google_optimization" \
  --mode code \
  --topic "CP-SAT solver constraint programming"
```

### Context7 Query Examples

**Good queries:**
- "CP-SAT solver constraint programming"
- "interval variables scheduling"
- "linear constraints OR-Tools"
- "objective function minimize maximize"

**Bad queries (too vague):**
- "solver"
- "optimization"
- "python"

### Integration with Project

**Before implementing OR-Tools features:**
1. ✅ Check Context7 for API reference
2. ✅ Review code examples from Context7
3. ✅ Verify syntax and method signatures
4. ❌ Don't guess CP-SAT API from memory

**Example workflow:**
```
User asks: "How do I add cumulative constraints?"
↓
Agent: Use context7-get-library-docs with topic "cumulative constraints"
↓
Review documentation and examples
↓
Implement with correct syntax
```

### Context7 Benefits

- **42,748 code snippets** for OR-Tools Python
- **Up-to-date** Google documentation
- **High quality** examples and patterns
- **Verified** API syntax

**Always prefer Context7 over guessing OR-Tools API! 📖✨**

---

**Last Updated**: 2025-12-25 - Added Context7 MCP requirement for OR-Tools

## Technology Prerequisites

### Overview
Buildings may require specific technologies to be researched in the Library before upgrading to certain levels.

### Example: Farm Requirements
- **Level 15+**: Requires "Crop rotation" (Library Level 1)
- **Level 25+**: Requires "Yoke" (Library Level 1)  
- **Level 30**: Requires "Cellar storeroom" (Library Level 1)

### Implementation
1. **Data Files**:
   - `data/technologies.json` - All technologies with their effects
   - `data/library_levels.json` - Library upgrade costs and times
   - `data/technology_prerequisites.json` - Building level → technology mapping

2. **Parser**: `scripts/parse_library.py` - Parses raw library data

3. **Model**: 
   - `Building.technology_prerequisites: dict[int, str]` maps level → tech name
   - `Building.can_upgrade_to()` checks both building AND tech prerequisites
   - `Technology` dataclass stores tech metadata

### Solver Integration (TODO)
- Track researched technologies in GameState
- Model library upgrades as CP-SAT tasks
- Add ordering constraints: Library must be upgraded before dependent buildings
- Ensure Farm 15/25/30 cannot start until Library 1 completes


## Dual-Queue System

### Game Mechanics
Lords and Knights has **two parallel construction queues**:

1. **Building Queue**: Regular buildings (Lumberjack, Farm, Keep, Arsenal, etc.)
2. **Research Queue**: Library building upgrades + Technology research

Both queues can operate simultaneously, allowing parallel upgrades.

### Technology Research System
- Technologies must be **researched separately** with their own costs and duration
- Technology research uses the **Research Queue** (same queue as Library upgrades)
- Each technology requires a minimum **Library level** before it can be researched
- Example technologies:
  - **Crop Rotation** (Library 1): 640W, 320S, 640I, 8 hours → Enables Farm Level 15
  - **Yoke** (Library 1): 840W, 840S, 1120I, 12 hours → Enables Farm Level 25
  - **Cellar Storeroom** (Library 1): 1200W, 2000S, 800I, 16 hours → Enables Farm Level 30

### Technology Prerequisites
Buildings may require specific technologies to be researched before upgrading:
- **Farm Level 15+**: Requires "Crop Rotation" to be researched
- **Farm Level 25+**: Requires "Yoke" to be researched
- **Farm Level 30**: Requires "Cellar Storeroom" to be researched

### CP-SAT Dual-Queue Solver
**File**: `solver_lnk/solvers/cpsat_dual_queue_solver.py`

**Features**:
- Separate interval variables for building queue and library queue
- `AddNoOverlap()` constraint for each queue independently
- Technology precedence constraints (e.g., Library 1 must complete before Farm 15 starts)
- Sequential upgrade constraints within same building
- Minimizes makespan (total completion time)

**Usage**:
```bash
uv run main.py --solver cpsat-dual-queue
```

**Default**: The dual-queue solver is now the default solver.

## Build Order Solver Architecture (UPDATED 2025-12-25)

### Current Issue: Incorrect Resource Modeling ⚠️

The current CP-SAT implementation does NOT correctly model continuous resource accumulation over time. The solver shows instant upgrades at time 00:00:00 which is incorrect.

### Root Cause

- No resource production modeling - resources don't accumulate over time from production buildings
- No wait time for resource gathering - solver assumes infinite resources
- Missing reservoir constraints for continuous resource flow

### Correct Solution: Reservoir Constraints with Time Discretization ✅

Use **OR-Tools CP-SAT with `add_reservoir_constraint`**:

#### 1. Time Discretization
- Use **minutes** or **10-second intervals** as base time unit
- Scale all durations to integers (e.g., 06:23 → 383 seconds)

#### 2. Resource Reservoir Modeling
Each resource (wood, stone, iron, food) needs:
- **Reservoir constraint** tracking level over time
- **Initial production**: Starting resources + base production rate
- **Consumption events**: Building upgrades consume resources (negative demand at start time)
- **Production events**: Production building upgrades increase rates (positive demand over time)

#### 3. Storage Constraints
- Use `max_level` parameter in reservoir constraint for storage limits
- Or add separate cumulative constraints for storage capacity

#### 4. Queue Constraints
- **Building queue**: `add_no_overlap` with all building upgrade intervals
- **Research queue**: `add_no_overlap` with library upgrades + tech research intervals
- Both queues are independent and can run in parallel

#### 5. Key Implementation Points

```python
from ortools.sat.python import cp_model

model = cp_model.CpModel()

# For each resource (wood, stone, iron, food):
resource_times = []  # Time variables for events
resource_demands = []  # Amount (positive=production, negative=consumption)

# Initial resources at time 0
resource_times.append(model.NewConstant(0))
resource_demands.append(initial_wood)

# Building upgrade consumes resources
upgrade_start = model.NewIntVar(0, horizon, 'upgrade_start')
resource_times.append(upgrade_start)
resource_demands.append(-cost_wood)  # Negative = consumption

# Production building adds continuous production
# Discretize: add production event every time period
for t in range(0, horizon, production_interval):
    resource_times.append(model.NewConstant(t))
    resource_demands.append(production_rate * production_interval)

# Add reservoir constraint
model.AddReservoirConstraint(
    times=resource_times,
    demands=resource_demands,
    min_level=0,
    max_level=storage_capacity
)
```

#### 6. Handling Continuous Production

Since CP-SAT works with discrete events, continuous production must be discretized:

**Option A: Fixed time intervals**
- Every N minutes, add a production event
- Finer intervals = more accurate but larger model

**Option B: Event-driven**
- Add production events only at key times (upgrade starts/ends)
- Less accurate but more efficient

**Option C: Piecewise linear**
- Model production rate as piecewise linear function
- Most accurate but complex

**Recommendation**: Start with Option A (1-minute intervals for first implementation)

#### 7. Objective Function

```python
# Minimize makespan (total completion time)
makespan = model.NewIntVar(0, horizon, 'makespan')
model.AddMaxEquality(makespan, all_upgrade_end_times)
model.Minimize(makespan)
```

### Implementation Priority

1. ✅ **First**: Implement reservoir constraints for resource accumulation
2. ✅ **Second**: Add production rate modeling (discretized over time)
3. ✅ **Third**: Add storage capacity constraints
4. ✅ **Fourth**: Verify solution correctness (no instant upgrades, resources accumulate properly)
5. ✅ **Fifth**: Optimize performance (adjust time discretization granularity)

### References

- OR-Tools `AddReservoirConstraint`: https://developers.google.com/optimization/reference/python/sat/python/cp_model#addreservoirconstraint
- Example from Context7: Consumer-producer problems with reservoir constraints
- Stack Overflow: Efficient reservoir formulations in CP solvers

### Current Status

**NOT ACCEPTABLE** ❌ - Must implement proper reservoir constraints before considering solution correct.

---

**Last Updated**: 2025-12-25 - Documented correct approach for resource accumulation with reservoir constraints

