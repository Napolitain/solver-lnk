package v3_test

import (
	"sort"
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	v3 "github.com/napolitain/solver-lnk/internal/solver/v3"
)

const dataDir = "../../../data"

func TestSolverBasic(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Quarry:     10,
		models.OreMine:    10,
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	solver := v3.NewSolver(buildings, technologies, targetLevels)
	solution := solver.Solve(initialState)

	if solution == nil {
		t.Fatal("Solution should not be nil")
	}

	if len(solution.BuildingActions) == 0 {
		t.Error("Should have building actions")
	}

	// Verify targets reached
	for bt, target := range targetLevels {
		if solution.FinalState.BuildingLevels[bt] < target {
			t.Errorf("%s should reach level %d, got %d", bt, target, solution.FinalState.BuildingLevels[bt])
		}
	}

	t.Logf("Completed in %.2f days with %d building actions",
		float64(solution.TotalTimeSeconds)/86400.0, len(solution.BuildingActions))
}

func TestSolverWithStrategy(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 15,
		models.Quarry:     15,
		models.OreMine:    15,
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	solver := v3.NewSolver(buildings, technologies, targetLevels)
	solution := solver.Solve(initialState)

	if solution == nil {
		t.Fatal("Solution should not be nil")
	}

	// With ROI-based selection, highest ROI building should be first
	if len(solution.BuildingActions) > 0 {
		first := solution.BuildingActions[0]
		t.Logf("First action: %s %d->%d (ROI-based)", first.BuildingType, first.FromLevel, first.ToLevel)
	}

	t.Logf("ROI-based completed in %.2f days", float64(solution.TotalTimeSeconds)/86400.0)
}

func TestSolveAllStrategies(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 20,
		models.Quarry:     20,
		models.OreMine:    20,
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	solution, strategyName, results := v3.SolveAllStrategies(buildings, technologies, initialState, targetLevels)

	if solution == nil {
		t.Fatal("Solution should not be nil")
	}

	t.Logf("Best strategy: %s", strategyName)
	t.Logf("Completion time: %.2f days", float64(solution.TotalTimeSeconds)/86400.0)
	t.Logf("Strategies tried: %d", len(results))
}

func TestFullBuildComparison(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	// Full castle targets
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack:     30,
		models.Quarry:         30,
		models.OreMine:        30,
		models.Farm:           30,
		models.WoodStore:      20,
		models.StoneStore:     20,
		models.OreStore:       20,
		models.Keep:           10,
		models.Arsenal:        30,
		models.Library:        10,
		models.Tavern:         10,
		models.Market:         8,
		models.Fortifications: 20,
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	solution, strategyName, _ := v3.SolveAllStrategies(buildings, technologies, initialState, targetLevels)

	if solution == nil {
		t.Fatal("Solution should not be nil")
	}

	days := float64(solution.TotalTimeSeconds) / 86400.0
	t.Logf("Best strategy: %s", strategyName)
	t.Logf("Completion time: %.2f days (%.0f hours)", days, days*24)
	t.Logf("Building actions: %d", len(solution.BuildingActions))
	t.Logf("Research actions: %d", len(solution.ResearchActions))

	// Verify ALL target buildings reached their target levels
	for bt, target := range targetLevels {
		final := solution.FinalState.BuildingLevels[bt]
		if final < target {
			t.Errorf("%s: target=%d, final=%d - NOT REACHED", bt, target, final)
		} else {
			t.Logf("%s: target=%d, final=%d ✓", bt, target, final)
		}
	}

	// Verify ALL loaded technologies are researched
	for techName := range technologies {
		if !solution.FinalState.ResearchedTechnologies[techName] {
			t.Errorf("Technology %s should be researched", techName)
		} else {
			t.Logf("Technology %s ✓", techName)
		}
	}
	t.Logf("Total technologies researched: %d/%d", len(solution.FinalState.ResearchedTechnologies), len(technologies))

	// Should complete in roughly 40-70 days based on V2 performance
	if days < 40 || days > 75 {
		t.Errorf("Completion time %.2f days is outside expected range [40, 75]", days)
	}
}

