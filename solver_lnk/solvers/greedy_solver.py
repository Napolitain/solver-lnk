"""Greedy build order solver with priority-based heuristics."""

from copy import deepcopy
from dataclasses import dataclass
from typing import ClassVar

from solver_lnk.models import Building, BuildingType, GameState, ResourceType


@dataclass
class UpgradeAction:
    """Represents a scheduled building upgrade."""

    building_type: BuildingType
    from_level: int
    to_level: int
    start_time: float  # seconds
    end_time: float  # seconds
    costs: dict[ResourceType, int]


class GreedyBuildOrderSolver:
    """Greedy solver using priority-based heuristics."""

    # Priority order: lower number = higher priority
    PRIORITY_MAP: ClassVar[dict[BuildingType, int]] = {
        BuildingType.LUMBERJACK: 1,
        BuildingType.QUARRY: 2,
        BuildingType.MINE: 3,
        BuildingType.FARM: 4,
        BuildingType.WOOD_STORE: 10,
        BuildingType.STONE_STORE: 11,
        BuildingType.ORE_STORE: 12,
        BuildingType.CASTLE: 20,
        BuildingType.KEEP: 21,
        BuildingType.ARSENAL: 22,
        BuildingType.LIBRARY: 23,
        BuildingType.TAVERN: 24,
        BuildingType.MARKET: 25,
        BuildingType.FORTIFICATIONS: 30,
    }

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
    ):
        self.buildings = buildings
        self.initial_state = deepcopy(initial_state)
        self.target_levels = target_levels

    def solve(self) -> list[UpgradeAction]:
        """
        Solve using greedy approach.

        Always upgrades highest priority affordable building.
        Returns list of upgrade actions in chronological order.
        """
        state = deepcopy(self.initial_state)
        actions: list[UpgradeAction] = []

        while not self._is_target_reached(state):
            # Get next upgrade to do
            next_upgrade = self._get_next_upgrade(state)

            if next_upgrade is None:
                # No more upgrades possible, wait for resources
                wait_time = self._calculate_wait_time(state, self.target_levels)
                if wait_time is None or wait_time > 1e9:  # Unreachable
                    break

                # Wait and accumulate resources
                production_rates = state.get_production_rates(self.buildings)
                state.update_resources(wait_time, production_rates)
                continue

            # Execute the upgrade
            building_type, target_level = next_upgrade
            current_level = state.building_levels.get(building_type, 0)

            building = self.buildings[building_type]
            level_data = building.get_level_data(target_level)
            if level_data is None:
                continue

            # Wait until we can afford it
            wait_time = self._time_until_affordable(state, level_data.costs)
            if wait_time > 0:
                production_rates = state.get_production_rates(self.buildings)
                state.update_resources(wait_time, production_rates)

            # Schedule the upgrade
            start_time = state.time_elapsed
            end_time = start_time + level_data.build_time_seconds

            action = UpgradeAction(
                building_type=building_type,
                from_level=current_level,
                to_level=target_level,
                start_time=start_time,
                end_time=end_time,
                costs=level_data.costs,
            )
            actions.append(action)

            # Update state
            state.spend_resources(level_data.costs)
            state.building_levels[building_type] = target_level

            # Advance time (building completes)
            production_rates = state.get_production_rates(self.buildings)
            state.update_resources(level_data.build_time_seconds, production_rates)

        return actions

    def _is_target_reached(self, state: GameState) -> bool:
        """Check if target building levels are reached."""
        for building_type, target_level in self.target_levels.items():
            current_level = state.building_levels.get(building_type, 0)
            if current_level < target_level:
                return False
        return True

    def _get_next_upgrade(self, state: GameState) -> tuple[BuildingType, int] | None:
        """Get the next building to upgrade based on priority and affordability."""
        candidates: list[tuple[BuildingType, int, int]] = []

        for building_type, target_level in self.target_levels.items():
            current_level = state.building_levels.get(building_type, 0)
            if current_level >= target_level:
                continue

            next_level = current_level + 1
            building = self.buildings.get(building_type)
            if building is None:
                continue

            if not building.can_upgrade_to(next_level, state.building_levels):
                continue

            level_data = building.get_level_data(next_level)
            if level_data is None:
                continue

            priority = self.PRIORITY_MAP.get(building_type, 100)
            candidates.append((building_type, next_level, priority))

        if not candidates:
            return None

        # Sort by priority (lower = better), return highest priority
        candidates.sort(key=lambda x: x[2])
        return (candidates[0][0], candidates[0][1])

    def _time_until_affordable(
        self, state: GameState, costs: dict[ResourceType, int]
    ) -> float:
        """Calculate time in seconds until we can afford the costs."""
        if state.can_afford(costs):
            return 0.0

        production_rates = state.get_production_rates(self.buildings)

        max_wait_time = 0.0
        for resource_type, cost in costs.items():
            available = state.resources.get(resource_type, 0)
            needed = cost - available

            if needed <= 0:
                continue

            rate = production_rates.get(resource_type, 0)
            if rate <= 0:
                # Cannot produce this resource, infinite wait
                return float("inf")

            wait_hours = needed / rate
            wait_seconds = wait_hours * 3600.0
            max_wait_time = max(max_wait_time, wait_seconds)

        return max_wait_time

    def _calculate_wait_time(
        self, state: GameState, targets: dict[BuildingType, int]
    ) -> float | None:
        """Calculate minimum wait time to make progress."""
        min_wait = float("inf")

        for building_type, target_level in targets.items():
            current_level = state.building_levels.get(building_type, 0)
            if current_level >= target_level:
                continue

            next_level = current_level + 1
            building = self.buildings.get(building_type)
            if building is None:
                continue

            level_data = building.get_level_data(next_level)
            if level_data is None:
                continue

            wait = self._time_until_affordable(state, level_data.costs)
            min_wait = min(min_wait, wait)

        return min_wait if min_wait != float("inf") else None
