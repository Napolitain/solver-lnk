package castle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

// TestGoldenBuildOrder verifies the exact build order doesn't change
// This is a golden master test to ensure optimizations don't alter results
func TestGoldenBuildOrder(t *testing.T) {
	buildings, err := loader.LoadBuildings("../../../data")
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	techs, err := loader.LoadTechnologies("../../../data")
	if err != nil {
		t.Fatalf("Failed to load technologies: %v", err)
	}

	missions := loader.LoadMissions()

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack:     30,
		models.Quarry:         30,
		models.OreMine:        30,
		models.Farm:           30,
		models.WoodStore:      20,
		models.StoneStore:     20,
		models.OreStore:       20,
		models.Keep:           20,
		models.Arsenal:        30,
		models.Library:        10,
		models.Tavern:         10,
		models.Market:         20,
		models.Fortifications: 20,
	}

	solver := NewTestSolver(buildings, techs, missions, targetLevels)
	initialState := &models.GameState{
		BuildingLevels:         map[models.BuildingType]int{},
		Resources:              map[models.ResourceType]float64{},
		ResearchedTechnologies: map[string]bool{},
		StorageCaps:            map[models.ResourceType]int{},
		ProductionRates:        map[models.ResourceType]float64{},
	}

	solution := solver.Solve(initialState)

	// Create a deterministic representation of the build order
	type Action struct {
		Step        int
		Type        string
		Name        string
		FromLevel   int
		ToLevel     int
		StartTime   int
		EndTime     int
		WoodCost    int
		StoneCost   int
		IronCost    int
		FoodCost    int
		FoodUsed    int
		FoodCapacity int
	}

	var actions []Action
	step := 1

	// Add building actions
	for _, ba := range solution.BuildingActions {
		actions = append(actions, Action{
			Step:        step,
			Type:        "Building",
			Name:        string(ba.BuildingType),
			FromLevel:   ba.FromLevel,
			ToLevel:     ba.ToLevel,
			StartTime:   ba.StartTime,
			EndTime:     ba.EndTime,
			WoodCost:    ba.Costs.Wood,
			StoneCost:   ba.Costs.Stone,
			IronCost:    ba.Costs.Iron,
			FoodCost:    ba.Costs.Food,
			FoodUsed:    ba.FoodUsed,
			FoodCapacity: ba.FoodCapacity,
		})
		step++
	}

	// Add research actions
	for _, ra := range solution.ResearchActions {
		actions = append(actions, Action{
			Step:        step,
			Type:        "Research",
			Name:        ra.TechnologyName,
			StartTime:   ra.StartTime,
			EndTime:     ra.EndTime,
			WoodCost:    ra.Costs.Wood,
			StoneCost:   ra.Costs.Stone,
			IronCost:    ra.Costs.Iron,
			FoodCost:    ra.Costs.Food,
			FoodUsed:    ra.FoodUsed,
			FoodCapacity: ra.FoodCapacity,
		})
		step++
	}

	// Sort by start time for deterministic order
	// (Note: we don't sort here to preserve the order as returned by solver)

	// Serialize to JSON
	jsonBytes, err := json.Marshal(actions)
	if err != nil {
		t.Fatalf("Failed to marshal actions: %v", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(jsonBytes)
	actualHash := hex.EncodeToString(hash[:])

	// Golden hash - UPDATE THIS if you intentionally change the algorithm
	// Generated from commit 82ff648 (with scarcity cache optimization)
	goldenHash := "2da1aa610e2740541e1d286a7d831e06e5896d3a4d346edcb40432cde0e24878"

	if actualHash != goldenHash {
		t.Errorf("Build order changed!\nExpected hash: %s\nActual hash:   %s\n\nIf this change is intentional, update goldenHash in the test.", goldenHash, actualHash)
		t.Logf("Total building actions: %d", len(solution.BuildingActions))
		t.Logf("Total research actions: %d", len(solution.ResearchActions))
		t.Logf("Total time: %d seconds", solution.TotalTimeSeconds)
		
		// Show first 10 actions for debugging
		t.Log("First 10 actions:")
		for i := 0; i < 10 && i < len(actions); i++ {
			t.Logf("  %d. %s: %s (Start: %d)", actions[i].Step, actions[i].Type, actions[i].Name, actions[i].StartTime)
		}
	}
}
