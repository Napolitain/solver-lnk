package castle

import (
	"fmt"

	"github.com/napolitain/solver-lnk/internal/models"
)

// SimulationState tracks state during simulation
type SimulationState struct {
	TimeMinutes            int
	Resources              map[models.ResourceType]float64
	BuildingLevels         map[models.BuildingType]int
	ProductionRates        map[models.ResourceType]float64
	ProductionBonus        float64 // Multiplier from production techs (1.0 = no bonus, 1.05 = 5% bonus)
	StorageCaps            map[models.ResourceType]int
	FoodUsed               int // Total workers (food) consumed by buildings
	FoodCapacity           int // Current food capacity from Farm
	BuildingQueueFreeAt    int
	ResearchQueueFreeAt    int
	CompletedActions       []models.BuildingUpgradeAction
	ResearchActions        []models.ResearchAction
	ResearchedTechnologies map[string]bool
}

// ResourceStrategy defines how resource buildings are prioritized
// WoodLead and QuarryLead indicate how many levels ahead of OreMine they should be
type ResourceStrategy struct {
	WoodLead   int // How many levels Lumberjack stays ahead of OreMine
	QuarryLead int // How many levels Quarry stays ahead of OreMine
}

// String returns the strategy name
func (s ResourceStrategy) String() string {
	if s.WoodLead == 0 && s.QuarryLead == 0 {
		return "RoundRobin"
	}
	return fmt.Sprintf("W+%d/Q+%d", s.WoodLead, s.QuarryLead)
}

// GreedySolver implements the greedy simulation solver
type GreedySolver struct {
	Buildings    map[models.BuildingType]*models.Building
	Technologies map[string]*models.Technology
	InitialState *models.GameState
	TargetLevels map[models.BuildingType]int
	Strategy     ResourceStrategy
}

// NewGreedySolver creates a new greedy solver with default round-robin strategy
func NewGreedySolver(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	initialState *models.GameState,
	targetLevels map[models.BuildingType]int,
) *GreedySolver {
	return &GreedySolver{
		Buildings:    buildings,
		Technologies: technologies,
		InitialState: initialState,
		TargetLevels: targetLevels,
		Strategy:     ResourceStrategy{0, 0},
	}
}

// NewGreedySolverWithStrategy creates a solver with a specific strategy
func NewGreedySolverWithStrategy(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	initialState *models.GameState,
	targetLevels map[models.BuildingType]int,
	strategy ResourceStrategy,
) *GreedySolver {
	return &GreedySolver{
		Buildings:    buildings,
		Technologies: technologies,
		InitialState: initialState,
		TargetLevels: targetLevels,
		Strategy:     strategy,
	}
}

// StrategyResult holds the result of running a single strategy
type StrategyResult struct {
	Strategy ResourceStrategy
	Solution *models.Solution
}

// SolveAllStrategies tries progressively higher wood/quarry leads until no improvement
// Tech research decisions are made automatically via breakeven heuristic
func SolveAllStrategies(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	initialState *models.GameState,
	targetLevels map[models.BuildingType]int,
) (*models.Solution, ResourceStrategy, []StrategyResult) {
	var bestSolution *models.Solution
	var bestStrategy ResourceStrategy
	var results []StrategyResult

	// Generate strategies: start with RoundRobin, then progressively add wood/quarry lead
	// Pattern: (0,0), (1,0), (1,1), (2,0), (2,1), (2,2), (3,0), ...
	// Stop when we've tried 5 consecutive strategies without improvement
	noImprovementCount := 0
	maxNoImprovement := 5

	for woodLead := 0; woodLead <= 10 && noImprovementCount < maxNoImprovement; woodLead++ {
		for quarryLead := 0; quarryLead <= woodLead && noImprovementCount < maxNoImprovement; quarryLead++ {
			strategy := ResourceStrategy{WoodLead: woodLead, QuarryLead: quarryLead}

			// Deep copy initial state for each run
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

			solver := NewGreedySolverWithStrategy(buildings, technologies, stateCopy, targetLevels, strategy)
			solution := solver.Solve()

			results = append(results, StrategyResult{Strategy: strategy, Solution: solution})

			if bestSolution == nil || solution.TotalTimeSeconds < bestSolution.TotalTimeSeconds {
				bestSolution = solution
				bestStrategy = strategy
				noImprovementCount = 0
			} else {
				noImprovementCount++
			}
		}
	}

	return bestSolution, bestStrategy, results
}

