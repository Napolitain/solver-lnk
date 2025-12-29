// Package converter provides conversions between proto and model types
package converter

import (
	"github.com/napolitain/solver-lnk/internal/models"
	pb "github.com/napolitain/solver-lnk/proto"
)

// ProtoToModelBuildingType converts proto BuildingType to model BuildingType
func ProtoToModelBuildingType(bt pb.BuildingType) models.BuildingType {
	switch bt {
	case pb.BuildingType_LUMBERJACK:
		return models.Lumberjack
	case pb.BuildingType_QUARRY:
		return models.Quarry
	case pb.BuildingType_ORE_MINE:
		return models.OreMine
	case pb.BuildingType_FARM:
		return models.Farm
	case pb.BuildingType_WOOD_STORE:
		return models.WoodStore
	case pb.BuildingType_STONE_STORE:
		return models.StoneStore
	case pb.BuildingType_ORE_STORE:
		return models.OreStore
	case pb.BuildingType_KEEP:
		return models.Keep
	case pb.BuildingType_ARSENAL:
		return models.Arsenal
	case pb.BuildingType_LIBRARY:
		return models.Library
	case pb.BuildingType_TAVERN:
		return models.Tavern
	case pb.BuildingType_MARKET:
		return models.Market
	case pb.BuildingType_FORTIFICATIONS:
		return models.Fortifications
	default:
		return models.Keep
	}
}

// ModelToProtoBuildingType converts model BuildingType to proto BuildingType
func ModelToProtoBuildingType(bt models.BuildingType) pb.BuildingType {
	switch bt {
	case models.Lumberjack:
		return pb.BuildingType_LUMBERJACK
	case models.Quarry:
		return pb.BuildingType_QUARRY
	case models.OreMine:
		return pb.BuildingType_ORE_MINE
	case models.Farm:
		return pb.BuildingType_FARM
	case models.WoodStore:
		return pb.BuildingType_WOOD_STORE
	case models.StoneStore:
		return pb.BuildingType_STONE_STORE
	case models.OreStore:
		return pb.BuildingType_ORE_STORE
	case models.Keep:
		return pb.BuildingType_KEEP
	case models.Arsenal:
		return pb.BuildingType_ARSENAL
	case models.Library:
		return pb.BuildingType_LIBRARY
	case models.Tavern:
		return pb.BuildingType_TAVERN
	case models.Market:
		return pb.BuildingType_MARKET
	case models.Fortifications:
		return pb.BuildingType_FORTIFICATIONS
	default:
		return pb.BuildingType_BUILDING_UNKNOWN
	}
}

// ProtoToModelResourceType converts proto ResourceType to model ResourceType
func ProtoToModelResourceType(rt pb.ResourceType) models.ResourceType {
	switch rt {
	case pb.ResourceType_WOOD:
		return models.Wood
	case pb.ResourceType_STONE:
		return models.Stone
	case pb.ResourceType_IRON:
		return models.Iron
	case pb.ResourceType_FOOD:
		return models.Food
	default:
		return models.Wood
	}
}

// ModelToProtoResourceType converts model ResourceType to proto ResourceType
func ModelToProtoResourceType(rt models.ResourceType) pb.ResourceType {
	switch rt {
	case models.Wood:
		return pb.ResourceType_WOOD
	case models.Stone:
		return pb.ResourceType_STONE
	case models.Iron:
		return pb.ResourceType_IRON
	case models.Food:
		return pb.ResourceType_FOOD
	default:
		return pb.ResourceType_RESOURCE_UNKNOWN
	}
}

// TechNameToProto converts technology name string to proto Technology enum
func TechNameToProto(name string) pb.Technology {
	switch name {
	case "Longbow":
		return pb.Technology_LONGBOW
	case "Crop rotation":
		return pb.Technology_CROP_ROTATION
	case "Yoke":
		return pb.Technology_YOKE
	case "Cellar storeroom":
		return pb.Technology_CELLAR_STOREROOM
	case "Stirrup":
		return pb.Technology_STIRRUP
	case "Crossbow":
		return pb.Technology_CROSSBOW
	case "Swordsmith":
		return pb.Technology_SWORDSMITH
	case "Horse armour":
		return pb.Technology_HORSE_ARMOUR
	case "Beer tester":
		return pb.Technology_BEER_TESTER
	case "Wheelbarrow":
		return pb.Technology_WHEELBARROW
	case "Weaponsmith":
		return pb.Technology_WEAPONSMITH
	case "Armoursmith":
		return pb.Technology_ARMOURSMITH
	case "Iron hardening":
		return pb.Technology_IRON_HARDENING
	case "Poison arrows":
		return pb.Technology_POISON_ARROWS
	case "Horse breeding":
		return pb.Technology_HORSE_BREEDING
	case "Flaming arrows":
		return pb.Technology_FLAMING_ARROWS
	case "Blacksmith":
		return pb.Technology_BLACKSMITH
	case "Map of area":
		return pb.Technology_MAP_OF_AREA
	case "Cistern":
		return pb.Technology_CISTERN
	case "Fortress construction":
		return pb.Technology_FORTRESS_CONSTRUCTION
	default:
		return pb.Technology_TECH_UNKNOWN
	}
}

// UnitNameToProto converts unit name string to proto UnitType enum
func UnitNameToProto(name string) pb.UnitType {
	switch name {
	case "spearman":
		return pb.UnitType_SPEARMAN
	case "swordsman":
		return pb.UnitType_SWORDSMAN
	case "archer":
		return pb.UnitType_ARCHER
	case "crossbowman":
		return pb.UnitType_CROSSBOWMAN
	case "horseman":
		return pb.UnitType_HORSEMAN
	case "lancer":
		return pb.UnitType_LANCER
	case "handcart":
		return pb.UnitType_HANDCART
	case "oxcart":
		return pb.UnitType_OXCART
	default:
		return pb.UnitType_UNIT_UNKNOWN
	}
}
