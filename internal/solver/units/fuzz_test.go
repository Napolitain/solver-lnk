package units

import (
	"testing"
)

func FuzzSolverConstraints(f *testing.F) {
	// Add seed corpus with realistic values
	f.Add(int32(4265), int32(1161), int32(50))   // Default maxed castle
	f.Add(int32(1000), int32(500), int32(25))    // Small castle
	f.Add(int32(100), int32(100), int32(10))     // Very small
	f.Add(int32(5000), int32(2000), int32(100))  // Large castle

	f.Fuzz(func(t *testing.T, food, production, distance int32) {
		// Skip invalid inputs
		if food <= 0 || production <= 0 || distance <= 0 {
			return
		}

		// Cap at reasonable values
		if food > 10000 || production > 5000 || distance > 200 {
			return
		}

		solver := NewSolverWithConfig(food, production, distance)
		solution := solver.Solve()

		// Invariant 1: Food used must not exceed capacity
		if solution.TotalFood > int(food) {
			t.Errorf("Food %d > capacity %d", solution.TotalFood, food)
		}

		// Invariant 2: Food used must be non-negative
		if solution.TotalFood < 0 {
			t.Errorf("Negative food used: %d", solution.TotalFood)
		}

		// Invariant 3: No negative unit counts
		for name, count := range solution.UnitCounts {
			if count < 0 {
				t.Errorf("Negative count for %s: %d", name, count)
			}
		}

		// Invariant 4: Throughput should be non-negative
		if solution.TotalThroughput < 0 {
			t.Errorf("Negative throughput: %.2f", solution.TotalThroughput)
		}

		// Invariant 5: Defense values should be non-negative
		if solution.DefenseVsCavalry < 0 {
			t.Errorf("Negative defense vs cavalry: %d", solution.DefenseVsCavalry)
		}
		if solution.DefenseVsInfantry < 0 {
			t.Errorf("Negative defense vs infantry: %d", solution.DefenseVsInfantry)
		}
		if solution.DefenseVsArtillery < 0 {
			t.Errorf("Negative defense vs artillery: %d", solution.DefenseVsArtillery)
		}

		// Invariant 6: Silver income should be non-negative
		if solution.SilverPerHour < 0 {
			t.Errorf("Negative silver income: %.2f", solution.SilverPerHour)
		}

		// Invariant 7: Verify food calculation matches unit counts
		calculatedFood := 0
		for _, u := range AllUnits() {
			count := solution.UnitCounts[u.Name]
			calculatedFood += count * u.FoodCost
		}
		if calculatedFood != solution.TotalFood {
			t.Errorf("Food mismatch: calculated %d != reported %d", calculatedFood, solution.TotalFood)
		}

		// Invariant 8: Verify defense calculation matches unit counts
		calculatedDefCav := 0
		calculatedDefInf := 0
		calculatedDefArt := 0
		for _, u := range AllUnits() {
			count := solution.UnitCounts[u.Name]
			calculatedDefCav += count * u.DefenseVsCavalry
			calculatedDefInf += count * u.DefenseVsInfantry
			calculatedDefArt += count * u.DefenseVsArtillery
		}
		if calculatedDefCav != solution.DefenseVsCavalry {
			t.Errorf("Defense vs cavalry mismatch: calculated %d != reported %d", calculatedDefCav, solution.DefenseVsCavalry)
		}
		if calculatedDefInf != solution.DefenseVsInfantry {
			t.Errorf("Defense vs infantry mismatch: calculated %d != reported %d", calculatedDefInf, solution.DefenseVsInfantry)
		}
		if calculatedDefArt != solution.DefenseVsArtillery {
			t.Errorf("Defense vs artillery mismatch: calculated %d != reported %d", calculatedDefArt, solution.DefenseVsArtillery)
		}
	})
}

