"""Parse library technologies from raw data file."""

import json
import re
from pathlib import Path


def parse_time_string(time_str: str) -> int:
    """Parse time string HH:MM:SS to seconds."""
    parts = time_str.split(":")
    hours, minutes, seconds = map(int, parts)
    return hours * 3600 + minutes * 60 + seconds


def parse_library_file(filepath: Path) -> dict:
    """Parse library data file to extract technologies and prerequisites."""
    lines = filepath.read_text().strip().split("\n")

    technologies = {}
    library_levels = {}

    i = 0
    while i < len(lines):
        line = lines[i].strip()

        if line.startswith("Upgrade level "):
            level = int(line.split("Upgrade level ")[1])

            # Extract technologies for this level
            techs = []
            j = i + 1

            # Look for technology pairs (name + "Enables: ...")
            while (
                j < len(lines)
                and not lines[j].strip().startswith("Upgrade level ")
                and not lines[j].strip().startswith("Costs")
            ):
                tech_line = lines[j].strip()

                # Skip empty lines
                if not tech_line:
                    j += 1
                    continue

                # Check if this is a technology name (next line has "Enables:")
                if j + 1 < len(lines) and lines[j + 1].strip().startswith("Enables:"):
                    tech_name = tech_line
                    enables = lines[j + 1].strip().replace("Enables: ", "")

                    techs.append({"name": tech_name, "enables": enables})

                    # Store in global tech registry
                    if tech_name not in technologies:
                        technologies[tech_name] = {
                            "name": tech_name,
                            "enables": enables,
                            "required_library_level": level,
                        }

                    j += 2  # Skip the "Enables:" line
                else:
                    j += 1

            # Look for costs
            costs = None
            build_time = None

            while j < len(lines) and not lines[j].strip().startswith("Upgrade level "):
                if lines[j].strip().startswith("Costs"):
                    # Next 4 lines are: wood, stone, iron, food
                    # Then time
                    try:
                        wood = int(lines[j + 1].strip())
                        stone = int(lines[j + 2].strip())
                        iron = int(lines[j + 3].strip())
                        food = int(lines[j + 4].strip())
                        time_str = lines[j + 5].strip()

                        costs = {
                            "wood": wood,
                            "stone": stone,
                            "iron": iron,
                            "food": food,
                        }
                        build_time = parse_time_string(time_str)
                    except (IndexError, ValueError):
                        pass

                    break
                j += 1

            # Store library level data
            library_levels[str(level)] = {
                "costs": costs,
                "build_time_seconds": build_time,
                "technologies": techs,
            }

            i = j
        else:
            i += 1

    return {"technologies": technologies, "library_levels": library_levels}


def extract_building_prerequisites(technologies: dict) -> dict:
    """Extract building level prerequisites from technologies."""
    prereqs = {}

    for tech_name, tech_data in technologies.items():
        enables = tech_data["enables"]

        # Parse "Farm Level 15" format
        match = re.match(r"(\w+(?:\s+\w+)*)\s+Level\s+(\d+)", enables)
        if match:
            building_name = match.group(1).lower().replace(" ", "_")
            level = int(match.group(2))
            required_lib_level = tech_data["required_library_level"]

            if building_name not in prereqs:
                prereqs[building_name] = {}

            prereqs[building_name][level] = {
                "library": required_lib_level,
                "technology": tech_name,
            }

    return prereqs


def main():
    """Parse library data and save to JSON."""
    data_dir = Path(__file__).parent.parent / "data"
    library_file = data_dir / "library"

    if not library_file.exists():
        print(f"❌ Library file not found: {library_file}")
        return

    print("📚 Parsing library data...")
    result = parse_library_file(library_file)

    # Save technologies
    tech_output = data_dir / "technologies.json"
    with tech_output.open("w") as f:
        json.dump(result["technologies"], f, indent=2)
    print(f"✅ Saved {len(result['technologies'])} technologies to {tech_output}")

    # Save library levels
    lib_output = data_dir / "library_levels.json"
    with lib_output.open("w") as f:
        json.dump(result["library_levels"], f, indent=2)
    print(f"✅ Saved {len(result['library_levels'])} library levels to {lib_output}")

    # Extract and save building prerequisites
    prereqs = extract_building_prerequisites(result["technologies"])
    prereq_output = data_dir / "technology_prerequisites.json"
    with prereq_output.open("w") as f:
        json.dump(prereqs, f, indent=2)
    print(f"✅ Saved technology prerequisites to {prereq_output}")

    # Show farm prerequisites
    if "farm" in prereqs:
        print("\n🌾 Farm technology prerequisites:")
        for level, req in sorted(prereqs["farm"].items()):
            print(
                f"  Level {level}: Requires Library {req['library']} "
                f"({req['technology']})"
            )


if __name__ == "__main__":
    main()
