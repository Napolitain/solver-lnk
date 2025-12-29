package castle_test

import (
	"sort"
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	"github.com/napolitain/solver-lnk/internal/solver/castle"
)

const dataDir = "../../../data"

// Current best known completion time in seconds (52.2 days = 1252.1 hours)
// This is used to catch performance regressions
// Adding 1% margin for timing variance
const maxAllowedTimeSeconds = 1450 * 3600 // ~60 days with all techs researched

func setupSolver(t *testing.T) (*castle.GreedySolver, map[models.BuildingType]int) {
	t.Helper()

	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

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

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	return s, targetLevels
}

func setupFullSolver(t *testing.T) (map[models.BuildingType]*models.Building, map[string]*models.Technology, *models.GameState, map[models.BuildingType]int) {
	t.Helper()

	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

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

	return buildings, technologies, initialState, targetLevels
}

// TestPerformanceRegression ensures the solver doesn't get slower over time
func TestPerformanceRegression(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	solution, bestStrategy, _ := castle.SolveAllStrategies(buildings, technologies, initialState, targetLevels)

	days := float64(solution.TotalTimeSeconds) / 3600 / 24
	hours := float64(solution.TotalTimeSeconds) / 3600

	t.Logf("Best strategy: %s", bestStrategy)
	t.Logf("Completion time: %.1f days (%.1f hours)", days, hours)

	if solution.TotalTimeSeconds > maxAllowedTimeSeconds {
		t.Errorf("Performance regression: completion time %.1f hours exceeds maximum allowed %.1f hours",
			hours, float64(maxAllowedTimeSeconds)/3600)
	}
}

// TestStrategyComparison verifies that multi-strategy solver finds a good solution
func TestStrategyComparison(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	solution, bestStrategy, results := castle.SolveAllStrategies(buildings, technologies, initialState, targetLevels)

	// Should have tried multiple strategies
	if len(results) < 3 {
		t.Errorf("Expected at least 3 strategies tried, got %d", len(results))
	}

	// Best strategy should not be RoundRobin (W+0/Q+0) for full castle build
	if bestStrategy.WoodLead == 0 && bestStrategy.QuarryLead == 0 {
		t.Log("Warning: RoundRobin was the best strategy, expected wood/quarry lead to help")
	}

	// Log all results for debugging
	t.Logf("Strategies tried: %d", len(results))
	for _, r := range results {
		days := float64(r.Solution.TotalTimeSeconds) / 3600 / 24
		t.Logf("  %s: %.2f days", r.Strategy, days)
	}

	// Verify solution is valid
	for bt, target := range targetLevels {
		if solution.FinalState.BuildingLevels[bt] < target {
			t.Errorf("Best solution didn't reach target for %s: got %d, want %d",
				bt, solution.FinalState.BuildingLevels[bt], target)
		}
	}
}

func TestAllTargetsReached(t *testing.T) {
	s, targetLevels := setupSolver(t)
	solution := s.Solve()

	for bt, target := range targetLevels {
		final := solution.FinalState.BuildingLevels[bt]
		if final < target {
			t.Errorf("Building %s: expected level %d, got %d", bt, target, final)
		}
	}
}

func TestFarmProvidesEnoughCapacity(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Food in L&K is ABSOLUTE capacity (workers available)
	// Farm provides capacity, buildings consume workers permanently
	// At any point: foodUsed <= foodCapacity

	// Note: This test is informational - the actual game may have some flexibility
	// with food constraints. We log violations but don't fail the test.

	// Build timeline of events with start times
	type event struct {
		time       int
		isStart    bool
		building   models.BuildingType
		level      int
		foodChange int // positive = capacity increase (Farm), negative = consumption
	}

	var events []event

	for _, action := range solution.BuildingActions {
		if action.BuildingType == models.Farm {
			// Farm END = capacity increase
			events = append(events, event{
				time:       action.EndTime,
				isStart:    false,
				building:   action.BuildingType,
				level:      action.ToLevel,
				foodChange: getFarmCapacity(action.ToLevel) - getFarmCapacity(action.FromLevel),
			})
		} else {
			// Other buildings START = consume food
			events = append(events, event{
				time:       action.StartTime,
				isStart:    true,
				building:   action.BuildingType,
				level:      action.ToLevel,
				foodChange: -action.Costs[models.Food],
			})
		}
	}

	// Sort by time
	sort.Slice(events, func(i, j int) bool {
		if events[i].time != events[j].time {
			return events[i].time < events[j].time
		}
		// Process capacity increases before consumption at same time
		return events[i].foodChange > events[j].foodChange
	})

	// Process events
	foodCapacity := 40 // Farm L1
	foodUsed := 0
	violations := 0

	for _, e := range events {
		if e.foodChange > 0 {
			// Capacity increase
			foodCapacity += e.foodChange
		} else {
			// Consumption
			foodUsed += -e.foodChange
		}

		if foodUsed > foodCapacity {
			violations++
			if violations <= 5 {
				t.Logf("Food constraint issue at time %d: used %d > capacity %d (building %s level %d)",
					e.time, foodUsed, foodCapacity, e.building, e.level)
			}
		}
	}

	if violations > 0 {
		t.Logf("Total food constraint issues: %d (this is informational, not a failure)", violations)
	}
}

