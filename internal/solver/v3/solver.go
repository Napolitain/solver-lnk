package v3

import (
	"sort"

	"github.com/napolitain/solver-lnk/internal/models"
)

// Solver is the V3 ROI-based solver
type Solver struct {
	Buildings    map[models.BuildingType]*models.Building
	Technologies map[string]*models.Technology
	TargetLevels map[models.BuildingType]int
}

// NewSolver creates a new V3 solver
func NewSolver(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	targetLevels map[models.BuildingType]int,
) *Solver {
	return &Solver{
		Buildings:    buildings,
		Technologies: technologies,
		TargetLevels: targetLevels,
	}
}

// Solve runs the solver and returns a solution
func (s *Solver) Solve(initialState *models.GameState) *models.Solution {
	state := NewState(initialState)
	s.initializeState(state)

	var buildingActions []models.BuildingUpgradeAction
	var researchActions []models.ResearchAction

	// Track pending production updates (building type -> new level)
	// These get applied when the building completes
	var pendingBuildingUpdate *BuildingAction

	maxIterations := 100000 // Safety limit
	iterations := 0

	for !s.allTargetsReached(state) && iterations < maxIterations {
		iterations++

		// 1. Wait for building queue if busy - and apply pending production update
		if state.Now < state.BuildingQueueFreeAt {
			s.advanceTime(state, state.BuildingQueueFreeAt-state.Now)
			// Now the previous building has completed - update production
			if pendingBuildingUpdate != nil {
				s.updateAfterBuild(state, pendingBuildingUpdate)
				pendingBuildingUpdate = nil
			}
		}

		// 2. Get next building action to execute
		action := s.getNextAction(state)
		if action == nil {
			break
		}

		// 3. Check prerequisites (food, storage, tech)
		prereqAction := s.checkPrerequisites(state, action)
		if prereqAction != nil {
			action = prereqAction
		}

		// 3b. Check if we need to research a technology first
		researchAction := s.checkResearchPrerequisite(state, action)
		if researchAction != nil {
			// This building needs a tech that's not yet researched
			// Start the research if not already in progress
			ra := s.executeResearch(state, researchAction, &researchActions)
			if ra == nil {
				// Couldn't start research (e.g., Library level too low)
				// checkPrerequisites should have handled Library upgrade
				continue
			}
			// Research is now in progress or just completed
			// We must wait for it to complete before starting this building
			if state.ResearchQueueFreeAt > state.Now {
				s.advanceTime(state, state.ResearchQueueFreeAt-state.Now)
			}
			// Now the tech should be researched, continue with the building
		}

		// 4. Wait for resources if needed
		waitTime := s.waitTimeForAction(state, action)
		if waitTime > 0 {
			s.advanceTime(state, waitTime)
		} else if waitTime < 0 {
			// Cannot afford and no production - stuck
			break
		}

		// 5. Execute action
		startTime := state.Now

		// Double-check we can afford it now
		if !s.canAfford(state, action.Costs()) {
			// Still can't afford, wait more
			s.advanceTime(state, 60)
			continue
		}

		// Check food
		foodCost := action.Costs()[models.Food]
		if state.FoodUsed+foodCost > state.FoodCapacity {
			// Need Farm - this should have been caught by prereq check
			continue
		}

		action.Execute(state)

		// Schedule production/storage update for when building completes
		pendingBuildingUpdate = action

		endTime := startTime + action.Duration()
		buildingActions = append(buildingActions, models.BuildingUpgradeAction{
			BuildingType: action.BuildingType,
			FromLevel:    action.FromLevel,
			ToLevel:      action.ToLevel,
			StartTime:    startTime,
			EndTime:      endTime,
			Costs:        action.Costs(),
			FoodUsed:     state.FoodUsed,
			FoodCapacity: state.FoodCapacity,
		})
	}

	// Apply any pending building update before research
	if pendingBuildingUpdate != nil {
		if state.Now < state.BuildingQueueFreeAt {
			s.advanceTime(state, state.BuildingQueueFreeAt-state.Now)
		}
		s.updateAfterBuild(state, pendingBuildingUpdate)
	}

	// Research remaining technologies (production techs)
	s.researchRemainingTechs(state, &researchActions)

	// Final time is when both queues are free
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

// checkResearchPrerequisite checks if a building upgrade requires research
func (s *Solver) checkResearchPrerequisite(state *State, action *BuildingAction) *ResearchAction {
	building := s.Buildings[action.BuildingType]
	if building == nil {
		return nil
	}

	techName, ok := building.TechnologyPrerequisites[action.ToLevel]
	if !ok {
		return nil
	}

	if state.ResearchedTechs[techName] {
		return nil
	}

	tech := s.Technologies[techName]
	if tech == nil {
		return nil
	}

	return &ResearchAction{Technology: tech}
}

// executeResearch executes a research action
func (s *Solver) executeResearch(state *State, ra *ResearchAction, researchActions *[]models.ResearchAction) *models.ResearchAction {
	// Check Library level
	libraryLevel := state.GetBuildingLevel(models.Library)
	if libraryLevel < ra.Technology.RequiredLibraryLevel {
		// Library upgrade needed - will be handled by checkPrerequisites
		return nil
	}

	// Wait for research queue
	if state.Now < state.ResearchQueueFreeAt {
		// Research queue busy, but we can continue with building
		// Just wait until it's free
		waitTime := state.ResearchQueueFreeAt - state.Now
		s.advanceTime(state, waitTime)
	}

	// Wait for resources
	waitTime := s.waitTimeForAction(state, ra)
	if waitTime > 0 {
		s.advanceTime(state, waitTime)
	} else if waitTime < 0 {
		return nil // Cannot afford
	}

	// Check food
	foodCost := ra.Costs()[models.Food]
	if state.FoodUsed+foodCost > state.FoodCapacity {
		return nil // Need more food capacity
	}

	// Execute research
	startTime := state.Now
	ra.Execute(state)
	endTime := startTime + ra.Duration()

	result := models.ResearchAction{
		TechnologyName: ra.Technology.Name,
		StartTime:      startTime,
		EndTime:        endTime,
		Costs:          ra.Costs(),
		FoodUsed:       state.FoodUsed,
		FoodCapacity:   state.FoodCapacity,
	}
	*researchActions = append(*researchActions, result)

	return &result
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

		ra := &ResearchAction{Technology: tech}

		// Wait for research queue
		if state.Now < state.ResearchQueueFreeAt {
			s.advanceTime(state, state.ResearchQueueFreeAt-state.Now)
		}

		// Wait for resources
		waitTime := s.waitTimeForAction(state, ra)
		if waitTime > 0 {
			s.advanceTime(state, waitTime)
		} else if waitTime < 0 {
			continue
		}

		// Check food
		foodCost := ra.Costs()[models.Food]
		if state.FoodUsed+foodCost > state.FoodCapacity {
			continue
		}

		// Execute
		startTime := state.Now
		ra.Execute(state)
		endTime := startTime + ra.Duration()

		*researchActions = append(*researchActions, models.ResearchAction{
			TechnologyName: tech.Name,
			StartTime:      startTime,
			EndTime:        endTime,
			Costs:          ra.Costs(),
			FoodUsed:       state.FoodUsed,
			FoodCapacity:   state.FoodCapacity,
		})
	}
}

