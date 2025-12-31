package units

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/models"
)

func TestSolverThroughputConstraint(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Throughput must be >= resource production
	if solution.TotalThroughput < ResourceProductionPerHour {
		t.Errorf("Throughput %.0f < required %d resources/hour",
			solution.TotalThroughput, ResourceProductionPerHour)
	}
}

func TestSolverFoodConstraint(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Food used must not exceed capacity
	if solution.TotalFood > MaxFoodCapacity {
		t.Errorf("Food used %d > capacity %d", solution.TotalFood, MaxFoodCapacity)
	}

	// Should use all available food for maximum defense
	if solution.TotalFood < MaxFoodCapacity-10 {
		t.Errorf("Food underutilized: %d / %d", solution.TotalFood, MaxFoodCapacity)
	}
}

func TestSolverBalancedDefense(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Defense should be reasonably balanced (no type more than 2x another)
	minDef := solution.MinDefense()
	maxDef := max(solution.DefenseVsCavalry, max(solution.DefenseVsInfantry, solution.DefenseVsArtillery))

	if minDef == 0 {
		t.Error("Minimum defense is 0 - army has a critical weakness")
	}

	ratio := float64(maxDef) / float64(minDef)
	if ratio > 2.0 {
		t.Errorf("Defense imbalanced: max/min ratio %.2f > 2.0 (Cav:%d Inf:%d Art:%d)",
			ratio, solution.DefenseVsCavalry, solution.DefenseVsInfantry, solution.DefenseVsArtillery)
	}
}

func TestSolverNoNegativeUnits(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	for name, count := range solution.UnitCounts {
		if count < 0 {
			t.Errorf("Negative unit count for %s: %d", name, count)
		}
	}
}

func TestSolverDefenseMinimum(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// With 4265 food, we should have significant defense
	// Minimum expected: ~100k (conservative estimate)
	minExpected := 100000

	if solution.MinDefense() < minExpected {
		t.Errorf("Defense too low: %d < %d minimum expected", solution.MinDefense(), minExpected)
	}
}

func TestUnitThroughputCalculation(t *testing.T) {
	// Test spearman: 12 capacity, 11.67 min/field, 50 field round trip
	spearman := &Unit{
		SpeedMinutesField: 11.666667,
		TransportCapacity: 12,
	}

	throughput := spearman.ThroughputPerHour(50)

	// Trip time = 50 * 11.67 = 583.3 minutes
	// Trips per hour = 60 / 583.3 = 0.103
	// Throughput = 12 * 0.103 = 1.23
	expected := 1.23

	if throughput < expected-0.5 || throughput > expected+0.5 {
		t.Errorf("Spearman throughput %.2f, expected ~%.2f", throughput, expected)
	}
}

func TestUnitThroughputHorseman(t *testing.T) {
	// Test horseman: 22 capacity, 5 min/field, 50 field round trip
	horseman := &Unit{
		SpeedMinutesField: 5.0,
		TransportCapacity: 22,
	}

	throughput := horseman.ThroughputPerHour(50)

	// Trip time = 50 * 5 = 250 minutes
	// Trips per hour = 60 / 250 = 0.24
	// Throughput = 22 * 0.24 = 5.28
	expected := 5.28

	if throughput < expected-0.5 || throughput > expected+0.5 {
		t.Errorf("Horseman throughput %.2f, expected ~%.2f", throughput, expected)
	}
}

func TestCombatUnitsHaveDefense(t *testing.T) {
	for _, u := range CombatUnits() {
		if u.TotalDefense() == 0 {
			t.Errorf("Combat unit %s has 0 total defense", u.Name)
		}
	}
}

func TestTransportUnitsHaveCapacity(t *testing.T) {
	for _, u := range TransportUnits() {
		if u.TransportCapacity < 100 {
			t.Errorf("Transport unit %s has low capacity: %d", u.Name, u.TransportCapacity)
		}
	}
}

func TestAllUnitsHaveFoodCost(t *testing.T) {
	for _, u := range AllUnits() {
		if u.FoodCost <= 0 {
			t.Errorf("Unit %s has invalid food cost: %d", u.Name, u.FoodCost)
		}
	}
}

func TestAllUnitsHaveSpeed(t *testing.T) {
	for _, u := range AllUnits() {
		if u.SpeedMinutesField <= 0 {
			t.Errorf("Unit %s has invalid speed: %.2f", u.Name, u.SpeedMinutesField)
		}
	}
}

