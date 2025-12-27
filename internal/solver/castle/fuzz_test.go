package castle_test

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	"github.com/napolitain/solver-lnk/internal/solver/castle"
)

func FuzzSolverResources(f *testing.F) {
	// Add seed corpus
	f.Add(int32(120), int32(120), int32(120), int32(40))
	f.Add(int32(0), int32(0), int32(0), int32(0))
	f.Add(int32(10000), int32(10000), int32(10000), int32(1000))
	f.Add(int32(1), int32(1), int32(1), int32(1))

	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		f.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		f.Fatalf("Failed to load technologies: %v", err)
	}

	f.Fuzz(func(t *testing.T, wood, stone, iron, food int32) {
		// Skip negative values
		if wood < 0 || stone < 0 || iron < 0 || food < 0 {
			return
		}

		// Cap at reasonable values to avoid extremely long runs
		if wood > 50000 || stone > 50000 || iron > 50000 || food > 5000 {
			return
		}

		initialState := models.NewGameState()
		initialState.Resources[models.Wood] = float64(wood)
		initialState.Resources[models.Stone] = float64(stone)
		initialState.Resources[models.Iron] = float64(iron)
		initialState.Resources[models.Food] = float64(food)

		for _, bt := range models.AllBuildingTypes() {
			initialState.BuildingLevels[bt] = 1
		}

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: 5,
			models.Quarry:     5,
		}

		s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
		solution := s.Solve()

		// Invariants that must hold
		if solution == nil {
			t.Error("Solution should never be nil")
			return
		}

		if solution.TotalTimeSeconds < 0 {
			t.Errorf("Time should never be negative: %d", solution.TotalTimeSeconds)
		}

		// All actions should have valid times
		for i, action := range solution.BuildingActions {
			if action.StartTime < 0 {
				t.Errorf("Action %d has negative start time: %d", i, action.StartTime)
			}
			if action.EndTime < action.StartTime {
				t.Errorf("Action %d end time before start: %d < %d", i, action.EndTime, action.StartTime)
			}
		}

		// Final state should reach targets
		if solution.FinalState.BuildingLevels[models.Lumberjack] < 5 {
			t.Errorf("Lumberjack should reach target 5, got %d", solution.FinalState.BuildingLevels[models.Lumberjack])
		}
		if solution.FinalState.BuildingLevels[models.Quarry] < 5 {
			t.Errorf("Quarry should reach target 5, got %d", solution.FinalState.BuildingLevels[models.Quarry])
		}
	})
}

func FuzzSolverBuildingLevels(f *testing.F) {
	// Add seed corpus
	f.Add(int32(1), int32(1), int32(1), int32(1))
	f.Add(int32(5), int32(5), int32(5), int32(5))
	f.Add(int32(10), int32(10), int32(10), int32(10))

	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		f.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		f.Fatalf("Failed to load technologies: %v", err)
	}

	f.Fuzz(func(t *testing.T, lumberLevel, quarryLevel, farmLevel, keepLevel int32) {
		// Clamp to valid range
		clamp := func(v int32) int {
			if v < 1 {
				return 1
			}
			if v > 30 {
				return 30
			}
			return int(v)
		}

		initialState := models.NewGameState()
		initialState.Resources[models.Wood] = 10000
		initialState.Resources[models.Stone] = 10000
		initialState.Resources[models.Iron] = 10000
		initialState.Resources[models.Food] = 1000

		initialState.BuildingLevels[models.Lumberjack] = clamp(lumberLevel)
		initialState.BuildingLevels[models.Quarry] = clamp(quarryLevel)
		initialState.BuildingLevels[models.Farm] = clamp(farmLevel)
		initialState.BuildingLevels[models.Keep] = clamp(keepLevel)

		// Set other buildings to 1
		for _, bt := range models.AllBuildingTypes() {
			if _, ok := initialState.BuildingLevels[bt]; !ok {
				initialState.BuildingLevels[bt] = 1
			}
		}

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: 10,
			models.Quarry:     10,
		}

		s := castle.NewGreedySolver(buildings, technologies, initialState, targetLevels)
		solution := s.Solve()

		// Should never panic and always return valid solution
		if solution == nil {
			t.Error("Solution should never be nil")
			return
		}

		// All actions should have valid structure
		for i, action := range solution.BuildingActions {
			if action.EndTime < action.StartTime {
				t.Errorf("Action %d: end %d < start %d", i, action.EndTime, action.StartTime)
			}
			if action.ToLevel <= action.FromLevel {
				t.Errorf("Action %d: toLevel %d <= fromLevel %d", i, action.ToLevel, action.FromLevel)
			}
		}
	})
}
