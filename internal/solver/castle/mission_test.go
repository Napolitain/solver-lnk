package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

// TestMissionUniqueness verifies that the same mission can't run twice simultaneously
func TestMissionUniqueness(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Tavern: 5,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())
	state.SetBuildingLevel(models.Tavern, 2)

	// Add enough units to run missions
	state.Army.Spearman = 100

	// Pick first mission
	mission1 := solver.pickBestMissionToStart(state)
	if mission1 == nil {
		t.Fatal("Expected to pick a mission")
	}

	// Simulate starting the mission
	missionState := &models.MissionState{
		Mission:   mission1,
		StartTime: 0,
		EndTime:   mission1.DurationMinutes * 60,
	}
	state.RunningMissions = append(state.RunningMissions, missionState)

	// Reserve units
	for _, req := range mission1.UnitsRequired {
		state.UnitsOnMission.Add(req.Type, req.Count)
		state.Army.Add(req.Type, -req.Count)
	}

	// Try to pick another mission - should NOT pick the same one
	mission2 := solver.pickBestMissionToStart(state)

	if mission2 != nil && mission2.Name == mission1.Name {
		t.Errorf("Should not pick the same mission twice, got %s both times", mission1.Name)
	}

	t.Logf("First mission: %s, Second mission: %v", mission1.Name, mission2)
}

// TestMissionTavernLevelMin verifies missions require minimum tavern level
func TestMissionTavernLevelMin(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Tavern: 10,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())

	// Set tavern to level 1
	state.SetBuildingLevel(models.Tavern, 1)

	// Add plenty of units
	state.Army.Spearman = 500
	state.Army.Archer = 500
	state.Army.Horseman = 500
	state.Army.Crossbowman = 500
	state.Army.Lancer = 500

	// Pick mission at level 1
	mission := solver.pickBestMissionToStart(state)

	if mission == nil {
		t.Fatal("Expected to pick a mission at tavern level 1")
	}

	// Mission should be available at tavern level 1
	if mission.TavernLevel > 1 {
		t.Errorf("Mission %s requires tavern level %d but we're at level 1",
			mission.Name, mission.TavernLevel)
	}

	t.Logf("Picked mission at tavern 1: %s (requires tavern %d)", mission.Name, mission.TavernLevel)
}

// TestMissionTavernLevelMax verifies missions become unavailable at higher tavern levels
func TestMissionTavernLevelMax(t *testing.T) {
	missions, err := loader.LoadMissionsFromFile("../../../data")
	if err != nil {
		t.Fatalf("Failed to load missions: %v", err)
	}

	// Find "Overtime wood" mission
	var overtimeWood *models.Mission
	for _, m := range missions {
		if m.Name == "Overtime wood" {
			overtimeWood = m
			break
		}
	}

	if overtimeWood == nil {
		t.Fatal("Overtime wood mission not found")
	}

	// Verify it has max tavern level set
	if overtimeWood.MaxTavernLevel == 0 {
		t.Error("Overtime wood should have max_tavern_level set")
	}

	t.Logf("Overtime wood: tavern %d-%d", overtimeWood.TavernLevel, overtimeWood.MaxTavernLevel)

	// Now test that solver respects max level
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Tavern: 10,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())

	// Set tavern to level 10 (above max for "Overtime wood")
	state.SetBuildingLevel(models.Tavern, 10)

	// Add plenty of units
	state.Army.Spearman = 500

	// Pick mission - should NOT pick "Overtime wood"
	for i := 0; i < 10; i++ {
		mission := solver.pickBestMissionToStart(state)
		if mission == nil {
			break
		}

		if mission.Name == "Overtime wood" {
			t.Errorf("Should not pick Overtime wood at tavern 10, but got it")
		}

		// Simulate running this mission
		state.RunningMissions = append(state.RunningMissions, &models.MissionState{
			Mission:   mission,
			StartTime: 0,
			EndTime:   mission.DurationMinutes * 60,
		})
	}
}

// TestMissionAvailabilityByTavernLevel verifies correct missions at each level
func TestMissionAvailabilityByTavernLevel(t *testing.T) {
	missions, err := loader.LoadMissionsFromFile("../../../data")
	if err != nil {
		t.Fatalf("Failed to load missions: %v", err)
	}

	// Expected missions per tavern level (based on data/tavern)
	expectedMissions := map[int][]string{
		1:  {"Overtime wood"},
		2:  {"Overtime wood", "Overtime stone", "Hunting"},
		3:  {"Overtime wood", "Overtime stone", "Hunting", "Overtime ore", "Chop wood"},
		4:  {"Mandatory overtime", "Hunting", "Chop wood", "Help stone cutters"},
		5:  {"Mandatory overtime", "Hunting", "Chop wood", "Help stone cutters", "Market day"},
		10: {"Mandatory overtime", "Forging tools", "Create a trading post", "Collect taxes", "Chase bandits away", "Castle festival", "Jousting"},
	}

	for level, expected := range expectedMissions {
		available := getAvailableMissions(missions, level)

		for _, expectedName := range expected {
			found := false
			for _, m := range available {
				if m.Name == expectedName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Tavern %d: expected mission '%s' to be available", level, expectedName)
			}
		}

		t.Logf("Tavern %d: %d missions available", level, len(available))
	}
}

