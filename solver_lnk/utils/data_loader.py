"""Data loaders for building information."""

import json
import re
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


def load_technology_prerequisites(
    json_path: Path | None = None,
) -> dict[str, dict[int, str]]:
    """
    Load technology prerequisites from JSON file.

    Returns:
        Dict mapping building_name -> {level: technology_name}
    """
    if json_path is None:
        json_path = (
            Path(__file__).parent.parent.parent
            / "data"
            / "technology_prerequisites.json"
        )

    if not json_path.exists():
        return {}

    with open(json_path) as f:
        data = json.load(f)

    # Convert to {building_name: {level: technology_name}}
    result = {}
    for building_name, prereqs in data.items():
        result[building_name] = {
            int(level): tech_data["technology"] for level, tech_data in prereqs.items()
        }

    return result


def load_buildings_from_json(
    json_path: Path | None = None,
) -> dict[BuildingType, Building]:
    """Load building data from JSON file."""
    if json_path is None:
        # Default to data/buildings.json
        json_path = Path(__file__).parent.parent.parent / "data" / "buildings.json"

    with open(json_path) as f:
        data = json.load(f)

    # Load technology prerequisites
    tech_prereqs = load_technology_prerequisites()

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

        # Load building prerequisites (TODO: from JSON when available)
        prerequisites: dict[int, dict[BuildingType, int]] = {}

        # Load technology prerequisites for this building
        technology_prerequisites = tech_prereqs.get(building_name, {})

        buildings[building_type] = Building(
            building_type=building_type,
            max_level=building_data["max_level"],
            levels=levels,
            prerequisites=prerequisites,
            technology_prerequisites=technology_prerequisites,
        )

    return buildings


def parse_tech_file(tech_path: Path) -> dict[str, str | int | None]:
    """
    Parse a technology text file.

    Expected format:
        Line 1: Technology name (or empty)
        Line 2: Technology name or Description
        Line 3: Description or "Costs"
        Line 4+: Cost values and duration

    Returns:
        Dict with tech data: name, description, costs, duration, enables
    """
    with open(tech_path) as f:
        lines = [line.strip() for line in f.readlines() if line.strip()]

    if len(lines) < 7:
        raise ValueError(f"Invalid tech file format: {tech_path}")

    # Parse flexible format - name could be line 0 or 1
    if lines[0].lower() == "costs" or not lines[0]:
        name = lines[1] if len(lines) > 1 else tech_path.stem
        desc_start = 2
    else:
        name = lines[0]
        desc_start = 1

    # Find "Costs" line
    costs_idx = None
    for i, line in enumerate(lines):
        if line.lower() == "costs":
            costs_idx = i
            break

    if costs_idx is None:
        raise ValueError(f"'Costs' line not found in {tech_path}")

    # Description is everything before Costs
    description = " ".join(lines[desc_start:costs_idx])

    # Costs are 3 numbers after "Costs" line
    wood_cost = int(lines[costs_idx + 1])
    stone_cost = int(lines[costs_idx + 2])
    iron_cost = int(lines[costs_idx + 3])
    duration = parse_time_string(lines[costs_idx + 4])

    # Parse enables information from description or enables lines
    enables_building = None
    enables_level = None

    # Check all text for "upgraded to level X" or "Farm can now be upgraded to level X"
    full_text = " ".join(lines)
    match = re.search(r"upgraded to level (\d+)", full_text, re.IGNORECASE)
    if match:
        enables_level = int(match.group(1))
        # Look for building name
        building_match = re.search(
            r"\b(farm|lumberjack|quarry|mine)\b", full_text, re.IGNORECASE
        )
        if building_match:
            enables_building = building_match.group(1).lower()

    # Check for explicit "Enables" sections (after costs)
    for i in range(costs_idx + 5, len(lines)):
        line = lines[i]
        if line.lower().startswith("enables"):
            continue  # Skip "Enables" header, next line has the info
        # Look for "Farm Level 25" pattern
        match = re.match(r"([A-Za-z\s]+)\s+Level\s+(\d+)", line, re.IGNORECASE)
        if match:
            enables_building = match.group(1).strip().lower()
            enables_level = int(match.group(2))
            break

    return {
        "name": name,
        "internal_name": tech_path.name,
        "description": description,
        "wood": wood_cost,
        "stone": stone_cost,
        "iron": iron_cost,
        "duration_seconds": duration,
        "enables_building": enables_building,
        "enables_level": enables_level,
    }


def load_technologies(
    techs_dir: Path | None = None,
) -> dict[str, dict[str, str | int | None]]:
    """
    Load all technologies from data/techs/ directory.

    Returns:
        Dict mapping internal_name -> tech_data
    """
    if techs_dir is None:
        techs_dir = Path(__file__).parent.parent.parent / "data" / "techs"

    if not techs_dir.exists():
        return {}

    technologies = {}

    for tech_file in techs_dir.iterdir():
        if tech_file.is_file():
            try:
                tech_data = parse_tech_file(tech_file)
                technologies[tech_data["internal_name"]] = tech_data
            except Exception as e:
                print(f"Warning: Failed to parse tech file {tech_file}: {e}")

    return technologies


def get_default_buildings() -> dict[BuildingType, Building]:
    """Get default building data from JSON file."""
    return load_buildings_from_json()
