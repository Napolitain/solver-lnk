package v3

import "github.com/napolitain/solver-lnk/internal/models"

// Action represents any action the solver can take
type Action interface {
	// Queue returns which queue this action uses
	Queue() QueueType

	// CanExecute returns true if the action can be executed now
	CanExecute(state *State) bool

	// Costs returns the resource costs
	Costs() models.Costs

	// Duration returns the action duration in seconds
	Duration() int

	// ROI calculates the return on investment for this action
	// Higher ROI = better action to take
	ROI(state *State) float64

	// Execute performs the action, modifying state
	Execute(state *State)

	// Description returns a human-readable description
	Description() string
}

// ActionResult holds information about a completed action
type ActionResult struct {
	Action      Action
	StartTime   int
	EndTime     int
	Description string
}

// BuildingAction represents a building upgrade action
type BuildingAction struct {
	BuildingType models.BuildingType
	FromLevel    int
	ToLevel      int
	Building     *models.Building
	LevelData    *models.BuildingLevel
}

func (a *BuildingAction) Queue() QueueType {
	return QueueBuilding
}

func (a *BuildingAction) CanExecute(state *State) bool {
	// Check if building queue is free
	if state.Now < state.BuildingQueueFreeAt {
		return false
	}

	// Check current level
	currentLevel := state.GetBuildingLevel(a.BuildingType)
	return currentLevel == a.FromLevel
}

func (a *BuildingAction) Costs() models.Costs {
	return a.LevelData.Costs
}

func (a *BuildingAction) Duration() int {
	return a.LevelData.BuildTimeSeconds
}

func (a *BuildingAction) ROI(state *State) float64 {
	// For production buildings: ROI considers production gain, build time, and resource scarcity
	if a.LevelData.ProductionRate == nil {
		// Non-production buildings get base ROI of 0
		// They're built reactively when needed
		return 0
	}

	// Get current production rate
	var currentRate float64
	if a.FromLevel > 0 {
		prevData := a.Building.GetLevelData(a.FromLevel)
		if prevData != nil && prevData.ProductionRate != nil {
			currentRate = *prevData.ProductionRate
		}
	}

	newRate := *a.LevelData.ProductionRate
	gain := newRate - currentRate
	buildHours := float64(a.LevelData.BuildTimeSeconds) / 3600.0

	if buildHours <= 0 {
		return gain * 1000 // Very fast builds have very high ROI
	}

	// Calculate resource scarcity multiplier
	// A resource is more valuable if its production is low relative to others
	scarcityMultiplier := a.calculateScarcityMultiplier(state)

	// Base ROI
	baseROI := gain / buildHours

	return baseROI * scarcityMultiplier
}

// calculateScarcityMultiplier returns a multiplier based on how scarce the produced resource is
func (a *BuildingAction) calculateScarcityMultiplier(state *State) float64 {
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
	// If demand > supply ratio, the resource is scarce
	var scarcity float64
	switch a.BuildingType {
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

func (a *BuildingAction) Execute(state *State) {
	costs := a.LevelData.Costs
	
	// Deduct costs
	if costs.Wood > 0 {
		state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
	}
	if costs.Stone > 0 {
		state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
	}
	if costs.Iron > 0 {
		state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))
	}

	// Deduct food (workers)
	state.FoodUsed += costs.Food

	// Update building level
	state.SetBuildingLevel(a.BuildingType, a.ToLevel)

	// Update queue
	state.BuildingQueueFreeAt = state.Now + a.LevelData.BuildTimeSeconds
}

func (a *BuildingAction) Description() string {
	return string(a.BuildingType)
}

// ResearchAction represents a technology research action
type ResearchAction struct {
	Technology *models.Technology
}

func (a *ResearchAction) Queue() QueueType {
	return QueueResearch
}

func (a *ResearchAction) CanExecute(state *State) bool {
	// Check if research queue is free
	if state.Now < state.ResearchQueueFreeAt {
		return false
	}

	// Check if already researched
	if state.ResearchedTechs[a.Technology.Name] {
		return false
	}

	// Check Library level
	libraryLevel := state.GetBuildingLevel(models.Library)
	return libraryLevel >= a.Technology.RequiredLibraryLevel
}

func (a *ResearchAction) Costs() models.Costs {
	return a.Technology.Costs
}

func (a *ResearchAction) Duration() int {
	return a.Technology.ResearchTimeSeconds
}

func (a *ResearchAction) ROI(state *State) float64 {
	// Research has high priority when it's a prerequisite
	// Otherwise moderate priority
	// Production techs (Beer tester, Wheelbarrow) have good ROI
	if a.Technology.Name == "Beer tester" || a.Technology.Name == "Wheelbarrow" {
		// 5% production boost is very valuable
		// Rough estimate: affects ~100 resources/hour, so 5/hour gain
		buildHours := float64(a.Technology.ResearchTimeSeconds) / 3600.0
		if buildHours <= 0 {
			return 100
		}
		return 5.0 / buildHours
	}

	// Prerequisite techs have base ROI (they're required, not optional)
	return 0.1
}

func (a *ResearchAction) Execute(state *State) {
	costs := a.Technology.Costs
	
	// Deduct costs
	if costs.Wood > 0 {
		state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
	}
	if costs.Stone > 0 {
		state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
	}
	if costs.Iron > 0 {
		state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))
	}

	// Deduct food if any
	state.FoodUsed += costs.Food

	// NOTE: Don't mark as researched yet - that happens at completion
	// Don't apply production bonus yet - that also happens at completion

	// Update queue
	state.ResearchQueueFreeAt = state.Now + a.Technology.ResearchTimeSeconds
}

