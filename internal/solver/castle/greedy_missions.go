package castle

import (
	"github.com/napolitain/solver-lnk/internal/models"
)

// SimulationStateWithMissions extends SimulationState with mission tracking
type SimulationStateWithMissions struct {
	*SimulationState
	Missions *MissionState
}

// MissionSolution extends Solution with mission data
type MissionSolution struct {
	*models.Solution
	MissionActions    []MissionAction
	TotalMissionGain  map[models.ResourceType]int
	MissionsCompleted int
}

// SolveWithMissions runs the greedy simulation with mission support
// This is a separate function to avoid breaking the existing Solve() behavior
func (s *GreedySolver) SolveWithMissions(enableMissions bool) *MissionSolution {
	if !enableMissions {
		// Just wrap the normal solution
		sol := s.Solve()
		return &MissionSolution{
			Solution:         sol,
			MissionActions:   []MissionAction{},
			TotalMissionGain: make(map[models.ResourceType]int),
		}
	}

	state := s.initStateWithMissions()
	queue := s.createPrioritizedQueue()

	for len(queue) > 0 {
		// Process any completed missions first (returns resources)
		s.processCompletedMissions(state)

		// Try to start new missions if beneficial
		s.tryStartMissions(state, queue)

		// Wait for building queue if needed
		if state.TimeMinutes < state.BuildingQueueFreeAt {
			// But check if a mission completes before that
			nextMissionEnd := state.Missions.NextMissionCompletionTime()
			if nextMissionEnd > 0 && nextMissionEnd < state.BuildingQueueFreeAt {
				// Advance to mission completion instead
				s.advanceTimeWithMissions(state, nextMissionEnd-state.TimeMinutes)
				continue
			}
			s.advanceTimeWithMissions(state, state.BuildingQueueFreeAt-state.TimeMinutes)
		}

		// Check if we should trigger production tech research based on strategy
		if s.shouldResearchProductionTech(state.SimulationState, &queue) {
			continue
		}

		// Select next upgrade
		nextUpgrade := s.selectNextUpgrade(state.SimulationState, queue)
		if nextUpgrade == nil {
			break
		}

		bType, targetLevel, queueIdx := nextUpgrade.bType, nextUpgrade.targetLevel, nextUpgrade.queueIdx

		currentLevel := state.BuildingLevels[bType]
		if currentLevel >= targetLevel {
			queue = removeFromQueue(queue, queueIdx)
			continue
		}

		toLevel := currentLevel + 1
		building := s.Buildings[bType]
		if building == nil {
			queue = removeFromQueue(queue, queueIdx)
			continue
		}

		levelData := building.GetLevelData(toLevel)
		if levelData == nil {
			queue = removeFromQueue(queue, queueIdx)
			continue
		}

		costs := levelData.Costs

		// Check technology prerequisite
		if techName := s.checkTechPrerequisite(state.SimulationState, building, toLevel); techName != "" {
			s.scheduleResearch(state.SimulationState, techName, &queue)
			continue
		}

		// Check food capacity
		foodCost := costs[models.Food]
		if state.FoodUsed+foodCost > state.FoodCapacity {
			farmLevel := state.BuildingLevels[models.Farm]
			queue = insertAtFront(queue, queueItem{models.Farm, farmLevel + 1})
			continue
		}

		// Check storage capacity
		if ok, storageNeeded := s.checkStorageCapacity(state.SimulationState, costs); !ok {
			if storageNeeded != nil {
				if storageNeeded.bType != bType || storageNeeded.targetLevel != targetLevel {
					queue = insertAtFront(queue, *storageNeeded)
				}
			}
			continue
		}

		// Check if we can afford - but now consider if missions could help
		canAfford, waitTime := s.canAffordOrWaitTime(state.SimulationState, costs)
		if !canAfford {
			if waitTime < 0 {
				queue = removeFromQueue(queue, queueIdx)
				continue
			}
			// While waiting, process missions
			s.advanceTimeWithMissions(state, waitTime)
		}

		// Start upgrade (same as original)
		startTime := state.TimeMinutes

		for _, resType := range models.AllResourceTypes() {
			cost := costs[resType]
			if cost == 0 {
				continue
			}
			state.Resources[resType] -= float64(cost)
		}
		state.FoodUsed += foodCost

		durationMinutes := max(1, levelData.BuildTimeSeconds/60)
		state.BuildingQueueFreeAt = state.TimeMinutes + durationMinutes

		s.advanceTimeWithMissions(state, durationMinutes)

		state.BuildingLevels[bType] = toLevel

		s.updateProductionRates(state.SimulationState, building, bType, toLevel)
		s.updateStorageCaps(state.SimulationState, building, bType, toLevel)

		if bType == models.Farm {
			state.FoodCapacity = s.getFoodCapacityForLevel(toLevel)
		}

		state.CompletedActions = append(state.CompletedActions, models.BuildingUpgradeAction{
			BuildingType: bType,
			FromLevel:    currentLevel,
			ToLevel:      toLevel,
			StartTime:    startTime * 60,
			EndTime:      state.TimeMinutes * 60,
			Costs:        costs,
			FoodUsed:     state.FoodUsed,
			FoodCapacity: state.FoodCapacity,
		})

		if state.BuildingLevels[bType] >= targetLevel {
			queue = removeFromQueue(queue, queueIdx)
		}
	}

	// Process any remaining missions
	s.finishRemainingMissions(state)

	// Research remaining technologies
	s.researchRemainingTechs(state.SimulationState)

	// Calculate total mission gains
	totalGain := make(map[models.ResourceType]int)
	for _, ma := range state.Missions.CompletedMissions {
		for rt, amount := range ma.ResourcesGained {
			totalGain[rt] += amount
		}
	}

	return &MissionSolution{
		Solution: &models.Solution{
			BuildingActions:  state.CompletedActions,
			ResearchActions:  state.ResearchActions,
			TotalTimeSeconds: state.TimeMinutes * 60,
			FinalState: &models.GameState{
				BuildingLevels:         state.BuildingLevels,
				Resources:              state.Resources,
				ResearchedTechnologies: state.ResearchedTechnologies,
				StorageCaps:            state.StorageCaps,
				ProductionRates:        state.ProductionRates,
			},
		},
		MissionActions:    state.Missions.CompletedMissions,
		TotalMissionGain:  totalGain,
		MissionsCompleted: len(state.Missions.CompletedMissions),
	}
}

