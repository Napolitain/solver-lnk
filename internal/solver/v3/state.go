package v3

import "github.com/napolitain/solver-lnk/internal/models"

// QueueType identifies which queue an action belongs to
type QueueType int

const (
	QueueBuilding QueueType = iota
	QueueResearch
	QueueUnits    // For future extension
	QueueMissions // For future extension
)

// State represents the complete simulation state
type State struct {
	// Time in seconds
	Now int

	// Resources
	Resources       [4]float64 // Wood, Stone, Iron, Food (indexed by ResourceIndex)
	StorageCaps     [3]int     // Wood, Stone, Iron caps (no cap for Food)
	ProductionRates [3]float64 // Wood, Stone, Iron per hour
	ProductionBonus float64    // Multiplier from techs (1.0 = none)

	// Buildings
	BuildingLevels      [13]int // Indexed by BuildingIndex
	BuildingQueueFreeAt int

	// Research
	ResearchedTechs     map[string]bool
	ResearchQueueFreeAt int

	// Food/Workers
	FoodCapacity int
	FoodUsed     int

	// Future: Units and Missions
	// Units map[UnitType]int
	// AvailableUnits map[UnitType]int
	// UnitQueueFreeAt int
	// RunningMissions map[string]*MissionState
}

// ResourceIndex maps ResourceType to array index
type ResourceIndex int

const (
	IdxWood ResourceIndex = iota
	IdxStone
	IdxIron
	IdxFood
)

// BuildingIndex maps BuildingType to array index
type BuildingIndex int

const (
	IdxLumberjack BuildingIndex = iota
	IdxQuarry
	IdxOreMine
	IdxFarm
	IdxWoodStore
	IdxStoneStore
	IdxOreStore
	IdxKeep
	IdxArsenal
	IdxLibrary
	IdxTavern
	IdxMarket
	IdxFortifications
)

// ResourceTypeToIndex converts ResourceType to index
func ResourceTypeToIndex(rt models.ResourceType) ResourceIndex {
	switch rt {
	case models.Wood:
		return IdxWood
	case models.Stone:
		return IdxStone
	case models.Iron:
		return IdxIron
	case models.Food:
		return IdxFood
	default:
		return IdxWood
	}
}

// BuildingTypeToIndex converts BuildingType to index
func BuildingTypeToIndex(bt models.BuildingType) BuildingIndex {
	switch bt {
	case models.Lumberjack:
		return IdxLumberjack
	case models.Quarry:
		return IdxQuarry
	case models.OreMine:
		return IdxOreMine
	case models.Farm:
		return IdxFarm
	case models.WoodStore:
		return IdxWoodStore
	case models.StoneStore:
		return IdxStoneStore
	case models.OreStore:
		return IdxOreStore
	case models.Keep:
		return IdxKeep
	case models.Arsenal:
		return IdxArsenal
	case models.Library:
		return IdxLibrary
	case models.Tavern:
		return IdxTavern
	case models.Market:
		return IdxMarket
	case models.Fortifications:
		return IdxFortifications
	default:
		return IdxKeep
	}
}

// IndexToBuildingType converts index back to BuildingType
func IndexToBuildingType(idx BuildingIndex) models.BuildingType {
	types := []models.BuildingType{
		models.Lumberjack, models.Quarry, models.OreMine, models.Farm,
		models.WoodStore, models.StoneStore, models.OreStore,
		models.Keep, models.Arsenal, models.Library, models.Tavern,
		models.Market, models.Fortifications,
	}
	if int(idx) < len(types) {
		return types[idx]
	}
	return models.Keep
}

// IndexToResourceType converts index back to ResourceType
func IndexToResourceType(idx ResourceIndex) models.ResourceType {
	types := []models.ResourceType{models.Wood, models.Stone, models.Iron, models.Food}
	if int(idx) < len(types) {
		return types[idx]
	}
	return models.Wood
}

