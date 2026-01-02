package v4

import (
	"sort"

	"github.com/napolitain/solver-lnk/internal/models"
)

// Solver is the V4 event-driven solver
type Solver struct {
	Buildings    map[models.BuildingType]*models.Building
	Technologies map[string]*models.Technology
	Missions     []*models.Mission
	TargetLevels map[models.BuildingType]int
}

// NewSolver creates a new V4 solver
func NewSolver(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	missions []*models.Mission,
	targetLevels map[models.BuildingType]int,
) *Solver {
	return &Solver{
		Buildings:    buildings,
		Technologies: technologies,
		Missions:     missions,
		TargetLevels: targetLevels,
	}
}

// Solve runs the event-driven solver and returns a solution
func (s *Solver) Solve(initialState *models.GameState) *models.Solution {
	state := NewState(initialState)
	s.initializeState(state)

	events := NewEventQueue()

	var buildingActions []models.BuildingUpgradeAction
	var researchActions []models.ResearchAction

	// Bootstrap: evaluate initial state
	events.Push(Event{Time: 0, Type: EventStateChanged})

	maxIterations := 1000000
	iterations := 0

	for !events.Empty() && !s.allTargetsReached(state) && iterations < maxIterations {
		iterations++

		event := events.Pop()

		// Advance time (accumulate resources)
		if event.Time > state.Now {
			s.advanceTime(state, event.Time-state.Now)
		}

		s.processEvent(state, event, events, &buildingActions, &researchActions)
	}

	// Process any remaining completion events (for buildings that were started but not yet recorded)
	for !events.Empty() {
		event := events.Pop()

		// Only process completion events, skip StateChanged
		if event.Type == EventStateChanged {
			continue
		}

		// Advance time
		if event.Time > state.Now {
			s.advanceTime(state, event.Time-state.Now)
		}

		s.processEvent(state, event, events, &buildingActions, &researchActions)
	}

	// Research ALL remaining technologies after buildings are done
	s.researchRemainingTechs(state, &researchActions)

	// Calculate final time
	finalTime := state.Now
	if state.BuildingQueueFreeAt > finalTime {
		finalTime = state.BuildingQueueFreeAt
	}
	if state.ResearchQueueFreeAt > finalTime {
		finalTime = state.ResearchQueueFreeAt
	}

	return &models.Solution{
		BuildingActions:  buildingActions,
		ResearchActions:  researchActions,
		TotalTimeSeconds: finalTime,
		FinalState:       state.ToGameState(),
	}
}

// processEvent handles a single event
func (s *Solver) processEvent(
	state *State,
	event Event,
	events *EventQueue,
	buildingActions *[]models.BuildingUpgradeAction,
	researchActions *[]models.ResearchAction,
) {
	switch event.Type {
	case EventMissionComplete:
		s.handleMissionComplete(state, event, events)

	case EventBuildingComplete:
		s.handleBuildingComplete(state, event, events, buildingActions)

	case EventResearchComplete:
		s.handleResearchComplete(state, event, events, researchActions)

	case EventTrainingComplete:
		s.handleTrainingComplete(state, event, events)

	case EventStateChanged:
		s.handleStateChanged(state, events, buildingActions, researchActions)
	}
}

// handleMissionComplete processes a completed mission
func (s *Solver) handleMissionComplete(state *State, event Event, events *EventQueue) {
	ms := event.Payload.(*models.MissionState)

	// Add resources from mission rewards
	for _, reward := range ms.Mission.Rewards {
		avgReward := reward.AverageReward()
		state.AddResource(reward.Type, avgReward)
	}

	// Return units from mission
	state.Army.AddFrom(ms.AssignedUnits)
	state.UnitsOnMission.Subtract(ms.AssignedUnits)

	// Remove from running missions
	s.removeMissionFromRunning(state, ms)

	// Trigger re-evaluation
	events.PushIfNotExists(Event{Time: state.Now, Type: EventStateChanged})
}

