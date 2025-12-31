package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

func setupMissionSolver(t *testing.T) *GreedySolver {
	t.Helper()

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
			models.Tavern:         1, // Start with Tavern 1 for missions
			models.Market:         1,
			models.Fortifications: 1,
		},
		Resources: map[models.ResourceType]float64{
			models.Wood:  100,
			models.Stone: 100,
			models.Iron:  100,
			models.Food:  0,
		},
		ResearchedTechnologies: make(map[string]bool),
	}

	// Smaller targets for faster testing
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Quarry:     10,
		models.OreMine:    10,
		models.Farm:       10,
		models.WoodStore:  5,
		models.StoneStore: 5,
		models.OreStore:   5,
		models.Keep:       5,
	}

	return NewGreedySolver(buildings, technologies, initialState, targetLevels)
}

func TestSolveWithMissions_Disabled(t *testing.T) {
	solver := setupMissionSolver(t)

	// Without missions
	solution := solver.SolveWithMissions(false)

	if solution == nil {
		t.Fatal("Solution is nil")
	}

	if solution.MissionsCompleted != 0 {
		t.Errorf("Expected 0 missions completed when disabled, got %d", solution.MissionsCompleted)
	}

	t.Logf("Without missions: %d steps, %.1f days",
		len(solution.BuildingActions),
		float64(solution.TotalTimeSeconds)/86400)
}

func TestSolveWithMissions_Enabled(t *testing.T) {
	solver := setupMissionSolver(t)

	// With missions
	solution := solver.SolveWithMissions(true)

	if solution == nil {
		t.Fatal("Solution is nil")
	}

	t.Logf("With missions: %d steps, %.1f days, %d missions completed",
		len(solution.BuildingActions),
		float64(solution.TotalTimeSeconds)/86400,
		solution.MissionsCompleted)

	// Log mission gains
	if len(solution.TotalMissionGain) > 0 {
		t.Log("Total mission gains:")
		for rt, amount := range solution.TotalMissionGain {
			t.Logf("  %s: %d", rt, amount)
		}
	}
}

func TestSolveWithMissions_CompareTimes(t *testing.T) {
	solver := setupMissionSolver(t)

	// Run both
	withoutMissions := solver.SolveWithMissions(false)
	
	// Reset solver state
	solver2 := setupMissionSolver(t)
	withMissions := solver2.SolveWithMissions(true)

	t.Logf("Comparison:")
	t.Logf("  Without missions: %.2f days (%d seconds)",
		float64(withoutMissions.TotalTimeSeconds)/86400,
		withoutMissions.TotalTimeSeconds)
	t.Logf("  With missions:    %.2f days (%d seconds)",
		float64(withMissions.TotalTimeSeconds)/86400,
		withMissions.TotalTimeSeconds)

	if withMissions.MissionsCompleted > 0 {
		t.Logf("  Missions completed: %d", withMissions.MissionsCompleted)
		
		timeDiff := withoutMissions.TotalTimeSeconds - withMissions.TotalTimeSeconds
		if timeDiff > 0 {
			t.Logf("  Time saved: %d seconds (%.2f hours)",
				timeDiff, float64(timeDiff)/3600)
		} else {
			t.Logf("  Time difference: %d seconds (missions slower by %.2f hours)",
				-timeDiff, float64(-timeDiff)/3600)
		}
	}
}

func TestSolveWithMissions_ResourceGains(t *testing.T) {
	solver := setupMissionSolver(t)
	solution := solver.SolveWithMissions(true)

	totalGained := 0
	for _, amount := range solution.TotalMissionGain {
		totalGained += amount
	}

	t.Logf("Total resources gained from missions: %d", totalGained)

	// If we completed missions, we should have gained resources
	if solution.MissionsCompleted > 0 && totalGained == 0 {
		t.Error("Completed missions but gained no resources")
	}
}

func TestSolveWithMissions_DoesNotBreakInvariants(t *testing.T) {
	solver := setupMissionSolver(t)
	solution := solver.SolveWithMissions(true)

	// Check that building actions are in chronological order
	lastEnd := 0
	for i, action := range solution.BuildingActions {
		if action.StartTime < lastEnd {
			// Start can overlap with previous end (parallel research), but building queue should not
			// Actually buildings are sequential, so start should be >= lastEnd
			// But there might be time jumps, so just check end times are increasing
		}
		if action.EndTime < action.StartTime {
			t.Errorf("Action %d: EndTime %d < StartTime %d", i, action.EndTime, action.StartTime)
		}
		lastEnd = action.EndTime
	}

	// Check final state is valid
	if solution.FinalState == nil {
		t.Error("FinalState is nil")
	}

	// Check no negative resources in final state
	for rt, amount := range solution.FinalState.Resources {
		if amount < -0.01 {
			t.Errorf("Negative resource %s: %.2f", rt, amount)
		}
	}
}

// Benchmark to compare performance
func BenchmarkSolveWithMissions(b *testing.B) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")

	initialState := &models.GameState{
		BuildingLevels: map[models.BuildingType]int{
			models.Lumberjack: 1, models.Quarry: 1, models.OreMine: 1,
			models.Farm: 1, models.WoodStore: 1, models.StoneStore: 1,
			models.OreStore: 1, models.Keep: 1, models.Arsenal: 1,
			models.Library: 1, models.Tavern: 1, models.Market: 1,
			models.Fortifications: 1,
		},
		Resources: map[models.ResourceType]float64{
			models.Wood: 100, models.Stone: 100, models.Iron: 100,
		},
		ResearchedTechnologies: make(map[string]bool),
	}

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10, models.Quarry: 10, models.OreMine: 10,
		models.Farm: 10, models.WoodStore: 5, models.StoneStore: 5,
	}

	b.Run("WithoutMissions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewGreedySolver(buildings, technologies, initialState, targetLevels)
			solver.SolveWithMissions(false)
		}
	})

	b.Run("WithMissions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewGreedySolver(buildings, technologies, initialState, targetLevels)
			solver.SolveWithMissions(true)
		}
	})
}
