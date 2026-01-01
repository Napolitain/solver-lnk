package models

// UnitType represents the different unit types for missions
// TODO: This should eventually be unified with units package
type UnitType string

const (
	Spearman    UnitType = "spearman"
	Swordsman   UnitType = "swordsman"
	Archer      UnitType = "archer"
	Crossbowman UnitType = "crossbowman"
	Horseman    UnitType = "horseman"
	Lancer      UnitType = "lancer"
)

// AllUnitTypes returns all combat unit types (excluding transport)
func AllUnitTypes() []UnitType {
	return []UnitType{Spearman, Swordsman, Archer, Crossbowman, Horseman, Lancer}
}

// ResourceReward represents a range of resources that can be rewarded
type ResourceReward struct {
	Type ResourceType
	Min  int
	Max  int
}

// AverageReward returns the average reward for this resource
func (r ResourceReward) AverageReward() float64 {
	return float64(r.Min+r.Max) / 2.0
}

// UnitRequirement represents units needed for a mission
type UnitRequirement struct {
	Type  UnitType
	Count int
}

// Mission represents a tavern mission
type Mission struct {
	Name            string
	Description     string
	DurationMinutes int
	TavernLevel     int // Required tavern level to unlock this mission

	// Units required to run this mission (they are busy during mission)
	UnitsRequired []UnitRequirement

	// Resources consumed when starting the mission
	ResourceCosts Costs

	// Rewards received upon completion
	Rewards []ResourceReward
}

// TotalUnitsRequired returns the total number of units needed
func (m *Mission) TotalUnitsRequired() int {
	total := 0
	for _, req := range m.UnitsRequired {
		total += req.Count
	}
	return total
}

// AverageTotalReward returns the average total resources rewarded
func (m *Mission) AverageTotalReward() float64 {
	total := 0.0
	for _, r := range m.Rewards {
		total += r.AverageReward()
	}
	return total
}

// NetAverageReward returns average reward minus resource costs
func (m *Mission) NetAverageReward() float64 {
	reward := m.AverageTotalReward()
	reward -= float64(m.ResourceCosts.Wood)
	reward -= float64(m.ResourceCosts.Stone)
	reward -= float64(m.ResourceCosts.Iron)
	reward -= float64(m.ResourceCosts.Food)
	return reward
}

// NetAverageRewardPerHour returns net reward per hour
func (m *Mission) NetAverageRewardPerHour() float64 {
	if m.DurationMinutes == 0 {
		return 0
	}
	return m.NetAverageReward() / (float64(m.DurationMinutes) / 60.0)
}

// NetAverageRewardPerUnitHour returns net reward per unit-hour
// This measures efficiency: how much resource gain per unit of time a single unit is busy
func (m *Mission) NetAverageRewardPerUnitHour() float64 {
	if m.DurationMinutes == 0 || m.TotalUnitsRequired() == 0 {
		return 0
	}
	hours := float64(m.DurationMinutes) / 60.0
	unitHours := hours * float64(m.TotalUnitsRequired())
	return m.NetAverageReward() / unitHours
}

// AverageRewardByType returns the average reward for a specific resource type
func (m *Mission) AverageRewardByType(rt ResourceType) float64 {
	for _, r := range m.Rewards {
		if r.Type == rt {
			return r.AverageReward()
		}
	}
	return 0
}

// NetRewardByType returns net reward (reward - cost) for a specific resource type
func (m *Mission) NetRewardByType(rt ResourceType) float64 {
	reward := m.AverageRewardByType(rt)
	reward -= float64(m.ResourceCosts.Get(rt))
	return reward
}

// MissionState tracks the state of a running mission
type MissionState struct {
	Mission       *Mission
	StartTime     int // seconds from simulation start
	EndTime       int // seconds when mission completes
	AssignedUnits []UnitRequirement
}

// IsComplete returns true if the mission has finished at the given time
func (ms *MissionState) IsComplete(currentTime int) bool {
	return currentTime >= ms.EndTime
}

// MissionScheduler tracks all missions and unit availability
type MissionScheduler struct {
	AvailableMissions []*Mission
	RunningMissions   []*MissionState
	AvailableUnits    map[UnitType]int // Units not currently on missions
	TotalUnits        map[UnitType]int // All units owned
}

// NewMissionScheduler creates a new scheduler
func NewMissionScheduler(missions []*Mission, units map[UnitType]int) *MissionScheduler {
	available := make(map[UnitType]int)
	for ut, count := range units {
		available[ut] = count
	}
	return &MissionScheduler{
		AvailableMissions: missions,
		RunningMissions:   make([]*MissionState, 0),
		AvailableUnits:    available,
		TotalUnits:        units,
	}
}

// CanStartMission checks if a mission can be started with current available units
func (ms *MissionScheduler) CanStartMission(m *Mission) bool {
	for _, req := range m.UnitsRequired {
		if ms.AvailableUnits[req.Type] < req.Count {
			return false
		}
	}
	return true
}

// StartMission starts a mission if possible, returns the mission state
func (ms *MissionScheduler) StartMission(m *Mission, currentTime int) *MissionState {
	if !ms.CanStartMission(m) {
		return nil
	}

	// Reserve units
	for _, req := range m.UnitsRequired {
		ms.AvailableUnits[req.Type] -= req.Count
	}

	state := &MissionState{
		Mission:       m,
		StartTime:     currentTime,
		EndTime:       currentTime + m.DurationMinutes*60,
		AssignedUnits: m.UnitsRequired,
	}

	ms.RunningMissions = append(ms.RunningMissions, state)
	return state
}

// CompleteMissions checks for completed missions and returns units
func (ms *MissionScheduler) CompleteMissions(currentTime int) []*MissionState {
	completed := make([]*MissionState, 0)
	stillRunning := make([]*MissionState, 0)

	for _, state := range ms.RunningMissions {
		if state.IsComplete(currentTime) {
			// Return units
			for _, req := range state.AssignedUnits {
				ms.AvailableUnits[req.Type] += req.Count
			}
			completed = append(completed, state)
		} else {
			stillRunning = append(stillRunning, state)
		}
	}

	ms.RunningMissions = stillRunning
	return completed
}

// NextCompletionTime returns the time when the next mission completes, or -1 if none
func (ms *MissionScheduler) NextCompletionTime() int {
	if len(ms.RunningMissions) == 0 {
		return -1
	}

	minTime := ms.RunningMissions[0].EndTime
	for _, state := range ms.RunningMissions[1:] {
		if state.EndTime < minTime {
			minTime = state.EndTime
		}
	}
	return minTime
}
