package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sort"

	"google.golang.org/grpc"

	"github.com/napolitain/solver-lnk/internal/converter"
	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	v3 "github.com/napolitain/solver-lnk/internal/solver/v3"
	"github.com/napolitain/solver-lnk/internal/solver/units"
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

// Solve implements the Solve RPC
func (s *server) Solve(ctx context.Context, req *pb.SolveRequest) (*pb.SolveResponse, error) {
	log.Printf("Received Solve request")

	// Convert proto to internal types
	initialState := converter.ProtoRequestToGameState(req)
	targetLevels := converter.ProtoTargetsToModelTargets(req.TargetLevels)

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
	solution, bestStrategy, _ := v3.SolveAllStrategies(s.buildings, s.technologies, initialState, targetLevels)

	// Convert solution to proto
	response := &pb.SolveResponse{
		TotalTimeSeconds: int32(solution.TotalTimeSeconds),
		Strategy:         bestStrategy,
	}

	// Add building actions (legacy)
	for _, action := range solution.BuildingActions {
		response.BuildingActions = append(response.BuildingActions, converter.BuildingActionToProto(action))
	}

	// Add research actions (legacy)
	for _, action := range solution.ResearchActions {
		response.ResearchActions = append(response.ResearchActions, converter.ResearchActionToProto(action))
	}

	// Build unified timeline - merge building and research actions chronologically
	response.Timeline = s.buildUnifiedTimeline(solution)

	// Set next immediate action (unified)
	if len(response.Timeline) > 0 {
		response.NextImmediateAction = response.Timeline[0]
	}

	// Set legacy next action fields for backward compatibility
	if len(solution.BuildingActions) > 0 {
		response.NextAction = converter.BuildingActionToProto(solution.BuildingActions[0])
	}
	if len(solution.ResearchActions) > 0 {
		response.NextResearchAction = converter.ResearchActionToProto(solution.ResearchActions[0])
	}

	// Check if build order is complete (no more building actions needed)
	buildOrderComplete := len(solution.BuildingActions) == 0
	if buildOrderComplete {
		log.Printf("Build order complete - generating units recommendation")
		response.UnitsRecommendation = s.generateUnitsRecommendation(initialState)
	}

	log.Printf("Returning solution with %d actions in timeline, strategy: %s",
		len(response.Timeline), response.Strategy)
	return response, nil
}

// buildUnifiedTimeline merges building and research actions into a chronological timeline
func (s *server) buildUnifiedTimeline(solution *models.Solution) []*pb.Action {
	var timeline []*pb.Action

	// Add all building actions
	for _, action := range solution.BuildingActions {
		timeline = append(timeline, converter.BuildingActionToUnifiedAction(action))
	}

	// Add all research actions
	for _, action := range solution.ResearchActions {
		timeline = append(timeline, converter.ResearchActionToUnifiedAction(action))
	}

	// Sort by start time
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].StartTimeSeconds < timeline[j].StartTimeSeconds
	})

	return timeline
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

// generateUnitsRecommendation creates a units recommendation based on current castle state
func (s *server) generateUnitsRecommendation(state *models.GameState) *pb.UnitsRecommendation {
	// Calculate food available for units based on Farm level
	farmLevel := state.BuildingLevels[models.Farm]
	farmBuilding := s.buildings[models.Farm]

	var foodCapacity int
	if farmBuilding != nil {
		if levelData := farmBuilding.GetLevelData(farmLevel); levelData != nil && levelData.StorageCapacity != nil {
			foodCapacity = *levelData.StorageCapacity
		}
	}
	if foodCapacity == 0 {
		foodCapacity = units.MaxFoodCapacity
	}

	// Calculate food used by buildings
	foodUsedByBuildings := 0
	for bt, level := range state.BuildingLevels {
		if building, ok := s.buildings[bt]; ok {
			for l := 1; l <= level; l++ {
				if levelData := building.GetLevelData(l); levelData != nil {
					foodUsedByBuildings += levelData.Costs.Food
				}
			}
		}
	}

	foodAvailable := foodCapacity - foodUsedByBuildings
	if foodAvailable < 0 {
		foodAvailable = 0
	}

	// Calculate resource production per hour
	resourceProdPerHour := 0.0
	productionBuildings := map[models.BuildingType]bool{
		models.Lumberjack: true,
		models.Quarry:     true,
		models.OreMine:    true,
	}
	for bt, level := range state.BuildingLevels {
		if !productionBuildings[bt] {
			continue
		}
		if building, ok := s.buildings[bt]; ok {
			if levelData := building.GetLevelData(level); levelData != nil && levelData.ProductionRate != nil {
				resourceProdPerHour += *levelData.ProductionRate
			}
		}
	}

	// Market distance based on Keep level (default 25 for Keep 10)
	marketDistance := int32(25)

	// Create units solver and solve
	unitsSolver := units.NewSolverWithConfig(int32(foodAvailable), int32(resourceProdPerHour), marketDistance)
	solution := unitsSolver.Solve()

	// Convert to proto
	recommendation := &pb.UnitsRecommendation{
		TotalFood:          int32(solution.TotalFood),
		TotalThroughput:    solution.TotalThroughput,
		DefenseVsCavalry:   int32(solution.DefenseVsCavalry),
		DefenseVsInfantry:  int32(solution.DefenseVsInfantry),
		DefenseVsArtillery: int32(solution.DefenseVsArtillery),
		SilverPerHour:      solution.SilverPerHour,
		BuildOrderComplete: true,
	}

	// Add unit counts
	for _, u := range units.AllUnits() {
		count := solution.UnitCounts[u.Name]
		if count > 0 {
			recommendation.UnitCounts = append(recommendation.UnitCounts, &pb.UnitCount{
				Type:  converter.UnitNameToProto(u.Name),
				Count: int32(count),
			})
		}
	}

	log.Printf("Units recommendation: %d food, %.0f throughput, defense (cav/inf/art): %d/%d/%d",
		recommendation.TotalFood, recommendation.TotalThroughput,
		recommendation.DefenseVsCavalry, recommendation.DefenseVsInfantry, recommendation.DefenseVsArtillery)

	return recommendation
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
