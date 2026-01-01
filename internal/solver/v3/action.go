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
	// For production buildings: ROI = production gain / build time (in hours)
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

	return gain / buildHours
}

func (a *BuildingAction) Execute(state *State) {
	// Deduct costs
	for rt, cost := range a.LevelData.Costs {
		if cost > 0 {
			current := state.GetResource(rt)
			state.SetResource(rt, current-float64(cost))
		}
	}

	// Deduct food (workers)
	foodCost := a.LevelData.Costs[models.Food]
	state.FoodUsed += foodCost

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
	// Deduct costs
	for rt, cost := range a.Technology.Costs {
		if cost > 0 {
			current := state.GetResource(rt)
			state.SetResource(rt, current-float64(cost))
		}
	}

	// Deduct food if any
	foodCost := a.Technology.Costs[models.Food]
	state.FoodUsed += foodCost

	// Mark as researched
	state.ResearchedTechs[a.Technology.Name] = true

	// Apply production bonus
	if a.Technology.Name == "Beer tester" || a.Technology.Name == "Wheelbarrow" {
		state.ProductionBonus += 0.05
	}

	// Update queue
	state.ResearchQueueFreeAt = state.Now + a.Technology.ResearchTimeSeconds
}

func (a *ResearchAction) Description() string {
	return a.Technology.Name
}
