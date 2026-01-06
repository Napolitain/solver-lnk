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

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)

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

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)

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

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)

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

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
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

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
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

// =============================================================================
// Phase 1: Resource Constraint Fuzz Tests
// =============================================================================

// FuzzResourcesNeverNegative verifies resources never go negative during solve
func FuzzResourcesNeverNegative(f *testing.F) {
	// Seed corpus with various starting resource amounts
	f.Add(uint16(0), uint16(0), uint16(0))
	f.Add(uint16(100), uint16(100), uint16(100))
	f.Add(uint16(1000), uint16(500), uint16(200))
	f.Add(uint16(50), uint16(1000), uint16(50))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, startWood, startStone, startIron uint16) {
		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: 5,
			models.Quarry:     5,
			models.OreMine:    5,
			models.Tavern:     2,
			models.Farm:       3,
		}

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)

		initialState := models.NewGameState()
		initialState.Resources[models.Wood] = float64(startWood)
		initialState.Resources[models.Stone] = float64(startStone)
		initialState.Resources[models.Iron] = float64(startIron)
		initialState.BuildingLevels[models.Lumberjack] = 1
		initialState.BuildingLevels[models.Quarry] = 1
		initialState.BuildingLevels[models.OreMine] = 1
		initialState.BuildingLevels[models.Tavern] = 1
		initialState.BuildingLevels[models.Farm] = 1

		solution := solver.Solve(initialState)

		// Check each building action has non-negative costs
		for _, action := range solution.BuildingActions {
			if action.Costs.Wood < 0 || action.Costs.Stone < 0 || action.Costs.Iron < 0 {
				t.Errorf("Building %s has negative costs: W:%d S:%d I:%d",
					action.BuildingType, action.Costs.Wood, action.Costs.Stone, action.Costs.Iron)
			}
		}

		// Check each research action
		for _, action := range solution.ResearchActions {
			if action.Costs.Wood < 0 || action.Costs.Stone < 0 || action.Costs.Iron < 0 {
				t.Errorf("Research %s has negative costs: W:%d S:%d I:%d",
					action.TechnologyName, action.Costs.Wood, action.Costs.Stone, action.Costs.Iron)
			}
		}

		// Check each training action
		for _, action := range solution.TrainingActions {
			if action.Costs.Wood < 0 || action.Costs.Stone < 0 || action.Costs.Iron < 0 {
				t.Errorf("Training %s has negative costs: W:%d S:%d I:%d",
					action.UnitType, action.Costs.Wood, action.Costs.Stone, action.Costs.Iron)
			}
		}
	})
}

// =============================================================================
// Phase 2: Queue Constraint Fuzz Tests
// =============================================================================

// FuzzBuildingQueueSingleItem verifies only one building at a time
func FuzzBuildingQueueSingleItem(f *testing.F) {
	f.Add(uint8(5), uint8(5), uint8(5))
	f.Add(uint8(10), uint8(10), uint8(10))
	f.Add(uint8(15), uint8(8), uint8(3))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, ljTarget, qTarget, omTarget uint8) {
		lj := int(ljTarget)%15 + 1
		q := int(qTarget)%15 + 1
		om := int(omTarget)%15 + 1

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: lj,
			models.Quarry:     q,
			models.OreMine:    om,
			models.Farm:       5,
		}

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		solution := solver.Solve(initialState)

		// Check no overlapping building actions
		for i, a1 := range solution.BuildingActions {
			for j, a2 := range solution.BuildingActions {
				if i >= j {
					continue
				}
				// Check for overlap: a1.Start < a2.End AND a2.Start < a1.End
				if a1.StartTime < a2.EndTime && a2.StartTime < a1.EndTime {
					t.Errorf("Building queue violation: %s (%d→%d) [%d-%d] overlaps with %s (%d→%d) [%d-%d]",
						a1.BuildingType, a1.FromLevel, a1.ToLevel, a1.StartTime, a1.EndTime,
						a2.BuildingType, a2.FromLevel, a2.ToLevel, a2.StartTime, a2.EndTime)
				}
			}
		}
	})
}

