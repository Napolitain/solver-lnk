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

// AllBuildingTypes returns all building types in deterministic order
func AllBuildingTypes() []BuildingType {
	return []BuildingType{
		Lumberjack, Quarry, OreMine, Farm,
		WoodStore, StoneStore, OreStore,
		Keep, Arsenal, Library, Tavern, Market, Fortifications,
	}
}

// BuildingLevelMap is a deterministic struct for building levels (replaces map[BuildingType]int)
type BuildingLevelMap struct {
	Lumberjack     int
	Quarry         int
	OreMine        int
	Farm           int
	WoodStore      int
	StoneStore     int
	OreStore       int
	Keep           int
	Arsenal        int
	Library        int
	Tavern         int
	Market         int
	Fortifications int
}

// Get returns the level for a building type
func (b *BuildingLevelMap) Get(bt BuildingType) int {
	switch bt {
	case Lumberjack:
		return b.Lumberjack
	case Quarry:
		return b.Quarry
	case OreMine:
		return b.OreMine
	case Farm:
		return b.Farm
	case WoodStore:
		return b.WoodStore
	case StoneStore:
		return b.StoneStore
	case OreStore:
		return b.OreStore
	case Keep:
		return b.Keep
	case Arsenal:
		return b.Arsenal
	case Library:
		return b.Library
	case Tavern:
		return b.Tavern
	case Market:
		return b.Market
	case Fortifications:
		return b.Fortifications
	}
	return 0
}

// Set sets the level for a building type
func (b *BuildingLevelMap) Set(bt BuildingType, level int) {
	switch bt {
	case Lumberjack:
		b.Lumberjack = level
	case Quarry:
		b.Quarry = level
	case OreMine:
		b.OreMine = level
	case Farm:
		b.Farm = level
	case WoodStore:
		b.WoodStore = level
	case StoneStore:
		b.StoneStore = level
	case OreStore:
		b.OreStore = level
	case Keep:
		b.Keep = level
	case Arsenal:
		b.Arsenal = level
	case Library:
		b.Library = level
	case Tavern:
		b.Tavern = level
	case Market:
		b.Market = level
	case Fortifications:
		b.Fortifications = level
	}
}

// Each iterates over all buildings in deterministic order
func (b *BuildingLevelMap) Each(fn func(BuildingType, int)) {
	fn(Lumberjack, b.Lumberjack)
	fn(Quarry, b.Quarry)
	fn(OreMine, b.OreMine)
	fn(Farm, b.Farm)
	fn(WoodStore, b.WoodStore)
	fn(StoneStore, b.StoneStore)
	fn(OreStore, b.OreStore)
	fn(Keep, b.Keep)
	fn(Arsenal, b.Arsenal)
	fn(Library, b.Library)
	fn(Tavern, b.Tavern)
	fn(Market, b.Market)
	fn(Fortifications, b.Fortifications)
}

// EachNonZero iterates over buildings with non-zero levels
func (b *BuildingLevelMap) EachNonZero(fn func(BuildingType, int)) {
	if b.Lumberjack > 0 {
		fn(Lumberjack, b.Lumberjack)
	}
	if b.Quarry > 0 {
		fn(Quarry, b.Quarry)
	}
	if b.OreMine > 0 {
		fn(OreMine, b.OreMine)
	}
	if b.Farm > 0 {
		fn(Farm, b.Farm)
	}
	if b.WoodStore > 0 {
		fn(WoodStore, b.WoodStore)
	}
	if b.StoneStore > 0 {
		fn(StoneStore, b.StoneStore)
	}
	if b.OreStore > 0 {
		fn(OreStore, b.OreStore)
	}
	if b.Keep > 0 {
		fn(Keep, b.Keep)
	}
	if b.Arsenal > 0 {
		fn(Arsenal, b.Arsenal)
	}
	if b.Library > 0 {
		fn(Library, b.Library)
	}
	if b.Tavern > 0 {
		fn(Tavern, b.Tavern)
	}
	if b.Market > 0 {
		fn(Market, b.Market)
	}
	if b.Fortifications > 0 {
		fn(Fortifications, b.Fortifications)
	}
}

// TechName represents technology names as constants
type TechName string

const (
	TechLongbow              TechName = "Longbow"
	TechCropRotation         TechName = "Crop rotation"
	TechYoke                 TechName = "Yoke"
	TechCellarStoreroom      TechName = "Cellar storeroom"
	TechStirrup              TechName = "Stirrup"
	TechWeaponsmith          TechName = "Weaponsmith"
	TechArmoursmith          TechName = "Armoursmith"
	TechBeerTester           TechName = "Beer tester"
	TechSwordsmith           TechName = "Swordsmith"
	TechIronHardening        TechName = "Iron hardening"
	TechCrossbow             TechName = "Crossbow"
	TechPoisonArrows         TechName = "Poison arrows"
	TechHorseBreeding        TechName = "Horse breeding"
	TechWeaponsManufacturing TechName = "Weapons manufacturing"
	TechHorseArmour          TechName = "Horse armour"
	TechWheelbarrow          TechName = "Wheelbarrow"
	TechFlamingArrows        TechName = "Flaming arrows"
	TechBlacksmith           TechName = "Blacksmith"
	TechMapOfArea            TechName = "Map of area"
	TechCistern              TechName = "Cistern"
)

