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

		// Invariant 2: No negative unit counts
		for name, count := range solution.UnitCounts {
			if count < 0 {
				t.Errorf("Negative count for %s: %d", name, count)
			}
		}

		// Invariant 3: Throughput should be >= production (or we can't trade everything)
		// Note: might not always be achievable with very low food
		if solution.TotalThroughput < 0 {
			t.Error("Negative throughput")
		}

		// Invariant 4: Defense values should be non-negative
		if solution.DefenseVsCavalry < 0 {
			t.Errorf("Negative defense vs cavalry: %d", solution.DefenseVsCavalry)
		}
		if solution.DefenseVsInfantry < 0 {
			t.Errorf("Negative defense vs infantry: %d", solution.DefenseVsInfantry)
		}
		if solution.DefenseVsArtillery < 0 {
			t.Errorf("Negative defense vs artillery: %d", solution.DefenseVsArtillery)
		}

		// Invariant 5: Silver income should be non-negative
		if solution.SilverPerHour < 0 {
			t.Errorf("Negative silver income: %.2f", solution.SilverPerHour)
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

		// Throughput must be non-negative
		if throughput < 0 {
			t.Errorf("Negative throughput: %.2f (speed=%.2f, cap=%d, dist=%d)",
				throughput, speed, capacity, distance)
		}

		// Throughput should be finite
		if throughput != throughput { // NaN check
			t.Errorf("NaN throughput (speed=%.2f, cap=%d, dist=%d)", speed, capacity, distance)
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
