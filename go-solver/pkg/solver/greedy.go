package solver

import (
	"github.com/napolitain/solver-lnk/pkg/models"
)

// SimulationState tracks state during simulation
type SimulationState struct {
	TimeMinutes            int
	Resources              map[models.ResourceType]float64
	BuildingLevels         map[models.BuildingType]int
	ProductionRates        map[models.ResourceType]float64
	StorageCaps            map[models.ResourceType]int
	BuildingQueueFreeAt    int
	ResearchQueueFreeAt    int
	CompletedActions       []models.BuildingUpgradeAction
	ResearchActions        []models.ResearchAction
	ResearchedTechnologies map[string]bool
}

// GreedySolver implements the greedy simulation solver
type GreedySolver struct {
	Buildings    map[models.BuildingType]*models.Building
	Technologies map[string]*models.Technology
	InitialState *models.GameState
	TargetLevels map[models.BuildingType]int
}

// NewGreedySolver creates a new greedy solver
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
	}
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

		// Deduct resources
		for resType, cost := range costs {
			state.Resources[resType] -= float64(cost)
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

		// Record action
		state.CompletedActions = append(state.CompletedActions, models.BuildingUpgradeAction{
			BuildingType: bType,
			FromLevel:    currentLevel,
			ToLevel:      toLevel,
			StartTime:    startTime * 60,
			EndTime:      state.TimeMinutes * 60,
			Costs:        costs,
		})

		// Remove from queue if target reached
		if state.BuildingLevels[bType] >= targetLevel {
			queue = removeFromQueue(queue, queueIdx)
		}
	}

	return &models.Solution{
		BuildingActions:  state.CompletedActions,
		ResearchActions:  state.ResearchActions,
		TotalTimeSeconds: state.TimeMinutes * 60,
		FinalState: &models.GameState{
			BuildingLevels:         state.BuildingLevels,
			Resources:              state.Resources,
			ResearchedTechnologies: state.ResearchedTechnologies,
		},
	}
}

func (s *GreedySolver) initState() *SimulationState {
	state := &SimulationState{
		TimeMinutes:            0,
		Resources:              make(map[models.ResourceType]float64),
		BuildingLevels:         make(map[models.BuildingType]int),
		ProductionRates:        make(map[models.ResourceType]float64),
		StorageCaps:            make(map[models.ResourceType]int),
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

	// Calculate initial production rates
	state.ProductionRates = s.calculateProductionRates(state.BuildingLevels)
	state.StorageCaps = s.calculateStorageCaps(state.BuildingLevels)

	return state
}

func (s *GreedySolver) advanceTime(state *SimulationState, minutes int) {
	hours := float64(minutes) / 60.0
	state.TimeMinutes += minutes

	for rt, rate := range state.ProductionRates {
		produced := rate * hours
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

	// Storage/Farm items at front get priority (dynamically inserted)
	storageTypes := map[models.BuildingType]bool{
		models.WoodStore:  true,
		models.StoneStore: true,
		models.OreStore:   true,
		models.Farm:       true,
	}

	first := queue[0]
	if storageTypes[first.bType] {
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

	for rt, cost := range costs {
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

	for rt, cost := range costs {
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

func (s *GreedySolver) scheduleResearch(state *SimulationState, techName string, queue *[]queueItem) {
	tech, ok := s.Technologies[techName]
	if !ok {
		// Tech not found, mark as researched to skip
		state.ResearchedTechnologies[techName] = true
		return
	}

	// Check Library level
	libraryLevel := state.BuildingLevels[models.Library]
	if libraryLevel < tech.RequiredLibraryLevel {
		*queue = insertAtFront(*queue, queueItem{models.Library, tech.RequiredLibraryLevel})
		return
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
		return
	}

	// Wait for resources
	canAfford, waitTime := s.canAffordOrWaitTime(state, tech.Costs)
	if !canAfford {
		if waitTime < 0 {
			// Can't produce, mark as researched to skip
			state.ResearchedTechnologies[techName] = true
			return
		}
		s.advanceTime(state, waitTime)
	}

	// Start research
	startTime := state.TimeMinutes

	// Deduct resources
	for rt, cost := range tech.Costs {
		state.Resources[rt] -= float64(cost)
	}

	// Calculate duration
	durationMinutes := max(1, tech.ResearchTimeSeconds/60)
	state.ResearchQueueFreeAt = state.TimeMinutes + durationMinutes

	// Record action
	state.ResearchActions = append(state.ResearchActions, models.ResearchAction{
		TechnologyName: techName,
		StartTime:      startTime * 60,
		EndTime:        (startTime + durationMinutes) * 60,
		Costs:          tech.Costs,
	})

	// Mark as researched
	state.ResearchedTechnologies[techName] = true
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

	// 1. Resource production buildings - interleaved
	resourceBuildings := []models.BuildingType{
		models.Lumberjack,
		models.Quarry,
		models.OreMine,
	}

	maxResourceLevel := 0
	for _, bt := range resourceBuildings {
		if target, ok := s.TargetLevels[bt]; ok && target > maxResourceLevel {
			maxResourceLevel = target
		}
	}

	for level := 2; level <= maxResourceLevel; level++ {
		for _, bt := range resourceBuildings {
			if target, ok := s.TargetLevels[bt]; ok && level <= target {
				queue = append(queue, queueItem{bt, level})
			}
		}
	}

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

func removeFromQueue(queue []queueItem, idx int) []queueItem {
	return append(queue[:idx], queue[idx+1:]...)
}

func insertAtFront(queue []queueItem, item queueItem) []queueItem {
	return append([]queueItem{item}, queue...)
}
