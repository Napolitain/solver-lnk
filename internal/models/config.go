package models

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"

	pb "github.com/napolitain/solver-lnk/proto"
)

// LoadCastleConfig loads castle configuration from binary protobuf file
func LoadCastleConfig(path string) (*pb.CastleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &pb.CastleConfig{}
	if err := proto.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("invalid protobuf: %w", err)
	}

	return config, nil
}

// LoadUnitsConfig loads units configuration from binary protobuf file
func LoadUnitsConfig(path string) (*pb.UnitsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &pb.UnitsConfig{}
	if err := proto.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("invalid protobuf: %w", err)
	}

	return config, nil
}

// ValidateCastleConfig checks that all required buildings are specified
func ValidateCastleConfig(c *pb.CastleConfig) error {
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

// CastleConfigToGameState converts protobuf CastleConfig to GameState
func CastleConfigToGameState(c *pb.CastleConfig) *GameState {
	state := NewGameState()

	// Set building levels from config
	for name, level := range c.BuildingLevels {
		state.BuildingLevels[BuildingType(name)] = int(level)
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

// GetTargetLevels returns fixed target levels (always max)
func GetTargetLevels() map[BuildingType]int {
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
