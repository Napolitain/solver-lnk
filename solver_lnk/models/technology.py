"""Technology data models."""

from dataclasses import dataclass


@dataclass(frozen=True)
class TechnologyCost:
    """Cost to research a technology."""

    wood: int
    stone: int
    iron: int
    duration_seconds: int


@dataclass(frozen=True)
class Technology:
    """Represents a technology that can be researched."""

    name: str
    internal_name: str  # e.g., "crop_rotation"
    description: str
    cost: TechnologyCost
    library_level_required: int = 1
    enables_building: str | None = None  # e.g., "Farm"
    enables_level: int | None = None  # e.g., 15