func TestDeterminism(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 15,
		models.Quarry:     15,
		models.OreMine:    15,
	}

	createInitialState := func() *models.GameState {
		s := models.NewGameState()
		s.Resources[models.Wood] = 120
		s.Resources[models.Stone] = 120
		s.Resources[models.Iron] = 120
		s.Resources[models.Food] = 40
		for _, bt := range models.AllBuildingTypes() {
			s.BuildingLevels[bt] = 1
		}
		return s
	}

	solver := v3.NewSolver(buildings, technologies, targetLevels)

	// Run multiple times
	var firstTime int
	var firstActionCount int

	for i := 0; i < 10; i++ {
		solution := solver.Solve(createInitialState())
		
		if i == 0 {
			firstTime = solution.TotalTimeSeconds
			firstActionCount = len(solution.BuildingActions)
		} else {
			if solution.TotalTimeSeconds != firstTime {
				t.Errorf("Run %d: time %d != first run %d", i, solution.TotalTimeSeconds, firstTime)
			}
			if len(solution.BuildingActions) != firstActionCount {
				t.Errorf("Run %d: action count %d != first run %d", i, len(solution.BuildingActions), firstActionCount)
			}
		}
	}

	t.Logf("Determinism verified across 10 runs (time=%d, actions=%d)", firstTime, firstActionCount)
}

func TestInvariants(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 20,
		models.Quarry:     20,
		models.OreMine:    20,
		models.Farm:       15,
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	solver := v3.NewSolver(buildings, technologies, targetLevels)
	solution := solver.Solve(initialState)

	// Invariant 1: Times are non-negative and ordered
	for i, action := range solution.BuildingActions {
		if action.StartTime < 0 {
			t.Errorf("Action %d has negative start time: %d", i, action.StartTime)
		}
		if action.EndTime < action.StartTime {
			t.Errorf("Action %d end time %d < start time %d", i, action.EndTime, action.StartTime)
		}
	}

	// Invariant 2: Levels increase by 1
	for i, action := range solution.BuildingActions {
		if action.ToLevel != action.FromLevel+1 {
			t.Errorf("Action %d: level change %d->%d (should be +1)", i, action.FromLevel, action.ToLevel)
		}
	}

	// Invariant 3: Food used <= food capacity
	for i, action := range solution.BuildingActions {
		if action.FoodUsed > action.FoodCapacity {
			t.Errorf("Action %d: food used %d > capacity %d", i, action.FoodUsed, action.FoodCapacity)
		}
	}

	// Invariant 4: Building queue is serial (no overlapping building actions)
	for i := 1; i < len(solution.BuildingActions); i++ {
		prev := solution.BuildingActions[i-1]
		curr := solution.BuildingActions[i]
		if curr.StartTime < prev.EndTime {
			t.Errorf("Building actions overlap: action %d ends at %d, action %d starts at %d",
				i-1, prev.EndTime, i, curr.StartTime)
		}
	}
}