// initializeState sets up initial production rates, storage caps, and food capacity
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

// getNextAction returns the next building action based on ROI
func (s *Solver) getNextAction(state *State) *BuildingAction {
	// Collect all possible building actions
	var candidates []*BuildingAction

	// Add all target buildings that aren't at target level yet
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

	// Sort by ROI (descending), then by building type name for determinism
	sort.Slice(candidates, func(i, j int) bool {
		roiI := candidates[i].ROI(state)
		roiJ := candidates[j].ROI(state)
		if roiI != roiJ {
			return roiI > roiJ
		}
		// Tie-breaker: alphabetical by building type
		return candidates[i].BuildingType < candidates[j].BuildingType
	})

	// Return highest ROI action
	return candidates[0]
}

// checkPrerequisites checks if action has prerequisites and returns a prereq action if needed
func (s *Solver) checkPrerequisites(state *State, action *BuildingAction) *BuildingAction {
	costs := action.Costs()

	// Check food capacity
	foodCost := costs[models.Food]
	if state.FoodUsed+foodCost > state.FoodCapacity {
		farmAction := s.createFarmUpgrade(state, state.FoodUsed+foodCost)
		if farmAction != nil {
			return farmAction.(*BuildingAction)
		}
	}

	// Check storage capacity
	for _, rt := range []models.ResourceType{models.Wood, models.Stone, models.Iron} {
		cost := costs[rt]
		if cost == 0 {
			continue
		}
		cap := state.GetStorageCap(rt)
		if cost > cap {
			storageAction := s.createStorageUpgrade(state, rt, cost)
			if storageAction != nil {
				return storageAction.(*BuildingAction)
			}
		}
	}

	// Check tech prerequisite
	building := s.Buildings[action.BuildingType]
	if building != nil {
		if techName, ok := building.TechnologyPrerequisites[action.ToLevel]; ok {
			if !state.ResearchedTechs[techName] {
				tech := s.Technologies[techName]
				if tech != nil {
					libraryLevel := state.GetBuildingLevel(models.Library)
					if libraryLevel < tech.RequiredLibraryLevel {
						return s.createLibraryUpgrade(state, tech.RequiredLibraryLevel).(*BuildingAction)
					}
				}
			}
		}
	}

	return nil
}

