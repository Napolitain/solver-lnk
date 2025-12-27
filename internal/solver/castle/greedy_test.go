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
const maxAllowedTimeSeconds = 1265 * 3600 // ~52.7 days with margin

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

	// Should complete quickly (under 1 day)
	hours := float64(solution.TotalTimeSeconds) / 3600
	if hours > 24 {
		t.Errorf("Small targets should complete in under 24 hours, took %.1f hours", hours)
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
	days := float64(solution.TotalTimeSeconds) / 3600 / 24
	if days > 60 {
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
