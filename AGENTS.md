# Agent Rules and Guidelines

## General Behavior

### When in Doubt - Always Ask or Search

**Rule**: When you have low confidence about any decision, implementation detail, tool usage, or best practice:

1. **Ask the user first** for clarification
2. **Search the internet** if it's a factual/technical question that can be verified
3. **Never guess or assume** when uncertain

### Examples of Low Confidence Situations

- Unfamiliar tools or libraries (e.g., "ty" vs "mypy" vs "pyright")
- Game mechanics or domain-specific rules
- API specifics or library usage patterns
- Best practices for a specific technology
- Ambiguous requirements or user intent

### Process

```
IF confidence < 80% THEN
    1. Stop current action
    2. Ask user: "I'm not sure about X. Could you clarify Y?"
    OR
    3. Search web: "What is the best way to do X with Y?"
    4. Wait for confirmation before proceeding
END IF
```

## Project-Specific Rules

### Technology Stack

- **Package Manager**: uv (Astral)
- **Linter/Formatter**: ruff (Astral)
- **Type Checker**: ty (Astral) - NOT mypy or pyright
- **Python Version**: 3.13+
- **Pretty Printing**: rich (for CLI output)

### Development Workflow

**Always follow this sequence when making changes:**

1. **Make code changes**
2. **Format code**: `uv run ruff format .`
3. **Lint code**: `uv run ruff check .` (or `uv run ruff check --fix .` for auto-fixes)
4. **Type check**: `uv run ty check .`
5. **Test run**: `uv run python main.py` (with appropriate args)
6. **Verify output**: Check that the program works as expected

**Example workflow:**
```bash
# After editing code
uv run ruff format .
uv run ruff check --fix .
uv run ty check .
uv run python main.py --target-lumberjack 10

# All must pass before committing
```

### Running the Project

**Main entry point:**
```bash
# Default run (solves castle-levelup problem)
uv run python main.py

# Quiet mode (for scripts)
uv run python main.py --quiet

# With custom config file
uv run python main.py --config my_castle.json

# Export solution
uv run python main.py --export build_plan.json

# See all options
uv run python main.py --help
```

**Problem types:**
- `castle-levelup` (default): Optimize building upgrades for castle development

**Config file format** (JSON):
```json
{
  "initial_buildings": {"lumberjack": 0},
  "initial_resources": {"wood": 1000, "stone": 1000, "iron": 500, "food": 100},
  "target_levels": {"lumberjack": 10}
}
```

### Code Quality Standards

- Full type hints on all functions
- Pass `ruff check` with no errors
- Pass `ty check` with no errors
- Keep functions focused and small
- Document complex logic
- Use `rich` for all user-facing output (not plain `print`)

### Communication Style

- Be concise and direct
- Ask clarifying questions early
- Confirm understanding before implementing
- Provide alternatives when multiple approaches exist

## Update History

- 2025-12-25: Initial creation - Added "ask or search when low confidence" rule
- 2025-12-25: Added development workflow and running instructions