// Solve runs the greedy simulation
func (s *GreedySolver) Solve() *models.Solution {
	state := s.initState()
	queue := s.createPrioritizedQueue()

	for len(queue) > 0 {
		// Wait for building queue if needed
		if state.TimeMinutes < state.BuildingQueueFreeAt {
			s.advanceTime(state, state.BuildingQueueFreeAt-state.TimeMinutes)
		}

		// Check if we should trigger production tech research based on strategy
		if s.shouldResearchProductionTech(state, &queue) {
			continue // Tech was scheduled or Library upgrade inserted, re-evaluate queue
		}

		// Select next upgrade
		nextUpgrade := s.selectNextUpgrade(state, queue)
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
		if techName := s.checkTechPrerequisite(state, building, toLevel); techName != "" {
			s.scheduleResearch(state, techName, &queue)
			continue
		}

		// Check food capacity - must have enough workers available
		foodCost := costs[models.Food]
		if state.FoodUsed+foodCost > state.FoodCapacity {
			// Need more Farm capacity first
			farmLevel := state.BuildingLevels[models.Farm]
			queue = insertAtFront(queue, queueItem{models.Farm, farmLevel + 1})
			continue
		}

		// Check storage capacity
		if ok, storageNeeded := s.checkStorageCapacity(state, costs); !ok {
			if storageNeeded != nil {
				// Only insert if not the current item (avoid infinite loop)
				if storageNeeded.bType != bType || storageNeeded.targetLevel != targetLevel {
					queue = insertAtFront(queue, *storageNeeded)
				}
			}
			continue
		}

		// Check if we can afford
		canAfford, waitTime := s.canAffordOrWaitTime(state, costs)
		if !canAfford {
			if waitTime < 0 {
				queue = removeFromQueue(queue, queueIdx)
				continue
			}
			s.advanceTime(state, waitTime)
		}

		// Start upgrade
		startTime := state.TimeMinutes

		// Deduct resources and food (deterministic order)
		for _, resType := range models.AllResourceTypes() {
			cost := costs[resType]
			if cost == 0 {
				continue
			}
			state.Resources[resType] -= float64(cost)
			// Invariant: resources should never go negative
			if state.Resources[resType] < -0.001 { // Small tolerance for float precision
				panic(fmt.Sprintf("BUG: resource %s went negative (%.2f) after deducting %d for %s level %d",
					resType, state.Resources[resType], cost, bType, toLevel))
			}
		}
		state.FoodUsed += foodCost

		// Invariant: food used should never exceed capacity
		if state.FoodUsed > state.FoodCapacity {
			panic(fmt.Sprintf("BUG: food used %d exceeds capacity %d after %s level %d",
				state.FoodUsed, state.FoodCapacity, bType, toLevel))
		}

		// Mark queue busy
		durationMinutes := max(1, levelData.BuildTimeSeconds/60)
		state.BuildingQueueFreeAt = state.TimeMinutes + durationMinutes

		// Advance time
		s.advanceTime(state, durationMinutes)

		// Complete upgrade
		state.BuildingLevels[bType] = toLevel

		// Update production rates
		s.updateProductionRates(state, building, bType, toLevel)

		// Update storage caps
		s.updateStorageCaps(state, building, bType, toLevel)

		// Update food capacity if Farm was upgraded
		if bType == models.Farm {
			state.FoodCapacity = s.getFoodCapacityForLevel(toLevel)
		}

		// Record action
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

		// Remove from queue if target reached
		if state.BuildingLevels[bType] >= targetLevel {
			queue = removeFromQueue(queue, queueIdx)
		}
	}

	// After all buildings are done, research remaining technologies
	s.researchRemainingTechs(state)

	return &models.Solution{
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
	}
}