func (a *ResearchAction) Description() string {
	return a.Technology.Name
}

// ProductionTechAction represents a production boost tech with its full cost path
// This includes any required Library upgrades
type ProductionTechAction struct {
	Technology           *models.Technology
	LibraryUpgradesNeeded int  // How many Library levels needed
	LibraryBuilding      *models.Building
	CurrentLibraryLevel  int
}

func (a *ProductionTechAction) Queue() QueueType {
	// This is a "meta-action" that triggers Library upgrades (building queue)
	// followed by research (research queue)
	return QueueBuilding
}

func (a *ProductionTechAction) CanExecute(state *State) bool {
	if state.ResearchedTechs[a.Technology.Name] {
		return false
	}
	// Can start if building queue is free (for Library) or Library already high enough
	return state.Now >= state.BuildingQueueFreeAt
}

// TotalLibraryCosts returns the sum of all Library upgrade costs needed
func (a *ProductionTechAction) TotalLibraryCosts() models.Costs {
	var total models.Costs
	if a.LibraryBuilding == nil {
		return total
	}
	for level := a.CurrentLibraryLevel + 1; level <= a.Technology.RequiredLibraryLevel; level++ {
		levelData := a.LibraryBuilding.GetLevelData(level)
		if levelData != nil {
			total.Wood += levelData.Costs.Wood
			total.Stone += levelData.Costs.Stone
			total.Iron += levelData.Costs.Iron
			total.Food += levelData.Costs.Food
		}
	}
	return total
}

// TotalLibraryBuildTime returns the sum of all Library upgrade times
func (a *ProductionTechAction) TotalLibraryBuildTime() int {
	var total int
	if a.LibraryBuilding == nil {
		return total
	}
	for level := a.CurrentLibraryLevel + 1; level <= a.Technology.RequiredLibraryLevel; level++ {
		levelData := a.LibraryBuilding.GetLevelData(level)
		if levelData != nil {
			total += levelData.BuildTimeSeconds
		}
	}
	return total
}

func (a *ProductionTechAction) Costs() models.Costs {
	// Return just the first action's cost (Library or Research)
	if a.LibraryUpgradesNeeded > 0 {
		levelData := a.LibraryBuilding.GetLevelData(a.CurrentLibraryLevel + 1)
		if levelData != nil {
			return levelData.Costs
		}
	}
	return a.Technology.Costs
}

func (a *ProductionTechAction) Duration() int {
	// Return just the first action's duration
	if a.LibraryUpgradesNeeded > 0 {
		levelData := a.LibraryBuilding.GetLevelData(a.CurrentLibraryLevel + 1)
		if levelData != nil {
			return levelData.BuildTimeSeconds
		}
	}
	return a.Technology.ResearchTimeSeconds
}

func (a *ProductionTechAction) ROI(state *State) float64 {
	// Calculate effective ROI for the full path to this production tech
	// 
	// The benefit: 5% of total production rate, forever
	// The cost: Library build time (blocks building queue) + research time (parallel)
	//
	// Since Library blocks building queue, we weight it heavily
	// Research runs in parallel, so it's "free" in terms of blocking
	
	totalProduction := state.GetProductionRate(models.Wood) +
		state.GetProductionRate(models.Stone) +
		state.GetProductionRate(models.Iron)
	
	// 5% boost = gain per hour
	gainPerHour := totalProduction * 0.05 * state.ProductionBonus
	
	// Library time is expensive (blocks building queue)
	libraryBuildSeconds := a.TotalLibraryBuildTime()
	libraryBuildHours := float64(libraryBuildSeconds) / 3600.0
	
	// Research time is cheap (runs in parallel) - weight it at 10%
	researchSeconds := a.Technology.ResearchTimeSeconds
	researchHours := float64(researchSeconds) / 3600.0 * 0.1
	
	totalTimeHours := libraryBuildHours + researchHours
	if totalTimeHours <= 0 {
		return gainPerHour * 100 // Very fast = very good
	}
	
	// ROI = perpetual gain / investment time
	// We estimate "perpetual" as 24 hours of benefit for comparison
	estimatedBenefitHours := 24.0
	totalBenefit := gainPerHour * estimatedBenefitHours
	
	return totalBenefit / totalTimeHours
}

func (a *ProductionTechAction) Execute(state *State) {
	// This shouldn't be called directly - the solver handles the multi-step execution
	// But if called, just start the first Library upgrade
	if a.LibraryUpgradesNeeded > 0 && a.LibraryBuilding != nil {
		levelData := a.LibraryBuilding.GetLevelData(a.CurrentLibraryLevel + 1)
		if levelData != nil {
			costs := levelData.Costs
			if costs.Wood > 0 {
				state.SetResource(models.Wood, state.GetResource(models.Wood)-float64(costs.Wood))
			}
			if costs.Stone > 0 {
				state.SetResource(models.Stone, state.GetResource(models.Stone)-float64(costs.Stone))
			}
			if costs.Iron > 0 {
				state.SetResource(models.Iron, state.GetResource(models.Iron)-float64(costs.Iron))
			}
			state.FoodUsed += costs.Food
			state.SetBuildingLevel(models.Library, a.CurrentLibraryLevel+1)
			state.BuildingQueueFreeAt = state.Now + levelData.BuildTimeSeconds
		}
	}
}

func (a *ProductionTechAction) Description() string {
	if a.LibraryUpgradesNeeded > 0 {
		return "Library (for " + a.Technology.Name + ")"
	}
	return a.Technology.Name
}
