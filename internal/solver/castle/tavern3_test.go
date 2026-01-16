package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

func TestAllMissionsAtTavern3Scheduled(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	techs, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	solver := NewSolver(buildings, techs, missions, map[models.BuildingType]int{
		models.Lumberjack: 15,
		models.Quarry:     15,
		models.OreMine:    15,
		models.Farm:       15,
		models.Tavern:     3,
		models.Arsenal:    5,
	})

	initialState := &models.GameState{
		BuildingLevels: map[models.BuildingType]int{
			models.Lumberjack: 1,
			models.Quarry:     1,
			models.OreMine:    1,
			models.Farm:       1,
			models.Tavern:     1,
			models.Arsenal:    1,
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

	// Count which missions were scheduled
	missionCounts := make(map[string]int)
	for _, mission := range solution.MissionActions {
		missionCounts[mission.MissionName]++
	}

	// Expected missions at tavern level 3: overtime_wood, overtime_stone, overtime_ore, hunting, chop_wood
	expectedMissions := []string{"Overtime wood", "Overtime stone", "Overtime ore", "Hunting", "Chop wood"}

	t.Logf("Total missions scheduled: %d", len(solution.MissionActions))
	for _, expectedName := range expectedMissions {
		count := missionCounts[expectedName]
		t.Logf("  %s: %d times", expectedName, count)
		if count == 0 {
			t.Errorf("Mission '%s' was not scheduled at all (expected at least once)", expectedName)
		}
	}

	// Check that spearmen were trained (required for all tavern 3 missions)
	spearmenTrained := 0
	for _, train := range solution.TrainingActions {
		if train.UnitType == models.Spearman {
			spearmenTrained += train.Count
		}
	}
	t.Logf("Spearmen trained: %d", spearmenTrained)
	if spearmenTrained < 20 {
		t.Errorf("Expected at least 20 spearmen to be trained (for chop_wood mission), got %d", spearmenTrained)
	}

	// Check that horsemen were trained (required for chop_wood mission)
	horsemenTrained := 0
	for _, train := range solution.TrainingActions {
		if train.UnitType == models.Horseman {
			horsemenTrained += train.Count
		}
	}
	t.Logf("Horsemen trained: %d", horsemenTrained)
	if horsemenTrained < 20 {
		t.Errorf("Expected at least 20 horsemen to be trained (for chop_wood mission), got %d", horsemenTrained)
	}
}
