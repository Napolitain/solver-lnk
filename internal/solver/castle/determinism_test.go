package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

// TestSolverDeterminism verifies that the solver produces identical results
// for the same input across multiple runs. This guards against non-deterministic
// behavior caused by map iteration order or other sources of randomness.
func TestSolverDeterminism(t *testing.T) {
	buildings, err := loader.LoadBuildings("../../../data")
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}
	technologies, _ := loader.LoadTechnologies("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 30, models.Quarry: 30, models.OreMine: 30,
		models.Farm: 30, models.WoodStore: 20, models.StoneStore: 20,
		models.OreStore: 20, models.Keep: 10, models.Arsenal: 30,
		models.Library: 10, models.Tavern: 10, models.Market: 8,
		models.Fortifications: 20,
	}

	// Test with a mid-game state (more interesting than fresh start)
	initialState := &models.GameState{
		BuildingLevels: map[models.BuildingType]int{
			models.Keep: 1, models.Arsenal: 1, models.Tavern: 1,
			models.Library: 1, models.Fortifications: 1, models.Market: 1,
			models.Farm: 4, models.Lumberjack: 12, models.WoodStore: 2,
			models.Quarry: 11, models.StoneStore: 2, models.OreMine: 9,
			models.OreStore: 1,
		},
		Resources: map[models.ResourceType]float64{
			models.Wood: 55, models.Stone: 65, models.Iron: 63, models.Food: 10,
		},
		ResearchedTechnologies: map[string]bool{},
	}

	const iterations = 100

	// Get first result as baseline
	firstSolution, firstStrategy, _ := SolveAllStrategies(buildings, technologies, copyState(initialState), targetLevels)

	if len(firstSolution.BuildingActions) == 0 {
		t.Fatal("First solution has no building actions")
	}

	firstAction := firstSolution.BuildingActions[0]
	t.Logf("Baseline: strategy=%s, time=%d, first_action=%s->%d",
		firstStrategy, firstSolution.TotalTimeSeconds, firstAction.BuildingType, firstAction.ToLevel)

	// Run many times and verify identical results
	for i := 1; i < iterations; i++ {
		solution, strategy, _ := SolveAllStrategies(buildings, technologies, copyState(initialState), targetLevels)

		if strategy != firstStrategy {
			t.Errorf("Iteration %d: strategy mismatch: got %s, want %s", i, strategy, firstStrategy)
		}

		if solution.TotalTimeSeconds != firstSolution.TotalTimeSeconds {
			t.Errorf("Iteration %d: time mismatch: got %d, want %d",
				i, solution.TotalTimeSeconds, firstSolution.TotalTimeSeconds)
		}

		if len(solution.BuildingActions) != len(firstSolution.BuildingActions) {
			t.Errorf("Iteration %d: action count mismatch: got %d, want %d",
				i, len(solution.BuildingActions), len(firstSolution.BuildingActions))
			continue
		}

		action := solution.BuildingActions[0]
		if action.BuildingType != firstAction.BuildingType || action.ToLevel != firstAction.ToLevel {
			t.Errorf("Iteration %d: first action mismatch: got %s->%d, want %s->%d",
				i, action.BuildingType, action.ToLevel, firstAction.BuildingType, firstAction.ToLevel)
		}
	}
}

// copyState creates a deep copy of GameState for test isolation
func copyState(s *models.GameState) *models.GameState {
	copy := &models.GameState{
		BuildingLevels:         make(map[models.BuildingType]int),
		Resources:              make(map[models.ResourceType]float64),
		ResearchedTechnologies: make(map[string]bool),
	}
	for k, v := range s.BuildingLevels {
		copy.BuildingLevels[k] = v
	}
	for k, v := range s.Resources {
		copy.Resources[k] = v
	}
	for k, v := range s.ResearchedTechnologies {
		copy.ResearchedTechnologies[k] = v
	}
	return copy
}
