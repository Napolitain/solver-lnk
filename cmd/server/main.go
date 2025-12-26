package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	"github.com/napolitain/solver-lnk/internal/solver"
	pb "github.com/napolitain/solver-lnk/proto"
)

var (
	port    = flag.Int("port", 50051, "The server port")
	dataDir = flag.String("data", "data", "Path to data directory")
)

// server is used to implement the CastleSolverService
type server struct {
	pb.UnimplementedCastleSolverServiceServer
	buildings    map[models.BuildingType]*models.Building
	technologies map[string]*models.Technology
}

// protoToModelBuildingType converts proto BuildingType to model BuildingType
func protoToModelBuildingType(bt pb.BuildingType) models.BuildingType {
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

// modelToProtoBuildingType converts model BuildingType to proto BuildingType
func modelToProtoBuildingType(bt models.BuildingType) pb.BuildingType {
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

// protoToModelResourceType converts proto ResourceType to model ResourceType
func protoToModelResourceType(rt pb.ResourceType) models.ResourceType {
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

// modelToProtoResourceType converts model ResourceType to proto ResourceType
func modelToProtoResourceType(rt models.ResourceType) pb.ResourceType {
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

// costsToProtoResources converts model Costs to proto Resources
func costsToProtoResources(costs models.Costs) []*pb.Resource {
	var resources []*pb.Resource
	for rt, amount := range costs {
		resources = append(resources, &pb.Resource{
			Type:   modelToProtoResourceType(rt),
			Amount: float64(amount),
		})
	}
	return resources
}

// protoRequestToGameState converts proto SolveRequest to internal GameState
func protoRequestToGameState(req *pb.SolveRequest) *models.GameState {
	state := models.NewGameState()

	if req.CastleConfig != nil {
		// Building levels
		for _, bl := range req.CastleConfig.BuildingLevels {
			state.BuildingLevels[protoToModelBuildingType(bl.Type)] = int(bl.Level)
		}

		// Resources
		for _, r := range req.CastleConfig.Resources {
			state.Resources[protoToModelResourceType(r.Type)] = r.Amount
		}

		// TODO: researched technologies
	}

	return state
}

// protoTargetsToModelTargets converts proto TargetLevels to model targets
func protoTargetsToModelTargets(targets *pb.TargetLevels) map[models.BuildingType]int {
	result := make(map[models.BuildingType]int)
	if targets != nil {
		for _, t := range targets.Targets {
			result[protoToModelBuildingType(t.Type)] = int(t.Level)
		}
	}
	return result
}

// buildingActionToProto converts model BuildingUpgradeAction to proto BuildingAction
func buildingActionToProto(action models.BuildingUpgradeAction) *pb.BuildingAction {
	return &pb.BuildingAction{
		BuildingType:     modelToProtoBuildingType(action.BuildingType),
		FromLevel:        int32(action.FromLevel),
		ToLevel:          int32(action.ToLevel),
		StartTimeSeconds: int32(action.StartTime),
		EndTimeSeconds:   int32(action.EndTime),
		Costs:            costsToProtoResources(action.Costs),
		FoodUsed:         int32(action.FoodUsed),
		FoodCapacity:     int32(action.FoodCapacity),
	}
}

// Solve implements the Solve RPC
func (s *server) Solve(ctx context.Context, req *pb.SolveRequest) (*pb.SolveResponse, error) {
	log.Printf("Received Solve request")

	// Convert proto to internal types
	initialState := protoRequestToGameState(req)
	targetLevels := protoTargetsToModelTargets(req.TargetLevels)

	// Use default targets if none provided
	if len(targetLevels) == 0 {
		targetLevels = map[models.BuildingType]int{
			models.Lumberjack:     30,
			models.Quarry:         30,
			models.OreMine:        30,
			models.Farm:           30,
			models.WoodStore:      20,
			models.StoneStore:     20,
			models.OreStore:       20,
			models.Keep:           10,
			models.Arsenal:        30,
			models.Library:        10,
			models.Tavern:         10,
			models.Market:         8,
			models.Fortifications: 20,
		}
	}

	// Run solver
	solution, bestStrategy, _ := solver.SolveAllStrategies(s.buildings, s.technologies, initialState, targetLevels)

	// Convert solution to proto
	response := &pb.SolveResponse{
		TotalTimeSeconds: int32(solution.TotalTimeSeconds),
		Strategy:         bestStrategy.String(),
	}

	// Add building actions
	for _, action := range solution.BuildingActions {
		response.BuildingActions = append(response.BuildingActions, buildingActionToProto(action))
	}

	// Set next action (first in list)
	if len(solution.BuildingActions) > 0 {
		response.NextAction = buildingActionToProto(solution.BuildingActions[0])
	}

	log.Printf("Returning solution with %d actions, strategy: %s", len(response.BuildingActions), response.Strategy)
	return response, nil
}

// GetNextAction implements the GetNextAction RPC
func (s *server) GetNextAction(ctx context.Context, req *pb.SolveRequest) (*pb.BuildingAction, error) {
	log.Printf("Received GetNextAction request")

	response, err := s.Solve(ctx, req)
	if err != nil {
		return nil, err
	}

	if response.NextAction == nil {
		return &pb.BuildingAction{}, nil
	}

	return response.NextAction, nil
}

func main() {
	flag.Parse()

	// Load game data
	buildings, err := loader.LoadBuildings(*dataDir)
	if err != nil {
		log.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies(*dataDir)
	if err != nil {
		log.Printf("Warning: could not load technologies: %v", err)
		technologies = make(map[string]*models.Technology)
	}

	log.Printf("Loaded %d buildings, %d technologies", len(buildings), len(technologies))

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterCastleSolverServiceServer(s, &server{
		buildings:    buildings,
		technologies: technologies,
	})

	log.Printf("gRPC server listening on port %d", *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
