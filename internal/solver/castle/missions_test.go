package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/models"
)

func TestMissionState_Basic(t *testing.T) {
	ms := NewMissionState()
	
	// Initially no units
	if len(ms.AvailableUnits) != 0 {
		t.Error("Expected no units initially")
	}
	
	// Train some units
	ms.TrainUnits(models.Spearman, 10)
	ms.TrainUnits(models.Archer, 20)
	
	if ms.AvailableUnits[models.Spearman] != 10 {
		t.Errorf("Expected 10 spearmen, got %d", ms.AvailableUnits[models.Spearman])
	}
	if ms.TotalUnits[models.Archer] != 20 {
		t.Errorf("Expected 20 archers total, got %d", ms.TotalUnits[models.Archer])
	}
}

func TestMissionState_StartAndComplete(t *testing.T) {
	ms := NewMissionState()
	
	// Train enough for Hunting (15 archers)
	ms.TrainUnits(models.Archer, 20)
	
	// Get available missions at tavern level 1
	available := ms.GetAvailableMissions(1)
	
	// Find Hunting
	var hunting *models.Mission
	for _, m := range available {
		if m.Name == "Hunting" {
			hunting = m
			break
		}
	}
	
	if hunting == nil {
		t.Fatal("Hunting mission not found")
	}
	
	// Start mission at time 0
	running := ms.StartMission(hunting, 0)
	if running == nil {
		t.Fatal("Failed to start mission")
	}
	
	// Units should be reserved
	if ms.AvailableUnits[models.Archer] != 5 {
		t.Errorf("Expected 5 available archers (20-15), got %d", ms.AvailableUnits[models.Archer])
	}
	
	// Complete at time 15 (mission duration)
	completed := ms.CompleteMissions(15)
	
	if len(completed) != 1 {
		t.Fatalf("Expected 1 completed mission, got %d", len(completed))
	}
	
	// Units should be returned
	if ms.AvailableUnits[models.Archer] != 20 {
		t.Errorf("Expected 20 available archers after completion, got %d", ms.AvailableUnits[models.Archer])
	}
	
	// Check rewards
	if completed[0].ResourcesGained[models.Wood] < 10 {
		t.Errorf("Expected at least 10 wood reward, got %d", completed[0].ResourcesGained[models.Wood])
	}
}

func TestMissionState_CannotStartWithoutUnits(t *testing.T) {
	ms := NewMissionState()
	
	// No units trained
	available := ms.GetAvailableMissions(1)
	
	// Should have no available missions (no units)
	if len(available) != 0 {
		t.Errorf("Expected no available missions without units, got %d", len(available))
	}
}

func TestMissionState_MultipleMissionsSimultaneously(t *testing.T) {
	ms := NewMissionState()
	
	// Train lots of units
	ms.TrainUnits(models.Spearman, 100)
	ms.TrainUnits(models.Swordsman, 100)
	ms.TrainUnits(models.Archer, 100)
	
	// Start multiple overtime missions at tavern level 1
	available := ms.GetAvailableMissions(1)
	
	started := 0
	for _, m := range available {
		if ms.CanStartMission(m) {
			running := ms.StartMission(m, 0)
			if running != nil {
				started++
			}
		}
	}
	
	if started < 3 {
		t.Errorf("Expected to start at least 3 missions, started %d", started)
	}
	
	if len(ms.RunningMissions) != started {
		t.Errorf("Running missions count mismatch: %d vs %d", len(ms.RunningMissions), started)
	}
	
	t.Logf("Started %d missions simultaneously", started)
}

func TestMissionState_GetBestMissionForBottleneck(t *testing.T) {
	ms := NewMissionState()
	
	// Train units for various missions
	ms.TrainUnits(models.Spearman, 50)
	ms.TrainUnits(models.Archer, 50)
	
	// Scenario: We need iron, have plenty of wood/stone
	currentResources := map[models.ResourceType]float64{
		models.Wood:  5000,
		models.Stone: 5000,
		models.Iron:  100,
	}
	neededResources := map[models.ResourceType]float64{
		models.Iron: 1000, // Need 900 more iron
	}
	productionRates := map[models.ResourceType]float64{
		models.Wood:  100, // 100/hour
		models.Stone: 100,
		models.Iron:  50, // Only 50/hour - bottleneck!
	}
	
	best := ms.GetBestMissionForState(1, currentResources, neededResources, productionRates)
	
	if best == nil {
		t.Fatal("Expected a best mission, got nil")
	}
	
	// Should pick a mission that produces iron
	ironReward := best.NetRewardByType(models.Iron)
	t.Logf("Best mission for iron bottleneck: %s (%.0f iron)", best.Name, ironReward)
	
	if ironReward <= 0 {
		t.Error("Expected mission that produces iron")
	}
}

func TestMissionState_NextCompletionTime(t *testing.T) {
	ms := NewMissionState()
	ms.TrainUnits(models.Spearman, 20)
	
	// No missions running
	if ms.NextMissionCompletionTime() != -1 {
		t.Error("Expected -1 when no missions running")
	}
	
	// Start a 5-minute mission at time 10
	for _, m := range ms.AllMissions {
		if m.DurationMinutes == 5 && ms.CanStartMission(m) {
			ms.StartMission(m, 10)
			break
		}
	}
	
	// Next completion should be at 15
	if ms.NextMissionCompletionTime() != 15 {
		t.Errorf("Expected next completion at 15, got %d", ms.NextMissionCompletionTime())
	}
}

func TestShouldInvestInTavern(t *testing.T) {
	tests := []struct {
		remainingMinutes int
		expected         bool
	}{
		{100000, true},  // Lots of time remaining - invest
		{2000, true},    // More than 2x break-even - invest
		{1000, false},   // Less than 2x break-even - don't invest
		{500, false},    // Way less - don't invest
	}
	
	for _, tc := range tests {
		result := ShouldInvestInTavern(0, 1, nil, tc.remainingMinutes)
		if result != tc.expected {
			t.Errorf("ShouldInvestInTavern with %d min remaining: got %v, want %v",
				tc.remainingMinutes, result, tc.expected)
		}
	}
}

// Benchmark mission evaluation
func BenchmarkGetBestMission(b *testing.B) {
	ms := NewMissionState()
	ms.TrainUnits(models.Spearman, 100)
	ms.TrainUnits(models.Swordsman, 100)
	ms.TrainUnits(models.Archer, 100)
	ms.TrainUnits(models.Horseman, 100)
	
	currentResources := map[models.ResourceType]float64{
		models.Wood:  1000,
		models.Stone: 1000,
		models.Iron:  500,
	}
	neededResources := map[models.ResourceType]float64{
		models.Iron: 2000,
	}
	productionRates := map[models.ResourceType]float64{
		models.Wood:  100,
		models.Stone: 100,
		models.Iron:  50,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.GetBestMissionForState(10, currentResources, neededResources, productionRates)
	}
}
