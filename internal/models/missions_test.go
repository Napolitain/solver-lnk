package models

import (
	"testing"
)

// Sample missions for testing - based on actual game data
func sampleMissions() []*Mission {
	return []*Mission{
		{
			Name:            "Hunting",
			DurationMinutes: 15,
			TavernLevel:     1,
			UnitsRequired: []UnitRequirement{
				{Type: Archer, Count: 15},
			},
			ResourceCosts: Costs{},
			Rewards: []ResourceReward{
				{Type: Wood, Min: 10, Max: 20},
				{Type: Stone, Min: 10, Max: 20},
				{Type: Iron, Min: 10, Max: 20},
			},
		},
		{
			Name:            "Chop Wood",
			DurationMinutes: 30,
			TavernLevel:     2,
			UnitsRequired: []UnitRequirement{
				{Type: Spearman, Count: 20},
				{Type: Swordsman, Count: 20},
			},
			ResourceCosts: Costs{},
			Rewards: []ResourceReward{
				{Type: Wood, Min: 50, Max: 100},
			},
		},
		{
			Name:            "Create Trading Post",
			DurationMinutes: 300, // 5 hours
			TavernLevel:     5,
			UnitsRequired: []UnitRequirement{
				{Type: Spearman, Count: 70},
				{Type: Swordsman, Count: 70},
				{Type: Archer, Count: 70},
				{Type: Horseman, Count: 30},
			},
			ResourceCosts: Costs{
				Wood:  700,
				Stone: 700,
				Iron:  700,
			},
			Rewards: []ResourceReward{
				{Type: Wood, Min: 750, Max: 1500},
				{Type: Stone, Min: 750, Max: 1500},
				{Type: Iron, Min: 1250, Max: 2500},
			},
		},
		{
			Name:            "Castle Festival",
			DurationMinutes: 720, // 12 hours
			TavernLevel:     10,
			UnitsRequired: []UnitRequirement{
				{Type: Spearman, Count: 100},
				{Type: Swordsman, Count: 100},
				{Type: Archer, Count: 100},
				{Type: Horseman, Count: 100},
			},
			ResourceCosts: Costs{},
			Rewards: []ResourceReward{
				{Type: Wood, Min: 800, Max: 1000},
				{Type: Stone, Min: 800, Max: 1000},
				{Type: Iron, Min: 1600, Max: 2000},
			},
		},
	}
}

// Property: Average reward should always be between min and max
func TestMission_AverageRewardInRange(t *testing.T) {
	for _, m := range sampleMissions() {
		for _, r := range m.Rewards {
			avg := r.AverageReward()
			if avg < float64(r.Min) || avg > float64(r.Max) {
				t.Errorf("%s: average reward %.2f not in range [%d, %d]",
					m.Name, avg, r.Min, r.Max)
			}
		}
	}
}

// Property: Total units required should be sum of all requirements
func TestMission_TotalUnitsRequired(t *testing.T) {
	for _, m := range sampleMissions() {
		expected := 0
		for _, req := range m.UnitsRequired {
			expected += req.Count
		}
		got := m.TotalUnitsRequired()
		if got != expected {
			t.Errorf("%s: TotalUnitsRequired() = %d, want %d", m.Name, got, expected)
		}
	}
}

// Property: Net reward per hour should be positive for worthwhile missions
func TestMission_NetRewardPerHour_Positive(t *testing.T) {
	// Hunting should always be profitable (no costs)
	hunting := sampleMissions()[0]
	if hunting.NetAverageRewardPerHour() <= 0 {
		t.Errorf("Hunting should have positive net reward per hour, got %.2f",
			hunting.NetAverageRewardPerHour())
	}
}

// Property: Net reward should equal total reward minus costs
func TestMission_NetRewardCalculation(t *testing.T) {
	for _, m := range sampleMissions() {
		totalReward := m.AverageTotalReward()
		totalCost := 0.0
		for _, cost := range m.ResourceCosts {
			totalCost += float64(cost)
		}
		expected := totalReward - totalCost
		got := m.NetAverageReward()

		if abs(got-expected) > 0.01 {
			t.Errorf("%s: NetAverageReward() = %.2f, want %.2f", m.Name, got, expected)
		}
	}
}

// Property: Missions with zero duration should have zero rate
func TestMission_ZeroDuration(t *testing.T) {
	m := &Mission{
		Name:            "Zero",
		DurationMinutes: 0,
		Rewards:         []ResourceReward{{Type: Wood, Min: 100, Max: 100}},
	}

	if m.NetAverageRewardPerHour() != 0 {
		t.Errorf("Zero duration mission should have 0 rate, got %.2f",
			m.NetAverageRewardPerHour())
	}

	if m.NetAverageRewardPerUnitHour() != 0 {
		t.Errorf("Zero duration mission should have 0 unit rate, got %.2f",
			m.NetAverageRewardPerUnitHour())
	}
}

// Property: Mission scheduler should never have negative available units
func TestMissionScheduler_NonNegativeUnits(t *testing.T) {
	missions := sampleMissions()
	units := map[UnitType]int{
		Spearman:  100,
		Swordsman: 100,
		Archer:    100,
		Horseman:  100,
	}

	scheduler := NewMissionScheduler(missions, units)

	// Start hunting mission
	hunting := missions[0]
	scheduler.StartMission(hunting, 0)

	// Check all units are non-negative
	for ut, count := range scheduler.AvailableUnits {
		if count < 0 {
			t.Errorf("Available units for %s went negative: %d", ut, count)
		}
	}
}