func (s *GreedySolver) initState() *SimulationState {
	state := &SimulationState{
		TimeMinutes:            0,
		Resources:              make(map[models.ResourceType]float64),
		BuildingLevels:         make(map[models.BuildingType]int),
		ProductionRates:        make(map[models.ResourceType]float64),
		ProductionBonus:        1.0, // No bonus initially
		StorageCaps:            make(map[models.ResourceType]int),
		FoodUsed:               0,
		FoodCapacity:           0,
		BuildingQueueFreeAt:    0,
		ResearchQueueFreeAt:    0,
		CompletedActions:       []models.BuildingUpgradeAction{},
		ResearchActions:        []models.ResearchAction{},
		ResearchedTechnologies: make(map[string]bool),
	}

	// Copy initial resources
	for rt, amount := range s.InitialState.Resources {
		state.Resources[rt] = amount
	}

	// Copy initial building levels (default to 1)
	for _, bt := range models.AllBuildingTypes() {
		if level, ok := s.InitialState.BuildingLevels[bt]; ok {
			state.BuildingLevels[bt] = level
		} else {
			state.BuildingLevels[bt] = 1
		}
	}

	// Copy researched technologies and calculate initial production bonus
	for k, v := range s.InitialState.ResearchedTechnologies {
		state.ResearchedTechnologies[k] = v
		if v && (k == "Beer tester" || k == "Wheelbarrow") {
			state.ProductionBonus += 0.05
		}
	}

	// Calculate initial production rates
	state.ProductionRates = s.calculateProductionRates(state.BuildingLevels)
	state.StorageCaps = s.calculateStorageCaps(state.BuildingLevels)

	// Calculate initial food capacity from Farm level
	state.FoodCapacity = s.getFoodCapacityForLevel(state.BuildingLevels[models.Farm])

	return state
}

func (s *GreedySolver) advanceTime(state *SimulationState, minutes int) {
	hours := float64(minutes) / 60.0
	state.TimeMinutes += minutes

	// Deterministic order for production
	for _, rt := range models.AllResourceTypes() {
		rate := state.ProductionRates[rt]
		if rate == 0 {
			continue
		}
		// Apply production bonus from researched technologies
		produced := rate * hours * state.ProductionBonus
		state.Resources[rt] += produced

		// Cap at storage limit
		if cap, ok := state.StorageCaps[rt]; ok {
			if state.Resources[rt] > float64(cap) {
				state.Resources[rt] = float64(cap)
			}
		}
	}
}

type upgradeSelection struct {
	bType       models.BuildingType
	targetLevel int
	queueIdx    int
}

func (s *GreedySolver) selectNextUpgrade(state *SimulationState, queue []queueItem) *upgradeSelection {
	if len(queue) == 0 {
		return nil
	}

	// Storage/Farm/Library items at front get priority (dynamically inserted)
	priorityTypes := map[models.BuildingType]bool{
		models.WoodStore:  true,
		models.StoneStore: true,
		models.OreStore:   true,
		models.Farm:       true,
		models.Library:    true, // For production tech prerequisites
	}

	first := queue[0]
	if priorityTypes[first.bType] {
		current := state.BuildingLevels[first.bType]
		if current < first.targetLevel {
			return &upgradeSelection{first.bType, first.targetLevel, 0}
		}
	}

	// Return first valid item in queue
	for idx, item := range queue {
		current := state.BuildingLevels[item.bType]
		if current < item.targetLevel {
			return &upgradeSelection{item.bType, item.targetLevel, idx}
		}
	}

	return nil
}

func (s *GreedySolver) canAffordOrWaitTime(state *SimulationState, costs models.Costs) (bool, int) {
	maxWait := 0

	// Deterministic order
	for _, rt := range models.AllResourceTypes() {
		cost := costs[rt]
		if cost == 0 {
			continue
		}
		available := state.Resources[rt]
		if available >= float64(cost) {
			continue
		}

		shortfall := float64(cost) - available
		rate := state.ProductionRates[rt]

		if rate <= 0 {
			return false, -1 // Cannot produce
		}

		hoursNeeded := shortfall / rate
		minutesNeeded := int(hoursNeeded*60) + 1

		if minutesNeeded > maxWait {
			maxWait = minutesNeeded
		}
	}

	if maxWait > 0 {
		return false, maxWait
	}
	return true, 0
}

