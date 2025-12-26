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
		BuildingLevels: map[string]int32{
			"lumberjack":     15,
			"quarry":         12,
			"ore_mine":       10,
			"farm":           8,
			"wood_store":     1,
			"stone_store":    1,
			"ore_store":      1,
			"keep":           1,
			"arsenal":        1,
			"library":        1,
			"tavern":         1,
			"market":         1,
			"fortifications": 1,
		},
		Resources: map[string]float64{
			"wood":  500,
			"stone": 400,
			"iron":  300,
		},
		ResearchedTechnologies: []string{},
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