// Property: Starting a mission should reduce available units
func TestMissionScheduler_UnitsReserved(t *testing.T) {
	missions := sampleMissions()
	units := map[UnitType]int{
		Archer: 50,
	}

	scheduler := NewMissionScheduler(missions, units)
	hunting := missions[0] // Requires 15 archers

	beforeArchers := scheduler.AvailableUnits[Archer]
	scheduler.StartMission(hunting, 0)
	afterArchers := scheduler.AvailableUnits[Archer]

	expectedReduction := 15
	actualReduction := beforeArchers - afterArchers

	if actualReduction != expectedReduction {
		t.Errorf("Expected %d archers reserved, got %d", expectedReduction, actualReduction)
	}
}

// Property: Completing a mission should return units
func TestMissionScheduler_UnitsReturned(t *testing.T) {
	missions := sampleMissions()
	units := map[UnitType]int{
		Archer: 50,
	}

	scheduler := NewMissionScheduler(missions, units)
	hunting := missions[0] // 15 min duration

	scheduler.StartMission(hunting, 0)
	
	// Before completion
	if scheduler.AvailableUnits[Archer] != 35 {
		t.Errorf("Expected 35 available archers during mission, got %d",
			scheduler.AvailableUnits[Archer])
	}

	// After completion (at 15 minutes = 900 seconds)
	completed := scheduler.CompleteMissions(900)
	
	if len(completed) != 1 {
		t.Errorf("Expected 1 completed mission, got %d", len(completed))
	}

	if scheduler.AvailableUnits[Archer] != 50 {
		t.Errorf("Expected 50 available archers after completion, got %d",
			scheduler.AvailableUnits[Archer])
	}
}

// Property: CanStartMission should return false when not enough units
func TestMissionScheduler_CanStartMission(t *testing.T) {
	missions := sampleMissions()
	units := map[UnitType]int{
		Archer: 10, // Not enough for hunting (needs 15)
	}

	scheduler := NewMissionScheduler(missions, units)
	hunting := missions[0]

	if scheduler.CanStartMission(hunting) {
		t.Error("Should not be able to start mission without enough units")
	}
}

// Property: NextCompletionTime should return earliest completion
func TestMissionScheduler_NextCompletionTime(t *testing.T) {
	missions := sampleMissions()
	units := map[UnitType]int{
		Spearman:  200,
		Swordsman: 200,
		Archer:    200,
		Horseman:  200,
	}

	scheduler := NewMissionScheduler(missions, units)

	// Start hunting (15 min) at t=0
	scheduler.StartMission(missions[0], 0)
	// Start chop wood (30 min) at t=100
	scheduler.StartMission(missions[1], 100)

	// Next completion should be hunting at t=900 (15*60)
	next := scheduler.NextCompletionTime()
	if next != 900 {
		t.Errorf("Expected next completion at 900, got %d", next)
	}
}

// Property: No running missions should return -1 for next completion
func TestMissionScheduler_NoRunningMissions(t *testing.T) {
	scheduler := NewMissionScheduler([]*Mission{}, map[UnitType]int{})
	
	if scheduler.NextCompletionTime() != -1 {
		t.Errorf("Expected -1 for no running missions, got %d",
			scheduler.NextCompletionTime())
	}
}

// Property: Multiple missions can run simultaneously
func TestMissionScheduler_SimultaneousMissions(t *testing.T) {
	missions := sampleMissions()
	units := map[UnitType]int{
		Spearman:  200,
		Swordsman: 200,
		Archer:    200,
		Horseman:  200,
	}

	scheduler := NewMissionScheduler(missions, units)

	// Start all missions at once
	started := 0
	for _, m := range missions {
		if scheduler.CanStartMission(m) {
			state := scheduler.StartMission(m, 0)
			if state != nil {
				started++
			}
		}
	}

	if len(scheduler.RunningMissions) != started {
		t.Errorf("Expected %d running missions, got %d", 
			started, len(scheduler.RunningMissions))
	}
}

// ROI Comparison tests

// Property: Mission ROI should be comparable to production building ROI
func TestMission_ROIComparison(t *testing.T) {
	hunting := sampleMissions()[0]
	
	// Hunting: ~45 resources avg in 15 min = 180 resources/hour
	// Uses 15 archers
	// Per-unit-hour: 180/15 = 12 resources per unit per hour
	
	roiPerUnitHour := hunting.NetAverageRewardPerUnitHour()
	
	// This should be a meaningful positive number
	if roiPerUnitHour <= 0 {
		t.Errorf("Hunting ROI per unit hour should be positive, got %.2f", roiPerUnitHour)
	}
	
	// Log for reference (not a test failure)
	t.Logf("Hunting ROI: %.2f resources per unit-hour", roiPerUnitHour)
}

// Property: Missions with costs can have negative ROI for specific resources
func TestMission_SpecificResourceROI(t *testing.T) {
	tradingPost := sampleMissions()[2] // Has 700 wood cost, 750-1500 wood reward

	woodROI := tradingPost.NetRewardByType(Wood)
	
	// Average wood reward: 1125, cost: 700, net: 425
	if woodROI <= 0 {
		t.Logf("Trading post wood ROI: %.2f (reward: %.2f, cost: %d)",
			woodROI, tradingPost.AverageRewardByType(Wood), tradingPost.ResourceCosts[Wood])
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