func (s *GreedySolver) checkStorageCapacity(state *SimulationState, costs models.Costs) (bool, *queueItem) {
	storageMap := map[models.ResourceType]models.BuildingType{
		models.Wood:  models.WoodStore,
		models.Stone: models.StoneStore,
		models.Iron:  models.OreStore,
		models.Food:  models.Farm,
	}

	// Deterministic order
	for _, rt := range models.AllResourceTypes() {
		cost := costs[rt]
		if cost == 0 {
			continue
		}
		if rt == models.Food {
			// Food: check current available
			available := state.Resources[models.Food]
			if available < float64(cost) {
				building := s.Buildings[models.Farm]
				if building == nil {
					continue
				}
				currentLevel := state.BuildingLevels[models.Farm]
				for level := currentLevel + 1; level <= 30; level++ {
					levelData := building.GetLevelData(level)
					if levelData == nil || levelData.StorageCapacity == nil {
						continue
					}
					if *levelData.StorageCapacity >= cost {
						return false, &queueItem{models.Farm, level}
					}
				}
				return false, nil
			}
		} else {
			// Wood/Stone/Iron: check storage capacity
			cap := state.StorageCaps[rt]
			if cost > cap {
				storageBuilding := storageMap[rt]
				building := s.Buildings[storageBuilding]
				if building == nil {
					continue
				}
				currentLevel := state.BuildingLevels[storageBuilding]
				for level := currentLevel + 1; level <= 30; level++ {
					levelData := building.GetLevelData(level)
					if levelData == nil || levelData.StorageCapacity == nil {
						continue
					}
					if *levelData.StorageCapacity >= cost {
						return false, &queueItem{storageBuilding, level}
					}
				}
				return false, nil
			}
		}
	}

	return true, nil
}

func (s *GreedySolver) checkTechPrerequisite(state *SimulationState, building *models.Building, toLevel int) string {
	techName, ok := building.TechnologyPrerequisites[toLevel]
	if !ok {
		return ""
	}
	if state.ResearchedTechnologies[techName] {
		return ""
	}
	return techName
}

// scheduleResearch attempts to schedule a technology research
// Returns true if research was successfully started, false if prerequisites not met
func (s *GreedySolver) scheduleResearch(state *SimulationState, techName string, queue *[]queueItem) bool {
	tech, ok := s.Technologies[techName]
	if !ok {
		// Tech not found, mark as researched to skip
		state.ResearchedTechnologies[techName] = true
		return true // Treat as "done"
	}

	// Check Library level
	libraryLevel := state.BuildingLevels[models.Library]
	if libraryLevel < tech.RequiredLibraryLevel {
		*queue = insertAtFront(*queue, queueItem{models.Library, tech.RequiredLibraryLevel})
		return false
	}

	// Wait for research queue
	if state.TimeMinutes < state.ResearchQueueFreeAt {
		s.advanceTime(state, state.ResearchQueueFreeAt-state.TimeMinutes)
	}

	// Check storage
	if ok, storageNeeded := s.checkStorageCapacity(state, tech.Costs); !ok {
		if storageNeeded != nil {
			*queue = insertAtFront(*queue, *storageNeeded)
		}
		return false
	}

	// Check food capacity for research
	foodCost := tech.Costs[models.Food]
	if foodCost > 0 && state.FoodUsed+foodCost > state.FoodCapacity {
		// Need more Farm capacity first - check if already in queue
		farmLevel := state.BuildingLevels[models.Farm]
		targetFarmLevel := farmLevel + 1
		farmInQueue := false
		for _, item := range *queue {
			if item.bType == models.Farm && item.targetLevel >= targetFarmLevel {
				farmInQueue = true
				break
			}
		}
		if !farmInQueue {
			*queue = insertAtFront(*queue, queueItem{models.Farm, targetFarmLevel})
		}
		return false
	}

	// Wait for resources
	canAfford, waitTime := s.canAffordOrWaitTime(state, tech.Costs)
	if !canAfford {
		if waitTime > 0 {
			s.advanceTime(state, waitTime)
		}
	}

	// Re-check affordability after waiting
	canAfford, _ = s.canAffordOrWaitTime(state, tech.Costs)
	if !canAfford {
		return false // Will retry next iteration
	}

	// Start research
	startTime := state.TimeMinutes

	// Deduct resources (deterministic order)
	for _, rt := range models.AllResourceTypes() {
		cost := tech.Costs[rt]
		if cost == 0 {
			continue
		}
		state.Resources[rt] -= float64(cost)
		// Invariant: resources should never go negative
		if state.Resources[rt] < -0.001 {
			panic(fmt.Sprintf("BUG: resource %s went negative (%.2f) after deducting %d for research %s",
				rt, state.Resources[rt], cost, techName))
		}
	}

	// Deduct food (workers) for research
	if foodCost > 0 {
		state.FoodUsed += foodCost
	}

	// Calculate duration
	durationMinutes := max(1, tech.ResearchTimeSeconds/60)
	researchEndTime := state.TimeMinutes + durationMinutes
	state.ResearchQueueFreeAt = researchEndTime

	// Record action
	state.ResearchActions = append(state.ResearchActions, models.ResearchAction{
		TechnologyName: techName,
		StartTime:      startTime * 60,
		EndTime:        researchEndTime * 60,
		Costs:          tech.Costs,
		FoodUsed:       state.FoodUsed,
		FoodCapacity:   state.FoodCapacity,
	})

	// Wait until research completes before marking as researched
	// This ensures buildings that require this tech wait for it
	s.advanceTime(state, durationMinutes)

	// NOW mark as researched (after completion)
	state.ResearchedTechnologies[techName] = true

	// Apply production bonus if this is a production tech
	if techName == "Beer tester" || techName == "Wheelbarrow" {
		state.ProductionBonus += 0.05
	}

	return true
}

