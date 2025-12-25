"""Lords and Knights build order solver CLI."""

import argparse
import json
from pathlib import Path

from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from solver_lnk.models import (
    BuildingType,
    BuildingUpgradeAction,
    GameState,
    LibraryResearchAction,
    ResourceType,
    Solution,
)
from solver_lnk.solvers.cpsat_dual_queue_solver import CPSATDualQueueSolver
from solver_lnk.solvers.cpsat_resource_solver import CPSATResourceSolver
from solver_lnk.solvers.cpsat_solver import CPSATBuildOrderSolver
from solver_lnk.solvers.greedy_solver import GreedyBuildOrderSolver
from solver_lnk.utils.data_loader import get_default_buildings

console = Console()


def format_time(seconds: float) -> str:
    """Format seconds to HH:MM:SS."""
    hours = int(seconds // 3600)
    minutes = int((seconds % 3600) // 60)
    secs = int(seconds % 60)
    return f"{hours:02d}:{minutes:02d}:{secs:02d}"


def create_dual_queue_table(solution: Solution) -> Table:
    """Create a rich table showing both building and research queues."""
    table = Table(
        title="Build Order (Dual Queue)", show_header=True, header_style="bold magenta"
    )

    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Queue", style="magenta", width=10)
    table.add_column("Action", style="cyan", width=25)
    table.add_column("Upgrade", style="green", width=8, justify="center")
    table.add_column("Start", style="yellow", width=10, justify="right")
    table.add_column("End", style="yellow", width=10, justify="right")
    table.add_column("Duration", style="blue", width=10, justify="right")
    table.add_column("Costs", style="white", width=35)

    all_actions = solution.get_all_actions_chronological()

    for i, action in enumerate(all_actions, 1):
        if isinstance(action, BuildingUpgradeAction):
            queue = "🏗️ Building"
            name = action.building_type.value.replace("_", " ").title()
            upgrade = f"{action.from_level} → {action.to_level}"
            costs = (
                f"W:{action.costs[ResourceType.WOOD]:>5} "
                f"S:{action.costs[ResourceType.STONE]:>5} "
                f"I:{action.costs[ResourceType.IRON]:>4} "
                f"F:{action.costs[ResourceType.FOOD]:>3}"
            )
        elif isinstance(action, LibraryResearchAction):
            queue = "📚 Research"

            # Check if it's a library upgrade or tech research
            if action.from_level > 0:
                # Library upgrade
                name = "Library"
                upgrade = f"{action.from_level} → {action.to_level}"
            else:
                # Technology research
                name = (
                    f"Tech: {action.technologies_unlocked[0]}"
                    if action.technologies_unlocked
                    else "Tech"
                )
                upgrade = "Research"

            costs = (
                f"W:{action.costs[ResourceType.WOOD]:>5} "
                f"S:{action.costs[ResourceType.STONE]:>5} "
                f"I:{action.costs[ResourceType.IRON]:>4} "
                f"F:{action.costs[ResourceType.FOOD]:>3}"
            )
        else:
            continue

        start = format_time(action.start_time)
        end = format_time(action.end_time)
        duration = format_time(action.end_time - action.start_time)

        table.add_row(str(i), queue, name, upgrade, start, end, duration, costs)

    return table


def create_build_order_table(solution: list) -> Table:
    """Create a rich table for build order display."""
    table = Table(title="Build Order", show_header=True, header_style="bold magenta")

    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Building", style="cyan", width=15)
    table.add_column("Upgrade", style="green", width=8, justify="center")
    table.add_column("Start", style="yellow", width=10, justify="right")
    table.add_column("End", style="yellow", width=10, justify="right")
    table.add_column("Duration", style="blue", width=10, justify="right")
    table.add_column("Costs", style="white", width=35)

    for i, action in enumerate(solution, 1):
        building_name = action.building_type.value.replace("_", " ").title()
        upgrade = f"{action.from_level} → {action.to_level}"
        start = format_time(action.start_time)
        end = format_time(action.end_time)
        duration = format_time(action.end_time - action.start_time)

        costs = (
            f"W:{action.costs[ResourceType.WOOD]:>5} "
            f"S:{action.costs[ResourceType.STONE]:>5} "
            f"I:{action.costs[ResourceType.IRON]:>4} "
            f"F:{action.costs[ResourceType.FOOD]:>3}"
        )

        table.add_row(str(i), building_name, upgrade, start, end, duration, costs)

    return table


def parse_args() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Lords and Knights Build Order Optimizer",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                            # Solve default problem (castle level-up)
  %(prog)s --problem castle-levelup   # Explicit problem selection
  %(prog)s --config my_castle.json    # Load custom configuration
  %(prog)s --quiet                    # Minimal output
  %(prog)s --export build_plan.json   # Export solution to JSON
        """,
    )

    parser.add_argument(
        "--problem",
        type=str,
        default="castle-levelup",
        choices=["castle-levelup"],
        help="Problem to solve (default: castle-levelup)",
    )

    parser.add_argument(
        "--solver",
        type=str,
        default="cpsat-dual-queue",
        choices=["greedy", "cpsat", "cpsat-resource", "cpsat-dual-queue"],
        help="Solver to use (default: cpsat-dual-queue)",
    )

    parser.add_argument(
        "--config",
        type=Path,
        help="Path to problem configuration JSON file",
    )

    parser.add_argument(
        "--quiet",
        "-q",
        action="store_true",
        help="Minimal output (only completion time)",
    )

    parser.add_argument("--export", type=Path, help="Export solution to JSON file")

    return parser.parse_args()


def load_castle_levelup_problem(config_path: Path | None = None) -> tuple:
    """
    Load the castle level-up problem configuration.

    Returns tuple of (buildings, initial_state, target_levels).
    """
    if config_path:
        with open(config_path) as f:
            config = json.load(f)

        initial_state = GameState(
            building_levels={
                BuildingType(k): v
                for k, v in config.get("initial_buildings", {}).items()
            },
            resources={
                ResourceType(k): v
                for k, v in config.get("initial_resources", {}).items()
            },
        )
        target_levels = {
            BuildingType(k): v for k, v in config.get("target_levels", {}).items()
        }
    else:
        # Default configuration: all buildings start at level 1
        initial_state = GameState(
            building_levels={
                BuildingType.LUMBERJACK: 1,
                BuildingType.QUARRY: 1,
                BuildingType.ORE_MINE: 1,
                BuildingType.FARM: 1,
                BuildingType.WOOD_STORE: 1,
                BuildingType.STONE_STORE: 1,
                BuildingType.ORE_STORE: 1,
                BuildingType.KEEP: 1,
                BuildingType.ARSENAL: 1,
                BuildingType.LIBRARY: 1,
                BuildingType.TAVERN: 1,
                BuildingType.MARKET: 1,
                BuildingType.FORTIFICATIONS: 1,
            },
            resources={
                ResourceType.WOOD: 120.0,
                ResourceType.STONE: 120.0,
                ResourceType.IRON: 120.0,
                ResourceType.FOOD: 40.0,
            },
        )
        # Target: all buildings to max level
        target_levels = {
            # Production buildings
            BuildingType.LUMBERJACK: 30,
            BuildingType.QUARRY: 30,
            BuildingType.ORE_MINE: 30,
            BuildingType.FARM: 30,
            # Storage
            BuildingType.WOOD_STORE: 20,
            BuildingType.STONE_STORE: 20,
            BuildingType.ORE_STORE: 20,
            # Core/Military
            BuildingType.KEEP: 10,
            BuildingType.ARSENAL: 30,
            BuildingType.LIBRARY: 10,
            BuildingType.TAVERN: 10,
            BuildingType.MARKET: 8,
            BuildingType.FORTIFICATIONS: 20,
        }

    buildings = get_default_buildings()
    return buildings, initial_state, target_levels


def main() -> None:
    """Run build order optimization with CLI."""
    args = parse_args()

    if args.problem == "castle-levelup":
        buildings, initial_state, target_levels = load_castle_levelup_problem(
            args.config
        )
    else:
        console.print(f"[red]Unknown problem: {args.problem}[/red]")
        return

    if not args.quiet:
        console.print(
            Panel.fit(
                "[bold cyan]Lords and Knights[/bold cyan]\n"
                "[yellow]Build Order Optimizer[/yellow]",
                border_style="blue",
            )
        )

        console.print(f"\n[bold]Problem:[/bold] [magenta]{args.problem}[/magenta]")

        console.print("\n[bold]Initial State:[/bold]")
        for building_type, level in initial_state.building_levels.items():
            name = building_type.value.replace("_", " ").title()
            console.print(f"  • {name}: Level [cyan]{level}[/cyan]")

        console.print("\n[bold]Resources:[/bold]")
        resources = initial_state.resources
        console.print(
            f"  Wood: [yellow]{resources[ResourceType.WOOD]:.0f}[/yellow]  "
            f"Stone: [yellow]{resources[ResourceType.STONE]:.0f}[/yellow]  "
            f"Iron: [yellow]{resources[ResourceType.IRON]:.0f}[/yellow]  "
            f"Food: [yellow]{resources[ResourceType.FOOD]:.0f}[/yellow]"
        )

        console.print("\n[bold]Targets:[/bold]")
        for building_type, level in target_levels.items():
            name = building_type.value.replace("_", " ").title()
            console.print(f"  • {name}: Level [green]{level}[/green]")

        console.print("\n[bold magenta]Solving...[/bold magenta]")

    # Solve with selected solver
    if args.solver == "greedy":
        solver = GreedyBuildOrderSolver(
            buildings=buildings,
            initial_state=initial_state,
            target_levels=target_levels,
        )
        solution_list = solver.solve()
        solution_obj = None
    elif args.solver == "cpsat":
        solver = CPSATBuildOrderSolver(
            buildings=buildings,
            initial_state=initial_state,
            target_levels=target_levels,
        )
        solution_list = solver.solve()
        solution_obj = None
    elif args.solver == "cpsat-resource":
        solver = CPSATResourceSolver(
            buildings=buildings,
            initial_state=initial_state,
            target_levels=target_levels,
            time_interval=60,  # 1 minute intervals for better accuracy
        )
        solution_list = solver.solve()
        solution_obj = None
    else:  # cpsat-dual-queue (default)
        solver = CPSATDualQueueSolver(
            buildings=buildings,
            initial_state=initial_state,
            target_levels=target_levels,
            time_scale_minutes=10,
        )
        solution_obj = solver.solve()
        solution_list = None

    # Handle solution display
    if solution_obj:  # New dual-queue solution
        if not args.quiet:
            console.print(
                f"\n[bold green]✓ Found solution with "
                f"{len(solution_obj.building_actions)} building upgrades "
                f"and {len(solution_obj.research_actions)} research tasks!"
                f"[/bold green]\n"
            )

            table = create_dual_queue_table(solution_obj)
            console.print(table)

            console.print(
                f"\n[bold]Total completion time:[/bold] "
                f"[cyan]{format_time(solution_obj.total_time_seconds)}[/cyan]"
            )
        else:
            print(format_time(solution_obj.total_time_seconds))

    elif solution_list:  # Old-style solution list
        if not args.quiet:
            console.print(
                f"\n[bold green]✓ Found solution with {len(solution_list)} "
                f"upgrades![/bold green]\n"
            )

            table = create_build_order_table(solution_list)
            console.print(table)

            total_time = max(a.end_time for a in solution_list)
            console.print(
                f"\n[bold]Total completion time:[/bold] "
                f"[cyan]{format_time(total_time)}[/cyan]"
            )

            for building_type in target_levels:
                final_level = max(
                    (
                        a.to_level
                        for a in solution_list
                        if a.building_type == building_type
                    ),
                    default=0,
                )
                if final_level > 0:
                    building = buildings.get(building_type)
                    if building:
                        level_data = building.get_level_data(final_level)
                        if level_data and level_data.production_rate:
                            name = building_type.value.replace("_", " ").title()
                            console.print(
                                f"[bold]Final {name} production:[/bold] "
                                f"[green]{level_data.production_rate:.0f}/hour[/green]"
                            )
        else:
            total_time = max(a.end_time for a in solution_list)
            print(format_time(total_time))

        if args.export:
            export_data = {
                "problem": args.problem,
                "initial_state": {
                    "buildings": {
                        k.value: v for k, v in initial_state.building_levels.items()
                    },
                    "resources": {
                        k.value: v for k, v in initial_state.resources.items()
                    },
                },
                "targets": {k.value: v for k, v in target_levels.items()},
                "solution": [
                    {
                        "building": a.building_type.value,
                        "from_level": a.from_level,
                        "to_level": a.to_level,
                        "start_time": a.start_time,
                        "end_time": a.end_time,
                        "costs": {k.value: v for k, v in a.costs.items()},
                    }
                    for a in solution_list
                ],
                "total_time": max(a.end_time for a in solution_list),
            }

            args.export.write_text(json.dumps(export_data, indent=2))
            console.print(f"\n[green]✓ Exported to {args.export}[/green]")
    else:
        console.print("[red]✗ No solution found[/red]")


if __name__ == "__main__":
    main()