// handleBuildingComplete processes a completed building
func (s *Solver) handleBuildingComplete(
	state *State,
	event Event,
	events *EventQueue,
	buildingActions *[]models.BuildingUpgradeAction,
) {
	ba := event.Payload.(*BuildingAction)

	// Update building level on completion
	state.SetBuildingLevel(ba.BuildingType, ba.ToLevel)

	// Update production rates and storage caps
	s.updateAfterBuild(state, ba)

	// Record action
	*buildingActions = append(*buildingActions, models.BuildingUpgradeAction{
		BuildingType: ba.BuildingType,
		FromLevel:    ba.FromLevel,
		ToLevel:      ba.ToLevel,
		StartTime:    state.Now - ba.Duration(),
		EndTime:      state.Now,
		Costs:        ba.Costs(),
		FoodUsed:     state.FoodUsed,
		FoodCapacity: state.FoodCapacity,
	})

	// Clear pending
	state.PendingBuilding = nil

	// Trigger re-evaluation
	events.PushIfNotExists(Event{Time: state.Now, Type: EventStateChanged})
}

// handleResearchComplete processes completed research
func (s *Solver) handleResearchComplete(
	state *State,
	event Event,
	events *EventQueue,
	researchActions *[]models.ResearchAction,
) {
	ra := event.Payload.(*ResearchAction)

	// Mark as researched
	state.ResearchedTechs[ra.Technology.Name] = true

	// Apply production bonus for production techs
	if ra.Technology.Name == "Beer tester" || ra.Technology.Name == "Wheelbarrow" {
		state.ProductionBonus += 0.05
	}

	// Record action
	*researchActions = append(*researchActions, models.ResearchAction{
		TechnologyName: ra.Technology.Name,
		StartTime:      state.Now - ra.Duration(),
		EndTime:        state.Now,
		Costs:          ra.Costs(),
		FoodUsed:       state.FoodUsed,
		FoodCapacity:   state.FoodCapacity,
	})

	// Clear pending
	state.PendingResearch = nil

	// Trigger re-evaluation
	events.PushIfNotExists(Event{Time: state.Now, Type: EventStateChanged})
}

// handleTrainingComplete processes a completed unit training
func (s *Solver) handleTrainingComplete(state *State, event Event, events *EventQueue) {
	ta := event.Payload.(*TrainUnitAction)

	// Add unit to army
	state.Army.Add(ta.UnitType, 1)

	// Clear pending
	state.PendingTraining = nil

	// Trigger re-evaluation
	events.PushIfNotExists(Event{Time: state.Now, Type: EventStateChanged})
}

// handleStateChanged makes decisions for all free queues
func (s *Solver) handleStateChanged(
	state *State,
	events *EventQueue,
	buildingActions *[]models.BuildingUpgradeAction,
	researchActions *[]models.ResearchAction,
) {
	// Building queue
	if state.Now >= state.BuildingQueueFreeAt && state.PendingBuilding == nil {
		s.tryStartBuilding(state, events)
	}

	// Research queue
	if state.Now >= state.ResearchQueueFreeAt && state.PendingResearch == nil {
		if action := s.pickBestResearchAction(state); action != nil {
			if s.canAfford(state, action.Costs()) && state.CanAffordFood(action.Costs().Food) {
				s.executeResearch(state, action, events)
			}
		}
	}

	// Training queue
	if state.Now >= state.TrainingQueueFreeAt && state.PendingTraining == nil {
		if action := s.pickBestTrainingAction(state); action != nil {
			if s.canAfford(state, action.Costs()) && state.CanAffordFood(action.FoodCost()) {
				s.executeTraining(state, action, events)
			}
		}
	}

	// Missions (can start multiple if units available)
	for {
		mission := s.pickBestMissionToStart(state)
		if mission == nil {
			break
		}
		if !s.canAfford(state, mission.ResourceCosts) {
			break
		}
		s.startMission(state, mission, events)
	}

	// Schedule wake-up when resources become available
	s.scheduleResourceWakeup(state, events)
}

// tryStartBuilding attempts to start a building, handling prerequisites
func (s *Solver) tryStartBuilding(state *State, events *EventQueue) {
	// Get best building action by ROI (ignoring prerequisites for now)
	action := s.pickBestBuildingActionIgnoringPrereqs(state)
	if action == nil {
		return
	}

	// Check if production tech (Beer Tester, Wheelbarrow) has better ROI
	// If so, prioritize Library upgrade to enable that research
	prodTechAction := s.getBestProductionTechAction(state)
	if prodTechAction != nil {
		buildingROI := s.buildingROI(state, action)
		techROI := s.productionTechROI(state, prodTechAction)
		if techROI > buildingROI {
			// Production tech is better - check if we need Library upgrade
			libraryLevel := state.GetBuildingLevel(models.Library)
			if libraryLevel < prodTechAction.RequiredLibraryLevel {
				// Need to upgrade Library first
				action = s.createLibraryUpgrade(state, prodTechAction.RequiredLibraryLevel)
				if action == nil {
					return
				}
			}
		}
	}

	// Check prerequisites and get the actual action to execute
	action = s.resolvePrerequisites(state, action)
	if action == nil {
		return
	}

	// Check if we can afford it
	costs := action.Costs()
	if !s.canAfford(state, costs) {
		return
	}

	// Check food
	if !state.CanAffordFood(costs.Food) {
		return
	}

	// Execute the action
	s.executeBuilding(state, action, events)
}