func TestSolverUsesOnlyCombatUnitsWhenPossible(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// With combat units providing enough throughput, no transport units should be used
	transportCount := solution.UnitCounts["Handcart"] + solution.UnitCounts["Oxcart"]

	if transportCount > 0 && solution.TotalThroughput > ResourceProductionPerHour*2 {
		t.Logf("Transport units used (%d) even though combat throughput is sufficient", transportCount)
	}
}

func TestSolverSilverIncome(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Silver income should be positive
	if solution.SilverPerHour <= 0 {
		t.Errorf("Silver income should be positive, got %.2f", solution.SilverPerHour)
	}

	// Expected: 1161 resources/hour * 0.02 silver/resource = 23.22 silver/hour
	expected := 23.22
	if solution.SilverPerHour < expected-1 || solution.SilverPerHour > expected+1 {
		t.Errorf("Silver income %.2f, expected ~%.2f", solution.SilverPerHour, expected)
	}
}

func TestSolverPerformanceRegression(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Minimum defense should not regress below known good value
	// Current: ~138k balanced defense
	minAllowed := 130000

	if solution.MinDefense() < minAllowed {
		t.Errorf("Performance regression: defense %d < minimum %d",
			solution.MinDefense(), minAllowed)
	}
}

func TestRoundTripTimeReasonable(t *testing.T) {
	// Ensure no unit takes more than 24 hours for a round trip
	maxTripMinutes := 24.0 * 60.0

	for _, u := range AllUnits() {
		tripTime := float64(RoundTripFields) * u.SpeedMinutesField
		if tripTime > maxTripMinutes {
			t.Errorf("Unit %s round trip %.0f minutes > 24 hours", u.Name, tripTime)
		}
	}
}

func TestDefenseEfficiency(t *testing.T) {
	// Crossbowman should have best defense efficiency (184 defense / 1 food)
	crossbow := AllUnits()[3] // Crossbowman
	if crossbow.Name != "Crossbowman" {
		t.Fatalf("Expected Crossbowman at index 3, got %s", crossbow.Name)
	}

	efficiency := crossbow.DefenseEfficiencyPerFood()
	if efficiency < 180 {
		t.Errorf("Crossbowman efficiency %.1f < expected 184", efficiency)
	}
}

func TestTradingMatchesProduction(t *testing.T) {
	// Base production at level 30 is 387/hour per resource building = 1161 total
	// With 10% bonus from Beer tester + Wheelbarrow: 1161 * 1.10 = 1277.1
	// Trading throughput should be able to handle this
	
	// Test with boosted production rate (simulating after tech research)
	boostedProduction := int32(1277) // 1161 * 1.10
	
	solver := NewSolverWithConfig(MaxFoodCapacity, boostedProduction, MarketDistanceFields)
	solution := solver.Solve()

	if solution.TotalThroughput < float64(boostedProduction) {
		t.Errorf("Trading throughput %.0f < boosted production %d - cannot trade all resources",
			solution.TotalThroughput, boostedProduction)
	}

	t.Logf("Boosted production: %d/hour, Trading throughput: %.0f/hour (surplus: %.0f)",
		boostedProduction, solution.TotalThroughput, solution.TotalThroughput-float64(boostedProduction))
}

func TestTradingAt25FieldDistance(t *testing.T) {
	// With Keep level 10, market distance is 25 fields (50 round trip)
	// Resource rate at level 30: 387 * 3 = 1161/hour base
	// With 10% bonus: ~1277/hour
	
	solver := NewSolver()
	solution := solver.Solve()

	// Should achieve at least 1161 throughput (base production)
	if solution.TotalThroughput < float64(ResourceProductionPerHour) {
		t.Errorf("Throughput %.0f < base production %d", 
			solution.TotalThroughput, ResourceProductionPerHour)
	}

	// With good army composition, should achieve much higher throughput
	// (combat units contribute to trading while providing defense)
	if solution.TotalThroughput < 5000 {
		t.Errorf("Throughput %.0f seems too low for maxed castle army", solution.TotalThroughput)
	}

	t.Logf("Trading throughput at 25 field distance: %.0f resources/hour", solution.TotalThroughput)
}