// updateAfterBuild updates state after a building is completed
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

// allTargetsReached returns true if all target levels have been reached
func (s *Solver) allTargetsReached(state *State) bool {
	for bt, target := range s.TargetLevels {
		if state.GetBuildingLevel(bt) < target {
			return false
		}
	}
	return true
}

// nextDecisionPoint returns the earliest time when a decision can be made
// nolint:unused // Reserved for future multi-queue extension
func (s *Solver) nextDecisionPoint(state *State) int {
	// At minimum, we can make a decision now
	next := state.Now

	// If building queue is busy, next point is when it's free
	if state.BuildingQueueFreeAt > next {
		next = state.BuildingQueueFreeAt
	}

	return next
}

// advanceTime advances the simulation by the given number of seconds
func (s *Solver) advanceTime(state *State, seconds int) {
	if seconds <= 0 {
		return
	}

	hours := float64(seconds) / 3600.0
	state.Now += seconds

	// Accumulate resources
	for i := 0; i < 3; i++ { // Wood, Stone, Iron only
		rate := state.ProductionRates[i]
		if rate <= 0 {
			continue
		}
		produced := rate * hours * state.ProductionBonus
		state.Resources[i] += produced

		// Cap at storage
		cap := state.StorageCaps[i]
		if cap > 0 && state.Resources[i] > float64(cap) {
			state.Resources[i] = float64(cap)
		}
	}
}

// getPossibleActions returns all actions that could potentially be executed
// nolint:unused // Reserved for future multi-queue extension
func (s *Solver) getPossibleActions(state *State) []Action {
	var actions []Action

	// Add all target buildings that aren't at target level yet
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

		actions = append(actions, &BuildingAction{
			BuildingType: bt,
			FromLevel:    current,
			ToLevel:      toLevel,
			Building:     building,
			LevelData:    levelData,
		})
	}

	// Research actions
	actions = append(actions, s.getResearchActions(state)...)

	return actions
}

// createFarmUpgrade creates a Farm upgrade action to reach required food capacity
func (s *Solver) createFarmUpgrade(state *State, requiredFood int) Action {
	farmBuilding := s.Buildings[models.Farm]
	if farmBuilding == nil {
		return nil
	}

	currentLevel := state.GetBuildingLevel(models.Farm)
	for level := currentLevel + 1; level <= 30; level++ {
		levelData := farmBuilding.GetLevelData(level)
		if levelData == nil || levelData.StorageCapacity == nil {
			continue
		}
		if *levelData.StorageCapacity >= requiredFood {
			return &BuildingAction{
				BuildingType: models.Farm,
				FromLevel:    currentLevel,
				ToLevel:      currentLevel + 1, // Upgrade one level at a time
				Building:     farmBuilding,
				LevelData:    farmBuilding.GetLevelData(currentLevel + 1),
			}
		}
	}
	return nil
}

// createStorageUpgrade creates a storage upgrade action
func (s *Solver) createStorageUpgrade(state *State, rt models.ResourceType, requiredCap int) Action {
	storageType := resourceToStorage(rt)
	building := s.Buildings[storageType]
	if building == nil {
		return nil
	}

	currentLevel := state.GetBuildingLevel(storageType)
	for level := currentLevel + 1; level <= 20; level++ {
		levelData := building.GetLevelData(level)
		if levelData == nil || levelData.StorageCapacity == nil {
			continue
		}
		if *levelData.StorageCapacity >= requiredCap {
			return &BuildingAction{
				BuildingType: storageType,
				FromLevel:    currentLevel,
				ToLevel:      currentLevel + 1,
				Building:     building,
				LevelData:    building.GetLevelData(currentLevel + 1),
			}
		}
	}
	return nil
}

// createLibraryUpgrade creates a Library upgrade action
func (s *Solver) createLibraryUpgrade(state *State, requiredLevel int) Action {
	building := s.Buildings[models.Library]
	if building == nil {
		return nil
	}

	currentLevel := state.GetBuildingLevel(models.Library)
	if currentLevel >= requiredLevel {
		return nil
	}

	levelData := building.GetLevelData(currentLevel + 1)
	if levelData == nil {
		return nil
	}

	return &BuildingAction{
		BuildingType: models.Library,
		FromLevel:    currentLevel,
		ToLevel:      currentLevel + 1,
		Building:     building,
		LevelData:    levelData,
	}
}

// getResearchActions returns possible research actions
// nolint:unused // Reserved for future research queue extension
func (s *Solver) getResearchActions(state *State) []Action {
	var actions []Action

	// Check for prerequisite technologies needed
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
					if tech != nil {
						actions = append(actions, &ResearchAction{Technology: tech})
					}
				}
			}
		}
	}

	// Add production techs if not researched
	for _, techName := range []string{"Beer tester", "Wheelbarrow"} {
		if !state.ResearchedTechs[techName] {
			tech := s.Technologies[techName]
			if tech != nil {
				actions = append(actions, &ResearchAction{Technology: tech})
			}
		}
	}

	return actions
}