// TestGameRulesValidation performs a full simulation replay to validate all game rules
func TestGameRulesValidation(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	// Full castle build
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack:     30,
		models.Quarry:         30,
		models.OreMine:        30,
		models.Farm:           30,
		models.WoodStore:      20,
		models.StoneStore:     20,
		models.OreStore:       20,
		models.Keep:           10,
		models.Arsenal:        30,
		models.Library:        10,
		models.Tavern:         10,
		models.Market:         8,
		models.Fortifications: 20,
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	solver := v3.NewSolver(buildings, technologies, targetLevels)
	solution := solver.Solve(initialState)

	// === REPLAY SIMULATION ===
	// Process all events in chronological order

	type event struct {
		time       int
		isStart    bool // true = start, false = complete
		isBuilding bool // true = building, false = research
		buildIdx   int
		resIdx     int
	}

	var events []event

	// Add building events
	for i, action := range solution.BuildingActions {
		events = append(events, event{time: action.StartTime, isStart: true, isBuilding: true, buildIdx: i})
		events = append(events, event{time: action.EndTime, isStart: false, isBuilding: true, buildIdx: i})
	}

	// Add research events
	for i, action := range solution.ResearchActions {
		events = append(events, event{time: action.StartTime, isStart: true, isBuilding: false, resIdx: i})
		events = append(events, event{time: action.EndTime, isStart: false, isBuilding: false, resIdx: i})
	}

	// Sort by time, completions before starts at same time
	sort.Slice(events, func(i, j int) bool {
		if events[i].time != events[j].time {
			return events[i].time < events[j].time
		}
		// At same time: completions before starts
		if events[i].isStart != events[j].isStart {
			return !events[i].isStart // completion first
		}
		return false
	})

	// Simulation state
	simTime := 0
	simResources := map[models.ResourceType]float64{
		models.Wood:  120,
		models.Stone: 120,
		models.Iron:  120,
		models.Food:  40,
	}
	simBuildingLevels := make(map[models.BuildingType]int)
	for _, bt := range models.AllBuildingTypes() {
		simBuildingLevels[bt] = 1
	}
	simResearchedTechs := make(map[string]bool)
	simFoodUsed := 0
	simBuildingQueueFreeAt := 0
	simResearchQueueFreeAt := 0

	// Helper functions
	getProductionRate := func(bt models.BuildingType, level int) float64 {
		building := buildings[bt]
		if building == nil {
			return 0
		}
		levelData := building.GetLevelData(level)
		if levelData == nil || levelData.ProductionRate == nil {
			return 0
		}
		return *levelData.ProductionRate
	}

	getStorageCap := func(bt models.BuildingType, level int) int {
		building := buildings[bt]
		if building == nil {
			return 0
		}
		levelData := building.GetLevelData(level)
		if levelData == nil || levelData.StorageCapacity == nil {
			return 0
		}
		return *levelData.StorageCapacity
	}

	simProductionRates := map[models.ResourceType]float64{
		models.Wood:  getProductionRate(models.Lumberjack, 1),
		models.Stone: getProductionRate(models.Quarry, 1),
		models.Iron:  getProductionRate(models.OreMine, 1),
	}

	simStorageCaps := map[models.ResourceType]int{
		models.Wood:  getStorageCap(models.WoodStore, 1),
		models.Stone: getStorageCap(models.StoneStore, 1),
		models.Iron:  getStorageCap(models.OreStore, 1),
	}

	simFoodCapacity := getStorageCap(models.Farm, 1)
	simProductionBonus := 1.0

	advanceSimTime := func(toTime int) {
		if toTime <= simTime {
			return
		}
		deltaSeconds := toTime - simTime
		deltaHours := float64(deltaSeconds) / 3600.0

		for _, rt := range []models.ResourceType{models.Wood, models.Stone, models.Iron} {
			rate := simProductionRates[rt]
			produced := rate * deltaHours * simProductionBonus
			simResources[rt] += produced

			cap := simStorageCaps[rt]
			if cap > 0 && simResources[rt] > float64(cap) {
				simResources[rt] = float64(cap)
			}
		}
		simTime = toTime
	}

	// Process events
	for _, ev := range events {
		advanceSimTime(ev.time)

		if ev.isBuilding {
			action := solution.BuildingActions[ev.buildIdx]
			building := buildings[action.BuildingType]

			if ev.isStart {
				// === BUILDING START ===
				levelData := building.GetLevelData(action.ToLevel)

				// Rule 1: Building queue must be free
				if ev.time < simBuildingQueueFreeAt {
					t.Errorf("Building %d (%s %d->%d): starts at %d but queue busy until %d",
						ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
						ev.time, simBuildingQueueFreeAt)
				}

				// Rule 2: FromLevel must match current level
				if action.FromLevel != simBuildingLevels[action.BuildingType] {
					t.Errorf("Building %d (%s): FromLevel=%d but current level is %d",
						ev.buildIdx, action.BuildingType, action.FromLevel, simBuildingLevels[action.BuildingType])
				}

				// Rule 3: Must have enough resources
				costs := levelData.Costs
				for rt, cost := range costs {
					if rt == models.Food {
						continue
					}
					if cost > 0 && simResources[rt] < float64(cost)-0.01 {
						t.Errorf("Building %d (%s %d->%d): needs %d %s but only have %.2f",
							ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
							cost, rt, simResources[rt])
					}
				}

				// Rule 4: Storage capacity check
				for rt, cost := range costs {
					if rt == models.Food {
						continue
					}
					cap := simStorageCaps[rt]
					if cost > cap {
						t.Errorf("Building %d (%s %d->%d): cost %d %s exceeds storage cap %d",
							ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
							cost, rt, cap)
					}
				}

				// Rule 5: Food capacity
				foodCost := costs[models.Food]
				if simFoodUsed+foodCost > simFoodCapacity {
					t.Errorf("Building %d (%s %d->%d): needs %d food workers, but %d/%d already used",
						ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
						foodCost, simFoodUsed, simFoodCapacity)
				}

				// Rule 6: Technology prerequisite must be researched BEFORE building starts
				if techName, ok := building.TechnologyPrerequisites[action.ToLevel]; ok {
					if !simResearchedTechs[techName] {
						t.Errorf("Building %d (%s %d->%d): requires tech '%s' which is not researched at start time %d",
							ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel, techName, ev.time)
					}
				}

				// Deduct resources at start
				for rt, cost := range costs {
					if rt == models.Food {
						continue
					}
					if cost > 0 {
						simResources[rt] -= float64(cost)
					}
				}
				simFoodUsed += foodCost

			} else {
				// === BUILDING COMPLETE ===
				simBuildingLevels[action.BuildingType] = action.ToLevel
				simBuildingQueueFreeAt = ev.time

				// Update production rates
				switch action.BuildingType {
				case models.Lumberjack:
					simProductionRates[models.Wood] = getProductionRate(models.Lumberjack, action.ToLevel)
				case models.Quarry:
					simProductionRates[models.Stone] = getProductionRate(models.Quarry, action.ToLevel)
				case models.OreMine:
					simProductionRates[models.Iron] = getProductionRate(models.OreMine, action.ToLevel)
				}

				// Update storage caps
				switch action.BuildingType {
				case models.WoodStore:
					simStorageCaps[models.Wood] = getStorageCap(models.WoodStore, action.ToLevel)
				case models.StoneStore:
					simStorageCaps[models.Stone] = getStorageCap(models.StoneStore, action.ToLevel)
				case models.OreStore:
					simStorageCaps[models.Iron] = getStorageCap(models.OreStore, action.ToLevel)
				case models.Farm:
					simFoodCapacity = getStorageCap(models.Farm, action.ToLevel)
				}
			}
		} else {
			// Research event
			action := solution.ResearchActions[ev.resIdx]
			tech := technologies[action.TechnologyName]

			if ev.isStart {
				// === RESEARCH START ===

				// Rule: Research queue must be free
				if ev.time < simResearchQueueFreeAt {
					t.Errorf("Research %d (%s): starts at %d but queue busy until %d",
						ev.resIdx, action.TechnologyName, ev.time, simResearchQueueFreeAt)
				}

				// Rule: Library level
				if tech != nil {
					libraryLevel := simBuildingLevels[models.Library]
					if libraryLevel < tech.RequiredLibraryLevel {
						t.Errorf("Research %d (%s): requires Library %d but have %d",
							ev.resIdx, action.TechnologyName, tech.RequiredLibraryLevel, libraryLevel)
					}
				}

				// Rule: Not already researched
				if simResearchedTechs[action.TechnologyName] {
					t.Errorf("Research %d (%s): already researched", ev.resIdx, action.TechnologyName)
				}

				// Rule: Must have enough resources
				if tech != nil {
					for rt, cost := range tech.Costs {
						if rt == models.Food {
							continue
						}
						if cost > 0 && simResources[rt] < float64(cost)-0.01 {
							t.Errorf("Research %d (%s): needs %d %s but only have %.2f",
								ev.resIdx, action.TechnologyName, cost, rt, simResources[rt])
						}
					}

					// Deduct resources
					for rt, cost := range tech.Costs {
						if rt == models.Food {
							continue
						}
						if cost > 0 {
							simResources[rt] -= float64(cost)
						}
					}
				}

			} else {
				// === RESEARCH COMPLETE ===
				simResearchedTechs[action.TechnologyName] = true
				simResearchQueueFreeAt = ev.time

				// Apply production bonus
				if action.TechnologyName == "Beer tester" || action.TechnologyName == "Wheelbarrow" {
					simProductionBonus += 0.05
				}
			}
		}
	}

	// Final validation
	for rt, amount := range simResources {
		if amount < -0.01 {
			t.Errorf("Final resources for %s is negative: %.2f", rt, amount)
		}
	}

	t.Logf("Game rules validation completed!")
	t.Logf("Final state: %d buildings upgraded, %d techs researched",
		len(solution.BuildingActions), len(solution.ResearchActions))
}