// researchRemainingTechs researches all technologies that haven't been researched yet
// Called after all building upgrades are complete
func (s *GreedySolver) researchRemainingTechs(state *SimulationState) {
	// Collect unresearched techs and sort by required library level
	type techToResearch struct {
		name         string
		tech         *models.Technology
		libraryLevel int
	}

	var remaining []techToResearch
	for name, tech := range s.Technologies {
		if !state.ResearchedTechnologies[name] {
			remaining = append(remaining, techToResearch{name, tech, tech.RequiredLibraryLevel})
		}
	}

	// Sort by library level (ascending) for deterministic order
	// Use bubble sort for simplicity and determinism
	for i := 0; i < len(remaining); i++ {
		for j := i + 1; j < len(remaining); j++ {
			if remaining[j].libraryLevel < remaining[i].libraryLevel ||
				(remaining[j].libraryLevel == remaining[i].libraryLevel && remaining[j].name < remaining[i].name) {
				remaining[i], remaining[j] = remaining[j], remaining[i]
			}
		}
	}

	// Research each tech in order
	for _, tr := range remaining {
		// Upgrade Library if needed
		for state.BuildingLevels[models.Library] < tr.libraryLevel {
			s.upgradeLibraryForTech(state, state.BuildingLevels[models.Library]+1)
		}

		// Wait for research queue
		if state.TimeMinutes < state.ResearchQueueFreeAt {
			s.advanceTime(state, state.ResearchQueueFreeAt-state.TimeMinutes)
		}

		// Wait for resources
		canAfford, waitTime := s.canAffordOrWaitTime(state, tr.tech.Costs)
		if !canAfford && waitTime > 0 {
			s.advanceTime(state, waitTime)
		}

		// Research the tech
		var dummyQueue []queueItem
		s.scheduleResearch(state, tr.name, &dummyQueue)
	}
}

// upgradeLibraryForTech upgrades Library to the specified level (for end-game tech research)
func (s *GreedySolver) upgradeLibraryForTech(state *SimulationState, toLevel int) {
	library := s.Buildings[models.Library]
	if library == nil {
		return
	}

	levelData := library.GetLevelData(toLevel)
	if levelData == nil {
		return
	}

	// Wait for building queue
	if state.TimeMinutes < state.BuildingQueueFreeAt {
		s.advanceTime(state, state.BuildingQueueFreeAt-state.TimeMinutes)
	}

	// Wait for resources
	canAfford, waitTime := s.canAffordOrWaitTime(state, levelData.Costs)
	if !canAfford && waitTime > 0 {
		s.advanceTime(state, waitTime)
	}

	// Deduct resources
	startTime := state.TimeMinutes
	for _, rt := range models.AllResourceTypes() {
		cost := levelData.Costs[rt]
		if cost > 0 {
			state.Resources[rt] -= float64(cost)
		}
	}

	// Upgrade
	durationMinutes := max(1, levelData.BuildTimeSeconds/60)
	state.BuildingQueueFreeAt = state.TimeMinutes + durationMinutes
	s.advanceTime(state, durationMinutes)

	fromLevel := state.BuildingLevels[models.Library]
	state.BuildingLevels[models.Library] = toLevel

	// Record action
	state.CompletedActions = append(state.CompletedActions, models.BuildingUpgradeAction{
		BuildingType: models.Library,
		FromLevel:    fromLevel,
		ToLevel:      toLevel,
		StartTime:    startTime * 60,
		EndTime:      state.TimeMinutes * 60,
		Costs:        levelData.Costs,
		FoodUsed:     state.FoodUsed,
		FoodCapacity: state.FoodCapacity,
	})
}

