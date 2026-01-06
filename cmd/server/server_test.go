package main

import (
	"context"
	"testing"
	"time"

	"github.com/napolitain/solver-lnk/internal/converter"
	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	castle "github.com/napolitain/solver-lnk/internal/solver/castle"
	pb "github.com/napolitain/solver-lnk/proto"
)

// testServer creates a server instance for testing without starting the gRPC listener
func testServer(t *testing.T) *server {
	t.Helper()

	buildings, err := loader.LoadBuildings("../../data")
	if err != nil {
		t.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies("../../data")
	if err != nil {
		t.Logf("Warning: could not load technologies: %v", err)
		technologies = make(map[string]*models.Technology)
	}

	missions := loader.LoadMissions()

	return &server{
		buildings:    buildings,
		technologies: technologies,
		missions:     missions,
	}
}

// TestSolveMatchesDirectSolver verifies that the gRPC Solve endpoint returns
// the same build order as calling the solver directly with identical inputs.
func TestSolveMatchesDirectSolver(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	// Test with default empty state
	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
	}

	// Call gRPC endpoint
	resp, err := srv.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Call solver directly with same inputs
	initialState := converter.ProtoRequestToGameState(req)
	targetLevels := map[models.BuildingType]int{
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

	targetTechs := models.GetTargetTechnologies(srv.technologies, targetLevels[models.Library])
	targetUnits := models.GetTargetUnits()
	solver := castle.NewSolver(srv.buildings, srv.technologies, srv.missions, targetLevels, targetTechs, targetUnits)
	solution := solver.Solve(initialState)

	// Verify total time matches
	if resp.TotalTimeSeconds != int32(solution.TotalTimeSeconds) {
		t.Errorf("TotalTimeSeconds mismatch: gRPC=%d, direct=%d",
			resp.TotalTimeSeconds, solution.TotalTimeSeconds)
	}

	// Verify building action count matches
	if len(resp.BuildingActions) != len(solution.BuildingActions) {
		t.Errorf("BuildingActions count mismatch: gRPC=%d, direct=%d",
			len(resp.BuildingActions), len(solution.BuildingActions))
	}

	// Verify research action count matches
	if len(resp.ResearchActions) != len(solution.ResearchActions) {
		t.Errorf("ResearchActions count mismatch: gRPC=%d, direct=%d",
			len(resp.ResearchActions), len(solution.ResearchActions))
	}

	// Verify each building action matches
	for i, action := range solution.BuildingActions {
		if i >= len(resp.BuildingActions) {
			break
		}
		grpcAction := resp.BuildingActions[i]
		expectedType := converter.ModelToProtoBuildingType(action.BuildingType)

		if grpcAction.BuildingType != expectedType {
			t.Errorf("BuildingAction[%d] type mismatch: gRPC=%v, direct=%v",
				i, grpcAction.BuildingType, expectedType)
		}
		if grpcAction.FromLevel != int32(action.FromLevel) {
			t.Errorf("BuildingAction[%d] FromLevel mismatch: gRPC=%d, direct=%d",
				i, grpcAction.FromLevel, action.FromLevel)
		}
		if grpcAction.ToLevel != int32(action.ToLevel) {
			t.Errorf("BuildingAction[%d] ToLevel mismatch: gRPC=%d, direct=%d",
				i, grpcAction.ToLevel, action.ToLevel)
		}
		if grpcAction.StartTimeSeconds != int32(action.StartTime) {
			t.Errorf("BuildingAction[%d] StartTime mismatch: gRPC=%d, direct=%d",
				i, grpcAction.StartTimeSeconds, action.StartTime)
		}
		if grpcAction.EndTimeSeconds != int32(action.EndTime) {
			t.Errorf("BuildingAction[%d] EndTime mismatch: gRPC=%d, direct=%d",
				i, grpcAction.EndTimeSeconds, action.EndTime)
		}
	}

	// Verify each research action matches
	for i, action := range solution.ResearchActions {
		if i >= len(resp.ResearchActions) {
			break
		}
		grpcAction := resp.ResearchActions[i]

		if grpcAction.TechnologyName != action.TechnologyName {
			t.Errorf("ResearchAction[%d] name mismatch: gRPC=%s, direct=%s",
				i, grpcAction.TechnologyName, action.TechnologyName)
		}
		if grpcAction.StartTimeSeconds != int32(action.StartTime) {
			t.Errorf("ResearchAction[%d] StartTime mismatch: gRPC=%d, direct=%d",
				i, grpcAction.StartTimeSeconds, action.StartTime)
		}
		if grpcAction.EndTimeSeconds != int32(action.EndTime) {
			t.Errorf("ResearchAction[%d] EndTime mismatch: gRPC=%d, direct=%d",
				i, grpcAction.EndTimeSeconds, action.EndTime)
		}
	}
}

// TestSolveWithCustomTargets verifies that custom target levels are respected
func TestSolveWithCustomTargets(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	// Request with custom targets
	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
		TargetLevels: &pb.TargetLevels{
			Targets: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 5},
				{Type: pb.BuildingType_QUARRY, Level: 5},
				{Type: pb.BuildingType_ORE_MINE, Level: 5},
			},
		},
	}

	resp, err := srv.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Call solver directly with same custom targets
	initialState := converter.ProtoRequestToGameState(req)
	targetLevels := converter.ProtoTargetsToModelTargets(req.TargetLevels)

	targetTechs := models.GetTargetTechnologies(srv.technologies, targetLevels[models.Library])
	targetUnits := models.GetTargetUnits()
	solver := castle.NewSolver(srv.buildings, srv.technologies, srv.missions, targetLevels, targetTechs, targetUnits)
	solution := solver.Solve(initialState)

	// Verify results match
	if resp.TotalTimeSeconds != int32(solution.TotalTimeSeconds) {
		t.Errorf("TotalTimeSeconds mismatch: gRPC=%d, direct=%d",
			resp.TotalTimeSeconds, solution.TotalTimeSeconds)
	}

	if len(resp.BuildingActions) != len(solution.BuildingActions) {
		t.Errorf("BuildingActions count mismatch: gRPC=%d, direct=%d",
			len(resp.BuildingActions), len(solution.BuildingActions))
	}
}

