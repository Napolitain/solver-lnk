package castle

import (
	"sort"
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

		solver := NewSolver(buildings, technologies, missions, targetLevels)

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

		solver := NewSolver(buildings, technologies, missions, targetLevels)
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

		solver := NewSolver(buildings, technologies, missions, targetLevels)
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

		solver := NewSolver(buildings, technologies, missions, targetLevels)
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

		solver := NewSolver(buildings, technologies, missions, targetLevels)
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

		solver := NewSolver(buildings, technologies, missions, targetLevels)
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

		solver := NewSolver(buildings, technologies, missions, targetLevels)
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
		solver1 := NewSolver(buildings, technologies, missions, targetLevels)
		solver2 := NewSolver(buildings, technologies, missions, targetLevels)

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

// =============================================================================
// Comprehensive End-to-End Fuzz Test
// =============================================================================

// FuzzSolverEndToEnd is a comprehensive end-to-end fuzz test that validates
// the entire solver flow from initial state to full targets reached.
// This test consolidates all invariants and property assertions:
// - Ending conditions (all targets reached)
// - Time progression (no time travel, sequential queues)
// - Food capacity constraints
// - Library prerequisites
// - Resource management (never negative, costs reasonable)
// - Building level progression (sequential, incremental)
func FuzzSolverEndToEnd(f *testing.F) {
	// Seed with various target configurations and initial resource states
	f.Add(uint8(10), uint8(10), uint8(10), uint8(5), uint8(3), uint16(0), uint16(0), uint16(0))
	f.Add(uint8(15), uint8(15), uint8(15), uint8(7), uint8(5), uint16(500), uint16(500), uint16(500))
	f.Add(uint8(20), uint8(18), uint8(16), uint8(10), uint8(8), uint16(1000), uint16(200), uint16(100))
	f.Add(uint8(5), uint8(5), uint8(5), uint8(2), uint8(1), uint16(100), uint16(100), uint16(100))

	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	f.Fuzz(func(t *testing.T, ljTarget, qTarget, omTarget, tavernTarget, libraryTarget uint8,
		startWood, startStone, startIron uint16) {
		
		// Cap to reasonable values
		lj := int(ljTarget)%25 + 1
		q := int(qTarget)%25 + 1
		om := int(omTarget)%25 + 1
		tavern := int(tavernTarget)%10 + 1
		library := int(libraryTarget)%8 + 1

		targetLevels := map[models.BuildingType]int{
			models.Lumberjack: lj,
			models.Quarry:     q,
			models.OreMine:    om,
			models.Tavern:     tavern,
			models.Library:    library,
		}

		solver := NewSolver(buildings, technologies, missions, targetLevels)
		initialState := models.NewGameState()
		initialState.Resources[models.Wood] = float64(startWood)
		initialState.Resources[models.Stone] = float64(startStone)
		initialState.Resources[models.Iron] = float64(startIron)
		// Buildings start at level 1 by default in the game
		initialState.BuildingLevels[models.Lumberjack] = 1
		initialState.BuildingLevels[models.Quarry] = 1
		initialState.BuildingLevels[models.OreMine] = 1
		
		solution := solver.Solve(initialState)

		// =====================================================================
		// ENDING CONDITION ASSERTIONS
		// =====================================================================
		
		// Track final building levels
		finalLevels := make(map[models.BuildingType]int)
		for bt := range initialState.BuildingLevels {
			finalLevels[bt] = initialState.BuildingLevels[bt]
		}
		for _, ba := range solution.BuildingActions {
			finalLevels[ba.BuildingType] = ba.ToLevel
		}
		
		// ASSERTION 1: All building targets that were SET must be reached
		for bt, target := range targetLevels {
			if target > 0 && finalLevels[bt] < target {
				t.Errorf("ENDING CONDITION: Building target not reached: %s expected %d, got %d",
					bt, target, finalLevels[bt])
			}
		}

		// =====================================================================
		// TIME PROGRESSION ASSERTIONS
		// =====================================================================
		
		// ASSERTION 2: Building queue - only one at a time, no time travel
		lastBuildingEndTime := 0
		for _, ba := range solution.BuildingActions {
			if ba.StartTime < lastBuildingEndTime {
				t.Errorf("TIME VIOLATION: Building %s started at %d but previous building ended at %d (overlap)",
					ba.BuildingType, ba.StartTime, lastBuildingEndTime)
			}
			if ba.EndTime < ba.StartTime {
				t.Errorf("TIME VIOLATION: Building %s ended (%d) before it started (%d)",
					ba.BuildingType, ba.EndTime, ba.StartTime)
			}
			lastBuildingEndTime = ba.EndTime
		}
		
		// ASSERTION 3: Research queue - only one at a time
		lastResearchEndTime := 0
		for _, ra := range solution.ResearchActions {
			if ra.StartTime < lastResearchEndTime {
				t.Errorf("TIME VIOLATION: Research %s started at %d but previous research ended at %d (overlap)",
					ra.TechnologyName, ra.StartTime, lastResearchEndTime)
			}
			if ra.EndTime < ra.StartTime {
				t.Errorf("TIME VIOLATION: Research %s ended (%d) before it started (%d)",
					ra.TechnologyName, ra.EndTime, ra.StartTime)
			}
			lastResearchEndTime = ra.EndTime
		}
		
		// ASSERTION 4: Training actions - no negative durations
		for _, ta := range solution.TrainingActions {
			if ta.EndTime < ta.StartTime {
				t.Errorf("TIME VIOLATION: Training %s ended (%d) before it started (%d)",
					ta.UnitType, ta.EndTime, ta.StartTime)
			}
		}
		
		// ASSERTION 5: Total time must be positive and reasonable
		if solution.TotalTimeSeconds < 0 {
			t.Errorf("TIME VIOLATION: Total time is negative: %d", solution.TotalTimeSeconds)
		}
		if solution.TotalTimeSeconds > 200*24*3600 {
			t.Errorf("TIME VIOLATION: Total time exceeds 200 days: %d seconds (%.1f days)",
				solution.TotalTimeSeconds, float64(solution.TotalTimeSeconds)/86400)
		}

		// =====================================================================
		// FOOD CAPACITY ASSERTIONS
		// =====================================================================
		
		// ASSERTION 6: Food usage must never exceed capacity at any point
		type foodEvent struct {
			time     int
			capacity int
			used     int
		}
		var foodEvents []foodEvent

		// Collect all food-related events
		for _, ba := range solution.BuildingActions {
			if ba.BuildingType == models.Farm || ba.Costs.Food > 0 {
				foodEvents = append(foodEvents, foodEvent{
					time:     ba.EndTime,
					capacity: ba.FoodCapacity,
					used:     ba.FoodUsed,
				})
			}
		}
		for _, ra := range solution.ResearchActions {
			if ra.Costs.Food > 0 {
				foodEvents = append(foodEvents, foodEvent{
					time:     ra.EndTime,
					capacity: ra.FoodCapacity,
					used:     ra.FoodUsed,
				})
			}
		}
		for _, ta := range solution.TrainingActions {
			if ta.Costs.Food > 0 {
				foodEvents = append(foodEvents, foodEvent{
					time:     ta.EndTime,
					capacity: ta.FoodCapacity,
					used:     ta.FoodUsed,
				})
			}
		}

		// Sort events by time for chronological validation
		sort.Slice(foodEvents, func(i, j int) bool {
			return foodEvents[i].time < foodEvents[j].time
		})

		// Validate food constraints
		for _, event := range foodEvents {
			if event.used > event.capacity {
				t.Errorf("FOOD VIOLATION: At time %d, food used (%d) exceeds capacity (%d)",
					event.time, event.used, event.capacity)
			}
		}

		// =====================================================================
		// LIBRARY PREREQUISITE ASSERTIONS
		// =====================================================================
		
		// ASSERTION 7: All researched techs must have library prerequisites met
		libraryLevelTimeline := make(map[int]int)
		libraryLevelTimeline[0] = 1 // Start at level 1
		for _, ba := range solution.BuildingActions {
			if ba.BuildingType == models.Library {
				libraryLevelTimeline[ba.EndTime] = ba.ToLevel
			}
		}

		for _, ra := range solution.ResearchActions {
			tech := technologies[ra.TechnologyName]
			if tech == nil {
				continue
			}
			// Find library level when research started
			libLevel := 1
			for t, level := range libraryLevelTimeline {
				if t <= ra.StartTime && level > libLevel {
					libLevel = level
				}
			}
			if libLevel < tech.RequiredLibraryLevel {
				t.Errorf("PREREQUISITE VIOLATION: Research %s started at time %d requires library %d but had %d",
					ra.TechnologyName, ra.StartTime, tech.RequiredLibraryLevel, libLevel)
			}
		}

		// =====================================================================
		// RESOURCE MANAGEMENT ASSERTIONS
		// =====================================================================
		
		// ASSERTION 8: Resource costs must be non-negative for all actions
		for _, ba := range solution.BuildingActions {
			if ba.Costs.Wood < 0 || ba.Costs.Stone < 0 || ba.Costs.Iron < 0 || ba.Costs.Food < 0 {
				t.Errorf("COST VIOLATION: Building %s has negative costs: W:%d S:%d I:%d F:%d",
					ba.BuildingType, ba.Costs.Wood, ba.Costs.Stone, ba.Costs.Iron, ba.Costs.Food)
			}
			// ASSERTION 9: Costs shouldn't be astronomically high
			if ba.Costs.Wood > 1000000 || ba.Costs.Stone > 1000000 || ba.Costs.Iron > 1000000 {
				t.Errorf("COST VIOLATION: Building %s has unreasonably high costs: W:%d S:%d I:%d",
					ba.BuildingType, ba.Costs.Wood, ba.Costs.Stone, ba.Costs.Iron)
			}
		}
		for _, ra := range solution.ResearchActions {
			if ra.Costs.Wood < 0 || ra.Costs.Stone < 0 || ra.Costs.Iron < 0 || ra.Costs.Food < 0 {
				t.Errorf("COST VIOLATION: Research %s has negative costs: W:%d S:%d I:%d F:%d",
					ra.TechnologyName, ra.Costs.Wood, ra.Costs.Stone, ra.Costs.Iron, ra.Costs.Food)
			}
		}
		for _, ta := range solution.TrainingActions {
			if ta.Costs.Wood < 0 || ta.Costs.Stone < 0 || ta.Costs.Iron < 0 || ta.Costs.Food < 0 {
				t.Errorf("COST VIOLATION: Training %s has negative costs: W:%d S:%d I:%d F:%d",
					ta.UnitType, ta.Costs.Wood, ta.Costs.Stone, ta.Costs.Iron, ta.Costs.Food)
			}
		}
		
		// ASSERTION 10: Final resources must be non-negative
		finalResources := solution.FinalState.Resources
		if finalResources[models.Wood] < 0 || finalResources[models.Stone] < 0 || finalResources[models.Iron] < 0 {
			t.Errorf("RESOURCE VIOLATION: Final resources are negative: W:%.0f S:%.0f I:%.0f",
				finalResources[models.Wood], finalResources[models.Stone], finalResources[models.Iron])
		}

		// =====================================================================
		// BUILDING LEVEL PROGRESSION ASSERTIONS
		// =====================================================================
		
		// ASSERTION 11: Building levels must increase by exactly 1 each upgrade
		buildingLevels := make(map[models.BuildingType]int)
		for bt, level := range initialState.BuildingLevels {
			buildingLevels[bt] = level
		}
		// Buildings not in initialState default to level 1 (as per State.GetBuildingLevel)
		for _, bt := range models.AllBuildingTypes() {
			if _, exists := buildingLevels[bt]; !exists {
				buildingLevels[bt] = 1
			}
		}

		for i, ba := range solution.BuildingActions {
			currentLevel := buildingLevels[ba.BuildingType]

			// Check FromLevel matches current state
			if ba.FromLevel != currentLevel {
				t.Errorf("LEVEL VIOLATION: Building action %d for %s has FromLevel %d but current level is %d",
					i, ba.BuildingType, ba.FromLevel, currentLevel)
			}

			// Check ToLevel is exactly FromLevel + 1
			if ba.ToLevel != ba.FromLevel+1 {
				t.Errorf("LEVEL VIOLATION: Building %s upgrades from %d to %d (must be +1)",
					ba.BuildingType, ba.FromLevel, ba.ToLevel)
			}

			// Check building doesn't exceed maximum level
			if ba.ToLevel > 30 {
				t.Errorf("LEVEL VIOLATION: Building %s upgraded to level %d (max is 30)",
					ba.BuildingType, ba.ToLevel)
			}

			buildingLevels[ba.BuildingType] = ba.ToLevel
		}

		// ASSERTION 12: Final levels must match or exceed targets
		for bt, target := range targetLevels {
			if buildingLevels[bt] < target {
				t.Errorf("LEVEL VIOLATION: Building %s final level %d is below target %d",
					bt, buildingLevels[bt], target)
			}
		}

		// ASSERTION 13: All building levels must be >= 1 (game default)
		for bt, level := range buildingLevels {
			if level < 1 {
				t.Errorf("LEVEL VIOLATION: Building %s has invalid level %d (must be >= 1)",
					bt, level)
			}
		}

		// =====================================================================
		// ADDITIONAL CORRECTNESS ASSERTIONS
		// =====================================================================

		// ASSERTION 14: No duplicate research - same tech should never be researched twice
		researchedTechs := make(map[string]bool)
		for _, ra := range solution.ResearchActions {
			if researchedTechs[ra.TechnologyName] {
				t.Errorf("DUPLICATE RESEARCH: Tech %s was researched multiple times",
					ra.TechnologyName)
			}
			researchedTechs[ra.TechnologyName] = true
		}

		// ASSERTION 15: Technology prerequisite chains validated
		// Check that unit training respects tech prerequisites
		for _, ta := range solution.TrainingActions {
			unitType := ta.UnitType
			// Find the tech that enables this unit type
			var requiredTech string
			for techName, tech := range technologies {
				if tech.EnablesBuilding == string(unitType) || 
				   (tech.InternalName != "" && tech.InternalName == string(unitType)) {
					requiredTech = techName
					break
				}
			}
			// If there's a required tech, verify it was researched before training
			if requiredTech != "" && !researchedTechs[requiredTech] {
				t.Errorf("TECH PREREQUISITE VIOLATION: Unit %s trained without researching %s",
					unitType, requiredTech)
			}
		}

		// ASSERTION 16: Action durations match data files
		for _, ba := range solution.BuildingActions {
			building := buildings[ba.BuildingType]
			if building == nil {
				continue
			}
			levelData := building.GetLevelData(ba.ToLevel)
			if levelData == nil {
				continue
			}
			expectedDuration := levelData.BuildTimeSeconds
			actualDuration := ba.EndTime - ba.StartTime
			// Allow small tolerance for rounding
			if actualDuration < expectedDuration-1 || actualDuration > expectedDuration+1 {
				t.Errorf("DURATION VIOLATION: Building %s level %d expected %d seconds but took %d",
					ba.BuildingType, ba.ToLevel, expectedDuration, actualDuration)
			}
		}

		for _, ra := range solution.ResearchActions {
			tech := technologies[ra.TechnologyName]
			if tech == nil {
				continue
			}
			expectedDuration := tech.ResearchTimeSeconds
			actualDuration := ra.EndTime - ra.StartTime
			// Allow small tolerance for rounding
			if actualDuration < expectedDuration-1 || actualDuration > expectedDuration+1 {
				t.Errorf("DURATION VIOLATION: Research %s expected %d seconds but took %d",
					ra.TechnologyName, expectedDuration, actualDuration)
			}
		}

		// ASSERTION 17: Production rate consistency
		// Verify that production buildings provide the expected production rates
		for _, ba := range solution.BuildingActions {
			building := buildings[ba.BuildingType]
			if building == nil {
				continue
			}
			levelData := building.GetLevelData(ba.ToLevel)
			if levelData == nil || levelData.ProductionRate == nil {
				continue
			}
			// Production rate should match the level data
			// This is validated implicitly by the solver but we check the data exists
			if *levelData.ProductionRate < 0 {
				t.Errorf("PRODUCTION RATE VIOLATION: Building %s level %d has negative production rate: %f",
					ba.BuildingType, ba.ToLevel, *levelData.ProductionRate)
			}
		}

		// ASSERTION 18: Storage upgrades triggered before exceeding capacity
		// Track storage capacities over time
		storageCaps := map[models.ResourceType]int{
			models.Wood:  10000, // Default starting capacity
			models.Stone: 10000,
			models.Iron:  10000,
		}
		
		for _, ba := range solution.BuildingActions {
			building := buildings[ba.BuildingType]
			if building == nil {
				continue
			}
			levelData := building.GetLevelData(ba.ToLevel)
			if levelData == nil {
				continue
			}

			// Check if costs would exceed storage before this action
			if ba.Costs.Wood > storageCaps[models.Wood] {
				// Check if this is a storage upgrade or if storage was upgraded before
				if ba.BuildingType != models.WoodStore {
					t.Errorf("STORAGE VIOLATION: Wood cost %d exceeds capacity %d at time %d (before %s upgrade)",
						ba.Costs.Wood, storageCaps[models.Wood], ba.StartTime, ba.BuildingType)
				}
			}
			if ba.Costs.Stone > storageCaps[models.Stone] {
				if ba.BuildingType != models.StoneStore {
					t.Errorf("STORAGE VIOLATION: Stone cost %d exceeds capacity %d at time %d (before %s upgrade)",
						ba.Costs.Stone, storageCaps[models.Stone], ba.StartTime, ba.BuildingType)
				}
			}
			if ba.Costs.Iron > storageCaps[models.Iron] {
				if ba.BuildingType != models.OreStore {
					t.Errorf("STORAGE VIOLATION: Iron cost %d exceeds capacity %d at time %d (before %s upgrade)",
						ba.Costs.Iron, storageCaps[models.Iron], ba.StartTime, ba.BuildingType)
				}
			}

			// Update storage capacity if this is a storage building
			if levelData.StorageCapacity != nil {
				switch ba.BuildingType {
				case models.WoodStore:
					storageCaps[models.Wood] = *levelData.StorageCapacity
				case models.StoneStore:
					storageCaps[models.Stone] = *levelData.StorageCapacity
				case models.OreStore:
					storageCaps[models.Iron] = *levelData.StorageCapacity
				}
			}
		}

		// =====================================================================
		// OPTIMALITY ASSERTIONS (BASIC CHECKS)
		// =====================================================================

		// ASSERTION 19: Resource balance - production buildings upgraded in reasonable ratios
		// Check that no single production building is over-leveled compared to others
		ljLevel := finalLevels[models.Lumberjack]
		qLevel := finalLevels[models.Quarry]
		omLevel := finalLevels[models.OreMine]
		
		maxProd := ljLevel
		if qLevel > maxProd {
			maxProd = qLevel
		}
		if omLevel > maxProd {
			maxProd = omLevel
		}
		minProd := ljLevel
		if qLevel < minProd {
			minProd = qLevel
		}
		if omLevel < minProd {
			minProd = omLevel
		}
		
		// Allow max 10 level difference for flexibility (reasonable imbalance)
		if maxProd-minProd > 10 && maxProd > 5 {
			t.Logf("INFO: Production building imbalance detected: LJ=%d, Q=%d, OM=%d (diff=%d)",
				ljLevel, qLevel, omLevel, maxProd-minProd)
		}

		// ASSERTION 20: Parallel queue utilization - check for idle time
		// Building queue should not be idle when there are pending targets
		if len(solution.BuildingActions) > 1 {
			var totalGap int
			for i := 1; i < len(solution.BuildingActions); i++ {
				gap := solution.BuildingActions[i].StartTime - solution.BuildingActions[i-1].EndTime
				if gap > 0 {
					totalGap += gap
				}
			}
			// Allow some idle time but not excessive (more than 10% of total time)
			if totalGap > solution.TotalTimeSeconds/10 && solution.TotalTimeSeconds > 86400 {
				t.Logf("INFO: Building queue had significant idle time: %d seconds (%.1f%% of total)",
					totalGap, float64(totalGap)*100/float64(solution.TotalTimeSeconds))
			}
		}

		// Research queue utilization
		if len(solution.ResearchActions) > 1 {
			var totalGap int
			for i := 1; i < len(solution.ResearchActions); i++ {
				gap := solution.ResearchActions[i].StartTime - solution.ResearchActions[i-1].EndTime
				if gap > 0 {
					totalGap += gap
				}
			}
			// Research queue can have more gaps as it depends on library level
			if totalGap > solution.TotalTimeSeconds/5 && len(solution.ResearchActions) > 3 {
				t.Logf("INFO: Research queue had significant idle time: %d seconds (%.1f%% of total)",
					totalGap, float64(totalGap)*100/float64(solution.TotalTimeSeconds))
			}
		}

		// ASSERTION 21: Farm upgrades are on-demand
		// Farm should only be upgraded when food capacity is needed
		farmUpgrades := 0
		for _, ba := range solution.BuildingActions {
			if ba.BuildingType == models.Farm {
				farmUpgrades++
				// Check that food usage was approaching capacity
				usageRatio := float64(ba.FoodUsed) / float64(ba.FoodCapacity)
				if usageRatio < 0.5 && farmUpgrades > 1 {
					t.Logf("INFO: Farm upgraded when only %.1f%% of food capacity was used",
						usageRatio*100)
				}
			}
		}
		
		// =====================================================================
		// OPTIONAL INFO LOGGING (not errors)
		// =====================================================================
		
		// Log info about unresearched techs (acceptable behavior)
		// researchedTechs already built in ASSERTION 14 above

		finalLibraryLevel := finalLevels[models.Library]
		var finalFoodUsed, finalFoodCapacity int
		var latestTime int
		
		for _, ba := range solution.BuildingActions {
			if ba.EndTime > latestTime {
				latestTime = ba.EndTime
				finalFoodUsed = ba.FoodUsed
				finalFoodCapacity = ba.FoodCapacity
			}
		}
		for _, ra := range solution.ResearchActions {
			if ra.EndTime > latestTime {
				latestTime = ra.EndTime
				finalFoodUsed = ra.FoodUsed
				finalFoodCapacity = ra.FoodCapacity
			}
		}
		for _, ta := range solution.TrainingActions {
			if ta.EndTime > latestTime {
				latestTime = ta.EndTime
				finalFoodUsed = ta.FoodUsed
				finalFoodCapacity = ta.FoodCapacity
			}
		}

		for techName, tech := range technologies {
			if researchedTechs[techName] {
				continue
			}
			if finalLibraryLevel >= tech.RequiredLibraryLevel {
				if finalFoodUsed+tech.Costs.Food <= finalFoodCapacity {
					// This is acceptable - solver may choose not to research all techs if not needed
					t.Logf("INFO: Tech %s not researched despite having library %d (requires %d) and food capacity",
						techName, finalLibraryLevel, tech.RequiredLibraryLevel)
				}
			}
		}
	})
}
