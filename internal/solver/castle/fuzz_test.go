package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

// FuzzROICalculation fuzzes the ROI calculation with various production rates
func FuzzROICalculation(f *testing.F) {
	// Seed corpus
	f.Add(uint8(10), uint8(10), uint8(5))
	f.Add(uint8(100), uint8(50), uint8(25))
	f.Add(uint8(1), uint8(1), uint8(1))
	f.Add(uint8(255), uint8(255), uint8(255))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Quarry:     10,
		models.OreMine:    10,
	}

	solver := NewSolver(buildings, technologies, missions, targetLevels)

	f.Fuzz(func(t *testing.T, woodRate, stoneRate, ironRate uint8) {
		state := NewState(models.NewGameState())

		// Set production rates (min 0.1 to avoid division by zero)
		state.SetProductionRate(models.Wood, float64(woodRate)+0.1)
		state.SetProductionRate(models.Stone, float64(stoneRate)+0.1)
		state.SetProductionRate(models.Iron, float64(ironRate)+0.1)
		state.StorageCaps = [3]int{10000, 10000, 10000}

		// Test ROI for Lumberjack level 2
		lj := buildings[models.Lumberjack]
		if lj == nil {
			return
		}
		levelData := lj.GetLevelData(2)
		if levelData == nil {
			return
		}

		action := &BuildingAction{
			BuildingType: models.Lumberjack,
			FromLevel:    1,
			ToLevel:      2,
			Building:     lj,
			LevelData:    levelData,
		}

		roi := solver.buildingROI(state, action)

		// Property: ROI should be finite and non-negative for production buildings
		if roi < 0 {
			t.Errorf("ROI should be non-negative, got %f", roi)
		}
		if roi != roi { // NaN check
			t.Errorf("ROI should not be NaN")
		}
	})
}

// FuzzDynamicScarcity fuzzes the scarcity calculation
func FuzzDynamicScarcity(f *testing.F) {
	// Seed corpus
	f.Add(uint8(10), uint8(10), uint8(10))
	f.Add(uint8(100), uint8(1), uint8(1))
	f.Add(uint8(1), uint8(100), uint8(100))
	f.Add(uint8(50), uint8(50), uint8(50))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Quarry:     10,
		models.OreMine:    10,
	}

	solver := NewSolver(buildings, technologies, missions, targetLevels)

	f.Fuzz(func(t *testing.T, woodRate, stoneRate, ironRate uint8) {
		state := NewState(models.NewGameState())

		// Set production rates (use max to ensure > 0)
		state.SetProductionRate(models.Wood, float64(woodRate)+0.1)
		state.SetProductionRate(models.Stone, float64(stoneRate)+0.1)
		state.SetProductionRate(models.Iron, float64(ironRate)+0.1)

		// Test scarcity for each building type
		for _, bt := range []models.BuildingType{models.Lumberjack, models.Quarry, models.OreMine} {
			scarcity := solver.calculateDynamicScarcity(state, bt)

			// Property: Scarcity must be bounded [0.5, 2.0]
			if scarcity < 0.5 || scarcity > 2.0 {
				t.Errorf("Scarcity for %s out of bounds: %f", bt, scarcity)
			}

			// Property: Scarcity must be finite
			if scarcity != scarcity { // NaN check
				t.Errorf("Scarcity for %s is NaN", bt)
			}
		}
	})
}

// FuzzMissionSelection fuzzes mission selection with various army sizes
func FuzzMissionSelection(f *testing.F) {
	// Seed corpus
	f.Add(uint8(1), uint16(0), uint16(0), uint16(0), uint16(0))
	f.Add(uint8(5), uint16(100), uint16(100), uint16(100), uint16(100))
	f.Add(uint8(10), uint16(500), uint16(500), uint16(500), uint16(500))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Tavern: 10,
	}

	solver := NewSolver(buildings, technologies, missions, targetLevels)

	f.Fuzz(func(t *testing.T, tavernLevel uint8, spearmen, archers, horsemen, crossbowmen uint16) {
		state := NewState(models.NewGameState())

		// Cap tavern level to valid range
		level := int(tavernLevel)
		if level < 1 {
			level = 1
		}
		if level > 10 {
			level = 10
		}
		state.SetBuildingLevel(models.Tavern, level)

		// Set army
		state.Army.Spearman = int(spearmen)
		state.Army.Archer = int(archers)
		state.Army.Horseman = int(horsemen)
		state.Army.Crossbowman = int(crossbowmen)

		// Pick a mission
		mission := solver.pickBestMissionToStart(state)

		if mission != nil {
			// Property: Selected mission must be within tavern level bounds
			if mission.TavernLevel > level {
				t.Errorf("Selected mission %s requires tavern %d but we have %d",
					mission.Name, mission.TavernLevel, level)
			}
			if mission.MaxTavernLevel > 0 && level > mission.MaxTavernLevel {
				t.Errorf("Selected mission %s max tavern %d but we have %d",
					mission.Name, mission.MaxTavernLevel, level)
			}

			// Property: Must have enough units
			for _, req := range mission.UnitsRequired {
				have := state.Army.Get(req.Type) - state.UnitsOnMission.Get(req.Type)
				if have < req.Count {
					t.Errorf("Selected mission %s needs %d %s but only have %d",
						mission.Name, req.Count, req.Type, have)
				}
			}
		}
	})
}

