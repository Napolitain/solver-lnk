package castle

import (
	"sort"

	"github.com/napolitain/solver-lnk/internal/models"
)

// Solver is the V4 event-driven solver
type Solver struct {
	Buildings    map[models.BuildingType]*models.Building
	Technologies map[string]*models.Technology
	Missions     []*models.Mission
	TargetLevels models.BuildingLevelMap
}

// MissionEvent tracks when a mission started and ended (for testing)
type MissionEvent struct {
	MissionName string
	StartTime   float64
	EndTime     float64
}

// NewSolver creates a new V4 solver
func NewSolver(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	missions []*models.Mission,
	targetLevels map[models.BuildingType]int,
) *Solver {
	// Convert map to struct for determinism
	var targets models.BuildingLevelMap
	for bt, level := range targetLevels {
		targets.Set(bt, level)
	}
	return &Solver{
		Buildings:    buildings,
		Technologies: technologies,
		Missions:     missions,
		TargetLevels: targets,
	}
}

// Solve runs the event-driven solver and returns a solution
func (s *Solver) Solve(initialState *models.GameState) *models.Solution {
	state := NewState(initialState)
	s.initializeState(state)

	events := NewEventQueue()

	var buildingActions []models.BuildingUpgradeAction
	var researchActions []models.ResearchAction
	var trainingActions []models.TrainUnitAction
	var missionActions []models.MissionAction

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

		s.processEvent(state, event, events, &buildingActions, &researchActions, &trainingActions, &missionActions)
	}

	// Phase 2: Continue simulation for research/training/missions after buildings complete
	// This handles all remaining events including mission completions
	// DO NOT drain events here - let runPostBuildingPhase handle them to avoid gaps
	s.runPostBuildingPhase(state, events, &buildingActions, &researchActions, &trainingActions, &missionActions)

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
		TrainingActions:  trainingActions,
		MissionActions:   missionActions,
		TotalTimeSeconds: finalTime,
		FinalState:       state.ToGameState(),
	}
}

// SolveWithMissionTracking is like Solve but also returns mission events for testing
func (s *Solver) SolveWithMissionTracking(initialState *models.GameState) (*models.Solution, []MissionEvent) {
	state := NewState(initialState)
	s.initializeState(state)

	events := NewEventQueue()
	var missionEvents []MissionEvent

	var buildingActions []models.BuildingUpgradeAction
	var researchActions []models.ResearchAction
	var trainingActions []models.TrainUnitAction
	var missionActions []models.MissionAction

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

		// Track mission starts from EventMissionComplete (previous mission ended)
		if event.Type == EventMissionComplete {
			// Record the mission that just completed
			ms := event.Payload.(*models.MissionState)
			_ = ms // Already tracked when started
		}

		// Process the event normally
		s.processEvent(state, event, events, &buildingActions, &researchActions, &trainingActions, &missionActions)

		// After processing StateChanged, check what missions were started
		if event.Type == EventStateChanged {
			for _, rm := range state.RunningMissions {
				// Check if this mission was just started (started at current time)
				if rm.StartTime == state.Now {
					missionEvents = append(missionEvents, MissionEvent{
						MissionName: rm.Mission.Name,
						StartTime:   float64(rm.StartTime),
						EndTime:     float64(rm.EndTime),
					})
				}
			}
		}
	}

	// Phase 2: Continue simulation for research/training/missions after buildings complete
	s.runPostBuildingPhase(state, events, &buildingActions, &researchActions, &trainingActions, &missionActions)

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
		TrainingActions:  trainingActions,
		MissionActions:   missionActions,
		TotalTimeSeconds: finalTime,
		FinalState:       state.ToGameState(),
	}, missionEvents
}

// processEvent handles a single event
func (s *Solver) processEvent(
	state *State,
	event Event,
	events *EventQueue,
	buildingActions *[]models.BuildingUpgradeAction,
	researchActions *[]models.ResearchAction,
	trainingActions *[]models.TrainUnitAction,
	missionActions *[]models.MissionAction,
) {
	switch event.Type {
	case EventMissionComplete:
		s.handleMissionComplete(state, event, events, missionActions)

	case EventBuildingComplete:
		s.handleBuildingComplete(state, event, events, buildingActions)

	case EventResearchComplete:
		s.handleResearchComplete(state, event, events, researchActions)

	case EventTrainingComplete:
		s.handleTrainingComplete(state, event, events, trainingActions)

	case EventStateChanged:
		s.handleStateChanged(state, events, buildingActions, researchActions)
	}
}

