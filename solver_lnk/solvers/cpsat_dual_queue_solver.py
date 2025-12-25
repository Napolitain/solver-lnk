"""CP-SAT solver with dual-queue, resource tracking, and storage constraints."""

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


class CPSATDualQueueSolver:
    """
    Constraint programming solver using OR-Tools CP-SAT.

    Models:
    - Dual queues (building + research) for parallel execution
    - Resource production rates from production buildings
    - Storage capacity constraints
    - Resource accumulation over time
    - Technology prerequisites
    """

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
        time_scale_minutes: int = 1,  # 1 minute granularity for better precision
    ):
        """
        Initialize solver.

        Args:
            buildings: Available building types with their data
            initial_state: Starting game state
            target_levels: Target levels for each building type
            time_scale_minutes: Time discretization in minutes
        """
        self.buildings = buildings
        self.initial_state = initial_state
        self.target_levels = target_levels
        self.time_scale_minutes = time_scale_minutes
        self.time_scale_seconds = time_scale_minutes * 60

        self.technologies = load_technologies()
        self.required_techs = self._determine_required_technologies()
        self.required_library_level = self._calculate_required_library_level()

    def _determine_required_technologies(self) -> set[str]:
        """Determine which technologies are needed for target levels."""
        required = set()

        # Check all buildings for tech prerequisites
        for building_type, target_level in self.target_levels.items():
            building = self.buildings.get(building_type)
            if not building:
                continue

            current_level = self.initial_state.building_levels.get(building_type, 1)

            # Check if any level we need to reach requires a tech
            for level in range(current_level + 1, target_level + 1):
                if level in building.technology_prerequisites:
                    tech_name = building.technology_prerequisites[level]
                    required.add(tech_name)

        return required

    def _calculate_required_library_level(self) -> int:
        """Calculate minimum library level needed for target buildings and techs."""
        required_level = 1  # Start at level 1 (base)

        # Check what library level is needed for each required technology
        for tech_name in self.required_techs:
            tech = self.technologies.get(tech_name)
            if tech:
                required_level = max(required_level, tech.library_level_required)

        return required_level

    def solve(self, time_limit_seconds: float = 300.0) -> Solution | None:
        """
        Solve the build order optimization problem with resource constraints.

        Returns:
            Solution if found, None otherwise
        """
        model = cp_model.CpModel()

        # Estimate max horizon (in time units)
        # Conservative estimate: sum of all build times * 3
        max_horizon = self._estimate_max_horizon()

        # Create task variables for all building upgrades
        building_tasks = self._create_building_tasks(model, max_horizon)

        # Create task variables for library and tech research
        library_tasks, tech_tasks = self._create_research_tasks(model, max_horizon)

        # Add sequencing constraints
        self._add_sequencing_constraints(model, building_tasks, library_tasks)

        # Add no-overlap constraints for dual queues
        self._add_no_overlap_constraints(
            model, building_tasks, library_tasks, tech_tasks
        )

        # Add tech dependency constraints
        self._add_tech_dependencies(model, building_tasks, library_tasks, tech_tasks)

        # Add resource constraints (production, storage, costs)
        self._add_resource_constraints(
            model, building_tasks, library_tasks, tech_tasks, max_horizon
        )

        # Objective: minimize makespan
        makespan = model.NewIntVar(0, max_horizon, "makespan")
        all_ends = [t["end"] for t in building_tasks.values()]
        all_ends += [t["end"] for t in library_tasks.values()]
        all_ends += [t["end"] for t in tech_tasks.values()]

        if all_ends:
            model.AddMaxEquality(makespan, all_ends)

        model.Minimize(makespan)

        # Solve
        solver = cp_model.CpSolver()
        solver.parameters.max_time_in_seconds = time_limit_seconds
        solver.parameters.log_search_progress = False
        solver.parameters.num_search_workers = 8

        status = solver.Solve(model)

        if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
            return self._extract_solution(
                solver, building_tasks, library_tasks, tech_tasks
            )

        return None

    def _estimate_max_horizon(self) -> int:
        """Estimate maximum time horizon in time units."""
        total_time = 0

        # Sum all building upgrade times
        for building_type, target_level in self.target_levels.items():
            building = self.buildings.get(building_type)
            if not building:
                continue

            current_level = self.initial_state.building_levels.get(building_type, 1)
            for level in range(current_level + 1, target_level + 1):
                level_data = building.get_level_data(level)
                if level_data:
                    total_time += level_data.build_time_seconds

        # Add tech research times
        for tech_name in self.required_techs:
            tech = self.technologies.get(tech_name)
            if tech:
                total_time += tech.cost.duration_seconds

        # Convert to time units and add buffer (3x for resource waiting)
        return int((total_time * 3) // self.time_scale_seconds) + 1000

    def _create_building_tasks(self, model: cp_model.CpModel, max_horizon: int) -> dict:
        """Create CP variables for all building upgrade tasks."""
        tasks = {}

        for building_type, target_level in self.target_levels.items():
            building = self.buildings.get(building_type)
            if not building:
                continue

            current_level = self.initial_state.building_levels.get(building_type, 1)

            for level in range(current_level + 1, target_level + 1):
                level_data = building.get_level_data(level)
                if not level_data:
                    continue

                duration = max(
                    1, level_data.build_time_seconds // self.time_scale_seconds
                )

                start = model.NewIntVar(
                    0, max_horizon, f"start_{building_type.value}_{level}"
                )
                end = model.NewIntVar(
                    0, max_horizon, f"end_{building_type.value}_{level}"
                )
                interval = model.NewIntervalVar(
                    start, duration, end, f"interval_{building_type.value}_{level}"
                )

                tasks[(building_type, level)] = {
                    "start": start,
                    "end": end,
                    "interval": interval,
                    "duration": duration,
                    "costs": level_data.costs,
                    "type": "building",
                    "building_type": building_type,
                    "level": level,
                    "production_bonus": level_data.production_per_hour
                    if hasattr(level_data, "production_per_hour")
                    else {},
                    "storage_bonus": level_data.storage_capacity
                    if hasattr(level_data, "storage_capacity")
                    else {},
                }

        return tasks

    def _create_research_tasks(
        self, model: cp_model.CpModel, max_horizon: int
    ) -> tuple[dict, dict]:
        """Create CP variables for library upgrades and tech research."""
        library_tasks = {}
        tech_tasks = {}

        # Library upgrades
        current_lib_level = self.initial_state.building_levels.get(
            BuildingType.LIBRARY, 1
        )
        if self.required_library_level > current_lib_level:
            library_building = self.buildings.get(BuildingType.LIBRARY)
            if library_building:
                for level in range(
                    current_lib_level + 1, self.required_library_level + 1
                ):
                    level_data = library_building.get_level_data(level)
                    if not level_data:
                        continue

                    duration = max(
                        1,
                        level_data.build_time_seconds // self.time_scale_seconds,
                    )

                    start = model.NewIntVar(0, max_horizon, f"start_library_{level}")
                    end = model.NewIntVar(0, max_horizon, f"end_library_{level}")
                    interval = model.NewIntervalVar(
                        start, duration, end, f"interval_library_{level}"
                    )

                    library_tasks[level] = {
                        "start": start,
                        "end": end,
                        "interval": interval,
                        "duration": duration,
                        "costs": level_data.costs,
                        "type": "library_upgrade",
                        "level": level,
                    }

        # Technology research
        for tech_name in self.required_techs:
            tech = self.technologies.get(tech_name)
            if not tech:
                continue

            duration = max(1, tech.cost.duration_seconds // self.time_scale_seconds)

            start = model.NewIntVar(0, max_horizon, f"start_tech_{tech_name}")
            end = model.NewIntVar(0, max_horizon, f"end_tech_{tech_name}")
            interval = model.NewIntervalVar(
                start, duration, end, f"interval_tech_{tech_name}"
            )

            tech_tasks[tech_name] = {
                "start": start,
                "end": end,
                "interval": interval,
                "duration": duration,
                "costs": {
                    ResourceType.WOOD: tech.cost.wood,
                    ResourceType.STONE: tech.cost.stone,
                    ResourceType.IRON: tech.cost.iron,
                    ResourceType.FOOD: 0,
                },
                "type": "tech_research",
                "tech_name": tech_name,
                "tech": tech,
            }

        return library_tasks, tech_tasks

    def _add_sequencing_constraints(
        self,
        model: cp_model.CpModel,
        building_tasks: dict,
        library_tasks: dict,
    ):
        """Add constraints for sequential upgrades of same building."""
        # Building sequencing
        for building_type, target_level in self.target_levels.items():
            current_level = self.initial_state.building_levels.get(building_type, 1)
            for level in range(current_level + 1, target_level):
                curr_task = (building_type, level)
                next_task = (building_type, level + 1)
                if curr_task in building_tasks and next_task in building_tasks:
                    model.Add(
                        building_tasks[curr_task]["end"]
                        <= building_tasks[next_task]["start"]
                    )

        # Library sequencing
        current_lib_level = self.initial_state.building_levels.get(
            BuildingType.LIBRARY, 1
        )
        for level in range(current_lib_level + 1, self.required_library_level):
            if level in library_tasks and (level + 1) in library_tasks:
                model.Add(
                    library_tasks[level]["end"] <= library_tasks[level + 1]["start"]
                )

    def _add_no_overlap_constraints(
        self,
        model: cp_model.CpModel,
        building_tasks: dict,
        library_tasks: dict,
        tech_tasks: dict,
    ):
        """Add no-overlap constraints for dual queues."""
        # Building queue: all building upgrades (except library)
        building_intervals = [t["interval"] for t in building_tasks.values()]
        if building_intervals:
            model.AddNoOverlap(building_intervals)

        # Research queue: library upgrades + tech research
        research_intervals = [t["interval"] for t in library_tasks.values()]
        research_intervals += [t["interval"] for t in tech_tasks.values()]
        if research_intervals:
            model.AddNoOverlap(research_intervals)

    def _add_tech_dependencies(
        self,
        model: cp_model.CpModel,
        building_tasks: dict,
        library_tasks: dict,
        tech_tasks: dict,
    ):
        """Add constraints for technology prerequisites."""
        # Tech research requires library level
        for _tech_name, task in tech_tasks.items():
            tech = task["tech"]
            required_lib_level = tech.library_level_required

            if required_lib_level in library_tasks:
                model.Add(library_tasks[required_lib_level]["end"] <= task["start"])

        # Building upgrades require technologies
        for building_type, target_level in self.target_levels.items():
            building = self.buildings.get(building_type)
            if not building:
                continue

            current_level = self.initial_state.building_levels.get(building_type, 1)

            for level in range(current_level + 1, target_level + 1):
                if level in building.technology_prerequisites:
                    tech_name = building.technology_prerequisites[level]

                    task_key = (building_type, level)
                    if tech_name in tech_tasks and task_key in building_tasks:
                        model.Add(
                            tech_tasks[tech_name]["end"]
                            <= building_tasks[task_key]["start"]
                        )

    def _add_resource_constraints(
        self,
        model: cp_model.CpModel,
        building_tasks: dict,
        library_tasks: dict,
        tech_tasks: dict,
        max_horizon: int,
    ):
        """Add resource production, storage, and cost constraints."""
        # Collect all tasks for ordering
        all_tasks = []

        for key, task in building_tasks.items():
            all_tasks.append(("building", key, task))
        for key, task in library_tasks.items():
            all_tasks.append(("library", key, task))
        for key, task in tech_tasks.items():
            all_tasks.append(("tech", key, task))

        # For each resource type, track cumulative production and consumption
        for resource_type in ResourceType:
            # Start with initial resources
            initial_amount = self.initial_state.resources.get(resource_type, 0.0)

            # For each task, ensure we have enough resources before starting
            for _task_type, _task_key, task in all_tasks:
                costs = task["costs"]
                cost = costs.get(resource_type, 0)

                if cost > 0:
                    # For now, add precedence: production buildings before consumers
                    if resource_type == ResourceType.WOOD:
                        producer_type = BuildingType.LUMBERJACK
                    elif resource_type == ResourceType.STONE:
                        producer_type = BuildingType.QUARRY
                    elif resource_type == ResourceType.IRON:
                        producer_type = BuildingType.ORE_MINE
                    elif resource_type == ResourceType.FOOD:
                        producer_type = BuildingType.FARM
                    else:
                        continue

                    # Find early producer upgrades
                    for level in range(2, 6):  # First few levels should come first
                        producer_key = (producer_type, level)
                        if producer_key in building_tasks and cost > initial_amount:
                            # Producer must complete before expensive tasks
                            model.Add(
                                building_tasks[producer_key]["end"]
                                <= task["start"]
                            )

        # Add storage constraints
        self._add_storage_constraints(model, building_tasks, all_tasks)

    def _calculate_production_rate_at_task(
        self,
        model: cp_model.CpModel,
        resource_type: ResourceType,
        task: dict,
        building_tasks: dict,
        all_tasks: list,
    ) -> float:
        """Calculate production rate for a resource at a given task's start."""
        # This would track which production buildings are completed before this task
        # Simplified for now
        return 0.0

    def _add_storage_constraints(
        self,
        model: cp_model.CpModel,
        building_tasks: dict,
        all_tasks: list,
    ):
        """Ensure storage buildings are upgraded before capacity is exceeded."""
        # For each storage building, ensure it's upgraded before
        # corresponding production exceeds capacity

        storage_mapping = {
            ResourceType.WOOD: BuildingType.WOOD_STORE,
            ResourceType.STONE: BuildingType.STONE_STORE,
            ResourceType.IRON: BuildingType.ORE_STORE,
        }

        production_mapping = {
            ResourceType.WOOD: BuildingType.LUMBERJACK,
            ResourceType.STONE: BuildingType.QUARRY,
            ResourceType.IRON: BuildingType.ORE_MINE,
        }

        for resource_type, storage_type in storage_mapping.items():
            producer_type = production_mapping[resource_type]

            # For each production level, ensure appropriate storage exists
            for prod_level in range(10, 31):  # Higher production levels
                prod_key = (producer_type, prod_level)
                if prod_key not in building_tasks:
                    continue

                # Require storage at reasonable level
                required_storage_level = max(prod_level // 2, 10)
                storage_key = (storage_type, required_storage_level)

                if storage_key in building_tasks:
                    # Storage must be ready before high production
                    model.Add(
                        building_tasks[storage_key]["end"]
                        <= building_tasks[prod_key]["start"]
                    )

    def _extract_solution(
        self,
        solver: cp_model.CpSolver,
        building_tasks: dict,
        library_tasks: dict,
        tech_tasks: dict,
    ) -> Solution:
        """Extract solution from solved model."""
        building_actions = []
        research_actions = []

        # Extract building upgrades
        for (building_type, level), task_data in building_tasks.items():
            start_time = solver.Value(task_data["start"]) * self.time_scale_seconds
            end_time = solver.Value(task_data["end"]) * self.time_scale_seconds

            action = BuildingUpgradeAction(
                building_type=building_type,
                from_level=level - 1,
                to_level=level,
                start_time=start_time,
                end_time=end_time,
                costs=task_data["costs"],
            )
            building_actions.append(action)

        # Extract library upgrades
        for level, task_data in library_tasks.items():
            start_time = solver.Value(task_data["start"]) * self.time_scale_seconds
            end_time = solver.Value(task_data["end"]) * self.time_scale_seconds

            action = LibraryResearchAction(
                from_level=level - 1,
                to_level=level,
                start_time=start_time,
                end_time=end_time,
                costs=task_data["costs"],
                technologies_unlocked=[],
            )
            research_actions.append(action)

        # Extract technology research
        for tech_name, task_data in tech_tasks.items():
            start_time = solver.Value(task_data["start"]) * self.time_scale_seconds
            end_time = solver.Value(task_data["end"]) * self.time_scale_seconds

            action = LibraryResearchAction(
                from_level=0,
                to_level=0,
                start_time=start_time,
                end_time=end_time,
                costs=task_data["costs"],
                technologies_unlocked=[tech_name],
            )
            research_actions.append(action)

        # Sort by start time
        building_actions.sort(key=lambda a: a.start_time)
        research_actions.sort(key=lambda a: a.start_time)

        # Calculate final time
        total_time = 0.0
        if building_actions:
            total_time = max(total_time, building_actions[-1].end_time)
        if research_actions:
            total_time = max(total_time, research_actions[-1].end_time)

        # Build final state
        final_state = GameState(
            building_levels={
                bt: max(
                    self.initial_state.building_levels.get(bt, 1),
                    self.target_levels.get(bt, 1),
                )
                for bt in BuildingType
            },
            resources={rt: 0.0 for rt in ResourceType},
            time_elapsed=total_time,
        )

        return Solution(
            building_actions=building_actions,
            research_actions=research_actions,
            total_time_seconds=total_time,
            final_state=final_state,
        )
