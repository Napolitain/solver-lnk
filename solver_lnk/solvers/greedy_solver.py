"""Greedy simulation-based solver with accurate resource tracking."""

from dataclasses import dataclass, field

from solver_lnk.models import (
    Building,
    BuildingType,
    BuildingUpgradeAction,
    GameState,
    LibraryResearchAction,
    ResourceType,
    Solution,
    Technology,
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
    research_actions: list[LibraryResearchAction] = field(default_factory=list)
    researched_technologies: set[str] = field(default_factory=set)


class GreedySolver:
    """
    Greedy solver that simulates time progression with continuous resource accumulation.

    Approach:
    1. Prioritize resource production buildings first
    2. Build storages as needed
    3. Build other buildings last
    4. Accurately simulate resource production over time
    5. Wait for resources if needed
    6. Research technologies in parallel when needed for building upgrades
    """

    def __init__(
        self,
        buildings: dict[BuildingType, Building],
        initial_state: GameState,
        target_levels: dict[BuildingType, int],
        technologies: dict[str, Technology] | None = None,
    ):
        """Initialize solver."""
        self.buildings = buildings
        self.initial_state = initial_state
        self.target_levels = target_levels
        self.technologies = technologies or {}

    def solve(self) -> Solution | None:
        """Solve using greedy simulation with dynamic production building selection."""
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
            # Check if we need to wait for queue
            if state.time_minutes < state.building_queue_free_at:
                self._advance_time(
                    state, state.building_queue_free_at - state.time_minutes
                )

            # Find next upgrade to do
            next_upgrade = self._select_next_upgrade(state, upgrade_queue)

            if next_upgrade is None:
                # No valid upgrade found
                break

            btype, target_level, queue_idx = next_upgrade

            current_level = state.building_levels.get(btype, 1)
            if current_level >= target_level:
                upgrade_queue.pop(queue_idx)
                continue

            # Try to upgrade to next level
            from_level = current_level
            to_level = current_level + 1

            building = self.buildings.get(btype)
            if not building:
                upgrade_queue.pop(queue_idx)
                continue

            level_data = building.get_level_data(to_level)
            if not level_data:
                upgrade_queue.pop(queue_idx)
                continue

            costs = level_data.costs
            duration_sec = level_data.build_time_seconds

            # Check technology prerequisites (e.g., Farm 15 needs Crop rotation)
            tech_needed = self._check_technology_prerequisite(state, building, to_level)
            if tech_needed is not None:
                # Need to research technology first
                self._schedule_research(state, tech_needed, upgrade_queue)
                continue

            # Check storage capacity FIRST (includes food/Farm capacity)
            storage_ok, storage_needed = self._check_storage_capacity(state, costs)
            if not storage_ok:
                if storage_needed is not None:
                    # Need to upgrade storage first - insert at front
                    upgrade_queue.insert(0, storage_needed)
                continue

            # Check if we have enough resources
            can_afford, wait_time = self._can_afford_or_wait_time(state, costs)

            if not can_afford:
                if wait_time < 0:
                    # Cannot produce resource (e.g., food) - storage check should handle
                    upgrade_queue.pop(queue_idx)
                    continue
                # Need to wait for resources to accumulate
                self._advance_time(state, wait_time)

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
                BuildingType.FARM,  # Farm provides food capacity
            ]
            if btype in storage_types:
                new_level_data = building.get_level_data(to_level)
                if new_level_data and new_level_data.storage_capacity:
                    # Map building type to resource type
                    storage_map = {
                        BuildingType.WOOD_STORE: ResourceType.WOOD,
                        BuildingType.STONE_STORE: ResourceType.STONE,
                        BuildingType.ORE_STORE: ResourceType.IRON,
                        BuildingType.FARM: ResourceType.FOOD,
                    }
                    res_type = storage_map.get(btype)
                    if res_type:
                        new_cap = int(new_level_data.storage_capacity)
                        state.storage_caps[res_type] = new_cap
                        # For FOOD: upgrading Farm refills workers to new capacity
                        if res_type == ResourceType.FOOD:
                            state.resources[ResourceType.FOOD] = float(new_cap)

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
                upgrade_queue.pop(queue_idx)

        return Solution(
            building_actions=state.completed_actions,
            research_actions=state.research_actions,
            total_time_seconds=state.time_minutes * 60,
            final_state=GameState(
                building_levels=state.building_levels,
                resources=state.resources,
                researched_technologies=state.researched_technologies,
            ),
        )

    def _advance_time(self, state: SimulationState, minutes: int) -> None:
        """Advance time and accumulate resources."""
        hours = minutes / 60.0
        state.time_minutes += minutes

        for resource_type, rate_per_hour in state.production_rates.items():
            produced = rate_per_hour * hours
            state.resources[resource_type] += produced

            # Cap at storage limit
            cap = state.storage_caps.get(resource_type, 999999)
            state.resources[resource_type] = min(state.resources[resource_type], cap)

    def _check_technology_prerequisite(
        self,
        state: SimulationState,
        building: Building,
        to_level: int,
    ) -> str | None:
        """
        Check if a technology is required for this upgrade.

        Returns: technology name if needed and not researched, else None
        """
        tech_name = building.technology_prerequisites.get(to_level)
        if tech_name is None:
            return None

        if tech_name in state.researched_technologies:
            return None

        return tech_name

    def _schedule_research(
        self,
        state: SimulationState,
        tech_name: str,
        upgrade_queue: list[tuple[BuildingType, int]],
    ) -> None:
        """
        Schedule research for a technology.

        This handles:
        1. Ensuring Library is at required level
        2. Waiting for research queue to be free
        3. Waiting for resources
        4. Performing the research
        """
        tech = self.technologies.get(tech_name)
        if tech is None:
            # Technology not found, skip
            return

        # Check if Library needs upgrading
        library_level = state.building_levels.get(BuildingType.LIBRARY, 1)
        if library_level < tech.required_library_level:
            # Insert Library upgrade at front of queue
            upgrade_queue.insert(0, (BuildingType.LIBRARY, tech.required_library_level))
            return

        # Wait for research queue to be free
        if state.time_minutes < state.research_queue_free_at:
            self._advance_time(state, state.research_queue_free_at - state.time_minutes)

        # Check storage for research costs
        storage_ok, storage_needed = self._check_storage_capacity(state, tech.costs)
        if not storage_ok:
            if storage_needed is not None:
                upgrade_queue.insert(0, storage_needed)
            return

        # Wait for resources
        can_afford, wait_time = self._can_afford_or_wait_time(state, tech.costs)
        if not can_afford:
            if wait_time < 0:
                return  # Cannot afford, will retry later
            self._advance_time(state, wait_time)

        # Start research
        start_time = state.time_minutes

        # Deduct resources
        for resource_type, cost in tech.costs.items():
            state.resources[resource_type] -= cost

        # Calculate duration
        duration_minutes = max(1, tech.research_time_seconds // 60)
        state.research_queue_free_at = state.time_minutes + duration_minutes

        # Advance time to completion (research happens in parallel with building)
        # Don't advance time here - let the main loop handle it

        # Record research action
        state.research_actions.append(
            LibraryResearchAction(
                from_level=library_level,
                to_level=library_level,  # Library level doesn't change
                start_time=start_time * 60,
                end_time=(start_time + duration_minutes) * 60,
                costs=tech.costs,
                technologies_unlocked=[tech_name],
            )
        )

        # Mark technology as researched (will be available after research completes)
        # For simplicity, mark it now - the time constraint is handled by queue
        state.researched_technologies.add(tech_name)

    def _select_next_upgrade(
        self,
        state: SimulationState,
        upgrade_queue: list[tuple[BuildingType, int]],
    ) -> tuple[BuildingType, int, int] | None:
        """
        Select the next upgrade to perform.

        Priority order for production: LJ > Q > OM (wood > stone > iron).
        This ensures we maximize resource production in the right order.

        Storage/Farm upgrades (dynamically inserted) are processed immediately.

        Returns: (building_type, target_level, queue_index) or None
        """
        if not upgrade_queue:
            return None

        # Check if first item is a storage/Farm upgrade (dynamically inserted)
        # These should be processed immediately
        storage_types = {
            BuildingType.WOOD_STORE,
            BuildingType.STONE_STORE,
            BuildingType.ORE_STORE,
            BuildingType.FARM,
        }
        first_btype, first_target = upgrade_queue[0]
        if first_btype in storage_types:
            current = state.building_levels.get(first_btype, 1)
            if current < first_target:
                return (first_btype, first_target, 0)

        # Simply return first valid item in queue order
        # The queue is already ordered by priority (LJ, Q, OM interleaved)
        for idx, (btype, target_level) in enumerate(upgrade_queue):
            current_level = state.building_levels.get(btype, 1)
            if current_level >= target_level:
                continue
            return (btype, target_level, idx)

        return None

    def _can_afford_or_wait_time(
        self, state: SimulationState, costs: dict[ResourceType, int]
    ) -> tuple[bool, int]:
        """
        Check if we can afford costs. If not, return wait time needed.

        Note: FOOD cannot be waited for - it has no production.
        If we can't afford food, caller must upgrade Farm first.

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
                # Cannot produce this resource - need storage upgrade (Farm for food)
                # Return special value to signal storage check needed
                return (False, -1)

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
        Check if storage/capacity is sufficient for costs.

        For wood/stone/iron: checks if storage can hold enough to accumulate.
        For food: checks if we have enough workers (current food >= cost).

        Returns: (ok, (building, target_level) if upgrade needed)
        """
        storage_map = {
            ResourceType.WOOD: BuildingType.WOOD_STORE,
            ResourceType.STONE: BuildingType.STONE_STORE,
            ResourceType.IRON: BuildingType.ORE_STORE,
            ResourceType.FOOD: BuildingType.FARM,  # Farm provides food capacity
        }

        for resource_type, cost in costs.items():
            if resource_type == ResourceType.FOOD:
                # Food: check if we have enough workers available
                available = state.resources.get(ResourceType.FOOD, 0)
                if available < cost:
                    # Need more food capacity from Farm
                    building = self.buildings.get(BuildingType.FARM)
                    if not building:
                        continue

                    current_level = state.building_levels.get(BuildingType.FARM, 1)

                    # Find first level that gives enough capacity
                    for level in range(current_level + 1, 31):
                        level_data = building.get_level_data(level)
                        if not level_data or level_data.storage_capacity is None:
                            continue

                        new_cap = level_data.storage_capacity
                        if new_cap >= cost:
                            return (False, (BuildingType.FARM, level))

                    return (False, None)
            else:
                # Wood/Stone/Iron: check storage capacity for accumulation
                cap = state.storage_caps.get(resource_type, 999999)

                if cost > cap:
                    # Need more storage
                    storage_building = storage_map.get(resource_type)
                    if not storage_building:
                        continue

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

                    return (False, None)

        return (True, None)

    def _create_prioritized_queue(self) -> list[tuple[BuildingType, int]]:
        """
        Create prioritized list of upgrades with INTERLEAVED resource buildings.

        Instead of completing all lumberjack upgrades before quarry,
        we interleave them: LJ→2, Q→2, OM→2, LJ→3, Q→3, OM→3, etc.

        Priority order:
        1. Resource production (Lumberjack, Quarry, Ore Mine) - interleaved
        2. Storage (Wood/Stone/Ore Store) - interleaved
        3. Farm (may require technology research for levels 15/25/30)
        4. Core buildings (Keep, Library)
        5. Military and other (Arsenal, Tavern, Market, Fortifications)
        """
        queue: list[tuple[BuildingType, int]] = []

        # 1. Interleave resource production buildings
        resource_buildings = [
            BuildingType.LUMBERJACK,
            BuildingType.QUARRY,
            BuildingType.ORE_MINE,
        ]

        # Find max target level among resource buildings
        max_resource_level = 0
        for btype in resource_buildings:
            if btype in self.target_levels:
                max_resource_level = max(max_resource_level, self.target_levels[btype])

        # Add resource upgrades interleaved by level
        for level in range(2, max_resource_level + 1):
            for btype in resource_buildings:
                if btype in self.target_levels and level <= self.target_levels[btype]:
                    queue.append((btype, level))

        # 2. Storage buildings - interleaved
        storage_buildings = [
            BuildingType.WOOD_STORE,
            BuildingType.STONE_STORE,
            BuildingType.ORE_STORE,
        ]

        max_storage_level = 0
        for btype in storage_buildings:
            if btype in self.target_levels:
                max_storage_level = max(max_storage_level, self.target_levels[btype])

        for level in range(2, max_storage_level + 1):
            for btype in storage_buildings:
                if btype in self.target_levels and level <= self.target_levels[btype]:
                    queue.append((btype, level))

        # 3. Farm - added after storage, before core buildings
        # Farm provides food capacity and may require technologies for levels 15/25/30
        if BuildingType.FARM in self.target_levels:
            target = self.target_levels[BuildingType.FARM]
            for level in range(2, target + 1):
                queue.append((BuildingType.FARM, level))

        # 4. Core buildings
        core_buildings = [BuildingType.KEEP, BuildingType.LIBRARY]
        for btype in core_buildings:
            if btype in self.target_levels:
                target = self.target_levels[btype]
                for level in range(2, target + 1):
                    queue.append((btype, level))

        # 5. Military and other buildings
        other_buildings = [
            BuildingType.ARSENAL,
            BuildingType.TAVERN,
            BuildingType.MARKET,
            BuildingType.FORTIFICATIONS,
        ]
        for btype in other_buildings:
            if btype in self.target_levels:
                target = self.target_levels[btype]
                for level in range(2, target + 1):
                    queue.append((btype, level))

        return queue

    def _calculate_production_rates(
        self, building_levels: dict[BuildingType, int]
    ) -> dict[ResourceType, float]:
        """Calculate current production rates.

        Note: FOOD has NO production. Farm provides absolute capacity,
        upgrades consume it permanently. Food is NOT accumulated.
        """
        rates = {
            ResourceType.WOOD: 0.0,
            ResourceType.STONE: 0.0,
            ResourceType.IRON: 0.0,
            ResourceType.FOOD: 0.0,  # NO production - absolute capacity only
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
        """Calculate current storage capacities.

        Note: Farm provides FOOD (population) capacity, not production.
        """
        caps: dict[ResourceType, int] = {
            ResourceType.WOOD: 999999,
            ResourceType.STONE: 999999,
            ResourceType.IRON: 999999,
            ResourceType.FOOD: 40,  # Default food capacity (Farm L1)
        }

        storage_buildings = {
            BuildingType.WOOD_STORE: ResourceType.WOOD,
            BuildingType.STONE_STORE: ResourceType.STONE,
            BuildingType.ORE_STORE: ResourceType.IRON,
            BuildingType.FARM: ResourceType.FOOD,  # Farm provides food capacity
        }

        for btype, resource in storage_buildings.items():
            level = building_levels.get(btype, 1)
            building = self.buildings.get(btype)
            if not building:
                continue

            level_data = building.get_level_data(level)
            if level_data and level_data.storage_capacity is not None:
                caps[resource] = int(level_data.storage_capacity)

        return caps
