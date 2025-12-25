"""Data models for Lords and Knights game entities."""

from dataclasses import dataclass
from enum import Enum


class ResourceType(Enum):
    """Types of resources in the game."""

    WOOD = "wood"
    STONE = "stone"
    IRON = "iron"
    FOOD = "food"


class BuildingType(Enum):
    """Types of buildings in the game."""

    # Resource production
    LUMBERJACK = "lumberjack"
    QUARRY = "quarry"
    ORE_MINE = "ore_mine"
    MINE = "ore_mine"  # Alias
    FARM = "farm"

    # Storage
    WOOD_STORE = "wood_store"
    STONE_STORE = "stone_store"
    ORE_STORE = "ore_store"

    # Core buildings
    KEEP = "keep"
    CASTLE = "castle"
    ARSENAL = "arsenal"
    TAVERN = "tavern"
    LIBRARY = "library"
    MARKET = "market"

    # Military
    FORTIFICATIONS = "fortifications"


@dataclass
class BuildingLevel:
    """Represents a specific level of a building."""

    level: int
    costs: dict[ResourceType, int]
    build_time_seconds: int
    production_rate: float | None = None  # Per hour for resource buildings
    storage_capacity: int | None = None  # For storage buildings


@dataclass
class Building:
    """Represents a building with all its levels and prerequisites."""

    building_type: BuildingType
    max_level: int
    levels: dict[int, BuildingLevel]
    prerequisites: dict[int, dict[BuildingType, int]]  # level -> {building: min_level}

    def get_level_data(self, level: int) -> BuildingLevel | None:
        """Get data for a specific level."""
        return self.levels.get(level)

    def can_upgrade_to(
        self, target_level: int, current_buildings: dict[BuildingType, int]
    ) -> bool:
        """Check if can upgrade to target level given current building levels."""
        if target_level > self.max_level or target_level < 1:
            return False

        prereqs = self.prerequisites.get(target_level, {})
        for req_building, req_level in prereqs.items():
            if current_buildings.get(req_building, 0) < req_level:
                return False

        return True


@dataclass
class GameState:
    """Current state of a castle."""

    building_levels: dict[BuildingType, int]
    resources: dict[ResourceType, float]
    time_elapsed: float = 0.0  # in seconds

    def get_production_rates(
        self, buildings: dict[BuildingType, Building]
    ) -> dict[ResourceType, float]:
        """Calculate current resource production rates per hour."""
        rates: dict[ResourceType, float] = {rt: 0.0 for rt in ResourceType}

        resource_map = {
            BuildingType.LUMBERJACK: ResourceType.WOOD,
            BuildingType.QUARRY: ResourceType.STONE,
            BuildingType.ORE_MINE: ResourceType.IRON,
            BuildingType.MINE: ResourceType.IRON,
            BuildingType.FARM: ResourceType.FOOD,
        }

        for building_type, level in self.building_levels.items():
            if level == 0:
                continue

            building = buildings.get(building_type)
            if building:
                level_data = building.get_level_data(level)
                if level_data and level_data.production_rate:
                    resource_type = resource_map.get(building_type)
                    if resource_type:
                        rates[resource_type] += level_data.production_rate

        return rates

    def update_resources(
        self, time_seconds: float, production_rates: dict[ResourceType, float]
    ) -> None:
        """Update resources based on production over time."""
        time_hours = time_seconds / 3600.0
        for resource_type, rate in production_rates.items():
            self.resources[resource_type] += rate * time_hours
        self.time_elapsed += time_seconds

    def can_afford(self, costs: dict[ResourceType, int]) -> bool:
        """Check if current resources can afford the given costs."""
        return all(
            self.resources.get(resource_type, 0) >= cost
            for resource_type, cost in costs.items()
        )

    def spend_resources(self, costs: dict[ResourceType, int]) -> None:
        """Spend resources for a building upgrade."""
        for resource_type, cost in costs.items():
            self.resources[resource_type] -= cost
