package castle

import "github.com/napolitain/solver-lnk/internal/models"

// State represents the complete simulation state
type State struct {
	// Time
	Now int // Current time in seconds

	// Queue busy-until times
	BuildingQueueFreeAt int
	ResearchQueueFreeAt int
	TrainingQueueFreeAt int

	// Pending actions (for completion handling)
	PendingBuilding *BuildingAction
	PendingResearch *ResearchAction
	PendingTraining *TrainUnitAction

	// Buildings
	BuildingLevels map[models.BuildingType]int

	// Resources (indexed: 0=Wood, 1=Stone, 2=Iron)
	Resources       [3]float64
	ProductionRates [3]float64
	StorageCaps     [3]int
	ProductionBonus float64 // Multiplier from Beer tester, Wheelbarrow (starts at 1.0)

	// Food
	FoodUsed     int
	FoodCapacity int

	// Research
	ResearchedTechs map[string]bool

	// Army (strict typing)
	Army           models.Army // Available units
	UnitsOnMission models.Army // Busy on missions

	// Missions
	RunningMissions []*models.MissionState
}

// NewState creates a new state from initial game state
func NewState(gs *models.GameState) *State {
	s := &State{
		Now:             0,
		BuildingLevels:  make(map[models.BuildingType]int),
		ProductionBonus: 1.0,
		ResearchedTechs: make(map[string]bool),
		RunningMissions: make([]*models.MissionState, 0),
	}

	// Copy building levels
	for bt, level := range gs.BuildingLevels {
		s.BuildingLevels[bt] = level
	}

	// Copy resources
	if wood, ok := gs.Resources[models.Wood]; ok {
		s.Resources[0] = wood
	}
	if stone, ok := gs.Resources[models.Stone]; ok {
		s.Resources[1] = stone
	}
	if iron, ok := gs.Resources[models.Iron]; ok {
		s.Resources[2] = iron
	}

	// Copy researched techs
	for tech, researched := range gs.ResearchedTechnologies {
		s.ResearchedTechs[tech] = researched
	}

	return s
}

// GetBuildingLevel returns the current level of a building
func (s *State) GetBuildingLevel(bt models.BuildingType) int {
	if level, ok := s.BuildingLevels[bt]; ok {
		return level
	}
	return 1 // Default to level 1
}

// SetBuildingLevel sets the level of a building
func (s *State) SetBuildingLevel(bt models.BuildingType, level int) {
	s.BuildingLevels[bt] = level
}

// GetResource returns the current amount of a resource
func (s *State) GetResource(rt models.ResourceType) float64 {
	switch rt {
	case models.Wood:
		return s.Resources[0]
	case models.Stone:
		return s.Resources[1]
	case models.Iron:
		return s.Resources[2]
	default:
		return 0
	}
}

// SetResource sets the amount of a resource
func (s *State) SetResource(rt models.ResourceType, amount float64) {
	switch rt {
	case models.Wood:
		s.Resources[0] = amount
	case models.Stone:
		s.Resources[1] = amount
	case models.Iron:
		s.Resources[2] = amount
	}
}

// AddResource adds to a resource amount
func (s *State) AddResource(rt models.ResourceType, amount float64) {
	switch rt {
	case models.Wood:
		s.Resources[0] += amount
	case models.Stone:
		s.Resources[1] += amount
	case models.Iron:
		s.Resources[2] += amount
	}

	// Cap at storage
	s.CapResources()
}

// CapResources ensures resources don't exceed storage caps
func (s *State) CapResources() {
	for i := 0; i < 3; i++ {
		if s.StorageCaps[i] > 0 && s.Resources[i] > float64(s.StorageCaps[i]) {
			s.Resources[i] = float64(s.StorageCaps[i])
		}
	}
}

// GetProductionRate returns the production rate for a resource
func (s *State) GetProductionRate(rt models.ResourceType) float64 {
	switch rt {
	case models.Wood:
		return s.ProductionRates[0]
	case models.Stone:
		return s.ProductionRates[1]
	case models.Iron:
		return s.ProductionRates[2]
	default:
		return 0
	}
}