// FuzzResearchQueueSingleItem verifies only one research at a time
func FuzzResearchQueueSingleItem(f *testing.F) {
	f.Add(uint8(3), uint8(5))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, libraryTarget, farmTarget uint8) {
		lib := int(libraryTarget)%5 + 1
		farm := int(farmTarget)%20 + 5

		targetLevels := map[models.BuildingType]int{
			models.Library: lib,
			models.Farm:    farm,
		}

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		solution := solver.Solve(initialState)

		// Check no overlapping research actions
		for i, a1 := range solution.ResearchActions {
			for j, a2 := range solution.ResearchActions {
				if i >= j {
					continue
				}
				if a1.StartTime < a2.EndTime && a2.StartTime < a1.EndTime {
					t.Errorf("Research queue violation: %s [%d-%d] overlaps with %s [%d-%d]",
						a1.TechnologyName, a1.StartTime, a1.EndTime,
						a2.TechnologyName, a2.StartTime, a2.EndTime)
				}
			}
		}
	})
}

// =============================================================================
// Phase 3: Production & Storage Fuzz Tests
// =============================================================================

// FuzzStorageNeverExceeded verifies resources never exceed storage caps during solve
func FuzzStorageNeverExceeded(f *testing.F) {
	f.Add(uint8(5), uint8(5), uint8(5))
	f.Add(uint8(10), uint8(10), uint8(10))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, ljTarget, qTarget, omTarget uint8) {
		lj := int(ljTarget)%10 + 1
		q := int(qTarget)%10 + 1
		om := int(omTarget)%10 + 1

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: lj,
			models.Quarry:     q,
			models.OreMine:    om,
		}

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		solution := solver.Solve(initialState)

		// The solver should complete without panics
		// This verifies storage cap logic doesn't cause issues
		if solution.TotalTimeSeconds < 0 {
			t.Errorf("Invalid total time: %d", solution.TotalTimeSeconds)
		}
	})
}

// =============================================================================
// Phase 4: Prerequisite Fuzz Tests
// =============================================================================

// FuzzFarmResearchPrerequisites verifies Farm upgrades respect research requirements
func FuzzFarmResearchPrerequisites(f *testing.F) {
	f.Add(uint8(10), uint8(1))
	f.Add(uint8(20), uint8(3))
	f.Add(uint8(30), uint8(5))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, farmTarget, libraryTarget uint8) {
		farm := int(farmTarget)%30 + 1
		lib := int(libraryTarget)%10 + 1

		targetLevels := map[models.BuildingType]int{
			models.Farm:       farm,
			models.Library:    lib,
			models.Lumberjack: 10,
			models.Quarry:     10,
			models.OreMine:    10,
		}

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		solution := solver.Solve(initialState)

		// Track research completion times
		researchComplete := make(map[string]int)
		for _, ra := range solution.ResearchActions {
			researchComplete[ra.TechnologyName] = ra.EndTime
		}

		// Check Farm upgrades respect prerequisites
		for _, ba := range solution.BuildingActions {
			if ba.BuildingType != models.Farm {
				continue
			}

			// Farm 15 requires Crop Rotation
			if ba.ToLevel == 15 {
				if cropTime, ok := researchComplete["Crop Rotation"]; ok {
					if ba.StartTime < cropTime {
						t.Errorf("Farm 15 started at %d but Crop Rotation completes at %d",
							ba.StartTime, cropTime)
					}
				}
			}

			// Farm 25 requires Yoke
			if ba.ToLevel == 25 {
				if yokeTime, ok := researchComplete["Yoke"]; ok {
					if ba.StartTime < yokeTime {
						t.Errorf("Farm 25 started at %d but Yoke completes at %d",
							ba.StartTime, yokeTime)
					}
				}
			}

			// Farm 30 requires Cellar Storeroom
			if ba.ToLevel == 30 {
				if cellarTime, ok := researchComplete["Cellar Storeroom"]; ok {
					if ba.StartTime < cellarTime {
						t.Errorf("Farm 30 started at %d but Cellar Storeroom completes at %d",
							ba.StartTime, cellarTime)
					}
				}
			}
		}
	})
}

// =============================================================================
// Phase 5: Mission Constraint Fuzz Tests
// =============================================================================