func (s *GreedySolver) initStateWithMissions() *SimulationStateWithMissions {
	baseState := s.initState()
	missionState := NewMissionState()

	// Give starting units for missions (simplified - would come from game state)
	// For testing, assume we start with some basic units
	// In reality, this would need to track unit training in the build order
	missionState.TrainUnits(models.Spearman, 20)

	return &SimulationStateWithMissions{
		SimulationState: baseState,
		Missions:        missionState,
	}
}

func (s *GreedySolver) advanceTimeWithMissions(state *SimulationStateWithMissions, minutes int) {
	// Process missions that complete during this time window
	endTime := state.TimeMinutes + minutes

	for {
		nextCompletion := state.Missions.NextMissionCompletionTime()
		if nextCompletion < 0 || nextCompletion > endTime {
			break
		}

		// Advance to mission completion
		delta := nextCompletion - state.TimeMinutes
		if delta > 0 {
			s.advanceTime(state.SimulationState, delta)
		}

		// Complete the mission and add resources
		completed := state.Missions.CompleteMissions(state.TimeMinutes)
		for _, ma := range completed {
			for rt, amount := range ma.ResourcesGained {
				state.Resources[rt] += float64(amount)
				// Cap at storage
				if cap, ok := state.StorageCaps[rt]; ok && state.Resources[rt] > float64(cap) {
					state.Resources[rt] = float64(cap)
				}
			}
		}
	}

	// Advance remaining time
	remaining := endTime - state.TimeMinutes
	if remaining > 0 {
		s.advanceTime(state.SimulationState, remaining)
	}
}