// shouldResearchProductionTech uses breakeven heuristic to decide if production tech is worth researching
// Returns true if a tech was scheduled or Library upgrade inserted (caller should re-evaluate queue)
func (s *GreedySolver) shouldResearchProductionTech(state *SimulationState, queue *[]queueItem) bool {
	// Check Beer tester (5% boost, requires Library 3)
	if !state.ResearchedTechnologies["Beer tester"] {
		if s.isTechWorthResearching(state, "Beer tester", 3, queue) {
			return true
		}
	}

	// Check Wheelbarrow (5% boost, requires Library 8)
	if !state.ResearchedTechnologies["Wheelbarrow"] {
		if s.isTechWorthResearching(state, "Wheelbarrow", 8, queue) {
			return true
		}
	}

	return false
}

// isTechWorthResearching calculates if researching a production tech is worth it using breakeven analysis
func (s *GreedySolver) isTechWorthResearching(state *SimulationState, techName string, requiredLibraryLevel int, queue *[]queueItem) bool {
	_, ok := s.Technologies[techName]
	if !ok {
		return false
	}

	// Calculate total investment cost (Library upgrades + tech research)
	investmentCost := s.calculateTechInvestment(state, techName, requiredLibraryLevel)

	// Calculate remaining resource needs to complete all targets
	remainingNeeds := s.estimateRemainingResourceNeeds(state)

	// The 5% boost effectively "saves" us 5% of remaining resource needs
	// because we'll produce them faster
	gain := 0.05 * remainingNeeds

	// Research is worth it if gain > investment
	// This means the resources we save by producing faster exceed the cost of researching
	if gain <= investmentCost {
		return false
	}

	// Check if we need to upgrade Library first
	currentLibraryLevel := state.BuildingLevels[models.Library]
	if currentLibraryLevel < requiredLibraryLevel {
		// Check if Library upgrade is already in queue
		libraryInQueue := false
		for _, item := range *queue {
			if item.bType == models.Library && item.targetLevel >= requiredLibraryLevel {
				libraryInQueue = true
				break
			}
		}
		if !libraryInQueue {
			*queue = insertAtFront(*queue, queueItem{models.Library, requiredLibraryLevel})
		}
		return false // Let normal queue processing handle the Library upgrade
	}

	// Check if research queue is free
	if state.TimeMinutes < state.ResearchQueueFreeAt {
		return false // Wait for research queue, don't block building queue
	}

	// Schedule the research - returns true only if research actually started
	return s.scheduleResearch(state, techName, queue)
}

// calculateTechInvestment returns total resource cost to research a tech (including Library upgrades)
func (s *GreedySolver) calculateTechInvestment(state *SimulationState, techName string, requiredLibraryLevel int) float64 {
	var totalCost float64

	// Add Library upgrade costs if needed
	currentLibraryLevel := state.BuildingLevels[models.Library]
	library := s.Buildings[models.Library]
	if library != nil {
		for level := currentLibraryLevel + 1; level <= requiredLibraryLevel; level++ {
			levelData := library.GetLevelData(level)
			if levelData != nil {
				for _, rt := range []models.ResourceType{models.Wood, models.Stone, models.Iron} {
					totalCost += float64(levelData.Costs[rt])
				}
			}
		}
	}

	// Add tech research cost
	tech, ok := s.Technologies[techName]
	if ok {
		for _, rt := range []models.ResourceType{models.Wood, models.Stone, models.Iron} {
			totalCost += float64(tech.Costs[rt])
		}
	}

	return totalCost
}

// estimateRemainingResourceNeeds calculates total resources still needed to complete all targets
func (s *GreedySolver) estimateRemainingResourceNeeds(state *SimulationState) float64 {
	var totalNeeds float64

	for bType, targetLevel := range s.TargetLevels {
		currentLevel := state.BuildingLevels[bType]
		building := s.Buildings[bType]
		if building == nil {
			continue
		}

		for level := currentLevel + 1; level <= targetLevel; level++ {
			levelData := building.GetLevelData(level)
			if levelData == nil {
				continue
			}
			for _, rt := range []models.ResourceType{models.Wood, models.Stone, models.Iron} {
				totalNeeds += float64(levelData.Costs[rt])
			}
		}
	}

	return totalNeeds
}