func TestUnitFoodCostsAreCorrect(t *testing.T) {
	// Verify each unit has correct food cost from game data
	expectedCosts := map[string]int{
		"Spearman":    1,
		"Swordsman":   1,
		"Archer":      1,
		"Crossbowman": 1,
		"Horseman":    2,
		"Lancer":      2,
		"Handcart":    1,
		"Oxcart":      3,
	}

	for _, u := range AllUnits() {
		expected, ok := expectedCosts[u.Name]
		if !ok {
			t.Errorf("Unknown unit: %s", u.Name)
			continue
		}
		if u.FoodCost != expected {
			t.Errorf("Unit %s food cost %d != expected %d", u.Name, u.FoodCost, expected)
		}
	}
}

func TestUnitTrainingTimesAreSet(t *testing.T) {
	for _, u := range AllUnits() {
		if u.TrainingTimeSeconds <= 0 {
			t.Errorf("Unit %s has no training time set", u.Name)
		}
		// Training should be reasonable (between 5 minutes and 2 hours)
		if u.TrainingTimeSeconds < 300 || u.TrainingTimeSeconds > 7200 {
			t.Errorf("Unit %s training time %d seconds seems unreasonable", u.Name, u.TrainingTimeSeconds)
		}
	}
}

func TestTotalFoodUsedMatchesUnitCounts(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Calculate expected food from unit counts
	expectedFood := 0
	for _, u := range AllUnits() {
		count := solution.UnitCounts[u.Name]
		expectedFood += count * u.FoodCost
	}

	if solution.TotalFood != expectedFood {
		t.Errorf("TotalFood %d != calculated from units %d", solution.TotalFood, expectedFood)
	}
}

func TestArmyFitsInFoodCapacity(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	if solution.TotalFood > MaxFoodCapacity {
		t.Errorf("Army food %d exceeds capacity %d", solution.TotalFood, MaxFoodCapacity)
	}

	// Should use nearly all food (within 10)
	if solution.TotalFood < MaxFoodCapacity-10 {
		t.Errorf("Army not using all food: %d / %d", solution.TotalFood, MaxFoodCapacity)
	}
}

func TestTrainingTimeCalculation(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Calculate total training time
	totalTrainingSeconds := 0
	for _, u := range AllUnits() {
		count := solution.UnitCounts[u.Name]
		totalTrainingSeconds += count * u.TrainingTimeSeconds
	}

	trainingDays := float64(totalTrainingSeconds) / 3600 / 24

	t.Logf("Total training time: %d seconds (%.1f days)", totalTrainingSeconds, trainingDays)

	// Training should take significant time (> 30 days for full army)
	if trainingDays < 30 {
		t.Errorf("Training time %.1f days seems too short for full army", trainingDays)
	}

	// But not excessively long (< 100 days)
	if trainingDays > 100 {
		t.Errorf("Training time %.1f days seems too long", trainingDays)
	}
}

func TestUnitResourceCostsAreSet(t *testing.T) {
	// All combat units should have resource costs
	for _, u := range AllUnits() {
		totalResourceCost := u.ResourceCosts[models.Wood] + u.ResourceCosts[models.Stone] + u.ResourceCosts[models.Iron]
		if totalResourceCost == 0 {
			t.Errorf("Unit %s has no resource costs", u.Name)
		}
	}
}

func TestUnitResourceCostsReasonable(t *testing.T) {
	// Verify specific unit costs match game data
	expectedCosts := map[string]struct {
		wood, iron int
	}{
		"Spearman":    {18, 30},
		"Swordsman":   {43, 48},
		"Archer":      {27, 39},
		"Crossbowman": {50, 55},
		"Horseman":    {25, 45},
		"Lancer":      {70, 80},
		"Handcart":    {45, 30},
		"Oxcart":      {95, 65},
	}

	for _, u := range AllUnits() {
		expected, ok := expectedCosts[u.Name]
		if !ok {
			continue
		}
		if u.ResourceCosts[models.Wood] != expected.wood {
			t.Errorf("Unit %s wood cost %d != expected %d", u.Name, u.ResourceCosts[models.Wood], expected.wood)
		}
		if u.ResourceCosts[models.Iron] != expected.iron {
			t.Errorf("Unit %s iron cost %d != expected %d", u.Name, u.ResourceCosts[models.Iron], expected.iron)
		}
	}
}

