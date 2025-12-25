"""Data loaders for building information."""

import json
from pathlib import Path

from solver_lnk.models import Building, BuildingLevel, BuildingType, ResourceType


def parse_time_string(time_str: str) -> int:
    """
    Parse time string in format HH:MM:SS to seconds.

    Examples:
        "00:06:23" -> 383 seconds
        "29:47:49" -> 107269 seconds
    """
    parts = time_str.split(":")
    if len(parts) != 3:
        raise ValueError(f"Invalid time format: {time_str}")

    hours, minutes, seconds = map(int, parts)
    return hours * 3600 + minutes * 60 + seconds


def load_buildings_from_json(
    json_path: Path | None = None,
) -> dict[BuildingType, Building]:
    """Load building data from JSON file."""
    if json_path is None:
        # Default to data/buildings.json
        json_path = Path(__file__).parent.parent.parent / "data" / "buildings.json"

    with open(json_path) as f:
        data = json.load(f)

    buildings: dict[BuildingType, Building] = {}

    for building_name, building_data in data.items():
        # Map building name to BuildingType enum
        try:
            building_type = BuildingType(building_name)
        except ValueError:
            print(f"Warning: Unknown building type '{building_name}', skipping")
            continue

        levels: dict[int, BuildingLevel] = {}
        for level_str, level_data in building_data["levels"].items():
            level = int(level_str)

            costs = {
                ResourceType.WOOD: level_data["costs"]["wood"],
                ResourceType.STONE: level_data["costs"]["stone"],
                ResourceType.IRON: level_data["costs"]["iron"],
                ResourceType.FOOD: level_data["costs"]["food"],
            }

            production_rate = level_data.get("production_rate")
            storage_capacity = level_data.get("storage_capacity")

            levels[level] = BuildingLevel(
                level=level,
                costs=costs,
                build_time_seconds=level_data["build_time_seconds"],
                production_rate=production_rate,
                storage_capacity=storage_capacity,
            )

        # TODO: Load prerequisites from JSON when available
        prerequisites: dict[int, dict[BuildingType, int]] = {}

        buildings[building_type] = Building(
            building_type=building_type,
            max_level=building_data["max_level"],
            levels=levels,
            prerequisites=prerequisites,
        )

    return buildings


def get_default_buildings() -> dict[BuildingType, Building]:
    """Get default building data from JSON file."""
    return load_buildings_from_json()