func (s *GreedySolver) updateProductionRates(state *SimulationState, building *models.Building, bType models.BuildingType, toLevel int) {
	resourceMap := map[models.BuildingType]models.ResourceType{
		models.Lumberjack: models.Wood,
		models.Quarry:     models.Stone,
		models.OreMine:    models.Iron,
	}

	if rt, ok := resourceMap[bType]; ok {
		levelData := building.GetLevelData(toLevel)
		if levelData != nil && levelData.ProductionRate != nil {
			state.ProductionRates[rt] = *levelData.ProductionRate
		}
	}
}

func (s *GreedySolver) updateStorageCaps(state *SimulationState, building *models.Building, bType models.BuildingType, toLevel int) {
	storageMap := map[models.BuildingType]models.ResourceType{
		models.WoodStore:  models.Wood,
		models.StoneStore: models.Stone,
		models.OreStore:   models.Iron,
		models.Farm:       models.Food,
	}

	if rt, ok := storageMap[bType]; ok {
		levelData := building.GetLevelData(toLevel)
		if levelData != nil && levelData.StorageCapacity != nil {
			state.StorageCaps[rt] = *levelData.StorageCapacity
			// Farm refills food to new capacity
			if rt == models.Food {
				state.Resources[models.Food] = float64(*levelData.StorageCapacity)
			}
		}
	}
}

func (s *GreedySolver) calculateProductionRates(buildingLevels map[models.BuildingType]int) map[models.ResourceType]float64 {
	rates := map[models.ResourceType]float64{
		models.Wood:  0,
		models.Stone: 0,
		models.Iron:  0,
		models.Food:  0, // No production
	}

	productionBuildings := map[models.BuildingType]models.ResourceType{
		models.Lumberjack: models.Wood,
		models.Quarry:     models.Stone,
		models.OreMine:    models.Iron,
	}

	for bType, rt := range productionBuildings {
		level := buildingLevels[bType]
		if building, ok := s.Buildings[bType]; ok {
			if levelData := building.GetLevelData(level); levelData != nil && levelData.ProductionRate != nil {
				rates[rt] = *levelData.ProductionRate
			}
		}
	}

	return rates
}

func (s *GreedySolver) calculateStorageCaps(buildingLevels map[models.BuildingType]int) map[models.ResourceType]int {
	caps := map[models.ResourceType]int{
		models.Wood:  999999,
		models.Stone: 999999,
		models.Iron:  999999,
		models.Food:  40, // Default Farm L1
	}

	storageBuildings := map[models.BuildingType]models.ResourceType{
		models.WoodStore:  models.Wood,
		models.StoneStore: models.Stone,
		models.OreStore:   models.Iron,
		models.Farm:       models.Food,
	}

	for bType, rt := range storageBuildings {
		level := buildingLevels[bType]
		if building, ok := s.Buildings[bType]; ok {
			if levelData := building.GetLevelData(level); levelData != nil && levelData.StorageCapacity != nil {
				caps[rt] = *levelData.StorageCapacity
			}
		}
	}

	return caps
}

type queueItem struct {
	bType       models.BuildingType
	targetLevel int
}

