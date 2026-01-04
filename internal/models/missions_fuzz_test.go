package models

import (
	"testing"
)

// FuzzMissionRewards tests that mission reward calculations are always valid
func FuzzMissionRewards(f *testing.F) {
	// Seed with realistic values
	f.Add(10, 20, 30, 60, 100)     // hunting-like: small rewards, short duration
	f.Add(800, 1000, 720, 0, 400)  // castle festival-like: large rewards, long duration
	f.Add(750, 1500, 300, 700, 70) // trading post-like: with costs

	f.Fuzz(func(t *testing.T, rewardMin, rewardMax, durationMin, costAmount, unitsRequired int) {
		// Ensure valid inputs
		if rewardMin < 0 || rewardMax < 0 || durationMin < 0 || costAmount < 0 || unitsRequired < 0 {
			return
		}
		if rewardMin > rewardMax {
			rewardMin, rewardMax = rewardMax, rewardMin
		}
		if durationMin > 10000 || unitsRequired > 10000 {
			return // Reasonable bounds
		}

		m := &Mission{
			Name:            "FuzzMission",
			DurationMinutes: durationMin,
			UnitsRequired: []UnitRequirement{
				{Type: Archer, Count: unitsRequired},
			},
			ResourceCosts: Costs{Wood: costAmount},
			Rewards: []ResourceReward{
				{Type: Wood, Min: rewardMin, Max: rewardMax},
			},
		}

		// Property: Average should be between min and max
		for _, r := range m.Rewards {
			avg := r.AverageReward()
			if avg < float64(r.Min) || avg > float64(r.Max) {
				t.Errorf("Average %.2f not in [%d, %d]", avg, r.Min, r.Max)
			}
		}

		// Property: Total units should match
		if m.TotalUnitsRequired() != unitsRequired {
			t.Errorf("TotalUnitsRequired = %d, want %d", m.TotalUnitsRequired(), unitsRequired)
		}

		// Property: Net reward should be (reward - cost)
		expectedNet := float64(rewardMin+rewardMax)/2.0 - float64(costAmount)
		if abs(m.NetAverageReward()-expectedNet) > 0.01 {
			t.Errorf("NetAverageReward = %.2f, want %.2f", m.NetAverageReward(), expectedNet)
		}

		// Property: Per-hour rate should be finite and well-defined
		if durationMin > 0 {
			perHour := m.NetAverageRewardPerHour()
			expectedPerHour := expectedNet / (float64(durationMin) / 60.0)
			if abs(perHour-expectedPerHour) > 0.01 {
				t.Errorf("NetAverageRewardPerHour = %.2f, want %.2f", perHour, expectedPerHour)
			}
		}

		// Property: Per-unit-hour should be finite and well-defined
		if durationMin > 0 && unitsRequired > 0 {
			perUnitHour := m.NetAverageRewardPerUnitHour()
			hours := float64(durationMin) / 60.0
			expectedPerUnitHour := expectedNet / (hours * float64(unitsRequired))
			if abs(perUnitHour-expectedPerUnitHour) > 0.01 {
				t.Errorf("NetAverageRewardPerUnitHour = %.2f, want %.2f", perUnitHour, expectedPerUnitHour)
			}
		}
	})
}