// handleMissionComplete processes a completed mission
func (s *Solver) handleMissionComplete(state *State, event Event, events *EventQueue, missionActions *[]models.MissionAction) {
	ms := event.Payload.(*models.MissionState)

	// Record mission action
	*missionActions = append(*missionActions, models.MissionAction{
		MissionName:  ms.Mission.Name,
		StartTime:    ms.StartTime,
		EndTime:      ms.EndTime,
		ResourceCost: ms.Mission.ResourceCosts,
		Rewards:      ms.Mission.Rewards,
	})

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
		state.ProductionBonus += ProductionTechBonus
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
func (s *Solver) handleTrainingComplete(state *State, event Event, events *EventQueue, trainingActions *[]models.TrainUnitAction) {
	ta := event.Payload.(*TrainUnitAction)

	// Record training action
	*trainingActions = append(*trainingActions, models.TrainUnitAction{
		UnitType:     ta.UnitType,
		Count:        1,
		StartTime:    state.Now - ta.Duration(),
		EndTime:      state.Now,
		Costs:        ta.Costs(),
		FoodUsed:     state.FoodUsed,
		FoodCapacity: state.FoodCapacity,
	})

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

// runPostBuildingPhase runs research, training, and missions after building targets are reached
// This uses event-driven simulation so missions run in parallel with training
func (s *Solver) runPostBuildingPhase(
	state *State,
	events *EventQueue,
	buildingActions *[]models.BuildingUpgradeAction,
	researchActions *[]models.ResearchAction,
	trainingActions *[]models.TrainUnitAction,
	missionActions *[]models.MissionAction,
) {
	// Bootstrap with initial event
	events.Push(Event{Time: state.Now, Type: EventStateChanged})

	// Hard cap to prevent infinite loops - this should be plenty for any reasonable scenario
	maxIterations := 500000
	iterations := 0

	// Track when all work is done (no more training/research/missions possible)
	for !events.Empty() && iterations < maxIterations {
		iterations++

		event := events.Pop()

		// Advance time (accumulate resources)
		if event.Time > state.Now {
			s.advanceTime(state, event.Time-state.Now)
		}

		switch event.Type {
		case EventStateChanged:
			// Check if all post-building work is done
			if s.isPostBuildingComplete(state) {
				// Drain remaining events without starting new work
				continue
			}
			s.handlePostBuildingStateChanged(state, events, researchActions, trainingActions)

		case EventBuildingComplete:
			// Handle any remaining building completions (but don't start new buildings)
			s.handleBuildingComplete(state, event, events, buildingActions)

		case EventResearchComplete:
			s.handleResearchComplete(state, event, events, researchActions)

		case EventTrainingComplete:
			s.handleTrainingComplete(state, event, events, trainingActions)

		case EventMissionComplete:
			ms := event.Payload.(*models.MissionState)

			// Record mission action
			*missionActions = append(*missionActions, models.MissionAction{
				MissionName:  ms.Mission.Name,
				StartTime:    ms.StartTime,
				EndTime:      ms.EndTime,
				ResourceCost: ms.Mission.ResourceCosts,
				Rewards:      ms.Mission.Rewards,
			})

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

			// Trigger re-evaluation only if not complete
			if !s.isPostBuildingComplete(state) {
				events.PushIfNotExists(Event{Time: state.Now, Type: EventStateChanged})
			}
		}
	}
}

// isPostBuildingComplete returns true when all post-building phase work is done
func (s *Solver) isPostBuildingComplete(state *State) bool {
	// Check if all research is done (or can't be done due to food)
	libraryLevel := state.GetBuildingLevel(models.Library)
	for _, techName := range models.AllTechNames() {
		name := string(techName)
		tech := s.Technologies[name]
		if tech == nil {
			continue
		}
		if state.ResearchedTechs[name] {
			continue
		}
		if tech.RequiredLibraryLevel > libraryLevel {
			continue // Can't research without higher Library
		}
		// Check if we can afford food for this research
		if state.FoodUsed+tech.Costs.Food <= state.FoodCapacity {
			return false // Still have research to do and can afford it
		}
		// Tech exists but we can't afford its food cost - skip it
	}

	// Check if all training is done (food capacity reached or can't train more)
	if state.FoodUsed < state.FoodCapacity {
		// Check if any unit can be trained
		for _, ut := range []models.UnitType{models.Spearman, models.Swordsman, models.Archer, models.Crossbowman, models.Horseman, models.Lancer} {
			def := models.GetUnitDefinition(ut)
			if def == nil {
				continue
			}
			if def.RequiredTech != "" && !state.ResearchedTechs[def.RequiredTech] {
				continue
			}
			if state.FoodUsed+def.FoodCost <= state.FoodCapacity {
				return false // Can still train units
			}
		}
	}

	return true // All done
}

// handlePostBuildingStateChanged handles state changes after building targets are reached
// Only processes research, training, and missions - not new buildings
func (s *Solver) handlePostBuildingStateChanged(
	state *State,
	events *EventQueue,
	researchActions *[]models.ResearchAction,
	trainingActions *[]models.TrainUnitAction,
) {
	// Research queue - research all remaining techs
	if state.Now >= state.ResearchQueueFreeAt && state.PendingResearch == nil {
		if action := s.pickNextRemainingResearch(state); action != nil {
			if s.canAfford(state, action.Costs()) && state.CanAffordFood(action.Costs().Food) {
				s.executeResearch(state, action, events)
			}
		}
	}

	// Training queue - train units for missions and defense
	if state.Now >= state.TrainingQueueFreeAt && state.PendingTraining == nil {
		if action := s.pickNextTrainingAction(state); action != nil {
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
	s.schedulePostBuildingWakeup(state, events)
}

// pickNextRemainingResearch picks the next technology to research
func (s *Solver) pickNextRemainingResearch(state *State) *ResearchAction {
	libraryLevel := state.GetBuildingLevel(models.Library)

	// First check known techs in deterministic order
	for _, techName := range models.AllTechNames() {
		name := string(techName)
		tech := s.Technologies[name]
		if tech == nil {
			continue
		}
		if state.ResearchedTechs[name] {
			continue
		}
		if tech.RequiredLibraryLevel > libraryLevel {
			continue
		}
		return &ResearchAction{
			Technology: tech,
		}
	}

	// Then check any additional techs not in AllTechNames (e.g., Fortress construction)
	// Sort by library level for determinism
	type techInfo struct {
		name  string
		tech  *models.Technology
		level int
	}
	var additionalTechs []techInfo
	for name, tech := range s.Technologies {
		// Skip if already researched
		if state.ResearchedTechs[name] {
			continue
		}
		// Skip if library level too low
		if tech.RequiredLibraryLevel > libraryLevel {
			continue
		}
		// Check if this tech is in AllTechNames
		found := false
		for _, knownTech := range models.AllTechNames() {
			if string(knownTech) == name {
				found = true
				break
			}
		}
		if !found {
			additionalTechs = append(additionalTechs, techInfo{name, tech, tech.RequiredLibraryLevel})
		}
	}

	// Sort by library level (lowest first) for determinism
	sort.Slice(additionalTechs, func(i, j int) bool {
		if additionalTechs[i].level != additionalTechs[j].level {
			return additionalTechs[i].level < additionalTechs[j].level
		}
		return additionalTechs[i].name < additionalTechs[j].name
	})

	if len(additionalTechs) > 0 {
		return &ResearchAction{Technology: additionalTechs[0].tech}
	}

	return nil
}

// pickNextTrainingAction picks the next unit to train for missions or defense
func (s *Solver) pickNextTrainingAction(state *State) *TrainUnitAction {
	// First, check if we need units for missions
	for _, mission := range s.Missions {
		tavernLevel := state.GetBuildingLevel(models.Tavern)
		if tavernLevel < mission.TavernLevel {
			continue
		}
		if mission.MaxTavernLevel > 0 && tavernLevel > mission.MaxTavernLevel {
			continue
		}

		for _, req := range mission.UnitsRequired {
			have := state.Army.Get(req.Type)
			onMission := state.UnitsOnMission.Get(req.Type)
			available := have - onMission
			if available < req.Count {
				// Need more of this unit type
				def := models.GetUnitDefinition(req.Type)
				if def != nil && (def.RequiredTech == "" || state.ResearchedTechs[def.RequiredTech]) {
					if state.FoodUsed+def.FoodCost <= state.FoodCapacity {
						return &TrainUnitAction{
							UnitType:   req.Type,
							Definition: def,
						}
					}
				}
			}
		}
	}

	// Then, train defense units if food available
	if state.FoodUsed < state.FoodCapacity {
		return s.pickDefenseUnit(state)
	}

	return nil
}

// pickDefenseUnit picks the best unit for balanced defense
func (s *Solver) pickDefenseUnit(state *State) *TrainUnitAction {
	combatUnits := []models.UnitType{
		models.Spearman,
		models.Swordsman,
		models.Archer,
		models.Crossbowman,
		models.Horseman,
		models.Lancer,
	}

	// Track current defense totals
	defCav, defInf, defArt := 0, 0, 0
	for _, ut := range combatUnits {
		count := state.Army.Get(ut)
		def := models.GetUnitDefinition(ut)
		if def != nil {
			defCav += count * def.DefenseVsCavalry
			defInf += count * def.DefenseVsInfantry
			defArt += count * def.DefenseVsArtillery
		}
	}

	// Find which defense type is weakest
	minDef := defCav
	if defInf < minDef {
		minDef = defInf
	}
	if defArt < minDef {
		minDef = defArt
	}

	// Find best unit to improve the weakest defense
	var bestUnit models.UnitType
	var bestDef *models.UnitDefinition
	var bestImprovement int
	found := false

	for _, ut := range combatUnits {
		def := models.GetUnitDefinition(ut)
		if def == nil {
			continue
		}

		// Check tech requirement
		if def.RequiredTech != "" && !state.ResearchedTechs[def.RequiredTech] {
			continue
		}

		// Check food
		if state.FoodUsed+def.FoodCost > state.FoodCapacity {
			continue
		}

		// Calculate improvement to minimum defense
		newCav := defCav + def.DefenseVsCavalry
		newInf := defInf + def.DefenseVsInfantry
		newArt := defArt + def.DefenseVsArtillery
		newMin := newCav
		if newInf < newMin {
			newMin = newInf
		}
		if newArt < newMin {
			newMin = newArt
		}
		improvement := newMin - minDef

		if !found || improvement > bestImprovement {
			bestImprovement = improvement
			bestUnit = ut
			bestDef = def
			found = true
		}
	}

	if !found {
		return nil
	}

	return &TrainUnitAction{
		UnitType:   bestUnit,
		Definition: bestDef,
	}
}

// schedulePostBuildingWakeup schedules wakeup for research/training/missions only
func (s *Solver) schedulePostBuildingWakeup(state *State, events *EventQueue) {
	var wakeupTime int = -1

	// Check if waiting for research to afford
	if state.Now >= state.ResearchQueueFreeAt && state.PendingResearch == nil {
		if action := s.pickNextRemainingResearch(state); action != nil {
			waitTime := s.waitTimeForCosts(state, action.Costs())
			if waitTime > 0 {
				t := state.Now + waitTime
				if wakeupTime < 0 || t < wakeupTime {
					wakeupTime = t
				}
			}
		}
	}

	// Check if waiting for training to afford
	if state.Now >= state.TrainingQueueFreeAt && state.PendingTraining == nil {
		if action := s.pickNextTrainingAction(state); action != nil {
			waitTime := s.waitTimeForCosts(state, action.Costs())
			if waitTime > 0 {
				t := state.Now + waitTime
				if wakeupTime < 0 || t < wakeupTime {
					wakeupTime = t
				}
			}
		}
	}

	// Check if waiting for mission to afford
	mission := s.pickBestMissionToStart(state)
	if mission != nil {
		waitTime := s.waitTimeForCosts(state, mission.ResourceCosts)
		if waitTime > 0 {
			t := state.Now + waitTime
			if wakeupTime < 0 || t < wakeupTime {
				wakeupTime = t
			}
		}
	}

	if wakeupTime > state.Now {
		events.PushIfNotExists(Event{Time: wakeupTime, Type: EventStateChanged})
	}
}

// tryStartBuilding attempts to start a building, handling prerequisites
func (s *Solver) tryStartBuilding(state *State, events *EventQueue) {
	// Get all building actions sorted by ROI
	candidates := s.getAllBuildingActionsSortedByROI(state)
	if len(candidates) == 0 {
		return
	}

	// Check if production tech (Beer Tester, Wheelbarrow) has better ROI than best building
	prodTechAction := s.getBestProductionTechAction(state)
	if prodTechAction != nil && len(candidates) > 0 {
		buildingROI := s.buildingROI(state, candidates[0])
		techROI := s.productionTechROI(state, prodTechAction)
		if techROI > buildingROI {
			// Production tech is better - check if we need Library upgrade
			libraryLevel := state.GetBuildingLevel(models.Library)
			if libraryLevel < prodTechAction.RequiredLibraryLevel {
				// Need to upgrade Library first - insert at front of candidates
				libAction := s.createLibraryUpgrade(state, prodTechAction.RequiredLibraryLevel)
				if libAction != nil {
					candidates = append([]*BuildingAction{libAction}, candidates...)
				}
			}
		}
	}

	// Try each candidate in order until we find one we can afford
	for _, action := range candidates {
		// Check prerequisites and get the actual action to execute
		resolved := s.resolvePrerequisites(state, action)
		if resolved == nil {
			continue
		}

		// Check if we can afford it
		costs := resolved.Costs()
		if !s.canAfford(state, costs) {
			continue
		}

		// Check food
		if !state.CanAffordFood(costs.Food) {
			continue
		}

		// Execute the action
		s.executeBuilding(state, resolved, events)
		return
	}
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
		candidates := s.getAllBuildingActionsSortedByROI(state)
		if len(candidates) > 0 {
			// Check first affordable candidate's wait time
			for _, action := range candidates {
				// Resolve to actual action (may be prerequisite)
				resolved := s.resolvePrerequisites(state, action)
				if resolved != nil {
					waitTime := s.waitTimeForCosts(state, resolved.Costs())
					if waitTime > 0 {
						t := state.Now + waitTime
						if wakeupTime < 0 || t < wakeupTime {
							wakeupTime = t
						}
					}
					break // Only need the first resolvable one
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
	allReached := true
	s.TargetLevels.Each(func(bt models.BuildingType, target int) {
		if target > 0 && state.GetBuildingLevel(bt) < target {
			allReached = false
		}
	})
	return allReached
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

// calculateDynamicScarcity calculates scarcity based on remaining build costs
// This replaces the hardcoded 40/40/20 ratios with actual demand from remaining buildings
func (s *Solver) calculateDynamicScarcity(state *State, bt models.BuildingType) float64 {
	// Calculate remaining costs for all target buildings
	var remainingWood, remainingStone, remainingIron float64

	s.TargetLevels.Each(func(targetBT models.BuildingType, targetLevel int) {
		if targetLevel == 0 {
			return
		}
		building := s.Buildings[targetBT]
		if building == nil {
			return
		}

		currentLevel := state.GetBuildingLevel(targetBT)
		for level := currentLevel + 1; level <= targetLevel; level++ {
			levelData := building.GetLevelData(level)
			if levelData != nil {
				remainingWood += float64(levelData.Costs.Wood)
				remainingStone += float64(levelData.Costs.Stone)
				remainingIron += float64(levelData.Costs.Iron)
			}
		}
	})

	totalRemaining := remainingWood + remainingStone + remainingIron
	if totalRemaining <= 0 {
		return 1.0 // No remaining costs, no scarcity adjustment
	}

	// Calculate demand ratios from remaining costs
	woodDemand := remainingWood / totalRemaining
	stoneDemand := remainingStone / totalRemaining
	ironDemand := remainingIron / totalRemaining

	// Get current production rates
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

	totalRate := woodRate + stoneRate + ironRate

	// Current production ratios (supply)
	woodSupply := woodRate / totalRate
	stoneSupply := stoneRate / totalRate
	ironSupply := ironRate / totalRate

	// Scarcity = demand / supply
	var scarcity float64
	switch bt {
	case models.Lumberjack:
		scarcity = woodDemand / woodSupply
	case models.Quarry:
		scarcity = stoneDemand / stoneSupply
	case models.OreMine:
		scarcity = ironDemand / ironSupply
	default:
		return 1.0
	}

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

// pickBestResearchAction selects the best research to start
func (s *Solver) pickBestResearchAction(state *State) *ResearchAction {
	libraryLevel := state.GetBuildingLevel(models.Library)

	// Check for prerequisite techs first (reactive) - only for the NEXT level upgrade
	// Don't schedule research for future levels until we're about to need them
	var prereqAction *ResearchAction
	s.TargetLevels.Each(func(bt models.BuildingType, target int) {
		if target == 0 || prereqAction != nil {
			return
		}
		building := s.Buildings[bt]
		if building == nil {
			return
		}

		current := state.GetBuildingLevel(bt)
		nextLevel := current + 1
		if nextLevel > target {
			return
		}
		// Only check the immediate next level for tech prerequisites
		if techName, ok := building.TechnologyPrerequisites[nextLevel]; ok {
			if !state.ResearchedTechs[techName] {
				tech := s.Technologies[techName]
				if tech != nil && libraryLevel >= tech.RequiredLibraryLevel {
					prereqAction = &ResearchAction{Technology: tech}
					return
				}
			}
		}
	})
	if prereqAction != nil {
		return prereqAction
	}

	// Unit techs - research techs needed for mission units
	unitTechs := s.getUnitTechsNeededForMissions(state)
	for _, techName := range unitTechs {
		if state.ResearchedTechs[techName] {
			continue
		}
		tech := s.Technologies[techName]
		if tech != nil && libraryLevel >= tech.RequiredLibraryLevel {
			return &ResearchAction{Technology: tech}
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

// getUnitTechsNeededForMissions returns techs needed for units required by missions
func (s *Solver) getUnitTechsNeededForMissions(state *State) []string {
	targetTavernLevel := s.TargetLevels.Get(models.Tavern)
	if targetTavernLevel == 0 {
		targetTavernLevel = state.GetBuildingLevel(models.Tavern)
	}

	// Get unit needs at target tavern level
	unitNeeds := s.calculateMissionUnitNeeds(targetTavernLevel)

	// Collect required techs in a deterministic order
	// Sort by library level required (lower first to unlock earlier)
	techsNeeded := make(map[string]bool)
	type techInfo struct {
		name         string
		libraryLevel int
	}
	var techList []techInfo

	// Check each unit type we need
	for unitType := range unitNeeds {
		def := models.GetUnitDefinition(unitType)
		if def != nil && def.RequiredTech != "" {
			if !techsNeeded[def.RequiredTech] {
				techsNeeded[def.RequiredTech] = true
				tech := s.Technologies[def.RequiredTech]
				libLevel := 0
				if tech != nil {
					libLevel = tech.RequiredLibraryLevel
				}
				techList = append(techList, techInfo{def.RequiredTech, libLevel})
			}
		}
	}

	// Sort by library level required
	sort.Slice(techList, func(i, j int) bool {
		return techList[i].libraryLevel < techList[j].libraryLevel
	})

	// Extract tech names in sorted order
	var techOrder []string
	for _, t := range techList {
		techOrder = append(techOrder, t.name)
	}

	return techOrder
}

// pickBestTrainingAction selects the best unit to train for missions
func (s *Solver) pickBestTrainingAction(state *State) *TrainUnitAction {
	if len(s.Missions) == 0 {
		return nil
	}

	// Check if we have food headroom for training
	// Units cost 1-2 food each; only train if we have spare capacity
	foodHeadroom := state.FoodCapacity - state.FoodUsed
	if foodHeadroom < MinFoodHeadroomForTraining {
		return nil // Not enough food buffer for training
	}

	currentTavernLevel := state.GetBuildingLevel(models.Tavern)
	targetTavernLevel := s.TargetLevels.Get(models.Tavern)
	if targetTavernLevel == 0 {
		targetTavernLevel = currentTavernLevel
	}

	arsenalLevel := state.GetBuildingLevel(models.Arsenal)
	if arsenalLevel < 1 {
		return nil // Can't train without arsenal
	}

	// Calculate maximum units needed for ALL missions at target tavern level
	// This ensures we train units proactively before tavern upgrades
	unitNeeds := s.calculateMissionUnitNeeds(targetTavernLevel)

	// Find which unit type we need most (compared to what we have)
	type unitDeficit struct {
		unitType models.UnitType
		deficit  int
		def      *models.UnitDefinition
	}

	var deficits []unitDeficit
	for unitType, needed := range unitNeeds {
		have := state.Army.Get(unitType)
		onMission := state.UnitsOnMission.Get(unitType)
		available := have - onMission

		if available < needed {
			def := models.GetUnitDefinition(unitType)
			if def == nil {
				continue
			}
			deficits = append(deficits, unitDeficit{
				unitType: unitType,
				deficit:  needed - available,
				def:      def,
			})
		}
	}

	if len(deficits) == 0 {
		return nil // Have enough units for all missions
	}

	// Sort by deficit (train most needed first), then by training time (fast units first)
	sort.Slice(deficits, func(i, j int) bool {
		if deficits[i].deficit != deficits[j].deficit {
			return deficits[i].deficit > deficits[j].deficit
		}
		return deficits[i].def.TrainingTimeSeconds < deficits[j].def.TrainingTimeSeconds
	})

	// Try each unit type in priority order
	for _, d := range deficits {
		// Check tech requirement
		if d.def.RequiredTech != "" && !state.ResearchedTechs[d.def.RequiredTech] {
			continue // Skip - need to research tech first (will be handled by tech queue)
		}

		// Check food cost for this unit
		if state.FoodUsed+d.def.FoodCost > state.FoodCapacity {
			continue // Would exceed food capacity
		}

		return &TrainUnitAction{
			UnitType:   d.unitType,
			Definition: d.def,
		}
	}

	return nil
}

// calculateMissionUnitNeeds returns the maximum units needed for all available missions
// at the given tavern level. This calculates the cumulative max across all missions.
func (s *Solver) calculateMissionUnitNeeds(tavernLevel int) map[models.UnitType]int {
	needs := make(map[models.UnitType]int)

	for _, mission := range s.Missions {
		// Skip missions not available at this tavern level
		if mission.TavernLevel > tavernLevel {
			continue
		}
		if mission.MaxTavernLevel > 0 && tavernLevel > mission.MaxTavernLevel {
			continue
		}

		// Track max needed for each unit type (missions can run in parallel)
		for _, req := range mission.UnitsRequired {
			if req.Count > needs[req.Type] {
				needs[req.Type] = req.Count
			}
		}
	}

	return needs
}

// pickBestMissionToStart selects the best mission to start
func (s *Solver) pickBestMissionToStart(state *State) *models.Mission {
	// Allow missions during building if we have units available
	// Missions provide resources which help building progress

	var best *models.Mission
	var bestROI float64

	tavernLevel := state.GetBuildingLevel(models.Tavern)

	for _, mission := range s.Missions {
		// Check tavern level (min)
		if tavernLevel < mission.TavernLevel {
			continue
		}

		// Check tavern level (max) - missions become unavailable at higher levels
		if mission.MaxTavernLevel > 0 && tavernLevel > mission.MaxTavernLevel {
			continue
		}

		// Check if this mission is already running (missions are unique - can't run same mission twice)
		alreadyRunning := false
		for _, running := range state.RunningMissions {
			if running.Mission.Name == mission.Name {
				alreadyRunning = true
				break
			}
		}
		if alreadyRunning {
			continue
		}

		// Check unit availability (subtract units on mission)
		canRun := true
		for _, req := range mission.UnitsRequired {
			have := state.Army.Get(req.Type)
			onMission := state.UnitsOnMission.Get(req.Type)
			available := have - onMission
			if available < req.Count {
				canRun = false
				break
			}
		}
		if !canRun {
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