// pickBestBuildingActionIgnoringPrereqs selects the best building without checking prereqs
func (s *Solver) pickBestBuildingActionIgnoringPrereqs(state *State) *BuildingAction {
	var candidates []*BuildingAction

	for bt, target := range s.TargetLevels {
		current := state.GetBuildingLevel(bt)
		if current >= target {
			continue
		}

		building := s.Buildings[bt]
		if building == nil {
			continue
		}

		toLevel := current + 1
		levelData := building.GetLevelData(toLevel)
		if levelData == nil {
			continue
		}

		candidates = append(candidates, &BuildingAction{
			BuildingType: bt,
			FromLevel:    current,
			ToLevel:      toLevel,
			Building:     building,
			LevelData:    levelData,
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Sort by ROI
	sort.Slice(candidates, func(i, j int) bool {
		roiI := s.buildingROI(state, candidates[i])
		roiJ := s.buildingROI(state, candidates[j])
		if roiI != roiJ {
			return roiI > roiJ
		}
		return candidates[i].BuildingType < candidates[j].BuildingType
	})

	return candidates[0]
}

// resolvePrerequisites checks if action needs prerequisites and returns the prereq action if so
func (s *Solver) resolvePrerequisites(state *State, action *BuildingAction) *BuildingAction {
	costs := action.Costs()

	// Check food capacity first
	if state.FoodUsed+costs.Food > state.FoodCapacity {
		farmAction := s.createFarmUpgrade(state, state.FoodUsed+costs.Food)
		if farmAction != nil {
			return farmAction
		}
	}

	// Check storage capacity for wood
	if costs.Wood > state.GetStorageCap(models.Wood) {
		storageAction := s.createStorageUpgrade(state, models.Wood, costs.Wood)
		if storageAction != nil {
			return storageAction
		}
	}

	// Check storage capacity for stone
	if costs.Stone > state.GetStorageCap(models.Stone) {
		storageAction := s.createStorageUpgrade(state, models.Stone, costs.Stone)
		if storageAction != nil {
			return storageAction
		}
	}

	// Check storage capacity for iron
	if costs.Iron > state.GetStorageCap(models.Iron) {
		storageAction := s.createStorageUpgrade(state, models.Iron, costs.Iron)
		if storageAction != nil {
			return storageAction
		}
	}

	// Check technology prerequisites
	building := s.Buildings[action.BuildingType]
	if building != nil {
		if techName, ok := building.TechnologyPrerequisites[action.ToLevel]; ok {
			if !state.ResearchedTechs[techName] {
				tech := s.Technologies[techName]
				if tech != nil {
					libraryLevel := state.GetBuildingLevel(models.Library)
					if libraryLevel < tech.RequiredLibraryLevel {
						return s.createLibraryUpgrade(state, tech.RequiredLibraryLevel)
					}
					// Tech needs to be researched - this is handled by research queue
					// For now, skip this building until tech is researched
					return nil
				}
			}
		}
	}

	// No prerequisites needed, return original action
	return action
}

// createFarmUpgrade creates a Farm upgrade action to reach required food capacity
func (s *Solver) createFarmUpgrade(state *State, requiredFood int) *BuildingAction {
	farmBuilding := s.Buildings[models.Farm]
	if farmBuilding == nil {
		return nil
	}

	currentLevel := state.GetBuildingLevel(models.Farm)

	// Check tech prerequisites for next farm level
	nextLevel := currentLevel + 1
	if techName, ok := farmBuilding.TechnologyPrerequisites[nextLevel]; ok {
		if !state.ResearchedTechs[techName] {
			// Need to research tech first - can't upgrade farm yet
			return nil
		}
	}

	levelData := farmBuilding.GetLevelData(nextLevel)
	if levelData == nil {
		return nil
	}

	return &BuildingAction{
		BuildingType: models.Farm,
		FromLevel:    currentLevel,
		ToLevel:      nextLevel,
		Building:     farmBuilding,
		LevelData:    levelData,
	}
}

// createStorageUpgrade creates a storage upgrade action
func (s *Solver) createStorageUpgrade(state *State, rt models.ResourceType, requiredCap int) *BuildingAction {
	storageType := resourceToStorage(rt)
	building := s.Buildings[storageType]
	if building == nil {
		return nil
	}

	currentLevel := state.GetBuildingLevel(storageType)
	nextLevel := currentLevel + 1
	levelData := building.GetLevelData(nextLevel)
	if levelData == nil {
		return nil
	}

	return &BuildingAction{
		BuildingType: storageType,
		FromLevel:    currentLevel,
		ToLevel:      nextLevel,
		Building:     building,
		LevelData:    levelData,
	}
}

// createLibraryUpgrade creates a Library upgrade action
func (s *Solver) createLibraryUpgrade(state *State, requiredLevel int) *BuildingAction {
	building := s.Buildings[models.Library]
	if building == nil {
		return nil
	}

	currentLevel := state.GetBuildingLevel(models.Library)
	if currentLevel >= requiredLevel {
		return nil
	}

	nextLevel := currentLevel + 1
	levelData := building.GetLevelData(nextLevel)
	if levelData == nil {
		return nil
	}

	return &BuildingAction{
		BuildingType: models.Library,
		FromLevel:    currentLevel,
		ToLevel:      nextLevel,
		Building:     building,
		LevelData:    levelData,
	}
}

func resourceToStorage(rt models.ResourceType) models.BuildingType {
	switch rt {
	case models.Wood:
		return models.WoodStore
	case models.Stone:
		return models.StoneStore
	case models.Iron:
		return models.OreStore
	default:
		return models.WoodStore
	}
}

// executeBuilding starts a building upgrade
func (s *Solver) executeBuilding(state *State, action *BuildingAction, events *EventQueue) {
	costs := action.Costs()

	// Deduct resources
	state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
	state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
	state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))

	// Deduct food
	state.FoodUsed += costs.Food

	// NOTE: Building level is updated on COMPLETION, not start
	// This is important for correct Library level checks in research

	// Set queue busy
	state.BuildingQueueFreeAt = state.Now + action.Duration()
	state.PendingBuilding = action

	// Schedule completion
	events.Push(Event{
		Time:    state.BuildingQueueFreeAt,
		Type:    EventBuildingComplete,
		Payload: action,
	})
}

// executeResearch starts a research
func (s *Solver) executeResearch(state *State, action *ResearchAction, events *EventQueue) {
	costs := action.Costs()

	// Deduct resources
	state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
	state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
	state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))

	// Deduct food
	state.FoodUsed += costs.Food

	// Set queue busy
	state.ResearchQueueFreeAt = state.Now + action.Duration()
	state.PendingResearch = action

	// Schedule completion
	events.Push(Event{
		Time:    state.ResearchQueueFreeAt,
		Type:    EventResearchComplete,
		Payload: action,
	})
}