// FuzzMissionScheduler tests scheduler invariants under random conditions
func FuzzMissionScheduler(f *testing.F) {
	// Seed: unitCount, missionUnitsNeeded, startTime, checkTime
	f.Add(100, 15, 0, 900)
	f.Add(50, 50, 0, 1000)
	f.Add(200, 30, 100, 500)

	f.Fuzz(func(t *testing.T, unitCount, missionUnitsNeeded, startTime, checkTime int) {
		// Validate inputs
		if unitCount < 0 || unitCount > 10000 {
			return
		}
		if missionUnitsNeeded < 0 || missionUnitsNeeded > 1000 {
			return
		}
		if startTime < 0 || checkTime < 0 {
			return
		}
		if startTime > 1000000 || checkTime > 1000000 {
			return
		}

		mission := &Mission{
			Name:            "FuzzTestMission",
			DurationMinutes: 15,
			UnitsRequired: []UnitRequirement{
				{Type: Archer, Count: missionUnitsNeeded},
			},
			Rewards: []ResourceReward{{Type: Wood, Min: 10, Max: 20}},
		}

		units := map[UnitType]int{Archer: unitCount}
		scheduler := NewMissionScheduler([]*Mission{mission}, units)

		// Property: Initial available should equal total
		if scheduler.AvailableUnits[Archer] != unitCount {
			t.Errorf("Initial available %d != total %d", scheduler.AvailableUnits[Archer], unitCount)
		}

		// Property: CanStartMission should match unit availability
		canStart := scheduler.CanStartMission(mission)
		shouldBeAble := unitCount >= missionUnitsNeeded
		if canStart != shouldBeAble {
			t.Errorf("CanStartMission = %v, but units=%d, needed=%d",
				canStart, unitCount, missionUnitsNeeded)
		}

		// Start mission if possible
		if canStart {
			state := scheduler.StartMission(mission, startTime)
			if state == nil {
				t.Error("StartMission returned nil when CanStartMission was true")
				return
			}

			// Property: Available units decreased by mission requirement
			expectedAvailable := unitCount - missionUnitsNeeded
			if scheduler.AvailableUnits[Archer] != expectedAvailable {
				t.Errorf("After start: available = %d, want %d",
					scheduler.AvailableUnits[Archer], expectedAvailable)
			}

			// Property: Available units should never be negative
			if scheduler.AvailableUnits[Archer] < 0 {
				t.Errorf("Available units went negative: %d", scheduler.AvailableUnits[Archer])
			}

			// Property: Mission end time should be start + duration
			expectedEnd := startTime + 15*60
			if state.EndTime != expectedEnd {
				t.Errorf("EndTime = %d, want %d", state.EndTime, expectedEnd)
			}

			// Check completion at checkTime
			completed := scheduler.CompleteMissions(checkTime)

			if checkTime >= expectedEnd {
				// Should have completed
				if len(completed) != 1 {
					t.Errorf("Expected 1 completed at time %d (end=%d), got %d",
						checkTime, expectedEnd, len(completed))
				}
				// Units should be back
				if scheduler.AvailableUnits[Archer] != unitCount {
					t.Errorf("After completion: available = %d, want %d",
						scheduler.AvailableUnits[Archer], unitCount)
				}
			} else {
				// Should not have completed yet
				if len(completed) != 0 {
					t.Errorf("Expected 0 completed at time %d (end=%d), got %d",
						checkTime, expectedEnd, len(completed))
				}
			}
		}
	})
}