func (s *GreedySolver) processCompletedMissions(state *SimulationStateWithMissions) {
	completed := state.Missions.CompleteMissions(state.TimeMinutes)
	for _, ma := range completed {
		for rt, amount := range ma.ResourcesGained {
			state.Resources[rt] += float64(amount)
			if cap, ok := state.StorageCaps[rt]; ok && state.Resources[rt] > float64(cap) {
				state.Resources[rt] = float64(cap)
			}
		}
	}
}

func (s *GreedySolver) tryStartMissions(state *SimulationStateWithMissions, queue []queueItem) {
	tavernLevel := state.BuildingLevels[models.Tavern]
	if tavernLevel < 1 {
		return // No tavern, no missions
	}

	// Calculate what resources we need for the next few upgrades
	neededResources := s.calculateNeededResources(state.SimulationState, queue, 3)

	// Find the best mission for current state
	best := state.Missions.GetBestMissionForState(
		tavernLevel,
		state.Resources,
		neededResources,
		state.ProductionRates,
	)

	if best == nil {
		return
	}

	// Check if we can afford mission costs
	for rt, cost := range best.ResourceCosts {
		if state.Resources[rt] < float64(cost) {
			return // Can't afford
		}
	}

	// Start the mission
	running := state.Missions.StartMission(best, state.TimeMinutes)
	if running != nil {
		// Deduct mission costs
		for rt, cost := range best.ResourceCosts {
			state.Resources[rt] -= float64(cost)
		}
	}
}

func (s *GreedySolver) calculateNeededResources(state *SimulationState, queue []queueItem, lookahead int) map[models.ResourceType]float64 {
	needed := make(map[models.ResourceType]float64)

	count := 0
	for _, item := range queue {
		if count >= lookahead {
			break
		}

		building := s.Buildings[item.bType]
		if building == nil {
			continue
		}

		currentLevel := state.BuildingLevels[item.bType]
		for level := currentLevel + 1; level <= item.targetLevel && count < lookahead; level++ {
			levelData := building.GetLevelData(level)
			if levelData == nil {
				continue
			}

			for rt, cost := range levelData.Costs {
				needed[rt] += float64(cost)
			}
			count++
		}
	}

	return needed
}

func (s *GreedySolver) finishRemainingMissions(state *SimulationStateWithMissions) {
	// Wait for all running missions to complete
	for len(state.Missions.RunningMissions) > 0 {
		nextCompletion := state.Missions.NextMissionCompletionTime()
		if nextCompletion < 0 {
			break
		}

		if nextCompletion > state.TimeMinutes {
			s.advanceTimeWithMissions(state, nextCompletion-state.TimeMinutes)
		}

		s.processCompletedMissions(state)
	}
}

// SolveAllStrategiesWithMissions tries all strategies with mission support
func SolveAllStrategiesWithMissions(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	initialState *models.GameState,
	targetLevels map[models.BuildingType]int,
	enableMissions bool,
) (*MissionSolution, ResourceStrategy, []StrategyResult) {
	// For now, just use the best strategy from normal solve and add missions
	bestSolution, bestStrategy, results := SolveAllStrategies(buildings, technologies, initialState, targetLevels)

	if !enableMissions {
		return &MissionSolution{
			Solution:         bestSolution,
			MissionActions:   []MissionAction{},
			TotalMissionGain: make(map[models.ResourceType]int),
		}, bestStrategy, results
	}

	// Re-run with missions using best strategy
	stateCopy := &models.GameState{
		BuildingLevels:         make(map[models.BuildingType]int),
		Resources:              make(map[models.ResourceType]float64),
		ResearchedTechnologies: make(map[string]bool),
	}
	for k, v := range initialState.BuildingLevels {
		stateCopy.BuildingLevels[k] = v
	}
	for k, v := range initialState.Resources {
		stateCopy.Resources[k] = v
	}
	for k, v := range initialState.ResearchedTechnologies {
		stateCopy.ResearchedTechnologies[k] = v
	}

	solver := NewGreedySolverWithStrategy(buildings, technologies, stateCopy, targetLevels, bestStrategy)
	missionSolution := solver.SolveWithMissions(true)

	return missionSolution, bestStrategy, results
}