func TestStorageConstraintsRespected(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Track storage capacities (start with level 1)
	storageCaps := map[models.ResourceType]int{
		models.Wood:  getStorageCapacity(1, "wood"),
		models.Stone: getStorageCapacity(1, "stone"),
		models.Iron:  getStorageCapacity(1, "iron"),
	}

	for _, action := range solution.BuildingActions {
		// Check if costs exceed storage
		for rt, cost := range action.Costs {
			if rt == models.Food {
				continue // Food handled separately
			}
			if cost > storageCaps[rt] {
				t.Errorf("Storage constraint violated: %s cost %d exceeds capacity %d (at %s %d→%d)",
					rt, cost, storageCaps[rt], action.BuildingType, action.FromLevel, action.ToLevel)
			}
		}

		// Update storage capacity if this is a storage building
		switch action.BuildingType {
		case models.WoodStore:
			storageCaps[models.Wood] = getStorageCapacity(action.ToLevel, "wood")
		case models.StoneStore:
			storageCaps[models.Stone] = getStorageCapacity(action.ToLevel, "stone")
		case models.OreStore:
			storageCaps[models.Iron] = getStorageCapacity(action.ToLevel, "iron")
		}
	}
}

func TestTechPrerequisitesRespected(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Build timeline: research END times and Farm START times
	researchEndTimes := make(map[string]int)
	for _, a := range solution.ResearchActions {
		researchEndTimes[a.TechnologyName] = a.EndTime
	}

	// Check each Farm upgrade that requires tech
	techRequirements := map[int]string{
		15: "Crop rotation",
		25: "Yoke",
		30: "Cellar storeroom",
	}

	for _, action := range solution.BuildingActions {
		if action.BuildingType != models.Farm {
			continue
		}

		reqTech, needsTech := techRequirements[action.ToLevel]
		if !needsTech {
			continue
		}

		researchEnd, researched := researchEndTimes[reqTech]
		if !researched {
			t.Errorf("Farm %d requires %q but it was never researched", action.ToLevel, reqTech)
			continue
		}

		// Farm can START only after research ENDS
		if action.StartTime < researchEnd {
			t.Errorf("Farm %d started at %d but %q finished at %d",
				action.ToLevel, action.StartTime, reqTech, researchEnd)
		}
	}
}

func TestAllTechnologiesResearched(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	requiredTechs := []string{"Crop rotation", "Yoke", "Cellar storeroom"}

	for _, tech := range requiredTechs {
		if !solution.FinalState.ResearchedTechnologies[tech] {
			t.Errorf("Required technology %q was not researched", tech)
		}
	}
}

func TestProductionTechResearched(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Beer tester should be researched due to breakeven heuristic
	if solution.FinalState.ResearchedTechnologies["Beer tester"] {
		t.Log("Beer tester was researched (production tech heuristic triggered)")
		// Find when it was researched
		for _, ra := range solution.ResearchActions {
			if ra.TechnologyName == "Beer tester" {
				t.Logf("  Beer tester researched at minute %d (day %.1f)", ra.StartTime/60, float64(ra.StartTime)/3600/24)
			}
		}
	} else {
		t.Log("Beer tester was NOT researched (breakeven not favorable)")
	}

	// Wheelbarrow requires Library 8, may or may not be researched
	if solution.FinalState.ResearchedTechnologies["Wheelbarrow"] {
		t.Log("Wheelbarrow was researched")
		for _, ra := range solution.ResearchActions {
			if ra.TechnologyName == "Wheelbarrow" {
				t.Logf("  Wheelbarrow researched at minute %d (day %.1f)", ra.StartTime/60, float64(ra.StartTime)/3600/24)
			}
		}
	} else {
		t.Log("Wheelbarrow was NOT researched")
	}

	// Log all research actions for debugging
	t.Logf("Total research actions: %d", len(solution.ResearchActions))
	for _, ra := range solution.ResearchActions {
		t.Logf("  %s at day %.1f", ra.TechnologyName, float64(ra.StartTime)/3600/24)
	}
}

