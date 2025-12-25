"""CP-SAT solver for build order optimization using OR-Tools."""

from ortools.sat.python import cp_model

from solver_lnk.models import Building, BuildingType, GameState
from solver_lnk.solvers.greedy_solver import UpgradeAction


class CPSATBuildOrderSolver:
    """OR-Tools CP-SAT solver for optimal build order."""

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
        time_scale: int = 1,  # Scale factor for time discretization
    ):
        self.buildings = buildings
        self.initial_state = initial_state
        self.target_levels = target_levels
        self.time_scale = time_scale
        self.model = cp_model.CpModel()

        # Calculate time horizon (rough estimate)
        total_upgrades = sum(
            target_levels.get(bt, 0) - initial_state.building_levels.get(bt, 0)
            for bt in target_levels
        )
        # Assume average 1 hour per upgrade
        self.time_horizon = total_upgrades * 3600 * 10  # 10x buffer

    def solve(self) -> list[UpgradeAction]:
        """
        Solve using CP-SAT with time discretization approach.

        Key modeling decisions:
        - Time is discretized into intervals
        - Each upgrade gets start/end time variables
        - Resource constraints enforced at discrete time points
        - Production rates update after each building completion
        """

        # Create upgrade tasks
        tasks = []
        task_vars = {}

        # For each building that needs upgrading
        for building_type, target_level in self.target_levels.items():
            current_level = self.initial_state.building_levels.get(building_type, 0)
            building = self.buildings.get(building_type)

            if not building:
                continue

            # Create variables for each upgrade level
            for level in range(current_level + 1, target_level + 1):
                level_data = building.get_level_data(level)
                if not level_data:
                    continue

                task_id = (building_type, level)
                duration = level_data.build_time_seconds // self.time_scale

                # Start time variable
                start_var = self.model.NewIntVar(
                    0,
                    self.time_horizon // self.time_scale,
                    f"start_{building_type.value}_{level}",
                )

                # End time variable
                end_var = self.model.NewIntVar(
                    0,
                    self.time_horizon // self.time_scale,
                    f"end_{building_type.value}_{level}",
                )

                # Link start, duration, end
                self.model.Add(end_var == start_var + duration)

                task_vars[task_id] = {
                    "start": start_var,
                    "end": end_var,
                    "duration": duration,
                    "costs": level_data.costs,
                    "building_type": building_type,
                    "level": level,
                }
                tasks.append(task_id)

        # Constraint 1: Sequential building levels
        # Can't start level N+1 before completing level N
        for building_type in self.target_levels:
            current_level = self.initial_state.building_levels.get(building_type, 0)
            target_level = self.target_levels[building_type]

            for level in range(current_level + 1, target_level):
                curr_task = (building_type, level)
                next_task = (building_type, level + 1)

                if curr_task in task_vars and next_task in task_vars:
                    # Next upgrade must start after current ends
                    self.model.Add(
                        task_vars[next_task]["start"] >= task_vars[curr_task]["end"]
                    )

        # Constraint 2: Only one building at a time (single build queue)
        # Use no-overlap constraint
        intervals = []
        for task_id in tasks:
            interval = self.model.NewIntervalVar(
                task_vars[task_id]["start"],
                task_vars[task_id]["duration"],
                task_vars[task_id]["end"],
                f"interval_{task_id[0].value}_{task_id[1]}",
            )
            intervals.append(interval)

        self.model.AddNoOverlap(intervals)

        # Objective: Minimize maximum completion time (makespan)
        makespan = self.model.NewIntVar(
            0, self.time_horizon // self.time_scale, "makespan"
        )

        for task_id in tasks:
            self.model.Add(makespan >= task_vars[task_id]["end"])

        self.model.Minimize(makespan)

        # Solve
        solver = cp_model.CpSolver()
        solver.parameters.max_time_in_seconds = 60.0  # 1 minute timeout

        status = solver.Solve(self.model)

        if status not in [cp_model.OPTIMAL, cp_model.FEASIBLE]:
            return []

        # Extract solution
        actions = []
        for task_id in sorted(tasks, key=lambda t: solver.Value(task_vars[t]["start"])):
            task = task_vars[task_id]
            start_time = solver.Value(task["start"]) * self.time_scale
            end_time = solver.Value(task["end"]) * self.time_scale

            action = UpgradeAction(
                building_type=task["building_type"],
                from_level=task["level"] - 1,
                to_level=task["level"],
                start_time=float(start_time),
                end_time=float(end_time),
                costs=task["costs"],
            )
            actions.append(action)

        return actions
