package models

import (
	"encoding/json"
	"fmt"
	"os"
)

// CastleConfig represents input configuration for castle solver
type CastleConfig struct {
	// Current building levels (ALL buildings must be specified)
	BuildingLevels map[string]int `json:"building_levels"`

	// Current resources
	Resources map[string]float64 `json:"resources,omitempty"`

	// Already researched technologies
	ResearchedTechnologies []string `json:"researched_technologies,omitempty"`
}

// UnitsConfig represents input configuration for units solver
type UnitsConfig struct {
	// Available food for units (default: 4265)
	FoodAvailable int `json:"food_available,omitempty"`

	// Resource production per hour (default: 1161)
	ResourceProductionPerHour int `json:"resource_production_per_hour,omitempty"`

	// Market distance in fields (default: 25)
	MarketDistanceFields int `json:"market_distance_fields,omitempty"`

	// Existing units (optional)
	ExistingUnits map[string]int `json:"existing_units,omitempty"`
}

// LoadCastleConfig loads castle configuration from JSON file
func LoadCastleConfig(path string) (*CastleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config CastleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadUnitsConfig loads units configuration from JSON file
func LoadUnitsConfig(path string) (*UnitsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config UnitsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetFoodAvailable returns the food available for units
func (c *UnitsConfig) GetFoodAvailable() int {
	return c.FoodAvailable
}

// GetResourceProductionPerHour returns the resource production rate
func (c *UnitsConfig) GetResourceProductionPerHour() int {
	return c.ResourceProductionPerHour
}

// GetMarketDistanceFields returns the market distance
func (c *UnitsConfig) GetMarketDistanceFields() int {
	return c.MarketDistanceFields
}

// ToGameState converts CastleConfig to GameState
func (c *CastleConfig) ToGameState() *GameState {
	state := NewGameState()

	// Set building levels from config (all must be specified)
	for name, level := range c.BuildingLevels {
		state.BuildingLevels[BuildingType(name)] = level
	}

	// Set resources (default to 120)
	state.Resources[Wood] = 120
	state.Resources[Stone] = 120
	state.Resources[Iron] = 120
	state.Resources[Food] = 40

	for name, amount := range c.Resources {
		state.Resources[ResourceType(name)] = amount
	}

	// Set researched technologies
	for _, tech := range c.ResearchedTechnologies {
		state.ResearchedTechnologies[tech] = true
	}

	return state
}

// Validate checks that all required buildings are specified
func (c *CastleConfig) Validate() error {
	required := []string{
		"lumberjack", "quarry", "ore_mine", "farm",
		"wood_store", "stone_store", "ore_store",
		"keep", "arsenal", "library", "tavern", "market", "fortifications",
	}

	for _, name := range required {
		if _, ok := c.BuildingLevels[name]; !ok {
			return fmt.Errorf("missing building level for: %s", name)
		}
	}

	return nil
}

// GetTargetLevels returns fixed target levels (always max)
func (c *CastleConfig) GetTargetLevels() map[BuildingType]int {
	return map[BuildingType]int{
		Lumberjack:     30,
		Quarry:         30,
		OreMine:        30,
		Farm:           30,
		WoodStore:      20,
		StoneStore:     20,
		OreStore:       20,
		Keep:           10,
		Arsenal:        30,
		Library:        10,
		Tavern:         10,
		Market:         8,
		Fortifications: 20,
	}
}
