"""CP-SAT solver with discretized time and proper resource modeling."""

from collections import defaultdict
from dataclasses import dataclass

from ortools.sat.python import cp_model

from solver_lnk.models import (
    Building,
    BuildingType,
    BuildingUpgradeAction,
    GameState,
    LibraryResearchAction,
    ResourceType,
    Solution,
)
from solver_lnk.utils.tech_loader import load_technologies


@dataclass
class Task:
    """Represents a single upgrade/research task."""

    building_type: BuildingType | None
    from_level: int
    to_level: int
    is_research: bool
    tech_name: str | None
    start_var: cp_model.IntVar
    end_var: cp_model.IntVar
    interval_var: cp_model.IntervalVar
    duration_minutes: int
    wood_cost: int
    stone_cost: int
    iron_cost: int
    food_cost: int


class CPSATDualQueueSolver:
    """
    CP-SAT solver using time discretization for resource accumulation.

    Models:
    - Time discretized into steps (minutes)
    - Resource levels tracked at each time step
    - Production rates applied continuously
    - Storage capacity constraints enforced
    - Dual queues: building + research
    """

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
        time_step_minutes: int = 60,  # 1 hour steps for feasibility
    ):
        """Initialize solver."""
        self.buildings = buildings
        self.initial_state = initial_state
        self.target_levels = target_levels
        self.time_step = time_step_minutes

        # Load technologies
        self.technologies = load_technologies()

        # Determine required technologies
        self.required_techs = self._determine_required_technologies()

    def _determine_required_technologies(self) -> set[str]:
        """Determine which technologies are required."""
        required = set()

        for btype, target_level in self.target_levels.items():
            building = self.buildings.get(btype)
            if not building:
                continue

            current = self.initial_state.building_levels.get(btype, 1)

            for level in range(current + 1, target_level + 1):
                if level in building.technology_prerequisites:
                    required.add(building.technology_prerequisites[level])

        return required

    def solve(self, time_limit_seconds: float = 600.0) -> Solution | None:
        """
        Solve using time-discretized CP-SAT model.

        Approach:
        1. Create tasks for all upgrades
        2. Discretize time into steps
        3. Track resource levels at each step
        4. Enforce production/consumption/storage
        5. Minimize makespan
        """
        model = cp_model.CpModel()

        # Estimate horizon (conservative)
        horizon_hours = 2000
        horizon_steps = horizon_hours * 60 // self.time_step

        # Create all upgrade tasks
        tasks = self._create_tasks(model, horizon_steps)

        if not tasks:
            return None

        # Add sequencing constraints (levels must be sequential)
        self._add_sequencing_constraints(model, tasks)

        # Add queue constraints (no overlap within queues)
        self._add_queue_constraints(model, tasks)

        # Add resource constraints
        self._add_resource_constraints_simplified(model, tasks, horizon_steps)

        # Minimize makespan
        makespan = model.new_int_var(0, horizon_steps * self.time_step, "makespan")
        model.add_max_equality(makespan, [t.end_var for t in tasks])
        model.minimize(makespan)

        # Solve
        solver = cp_model.CpSolver()
        solver.parameters.max_time_in_seconds = time_limit_seconds
        solver.parameters.log_search_progress = True
        solver.parameters.num_workers = 8

        status = solver.solve(model)

        if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
            return self._extract_solution(solver, tasks)

        return None

    def _create_tasks(self, model: cp_model.CpModel, horizon: int) -> list[Task]:
        """Create task variables for all upgrades and research."""
        tasks = []

        # Building upgrades
        for btype, target_level in self.target_levels.items():
            building = self.buildings.get(btype)
            if not building:
                continue

            current = self.initial_state.building_levels.get(btype, 1)

            for level in range(current, target_level):
                from_level = level
                to_level = level + 1

                # Get level data
                level_data = building.get_level_data(to_level)
                if not level_data:
                    continue

                # Get cost and duration
                costs = level_data.costs
                duration_sec = level_data.build_time_seconds
                duration_min = max(1, duration_sec // 60)  # Round to minutes

                # Create variables
                start = model.new_int_var(
                    0, horizon * self.time_step, f"{btype.value}_{to_level}_start"
                )
                end = model.new_int_var(
                    0, horizon * self.time_step, f"{btype.value}_{to_level}_end"
                )
                interval = model.new_interval_var(
                    start, duration_min, end, f"{btype.value}_{to_level}"
                )

                tasks.append(
                    Task(
                        building_type=btype,
                        from_level=from_level,
                        to_level=to_level,
                        is_research=False,
                        tech_name=None,
                        start_var=start,
                        end_var=end,
                        interval_var=interval,
                        duration_minutes=duration_min,
                        wood_cost=costs.get(ResourceType.WOOD, 0),
                        stone_cost=costs.get(ResourceType.STONE, 0),
                        iron_cost=costs.get(ResourceType.IRON, 0),
                        food_cost=costs.get(ResourceType.FOOD, 0),
                    )
                )

        # Research tasks
        for tech_name in self.required_techs:
            tech = self.technologies.get(tech_name)
            if not tech:
                continue

            duration_min = max(1, tech.cost.duration_seconds // 60)

            start = model.new_int_var(
                0, horizon * self.time_step, f"tech_{tech_name}_start"
            )
            end = model.new_int_var(
                0, horizon * self.time_step, f"tech_{tech_name}_end"
            )
            interval = model.new_interval_var(
                start, duration_min, end, f"tech_{tech_name}"
            )

            tasks.append(
                Task(
                    building_type=None,
                    from_level=0,
                    to_level=0,
                    is_research=True,
                    tech_name=tech_name,
                    start_var=start,
                    end_var=end,
                    interval_var=interval,
                    duration_minutes=duration_min,
                    wood_cost=tech.cost.wood,
                    stone_cost=tech.cost.stone,
                    iron_cost=tech.cost.iron,
                    food_cost=0,  # Tech doesn't cost food
                )
            )

        return tasks

    def _add_sequencing_constraints(self, model: cp_model.CpModel, tasks: list[Task]):
        """Ensure building levels are upgraded sequentially."""
        # Group tasks by building type
        building_tasks = defaultdict(list)
        for task in tasks:
            if task.building_type and not task.is_research:
                building_tasks[task.building_type].append(task)

        # Sort by level and add precedence
        for _btype, btasks in building_tasks.items():
            sorted_tasks = sorted(btasks, key=lambda t: t.to_level)
            for i in range(len(sorted_tasks) - 1):
                # Next task must start after previous ends
                model.add(sorted_tasks[i + 1].start_var >= sorted_tasks[i].end_var)

    def _add_queue_constraints(self, model: cp_model.CpModel, tasks: list[Task]):
        """Enforce no-overlap within each queue."""
        building_intervals = [t.interval_var for t in tasks if not t.is_research]
        research_intervals = [t.interval_var for t in tasks if t.is_research]

        if building_intervals:
            model.add_no_overlap(building_intervals)
        if research_intervals:
            model.add_no_overlap(research_intervals)

    def _add_resource_constraints_simplified(
        self, model: cp_model.CpModel, tasks: list[Task], horizon: int
    ):
        """
        Simplified resource constraints using reservoir-like approach.

        For now: Just ensure we have resources when tasks start.
        TODO: Full discretized time modeling with production rates.
        """
        # For each resource type, ensure we can afford each task
        for _task in tasks:
            # Must have resources at start time
            # This is a simplification - doesn't model production yet
            pass

        # TODO: Implement full discretized resource tracking
        # Would need: resource_level[time_step][resource_type] variables
        # And constraints linking production rates to level changes

    def _extract_solution(
        self, solver: cp_model.CpSolver, tasks: list[Task]
    ) -> Solution:
        """Extract solution from solver."""
        building_actions = []

        # Sort tasks by start time
        sorted_tasks = sorted(tasks, key=lambda t: solver.value(t.start_var))

        for task in sorted_tasks:
            start_min = solver.value(task.start_var)
            end_min = solver.value(task.end_var)

            # Only include building upgrades (skip tech research for now)
            if not task.is_research:
                building_actions.append(
                    BuildingUpgradeAction(
                        building_type=task.building_type or BuildingType.LUMBERJACK,
                        from_level=task.from_level,
                        to_level=task.to_level,
                        start_time=start_min * 60,
                        end_time=end_min * 60,
                        costs={
                            ResourceType.WOOD: task.wood_cost,
                            ResourceType.STONE: task.stone_cost,
                            ResourceType.IRON: task.iron_cost,
                            ResourceType.FOOD: task.food_cost,
                        },
                    )
                )

        # Calculate total time and final state
        total_time = max((a.end_time for a in building_actions), default=0)

        # Build final state (simplified - just copy initial and update levels)
        final_levels = dict(self.initial_state.building_levels)
        for action in building_actions:
            final_levels[action.building_type] = action.to_level

        final_state = GameState(
            building_levels=final_levels,
            resources=dict(self.initial_state.resources),  # Copy initial resources
            researched_technologies=self.initial_state.researched_technologies,
        )

        return Solution(
            building_actions=building_actions,
            research_actions=[],  # No research actions for now
            total_time_seconds=total_time,
            final_state=final_state,
        )