// executeTraining starts training a unit
func (s *Solver) executeTraining(state *State, action *TrainUnitAction, events *EventQueue) {
	costs := action.Costs()

	// Deduct resources
	state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
	state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
	state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))

	// Deduct food
	state.FoodUsed += action.FoodCost()

	// Set queue busy
	state.TrainingQueueFreeAt = state.Now + action.Duration()
	state.PendingTraining = action

	// Schedule completion
	events.Push(Event{
		Time:    state.TrainingQueueFreeAt,
		Type:    EventTrainingComplete,
		Payload: action,
	})
}

// startMission starts a tavern mission
func (s *Solver) startMission(state *State, mission *models.Mission, events *EventQueue) {
	costs := mission.ResourceCosts

	// Deduct resources
	state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
	state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
	state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))

	// Assign units
	state.Army.Subtract(mission.UnitsRequired)
	state.UnitsOnMission.AddFrom(mission.UnitsRequired)

	// Create mission state
	ms := &models.MissionState{
		Mission:       mission,
		StartTime:     state.Now,
		EndTime:       state.Now + mission.DurationMinutes*60,
		AssignedUnits: mission.UnitsRequired,
	}
	state.RunningMissions = append(state.RunningMissions, ms)

	// Schedule completion
	events.Push(Event{
		Time:    ms.EndTime,
		Type:    EventMissionComplete,
		Payload: ms,
	})
}

