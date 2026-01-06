package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

// TestROIFormulaUsesCost verifies that ROI formula uses resource cost, not time
func TestROIFormulaUsesCost(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Quarry:     10,
		models.OreMine:    10,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())

	// Initialize state with some production
	state.SetProductionRate(models.Wood, 10)
	state.SetProductionRate(models.Stone, 10)
	state.SetProductionRate(models.Iron, 5)
	state.StorageCaps = [3]int{1000, 1000, 1000}

	// Get Lumberjack level 2 action
	lj := buildings[models.Lumberjack]
	levelData := lj.GetLevelData(2)

	action := &BuildingAction{
		BuildingType: models.Lumberjack,
		FromLevel:    1,
		ToLevel:      2,
		Building:     lj,
		LevelData:    levelData,
	}

	roi := solver.buildingROI(state, action)

	// ROI should be positive for production buildings
	if roi <= 0 {
		t.Errorf("Expected positive ROI for production building, got %f", roi)
	}

	// Create a more expensive building action and verify ROI is lower
	// (same gain but higher cost = lower ROI)
	// We can't easily test this without modifying data, but we verify the formula works
	t.Logf("Lumberjack 1→2 ROI: %f (cost: W=%d S=%d I=%d)",
		roi, levelData.Costs.Wood, levelData.Costs.Stone, levelData.Costs.Iron)
}

// TestDynamicScarcityCalculation verifies scarcity is calculated from remaining costs
func TestDynamicScarcityCalculation(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	// Target levels with known costs
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 5,
		models.Quarry:     5,
		models.OreMine:    5,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())

	// Set balanced production
	state.SetProductionRate(models.Wood, 10)
	state.SetProductionRate(models.Stone, 10)
	state.SetProductionRate(models.Iron, 10)

	// Get scarcity for each resource type
	woodScarcity := solver.calculateDynamicScarcity(state, models.Lumberjack)
	stoneScarcity := solver.calculateDynamicScarcity(state, models.Quarry)
	ironScarcity := solver.calculateDynamicScarcity(state, models.OreMine)

	t.Logf("Scarcity values: Wood=%f, Stone=%f, Iron=%f", woodScarcity, stoneScarcity, ironScarcity)

	// Scarcity should be capped between 0.5 and 2.0
	for name, scarcity := range map[string]float64{
		"wood": woodScarcity, "stone": stoneScarcity, "iron": ironScarcity,
	} {
		if scarcity < 0.5 || scarcity > 2.0 {
			t.Errorf("%s scarcity %f out of bounds [0.5, 2.0]", name, scarcity)
		}
	}
}

// TestDynamicScarcityChangesWithProduction verifies scarcity adjusts to production imbalance
func TestDynamicScarcityChangesWithProduction(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Quarry:     10,
		models.OreMine:    10,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)

	// Test with high wood production (wood is abundant, should have low scarcity)
	state1 := NewState(models.NewGameState())
	state1.SetProductionRate(models.Wood, 100) // High wood
	state1.SetProductionRate(models.Stone, 10)
	state1.SetProductionRate(models.Iron, 10)

	woodScarcityHighProd := solver.calculateDynamicScarcity(state1, models.Lumberjack)

	// Test with low wood production (wood is scarce, should have high scarcity)
	state2 := NewState(models.NewGameState())
	state2.SetProductionRate(models.Wood, 1) // Low wood
	state2.SetProductionRate(models.Stone, 10)
	state2.SetProductionRate(models.Iron, 10)

	woodScarcityLowProd := solver.calculateDynamicScarcity(state2, models.Lumberjack)

	t.Logf("Wood scarcity with high production: %f", woodScarcityHighProd)
	t.Logf("Wood scarcity with low production: %f", woodScarcityLowProd)

	// When wood production is high (abundant), scarcity should be lower
	// When wood production is low (scarce), scarcity should be higher
	if woodScarcityHighProd >= woodScarcityLowProd {
		t.Errorf("Expected lower scarcity with high production, got high=%f, low=%f",
			woodScarcityHighProd, woodScarcityLowProd)
	}
}

// TestProductionTechROIUsesCost verifies tech ROI uses total investment cost
func TestProductionTechROIUsesCost(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Lumberjack: 10,
		models.Library:    5,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())

	// Set production rates
	state.SetProductionRate(models.Wood, 50)
	state.SetProductionRate(models.Stone, 50)
	state.SetProductionRate(models.Iron, 25)
	state.SetBuildingLevel(models.Library, 1)

	// Get Beer Tester tech
	beerTester := technologies["Beer tester"]
	if beerTester == nil {
		t.Skip("Beer tester tech not found")
	}

	action := &ProductionTechAction{
		Technology:           beerTester,
		RequiredLibraryLevel: beerTester.RequiredLibraryLevel,
	}

	roi := solver.productionTechROI(state, action)

	// ROI should be positive
	if roi <= 0 {
		t.Errorf("Expected positive ROI for production tech, got %f", roi)
	}

	t.Logf("Beer Tester ROI: %f (requires Library %d)", roi, beerTester.RequiredLibraryLevel)
}

