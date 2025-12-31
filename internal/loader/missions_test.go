package loader

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/models"
)

func TestLoadMissions(t *testing.T) {
	missions := LoadMissions()
	
	if len(missions) == 0 {
		t.Fatal("No missions loaded")
	}
	
	t.Logf("Loaded %d missions", len(missions))
	
	// Verify all missions have required fields
	for _, m := range missions {
		if m.Name == "" {
			t.Error("Mission has empty name")
		}
		if m.DurationMinutes <= 0 {
			t.Errorf("Mission %s has invalid duration: %d", m.Name, m.DurationMinutes)
		}
		if m.TavernLevel <= 0 {
			t.Errorf("Mission %s has invalid tavern level: %d", m.Name, m.TavernLevel)
		}
		if len(m.UnitsRequired) == 0 {
			t.Errorf("Mission %s has no unit requirements", m.Name)
		}
		if len(m.Rewards) == 0 {
			t.Errorf("Mission %s has no rewards", m.Name)
		}
	}
}

func TestGetMissionsForTavernLevel(t *testing.T) {
	tests := []struct {
		level    int
		minCount int
	}{
		{1, 4},  // Overtime missions + Hunting
		{2, 7},  // +3 more (Chop Wood, Help Stone, Mandatory)
		{5, 11}, // Several more
		{10, 16}, // All missions
	}
	
	for _, tc := range tests {
		available := GetMissionsForTavernLevel(tc.level)
		if len(available) < tc.minCount {
			t.Errorf("Tavern level %d: expected at least %d missions, got %d",
				tc.level, tc.minCount, len(available))
		}
	}
}

func TestMissionROI(t *testing.T) {
	missions := LoadMissions()
	
	t.Log("Mission ROI Analysis:")
	t.Log("---------------------")
	
	for _, m := range missions {
		perHour := m.NetAverageRewardPerHour()
		perUnitHour := m.NetAverageRewardPerUnitHour()
		
		t.Logf("%-25s Tavern %2d | %3d min | %3d units | %.0f res/h | %.2f res/unit-h",
			m.Name, m.TavernLevel, m.DurationMinutes, m.TotalUnitsRequired(),
			perHour, perUnitHour)
	}
}

func TestGetBestMissionForBottleneck(t *testing.T) {
	// Test iron bottleneck at different tavern levels
	tests := []struct {
		tavernLevel int
		bottleneck  models.ResourceType
		expectName  string
	}{
		{1, models.Iron, "Overtime Ore"},      // Only overtime available
		{6, models.Iron, "Feed Miners"},       // Feed Miners is iron-focused
	}
	
	for _, tc := range tests {
		best := GetBestMissionForBottleneck(tc.tavernLevel, tc.bottleneck)
		if best == nil {
			t.Errorf("No mission found for %s at tavern %d", tc.bottleneck, tc.tavernLevel)
			continue
		}
		t.Logf("Best %s mission at Tavern %d: %s (%.0f %s/hour)",
			tc.bottleneck, tc.tavernLevel, best.Name,
			best.NetRewardByType(tc.bottleneck)/float64(best.DurationMinutes)*60,
			tc.bottleneck)
	}
}

func TestMissionInvestmentAnalysis(t *testing.T) {
	// Calculate ROI for early game mission investment
	// Question: Is it worth building Tavern 1 + training 15 archers for Hunting?
	
	hunting := GetMissionsForTavernLevel(1)[3] // Hunting
	if hunting.Name != "Hunting" {
		// Find hunting
		for _, m := range GetMissionsForTavernLevel(1) {
			if m.Name == "Hunting" {
				hunting = m
				break
			}
		}
	}
	
	// Investment costs (from game data - simplified)
	// Tavern 1: ~450 total resources, ~1800 seconds build time
	tavernCost := 450.0
	tavernBuildTime := 1800 // 30 min
	
	// 15 Archers: 78 resources each, 900 seconds training each (sequential)
	archerCount := 15
	archerCostEach := 78.0
	archerTrainEach := 900 // 15 min
	
	totalCost := tavernCost + float64(archerCount)*archerCostEach
	totalSetupTime := tavernBuildTime + archerCount*archerTrainEach // Sequential
	
	// Hunting: 15 min, ~45 resources
	huntingRate := hunting.NetAverageRewardPerHour()
	
	// Break-even: when does hunting recover the investment?
	breakEvenHours := totalCost / huntingRate
	totalBreakEvenMinutes := float64(totalSetupTime)/60 + breakEvenHours*60
	
	t.Log("Hunting Investment Analysis:")
	t.Logf("  Setup cost: %.0f resources", totalCost)
	t.Logf("  Setup time: %d minutes", totalSetupTime/60)
	t.Logf("  Hunting rate: %.0f resources/hour", huntingRate)
	t.Logf("  Break-even: %.1f hours (%.0f minutes total)", breakEvenHours, totalBreakEvenMinutes)
	
	// For reference: that's step ~N in a 59.6 day build order
	// 59.6 days = 85824 minutes total
	// Break-even at ~270 minutes = step 270/85824 ≈ 0.3% into build
	buildOrderMinutes := 59.6 * 24 * 60
	breakEvenPercent := totalBreakEvenMinutes / buildOrderMinutes * 100
	t.Logf("  Break-even at step %.0f/%.0f (%.1f%% into build)",
		totalBreakEvenMinutes, buildOrderMinutes, breakEvenPercent)
}
