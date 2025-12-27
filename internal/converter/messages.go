package converter

import (
	"github.com/napolitain/solver-lnk/internal/models"
	pb "github.com/napolitain/solver-lnk/proto"
)

// CostsToProtoResources converts model Costs to proto Resources
func CostsToProtoResources(costs models.Costs) []*pb.Resource {
	var resources []*pb.Resource
	for rt, amount := range costs {
		resources = append(resources, &pb.Resource{
			Type:   ModelToProtoResourceType(rt),
			Amount: float64(amount),
		})
	}
	return resources
}

// ProtoRequestToGameState converts proto SolveRequest to internal GameState
func ProtoRequestToGameState(req *pb.SolveRequest) *models.GameState {
	state := models.NewGameState()

	if req.CastleConfig != nil {
		// Building levels
		for _, bl := range req.CastleConfig.BuildingLevels {
			state.BuildingLevels[ProtoToModelBuildingType(bl.Type)] = int(bl.Level)
		}

		// Resources
		for _, r := range req.CastleConfig.Resources {
			state.Resources[ProtoToModelResourceType(r.Type)] = r.Amount
		}

		// TODO: researched technologies
	}

	return state
}

// ProtoTargetsToModelTargets converts proto TargetLevels to model targets
func ProtoTargetsToModelTargets(targets *pb.TargetLevels) map[models.BuildingType]int {
	result := make(map[models.BuildingType]int)
	if targets != nil {
		for _, t := range targets.Targets {
			result[ProtoToModelBuildingType(t.Type)] = int(t.Level)
		}
	}
	return result
}

// BuildingActionToProto converts model BuildingUpgradeAction to proto BuildingAction
func BuildingActionToProto(action models.BuildingUpgradeAction) *pb.BuildingAction {
	return &pb.BuildingAction{
		BuildingType:     ModelToProtoBuildingType(action.BuildingType),
		FromLevel:        int32(action.FromLevel),
		ToLevel:          int32(action.ToLevel),
		StartTimeSeconds: int32(action.StartTime),
		EndTimeSeconds:   int32(action.EndTime),
		Costs:            CostsToProtoResources(action.Costs),
		FoodUsed:         int32(action.FoodUsed),
		FoodCapacity:     int32(action.FoodCapacity),
	}
}

// ResearchActionToProto converts model ResearchAction to proto ResearchAction
func ResearchActionToProto(action models.ResearchAction) *pb.ResearchAction {
	return &pb.ResearchAction{
		Technology:       TechNameToProto(action.TechnologyName),
		StartTimeSeconds: int32(action.StartTime),
		EndTimeSeconds:   int32(action.EndTime),
		Costs:            CostsToProtoResources(action.Costs),
	}
}