// scheduleResourceWakeup schedules a StateChanged event when resources become available
func (s *Solver) scheduleResourceWakeup(state *State, events *EventQueue) {
	// Find the earliest time when any pending action becomes affordable
	var wakeupTime int = -1

	// Check building queue
	if state.Now >= state.BuildingQueueFreeAt && state.PendingBuilding == nil {
		if action := s.pickBestBuildingActionIgnoringPrereqs(state); action != nil {
			// Resolve to actual action (may be prerequisite)
			action = s.resolvePrerequisites(state, action)
			if action != nil {
				waitTime := s.waitTimeForCosts(state, action.Costs())
				if waitTime > 0 {
					t := state.Now + waitTime
					if wakeupTime < 0 || t < wakeupTime {
						wakeupTime = t
					}
				}
			}
		}
	}

	// Check research queue
	if state.Now >= state.ResearchQueueFreeAt && state.PendingResearch == nil {
		if action := s.pickBestResearchAction(state); action != nil {
			waitTime := s.waitTimeForCosts(state, action.Costs())
			if waitTime > 0 {
				t := state.Now + waitTime
				if wakeupTime < 0 || t < wakeupTime {
					wakeupTime = t
				}
			}
		}
	}

	// Check training queue
	if state.Now >= state.TrainingQueueFreeAt && state.PendingTraining == nil {
		if action := s.pickBestTrainingAction(state); action != nil {
			waitTime := s.waitTimeForCosts(state, action.Costs())
			if waitTime > 0 {
				t := state.Now + waitTime
				if wakeupTime < 0 || t < wakeupTime {
					wakeupTime = t
				}
			}
		}
	}

	if wakeupTime > state.Now {
		events.PushIfNotExists(Event{Time: wakeupTime, Type: EventStateChanged})
	}
}

// Helper functions

func (s *Solver) removeMissionFromRunning(state *State, ms *models.MissionState) {
	for i, running := range state.RunningMissions {
		if running == ms {
			state.RunningMissions = append(state.RunningMissions[:i], state.RunningMissions[i+1:]...)
			return
		}
	}
}

func (s *Solver) canAfford(state *State, costs models.Costs) bool {
	if costs.Wood > 0 && state.GetResource(models.Wood) < float64(costs.Wood) {
		return false
	}
	if costs.Stone > 0 && state.GetResource(models.Stone) < float64(costs.Stone) {
		return false
	}
	if costs.Iron > 0 && state.GetResource(models.Iron) < float64(costs.Iron) {
		return false
	}
	return true
}

func (s *Solver) waitTimeForCosts(state *State, costs models.Costs) int {
	maxWait := 0

	checkResource := func(rt models.ResourceType, cost int) {
		if cost == 0 {
			return
		}
		available := state.GetResource(rt)
		if available >= float64(cost) {
			return
		}
		shortfall := float64(cost) - available
		rate := state.GetProductionRate(rt) * state.ProductionBonus
		if rate <= 0 {
			maxWait = -1
			return
		}
		secondsNeeded := int(shortfall/rate*3600) + 1
		if secondsNeeded > maxWait && maxWait >= 0 {
			maxWait = secondsNeeded
		}
	}

	checkResource(models.Wood, costs.Wood)
	if maxWait < 0 {
		return -1
	}
	checkResource(models.Stone, costs.Stone)
	if maxWait < 0 {
		return -1
	}
	checkResource(models.Iron, costs.Iron)

	return maxWait
}

func (s *Solver) advanceTime(state *State, seconds int) {
	if seconds <= 0 {
		return
	}

	hours := float64(seconds) / 3600.0
	state.Now += seconds

	// Accumulate resources
	for i := 0; i < 3; i++ {
		rate := state.ProductionRates[i]
		if rate <= 0 {
			continue
		}
		produced := rate * hours * state.ProductionBonus
		state.Resources[i] += produced

		// Cap at storage
		if state.StorageCaps[i] > 0 && state.Resources[i] > float64(state.StorageCaps[i]) {
			state.Resources[i] = float64(state.StorageCaps[i])
		}
	}
}

func (s *Solver) allTargetsReached(state *State) bool {
	for bt, target := range s.TargetLevels {
		if state.GetBuildingLevel(bt) < target {
			return false
		}
	}
	return true
}