// NewState creates a new simulation state from initial game state
func NewState(initial *models.GameState) *State {
	s := &State{
		Now:             0,
		ProductionBonus: 1.0,
		ResearchedTechs: make(map[string]bool),
	}

	// Copy resources
	for rt, amount := range initial.Resources {
		s.Resources[ResourceTypeToIndex(rt)] = amount
	}

	// Copy building levels (default to 1)
	for i := range s.BuildingLevels {
		s.BuildingLevels[i] = 1
	}
	for bt, level := range initial.BuildingLevels {
		s.BuildingLevels[BuildingTypeToIndex(bt)] = level
	}

	// Copy researched technologies
	for k, v := range initial.ResearchedTechnologies {
		s.ResearchedTechs[k] = v
		if v && (k == "Beer tester" || k == "Wheelbarrow") {
			s.ProductionBonus += 0.05
		}
	}

	return s
}

// GetResource returns current amount of a resource
func (s *State) GetResource(rt models.ResourceType) float64 {
	return s.Resources[ResourceTypeToIndex(rt)]
}

// SetResource sets the amount of a resource
func (s *State) SetResource(rt models.ResourceType, amount float64) {
	s.Resources[ResourceTypeToIndex(rt)] = amount
}

// GetBuildingLevel returns current level of a building
func (s *State) GetBuildingLevel(bt models.BuildingType) int {
	return s.BuildingLevels[BuildingTypeToIndex(bt)]
}

// SetBuildingLevel sets the level of a building
func (s *State) SetBuildingLevel(bt models.BuildingType, level int) {
	s.BuildingLevels[BuildingTypeToIndex(bt)] = level
}

// GetStorageCap returns storage capacity for a resource (0 for Food)
func (s *State) GetStorageCap(rt models.ResourceType) int {
	idx := ResourceTypeToIndex(rt)
	if idx == IdxFood {
		return 0 // Food has no storage cap
	}
	return s.StorageCaps[idx]
}

// SetStorageCap sets storage capacity for a resource
func (s *State) SetStorageCap(rt models.ResourceType, cap int) {
	idx := ResourceTypeToIndex(rt)
	if idx < 3 {
		s.StorageCaps[idx] = cap
	}
}

// GetProductionRate returns production rate for a resource (0 for Food)
func (s *State) GetProductionRate(rt models.ResourceType) float64 {
	idx := ResourceTypeToIndex(rt)
	if idx == IdxFood {
		return 0 // Food is not produced
	}
	return s.ProductionRates[idx]
}

// SetProductionRate sets production rate for a resource
func (s *State) SetProductionRate(rt models.ResourceType, rate float64) {
	idx := ResourceTypeToIndex(rt)
	if idx < 3 {
		s.ProductionRates[idx] = rate
	}
}

// Clone creates a deep copy of the state
func (s *State) Clone() *State {
	cp := &State{
		Now:                 s.Now,
		Resources:           s.Resources,
		StorageCaps:         s.StorageCaps,
		ProductionRates:     s.ProductionRates,
		ProductionBonus:     s.ProductionBonus,
		BuildingLevels:      s.BuildingLevels,
		BuildingQueueFreeAt: s.BuildingQueueFreeAt,
		ResearchQueueFreeAt: s.ResearchQueueFreeAt,
		FoodCapacity:        s.FoodCapacity,
		FoodUsed:            s.FoodUsed,
		ResearchedTechs:     make(map[string]bool, len(s.ResearchedTechs)),
	}
	for k, v := range s.ResearchedTechs {
		cp.ResearchedTechs[k] = v
	}
	return cp
}

// ToGameState converts State back to models.GameState
func (s *State) ToGameState() *models.GameState {
	gs := models.NewGameState()

	for i, v := range s.Resources {
		gs.Resources[IndexToResourceType(ResourceIndex(i))] = v
	}

	for i, v := range s.BuildingLevels {
		gs.BuildingLevels[IndexToBuildingType(BuildingIndex(i))] = v
	}

	for i, v := range s.StorageCaps {
		gs.StorageCaps[IndexToResourceType(ResourceIndex(i))] = v
	}

	for i, v := range s.ProductionRates {
		gs.ProductionRates[IndexToResourceType(ResourceIndex(i))] = v
	}

	for k, v := range s.ResearchedTechs {
		gs.ResearchedTechnologies[k] = v
	}

	return gs
}