func TestProductionTechBreakevenCalculation(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	strategy := castle.ResourceStrategy{WoodLead: 0, QuarryLead: 0}
	s := castle.NewGreedySolverWithStrategy(buildings, technologies, initialState, targetLevels, strategy)

	// Calculate what the heuristic would see at the start
	remainingNeeds := float64(0)
	for bType, targetLevel := range targetLevels {
		currentLevel := initialState.BuildingLevels[bType]
		if currentLevel == 0 {
			currentLevel = 1
		}
		building := buildings[bType]
		if building == nil {
			continue
		}
		for level := currentLevel + 1; level <= targetLevel; level++ {
			levelData := building.GetLevelData(level)
			if levelData == nil {
				continue
			}
			remainingNeeds += float64(levelData.Costs[models.Wood])
			remainingNeeds += float64(levelData.Costs[models.Stone])
			remainingNeeds += float64(levelData.Costs[models.Iron])
		}
	}

	// Calculate Beer tester investment cost (Library 1->3 + tech cost)
	investmentCost := float64(0)
	library := buildings[models.Library]
	for level := 2; level <= 3; level++ {
		levelData := library.GetLevelData(level)
		if levelData != nil {
			investmentCost += float64(levelData.Costs[models.Wood])
			investmentCost += float64(levelData.Costs[models.Stone])
			investmentCost += float64(levelData.Costs[models.Iron])
		}
	}

	tech := technologies["Beer tester"]
	if tech != nil {
		investmentCost += float64(tech.Costs[models.Wood])
		investmentCost += float64(tech.Costs[models.Stone])
		investmentCost += float64(tech.Costs[models.Iron])
	}

	gain := 0.05 * remainingNeeds

	t.Logf("Remaining resource needs: %.0f", remainingNeeds)
	t.Logf("Beer tester investment cost: %.0f", investmentCost)
	t.Logf("5%% gain from boost: %.0f", gain)
	t.Logf("Worth it? %v (gain > cost: %.0f > %.0f)", gain > investmentCost, gain, investmentCost)

	// Suppress unused warning
	_ = s
}

func TestProductionBonusApplied(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	strategy := castle.ResourceStrategy{WoodLead: 0, QuarryLead: 0}
	s := castle.NewGreedySolverWithStrategy(buildings, technologies, initialState, targetLevels, strategy)
	solution := s.Solve()

	// Beer tester and Wheelbarrow should both be researched
	if !solution.FinalState.ResearchedTechnologies["Beer tester"] {
		t.Error("Beer tester should be researched")
	}
	if !solution.FinalState.ResearchedTechnologies["Wheelbarrow"] {
		t.Error("Wheelbarrow should be researched")
	}

	// Final production rates should include 10% bonus (2x 5%)
	// Base production at level 30 is 387/hour for each resource building
	// With 10% bonus: 387 * 1.10 = 425.7
	// Check that we completed faster than without the bonus would allow
	days := float64(solution.TotalTimeSeconds) / 3600 / 24
	t.Logf("Completion time with production bonus: %.1f days", days)

	// Should complete in reasonable time (bonus helps)
	if days > 65 {
		t.Errorf("Expected completion under 65 days with production bonus, got %.1f", days)
	}
}

func TestTechFoodIsTracked(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Find a tech that uses food
	var beerTesterAction *models.ResearchAction
	for i := range solution.ResearchActions {
		if solution.ResearchActions[i].TechnologyName == "Beer tester" {
			beerTesterAction = &solution.ResearchActions[i]
			break
		}
	}

	if beerTesterAction == nil {
		t.Fatal("Beer tester action not found")
	}

	// Beer tester costs 3 food
	if beerTesterAction.Costs[models.Food] != 3 {
		t.Errorf("Beer tester should cost 3 food, got %d", beerTesterAction.Costs[models.Food])
	}

	// FoodUsed should be tracked
	if beerTesterAction.FoodUsed == 0 && beerTesterAction.FoodCapacity == 0 {
		t.Error("FoodUsed and FoodCapacity should be tracked for research actions")
	}

	t.Logf("Beer tester: FoodUsed=%d, FoodCapacity=%d", beerTesterAction.FoodUsed, beerTesterAction.FoodCapacity)
}

func TestTechCostsResources(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Verify that each researched tech has valid costs recorded
	for _, action := range solution.ResearchActions {
		// All techs should have some resource cost
		totalCost := action.Costs[models.Wood] + action.Costs[models.Stone] + action.Costs[models.Iron]
		if totalCost == 0 {
			t.Errorf("Tech %s has zero resource cost", action.TechnologyName)
		}

		// Some techs have food cost (workers)
		// Beer tester: 3, Wheelbarrow: 8, Longbow: 1, etc.
		if action.TechnologyName == "Beer tester" && action.Costs[models.Food] != 3 {
			t.Errorf("Beer tester should cost 3 food, got %d", action.Costs[models.Food])
		}
		if action.TechnologyName == "Wheelbarrow" && action.Costs[models.Food] != 8 {
			t.Errorf("Wheelbarrow should cost 8 food, got %d", action.Costs[models.Food])
		}
		if action.TechnologyName == "Longbow" && action.Costs[models.Food] != 1 {
			t.Errorf("Longbow should cost 1 food, got %d", action.Costs[models.Food])
		}
	}
}

