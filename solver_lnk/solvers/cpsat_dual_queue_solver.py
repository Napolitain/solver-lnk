"""CP-SAT solver with dual-queue support for buildings and research."""

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

    Implements separate building and research queues for parallel execution.
    """

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
        time_scale_minutes: int = 10,
    ):
        """
        Initialize solver.

        Args:
            buildings: Available building types with their data
            initial_state: Starting game state
            target_levels: Target levels for each building type
            time_scale_minutes: Time discretization (default: 10 minutes)
        """
        self.buildings = buildings
        self.initial_state = initial_state
        self.target_levels = target_levels
        self.time_scale_minutes = time_scale_minutes
        self.time_scale_seconds = time_scale_minutes * 60

        # Load library and technology data
        self.technologies = load_technologies()

        # Determine which techs are needed
        self.required_techs = self._determine_required_technologies()

        # Calculate required library level for targets
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
        Solve the build order optimization problem.

        Returns:
            Solution if found, None otherwise
        """
        model = cp_model.CpModel()

        # Create variables for each building upgrade
        building_tasks = {}
        building_starts = {}
        building_ends = {}
        building_intervals = {}

        max_horizon = 100000  # Very large horizon in time units

        # Building upgrade tasks
        for building_type, target_level in self.target_levels.items():
            building = self.buildings.get(building_type)
            if not building:
                continue

            current_level = self.initial_state.building_levels.get(building_type, 1)

            for level in range(current_level + 1, target_level + 1):
                level_data = building.get_level_data(level)
                if not level_data:
                    continue

                task_name = f"{building_type.value}_{level}"
                duration = level_data.build_time_seconds // self.time_scale_seconds

                start = model.NewIntVar(0, max_horizon, f"start_{task_name}")
                end = model.NewIntVar(0, max_horizon, f"end_{task_name}")
                interval = model.NewIntervalVar(
                    start, duration, end, f"interval_{task_name}"
                )

                building_tasks[(building_type, level)] = {
                    "start": start,
                    "end": end,
                    "interval": interval,
                    "duration": duration,
                    "costs": level_data.costs,
                }
                building_starts[task_name] = start
                building_ends[task_name] = end
                building_intervals[task_name] = interval

        # Research queue tasks: Library upgrades + Technology research
        research_starts = {}
        research_ends = {}
        research_intervals = {}

        # 1. Library building upgrades (if needed)
        library_upgrade_tasks = {}
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

                    task_name = f"library_upgrade_{level}"
                    duration = level_data.build_time_seconds // self.time_scale_seconds

                    start = model.NewIntVar(0, max_horizon, f"start_{task_name}")
                    end = model.NewIntVar(0, max_horizon, f"end_{task_name}")
                    interval = model.NewIntervalVar(
                        start, duration, end, f"interval_{task_name}"
                    )

                    library_upgrade_tasks[level] = {
                        "start": start,
                        "end": end,
                        "interval": interval,
                        "duration": duration,
                        "costs": level_data.costs,
                        "type": "library_upgrade",
                    }
                    research_starts[task_name] = start
                    research_ends[task_name] = end
                    research_intervals[task_name] = interval

        # 2. Technology research tasks
        tech_research_tasks = {}
        for tech_name in self.required_techs:
            tech = self.technologies.get(tech_name)
            if not tech:
                continue

            task_name = f"tech_{tech_name}"
            duration = tech.cost.duration_seconds // self.time_scale_seconds

            start = model.NewIntVar(0, max_horizon, f"start_{task_name}")
            end = model.NewIntVar(0, max_horizon, f"end_{task_name}")
            interval = model.NewIntervalVar(
                start, duration, end, f"interval_{task_name}"
            )

            tech_research_tasks[tech_name] = {
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
                "tech": tech,
                "type": "tech_research",
            }
            research_starts[task_name] = start
            research_ends[task_name] = end
            research_intervals[task_name] = interval

        # Constraint: Sequential upgrades for same building (including Library)
        for building_type, target_level in self.target_levels.items():
            current_level = self.initial_state.building_levels.get(building_type, 1)
            for level in range(current_level + 1, target_level):
                if (
                    building_type,
                    level,
                ) in building_tasks and (building_type, level + 1) in building_tasks:
                    model.Add(
                        building_tasks[(building_type, level)]["end"]
                        <= building_tasks[(building_type, level + 1)]["start"]
                    )

        # Constraint: Sequential library upgrades
        for level in range(current_lib_level + 1, self.required_library_level):
            if level in library_upgrade_tasks and (level + 1) in library_upgrade_tasks:
                model.Add(
                    library_upgrade_tasks[level]["end"]
                    <= library_upgrade_tasks[level + 1]["start"]
                )

        # Constraint: No overlap on building queue
        if building_intervals:
            model.AddNoOverlap(list(building_intervals.values()))

        # Constraint: No overlap on research queue (library upgrades + tech research)
        if research_intervals:
            model.AddNoOverlap(list(research_intervals.values()))

        # Constraint: Technology research requires library level
        for task in tech_research_tasks.values():
            tech = task["tech"]
            required_lib_level = tech.library_level_required

            # Tech research can only start after library reaches required level
            if required_lib_level in library_upgrade_tasks:
                model.Add(
                    library_upgrade_tasks[required_lib_level]["end"] <= task["start"]
                )

        # Constraint: Building upgrades that require technologies
        for building_type, target_level in self.target_levels.items():
            building = self.buildings.get(building_type)
            if not building:
                continue

            current_level = self.initial_state.building_levels.get(building_type, 1)

            for level in range(current_level + 1, target_level + 1):
                if level in building.technology_prerequisites:
                    tech_name = building.technology_prerequisites[level]

                    # Building upgrade requires tech to be researched first
                    if (
                        tech_name in tech_research_tasks
                        and (building_type, level) in building_tasks
                    ):
                        model.Add(
                            tech_research_tasks[tech_name]["end"]
                            <= building_tasks[(building_type, level)]["start"]
                        )

        # Objective: Minimize makespan (total time)
        makespan = model.NewIntVar(0, max_horizon, "makespan")

        # Makespan is max of all end times
        all_ends = list(building_ends.values()) + list(research_ends.values())
        if all_ends:
            model.AddMaxEquality(makespan, all_ends)

        model.Minimize(makespan)

        # Solve
        solver = cp_model.CpSolver()
        solver.parameters.max_time_in_seconds = time_limit_seconds
        solver.parameters.log_search_progress = False

        status = solver.Solve(model)

        if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
            return self._extract_solution(
                solver, building_tasks, library_upgrade_tasks, tech_research_tasks
            )

        return None

    def _extract_solution(
        self,
        solver: cp_model.CpSolver,
        building_tasks: dict,
        library_upgrade_tasks: dict,
        tech_research_tasks: dict,
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

        # Extract library upgrades and tech research
        research_actions = []

        # Library upgrades
        for level, task_data in library_upgrade_tasks.items():
            start_time = solver.Value(task_data["start"]) * self.time_scale_seconds
            end_time = solver.Value(task_data["end"]) * self.time_scale_seconds

            action = LibraryResearchAction(
                from_level=level - 1,
                to_level=level,
                start_time=start_time,
                end_time=end_time,
                costs=task_data["costs"],
                # Library upgrade doesn't unlock tech, research does
                technologies_unlocked=[],
            )
            research_actions.append(action)

        # Technology research
        for _tech_name, task_data in tech_research_tasks.items():
            start_time = solver.Value(task_data["start"]) * self.time_scale_seconds
            end_time = solver.Value(task_data["end"]) * self.time_scale_seconds

            tech = task_data["tech"]

            # Create a special research action for technology
            action = LibraryResearchAction(
                from_level=0,  # Not a level upgrade
                to_level=0,
                start_time=start_time,
                end_time=end_time,
                costs=task_data["costs"],
                technologies_unlocked=[tech.name],
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
            resources={rt: 0.0 for rt in ResourceType},  # Not tracking resources here
            time_elapsed=total_time,
        )

        return Solution(
            building_actions=building_actions,
            research_actions=research_actions,
            total_time_seconds=total_time,
            final_state=final_state,
        )