// getAvailableMissions returns missions available at a specific tavern level
func getAvailableMissions(missions []*models.Mission, tavernLevel int) []*models.Mission {
	var available []*models.Mission
	for _, m := range missions {
		if m.TavernLevel <= tavernLevel {
			if m.MaxTavernLevel == 0 || tavernLevel <= m.MaxTavernLevel {
				available = append(available, m)
			}
		}
	}
	return available
}

// TestNoParallelDuplicateMissions runs a simulation and verifies no duplicate missions
func TestNoParallelDuplicateMissions(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Quarry:     10,
		models.OreMine:    10,
		models.Tavern:     5,
		models.Farm:       10,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	initialState := models.NewGameState()
	initialState.BuildingLevels[models.Lumberjack] = 1
	initialState.BuildingLevels[models.Quarry] = 1
	initialState.BuildingLevels[models.OreMine] = 1
	initialState.BuildingLevels[models.Tavern] = 1
	initialState.BuildingLevels[models.Farm] = 1

	solution := solver.Solve(initialState)

	// Check for duplicate parallel missions
	for i, m1 := range solution.MissionActions {
		for j, m2 := range solution.MissionActions {
			if i >= j {
				continue
			}

			// Same mission name
			if m1.MissionName != m2.MissionName {
				continue
			}

			// Check if they overlap in time
			if m1.StartTime < m2.EndTime && m2.StartTime < m1.EndTime {
				t.Errorf("Duplicate parallel mission detected: %s at %d-%d and %d-%d",
					m1.MissionName, m1.StartTime, m1.EndTime, m2.StartTime, m2.EndTime)
			}
		}
	}

	t.Logf("Checked %d mission actions for duplicates", len(solution.MissionActions))
}

// TestMissionResourceCosts verifies mission resource costs are tracked
func TestMissionResourceCosts(t *testing.T) {
	missions, err := loader.LoadMissionsFromFile("../../../data")
	if err != nil {
		t.Fatalf("Failed to load missions: %v", err)
	}

	// Find missions with resource costs
	var missionsWithCosts []*models.Mission
	for _, m := range missions {
		if m.ResourceCosts.Wood > 0 || m.ResourceCosts.Stone > 0 || m.ResourceCosts.Iron > 0 {
			missionsWithCosts = append(missionsWithCosts, m)
		}
	}

	if len(missionsWithCosts) == 0 {
		t.Fatal("Expected some missions to have resource costs")
	}

	for _, m := range missionsWithCosts {
		t.Logf("Mission '%s' costs: W=%d S=%d I=%d",
			m.Name, m.ResourceCosts.Wood, m.ResourceCosts.Stone, m.ResourceCosts.Iron)

		// Net reward should still be positive
		netReward := m.NetAverageReward()
		if netReward <= 0 {
			t.Errorf("Mission '%s' has non-positive net reward: %f", m.Name, netReward)
		}
	}
}

// TestMissionUnitNeedsAtTavern10 verifies that Lancers are needed for Tavern 10 missions
func TestMissionUnitNeedsAtTavern10(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Tavern: 10,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)

	// Calculate unit needs at Tavern 10
	needs := solver.calculateMissionUnitNeeds(10)

	t.Logf("Unit needs at Tavern 10:")
	for ut, count := range needs {
		t.Logf("  %s: %d", ut, count)
	}

	// Verify Lancers are needed (for Jousting and Castle festival)
	if needs[models.Lancer] < 100 {
		t.Errorf("Expected at least 100 Lancers needed, got %d", needs[models.Lancer])
	}

	// Verify other unit types
	if needs[models.Spearman] < 200 {
		t.Errorf("Expected at least 200 Spearmen needed, got %d", needs[models.Spearman])
	}
	if needs[models.Archer] < 200 {
		t.Errorf("Expected at least 200 Archers needed, got %d", needs[models.Archer])
	}
	if needs[models.Horseman] < 100 {
		t.Errorf("Expected at least 100 Horsemen needed, got %d", needs[models.Horseman])
	}
	if needs[models.Crossbowman] < 100 {
		t.Errorf("Expected at least 100 Crossbowmen needed, got %d", needs[models.Crossbowman])
	}
}