func (s *Solver) initializeState(state *State) {
	// Calculate production rates
	for _, bt := range []models.BuildingType{models.Lumberjack, models.Quarry, models.OreMine} {
		building := s.Buildings[bt]
		if building == nil {
			continue
		}
		level := state.GetBuildingLevel(bt)
		levelData := building.GetLevelData(level)
		if levelData != nil && levelData.ProductionRate != nil {
			rt := buildingToResource(bt)
			state.SetProductionRate(rt, *levelData.ProductionRate)
		}
	}

	// Calculate storage caps
	for _, bt := range []models.BuildingType{models.WoodStore, models.StoneStore, models.OreStore} {
		building := s.Buildings[bt]
		if building == nil {
			continue
		}
		level := state.GetBuildingLevel(bt)
		levelData := building.GetLevelData(level)
		if levelData != nil && levelData.StorageCapacity != nil {
			rt := buildingToResource(bt)
			state.SetStorageCap(rt, *levelData.StorageCapacity)
		}
	}

	// Calculate food capacity
	farmBuilding := s.Buildings[models.Farm]
	if farmBuilding != nil {
		level := state.GetBuildingLevel(models.Farm)
		levelData := farmBuilding.GetLevelData(level)
		if levelData != nil && levelData.StorageCapacity != nil {
			state.FoodCapacity = *levelData.StorageCapacity
		}
	}
}

func (s *Solver) updateAfterBuild(state *State, ba *BuildingAction) {
	building := s.Buildings[ba.BuildingType]
	if building == nil {
		return
	}

	levelData := building.GetLevelData(ba.ToLevel)
	if levelData == nil {
		return
	}

	// Update production rate
	if levelData.ProductionRate != nil {
		rt := buildingToResource(ba.BuildingType)
		state.SetProductionRate(rt, *levelData.ProductionRate)
	}

	// Update storage cap
	if levelData.StorageCapacity != nil {
		switch ba.BuildingType {
		case models.WoodStore, models.StoneStore, models.OreStore:
			rt := buildingToResource(ba.BuildingType)
			state.SetStorageCap(rt, *levelData.StorageCapacity)
		case models.Farm:
			state.FoodCapacity = *levelData.StorageCapacity
		}
	}
}

func buildingToResource(bt models.BuildingType) models.ResourceType {
	switch bt {
	case models.Lumberjack, models.WoodStore:
		return models.Wood
	case models.Quarry, models.StoneStore:
		return models.Stone
	case models.OreMine, models.OreStore:
		return models.Iron
	default:
		return models.Wood
	}
}

func (s *Solver) buildingROI(state *State, action *BuildingAction) float64 {
	if action.LevelData.ProductionRate == nil {
		return 0
	}

	var currentRate float64
	if action.FromLevel > 0 {
		prevData := action.Building.GetLevelData(action.FromLevel)
		if prevData != nil && prevData.ProductionRate != nil {
			currentRate = *prevData.ProductionRate
		}
	}

	newRate := *action.LevelData.ProductionRate
	gain := newRate - currentRate
	buildHours := float64(action.LevelData.BuildTimeSeconds) / 3600.0

	if buildHours <= 0 {
		return gain * 1000
	}

	// Base ROI
	baseROI := gain / buildHours

	// Apply scarcity multiplier
	scarcityMultiplier := s.calculateScarcityMultiplier(state, action.BuildingType)

	return baseROI * scarcityMultiplier
}

// calculateScarcityMultiplier returns a multiplier based on how scarce the produced resource is
func (s *Solver) calculateScarcityMultiplier(state *State, bt models.BuildingType) float64 {
	// Get production rates
	woodRate := state.GetProductionRate(models.Wood)
	stoneRate := state.GetProductionRate(models.Stone)
	ironRate := state.GetProductionRate(models.Iron)

	// Avoid division by zero
	if woodRate < 0.1 {
		woodRate = 0.1
	}
	if stoneRate < 0.1 {
		stoneRate = 0.1
	}
	if ironRate < 0.1 {
		ironRate = 0.1
	}

	// Total production
	totalRate := woodRate + stoneRate + ironRate

	// Resource demand ratios (based on typical building costs)
	// Early game buildings need roughly: Wood 40%, Stone 40%, Iron 20%
	woodDemand := 0.40
	stoneDemand := 0.40
	ironDemand := 0.20

	// Current production ratios
	woodRatio := woodRate / totalRate
	stoneRatio := stoneRate / totalRate
	ironRatio := ironRate / totalRate

	// Scarcity = demand / supply ratio
	var scarcity float64
	switch bt {
	case models.Lumberjack:
		scarcity = woodDemand / woodRatio
	case models.Quarry:
		scarcity = stoneDemand / stoneRatio
	case models.OreMine:
		scarcity = ironDemand / ironRatio
	default:
		return 1.0
	}

	// Normalize: scarcity of 1.0 means balanced
	// Cap the multiplier to avoid extreme values
	if scarcity < 0.5 {
		scarcity = 0.5
	}
	if scarcity > 2.0 {
		scarcity = 2.0
	}

	return scarcity
}

