package castle

import (
	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
)

// MissionState tracks mission-related state during simulation
type MissionState struct {
	// Units available for missions (not busy)
	AvailableUnits map[models.UnitType]int
	
	// Total units owned
	TotalUnits map[models.UnitType]int
	
	// Running missions
	RunningMissions []*RunningMission
	
	// Completed mission actions (for timeline)
	CompletedMissions []MissionAction
	
	// All available missions (loaded once)
	AllMissions []*models.Mission
}

// RunningMission tracks a mission in progress
type RunningMission struct {
	Mission   *models.Mission
	StartTime int // minutes
	EndTime   int // minutes
	Units     map[models.UnitType]int
}

// MissionAction records a completed mission for the timeline
type MissionAction struct {
	MissionName    string
	StartTime      int // seconds
	EndTime        int // seconds
	ResourcesGained models.Costs
	UnitsUsed      map[models.UnitType]int
}

// NewMissionState creates a new mission state
func NewMissionState() *MissionState {
	return &MissionState{
		AvailableUnits:    make(map[models.UnitType]int),
		TotalUnits:        make(map[models.UnitType]int),
		RunningMissions:   make([]*RunningMission, 0),
		CompletedMissions: make([]MissionAction, 0),
		AllMissions:       loader.LoadMissions(),
	}
}

// GetAvailableMissions returns missions that can be started given tavern level and units
func (ms *MissionState) GetAvailableMissions(tavernLevel int) []*models.Mission {
	available := make([]*models.Mission, 0)
	
	for _, m := range ms.AllMissions {
		if m.TavernLevel > tavernLevel {
			continue
		}
		if ms.CanStartMission(m) {
			available = append(available, m)
		}
	}
	
	return available
}

// CanStartMission checks if we have enough available units
func (ms *MissionState) CanStartMission(m *models.Mission) bool {
	for _, req := range m.UnitsRequired {
		if ms.AvailableUnits[req.Type] < req.Count {
			return false
		}
	}
	return true
}

// StartMission starts a mission, reserving units
func (ms *MissionState) StartMission(m *models.Mission, currentTimeMinutes int) *RunningMission {
	if !ms.CanStartMission(m) {
		return nil
	}
	
	// Reserve units
	units := make(map[models.UnitType]int)
	for _, req := range m.UnitsRequired {
		ms.AvailableUnits[req.Type] -= req.Count
		units[req.Type] = req.Count
	}
	
	running := &RunningMission{
		Mission:   m,
		StartTime: currentTimeMinutes,
		EndTime:   currentTimeMinutes + m.DurationMinutes,
		Units:     units,
	}
	
	ms.RunningMissions = append(ms.RunningMissions, running)
	return running
}

// CompleteMissions completes any missions that have finished, returning resources and units
func (ms *MissionState) CompleteMissions(currentTimeMinutes int) []MissionAction {
	completed := make([]MissionAction, 0)
	stillRunning := make([]*RunningMission, 0)
	
	for _, running := range ms.RunningMissions {
		if currentTimeMinutes >= running.EndTime {
			// Return units
			for ut, count := range running.Units {
				ms.AvailableUnits[ut] += count
			}
			
			// Calculate rewards (use average)
			rewards := make(models.Costs)
			for _, r := range running.Mission.Rewards {
				rewards[r.Type] = int(r.AverageReward())
			}
			
			action := MissionAction{
				MissionName:     running.Mission.Name,
				StartTime:       running.StartTime * 60,
				EndTime:         running.EndTime * 60,
				ResourcesGained: rewards,
				UnitsUsed:       running.Units,
			}
			
			completed = append(completed, action)
			ms.CompletedMissions = append(ms.CompletedMissions, action)
		} else {
			stillRunning = append(stillRunning, running)
		}
	}
	
	ms.RunningMissions = stillRunning
	return completed
}

// NextMissionCompletionTime returns when the next mission completes, or -1 if none
func (ms *MissionState) NextMissionCompletionTime() int {
	if len(ms.RunningMissions) == 0 {
		return -1
	}
	
	minTime := ms.RunningMissions[0].EndTime
	for _, running := range ms.RunningMissions[1:] {
		if running.EndTime < minTime {
			minTime = running.EndTime
		}
	}
	return minTime
}

// TrainUnits adds units to the pool (called when arsenal trains units)
func (ms *MissionState) TrainUnits(unitType models.UnitType, count int) {
	ms.TotalUnits[unitType] += count
	ms.AvailableUnits[unitType] += count
}

// GetBestMissionForState returns the best mission to run given current state
// Considers bottleneck resources and ROI
func (ms *MissionState) GetBestMissionForState(
	tavernLevel int,
	currentResources map[models.ResourceType]float64,
	neededResources map[models.ResourceType]float64,
	productionRates map[models.ResourceType]float64,
) *models.Mission {
	available := ms.GetAvailableMissions(tavernLevel)
	if len(available) == 0 {
		return nil
	}
	
	// Find bottleneck resource (which one takes longest to accumulate?)
	var bottleneck models.ResourceType
	var maxWaitTime float64
	
	for rt, needed := range neededResources {
		if needed <= 0 {
			continue
		}
		current := currentResources[rt]
		if current >= needed {
			continue
		}
		
		deficit := needed - current
		rate := productionRates[rt]
		if rate <= 0 {
			// Can't produce this resource, it's definitely the bottleneck
			bottleneck = rt
			// Skip setting maxWaitTime as it will be recalculated below if needed
			break
		}
		
		waitTime := deficit / rate // hours
		if waitTime > maxWaitTime {
			maxWaitTime = waitTime
			bottleneck = rt
		}
	}
	
	// If no bottleneck, pick mission with best overall ROI
	if bottleneck == "" {
		var best *models.Mission
		var bestROI float64
		for _, m := range available {
			roi := m.NetAverageRewardPerHour()
			if roi > bestROI {
				bestROI = roi
				best = m
			}
		}
		return best
	}
	
	// Pick mission with best ROI for bottleneck resource
	var best *models.Mission
	var bestROI float64
	for _, m := range available {
		// ROI = net reward of bottleneck resource per hour
		roi := m.NetRewardByType(bottleneck) / (float64(m.DurationMinutes) / 60.0)
		if roi > bestROI {
			bestROI = roi
			best = m
		}
	}
	
	return best
}

// ShouldInvestInTavern decides if investing in tavern + units is worth it
// Returns true if the break-even time is less than remaining build time
func ShouldInvestInTavern(
	currentTavernLevel int,
	targetTavernLevel int,
	currentUnits map[models.UnitType]int,
	remainingBuildTimeMinutes int,
) bool {
	if currentTavernLevel >= targetTavernLevel {
		return false
	}
	
	// Simple heuristic: if we have more than 10% of build remaining, invest in tavern
	// The break-even analysis showed ~0.9% into build is break-even for Hunting
	// So anything beyond that is profitable
	
	// More sophisticated: calculate actual break-even based on available missions
	// For now, use simple threshold
	breakEvenMinutes := 795 // From test analysis
	
	return remainingBuildTimeMinutes > breakEvenMinutes*2 // 2x margin for safety
}
