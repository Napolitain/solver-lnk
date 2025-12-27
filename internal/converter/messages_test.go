package converter

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/models"
	pb "github.com/napolitain/solver-lnk/proto"
)

func TestCostsToProtoResources(t *testing.T) {
	costs := models.Costs{
		models.Wood:  100,
		models.Stone: 200,
		models.Iron:  50,
		models.Food:  10,
	}

	resources := CostsToProtoResources(costs)

	if len(resources) != 4 {
		t.Errorf("Expected 4 resources, got %d", len(resources))
	}

	// Check all resources are present with correct amounts
	found := make(map[pb.ResourceType]float64)
	for _, r := range resources {
		found[r.Type] = r.Amount
	}

	if found[pb.ResourceType_WOOD] != 100 {
		t.Errorf("Wood: got %.0f, want 100", found[pb.ResourceType_WOOD])
	}
	if found[pb.ResourceType_STONE] != 200 {
		t.Errorf("Stone: got %.0f, want 200", found[pb.ResourceType_STONE])
	}
	if found[pb.ResourceType_IRON] != 50 {
		t.Errorf("Iron: got %.0f, want 50", found[pb.ResourceType_IRON])
	}
	if found[pb.ResourceType_FOOD] != 10 {
		t.Errorf("Food: got %.0f, want 10", found[pb.ResourceType_FOOD])
	}
}

func TestCostsToProtoResourcesEmpty(t *testing.T) {
	costs := models.Costs{}
	resources := CostsToProtoResources(costs)

	if len(resources) != 0 {
		t.Errorf("Expected 0 resources for empty costs, got %d", len(resources))
	}
}

func TestProtoRequestToGameState(t *testing.T) {
	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{
			BuildingLevels: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 10},
				{Type: pb.BuildingType_QUARRY, Level: 5},
				{Type: pb.BuildingType_KEEP, Level: 3},
			},
			Resources: []*pb.Resource{
				{Type: pb.ResourceType_WOOD, Amount: 1000},
				{Type: pb.ResourceType_STONE, Amount: 500},
				{Type: pb.ResourceType_IRON, Amount: 200},
				{Type: pb.ResourceType_FOOD, Amount: 100},
			},
		},
	}

	state := ProtoRequestToGameState(req)

	// Check building levels
	if state.BuildingLevels[models.Lumberjack] != 10 {
		t.Errorf("Lumberjack level: got %d, want 10", state.BuildingLevels[models.Lumberjack])
	}
	if state.BuildingLevels[models.Quarry] != 5 {
		t.Errorf("Quarry level: got %d, want 5", state.BuildingLevels[models.Quarry])
	}
	if state.BuildingLevels[models.Keep] != 3 {
		t.Errorf("Keep level: got %d, want 3", state.BuildingLevels[models.Keep])
	}

	// Check resources
	if state.Resources[models.Wood] != 1000 {
		t.Errorf("Wood: got %.0f, want 1000", state.Resources[models.Wood])
	}
	if state.Resources[models.Stone] != 500 {
		t.Errorf("Stone: got %.0f, want 500", state.Resources[models.Stone])
	}
	if state.Resources[models.Iron] != 200 {
		t.Errorf("Iron: got %.0f, want 200", state.Resources[models.Iron])
	}
	if state.Resources[models.Food] != 100 {
		t.Errorf("Food: got %.0f, want 100", state.Resources[models.Food])
	}
}

func TestProtoRequestToGameStateNilConfig(t *testing.T) {
	req := &pb.SolveRequest{
		CastleConfig: nil,
	}

	state := ProtoRequestToGameState(req)

	// Should return default state without panicking
	if state == nil {
		t.Error("Expected non-nil state")
	}
}

func TestProtoRequestToGameStateEmptyConfig(t *testing.T) {
	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
	}

	state := ProtoRequestToGameState(req)

	if state == nil {
		t.Error("Expected non-nil state")
	}
}

func TestProtoTargetsToModelTargets(t *testing.T) {
	targets := &pb.TargetLevels{
		Targets: []*pb.BuildingLevel{
			{Type: pb.BuildingType_LUMBERJACK, Level: 30},
			{Type: pb.BuildingType_QUARRY, Level: 30},
			{Type: pb.BuildingType_KEEP, Level: 10},
		},
	}

	result := ProtoTargetsToModelTargets(targets)

	if result[models.Lumberjack] != 30 {
		t.Errorf("Lumberjack target: got %d, want 30", result[models.Lumberjack])
	}
	if result[models.Quarry] != 30 {
		t.Errorf("Quarry target: got %d, want 30", result[models.Quarry])
	}
	if result[models.Keep] != 10 {
		t.Errorf("Keep target: got %d, want 10", result[models.Keep])
	}
}

func TestProtoTargetsToModelTargetsNil(t *testing.T) {
	result := ProtoTargetsToModelTargets(nil)

	if result == nil {
		t.Error("Expected non-nil map")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(result))
	}
}

func TestBuildingActionToProto(t *testing.T) {
	action := models.BuildingUpgradeAction{
		BuildingType: models.Lumberjack,
		FromLevel:    5,
		ToLevel:      6,
		StartTime:    1000,
		EndTime:      2000,
		Costs: models.Costs{
			models.Wood:  100,
			models.Stone: 50,
		},
		FoodUsed:     100,
		FoodCapacity: 500,
	}

	proto := BuildingActionToProto(action)

	if proto.BuildingType != pb.BuildingType_LUMBERJACK {
		t.Errorf("BuildingType: got %v, want LUMBERJACK", proto.BuildingType)
	}
	if proto.FromLevel != 5 {
		t.Errorf("FromLevel: got %d, want 5", proto.FromLevel)
	}
	if proto.ToLevel != 6 {
		t.Errorf("ToLevel: got %d, want 6", proto.ToLevel)
	}
	if proto.StartTimeSeconds != 1000 {
		t.Errorf("StartTimeSeconds: got %d, want 1000", proto.StartTimeSeconds)
	}
	if proto.EndTimeSeconds != 2000 {
		t.Errorf("EndTimeSeconds: got %d, want 2000", proto.EndTimeSeconds)
	}
	if proto.FoodUsed != 100 {
		t.Errorf("FoodUsed: got %d, want 100", proto.FoodUsed)
	}
	if proto.FoodCapacity != 500 {
		t.Errorf("FoodCapacity: got %d, want 500", proto.FoodCapacity)
	}
	if len(proto.Costs) != 2 {
		t.Errorf("Costs length: got %d, want 2", len(proto.Costs))
	}
}

func TestResearchActionToProto(t *testing.T) {
	action := models.ResearchAction{
		TechnologyName: "longbow",
		StartTime:      500,
		EndTime:        1500,
		Costs: models.Costs{
			models.Wood: 200,
			models.Iron: 100,
		},
	}

	proto := ResearchActionToProto(action)

	if proto.Technology != pb.Technology_LONGBOW {
		t.Errorf("Technology: got %v, want LONGBOW", proto.Technology)
	}
	if proto.StartTimeSeconds != 500 {
		t.Errorf("StartTimeSeconds: got %d, want 500", proto.StartTimeSeconds)
	}
	if proto.EndTimeSeconds != 1500 {
		t.Errorf("EndTimeSeconds: got %d, want 1500", proto.EndTimeSeconds)
	}
	if len(proto.Costs) != 2 {
		t.Errorf("Costs length: got %d, want 2", len(proto.Costs))
	}
}