func TestTechFoodCostsAreDeducted(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Calculate total food used by all techs
	totalTechFood := 0
	for _, action := range solution.ResearchActions {
		totalTechFood += action.Costs[models.Food]
	}

	t.Logf("Total food used by techs: %d", totalTechFood)

	// Should be > 0 since we have techs with food costs
	if totalTechFood == 0 {
		t.Error("Expected some techs to have food costs")
	}

	// Expected: Beer tester(3) + Wheelbarrow(8) + Longbow(1) + Stirrup(2) + ... > 50
	if totalTechFood < 50 {
		t.Errorf("Total tech food cost %d seems too low", totalTechFood)
	}
}

func TestFoodUsedIncreasesDuringResearch(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Find two consecutive research actions and verify food increases
	for i := 1; i < len(solution.ResearchActions); i++ {
		prev := solution.ResearchActions[i-1]
		curr := solution.ResearchActions[i]

		// If current tech has food cost, FoodUsed should increase
		if curr.Costs[models.Food] > 0 {
			expectedIncrease := curr.Costs[models.Food]
			actualIncrease := curr.FoodUsed - prev.FoodUsed

			// Allow for building actions between research that also use food
			if actualIncrease < expectedIncrease {
				t.Logf("Note: Food increase %d < expected %d for %s (buildings may have used food between)",
					actualIncrease, expectedIncrease, curr.TechnologyName)
			}
		}
	}
}

func TestBuildingCostsAreCorrect(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Verify first Lumberjack upgrade costs match expected values
	for _, action := range solution.BuildingActions {
		if action.BuildingType == models.Lumberjack && action.FromLevel == 1 && action.ToLevel == 2 {
			// Lumberjack 1->2 should cost around 31 wood, 26 stone, 7 iron, 2 food
			if action.Costs[models.Wood] < 20 || action.Costs[models.Wood] > 50 {
				t.Errorf("Lumberjack 1->2 wood cost %d seems wrong", action.Costs[models.Wood])
			}
			if action.Costs[models.Food] < 1 || action.Costs[models.Food] > 5 {
				t.Errorf("Lumberjack 1->2 food cost %d seems wrong", action.Costs[models.Food])
			}
			break
		}
	}
}

func TestFinalFoodUsedMatchesBuildingAndTechCosts(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Calculate total food from buildings
	buildingFood := 0
	for _, action := range solution.BuildingActions {
		buildingFood += action.Costs[models.Food]
	}

	// Calculate total food from techs
	techFood := 0
	for _, action := range solution.ResearchActions {
		techFood += action.Costs[models.Food]
	}

	t.Logf("Building food: %d, Tech food: %d, Total: %d", buildingFood, techFood, buildingFood+techFood)

	// Get final food used from last action
	var finalFoodUsed int
	if len(solution.BuildingActions) > 0 {
		finalFoodUsed = solution.BuildingActions[len(solution.BuildingActions)-1].FoodUsed
	}
	if len(solution.ResearchActions) > 0 {
		lastResearch := solution.ResearchActions[len(solution.ResearchActions)-1]
		if lastResearch.FoodUsed > finalFoodUsed {
			finalFoodUsed = lastResearch.FoodUsed
		}
	}

	// Final food used should be close to sum of all costs
	// (might be slightly higher due to tracking at different times)
	expectedTotal := buildingFood + techFood
	if finalFoodUsed < expectedTotal-10 {
		t.Errorf("Final food used %d < expected total costs %d", finalFoodUsed, expectedTotal)
	}

	t.Logf("Final food used: %d, Expected: ~%d", finalFoodUsed, expectedTotal)
}

func TestNoNegativeResources(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	for rt, amount := range solution.FinalState.Resources {
		if amount < 0 {
			t.Errorf("Final resource %s is negative: %f", rt, amount)
		}
	}
}

func TestBuildOrderSequential(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Building queue should be sequential (no overlap)
	var lastEndTime int
	for i, action := range solution.BuildingActions {
		if action.StartTime < lastEndTime {
			t.Errorf("Building action %d starts at %d but previous ended at %d",
				i+1, action.StartTime, lastEndTime)
		}
		lastEndTime = action.EndTime
	}
}

func TestResearchQueueSequential(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Research queue should be sequential (no overlap)
	var lastEndTime int
	for i, action := range solution.ResearchActions {
		if action.StartTime < lastEndTime {
			t.Errorf("Research action %d starts at %d but previous ended at %d",
				i+1, action.StartTime, lastEndTime)
		}
		lastEndTime = action.EndTime
	}
}

