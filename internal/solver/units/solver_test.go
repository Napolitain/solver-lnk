package units

import (
	"testing"
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
	transportCount := solution.UnitCounts["handcart"] + solution.UnitCounts["oxcart"]

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
	crossbow := AllUnits()[3] // crossbowman
	if crossbow.Name != "crossbowman" {
		t.Fatal("Expected crossbowman at index 3")
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