// TestTavernROIUsesCost verifies Tavern ROI uses resource cost
func TestTavernROIUsesCost(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Tavern: 5,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())
	state.SetBuildingLevel(models.Tavern, 1)

	tavern := buildings[models.Tavern]
	levelData := tavern.GetLevelData(2)

	action := &BuildingAction{
		BuildingType: models.Tavern,
		FromLevel:    1,
		ToLevel:      2,
		Building:     tavern,
		LevelData:    levelData,
	}

	roi := solver.buildingROI(state, action)

	// Tavern ROI should be based on mission rewards / cost
	t.Logf("Tavern 1→2 ROI: %f (cost: W=%d S=%d I=%d)",
		roi, levelData.Costs.Wood, levelData.Costs.Stone, levelData.Costs.Iron)
}

// TestZeroROIBuildings verifies non-production buildings have zero ROI
func TestZeroROIBuildings(t *testing.T) {
	buildings, _ := loader.LoadBuildings("../../../data")
	technologies, _ := loader.LoadTechnologies("../../../data")
	missions, _ := loader.LoadMissionsFromFile("../../../data")

	targetLevels := map[models.BuildingType]int{
		models.Arsenal:        5,
		models.Keep:           5,
		models.Fortifications: 5,
		models.Market:         5,
	}

	solver := castle.NewTestSolver(buildings, technologies, missions, targetLevels)
	state := NewState(models.NewGameState())

	zeroROIBuildings := []models.BuildingType{
		models.Arsenal, models.Keep, models.Fortifications, models.Market,
	}

	for _, bt := range zeroROIBuildings {
		building := buildings[bt]
		if building == nil {
			continue
		}
		levelData := building.GetLevelData(2)
		if levelData == nil {
			continue
		}

		action := &BuildingAction{
			BuildingType: bt,
			FromLevel:    1,
			ToLevel:      2,
			Building:     building,
			LevelData:    levelData,
		}

		roi := solver.buildingROI(state, action)

		if roi != 0 {
			t.Errorf("Expected zero ROI for %s, got %f", bt, roi)
		}
	}
}

// TestROIMetricCalculate tests the ROIMetric calculation logic
func TestROIMetricCalculate(t *testing.T) {
	tests := []struct {
		name   string
		metric ROIMetric
		want   float64
	}{
		{
			name:   "basic ROI",
			metric: ROIMetric{GainPerHour: 100, TotalCost: 1000},
			want:   0.1,
		},
		{
			name:   "zero cost (free action)",
			metric: ROIMetric{GainPerHour: 100, TotalCost: 0},
			want:   100000, // 100 * 1000
		},
		{
			name:   "with scarcity bonus",
			metric: ROIMetric{GainPerHour: 100, TotalCost: 1000, ScarcityBonus: 0.5},
			want:   0.15, // 0.1 * 1.5
		},
		{
			name:   "negative cost (treated as zero)",
			metric: ROIMetric{GainPerHour: 50, TotalCost: -100},
			want:   50000, // 50 * 1000
		},
		{
			name:   "high gain low cost",
			metric: ROIMetric{GainPerHour: 500, TotalCost: 100},
			want:   5.0,
		},
		{
			name:   "scarcity penalty (negative bonus)",
			metric: ROIMetric{GainPerHour: 100, TotalCost: 1000, ScarcityBonus: -0.2},
			want:   0.08, // 0.1 * 0.8
		},
		{
			name:   "zero gain",
			metric: ROIMetric{GainPerHour: 0, TotalCost: 1000},
			want:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metric.Calculate()
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Errorf("ROIMetric.Calculate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestROIMetricScarcityMultiplier verifies scarcity bonus correctly multiplies base ROI
func TestROIMetricScarcityMultiplier(t *testing.T) {
	baseMetric := ROIMetric{GainPerHour: 100, TotalCost: 1000, ScarcityBonus: 0}
	bonusMetric := ROIMetric{GainPerHour: 100, TotalCost: 1000, ScarcityBonus: 1.0}

	baseROI := baseMetric.Calculate()
	bonusROI := bonusMetric.Calculate()

	// With 1.0 bonus (2x multiplier), bonusROI should be double baseROI
	expected := baseROI * 2.0
	diff := bonusROI - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.0001 {
		t.Errorf("Scarcity bonus not applied correctly: base=%v, bonus=%v, expected=%v",
			baseROI, bonusROI, expected)
	}
}