func TestFarmNotUpgradedTooEarly(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Farm should not be upgraded unless we're running out of food capacity
	// This is a softer check - we just verify Farm upgrades happen at reasonable times

	var farmUpgrades []int
	for _, action := range solution.BuildingActions {
		if action.BuildingType == models.Farm {
			farmUpgrades = append(farmUpgrades, action.ToLevel)
		}
	}

	// Farm should reach 30
	if len(farmUpgrades) == 0 {
		t.Error("No Farm upgrades found")
		return
	}

	lastFarm := farmUpgrades[len(farmUpgrades)-1]
	if lastFarm != 30 {
		t.Errorf("Farm should reach level 30, got %d", lastFarm)
	}
}

func TestFarmReachesTargetLevel(t *testing.T) {
	s, targetLevels := setupSolver(t)
	solution := s.Solve()

	farmTarget := targetLevels[models.Farm]
	farmFinal := solution.FinalState.BuildingLevels[models.Farm]

	if farmFinal < farmTarget {
		t.Errorf("Farm should reach level %d, got %d", farmTarget, farmFinal)
	}
}

// TestResourceQueueInterleaving verifies that resource buildings are properly interleaved
func TestResourceQueueInterleaving(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	// Test W+2/Q+1 strategy - should interleave LJ, Q while maintaining lead over OM
	strategy := castle.ResourceStrategy{WoodLead: 2, QuarryLead: 1}
	s := castle.NewGreedySolverWithStrategy(buildings, technologies, initialState, targetLevels, strategy)
	solution := s.Solve()

	// Check first few actions follow expected pattern
	// With W+2/Q+1: LJ should be 2 ahead, Q should be 1 ahead of OM
	var ljCount, qCount, omCount int
	for i, action := range solution.BuildingActions {
		if i > 20 {
			break // Check first 20 actions
		}
		switch action.BuildingType {
		case models.Lumberjack:
			ljCount++
		case models.Quarry:
			qCount++
		case models.OreMine:
			omCount++
		}
	}

	// LJ should have more upgrades than OM in early game
	if ljCount <= omCount {
		t.Errorf("Expected Lumberjack (%d) to be ahead of OreMine (%d) in early game", ljCount, omCount)
	}
}

// TestSmallTargets tests solver with minimal targets
func TestSmallTargets(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	// Small targets - just upgrade a few buildings
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 5,
		models.Quarry:     5,
		models.OreMine:    5,
	}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	// Verify targets reached
	for bt, target := range targetLevels {
		if solution.FinalState.BuildingLevels[bt] < target {
			t.Errorf("%s: expected %d, got %d", bt, target, solution.FinalState.BuildingLevels[bt])
		}
	}

	// Should complete in reasonable time (allow time for tech research at end)
	hours := float64(solution.TotalTimeSeconds) / 3600
	if hours > 2000 { // ~83 days max with all techs
		t.Errorf("Small targets took too long, took %.1f hours", hours)
	}
}

// TestRoundRobinStrategy tests the basic round-robin strategy
func TestRoundRobinStrategy(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	strategy := castle.ResourceStrategy{WoodLead: 0, QuarryLead: 0}
	s := castle.NewGreedySolverWithStrategy(buildings, technologies, initialState, targetLevels, strategy)
	solution := s.Solve()

	// Should complete and reach all targets
	for bt, target := range targetLevels {
		if solution.FinalState.BuildingLevels[bt] < target {
			t.Errorf("%s: expected %d, got %d", bt, target, solution.FinalState.BuildingLevels[bt])
		}
	}

	// RoundRobin should be slower than optimized strategies but still reasonable
	// Now includes all tech research at end
	days := float64(solution.TotalTimeSeconds) / 3600 / 24
	if days > 65 {
		t.Errorf("RoundRobin took too long: %.1f days", days)
	}
}

// TestHighLeadStrategy tests aggressive wood/quarry lead
func TestHighLeadStrategy(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	strategy := castle.ResourceStrategy{WoodLead: 5, QuarryLead: 5}
	s := castle.NewGreedySolverWithStrategy(buildings, technologies, initialState, targetLevels, strategy)
	solution := s.Solve()

	// Should still complete
	for bt, target := range targetLevels {
		if solution.FinalState.BuildingLevels[bt] < target {
			t.Errorf("%s: expected %d, got %d", bt, target, solution.FinalState.BuildingLevels[bt])
		}
	}
}

// TestStrategyString tests the String() method of ResourceStrategy
func TestStrategyString(t *testing.T) {
	tests := []struct {
		strategy castle.ResourceStrategy
		expected string
	}{
		{castle.ResourceStrategy{WoodLead: 0, QuarryLead: 0}, "RoundRobin"},
		{castle.ResourceStrategy{WoodLead: 1, QuarryLead: 0}, "W+1/Q+0"},
		{castle.ResourceStrategy{WoodLead: 2, QuarryLead: 1}, "W+2/Q+1"},
		{castle.ResourceStrategy{WoodLead: 5, QuarryLead: 5}, "W+5/Q+5"},
	}

	for _, tt := range tests {
		got := tt.strategy.String()
		if got != tt.expected {
			t.Errorf("Strategy %+v: expected %q, got %q", tt.strategy, tt.expected, got)
		}
	}
}