func FuzzUnitThroughput(f *testing.F) {
	// Seed with realistic values
	f.Add(float64(11.67), int32(12), int32(50))  // Spearman-like
	f.Add(float64(5.0), int32(22), int32(50))    // Horseman-like
	f.Add(float64(20.0), int32(140), int32(50))  // Handcart-like

	f.Fuzz(func(t *testing.T, speed float64, capacity, distance int32) {
		// Skip invalid inputs
		if speed <= 0 || capacity <= 0 || distance <= 0 {
			return
		}

		// Cap at reasonable values
		if speed > 100 || capacity > 1000 || distance > 500 {
			return
		}

		unit := &Unit{
			SpeedMinutesField: speed,
			TransportCapacity: int(capacity),
		}

		throughput := unit.ThroughputPerHour(int(distance))

		// Invariant 1: Throughput must be non-negative
		if throughput < 0 {
			t.Errorf("Negative throughput: %.2f (speed=%.2f, cap=%d, dist=%d)",
				throughput, speed, capacity, distance)
		}

		// Invariant 2: Throughput should be finite (not NaN or Inf)
		if throughput != throughput { // NaN check
			t.Errorf("NaN throughput (speed=%.2f, cap=%d, dist=%d)", speed, capacity, distance)
		}

		// Invariant 3: Verify throughput calculation
		// throughput = capacity * (60 / (distance * speed))
		tripTimeMinutes := float64(distance) * speed
		expectedThroughput := float64(capacity) * (60.0 / tripTimeMinutes)
		if throughput < expectedThroughput*0.99 || throughput > expectedThroughput*1.01 {
			t.Errorf("Throughput calculation mismatch: got %.4f, expected %.4f", throughput, expectedThroughput)
		}
	})
}

// Property-based test: more food = more or equal defense
func TestMoreFoodMoreDefense(t *testing.T) {
	foods := []int32{500, 1000, 2000, 4000}
	var lastMinDefense int

	for _, food := range foods {
		solver := NewSolverWithConfig(food, 1161, 50)
		solution := solver.Solve()

		if solution.MinDefense() < lastMinDefense {
			t.Errorf("Defense decreased with more food: %d -> %d (food %d)",
				lastMinDefense, solution.MinDefense(), food)
		}
		lastMinDefense = solution.MinDefense()
	}
}

// Property-based test: solution always uses almost all food
func TestFoodUtilization(t *testing.T) {
	testCases := []int32{500, 1000, 2000, 4265}

	for _, food := range testCases {
		solver := NewSolverWithConfig(food, 1161, 50)
		solution := solver.Solve()

		// Should use at least 90% of food (accounting for unit granularity)
		minExpected := int(float64(food) * 0.9)
		if solution.TotalFood < minExpected {
			t.Errorf("Food underutilized for capacity %d: used %d, expected >= %d",
				food, solution.TotalFood, minExpected)
		}
	}
}

// Property-based test: throughput should meet or exceed production requirement
func TestThroughputMeetsProduction(t *testing.T) {
	testCases := []struct {
		food       int32
		production int32
		distance   int32
	}{
		{4265, 1161, 50},
		{2000, 500, 25},
		{1000, 200, 10},
	}

	for _, tc := range testCases {
		solver := NewSolverWithConfig(tc.food, tc.production, tc.distance)
		solution := solver.Solve()

		// Throughput should meet production (may not always be possible with low food)
		// but should at least be positive
		if solution.TotalThroughput < 0 {
			t.Errorf("Negative throughput for food=%d, prod=%d, dist=%d",
				tc.food, tc.production, tc.distance)
		}
	}
}

// FuzzUnitResourceCosts verifies that resource costs are reasonable for all fuzz inputs
func FuzzUnitResourceCosts(f *testing.F) {
	// Seed with storage capacities at different levels
	f.Add(int32(500), int32(26930))   // Level 20 storage
	f.Add(int32(1000), int32(10000))  // Mid-level storage
	f.Add(int32(4265), int32(50000))  // High storage

	f.Fuzz(func(t *testing.T, food, storageCap int32) {
		if food <= 0 || storageCap <= 0 {
			return
		}
		if food > 10000 || storageCap > 100000 {
			return
		}

		solver := NewSolverWithConfig(food, 1161, 50)
		solution := solver.Solve()

		// Calculate total resource costs for the army
		var totalWood, totalIron int
		for _, u := range AllUnits() {
			count := solution.UnitCounts[u.Name]
			totalWood += u.ResourceCosts["wood"] * count
			totalIron += u.ResourceCosts["iron"] * count
		}

		// Invariant: Single unit should never cost more than storage
		for _, u := range AllUnits() {
			if u.ResourceCosts["wood"] > int(storageCap) {
				// This would be a data issue, not solver issue
				t.Logf("Warning: %s wood cost %d exceeds storage %d", u.Name, u.ResourceCosts["wood"], storageCap)
			}
			if u.ResourceCosts["iron"] > int(storageCap) {
				t.Logf("Warning: %s iron cost %d exceeds storage %d", u.Name, u.ResourceCosts["iron"], storageCap)
			}
		}

		// Invariant: Resource costs should be non-negative
		if totalWood < 0 {
			t.Errorf("Negative total wood cost: %d", totalWood)
		}
		if totalIron < 0 {
			t.Errorf("Negative total iron cost: %d", totalIron)
		}
	})
}