// TestUnitTechsNeededForMissions verifies that Horse armour is detected as needed
func TestUnitTechsNeededForMissions(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Tavern: 10,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())
	state.SetBuildingLevel(models.Tavern, 10)

	techs := solver.getUnitTechsNeededForMissions(state)

	t.Logf("Unit techs needed: %v", techs)

	// Check for expected techs
	techMap := make(map[string]bool)
	for _, tech := range techs {
		techMap[tech] = true
	}

	expectedTechs := []string{"Longbow", "Crossbow", "Stirrup", "Horse armour"}
	for _, expected := range expectedTechs {
		if !techMap[expected] {
			t.Errorf("Expected tech %q not found in unit techs", expected)
		}
	}
}

// TestAllMissionsSchedulable verifies that Jousting and Castle Festival are scheduled
func TestAllMissionsSchedulable(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	techs, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	solver := castle.NewTestSolver(buildings, techs, missions, map[models.BuildingType]int{
		models.Lumberjack:     30,
		models.Quarry:         30,
		models.OreMine:        30,
		models.WoodStore:      20,
		models.StoneStore:     20,
		models.OreStore:       20,
		models.Farm:           30,
		models.Tavern:         10,
		models.Keep:           10,
		models.Arsenal:        30,
		models.Fortifications: 20,
		models.Market:         8,
		models.Library:        10,
	})

	initialState := &models.GameState{
		BuildingLevels: map[models.BuildingType]int{
			models.Lumberjack:     1,
			models.Quarry:         1,
			models.OreMine:        1,
			models.WoodStore:      1,
			models.StoneStore:     1,
			models.OreStore:       1,
			models.Farm:           1,
			models.Tavern:         1,
			models.Keep:           1,
			models.Arsenal:        1,
			models.Fortifications: 1,
			models.Market:         1,
			models.Library:        1,
		},
		Resources: map[models.ResourceType]float64{
			models.Wood:  1000,
			models.Stone: 1000,
			models.Iron:  1000,
		},
	}

	solution := solver.Solve(initialState)
	if solution == nil {
		t.Fatal("Expected solution")
	}

	// Check that Jousting and Castle Festival are scheduled
	joustingFound := false
	castleFestivalFound := false

	for _, mission := range solution.MissionActions {
		if mission.MissionName == "Jousting" {
			joustingFound = true
		}
		if mission.MissionName == "Castle festival" {
			castleFestivalFound = true
		}
	}

	if !joustingFound {
		t.Error("Jousting mission not scheduled - need 100 Lancers for Tavern 10 missions")
	}
	if !castleFestivalFound {
		t.Error("Castle Festival mission not scheduled - need 100 Lancers for Tavern 10 missions")
	}
}

// TestLancerCountForMissions verifies at least 100 Lancers are trained for missions
func TestLancerCountForMissions(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	techs, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	solver := castle.NewTestSolver(buildings, techs, missions, map[models.BuildingType]int{
		models.Lumberjack:     30,
		models.Quarry:         30,
		models.OreMine:        30,
		models.WoodStore:      20,
		models.StoneStore:     20,
		models.OreStore:       20,
		models.Farm:           30,
		models.Tavern:         10,
		models.Keep:           10,
		models.Arsenal:        30,
		models.Fortifications: 20,
		models.Market:         8,
		models.Library:        10,
	})

	initialState := &models.GameState{
		BuildingLevels: map[models.BuildingType]int{
			models.Lumberjack:     1,
			models.Quarry:         1,
			models.OreMine:        1,
			models.WoodStore:      1,
			models.StoneStore:     1,
			models.OreStore:       1,
			models.Farm:           1,
			models.Tavern:         1,
			models.Keep:           1,
			models.Arsenal:        1,
			models.Fortifications: 1,
			models.Market:         1,
			models.Library:        1,
		},
		Resources: map[models.ResourceType]float64{
			models.Wood:  1000,
			models.Stone: 1000,
			models.Iron:  1000,
		},
	}

	solution := solver.Solve(initialState)
	if solution == nil {
		t.Fatal("Expected solution")
	}

	// Count Lancers trained
	lancerCount := 0
	for _, train := range solution.TrainingActions {
		if train.UnitType == models.Lancer {
			lancerCount += train.Count
		}
	}

	// Need at least 100 for Jousting/Castle Festival
	if lancerCount < 100 {
		t.Errorf("Expected at least 100 Lancers trained for missions, got %d", lancerCount)
	}
}