// TestSolveWithInitialState verifies that initial building levels are respected
func TestSolveWithInitialState(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	// Request with initial building levels
	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{
			BuildingLevels: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 10},
				{Type: pb.BuildingType_QUARRY, Level: 10},
				{Type: pb.BuildingType_ORE_MINE, Level: 10},
				{Type: pb.BuildingType_FARM, Level: 10},
			},
			Resources: []*pb.Resource{
				{Type: pb.ResourceType_WOOD, Amount: 1000},
				{Type: pb.ResourceType_STONE, Amount: 1000},
				{Type: pb.ResourceType_IRON, Amount: 1000},
			},
		},
		TargetLevels: &pb.TargetLevels{
			Targets: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 15},
				{Type: pb.BuildingType_QUARRY, Level: 15},
				{Type: pb.BuildingType_ORE_MINE, Level: 15},
			},
		},
	}

	resp, err := srv.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Call solver directly
	initialState := converter.ProtoRequestToGameState(req)
	targetLevels := converter.ProtoTargetsToModelTargets(req.TargetLevels)

	targetTechs := models.GetTargetTechnologies(srv.technologies, targetLevels[models.Library])
	targetUnits := models.GetTargetUnits()
	solver := castle.NewSolver(srv.buildings, srv.technologies, srv.missions, targetLevels, targetTechs, targetUnits)
	solution := solver.Solve(initialState)

	// Verify results match
	if resp.TotalTimeSeconds != int32(solution.TotalTimeSeconds) {
		t.Errorf("TotalTimeSeconds mismatch: gRPC=%d, direct=%d",
			resp.TotalTimeSeconds, solution.TotalTimeSeconds)
	}

	if len(resp.BuildingActions) != len(solution.BuildingActions) {
		t.Errorf("BuildingActions count mismatch: gRPC=%d, direct=%d",
			len(resp.BuildingActions), len(solution.BuildingActions))
	}

	// Verify building actions start from level 10 (not 0)
	for _, action := range resp.BuildingActions {
		if action.BuildingType == pb.BuildingType_LUMBERJACK && action.FromLevel < 10 {
			t.Errorf("Lumberjack action starts from level %d, expected >= 10", action.FromLevel)
		}
	}
}

// TestSolveTimelineChronological verifies timeline is sorted by start time
func TestSolveTimelineChronological(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
		TargetLevels: &pb.TargetLevels{
			Targets: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 10},
				{Type: pb.BuildingType_LIBRARY, Level: 5},
			},
		},
	}

	resp, err := srv.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Verify timeline is sorted by start time
	for i := 1; i < len(resp.Timeline); i++ {
		if resp.Timeline[i].StartTimeSeconds < resp.Timeline[i-1].StartTimeSeconds {
			t.Errorf("Timeline not chronological at index %d: %d < %d",
				i, resp.Timeline[i].StartTimeSeconds, resp.Timeline[i-1].StartTimeSeconds)
		}
	}

	// Verify NextImmediateAction matches first timeline action
	if len(resp.Timeline) > 0 && resp.NextImmediateAction != nil {
		if resp.NextImmediateAction.StartTimeSeconds != resp.Timeline[0].StartTimeSeconds {
			t.Errorf("NextImmediateAction doesn't match first timeline action")
		}
	}
}

