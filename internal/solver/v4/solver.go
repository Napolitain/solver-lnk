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
		if action := s.pickBestBuildingAction(state); action != nil {
			if s.canAfford(state, action.Costs()) && state.CanAffordFood(action.Costs().Food) {
				s.executeBuilding(state, action, events)
			}
		}
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

// executeBuilding starts a building upgrade
func (s *Solver) executeBuilding(state *State, action *BuildingAction, events *EventQueue) {
	costs := action.Costs()

	// Deduct resources
	state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
	state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
	state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))

	// Deduct food
	state.FoodUsed += costs.Food

	// Update building level immediately (workers assigned)
	state.SetBuildingLevel(action.BuildingType, action.ToLevel)

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
		if action := s.pickBestBuildingAction(state); action != nil {
			waitTime := s.waitTimeForCosts(state, action.Costs())
			if waitTime > 0 {
				t := state.Now + waitTime
				if wakeupTime < 0 || t < wakeupTime {
					wakeupTime = t
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

// pickBestBuildingAction selects the best building to upgrade
func (s *Solver) pickBestBuildingAction(state *State) *BuildingAction {
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

		// Check prerequisites
		if !s.checkBuildingPrerequisites(state, building, toLevel) {
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

func (s *Solver) checkBuildingPrerequisites(state *State, building *models.Building, level int) bool {
	// Check technology prerequisites
	if techName, ok := building.TechnologyPrerequisites[level]; ok {
		if !state.ResearchedTechs[techName] {
			return false
		}
	}

	// Check building prerequisites
	if prereqs, ok := building.Prerequisites[level]; ok {
		for reqBuilding, reqLevel := range prereqs {
			if state.GetBuildingLevel(reqBuilding) < reqLevel {
				return false
			}
		}
	}

	return true
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

	return gain / buildHours
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

// pickBestTrainingAction selects the best unit to train
func (s *Solver) pickBestTrainingAction(state *State) *TrainUnitAction {
	// For now, only train units if we need them for missions
	// TODO: Implement mission-based unit training

	// Check if any mission needs more units
	for _, mission := range s.Missions {
		if state.GetBuildingLevel(models.Tavern) < mission.TavernLevel {
			continue
		}

		for _, req := range mission.UnitsRequired {
			have := state.Army.Get(req.Type)
			if have < req.Count {
				// Need more of this unit type
				def := models.GetUnitDefinition(req.Type)
				if def == nil {
					continue
				}

				// Check tech requirement
				if def.RequiredTech != "" && !state.ResearchedTechs[def.RequiredTech] {
					continue
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