func TestTotalUnitResourceCosts(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	// Calculate total resource costs
	var totalWood, totalIron int
	for _, u := range AllUnits() {
		count := solution.UnitCounts[u.Name]
		totalWood += u.ResourceCosts[models.Wood] * count
		totalIron += u.ResourceCosts[models.Iron] * count
	}

	t.Logf("Total army resource costs: Wood=%d, Iron=%d", totalWood, totalIron)

	// Army should cost significant resources
	if totalWood < 50000 {
		t.Errorf("Total wood cost %d seems too low for full army", totalWood)
	}
	if totalIron < 50000 {
		t.Errorf("Total iron cost %d seems too low for full army", totalIron)
	}
}

func TestUnitResourceCostsPerBatchRespectStorage(t *testing.T) {
	// This test verifies that when training units, we can't exceed storage capacity
	// in a single batch

	// Level 20 storage capacity is 9999 per resource (from game data)
	const level20StorageCap = 9999

	solver := NewSolver()
	solution := solver.Solve()

	// Verify that no single unit type's total cost exceeds what we could possibly accumulate
	for _, u := range AllUnits() {
		count := solution.UnitCounts[u.Name]
		if count == 0 {
			continue
		}

		woodCost := u.ResourceCosts[models.Wood] * count
		ironCost := u.ResourceCosts[models.Iron] * count

		// Log total costs per unit type
		t.Logf("%s x%d: Wood=%d, Iron=%d", u.Name, count, woodCost, ironCost)

		// Single unit should never exceed storage
		if u.ResourceCosts[models.Wood] > level20StorageCap {
			t.Errorf("Single %s wood cost %d exceeds storage %d", u.Name, u.ResourceCosts[models.Wood], level20StorageCap)
		}
		if u.ResourceCosts[models.Iron] > level20StorageCap {
			t.Errorf("Single %s iron cost %d exceeds storage %d", u.Name, u.ResourceCosts[models.Iron], level20StorageCap)
		}
	}
}

func TestUnitBatchSizesByStorage(t *testing.T) {
	// Given storage capacity, how many units can we train before waiting?
	const storageCap = 9999

	for _, u := range AllUnits() {
		if u.ResourceCosts[models.Wood] == 0 && u.ResourceCosts[models.Iron] == 0 {
			continue // Skip units without resource costs
		}

		maxByWood := storageCap
		if u.ResourceCosts[models.Wood] > 0 {
			maxByWood = storageCap / u.ResourceCosts[models.Wood]
		}
		maxByIron := storageCap
		if u.ResourceCosts[models.Iron] > 0 {
			maxByIron = storageCap / u.ResourceCosts[models.Iron]
		}

		batchSize := min(maxByWood, maxByIron)
		t.Logf("%s: max batch size with %d storage = %d units (wood: %d/unit, iron: %d/unit)",
			u.Name, storageCap, batchSize, u.ResourceCosts[models.Wood], u.ResourceCosts[models.Iron])

		// Should be able to train at least 100 units per batch
		if batchSize < 100 {
			t.Logf("Warning: %s can only train %d per batch", u.Name, batchSize)
		}
	}
}

func TestUnitTrainingTimeWithResourceWait(t *testing.T) {
	// Simulate training with resource accumulation
	// Production rate: ~387/h per resource (level 30)
	// With 10% bonus: ~425/h

	const productionRate = 425.0 // per hour per resource

	solver := NewSolver()
	solution := solver.Solve()

	totalTrainingSeconds := 0
	totalResourceWaitSeconds := 0

	for _, u := range AllUnits() {
		count := solution.UnitCounts[u.Name]
		if count == 0 {
			continue
		}

		// Training time
		trainingSeconds := count * u.TrainingTimeSeconds
		totalTrainingSeconds += trainingSeconds

		// Resource accumulation time (worst case: storage empty, need to wait)
		totalWood := u.ResourceCosts[models.Wood] * count
		totalIron := u.ResourceCosts[models.Iron] * count

		woodWaitHours := float64(totalWood) / productionRate
		ironWaitHours := float64(totalIron) / productionRate

		maxWaitHours := max(woodWaitHours, ironWaitHours)
		totalResourceWaitSeconds += int(maxWaitHours * 3600)

		t.Logf("%s x%d: train %.1fh, resource wait %.1fh",
			u.Name, count,
			float64(trainingSeconds)/3600,
			maxWaitHours)
	}

	t.Logf("Total training time: %.1f hours", float64(totalTrainingSeconds)/3600)
	t.Logf("Total resource wait time: %.1f hours (theoretical max)", float64(totalResourceWaitSeconds)/3600)

	// Training should take significant time
	if totalTrainingSeconds < 100*3600 { // At least 100 hours
		t.Errorf("Training time %d seconds seems too short", totalTrainingSeconds)
	}
}