// TestSolveDeterministic verifies that repeated calls produce identical results
func TestSolveDeterministic(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
		TargetLevels: &pb.TargetLevels{
			Targets: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 15},
				{Type: pb.BuildingType_QUARRY, Level: 15},
				{Type: pb.BuildingType_ORE_MINE, Level: 15},
			},
		},
	}

	// Run multiple times
	var results []*pb.SolveResponse
	for i := 0; i < 5; i++ {
		resp, err := srv.Solve(ctx, req)
		if err != nil {
			t.Fatalf("Solve failed on iteration %d: %v", i, err)
		}
		results = append(results, resp)
	}

	// All results should be identical
	first := results[0]
	for i := 1; i < len(results); i++ {
		if results[i].TotalTimeSeconds != first.TotalTimeSeconds {
			t.Errorf("Non-deterministic: TotalTimeSeconds differs at iteration %d", i)
		}
		if len(results[i].BuildingActions) != len(first.BuildingActions) {
			t.Errorf("Non-deterministic: BuildingActions count differs at iteration %d", i)
		}
		if len(results[i].Timeline) != len(first.Timeline) {
			t.Errorf("Non-deterministic: Timeline count differs at iteration %d", i)
		}
	}
}

// TestGetNextAction verifies GetNextAction returns the same first action as Solve
func TestGetNextAction(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
		TargetLevels: &pb.TargetLevels{
			Targets: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 5},
			},
		},
	}

	// Get full solution
	fullResp, err := srv.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Get next action only
	nextAction, err := srv.GetNextAction(ctx, req)
	if err != nil {
		t.Fatalf("GetNextAction failed: %v", err)
	}

	// Should match
	if fullResp.NextAction != nil && nextAction != nil {
		if nextAction.BuildingType != fullResp.NextAction.BuildingType {
			t.Errorf("GetNextAction type mismatch: %v vs %v",
				nextAction.BuildingType, fullResp.NextAction.BuildingType)
		}
		if nextAction.ToLevel != fullResp.NextAction.ToLevel {
			t.Errorf("GetNextAction level mismatch: %v vs %v",
				nextAction.ToLevel, fullResp.NextAction.ToLevel)
		}
	}
}

// BenchmarkSolve measures the performance of the Solve endpoint
func BenchmarkSolve(b *testing.B) {
	buildings, err := loader.LoadBuildings("../../data")
	if err != nil {
		b.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies("../../data")
	if err != nil {
		technologies = make(map[string]*models.Technology)
	}

	missions := loader.LoadMissions()

	srv := &server{
		buildings:    buildings,
		technologies: technologies,
		missions:     missions,
	}

	ctx := context.Background()
	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.Solve(ctx, req)
		if err != nil {
			b.Fatalf("Solve failed: %v", err)
		}
	}
}

// BenchmarkSolveWithCustomTargets benchmarks with custom targets
func BenchmarkSolveWithCustomTargets(b *testing.B) {
	buildings, err := loader.LoadBuildings("../../data")
	if err != nil {
		b.Fatalf("Failed to load buildings: %v", err)
	}

	technologies, err := loader.LoadTechnologies("../../data")
	if err != nil {
		technologies = make(map[string]*models.Technology)
	}

	missions := loader.LoadMissions()

	srv := &server{
		buildings:    buildings,
		technologies: technologies,
		missions:     missions,
	}

	ctx := context.Background()
	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
		TargetLevels: &pb.TargetLevels{
			Targets: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 30},
				{Type: pb.BuildingType_QUARRY, Level: 30},
				{Type: pb.BuildingType_ORE_MINE, Level: 30},
				{Type: pb.BuildingType_FARM, Level: 30},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.Solve(ctx, req)
		if err != nil {
			b.Fatalf("Solve failed: %v", err)
		}
	}
}

// TestSolvePerformanceTracing verifies that performance metrics are captured
func TestSolvePerformanceTracing(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	req := &pb.SolveRequest{
		CastleConfig: &pb.CastleConfig{},
		TargetLevels: &pb.TargetLevels{
			Targets: []*pb.BuildingLevel{
				{Type: pb.BuildingType_LUMBERJACK, Level: 10},
			},
		},
	}

	start := time.Now()
	_, err := srv.Solve(ctx, req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Solve should complete in reasonable time (< 1 second for small targets)
	if elapsed > time.Second {
		t.Errorf("Solve took too long: %v", elapsed)
	}

	t.Logf("Solve completed in %v", elapsed)
}
