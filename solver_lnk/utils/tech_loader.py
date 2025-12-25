"""Technology data loader."""

import json
import re
from pathlib import Path

from solver_lnk.models.technology import Technology, TechnologyCost


def load_tech_library_requirements(
    json_path: Path | None = None,
) -> dict[str, int]:
    """Load library level requirements for each technology."""
    if json_path is None:
        json_path = (
            Path(__file__).parent.parent.parent
            / "data"
            / "tech_library_requirements.json"
        )

    if not json_path.exists():
        # Default: all techs require library level 1
        return {}

    with open(json_path) as f:
        data = json.load(f)

    return {tech_name: info["library_level"] for tech_name, info in data.items()}


def parse_tech_file(file_path: Path) -> Technology:
    """
    Parse a technology file.

    Expected format:
        Name
        Description
        Costs
        <wood>
        <stone>
        <iron>
        <duration HH:MM:SS>

        Enables
        <Building> Level <level>
    """
    with open(file_path) as f:
        content = f.read()

    lines = [line.strip() for line in content.split("\n") if line.strip()]

    name = lines[0]
    description = lines[1]

    # Find costs section
    costs_idx = next(i for i, line in enumerate(lines) if line == "Costs")
    wood = int(lines[costs_idx + 1])
    stone = int(lines[costs_idx + 2])
    iron = int(lines[costs_idx + 3])

    # Parse duration HH:MM:SS
    duration_str = lines[costs_idx + 4]
    parts = duration_str.split(":")
    duration_seconds = int(parts[0]) * 3600 + int(parts[1]) * 60 + int(parts[2])

    # Find enables section (optional)
    enables_building = None
    enables_level = None

    try:
        enables_idx = next(i for i, line in enumerate(lines) if line == "Enables")
        enables_line = lines[enables_idx + 1]
        # Parse "Farm Level 25" or similar
        match = re.match(r"(.+?)\s+Level\s+(\d+)", enables_line)
        if match:
            enables_building = match.group(1)
            enables_level = int(match.group(2))
    except StopIteration:
        pass

    internal_name = file_path.name

    cost = TechnologyCost(
        wood=wood,
        stone=stone,
        iron=iron,
        duration_seconds=duration_seconds,
    )

    return Technology(
        name=name,
        internal_name=internal_name,
        description=description,
        cost=cost,
        library_level_required=1,  # Will be set later
        enables_building=enables_building,
        enables_level=enables_level,
    )


def load_technologies(data_dir: Path | None = None) -> dict[str, Technology]:
    """Load all technologies from data/techs directory."""
    if data_dir is None:
        data_dir = Path(__file__).parent.parent.parent / "data" / "techs"

    # Load library requirements
    library_reqs = load_tech_library_requirements()

    technologies = {}

    for tech_file in data_dir.iterdir():
        if tech_file.is_file():
            tech = parse_tech_file(tech_file)

            # Set library level requirement from config
            if tech.internal_name in library_reqs:
                tech = Technology(
                    name=tech.name,
                    internal_name=tech.internal_name,
                    description=tech.description,
                    cost=tech.cost,
                    library_level_required=library_reqs[tech.internal_name],
                    enables_building=tech.enables_building,
                    enables_level=tech.enables_level,
                )

            technologies[tech.internal_name] = tech

    return technologies
