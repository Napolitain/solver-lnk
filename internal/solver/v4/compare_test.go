package v4

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	v3 "github.com/napolitain/solver-lnk/internal/solver/v3"
)

func TestV4MatchesV3(t *testing.T) {
	// Load data
	buildings, err := loader.LoadBuildings("../../../data")
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies("../../../data")
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	// Target levels (same as default in CLI)
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

	// Initial state
	initialState := models.NewGameState()
	for bt := range targetLevels {
		initialState.BuildingLevels[bt] = 1
	}
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120

	// Run V3
	v3Solver := v3.NewSolver(buildings, technologies, targetLevels)
	v3Solution := v3Solver.Solve(cloneGameState(initialState))

	// Run V4
	v4Solver := NewSolver(buildings, technologies, nil, targetLevels)
	v4Solution := v4Solver.Solve(cloneGameState(initialState))

	// Compare results
	t.Logf("V3: %d building actions, %d research actions, total time: %d seconds (%.1f days)",
		len(v3Solution.BuildingActions), len(v3Solution.ResearchActions),
		v3Solution.TotalTimeSeconds, float64(v3Solution.TotalTimeSeconds)/3600/24)

	t.Logf("V4: %d building actions, %d research actions, total time: %d seconds (%.1f days)",
		len(v4Solution.BuildingActions), len(v4Solution.ResearchActions),
		v4Solution.TotalTimeSeconds, float64(v4Solution.TotalTimeSeconds)/3600/24)

	// Count actions per building type
	v3Counts := make(map[models.BuildingType]int)
	v4Counts := make(map[models.BuildingType]int)
	for _, a := range v3Solution.BuildingActions {
		v3Counts[a.BuildingType]++
	}
	for _, a := range v4Solution.BuildingActions {
		v4Counts[a.BuildingType]++
	}

	// Show per-building comparison
	for bt, target := range targetLevels {
		v3Count := v3Counts[bt]
		v4Count := v4Counts[bt]
		v3Final := v3Solution.FinalState.BuildingLevels[bt]
		v4Final := v4Solution.FinalState.BuildingLevels[bt]

		if v3Count != v4Count || v3Final != v4Final {
			t.Logf("  %s: v3=%d actions (final %d), v4=%d actions (final %d), target=%d",
				bt, v3Count, v3Final, v4Count, v4Final, target)
		}
	}

	// Check building action counts match
	if len(v3Solution.BuildingActions) != len(v4Solution.BuildingActions) {
		t.Errorf("Building action count mismatch: v3=%d, v4=%d",
			len(v3Solution.BuildingActions), len(v4Solution.BuildingActions))
	}

	// Check final building levels match
	allMatch := true
	for bt, target := range targetLevels {
		v3Level := v3Solution.FinalState.BuildingLevels[bt]
		v4Level := v4Solution.FinalState.BuildingLevels[bt]

		if v3Level != v4Level {
			t.Errorf("Final level mismatch for %s: v3=%d, v4=%d (target=%d)",
				bt, v3Level, v4Level, target)
			allMatch = false
		}

		if v4Level < target {
			t.Errorf("V4 did not reach target for %s: got=%d, want=%d",
				bt, v4Level, target)
			allMatch = false
		}
	}

	if allMatch {
		t.Log("All building levels match!")
	}

	// V4 may differ in timing - just verify all targets are reached
	// and time is reasonable (not more than 20% longer than V3)
	if v4Solution.TotalTimeSeconds > v3Solution.TotalTimeSeconds {
		timeDiff := v4Solution.TotalTimeSeconds - v3Solution.TotalTimeSeconds
		tolerance := v3Solution.TotalTimeSeconds / 5 // 20%
		if timeDiff > tolerance {
			t.Errorf("V4 is too slow: v3=%d, v4=%d (diff=%d, tolerance=%d)",
				v3Solution.TotalTimeSeconds, v4Solution.TotalTimeSeconds, timeDiff, tolerance)
		}
	} else {
		t.Logf("V4 is faster than V3: v3=%.1f days, v4=%.1f days",
			float64(v3Solution.TotalTimeSeconds)/3600/24,
			float64(v4Solution.TotalTimeSeconds)/3600/24)
	}
}

func cloneGameState(gs *models.GameState) *models.GameState {
	clone := models.NewGameState()
	for bt, level := range gs.BuildingLevels {
		clone.BuildingLevels[bt] = level
	}
	for rt, amount := range gs.Resources {
		clone.Resources[rt] = amount
	}
	for tech, researched := range gs.ResearchedTechnologies {
		clone.ResearchedTechnologies[tech] = researched
	}
	return clone
}