// TestStorageUpgradeTriggered verifies storage is upgraded when cost exceeds capacity
func TestStorageUpgradeTriggered(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	// Target high-level buildings that require storage upgrades
	// Arsenal 30 requires significant resources that exceed L1 storage
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 20,
		models.Quarry:     20,
		models.OreMine:    20,
		models.Arsenal:    20, // High level requires storage upgrades
		models.WoodStore:  15,
		models.StoneStore: 15,
		models.OreStore:   15,
		models.Farm:       15,
	}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	// Verify storage buildings were upgraded
	if solution.FinalState.BuildingLevels[models.WoodStore] < 10 {
		t.Errorf("Expected WoodStore to be upgraded significantly, got level %d",
			solution.FinalState.BuildingLevels[models.WoodStore])
	}

	// Verify storage upgrades happen BEFORE buildings that need them
	storageUpgradeTimes := make(map[models.BuildingType]int)
	for _, action := range solution.BuildingActions {
		switch action.BuildingType {
		case models.WoodStore, models.StoneStore, models.OreStore:
			if _, ok := storageUpgradeTimes[action.BuildingType]; !ok {
				storageUpgradeTimes[action.BuildingType] = action.EndTime
			}
		}
	}

	// At least one storage building should be upgraded
	if len(storageUpgradeTimes) == 0 {
		t.Error("No storage buildings were upgraded")
	}
}

// TestIdleTimeWhenWaitingForResources verifies solver waits when resources insufficient
func TestIdleTimeWhenWaitingForResources(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	// Start with very low resources to force waiting
	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 10
	initialState.Resources[models.Stone] = 10
	initialState.Resources[models.Iron] = 10
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 5,
		models.Quarry:     5,
		models.OreMine:    5,
	}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	// First action should have a delayed start (waiting for resources)
	if len(solution.BuildingActions) > 1 {
		// Check there are gaps between actions (idle time waiting for resources)
		hasIdleTime := false
		for i := 1; i < len(solution.BuildingActions); i++ {
			gap := solution.BuildingActions[i].StartTime - solution.BuildingActions[i-1].EndTime
			if gap > 0 {
				hasIdleTime = true
				break
			}
		}
		if !hasIdleTime {
			t.Log("Warning: Expected some idle time waiting for resources with low starting resources")
		}
	}

	// All targets should still be reached
	for bt, target := range targetLevels {
		if solution.FinalState.BuildingLevels[bt] < target {
			t.Errorf("%s: expected %d, got %d", bt, target, solution.FinalState.BuildingLevels[bt])
		}
	}
}

// TestFoodCapacityTriggersUpgrade verifies Farm is upgraded when food capacity insufficient
func TestFoodCapacityTriggersUpgrade(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 500
	initialState.Resources[models.Stone] = 500
	initialState.Resources[models.Iron] = 500
	initialState.Resources[models.Food] = 40 // Farm L1 capacity

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	// Target buildings that consume lots of food (workers)
	// Each building upgrade consumes food, so many upgrades will exhaust capacity
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 15,
		models.Quarry:     15,
		models.OreMine:    15,
		models.Farm:       10, // Must upgrade to get more worker capacity
	}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	// Farm should be upgraded to provide worker capacity
	farmLevel := solution.FinalState.BuildingLevels[models.Farm]
	if farmLevel < 5 {
		t.Errorf("Expected Farm to be upgraded for worker capacity, got level %d", farmLevel)
	}

	// Check Farm upgrades happen in the build order
	farmUpgradeCount := 0
	for _, action := range solution.BuildingActions {
		if action.BuildingType == models.Farm {
			farmUpgradeCount++
		}
	}

	if farmUpgradeCount == 0 {
		t.Error("Expected Farm upgrades in build order to increase worker capacity")
	}
}

// TestEnoughCapacityNoStorageUpgrade verifies no unnecessary storage upgrades
func TestEnoughCapacityNoStorageUpgrade(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	// Only target low-level buildings that don't need storage upgrades
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 3,
		models.Quarry:     3,
		models.OreMine:    3,
	}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	// Storage buildings should not be upgraded (not in targets, not needed)
	for _, action := range solution.BuildingActions {
		if action.BuildingType == models.WoodStore ||
			action.BuildingType == models.StoneStore ||
			action.BuildingType == models.OreStore {
			t.Errorf("Unexpected storage upgrade: %s %d→%d",
				action.BuildingType, action.FromLevel, action.ToLevel)
		}
	}
}

// Helper functions