// AllTechNames returns all technology names in deterministic order (by library level)
func AllTechNames() []TechName {
	return []TechName{
		TechLongbow, TechCropRotation, TechYoke, TechCellarStoreroom, // lib 1
		TechStirrup,                                                   // lib 2
		TechWeaponsmith, TechArmoursmith, TechBeerTester,              // lib 3
		TechSwordsmith, TechIronHardening,                             // lib 4
		TechCrossbow,                                                  // lib 5
		TechPoisonArrows, TechHorseBreeding,                           // lib 6
		TechWeaponsManufacturing, TechHorseArmour,                     // lib 7
		TechWheelbarrow, TechFlamingArrows,                            // lib 8
		TechBlacksmith,                                                // lib 9
		TechMapOfArea, TechCistern,                                    // lib 10
	}
}

// TechFlags tracks which technologies are researched (deterministic)
type TechFlags struct {
	Longbow              bool
	CropRotation         bool
	Yoke                 bool
	CellarStoreroom      bool
	Stirrup              bool
	Weaponsmith          bool
	Armoursmith          bool
	BeerTester           bool
	Swordsmith           bool
	IronHardening        bool
	Crossbow             bool
	PoisonArrows         bool
	HorseBreeding        bool
	WeaponsManufacturing bool
	HorseArmour          bool
	Wheelbarrow          bool
	FlamingArrows        bool
	Blacksmith           bool
	MapOfArea            bool
	Cistern              bool
}

// Get returns whether a tech is researched
func (t *TechFlags) Get(name TechName) bool {
	switch name {
	case TechLongbow:
		return t.Longbow
	case TechCropRotation:
		return t.CropRotation
	case TechYoke:
		return t.Yoke
	case TechCellarStoreroom:
		return t.CellarStoreroom
	case TechStirrup:
		return t.Stirrup
	case TechWeaponsmith:
		return t.Weaponsmith
	case TechArmoursmith:
		return t.Armoursmith
	case TechBeerTester:
		return t.BeerTester
	case TechSwordsmith:
		return t.Swordsmith
	case TechIronHardening:
		return t.IronHardening
	case TechCrossbow:
		return t.Crossbow
	case TechPoisonArrows:
		return t.PoisonArrows
	case TechHorseBreeding:
		return t.HorseBreeding
	case TechWeaponsManufacturing:
		return t.WeaponsManufacturing
	case TechHorseArmour:
		return t.HorseArmour
	case TechWheelbarrow:
		return t.Wheelbarrow
	case TechFlamingArrows:
		return t.FlamingArrows
	case TechBlacksmith:
		return t.Blacksmith
	case TechMapOfArea:
		return t.MapOfArea
	case TechCistern:
		return t.Cistern
	}
	return false
}

// GetByString returns whether a tech is researched (for compatibility with string keys)
func (t *TechFlags) GetByString(name string) bool {
	return t.Get(TechName(name))
}

// Set sets whether a tech is researched
func (t *TechFlags) Set(name TechName, researched bool) {
	switch name {
	case TechLongbow:
		t.Longbow = researched
	case TechCropRotation:
		t.CropRotation = researched
	case TechYoke:
		t.Yoke = researched
	case TechCellarStoreroom:
		t.CellarStoreroom = researched
	case TechStirrup:
		t.Stirrup = researched
	case TechWeaponsmith:
		t.Weaponsmith = researched
	case TechArmoursmith:
		t.Armoursmith = researched
	case TechBeerTester:
		t.BeerTester = researched
	case TechSwordsmith:
		t.Swordsmith = researched
	case TechIronHardening:
		t.IronHardening = researched
	case TechCrossbow:
		t.Crossbow = researched
	case TechPoisonArrows:
		t.PoisonArrows = researched
	case TechHorseBreeding:
		t.HorseBreeding = researched
	case TechWeaponsManufacturing:
		t.WeaponsManufacturing = researched
	case TechHorseArmour:
		t.HorseArmour = researched
	case TechWheelbarrow:
		t.Wheelbarrow = researched
	case TechFlamingArrows:
		t.FlamingArrows = researched
	case TechBlacksmith:
		t.Blacksmith = researched
	case TechMapOfArea:
		t.MapOfArea = researched
	case TechCistern:
		t.Cistern = researched
	}
}

// SetByString sets whether a tech is researched (for compatibility)
func (t *TechFlags) SetByString(name string, researched bool) {
	t.Set(TechName(name), researched)
}

// Each iterates over all techs in deterministic order
func (t *TechFlags) Each(fn func(TechName, bool)) {
	fn(TechLongbow, t.Longbow)
	fn(TechCropRotation, t.CropRotation)
	fn(TechYoke, t.Yoke)
	fn(TechCellarStoreroom, t.CellarStoreroom)
	fn(TechStirrup, t.Stirrup)
	fn(TechWeaponsmith, t.Weaponsmith)
	fn(TechArmoursmith, t.Armoursmith)
	fn(TechBeerTester, t.BeerTester)
	fn(TechSwordsmith, t.Swordsmith)
	fn(TechIronHardening, t.IronHardening)
	fn(TechCrossbow, t.Crossbow)
	fn(TechPoisonArrows, t.PoisonArrows)
	fn(TechHorseBreeding, t.HorseBreeding)
	fn(TechWeaponsManufacturing, t.WeaponsManufacturing)
	fn(TechHorseArmour, t.HorseArmour)
	fn(TechWheelbarrow, t.Wheelbarrow)
	fn(TechFlamingArrows, t.FlamingArrows)
	fn(TechBlacksmith, t.Blacksmith)
	fn(TechMapOfArea, t.MapOfArea)
	fn(TechCistern, t.Cistern)
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
