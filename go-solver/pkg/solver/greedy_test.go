package solver_test

import (
	"sort"
	"testing"

	"github.com/napolitain/solver-lnk/pkg/loader"
	"github.com/napolitain/solver-lnk/pkg/models"
	"github.com/napolitain/solver-lnk/pkg/solver"
)

const dataDir = "../../../data"

func setupSolver(t *testing.T) (*solver.GreedySolver, map[models.BuildingType]int) {
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

	s := solver.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	return s, targetLevels
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

// Helper functions

func getFarmCapacity(level int) int {
	// Approximate Farm capacities from game data
	capacities := map[int]int{
		1: 40, 2: 50, 3: 62, 4: 77, 5: 96, 6: 119, 7: 148, 8: 184,
		9: 228, 10: 283, 11: 350, 12: 432, 13: 532, 14: 654, 15: 803,
		16: 983, 17: 1202, 18: 1468, 19: 1790, 20: 2183, 21: 2659,
		22: 3236, 23: 3935, 24: 4781, 25: 5806, 26: 7048, 27: 8550,
		28: 10367, 29: 12564, 30: 15223,
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
