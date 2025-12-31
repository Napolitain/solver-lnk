package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

// TestFullBuildWithMissions tests full castle build (all 30s) with mission support
func TestFullBuildWithMissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full build test in short mode")
	}

	buildings, err := loader.LoadBuildings("../../../data")
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies("../../../data")
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	initialState := &models.GameState{
		BuildingLevels: map[models.BuildingType]int{
			models.Lumberjack:     1,
			models.Quarry:         1,
			models.OreMine:        1,
			models.Farm:           1,
			models.WoodStore:      1,
			models.StoneStore:     1,
			models.OreStore:       1,
			models.Keep:           1,
			models.Arsenal:        1,
			models.Library:        1,
			models.Tavern:         1,
			models.Market:         1,
			models.Fortifications: 1,
		},
		Resources: map[models.ResourceType]float64{
			models.Wood:  120,
			models.Stone: 120,
			models.Iron:  120,
			models.Food:  40,
		},
		ResearchedTechnologies: make(map[string]bool),
	}

	// Full targets (matching CLI defaults)
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

	// Use best known strategy
	strategy := ResourceStrategy{
		WoodLead:       3,
		QuarryLead:     3,
		SwitchLevel:    15,
		LateWoodLead:   1,
		LateQuarryLead: 1,
	}

	t.Log("Running full build comparison...")

	// Without missions
	solver1 := NewGreedySolverWithStrategy(buildings, technologies, copyGameState(initialState), targetLevels, strategy)
	withoutMissions := solver1.SolveWithMissions(false)

	// With missions
	solver2 := NewGreedySolverWithStrategy(buildings, technologies, copyGameState(initialState), targetLevels, strategy)
	withMissions := solver2.SolveWithMissions(true)

	daysWithout := float64(withoutMissions.TotalTimeSeconds) / 86400
	daysWith := float64(withMissions.TotalTimeSeconds) / 86400

	t.Logf("=== FULL BUILD COMPARISON ===")
	t.Logf("Strategy: %s", strategy.String())
	t.Logf("")
	t.Logf("Without missions:")
	t.Logf("  Total time: %.2f days", daysWithout)
	t.Logf("  Building steps: %d", len(withoutMissions.BuildingActions))
	t.Logf("")
	t.Logf("With missions:")
	t.Logf("  Total time: %.2f days", daysWith)
	t.Logf("  Building steps: %d", len(withMissions.BuildingActions))
	t.Logf("  Missions completed: %d", withMissions.MissionsCompleted)
	t.Logf("")

	if withMissions.MissionsCompleted > 0 {
		timeSaved := withoutMissions.TotalTimeSeconds - withMissions.TotalTimeSeconds
		daysSaved := float64(timeSaved) / 86400
		t.Logf("Time saved: %.2f days (%.0f hours)", daysSaved, float64(timeSaved)/3600)

		// Log resource gains
		totalGain := 0
		for _, amount := range withMissions.TotalMissionGain {
			totalGain += amount
		}
		t.Logf("Total resources from missions: %d", totalGain)
	}

	// The baseline is 59.6 days - let's see if we beat it
	baseline := 59.6
	t.Logf("")
	t.Logf("Baseline: %.1f days", baseline)
	if daysWith < baseline {
		t.Logf("✓ BEAT BASELINE by %.2f days!", baseline-daysWith)
	} else {
		t.Logf("✗ Did not beat baseline (%.2f days over)", daysWith-baseline)
	}
}

func copyGameState(state *models.GameState) *models.GameState {
	copy := &models.GameState{
		BuildingLevels:         make(map[models.BuildingType]int),
		Resources:              make(map[models.ResourceType]float64),
		ResearchedTechnologies: make(map[string]bool),
	}
	for k, v := range state.BuildingLevels {
		copy.BuildingLevels[k] = v
	}
	for k, v := range state.Resources {
		copy.Resources[k] = v
	}
	for k, v := range state.ResearchedTechnologies {
		copy.ResearchedTechnologies[k] = v
	}
	return copy
}