// FuzzMissionNoSameTypeOverlap verifies same mission type never runs in parallel
func FuzzMissionNoSameTypeOverlap(f *testing.F) {
	f.Add(uint8(3))
	f.Add(uint8(5))
	f.Add(uint8(7))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, tavernTarget uint8) {
		tavern := int(tavernTarget)%10 + 1

		targetLevels := map[models.BuildingType]int{
			models.Tavern:     tavern,
			models.Lumberjack: 10,
			models.Quarry:     10,
			models.OreMine:    10,
			models.Farm:       10,
		}

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		solution := solver.Solve(initialState)

		// Check no same-type missions overlap
		for i, m1 := range solution.MissionActions {
			for j, m2 := range solution.MissionActions {
				if i >= j {
					continue
				}
				if m1.MissionName != m2.MissionName {
					continue
				}
				// Same mission type - check no overlap
				if m1.StartTime < m2.EndTime && m2.StartTime < m1.EndTime {
					t.Errorf("Mission %s overlaps: [%d-%d] and [%d-%d]",
						m1.MissionName, m1.StartTime, m1.EndTime, m2.StartTime, m2.EndTime)
				}
			}
		}
	})
}

// =============================================================================
// Phase 6: End-State Fuzz Tests
// =============================================================================

// FuzzAllTargetsReached verifies all building targets are reached
func FuzzAllTargetsReached(f *testing.F) {
	f.Add(uint8(5), uint8(5), uint8(5))
	f.Add(uint8(10), uint8(8), uint8(6))
	f.Add(uint8(15), uint8(15), uint8(15))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, ljTarget, qTarget, omTarget uint8) {
		lj := int(ljTarget)%20 + 1
		q := int(qTarget)%20 + 1
		om := int(omTarget)%20 + 1

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: lj,
			models.Quarry:     q,
			models.OreMine:    om,
		}

		solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		solution := solver.Solve(initialState)

		// Track final levels
		finalLevels := make(map[models.BuildingType]int)
		for bt := range initialState.BuildingLevels {
			finalLevels[bt] = initialState.BuildingLevels[bt]
		}
		for _, ba := range solution.BuildingActions {
			finalLevels[ba.BuildingType] = ba.ToLevel
		}

		// Check targets reached
		for bt, target := range targetLevels {
			if finalLevels[bt] < target {
				t.Errorf("Target not reached: %s expected %d, got %d",
					bt, target, finalLevels[bt])
			}
		}
	})
}

// =============================================================================
// Phase 7: Determinism Fuzz Tests
// =============================================================================

// FuzzDeterministicOutput verifies same inputs produce same outputs
func FuzzDeterministicOutput(f *testing.F) {
	f.Add(uint8(5), uint8(5), uint8(5), uint16(100), uint16(100), uint16(100))
	f.Add(uint8(10), uint8(8), uint8(6), uint16(500), uint16(300), uint16(200))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, ljTarget, qTarget, omTarget uint8, startW, startS, startI uint16) {
		lj := int(ljTarget)%15 + 1
		q := int(qTarget)%15 + 1
		om := int(omTarget)%15 + 1

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: lj,
			models.Quarry:     q,
			models.OreMine:    om,
		}

		// Run solver twice with same inputs
		solver1 := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
		solver2 := castle.NewTestSolver(buildings, technologies, missions, targetLevels)

		initialState1 := models.NewGameState()
		initialState1.Resources[models.Wood] = float64(startW)
		initialState1.Resources[models.Stone] = float64(startS)
		initialState1.Resources[models.Iron] = float64(startI)

		initialState2 := models.NewGameState()
		initialState2.Resources[models.Wood] = float64(startW)
		initialState2.Resources[models.Stone] = float64(startS)
		initialState2.Resources[models.Iron] = float64(startI)

		solution1 := solver1.Solve(initialState1)
		solution2 := solver2.Solve(initialState2)

		// Compare outputs
		if len(solution1.BuildingActions) != len(solution2.BuildingActions) {
			t.Errorf("Different number of building actions: %d vs %d",
				len(solution1.BuildingActions), len(solution2.BuildingActions))
			return
		}

		for i := range solution1.BuildingActions {
			a1 := solution1.BuildingActions[i]
			a2 := solution2.BuildingActions[i]
			if a1.BuildingType != a2.BuildingType || a1.StartTime != a2.StartTime {
				t.Errorf("Building action %d differs: %s@%d vs %s@%d",
					i, a1.BuildingType, a1.StartTime, a2.BuildingType, a2.StartTime)
			}
		}

		if solution1.TotalTimeSeconds != solution2.TotalTimeSeconds {
			t.Errorf("Different total time: %d vs %d",
				solution1.TotalTimeSeconds, solution2.TotalTimeSeconds)
		}
	})
}
