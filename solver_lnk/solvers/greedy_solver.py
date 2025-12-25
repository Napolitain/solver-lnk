"""Greedy simulation-based solver with accurate resource tracking."""

from dataclasses import dataclass

from solver_lnk.models import (
    Building,
    BuildingType,
    BuildingUpgradeAction,
    GameState,
    ResourceType,
    Solution,
)


@dataclass
class SimulationState:
    """Current state during simulation."""

    time_minutes: int
    resources: dict[ResourceType, float]
    building_levels: dict[BuildingType, int]
    production_rates: dict[ResourceType, float]  # per hour
    storage_caps: dict[ResourceType, int]
    building_queue_free_at: int  # minutes
    research_queue_free_at: int  # minutes
    completed_actions: list[BuildingUpgradeAction]


class GreedySolver:
    """
    Greedy solver that simulates time progression with continuous resource accumulation.

    Approach:
    1. Prioritize resource production buildings first
    2. Build storages as needed
    3. Build other buildings last
    4. Accurately simulate resource production over time
    5. Wait for resources if needed
    """

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
    ):
        """Initialize solver."""
        self.buildings = buildings
        self.initial_state = initial_state
        self.target_levels = target_levels

    def solve(self) -> Solution | None:
        """Solve using greedy simulation."""
        # Initialize simulation state
        state = SimulationState(
            time_minutes=0,
            resources=dict(self.initial_state.resources),
            building_levels=dict(self.initial_state.building_levels),
            production_rates=self._calculate_production_rates(
                self.initial_state.building_levels
            ),
            storage_caps=self._calculate_storage_capacities(
                self.initial_state.building_levels
            ),
            building_queue_free_at=0,
            research_queue_free_at=0,
            completed_actions=[],
        )

        # Create upgrade queue (prioritized)
        upgrade_queue = self._create_prioritized_queue()

        # Simulate
        while upgrade_queue:
            next_upgrade = upgrade_queue[0]
            btype, target_level = next_upgrade

            current_level = state.building_levels.get(btype, 1)
            if current_level >= target_level:
                upgrade_queue.pop(0)
                continue

            # Try to upgrade to next level
            from_level = current_level
            to_level = current_level + 1

            building = self.buildings.get(btype)
            if not building:
                upgrade_queue.pop(0)
                continue

            level_data = building.get_level_data(to_level)
            if not level_data:
                upgrade_queue.pop(0)
                continue

            costs = level_data.costs
            duration_sec = level_data.build_time_seconds

            # Check if we need to wait for queue
            if state.time_minutes < state.building_queue_free_at:
                # Fast forward to when queue is free
                self._advance_time(
                    state, state.building_queue_free_at - state.time_minutes
                )

            # Check if we have enough resources
            can_afford, wait_time = self._can_afford_or_wait_time(state, costs)

            if not can_afford:
                # Need to wait for resources to accumulate
                self._advance_time(state, wait_time)

            # Check storage capacity
            storage_ok, storage_needed = self._check_storage_capacity(state, costs)
            if not storage_ok:
                # Need to upgrade storage first - insert at front
                upgrade_queue.insert(0, storage_needed)
                continue

            # Start the upgrade
            start_time = state.time_minutes

            # Deduct resources
            for resource_type, cost in costs.items():
                state.resources[resource_type] -= cost

            # Mark queue as busy
            duration_minutes = max(1, duration_sec // 60)
            state.building_queue_free_at = state.time_minutes + duration_minutes

            # Advance time to completion
            self._advance_time(state, duration_minutes)

            # Complete upgrade
            state.building_levels[btype] = to_level

            # Update production rates if this is a production building
            if to_level <= target_level:
                prod_building_types = [
                    BuildingType.LUMBERJACK,
                    BuildingType.QUARRY,
                    BuildingType.ORE_MINE,
                    BuildingType.FARM,
                ]
                if btype in prod_building_types:
                    new_level_data = building.get_level_data(to_level)
                    if new_level_data and new_level_data.production_rate:
                        # Map building type to resource type
                        resource_map = {
                            BuildingType.LUMBERJACK: ResourceType.WOOD,
                            BuildingType.QUARRY: ResourceType.STONE,
                            BuildingType.ORE_MINE: ResourceType.IRON,
                            BuildingType.FARM: ResourceType.FOOD,
                        }
                        res_type = resource_map.get(btype)
                        if res_type:
                            state.production_rates[res_type] = (
                                new_level_data.production_rate
                            )

            # Update storage capacity if this is a storage building
            storage_types = [
                BuildingType.WOOD_STORE,
                BuildingType.STONE_STORE,
                BuildingType.ORE_STORE,
            ]
            if btype in storage_types:
                new_level_data = building.get_level_data(to_level)
                if new_level_data and new_level_data.storage_capacity:
                    # Map building type to resource type
                    storage_map = {
                        BuildingType.WOOD_STORE: ResourceType.WOOD,
                        BuildingType.STONE_STORE: ResourceType.STONE,
                        BuildingType.ORE_STORE: ResourceType.IRON,
                    }
                    res_type = storage_map.get(btype)
                    if res_type:
                        state.storage_caps[res_type] = (
                            new_level_data.storage_capacity
                        )

            # Record action
            state.completed_actions.append(
                BuildingUpgradeAction(
                    building_type=btype,
                    from_level=from_level,
                    to_level=to_level,
                    start_time=start_time * 60,  # Convert to seconds
                    end_time=state.time_minutes * 60,
                    costs=costs,
                )
            )

            # Check if we reached target for this building
            if state.building_levels[btype] >= target_level:
                upgrade_queue.pop(0)

        return Solution(
            building_actions=state.completed_actions,
            research_actions=[],
            total_time_seconds=state.time_minutes * 60,
            final_state=GameState(
                building_levels=state.building_levels,
                resources=state.resources,
            ),
        )

    def _advance_time(self, state: SimulationState, minutes: int):
        """Advance time and accumulate resources."""
        hours = minutes / 60.0
        state.time_minutes += minutes

        for resource_type, rate_per_hour in state.production_rates.items():
            produced = rate_per_hour * hours
            state.resources[resource_type] += produced

            # Cap at storage limit
            cap = state.storage_caps.get(resource_type, 999999)
            state.resources[resource_type] = min(state.resources[resource_type], cap)

    def _can_afford_or_wait_time(
        self, state: SimulationState, costs: dict[ResourceType, int]
    ) -> tuple[bool, int]:
        """
        Check if we can afford costs. If not, return wait time needed.

        Returns: (can_afford, wait_minutes_needed)
        """
        max_wait_minutes = 0

        for resource_type, cost in costs.items():
            available = state.resources.get(resource_type, 0)

            if available >= cost:
                continue

            # Need to wait for production
            shortfall = cost - available
            production_rate = state.production_rates.get(resource_type, 0)

            if production_rate <= 0:
                # Cannot produce this resource!
                return (False, 999999)

            # Time needed = shortfall / (rate_per_hour) * 60
            hours_needed = shortfall / production_rate
            minutes_needed = int(hours_needed * 60) + 1  # Round up

            max_wait_minutes = max(max_wait_minutes, minutes_needed)

        if max_wait_minutes > 0:
            return (False, max_wait_minutes)

        return (True, 0)

    def _check_storage_capacity(
        self, state: SimulationState, costs: dict[ResourceType, int]
    ) -> tuple[bool, tuple[BuildingType, int] | None]:
        """
        Check if storage is sufficient for costs.

        Returns: (storage_ok, (storage_building, target_level) if upgrade needed)
        """
        storage_map = {
            ResourceType.WOOD: BuildingType.WOOD_STORE,
            ResourceType.STONE: BuildingType.STONE_STORE,
            ResourceType.IRON: BuildingType.ORE_STORE,
        }

        for resource_type, cost in costs.items():
            cap = state.storage_caps.get(resource_type, 999999)

            if cost > cap:
                # Need more storage
                storage_building = storage_map.get(resource_type)
                if not storage_building:
                    continue

                # Find what level we need
                building = self.buildings.get(storage_building)
                if not building:
                    continue

                current_level = state.building_levels.get(storage_building, 1)

                # Find first level that has enough capacity
                for level in range(current_level + 1, 31):
                    level_data = building.get_level_data(level)
                    if not level_data or level_data.storage_capacity is None:
                        continue

                    new_cap = level_data.storage_capacity
                    if new_cap >= cost:
                        return (False, (storage_building, level))

                # Even max level not enough - problem!
                return (False, None)

        return (True, None)

    def _create_prioritized_queue(self) -> list[tuple[BuildingType, int]]:
        """
        Create prioritized list of upgrades.

        Priority:
        1. Resource production (Lumberjack, Quarry, Ore Mine, Farm)
        2. Storage (Wood Store, Stone Store, Ore Store)
        3. Core buildings (Keep, Library, etc.)
        4. Military (Arsenal, Fortifications, etc.)
        """
        priority_groups = [
            [
                BuildingType.LUMBERJACK,
                BuildingType.QUARRY,
                BuildingType.ORE_MINE,
                BuildingType.FARM,
            ],
            [
                BuildingType.WOOD_STORE,
                BuildingType.STONE_STORE,
                BuildingType.ORE_STORE,
            ],
            [BuildingType.KEEP, BuildingType.LIBRARY],
            [
                BuildingType.ARSENAL,
                BuildingType.TAVERN,
                BuildingType.MARKET,
                BuildingType.FORTIFICATIONS,
            ],
        ]

        queue = []

        for group in priority_groups:
            for btype in group:
                if btype in self.target_levels:
                    target = self.target_levels[btype]
                    queue.append((btype, target))

        return queue

    def _calculate_production_rates(
        self, building_levels: dict[BuildingType, int]
    ) -> dict[ResourceType, float]:
        """Calculate current production rates."""
        rates = {
            ResourceType.WOOD: 0.0,
            ResourceType.STONE: 0.0,
            ResourceType.IRON: 0.0,
            ResourceType.FOOD: 0.0,
        }

        production_buildings = {
            BuildingType.LUMBERJACK: ResourceType.WOOD,
            BuildingType.QUARRY: ResourceType.STONE,
            BuildingType.ORE_MINE: ResourceType.IRON,
            BuildingType.FARM: ResourceType.FOOD,
        }

        for btype, resource in production_buildings.items():
            level = building_levels.get(btype, 1)
            building = self.buildings.get(btype)
            if not building:
                continue

            level_data = building.get_level_data(level)
            if level_data and level_data.production_rate is not None:
                rates[resource] = level_data.production_rate

        return rates

    def _calculate_storage_capacities(
        self, building_levels: dict[BuildingType, int]
    ) -> dict[ResourceType, int]:
        """Calculate current storage capacities."""
        caps = {
            ResourceType.WOOD: 999999,
            ResourceType.STONE: 999999,
            ResourceType.IRON: 999999,
            ResourceType.FOOD: 999999,
        }

        storage_buildings = {
            BuildingType.WOOD_STORE: ResourceType.WOOD,
            BuildingType.STONE_STORE: ResourceType.STONE,
            BuildingType.ORE_STORE: ResourceType.IRON,
        }

        for btype, resource in storage_buildings.items():
            level = building_levels.get(btype, 1)
            building = self.buildings.get(btype)
            if not building:
                continue

            level_data = building.get_level_data(level)
            if level_data and level_data.storage_capacity is not None:
                caps[resource] = level_data.storage_capacity

        return caps