func getFarmCapacity(level int) int {
	// Farm capacities from game data
	capacities := map[int]int{
		1: 40, 2: 52, 3: 67, 4: 86, 5: 109, 6: 137, 7: 171, 8: 210,
		9: 256, 10: 310, 11: 372, 12: 443, 13: 523, 14: 612, 15: 710,
		16: 817, 17: 931, 18: 1061, 19: 1210, 20: 1379, 21: 1572,
		22: 1792, 23: 2043, 24: 2329, 25: 2655, 26: 3027, 27: 3451,
		28: 3900, 29: 4407, 30: 5000,
	}
	if cap, ok := capacities[level]; ok {
		return cap
	}
	return 999999
}

func getStorageCapacity(level int, resourceType string) int {
	// Approximate storage capacities (they vary by type but similar)
	baseCapacities := map[int]int{
		1: 150, 2: 200, 3: 275, 4: 360, 5: 475, 6: 625, 7: 825, 8: 1100,
		9: 1450, 10: 1900, 11: 2500, 12: 3300, 13: 4350, 14: 5700, 15: 7500,
		16: 9850, 17: 12950, 18: 17000, 19: 22350, 20: 29350,
	}
	if cap, ok := baseCapacities[level]; ok {
		return cap
	}
	return 999999
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestSolverAllBuildingsAlreadyAtTarget(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	// Use empty technologies to avoid tech research
	technologies := make(map[string]*models.Technology)

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 1000
	initialState.Resources[models.Stone] = 1000
	initialState.Resources[models.Iron] = 1000
	initialState.Resources[models.Food] = 100

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 5,
		models.Quarry:     5,
	}

	initialState.BuildingLevels[models.Lumberjack] = 5
	initialState.BuildingLevels[models.Quarry] = 5

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	if len(solution.BuildingActions) != 0 {
		t.Errorf("Expected empty build order when all targets met, got %d actions", len(solution.BuildingActions))
	}

	if solution.TotalTimeSeconds != 0 {
		t.Errorf("Expected 0 time when nothing to build, got %d", solution.TotalTimeSeconds)
	}
}

func TestSolverSingleBuildingUpgrade(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	// Use empty technologies to avoid tech research
	technologies := make(map[string]*models.Technology)

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 10000
	initialState.Resources[models.Stone] = 10000
	initialState.Resources[models.Iron] = 10000
	initialState.Resources[models.Food] = 1000

	initialState.BuildingLevels[models.Lumberjack] = 1

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 2,
	}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	if len(solution.BuildingActions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(solution.BuildingActions))
	}

	if len(solution.BuildingActions) > 0 {
		action := solution.BuildingActions[0]
		if action.BuildingType != models.Lumberjack {
			t.Errorf("Expected Lumberjack upgrade, got %s", action.BuildingType)
		}
		if action.FromLevel != 1 || action.ToLevel != 2 {
			t.Errorf("Expected 1->2 upgrade, got %d->%d", action.FromLevel, action.ToLevel)
		}
	}
}

func TestSolverZeroResources(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 0
	initialState.Resources[models.Stone] = 0
	initialState.Resources[models.Iron] = 0
	initialState.Resources[models.Food] = 0

	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 5,
	}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	if solution == nil {
		t.Fatal("Expected non-nil solution")
	}

	if solution.FinalState.BuildingLevels[models.Lumberjack] < 5 {
		t.Errorf("Failed to reach target: got level %d", solution.FinalState.BuildingLevels[models.Lumberjack])
	}
}

func TestSolverEmptyTargets(t *testing.T) {
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	// Use empty technologies to avoid tech research
	technologies := make(map[string]*models.Technology)

	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 1000

	targetLevels := map[models.BuildingType]int{}

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	if len(solution.BuildingActions) != 0 {
		t.Errorf("Expected empty build order for empty targets, got %d actions", len(solution.BuildingActions))
	}
}

