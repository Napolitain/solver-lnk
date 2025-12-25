"""Data models for Lords and Knights game entities."""

from dataclasses import dataclass, field
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
class Technology:
    """Represents a technology that can be researched in the library."""

    name: str
    internal_name: str
    required_library_level: int
    costs: dict[ResourceType, int]
    research_time_seconds: int
    enables_building: str | None = None
    enables_level: int | None = None


@dataclass
class Building:
    """Represents a building with all its levels and prerequisites."""

    building_type: BuildingType
    max_level: int
    levels: dict[int, BuildingLevel]
    prerequisites: dict[int, dict[BuildingType, int]]  # level -> {building: min_level}
    technology_prerequisites: dict[int, str] = field(
        default_factory=dict
    )  # level -> technology_name

    def get_level_data(self, level: int) -> BuildingLevel | None:
        """Get data for a specific level."""
        return self.levels.get(level)

    def can_upgrade_to(
        self,
        target_level: int,
        current_buildings: dict[BuildingType, int],
        researched_technologies: set[str] | None = None,
    ) -> bool:
        """Check if can upgrade to target level."""
        if target_level > self.max_level or target_level < 1:
            return False

        # Check building prerequisites
        prereqs = self.prerequisites.get(target_level, {})
        for req_building, req_level in prereqs.items():
            if current_buildings.get(req_building, 0) < req_level:
                return False

        # Check technology prerequisites
        if researched_technologies is not None:
            tech_req = self.technology_prerequisites.get(target_level)
            if tech_req and tech_req not in researched_technologies:
                return False

        return True


@dataclass
class GameState:
    """Current state of a castle."""

    building_levels: dict[BuildingType, int]
    resources: dict[ResourceType, float]
    researched_technologies: set[str] = field(default_factory=set)
    time_elapsed: float = 0.0  # in seconds
    building_queue_busy_until: float = 0.0  # timestamp when building queue is free
    research_queue_busy_until: float = 0.0  # timestamp when research queue is free

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


@dataclass
class BuildingUpgradeAction:
    """Represents a building upgrade action in the solution."""

    building_type: BuildingType
    from_level: int
    to_level: int
    start_time: float  # seconds from start
    end_time: float  # seconds from start
    costs: dict[ResourceType, int]

    def __str__(self) -> str:
        """Human-readable representation."""
        building_name = self.building_type.value.replace("_", " ").title()
        return f"{building_name} {self.from_level}→{self.to_level}"


@dataclass
class LibraryResearchAction:
    """Represents a library upgrade action that unlocks technologies."""

    from_level: int
    to_level: int
    start_time: float  # seconds from start
    end_time: float  # seconds from start
    costs: dict[ResourceType, int]
    technologies_unlocked: list[str]  # List of tech names unlocked at this level

    def __str__(self) -> str:
        """Human-readable representation."""
        techs = (
            ", ".join(self.technologies_unlocked)
            if self.technologies_unlocked
            else "None"
        )
        return f"Library {self.from_level}→{self.to_level} (Unlocks: {techs})"


@dataclass
class Solution:
    """Complete solution with both building and research actions."""

    building_actions: list[BuildingUpgradeAction]
    research_actions: list[LibraryResearchAction]
    total_time_seconds: float
    final_state: GameState

    def get_all_actions_chronological(
        self,
    ) -> list[BuildingUpgradeAction | LibraryResearchAction]:
        """Get all actions sorted by start time."""
        all_actions: list[BuildingUpgradeAction | LibraryResearchAction] = []
        all_actions.extend(self.building_actions)
        all_actions.extend(self.research_actions)
        return sorted(all_actions, key=lambda a: a.start_time)
