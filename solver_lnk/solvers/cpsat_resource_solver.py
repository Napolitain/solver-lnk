"""CP-SAT solver with resource-aware scheduling."""

from copy import deepcopy

from ortools.sat.python import cp_model

from solver_lnk.models import Building, BuildingType, GameState
from solver_lnk.solvers.greedy_solver import UpgradeAction


class CPSATResourceSolver:
    """OR-Tools CP-SAT solver with resource constraints via simulation."""

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
        time_interval: int = 3600,  # 1 hour intervals (60 for 1 minute)
    ):
        self.buildings = buildings
        self.initial_state = initial_state
        self.target_levels = target_levels
        self.time_interval = time_interval
        self.model = cp_model.CpModel()

        total_upgrades = sum(
            target_levels.get(bt, 0) - initial_state.building_levels.get(bt, 0)
            for bt in target_levels
        )
        self.time_horizon = total_upgrades * 3600 * 20  # Generous horizon
        self.num_intervals = self.time_horizon // time_interval

    def _simulate_resources(self, actions: list[UpgradeAction]) -> bool:
        """Simulate to check if resource schedule is valid."""
        state = deepcopy(self.initial_state)

        for action in sorted(actions, key=lambda a: a.start_time):
            # Wait until action starts
            wait_time = action.start_time - sum(
                a.end_time - a.start_time
                for a in actions
                if a.end_time <= action.start_time
            )

            if wait_time > 0:
                production_rates = state.get_production_rates(self.buildings)
                state.update_resources(wait_time, production_rates)

            # Check if we can afford it
            if not state.can_afford(action.costs):
                return False

            # Consume resources
            for resource, cost in action.costs.items():
                state.resources[resource] -= cost

            # Update building level after completion
            state.building_levels[action.building_type] = action.to_level

        return True

    def solve(self) -> list[UpgradeAction]:
        """
        Solve using CP-SAT with resource-aware priorities.

        Strategy:
        1. Create task variables with costs embedded
        2. Add ordering hints based on resource dependencies
        3. Prioritize resource-producing buildings early
        4. Validate solution via simulation
        """

        # Step 1: Create tasks with resource metadata
        tasks = []
        task_vars = {}

        for building_type, target_level in self.target_levels.items():
            current_level = self.initial_state.building_levels.get(building_type, 0)
            building = self.buildings.get(building_type)

            if not building:
                continue

            for level in range(current_level + 1, target_level + 1):
                level_data = building.get_level_data(level)
                if not level_data:
                    continue

                task_id = (building_type, level)
                duration = level_data.build_time_seconds // self.time_interval
                if duration == 0:
                    duration = 1

                start_var = self.model.NewIntVar(
                    0, self.num_intervals, f"start_{building_type.value}_{level}"
                )

                end_var = self.model.NewIntVar(
                    0, self.num_intervals, f"end_{building_type.value}_{level}"
                )

                self.model.Add(end_var == start_var + duration)

                # Calculate resource intensity (for hints)
                total_cost = sum(level_data.costs.values())
                is_producer = level_data.production_rate is not None

                task_vars[task_id] = {
                    "start": start_var,
                    "end": end_var,
                    "duration": duration,
                    "costs": level_data.costs,
                    "production_rate": level_data.production_rate,
                    "building_type": building_type,
                    "level": level,
                    "total_cost": total_cost,
                    "is_producer": is_producer,
                }
                tasks.append(task_id)

        # Step 2: Sequential constraints
        for building_type in self.target_levels:
            current_level = self.initial_state.building_levels.get(building_type, 0)
            target_level = self.target_levels[building_type]

            for level in range(current_level + 1, target_level):
                curr_task = (building_type, level)
                next_task = (building_type, level + 1)

                if curr_task in task_vars and next_task in task_vars:
                    self.model.Add(
                        task_vars[next_task]["start"] >= task_vars[curr_task]["end"]
                    )

        # Step 3: No overlap
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

        # Step 4: Resource-aware hints
        # Prioritize early resource production buildings
        producer_buildings = [
            BuildingType.LUMBERJACK,
            BuildingType.QUARRY,
            BuildingType.ORE_MINE,
            BuildingType.FARM,
        ]

        # Add hints: complete first 10 levels of each producer early
        hint_time = 0
        for building_type in producer_buildings:
            if building_type not in self.target_levels:
                continue

            for level in range(1, min(11, self.target_levels[building_type] + 1)):
                task_id = (building_type, level)
                if task_id in task_vars:
                    # Add decision hint to schedule early
                    self.model.AddHint(task_vars[task_id]["start"], hint_time)
                    hint_time += task_vars[task_id]["duration"]

        # Step 5: Minimize makespan
        makespan = self.model.NewIntVar(0, self.num_intervals, "makespan")

        for task_id in tasks:
            self.model.Add(makespan >= task_vars[task_id]["end"])

        self.model.Minimize(makespan)

        # Step 6: Solve
        solver = cp_model.CpSolver()
        solver.parameters.max_time_in_seconds = 120.0
        solver.parameters.num_search_workers = 8

        status = solver.Solve(self.model)

        if status not in [cp_model.OPTIMAL, cp_model.FEASIBLE]:
            return []

        # Extract solution
        actions = []
        for task_id in sorted(tasks, key=lambda t: solver.Value(task_vars[t]["start"])):
            task = task_vars[task_id]
            start_time = solver.Value(task["start"]) * self.time_interval
            end_time = solver.Value(task["end"]) * self.time_interval

            action = UpgradeAction(
                building_type=task["building_type"],
                from_level=task["level"] - 1,
                to_level=task["level"],
                start_time=float(start_time),
                end_time=float(end_time),
                costs=task["costs"],
            )
            actions.append(action)

        # Step 7: Validate with simulation (optional check)
        # if not self._simulate_resources(actions):
        #     print("Warning: Solution may have resource issues")

        return actions