// selectBestActions selects the best action for each free queue
// nolint:unused // Reserved for future multi-queue extension
func (s *Solver) selectBestActions(state *State, actions []Action) []Action {
	var selected []Action

	// Group actions by queue
	byQueue := make(map[QueueType][]Action)
	for _, a := range actions {
		byQueue[a.Queue()] = append(byQueue[a.Queue()], a)
	}

	// Select best action for building queue if free
	if state.Now >= state.BuildingQueueFreeAt {
		buildingActions := byQueue[QueueBuilding]
		if len(buildingActions) > 0 {
			best := s.selectBestAffordable(state, buildingActions)
			if best != nil {
				selected = append(selected, best)
			}
		}
	}

	// Select best action for research queue if free
	if state.Now >= state.ResearchQueueFreeAt {
		researchActions := byQueue[QueueResearch]
		if len(researchActions) > 0 {
			best := s.selectBestAffordable(state, researchActions)
			if best != nil {
				// Check Library level for research
				if ra, ok := best.(*ResearchAction); ok {
					libraryLevel := state.GetBuildingLevel(models.Library)
					if libraryLevel >= ra.Technology.RequiredLibraryLevel {
						selected = append(selected, best)
					}
				}
			}
		}
	}

	return selected
}

// selectBestAffordable selects the best affordable action from the list
// nolint:unused // Reserved for future multi-queue extension
func (s *Solver) selectBestAffordable(state *State, actions []Action) Action {
	// Sort by ROI (descending)
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].ROI(state) > actions[j].ROI(state)
	})

	// Find first affordable action
	for _, action := range actions {
		if s.canAfford(state, action.Costs()) {
			return action
		}
	}

	return nil
}

// canAfford returns true if the state has enough resources for the costs
func (s *Solver) canAfford(state *State, costs models.Costs) bool {
	for rt, cost := range costs {
		if cost == 0 {
			continue
		}
		if rt == models.Food {
			// Food is checked separately (capacity vs used)
			continue
		}
		if state.GetResource(rt) < float64(cost) {
			return false
		}
	}
	return true
}

// calculateWaitTime calculates how long to wait before any action becomes affordable
// nolint:unused // Reserved for future multi-queue extension
func (s *Solver) calculateWaitTime(state *State, actions []Action) int {
	minWait := -1

	for _, action := range actions {
		wait := s.waitTimeForAction(state, action)
		if wait >= 0 {
			if minWait < 0 || wait < minWait {
				minWait = wait
			}
		}
	}

	// Also consider queue wait times
	if state.BuildingQueueFreeAt > state.Now {
		queueWait := state.BuildingQueueFreeAt - state.Now
		if minWait < 0 || queueWait < minWait {
			minWait = queueWait
		}
	}
	if state.ResearchQueueFreeAt > state.Now {
		queueWait := state.ResearchQueueFreeAt - state.Now
		if minWait < 0 || queueWait < minWait {
			minWait = queueWait
		}
	}

	if minWait < 0 {
		return 60 // Default 1 minute if stuck
	}
	return minWait
}

// waitTimeForAction calculates how long to wait before an action becomes affordable
func (s *Solver) waitTimeForAction(state *State, action Action) int {
	costs := action.Costs()
	maxWait := 0

	for _, rt := range []models.ResourceType{models.Wood, models.Stone, models.Iron} {
		cost := costs[rt]
		if cost == 0 {
			continue
		}

		available := state.GetResource(rt)
		if available >= float64(cost) {
			continue
		}

		shortfall := float64(cost) - available
		rate := state.GetProductionRate(rt) * state.ProductionBonus
		if rate <= 0 {
			return -1 // Cannot produce this resource
		}

		secondsNeeded := int(shortfall/rate*3600) + 1
		if secondsNeeded > maxWait {
			maxWait = secondsNeeded
		}
	}

	return maxWait
}

// Helper functions

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

// SolveAllStrategies runs the ROI-based solver (no longer needs multiple strategies)
// Kept for API compatibility with castle CLI
func SolveAllStrategies(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	initialState *models.GameState,
	targetLevels map[models.BuildingType]int,
) (*models.Solution, string, []StrategyResult) {
	solver := NewSolver(buildings, technologies, targetLevels)

	// Clone initial state
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

	solution := solver.Solve(stateCopy)

	// Return single result for compatibility
	results := []StrategyResult{{
		Strategy: "ROI-based",
		Solution: solution,
	}}

	return solution, "ROI-based", results
}

// StrategyResult holds the result of running the solver
type StrategyResult struct {
	Strategy string
	Solution *models.Solution
}