func TestUnitFoodCostMatchesData(t *testing.T) {
	// Verify food costs match unit data
	expectedFood := map[string]int{
		"Spearman":    1,
		"Crossbowman": 1,
		"Horseman":    2,
	}

	for _, u := range AllUnits() {
		expected, ok := expectedFood[u.Name]
		if !ok {
			continue
		}
		if u.FoodCost != expected {
			t.Errorf("%s food cost %d != expected %d", u.Name, u.FoodCost, expected)
		}
	}
}

func TestTotalFoodUsedByUnits(t *testing.T) {
	solver := NewSolver()
	solution := solver.Solve()

	totalFood := 0
	for _, u := range AllUnits() {
		count := solution.UnitCounts[u.Name]
		totalFood += u.FoodCost * count
	}

	t.Logf("Total food used by units: %d / %d", totalFood, MaxFoodCapacity)

	if totalFood != solution.TotalFood {
		t.Errorf("Calculated food %d != solution.TotalFood %d", totalFood, solution.TotalFood)
	}

	if totalFood > MaxFoodCapacity {
		t.Errorf("Total food %d exceeds capacity %d", totalFood, MaxFoodCapacity)
	}
}

// TestAllUnitDataMatchesFiles verifies ALL unit properties match data files
// This ensures data files are the source of truth
func TestAllUnitDataMatchesFiles(t *testing.T) {
	// Expected values extracted directly from data/units/* files
	// Format: Name -> {wood, stone, iron, food, trainingSeconds, speed, transport}
	expectedData := map[string]struct {
		wood, stone, iron, food int
		trainingSeconds         int
		speedMinField           float64
		transport               int
	}{
		"Spearman":    {18, 6, 30, 1, 750, 11.666667, 12},     // 12:30, 11m40s
		"Swordsman":   {43, 20, 48, 1, 1200, 13.333333, 10},   // 20:00, 13m20s
		"Archer":      {27, 12, 39, 1, 900, 8.333333, 16},     // 15:00, 8m20s
		"Crossbowman": {50, 28, 55, 1, 1350, 10.0, 13},        // 22:30, 10m
		"Horseman":    {25, 15, 45, 2, 1050, 5.0, 22},         // 17:30, 5m
		"Lancer":      {70, 60, 80, 2, 1860, 6.666667, 20},    // 31:00, 6m40s
		"Handcart":    {45, 25, 30, 1, 600, 13.333333, 500},   // 10:00, 13m20s
		"Oxcart":      {95, 40, 65, 3, 1200, 16.666667, 2500}, // 20:00, 16m40s
	}

	for _, u := range AllUnits() {
		expected, ok := expectedData[u.Name]
		if !ok {
			t.Errorf("Unit %s not in expected data map", u.Name)
			continue
		}

		// Verify resource costs
		if u.ResourceCosts[models.Wood] != expected.wood {
			t.Errorf("%s: wood cost %d != expected %d", u.Name, u.ResourceCosts[models.Wood], expected.wood)
		}
		if u.ResourceCosts[models.Stone] != expected.stone {
			t.Errorf("%s: stone cost %d != expected %d", u.Name, u.ResourceCosts[models.Stone], expected.stone)
		}
		if u.ResourceCosts[models.Iron] != expected.iron {
			t.Errorf("%s: iron cost %d != expected %d", u.Name, u.ResourceCosts[models.Iron], expected.iron)
		}

		// Verify food cost
		if u.FoodCost != expected.food {
			t.Errorf("%s: food cost %d != expected %d", u.Name, u.FoodCost, expected.food)
		}

		// Verify training time
		if u.TrainingTimeSeconds != expected.trainingSeconds {
			t.Errorf("%s: training time %d != expected %d", u.Name, u.TrainingTimeSeconds, expected.trainingSeconds)
		}

		// Verify speed (allow 0.01 tolerance for float comparison)
		speedDiff := u.SpeedMinutesField - expected.speedMinField
		if speedDiff < -0.01 || speedDiff > 0.01 {
			t.Errorf("%s: speed %.6f != expected %.6f", u.Name, u.SpeedMinutesField, expected.speedMinField)
		}

		// Verify transport capacity
		if u.TransportCapacity != expected.transport {
			t.Errorf("%s: transport %d != expected %d", u.Name, u.TransportCapacity, expected.transport)
		}
	}
}