// ProductionTechAction represents a production tech that can be researched
type ProductionTechAction struct {
	Technology           *models.Technology
	RequiredLibraryLevel int
}

// getBestProductionTechAction returns the best production tech to research
func (s *Solver) getBestProductionTechAction(state *State) *ProductionTechAction {
	libraryBuilding := s.Buildings[models.Library]
	if libraryBuilding == nil {
		return nil
	}

	var best *ProductionTechAction
	var bestROI float64

	for _, techName := range []string{"Beer tester", "Wheelbarrow"} {
		if state.ResearchedTechs[techName] {
			continue
		}
		tech := s.Technologies[techName]
		if tech == nil {
			continue
		}

		action := &ProductionTechAction{
			Technology:           tech,
			RequiredLibraryLevel: tech.RequiredLibraryLevel,
		}

		roi := s.productionTechROI(state, action)
		if roi > bestROI {
			bestROI = roi
			best = action
		}
	}

	return best
}

// productionTechROI calculates the ROI for a production tech
func (s *Solver) productionTechROI(state *State, action *ProductionTechAction) float64 {
	// Production bonus: 5% = 0.05 multiplier on all production
	bonusMultiplier := 0.05

	// Current total production rate
	totalRate := state.GetProductionRate(models.Wood) +
		state.GetProductionRate(models.Stone) +
		state.GetProductionRate(models.Iron)

	// Gain in production rate
	gain := totalRate * bonusMultiplier

	// Time to get the tech (research time + Library upgrades if needed)
	researchHours := float64(action.Technology.ResearchTimeSeconds) / 3600.0

	libraryLevel := state.GetBuildingLevel(models.Library)
	libraryUpgradeHours := 0.0
	if libraryLevel < action.RequiredLibraryLevel {
		// Calculate time needed for Library upgrades
		libraryBuilding := s.Buildings[models.Library]
		if libraryBuilding != nil {
			for level := libraryLevel + 1; level <= action.RequiredLibraryLevel; level++ {
				levelData := libraryBuilding.GetLevelData(level)
				if levelData != nil {
					libraryUpgradeHours += float64(levelData.BuildTimeSeconds) / 3600.0
				}
			}
		}
	}

	totalHours := researchHours + libraryUpgradeHours
	if totalHours <= 0 {
		return gain * 1000
	}

	return gain / totalHours
}

// pickBestResearchAction selects the best research to start
func (s *Solver) pickBestResearchAction(state *State) *ResearchAction {
	libraryLevel := state.GetBuildingLevel(models.Library)

	// Check for prerequisite techs first (reactive)
	for bt, target := range s.TargetLevels {
		building := s.Buildings[bt]
		if building == nil {
			continue
		}

		current := state.GetBuildingLevel(bt)
		for level := current + 1; level <= target; level++ {
			if techName, ok := building.TechnologyPrerequisites[level]; ok {
				if !state.ResearchedTechs[techName] {
					tech := s.Technologies[techName]
					if tech != nil && libraryLevel >= tech.RequiredLibraryLevel {
						return &ResearchAction{Technology: tech}
					}
				}
			}
		}
	}

	// Production techs
	for _, techName := range []string{"Beer tester", "Wheelbarrow"} {
		if state.ResearchedTechs[techName] {
			continue
		}
		tech := s.Technologies[techName]
		if tech != nil && libraryLevel >= tech.RequiredLibraryLevel {
			return &ResearchAction{Technology: tech}
		}
	}

	return nil
}

