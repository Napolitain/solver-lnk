package converter

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/models"
	pb "github.com/napolitain/solver-lnk/proto"
)

func TestProtoToModelBuildingType(t *testing.T) {
	tests := []struct {
		name  string
		input pb.BuildingType
		want  models.BuildingType
	}{
		{"Lumberjack", pb.BuildingType_LUMBERJACK, models.Lumberjack},
		{"Quarry", pb.BuildingType_QUARRY, models.Quarry},
		{"OreMine", pb.BuildingType_ORE_MINE, models.OreMine},
		{"Farm", pb.BuildingType_FARM, models.Farm},
		{"WoodStore", pb.BuildingType_WOOD_STORE, models.WoodStore},
		{"StoneStore", pb.BuildingType_STONE_STORE, models.StoneStore},
		{"OreStore", pb.BuildingType_ORE_STORE, models.OreStore},
		{"Keep", pb.BuildingType_KEEP, models.Keep},
		{"Arsenal", pb.BuildingType_ARSENAL, models.Arsenal},
		{"Library", pb.BuildingType_LIBRARY, models.Library},
		{"Tavern", pb.BuildingType_TAVERN, models.Tavern},
		{"Market", pb.BuildingType_MARKET, models.Market},
		{"Fortifications", pb.BuildingType_FORTIFICATIONS, models.Fortifications},
		{"Unknown defaults to Keep", pb.BuildingType_BUILDING_UNKNOWN, models.Keep},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProtoToModelBuildingType(tt.input)
			if got != tt.want {
				t.Errorf("ProtoToModelBuildingType(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestModelToProtoBuildingType(t *testing.T) {
	tests := []struct {
		name  string
		input models.BuildingType
		want  pb.BuildingType
	}{
		{"Lumberjack", models.Lumberjack, pb.BuildingType_LUMBERJACK},
		{"Quarry", models.Quarry, pb.BuildingType_QUARRY},
		{"OreMine", models.OreMine, pb.BuildingType_ORE_MINE},
		{"Farm", models.Farm, pb.BuildingType_FARM},
		{"WoodStore", models.WoodStore, pb.BuildingType_WOOD_STORE},
		{"StoneStore", models.StoneStore, pb.BuildingType_STONE_STORE},
		{"OreStore", models.OreStore, pb.BuildingType_ORE_STORE},
		{"Keep", models.Keep, pb.BuildingType_KEEP},
		{"Arsenal", models.Arsenal, pb.BuildingType_ARSENAL},
		{"Library", models.Library, pb.BuildingType_LIBRARY},
		{"Tavern", models.Tavern, pb.BuildingType_TAVERN},
		{"Market", models.Market, pb.BuildingType_MARKET},
		{"Fortifications", models.Fortifications, pb.BuildingType_FORTIFICATIONS},
		{"Unknown type", models.BuildingType("invalid"), pb.BuildingType_BUILDING_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModelToProtoBuildingType(tt.input)
			if got != tt.want {
				t.Errorf("ModelToProtoBuildingType(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestProtoToModelResourceType(t *testing.T) {
	tests := []struct {
		name  string
		input pb.ResourceType
		want  models.ResourceType
	}{
		{"Wood", pb.ResourceType_WOOD, models.Wood},
		{"Stone", pb.ResourceType_STONE, models.Stone},
		{"Iron", pb.ResourceType_IRON, models.Iron},
		{"Food", pb.ResourceType_FOOD, models.Food},
		{"Unknown defaults to Wood", pb.ResourceType_RESOURCE_UNKNOWN, models.Wood},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProtoToModelResourceType(tt.input)
			if got != tt.want {
				t.Errorf("ProtoToModelResourceType(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestModelToProtoResourceType(t *testing.T) {
	tests := []struct {
		name  string
		input models.ResourceType
		want  pb.ResourceType
	}{
		{"Wood", models.Wood, pb.ResourceType_WOOD},
		{"Stone", models.Stone, pb.ResourceType_STONE},
		{"Iron", models.Iron, pb.ResourceType_IRON},
		{"Food", models.Food, pb.ResourceType_FOOD},
		{"Unknown type", models.ResourceType("invalid"), pb.ResourceType_RESOURCE_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModelToProtoResourceType(tt.input)
			if got != tt.want {
				t.Errorf("ModelToProtoResourceType(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTechNameToProto(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  pb.Technology
	}{
		{"Longbow", "Longbow", pb.Technology_LONGBOW},
		{"CropRotation", "Crop rotation", pb.Technology_CROP_ROTATION},
		{"Yoke", "Yoke", pb.Technology_YOKE},
		{"CellarStoreroom", "Cellar storeroom", pb.Technology_CELLAR_STOREROOM},
		{"Stirrup", "Stirrup", pb.Technology_STIRRUP},
		{"Crossbow", "Crossbow", pb.Technology_CROSSBOW},
		{"Swordsmith", "Swordsmith", pb.Technology_SWORDSMITH},
		{"HorseArmour", "Horse armour", pb.Technology_HORSE_ARMOUR},
		{"BeerTester", "Beer tester", pb.Technology_BEER_TESTER},
		{"Wheelbarrow", "Wheelbarrow", pb.Technology_WHEELBARROW},
		{"Unknown", "unknown_tech", pb.Technology_TECH_UNKNOWN},
		{"Empty", "", pb.Technology_TECH_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TechNameToProto(tt.input)
			if got != tt.want {
				t.Errorf("TechNameToProto(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnitNameToProto(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  pb.UnitType
	}{
		{"Spearman", "spearman", pb.UnitType_SPEARMAN},
		{"Swordsman", "swordsman", pb.UnitType_SWORDSMAN},
		{"Archer", "archer", pb.UnitType_ARCHER},
		{"Crossbowman", "crossbowman", pb.UnitType_CROSSBOWMAN},
		{"Horseman", "horseman", pb.UnitType_HORSEMAN},
		{"Lancer", "lancer", pb.UnitType_LANCER},
		{"Handcart", "handcart", pb.UnitType_HANDCART},
		{"Oxcart", "oxcart", pb.UnitType_OXCART},
		{"Unknown", "unknown_unit", pb.UnitType_UNIT_UNKNOWN},
		{"Empty", "", pb.UnitType_UNIT_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnitNameToProto(tt.input)
			if got != tt.want {
				t.Errorf("UnitNameToProto(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Test roundtrip conversions
func TestBuildingTypeRoundtrip(t *testing.T) {
	buildingTypes := []models.BuildingType{
		models.Lumberjack, models.Quarry, models.OreMine, models.Farm,
		models.WoodStore, models.StoneStore, models.OreStore,
		models.Keep, models.Arsenal, models.Library, models.Tavern,
		models.Market, models.Fortifications,
	}

	for _, bt := range buildingTypes {
		proto := ModelToProtoBuildingType(bt)
		back := ProtoToModelBuildingType(proto)
		if back != bt {
			t.Errorf("Roundtrip failed for %v: got %v", bt, back)
		}
	}
}

func TestResourceTypeRoundtrip(t *testing.T) {
	resourceTypes := []models.ResourceType{
		models.Wood, models.Stone, models.Iron, models.Food,
	}

	for _, rt := range resourceTypes {
		proto := ModelToProtoResourceType(rt)
		back := ProtoToModelResourceType(proto)
		if back != rt {
			t.Errorf("Roundtrip failed for %v: got %v", rt, back)
		}
	}
}
