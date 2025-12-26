package models

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"

	pb "github.com/napolitain/solver-lnk/proto"
)

// Proto to internal type mappings
var protoBuildingToInternal = map[pb.BuildingType]BuildingType{
	pb.BuildingType_LUMBERJACK:     Lumberjack,
	pb.BuildingType_QUARRY:         Quarry,
	pb.BuildingType_ORE_MINE:       OreMine,
	pb.BuildingType_FARM:           Farm,
	pb.BuildingType_WOOD_STORE:     WoodStore,
	pb.BuildingType_STONE_STORE:    StoneStore,
	pb.BuildingType_ORE_STORE:      OreStore,
	pb.BuildingType_KEEP:           Keep,
	pb.BuildingType_ARSENAL:        Arsenal,
	pb.BuildingType_LIBRARY:        Library,
	pb.BuildingType_TAVERN:         Tavern,
	pb.BuildingType_MARKET:         Market,
	pb.BuildingType_FORTIFICATIONS: Fortifications,
}

var protoResourceToInternal = map[pb.ResourceType]ResourceType{
	pb.ResourceType_WOOD:  Wood,
	pb.ResourceType_STONE: Stone,
	pb.ResourceType_IRON:  Iron,
	pb.ResourceType_FOOD:  Food,
}

var protoTechToString = map[pb.Technology]string{
	pb.Technology_LONGBOW:          "Longbow",
	pb.Technology_CROP_ROTATION:    "Crop rotation",
	pb.Technology_YOKE:             "Yoke",
	pb.Technology_CELLAR_STOREROOM: "Cellar storeroom",
	pb.Technology_STIRRUP:          "Stirrup",
	pb.Technology_CROSSBOW:         "Crossbow",
	pb.Technology_SWORDSMITH:       "Swordsmith",
	pb.Technology_HORSE_ARMOUR:     "Horse armour",
}

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
	required := []pb.BuildingType{
		pb.BuildingType_LUMBERJACK,
		pb.BuildingType_QUARRY,
		pb.BuildingType_ORE_MINE,
		pb.BuildingType_FARM,
		pb.BuildingType_WOOD_STORE,
		pb.BuildingType_STONE_STORE,
		pb.BuildingType_ORE_STORE,
		pb.BuildingType_KEEP,
		pb.BuildingType_ARSENAL,
		pb.BuildingType_LIBRARY,
		pb.BuildingType_TAVERN,
		pb.BuildingType_MARKET,
		pb.BuildingType_FORTIFICATIONS,
	}

	found := make(map[pb.BuildingType]bool)
	for _, bl := range c.BuildingLevels {
		found[bl.Type] = true
	}

	for _, r := range required {
		if !found[r] {
			return fmt.Errorf("missing building level for: %s", r.String())
		}
	}

	return nil
}

// CastleConfigToGameState converts protobuf CastleConfig to GameState
func CastleConfigToGameState(c *pb.CastleConfig) *GameState {
	state := NewGameState()

	// Set building levels from config
	for _, bl := range c.BuildingLevels {
		if internal, ok := protoBuildingToInternal[bl.Type]; ok {
			state.BuildingLevels[internal] = int(bl.Level)
		}
	}

	// Set resources (default to 120)
	state.Resources[Wood] = 120
	state.Resources[Stone] = 120
	state.Resources[Iron] = 120
	state.Resources[Food] = 40

	for _, r := range c.Resources {
		if internal, ok := protoResourceToInternal[r.Type]; ok {
			state.Resources[internal] = r.Amount
		}
	}

	// Set researched technologies
	for _, tech := range c.ResearchedTechnologies {
		if name, ok := protoTechToString[tech]; ok {
			state.ResearchedTechnologies[name] = true
		}
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
