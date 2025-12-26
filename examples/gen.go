//go:build ignore

package main

import (
	"os"

	"google.golang.org/protobuf/proto"

	pb "github.com/napolitain/solver-lnk/proto"
)

func main() {
	// Castle midgame example
	castle := &pb.CastleConfig{
		BuildingLevels: []*pb.BuildingLevel{
			{Type: pb.BuildingType_LUMBERJACK, Level: 15},
			{Type: pb.BuildingType_QUARRY, Level: 12},
			{Type: pb.BuildingType_ORE_MINE, Level: 10},
			{Type: pb.BuildingType_FARM, Level: 8},
			{Type: pb.BuildingType_WOOD_STORE, Level: 1},
			{Type: pb.BuildingType_STONE_STORE, Level: 1},
			{Type: pb.BuildingType_ORE_STORE, Level: 1},
			{Type: pb.BuildingType_KEEP, Level: 1},
			{Type: pb.BuildingType_ARSENAL, Level: 1},
			{Type: pb.BuildingType_LIBRARY, Level: 1},
			{Type: pb.BuildingType_TAVERN, Level: 1},
			{Type: pb.BuildingType_MARKET, Level: 1},
			{Type: pb.BuildingType_FORTIFICATIONS, Level: 1},
		},
		Resources: []*pb.Resource{
			{Type: pb.ResourceType_WOOD, Amount: 500},
			{Type: pb.ResourceType_STONE, Amount: 400},
			{Type: pb.ResourceType_IRON, Amount: 300},
		},
		ResearchedTechnologies: []pb.Technology{},
	}

	data, _ := proto.Marshal(castle)
	os.WriteFile("examples/castle_midgame.pb", data, 0644)

	// Units custom example
	units := &pb.UnitsConfig{
		FoodAvailable:             3000,
		ResourceProductionPerHour: 800,
		MarketDistanceFields:      30,
	}

	data, _ = proto.Marshal(units)
	os.WriteFile("examples/units_custom.pb", data, 0644)
}
