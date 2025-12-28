package castle

import (
	"math/rand"
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

// FuzzSolverDeterminism tests that any valid random input produces deterministic output.
// For any generated state X, solving twice must produce identical Y.
func FuzzSolverDeterminism(f *testing.F) {
	// Add seed corpus
	f.Add(int64(12345), 5, 5, 5, 100, 100, 100, 20)
	f.Add(int64(99999), 1, 1, 1, 50, 50, 50, 10)
	f.Add(int64(11111), 15, 12, 10, 500, 400, 300, 50)
	f.Add(int64(22222), 20, 20, 20, 1000, 1000, 1000, 100)

	buildings, err := loader.LoadBuildings("../../../data")
	if err != nil {
		f.Fatalf("Failed to load buildings: %v", err)
	}
	technologies, _ := loader.LoadTechnologies("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 30, models.Quarry: 30, models.OreMine: 30,
		models.Farm: 30, models.WoodStore: 20, models.StoneStore: 20,
		models.OreStore: 20, models.Keep: 10, models.Arsenal: 30,
		models.Library: 10, models.Tavern: 10, models.Market: 8,
		models.Fortifications: 20,
	}

	f.Fuzz(func(t *testing.T, seed int64, ljLevel, qLevel, omLevel int, wood, stone, iron, food int) {
		// Clamp values to valid ranges
		ljLevel = clamp(ljLevel, 1, 25)
		qLevel = clamp(qLevel, 1, 25)
		omLevel = clamp(omLevel, 1, 25)
		wood = clamp(wood, 0, 10000)
		stone = clamp(stone, 0, 10000)
		iron = clamp(iron, 0, 10000)
		food = clamp(food, 0, 500)

		// Generate consistent random state from seed
		rng := rand.New(rand.NewSource(seed))

		state := &models.GameState{
			BuildingLevels: map[models.BuildingType]int{
				models.Lumberjack:     ljLevel,
				models.Quarry:         qLevel,
				models.OreMine:        omLevel,
				models.Farm:           clamp(rng.Intn(20)+1, 1, 25),
				models.WoodStore:      clamp(rng.Intn(15)+1, 1, 18),
				models.StoneStore:     clamp(rng.Intn(15)+1, 1, 18),
				models.OreStore:       clamp(rng.Intn(15)+1, 1, 18),
				models.Keep:           clamp(rng.Intn(8)+1, 1, 9),
				models.Arsenal:        clamp(rng.Intn(20)+1, 1, 25),
				models.Library:        clamp(rng.Intn(8)+1, 1, 9),
				models.Tavern:         clamp(rng.Intn(8)+1, 1, 9),
				models.Market:         clamp(rng.Intn(6)+1, 1, 7),
				models.Fortifications: clamp(rng.Intn(15)+1, 1, 18),
			},
			Resources: map[models.ResourceType]float64{
				models.Wood:  float64(wood),
				models.Stone: float64(stone),
				models.Iron:  float64(iron),
				models.Food:  float64(food),
			},
			ResearchedTechnologies: map[string]bool{},
		}

		// Solve twice with same input
		solution1, strategy1, _ := SolveAllStrategies(buildings, technologies, copyState(state), targetLevels)
		solution2, strategy2, _ := SolveAllStrategies(buildings, technologies, copyState(state), targetLevels)

		// Must be identical
		if strategy1 != strategy2 {
			t.Errorf("Strategy mismatch for seed %d: %s vs %s", seed, strategy1, strategy2)
		}

		if solution1.TotalTimeSeconds != solution2.TotalTimeSeconds {
			t.Errorf("Time mismatch for seed %d: %d vs %d", seed, solution1.TotalTimeSeconds, solution2.TotalTimeSeconds)
		}

		if len(solution1.BuildingActions) != len(solution2.BuildingActions) {
			t.Errorf("Action count mismatch for seed %d: %d vs %d",
				seed, len(solution1.BuildingActions), len(solution2.BuildingActions))
			return
		}

		// Check first 5 actions match
		for i := 0; i < min(5, len(solution1.BuildingActions)); i++ {
			a1, a2 := solution1.BuildingActions[i], solution2.BuildingActions[i]
			if a1.BuildingType != a2.BuildingType || a1.ToLevel != a2.ToLevel {
				t.Errorf("Action %d mismatch for seed %d: %s->%d vs %s->%d",
					i, seed, a1.BuildingType, a1.ToLevel, a2.BuildingType, a2.ToLevel)
			}
		}
	})
}

// copyState creates a deep copy of GameState for test isolation
func copyState(s *models.GameState) *models.GameState {
	cp := &models.GameState{
		BuildingLevels:         make(map[models.BuildingType]int),
		Resources:              make(map[models.ResourceType]float64),
		ResearchedTechnologies: make(map[string]bool),
	}
	for k, v := range s.BuildingLevels {
		cp.BuildingLevels[k] = v
	}
	for k, v := range s.Resources {
		cp.Resources[k] = v
	}
	for k, v := range s.ResearchedTechnologies {
		cp.ResearchedTechnologies[k] = v
	}
	return cp
}

func clamp(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