// pickBestTrainingAction selects the best unit to train for missions
func (s *Solver) pickBestTrainingAction(state *State) *TrainUnitAction {
	if len(s.Missions) == 0 {
		return nil
	}

	// Only train units for missions AFTER all building targets are reached
	// This prevents training from competing with buildings for resources/food
	if !s.allTargetsReached(state) {
		return nil
	}

	tavernLevel := state.GetBuildingLevel(models.Tavern)

	// Sort missions by ROI (best first)
	sortedMissions := make([]*models.Mission, len(s.Missions))
	copy(sortedMissions, s.Missions)
	sort.Slice(sortedMissions, func(i, j int) bool {
		return sortedMissions[i].NetAverageRewardPerHour() > sortedMissions[j].NetAverageRewardPerHour()
	})

	// Find the best mission we can train for
	for _, mission := range sortedMissions {
		if tavernLevel < mission.TavernLevel {
			continue
		}

		// Check each unit requirement
		for _, req := range mission.UnitsRequired {
			have := state.Army.Get(req.Type)
			onMission := state.UnitsOnMission.Get(req.Type)
			available := have - onMission

			if available < req.Count {
				// Need more of this unit type
				def := models.GetUnitDefinition(req.Type)
				if def == nil {
					continue
				}

				// Check tech requirement
				if def.RequiredTech != "" && !state.ResearchedTechs[def.RequiredTech] {
					continue // Skip - need to research tech first
				}

				// Check arsenal level
				arsenalLevel := state.GetBuildingLevel(models.Arsenal)
				if arsenalLevel < 1 {
					continue // Can't train without arsenal
				}

				return &TrainUnitAction{
					UnitType:   req.Type,
					Definition: def,
				}
			}
		}
	}

	return nil
}

// pickBestMissionToStart selects the best mission to start
func (s *Solver) pickBestMissionToStart(state *State) *models.Mission {
	// Only start missions AFTER all building targets are reached
	if !s.allTargetsReached(state) {
		return nil
	}

	var best *models.Mission
	var bestROI float64

	for _, mission := range s.Missions {
		// Check tavern level
		if state.GetBuildingLevel(models.Tavern) < mission.TavernLevel {
			continue
		}

		// Check unit availability
		if !state.Army.CanSatisfy(mission.UnitsRequired) {
			continue
		}

		// Calculate ROI
		roi := mission.NetAverageRewardPerHour()
		if roi > bestROI {
			bestROI = roi
			best = mission
		}
	}

	return best
}

// researchRemainingTechs researches ALL remaining technologies after buildings are done
func (s *Solver) researchRemainingTechs(state *State, researchActions *[]models.ResearchAction) {
	// Research all technologies in order of Library level requirement
	type techWithLevel struct {
		name  string
		level int
	}
	var techsToResearch []techWithLevel

	for name, tech := range s.Technologies {
		if state.ResearchedTechs[name] {
			continue
		}
		techsToResearch = append(techsToResearch, techWithLevel{name, tech.RequiredLibraryLevel})
	}

	// Sort by Library level requirement, then by name for determinism
	sort.Slice(techsToResearch, func(i, j int) bool {
		if techsToResearch[i].level != techsToResearch[j].level {
			return techsToResearch[i].level < techsToResearch[j].level
		}
		return techsToResearch[i].name < techsToResearch[j].name
	})

	libraryLevel := state.GetBuildingLevel(models.Library)

	for _, t := range techsToResearch {
		tech := s.Technologies[t.name]
		if tech == nil {
			continue
		}

		// Check Library level
		if libraryLevel < tech.RequiredLibraryLevel {
			continue // Can't research without Library
		}

		// Wait for research queue
		if state.Now < state.ResearchQueueFreeAt {
			s.advanceTime(state, state.ResearchQueueFreeAt-state.Now)
		}

		// Wait for resources
		costs := tech.Costs
		waitTime := s.waitTimeForCosts(state, costs)
		if waitTime > 0 {
			s.advanceTime(state, waitTime)
		} else if waitTime < 0 {
			continue
		}

		// Check food
		foodCost := costs.Food
		if state.FoodUsed+foodCost > state.FoodCapacity {
			continue
		}

		// Execute research
		startTime := state.Now

		// Deduct resources
		state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
		state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
		state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))
		state.FoodUsed += foodCost

		// Update queue
		state.ResearchQueueFreeAt = state.Now + tech.ResearchTimeSeconds
		endTime := state.ResearchQueueFreeAt

		// Mark as researched immediately (no more building decisions depend on this)
		state.ResearchedTechs[tech.Name] = true

		// Apply production bonus
		if tech.Name == "Beer tester" || tech.Name == "Wheelbarrow" {
			state.ProductionBonus += 0.05
		}

		*researchActions = append(*researchActions, models.ResearchAction{
			TechnologyName: tech.Name,
			StartTime:      startTime,
			EndTime:        endTime,
			Costs:          costs,
			FoodUsed:       state.FoodUsed,
			FoodCapacity:   state.FoodCapacity,
		})
	}
}