func (s *GreedySolver) createPrioritizedQueue() []queueItem {
	var queue []queueItem

	// 1. Resource production buildings - based on strategy
	queue = append(queue, s.createResourceQueue()...)

	// 2. Storage buildings - interleaved
	storageBuildings := []models.BuildingType{
		models.WoodStore,
		models.StoneStore,
		models.OreStore,
	}

	maxStorageLevel := 0
	for _, bt := range storageBuildings {
		if target, ok := s.TargetLevels[bt]; ok && target > maxStorageLevel {
			maxStorageLevel = target
		}
	}

	for level := 2; level <= maxStorageLevel; level++ {
		for _, bt := range storageBuildings {
			if target, ok := s.TargetLevels[bt]; ok && level <= target {
				queue = append(queue, queueItem{bt, level})
			}
		}
	}

	// 3. Core buildings
	coreBuildings := []models.BuildingType{models.Keep, models.Library}
	for _, bt := range coreBuildings {
		if target, ok := s.TargetLevels[bt]; ok {
			for level := 2; level <= target; level++ {
				queue = append(queue, queueItem{bt, level})
			}
		}
	}

	// 4. Military and other
	otherBuildings := []models.BuildingType{
		models.Arsenal,
		models.Tavern,
		models.Market,
		models.Fortifications,
	}
	for _, bt := range otherBuildings {
		if target, ok := s.TargetLevels[bt]; ok {
			for level := 2; level <= target; level++ {
				queue = append(queue, queueItem{bt, level})
			}
		}
	}

	// 5. Farm at end
	if target, ok := s.TargetLevels[models.Farm]; ok {
		for level := 2; level <= target; level++ {
			queue = append(queue, queueItem{models.Farm, level})
		}
	}

	return queue
}

// createResourceQueue builds the resource building queue based on strategy
// WoodLead and QuarryLead define how many levels ahead of OreMine each should be
// Pattern: interleave LJ/Q while maintaining lead over OM
// E.g., W+2/Q+1: LJ2, Q2, LJ3, Q3, LJ4, OM2, LJ5, Q4, OM3, ...
func (s *GreedySolver) createResourceQueue() []queueItem {
	var queue []queueItem

	ljTarget := s.TargetLevels[models.Lumberjack]
	qTarget := s.TargetLevels[models.Quarry]
	omTarget := s.TargetLevels[models.OreMine]

	woodLead := s.Strategy.WoodLead
	quarryLead := s.Strategy.QuarryLead

	ljLevel, qLevel, omLevel := 2, 2, 2

	for ljLevel <= ljTarget || qLevel <= qTarget || omLevel <= omTarget {
		added := false

		// Try to add one LJ if needed to maintain lead
		if ljLevel <= ljTarget && ljLevel < omLevel+woodLead+1 {
			queue = append(queue, queueItem{models.Lumberjack, ljLevel})
			ljLevel++
			added = true
		}

		// Try to add one Q if needed to maintain lead
		if qLevel <= qTarget && qLevel < omLevel+quarryLead+1 {
			queue = append(queue, queueItem{models.Quarry, qLevel})
			qLevel++
			added = true
		}

		// If both LJ and Q are far enough ahead, add OM
		if !added || (ljLevel >= omLevel+woodLead+1 && qLevel >= omLevel+quarryLead+1) {
			if omLevel <= omTarget {
				queue = append(queue, queueItem{models.OreMine, omLevel})
				omLevel++
				added = true
			}
		}

		// If nothing was added, finish remaining buildings
		if !added {
			if ljLevel <= ljTarget {
				queue = append(queue, queueItem{models.Lumberjack, ljLevel})
				ljLevel++
			} else if qLevel <= qTarget {
				queue = append(queue, queueItem{models.Quarry, qLevel})
				qLevel++
			}
		}
	}

	return queue
}

func removeFromQueue(queue []queueItem, idx int) []queueItem {
	return append(queue[:idx], queue[idx+1:]...)
}

func insertAtFront(queue []queueItem, item queueItem) []queueItem {
	return append([]queueItem{item}, queue...)
}

// getFoodCapacityForLevel returns food (worker) capacity for a given Farm level
func (s *GreedySolver) getFoodCapacityForLevel(farmLevel int) int {
	// Get from building data
	if farm, ok := s.Buildings[models.Farm]; ok {
		if levelData := farm.GetLevelData(farmLevel); levelData != nil && levelData.StorageCapacity != nil {
			return *levelData.StorageCapacity
		}
	}
	// Fallback defaults (from game data)
	capacities := map[int]int{
		1: 40, 2: 52, 3: 67, 4: 86, 5: 109, 6: 137, 7: 171, 8: 210,
		9: 256, 10: 310, 11: 372, 12: 443, 13: 523, 14: 612, 15: 710,
		16: 817, 17: 931, 18: 1061, 19: 1210, 20: 1379, 21: 1572,
		22: 1792, 23: 2043, 24: 2329, 25: 2655, 26: 3027, 27: 3451,
		28: 3900, 29: 4407, 30: 5000,
	}
	if cap, ok := capacities[farmLevel]; ok {
		return cap
	}
	return 40
}
