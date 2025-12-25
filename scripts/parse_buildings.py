"""Parse raw building data files into structured JSON format."""

import contextlib
import json
import re
from pathlib import Path


def parse_time_string(time_str: str) -> int:
    """Parse time string HH:MM:SS to seconds."""
    parts = time_str.split(":")
    hours, minutes, seconds = map(int, parts)
    return hours * 3600 + minutes * 60 + seconds


def parse_production_or_storage(line: str) -> tuple[str, float] | None:
    """Parse production rate or storage capacity."""
    # Production: +5/h, +24/h
    match = re.search(r"\+(\d+)/h", line)
    if match:
        return ("production", float(match.group(1)))

    # Storage: 120/180 (capacity is second number)
    match = re.search(r"(\d+)/(\d+)", line)
    if match:
        return ("storage", float(match.group(2)))

    return None


def parse_building_file(filepath: Path) -> dict:
    """Parse a single building data file."""
    lines = filepath.read_text().strip().split("\n")
    building_name = filepath.stem
    levels = {}

    # First pass: parse levels with costs
    i = 0
    while i < len(lines):
        line = lines[i].strip()

        if line.startswith("Upgrade level "):
            level = int(line.split("Upgrade level ")[1])

            # Get production/storage
            production = None
            storage = None
            for j in range(i + 1, min(i + 10, len(lines))):
                result = parse_production_or_storage(lines[j])
                if result:
                    if result[0] == "production":
                        production = result[1]
                    elif result[0] == "storage":
                        storage = result[1]
                    break

            # Look for costs
            costs_idx = None
            for j in range(i + 1, min(i + 25, len(lines))):
                if lines[j].strip() == "Costs":
                    costs_idx = j
                    break

            if costs_idx:
                cost_values = []
                time_str = None
                for j in range(costs_idx + 1, min(costs_idx + 10, len(lines))):
                    val = lines[j].strip()
                    if ":" in val and len(val.split(":")) == 3:
                        time_str = val
                        break
                    with contextlib.suppress(ValueError):
                        cost_values.append(int(val))

                if cost_values and time_str:
                    levels[level] = {
                        "costs": {
                            "wood": cost_values[0],
                            "stone": cost_values[1],
                            "iron": cost_values[2],
                            "food": cost_values[3] if len(cost_values) > 3 else 0,
                        },
                        "build_time_seconds": parse_time_string(time_str),
                    }
                    if production:
                        levels[level]["production_rate"] = production
                    if storage:
                        levels[level]["storage_capacity"] = storage

        i += 1

    # Second pass: fill missing early levels
    if levels:
        first_level = min(levels.keys())
        template = levels[first_level]

        for i, line in enumerate(lines):
            if line.strip().startswith("Upgrade level "):
                lvl = int(line.strip().split("Upgrade level ")[1])
                if lvl not in levels:
                    production = None
                    storage = None
                    for j in range(i + 1, min(i + 10, len(lines))):
                        result = parse_production_or_storage(lines[j])
                        if result:
                            if result[0] == "production":
                                production = result[1]
                            elif result[0] == "storage":
                                storage = result[1]
                            break

                    levels[lvl] = {
                        "costs": template["costs"].copy(),
                        "build_time_seconds": template["build_time_seconds"],
                    }
                    if production:
                        levels[lvl]["production_rate"] = production
                    if storage:
                        levels[lvl]["storage_capacity"] = storage

    return {
        "building_type": building_name,
        "max_level": max(levels.keys()) if levels else 0,
        "levels": {str(k): v for k, v in sorted(levels.items())},
    }


def main() -> None:
    """Parse all building data files and create JSON."""
    data_dir = Path("data")
    output_file = Path("data/buildings.json")

    buildings = {}

    # All available buildings
    all_buildings = [
        # Resource production
        "lumberjack",
        "quarry",
        "ore_mine",
        "farm",
        # Storage
        "wood_store",
        "stone_store",
        "ore_store",
        # Core buildings
        "keep",
        "arsenal",
        "library",
        "tavern",
        "market",
        "fortifications",
    ]

    for building_name in all_buildings:
        filepath = data_dir / building_name
        if filepath.exists():
            print(f"Parsing {building_name}...")
            building_data = parse_building_file(filepath)
            buildings[building_name] = building_data
            level_count = len(building_data["levels"])
            print(f"  -> {level_count} levels (1-{building_data['max_level']})")
        else:
            print(f"⚠️  Skipping {building_name} (file not found)")

    output_file.write_text(json.dumps(buildings, indent=2))
    print(f"\n✓ Saved to {output_file}")
    print(f"  Parsed {len(buildings)} buildings")


if __name__ == "__main__":
    main()