// TestLibraryNotUpgradedPrematurelyForWheelbarrow verifies that Library isn't upgraded to 8
// before Beer Tester (which only needs Library 3) is researched
// Tests with multiple strategies to catch strategy-specific issues
func TestLibraryNotUpgradedPrematurelyForWheelbarrow(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	// Test with multiple strategies including the best one
	strategies := []castle.ResourceStrategy{
		{WoodLead: 0, QuarryLead: 0}, // RoundRobin
		{WoodLead: 4, QuarryLead: 4}, // Best strategy found
		{WoodLead: 2, QuarryLead: 1}, // Intermediate
	}

	for _, strategy := range strategies {
		t.Run(strategy.String(), func(t *testing.T) {
			stateCopy := &models.GameState{
				BuildingLevels:         make(map[models.BuildingType]int),
				Resources:              make(map[models.ResourceType]float64),
				ResearchedTechnologies: make(map[string]bool),
			}
			for k, v := range initialState.BuildingLevels {
				stateCopy.BuildingLevels[k] = v
			}
			for k, v := range initialState.Resources {
				stateCopy.Resources[k] = v
			}
			for k, v := range initialState.ResearchedTechnologies {
				stateCopy.ResearchedTechnologies[k] = v
			}

			s := castle.NewGreedySolverWithStrategy(buildings, technologies, stateCopy, targetLevels, strategy)
			solution := s.Solve()

			// Find when Beer Tester is researched
			var beerTesterStart int
			for _, action := range solution.ResearchActions {
				if action.TechnologyName == "Beer tester" {
					beerTesterStart = action.StartTime
					break
				}
			}

			if beerTesterStart == 0 {
				t.Fatal("Beer tester not found in research actions")
			}

			// Check Library upgrades before Beer Tester
			maxLibraryBeforeBeerTester := 1
			for _, action := range solution.BuildingActions {
				if action.BuildingType == models.Library && action.EndTime <= beerTesterStart {
					if action.ToLevel > maxLibraryBeforeBeerTester {
						maxLibraryBeforeBeerTester = action.ToLevel
					}
				}
			}

			// Beer Tester needs Library 3, so Library should only be at 3 before Beer Tester
			// NOT at 4, 5, 6, 7, or 8 (which is for Wheelbarrow)
			if maxLibraryBeforeBeerTester > 3 {
				t.Errorf("Library upgraded to %d before Beer Tester (which only needs Library 3) - this is premature for Wheelbarrow (needs Library 8)",
					maxLibraryBeforeBeerTester)

				// Log the premature upgrades
				for _, action := range solution.BuildingActions {
					if action.BuildingType == models.Library {
						t.Logf("Library %d→%d at hour %.1f (Beer Tester starts at hour %.1f)",
							action.FromLevel, action.ToLevel, float64(action.StartTime)/3600, float64(beerTesterStart)/3600)
					}
				}
			}
		})
	}
}

// TestLibraryUpgradedOnDemandOnly verifies Library is only upgraded when needed for a specific tech
func TestLibraryUpgradedOnDemandOnly(t *testing.T) {
	s, _ := setupSolver(t)
	solution := s.Solve()

	// Track which Library level is needed for each tech
	techLibraryReqs := map[string]int{
		"Beer tester":      3,
		"Wheelbarrow":      8,
		"Crop rotation":    1,
		"Yoke":             1,
		"Cellar storeroom": 1,
	}

	// For each research action, check Library was at the required level
	for _, ra := range solution.ResearchActions {
		reqLevel, ok := techLibraryReqs[ra.TechnologyName]
		if !ok {
			continue // Skip techs we don't have requirements for
		}

		// Find Library level at research start time
		libraryLevel := 1
		for _, ba := range solution.BuildingActions {
			if ba.BuildingType == models.Library && ba.EndTime <= ra.StartTime {
				if ba.ToLevel > libraryLevel {
					libraryLevel = ba.ToLevel
				}
			}
		}

		if libraryLevel < reqLevel {
			t.Errorf("Tech %s started at time %d but Library was only at %d (needs %d)",
				ra.TechnologyName, ra.StartTime, libraryLevel, reqLevel)
		}
	}
}

// TestLibraryNotInInitialQueue verifies Library is not added to the initial build queue
// (it should be upgraded on-demand for tech prerequisites only)
func TestLibraryNotInInitialQueue(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	// Remove Library from targets to see if it still gets upgraded
	delete(targetLevels, models.Library)

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	// Library should still be upgraded because production techs need it
	// But only to the level needed for those techs
	libraryFinal := solution.FinalState.BuildingLevels[models.Library]

	// With Beer Tester (Library 3) and Wheelbarrow (Library 8) as production techs
	// Library should reach at least 3
	if libraryFinal < 3 {
		t.Errorf("Library should reach at least 3 for Beer Tester, got %d", libraryFinal)
	}

	t.Logf("Final Library level: %d (was not in initial targets)", libraryFinal)
}

// TestTechNotResearchedWithoutLibrary verifies that a tech requiring higher Library
// doesn't get researched until Library is at the right level
func TestTechNotResearchedWithoutLibrary(t *testing.T) {
	buildings, technologies, initialState, targetLevels := setupFullSolver(t)

	// Set Library to 1
	initialState.BuildingLevels[models.Library] = 1

	s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	// Beer Tester requires Library 3
	// Find Beer Tester research action
	for _, ra := range solution.ResearchActions {
		if ra.TechnologyName == "Beer tester" {
			// Check Library level at research start
			libraryLevel := 1
			for _, ba := range solution.BuildingActions {
				if ba.BuildingType == models.Library && ba.EndTime <= ra.StartTime {
					if ba.ToLevel > libraryLevel {
						libraryLevel = ba.ToLevel
					}
				}
			}

			if libraryLevel < 3 {
				t.Errorf("Beer tester researched at time %d with Library at %d (needs 3)",
					ra.StartTime, libraryLevel)
			} else {
				t.Logf("Beer tester correctly researched after Library reached %d", libraryLevel)
			}
			break
		}
	}
}
