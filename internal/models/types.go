package models

// ResourceType represents the different resource types in the game
type ResourceType string

const (
	Wood  ResourceType = "wood"
	Stone ResourceType = "stone"
	Iron  ResourceType = "iron"
	Food  ResourceType = "food"
)

// AllResourceTypes returns all resource types
func AllResourceTypes() []ResourceType {
	return []ResourceType{Wood, Stone, Iron, Food}
}

// BuildingType represents the different building types
type BuildingType string

const (
	Lumberjack     BuildingType = "lumberjack"
	Quarry         BuildingType = "quarry"
	OreMine        BuildingType = "ore_mine"
	Farm           BuildingType = "farm"
	WoodStore      BuildingType = "wood_store"
	StoneStore     BuildingType = "stone_store"
	OreStore       BuildingType = "ore_store"
	Keep           BuildingType = "keep"
	Arsenal        BuildingType = "arsenal"
	Library        BuildingType = "library"
	Tavern         BuildingType = "tavern"
	Market         BuildingType = "market"
	Fortifications BuildingType = "fortifications"
)

// AllBuildingTypes returns all building types
func AllBuildingTypes() []BuildingType {
	return []BuildingType{
		Lumberjack, Quarry, OreMine, Farm,
		WoodStore, StoneStore, OreStore,
		Keep, Arsenal, Library, Tavern, Market, Fortifications,
	}
}

// Costs represents resource costs for an upgrade (no maps)
type Costs struct {
	Wood  int
	Stone int
	Iron  int
	Food  int
}

// Get returns the cost for a specific resource type
func (c Costs) Get(rt ResourceType) int {
	switch rt {
	case Wood:
		return c.Wood
	case Stone:
		return c.Stone
	case Iron:
		return c.Iron
	case Food:
		return c.Food
	}
	return 0
}

// BuildingLevel represents data for a specific building level
type BuildingLevel struct {
	Costs            Costs
	BuildTimeSeconds int
	ProductionRate   *float64 // nil if not a production building
	StorageCapacity  *int     // nil if not a storage building
}

// Building represents a building with all its levels
type Building struct {
	Type                    BuildingType
	MaxLevel                int
	Levels                  map[int]*BuildingLevel
	Prerequisites           map[int]map[BuildingType]int // level -> {building: min_level}
	TechnologyPrerequisites map[int]string               // level -> technology_name
}

// GetLevelData returns the level data for a specific level
func (b *Building) GetLevelData(level int) *BuildingLevel {
	if data, ok := b.Levels[level]; ok {
		return data
	}
	return nil
}

// Technology represents a researchable technology
type Technology struct {
	Name                 string
	InternalName         string
	RequiredLibraryLevel int
	Costs                Costs
	ResearchTimeSeconds  int
	EnablesBuilding      string
	EnablesLevel         int
}

// GameState represents the current game state
type GameState struct {
	BuildingLevels         map[BuildingType]int
	Resources              map[ResourceType]float64
	ResearchedTechnologies map[string]bool
	StorageCaps            map[ResourceType]int // Storage capacities for wood/stone/iron
	ProductionRates        map[ResourceType]float64
}

// NewGameState creates a new game state with defaults
func NewGameState() *GameState {
	return &GameState{
		BuildingLevels:         make(map[BuildingType]int),
		Resources:              make(map[ResourceType]float64),
		ResearchedTechnologies: make(map[string]bool),
		StorageCaps:            make(map[ResourceType]int),
		ProductionRates:        make(map[ResourceType]float64),
	}
}

// BuildingUpgradeAction represents a building upgrade action
type BuildingUpgradeAction struct {
	BuildingType BuildingType
	FromLevel    int
	ToLevel      int
	StartTime    int // seconds
	EndTime      int // seconds
	Costs        Costs
	FoodUsed     int // Total food used after this action
	FoodCapacity int // Food capacity after this action
}

// ResearchAction represents a technology research action
type ResearchAction struct {
	TechnologyName string
	StartTime      int // seconds
	EndTime        int // seconds
	Costs          Costs
	FoodUsed       int // Total food used after this action
	FoodCapacity   int // Food capacity after this action
}

// TrainUnitAction represents training units at Arsenal
type TrainUnitAction struct {
	UnitType     UnitType
	Count        int
	StartTime    int
	EndTime      int
	Costs        Costs
	FoodUsed     int
	FoodCapacity int
}

// MissionAction represents a tavern mission run
type MissionAction struct {
	MissionName  string
	StartTime    int
	EndTime      int
	ResourceCost Costs
	Rewards      []ResourceReward // Expected rewards
}

// Solution represents a complete build order solution
type Solution struct {
	BuildingActions  []BuildingUpgradeAction
	ResearchActions  []ResearchAction
	TrainingActions  []TrainUnitAction  // Unit training during build phase
	MissionActions   []MissionAction    // Missions run during build phase
	TotalTimeSeconds int
	FinalState       *GameState
}