// FuzzSolverMissionNoOverlap fuzzes solver to verify no mission overlaps
func FuzzSolverMissionNoOverlap(f *testing.F) {
	// Seed corpus with different target configurations
	f.Add(uint8(5), uint8(5), uint8(5), uint8(3))
	f.Add(uint8(10), uint8(10), uint8(10), uint8(5))
	f.Add(uint8(3), uint8(3), uint8(3), uint8(2))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, ljTarget, qTarget, omTarget, tavernTarget uint8) {
		// Cap targets to reasonable values
		lj := int(ljTarget)
		if lj < 1 {
			lj = 1
		}
		if lj > 15 {
			lj = 15
		}
		q := int(qTarget)
		if q < 1 {
			q = 1
		}
		if q > 15 {
			q = 15
		}
		om := int(omTarget)
		if om < 1 {
			om = 1
		}
		if om > 15 {
			om = 15
		}
		tavern := int(tavernTarget)
		if tavern < 1 {
			tavern = 1
		}
		if tavern > 5 {
			tavern = 5
		}

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: lj,
			models.Quarry:     q,
			models.OreMine:    om,
			models.Tavern:     tavern,
			models.Farm:       5,
		}

		solver := NewSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		initialState.BuildingLevels[models.Lumberjack] = 1
		initialState.BuildingLevels[models.Quarry] = 1
		initialState.BuildingLevels[models.OreMine] = 1
		initialState.BuildingLevels[models.Tavern] = 1
		initialState.BuildingLevels[models.Farm] = 1

		solution := solver.Solve(initialState)

		// Property: No duplicate missions running at the same time
		for i, m1 := range solution.MissionActions {
			for j, m2 := range solution.MissionActions {
				if i >= j {
					continue
				}
				if m1.MissionName != m2.MissionName {
					continue
				}

				// Check overlap
				if m1.StartTime < m2.EndTime && m2.StartTime < m1.EndTime {
					t.Errorf("Duplicate parallel mission: %s at %d-%d and %d-%d",
						m1.MissionName, m1.StartTime, m1.EndTime, m2.StartTime, m2.EndTime)
				}
			}
		}
	})
}

// FuzzROINonNegative ensures ROI is never negative
func FuzzROINonNegative(f *testing.F) {
	f.Add(uint8(1), uint8(5), uint8(10), uint8(20))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, ljLevel, qLevel, omLevel, farmLevel uint8) {
		// Cap levels
		lj := int(ljLevel) % 30
		if lj < 1 {
			lj = 1
		}
		q := int(qLevel) % 30
		if q < 1 {
			q = 1
		}
		om := int(omLevel) % 30
		if om < 1 {
			om = 1
		}
		farm := int(farmLevel) % 30
		if farm < 1 {
			farm = 1
		}

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: 30,
			models.Quarry:     30,
			models.OreMine:    30,
		}

		solver := NewSolver(buildings, technologies, missions, targetLevels)
		state := NewState(models.NewGameState())
		state.SetBuildingLevel(models.Lumberjack, lj)
		state.SetBuildingLevel(models.Quarry, q)
		state.SetBuildingLevel(models.OreMine, om)
		state.SetBuildingLevel(models.Farm, farm)

		// Set production based on levels
		state.SetProductionRate(models.Wood, float64(lj*5))
		state.SetProductionRate(models.Stone, float64(q*5))
		state.SetProductionRate(models.Iron, float64(om*3))
		state.StorageCaps = [3]int{10000, 10000, 10000}

		// Test ROI for all production buildings
		for _, bt := range []models.BuildingType{models.Lumberjack, models.Quarry, models.OreMine} {
			building := buildings[bt]
			if building == nil {
				continue
			}

			currentLevel := state.GetBuildingLevel(bt)
			if currentLevel >= 30 {
				continue
			}

			levelData := building.GetLevelData(currentLevel + 1)
			if levelData == nil {
				continue
			}

			action := &BuildingAction{
				BuildingType: bt,
				FromLevel:    currentLevel,
				ToLevel:      currentLevel + 1,
				Building:     building,
				LevelData:    levelData,
			}

			roi := solver.buildingROI(state, action)

			// Property: ROI must be non-negative
			if roi < 0 {
				t.Errorf("Negative ROI for %s %d→%d: %f", bt, currentLevel, currentLevel+1, roi)
			}
		}
	})
}