// SetProductionRate sets the production rate for a resource
func (s *State) SetProductionRate(rt models.ResourceType, rate float64) {
	switch rt {
	case models.Wood:
		s.ProductionRates[0] = rate
	case models.Stone:
		s.ProductionRates[1] = rate
	case models.Iron:
		s.ProductionRates[2] = rate
	}
}

// GetStorageCap returns the storage capacity for a resource
func (s *State) GetStorageCap(rt models.ResourceType) int {
	switch rt {
	case models.Wood:
		return s.StorageCaps[0]
	case models.Stone:
		return s.StorageCaps[1]
	case models.Iron:
		return s.StorageCaps[2]
	default:
		return 0
	}
}

// SetStorageCap sets the storage capacity for a resource
func (s *State) SetStorageCap(rt models.ResourceType, cap int) {
	switch rt {
	case models.Wood:
		s.StorageCaps[0] = cap
	case models.Stone:
		s.StorageCaps[1] = cap
	case models.Iron:
		s.StorageCaps[2] = cap
	}
}

// AvailableFood returns remaining food capacity
func (s *State) AvailableFood() int {
	return s.FoodCapacity - s.FoodUsed
}

// CanAffordFood returns true if there's enough food for the cost
func (s *State) CanAffordFood(cost int) bool {
	return s.FoodUsed+cost <= s.FoodCapacity
}

// TotalArmy returns the combined available + on-mission army
func (s *State) TotalArmy() models.Army {
	return models.Army{
		Spearman:    s.Army.Spearman + s.UnitsOnMission.Spearman,
		Swordsman:   s.Army.Swordsman + s.UnitsOnMission.Swordsman,
		Archer:      s.Army.Archer + s.UnitsOnMission.Archer,
		Crossbowman: s.Army.Crossbowman + s.UnitsOnMission.Crossbowman,
		Horseman:    s.Army.Horseman + s.UnitsOnMission.Horseman,
		Lancer:      s.Army.Lancer + s.UnitsOnMission.Lancer,
		Handcart:    s.Army.Handcart + s.UnitsOnMission.Handcart,
		Oxcart:      s.Army.Oxcart + s.UnitsOnMission.Oxcart,
	}
}

// Clone creates a deep copy of the state
func (s *State) Clone() *State {
	clone := &State{
		Now:                 s.Now,
		BuildingQueueFreeAt: s.BuildingQueueFreeAt,
		ResearchQueueFreeAt: s.ResearchQueueFreeAt,
		TrainingQueueFreeAt: s.TrainingQueueFreeAt,
		BuildingLevels:      make(map[models.BuildingType]int),
		Resources:           s.Resources,
		ProductionRates:     s.ProductionRates,
		StorageCaps:         s.StorageCaps,
		ProductionBonus:     s.ProductionBonus,
		FoodUsed:            s.FoodUsed,
		FoodCapacity:        s.FoodCapacity,
		ResearchedTechs:     make(map[string]bool),
		Army:                s.Army.Clone(),
		UnitsOnMission:      s.UnitsOnMission.Clone(),
		RunningMissions:     make([]*models.MissionState, len(s.RunningMissions)),
	}

	for bt, level := range s.BuildingLevels {
		clone.BuildingLevels[bt] = level
	}
	for tech, researched := range s.ResearchedTechs {
		clone.ResearchedTechs[tech] = researched
	}
	copy(clone.RunningMissions, s.RunningMissions)

	return clone
}

// ToGameState converts back to models.GameState
func (s *State) ToGameState() *models.GameState {
	gs := models.NewGameState()

	for bt, level := range s.BuildingLevels {
		gs.BuildingLevels[bt] = level
	}

	gs.Resources[models.Wood] = s.Resources[0]
	gs.Resources[models.Stone] = s.Resources[1]
	gs.Resources[models.Iron] = s.Resources[2]

	for tech, researched := range s.ResearchedTechs {
		gs.ResearchedTechnologies[tech] = researched
	}

	return gs
}