// FuzzMultipleMissions tests scheduler with multiple simultaneous missions
func FuzzMultipleMissions(f *testing.F) {
	f.Add(500, 3, 0)
	f.Add(100, 5, 100)
	f.Add(1000, 10, 0)

	f.Fuzz(func(t *testing.T, totalUnits, numMissions, startTime int) {
		if totalUnits < 0 || totalUnits > 10000 {
			return
		}
		if numMissions < 1 || numMissions > 50 {
			return
		}
		if startTime < 0 || startTime > 1000000 {
			return
		}

		// Create missions with varying unit requirements
		missions := make([]*Mission, numMissions)
		for i := 0; i < numMissions; i++ {
			missions[i] = &Mission{
				Name:            "Mission",
				DurationMinutes: 15 + i*5, // Varying durations
				UnitsRequired: []UnitRequirement{
					{Type: Archer, Count: 10 + i},
				},
				Rewards: []ResourceReward{{Type: Wood, Min: 10, Max: 20}},
			}
		}

		units := map[UnitType]int{Archer: totalUnits}
		scheduler := NewMissionScheduler(missions, units)

		// Start as many missions as possible
		startedCount := 0
		for _, m := range missions {
			if scheduler.CanStartMission(m) {
				state := scheduler.StartMission(m, startTime)
				if state != nil {
					startedCount++
				}
			}
		}

		// Property: Running missions should equal started count
		if len(scheduler.RunningMissions) != startedCount {
			t.Errorf("Running = %d, started = %d", len(scheduler.RunningMissions), startedCount)
		}

		// Property: Available units should never be negative
		if scheduler.AvailableUnits[Archer] < 0 {
			t.Errorf("Available units negative: %d", scheduler.AvailableUnits[Archer])
		}

		// Property: Total units (available + in missions) should equal original
		inMissions := 0
		for _, state := range scheduler.RunningMissions {
			for _, req := range state.AssignedUnits {
				if req.Type == Archer {
					inMissions += req.Count
				}
			}
		}
		totalAccountedFor := scheduler.AvailableUnits[Archer] + inMissions
		if totalAccountedFor != totalUnits {
			t.Errorf("Unit accounting error: available=%d + inMissions=%d = %d, want %d",
				scheduler.AvailableUnits[Archer], inMissions, totalAccountedFor, totalUnits)
		}

		// Complete all missions
		maxEndTime := 0
		for _, state := range scheduler.RunningMissions {
			if state.EndTime > maxEndTime {
				maxEndTime = state.EndTime
			}
		}

		scheduler.CompleteMissions(maxEndTime + 1)

		// Property: After all complete, all units should be available
		if scheduler.AvailableUnits[Archer] != totalUnits {
			t.Errorf("After all complete: available = %d, want %d",
				scheduler.AvailableUnits[Archer], totalUnits)
		}
	})
}

// FuzzROIComparison tests that ROI calculations remain consistent
func FuzzROIComparison(f *testing.F) {
	// Different mission profiles
	f.Add(15, 15, 45, 0)       // Short, low units, small reward, no cost
	f.Add(300, 240, 2000, 700) // Long, many units, large reward, significant cost
	f.Add(60, 50, 200, 100)    // Medium everything

	f.Fuzz(func(t *testing.T, durationMin, unitsRequired, avgReward, cost int) {
		if durationMin <= 0 || durationMin > 1440 { // max 24 hours
			return
		}
		if unitsRequired <= 0 || unitsRequired > 500 {
			return
		}
		if avgReward < 0 || avgReward > 10000 {
			return
		}
		if cost < 0 || cost > avgReward*2 { // Cost shouldn't be absurdly high
			return
		}

		m := &Mission{
			Name:            "ROITest",
			DurationMinutes: durationMin,
			UnitsRequired: []UnitRequirement{
				{Type: Archer, Count: unitsRequired},
			},
			ResourceCosts: Costs{Wood: cost},
			Rewards: []ResourceReward{
				{Type: Wood, Min: avgReward, Max: avgReward}, // Fixed reward for predictability
			},
		}

		netReward := float64(avgReward - cost)
		hours := float64(durationMin) / 60.0
		unitHours := hours * float64(unitsRequired)

		// Property: Per-hour ROI = net / hours
		expectedPerHour := netReward / hours
		actualPerHour := m.NetAverageRewardPerHour()
		if abs(expectedPerHour-actualPerHour) > 0.01 {
			t.Errorf("PerHour ROI: expected %.2f, got %.2f", expectedPerHour, actualPerHour)
		}

		// Property: Per-unit-hour ROI = net / unitHours
		expectedPerUnitHour := netReward / unitHours
		actualPerUnitHour := m.NetAverageRewardPerUnitHour()
		if abs(expectedPerUnitHour-actualPerUnitHour) > 0.01 {
			t.Errorf("PerUnitHour ROI: expected %.2f, got %.2f", expectedPerUnitHour, actualPerUnitHour)
		}

		// Property: PerHour * hours = PerUnitHour * unitHours = net reward
		if abs(actualPerHour*hours-netReward) > 0.01 {
			t.Errorf("PerHour * hours = %.2f, want %.2f", actualPerHour*hours, netReward)
		}
		if abs(actualPerUnitHour*unitHours-netReward) > 0.01 {
			t.Errorf("PerUnitHour * unitHours = %.2f, want %.2f", actualPerUnitHour*unitHours, netReward)
		}
	})
}
