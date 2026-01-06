package castle

import "github.com/napolitain/solver-lnk/internal/models"

// Action is the interface for all solver actions
type Action interface {
	Costs() models.Costs
	Duration() int
	Description() string
}

// BuildingAction represents a building upgrade
type BuildingAction struct {
	BuildingType models.BuildingType
	FromLevel    int
	ToLevel      int
	Building     *models.Building
	LevelData    *models.BuildingLevel
}

func (a *BuildingAction) Costs() models.Costs {
	return a.LevelData.Costs
}

func (a *BuildingAction) Duration() int {
	return a.LevelData.BuildTimeSeconds
}

func (a *BuildingAction) Description() string {
	return string(a.BuildingType)
}

// ResearchAction represents a technology research
type ResearchAction struct {
	Technology *models.Technology
}

func (a *ResearchAction) Costs() models.Costs {
	return a.Technology.Costs
}

func (a *ResearchAction) Duration() int {
	return a.Technology.ResearchTimeSeconds
}

func (a *ResearchAction) Description() string {
	return a.Technology.Name
}

// TrainUnitAction represents training units (can be batched)
type TrainUnitAction struct {
	UnitType   models.UnitType
	Definition *models.UnitDefinition
	Count      int // Number of units in this batch (default 1)
}

func (a *TrainUnitAction) Costs() models.Costs {
	count := a.Count
	if count == 0 {
		count = 1 // Default to 1 if not set
	}
	return models.Costs{
		Wood:  a.Definition.ResourceCosts.Wood * count,
		Stone: a.Definition.ResourceCosts.Stone * count,
		Iron:  a.Definition.ResourceCosts.Iron * count,
		Food:  a.Definition.FoodCost * count,
	}
}

func (a *TrainUnitAction) Duration() int {
	count := a.Count
	if count == 0 {
		count = 1 // Default to 1 if not set
	}
	return a.Definition.TrainingTimeSeconds * count
}

func (a *TrainUnitAction) Description() string {
	return a.Definition.Name
}

// FoodCost returns the total food cost for this batch
func (a *TrainUnitAction) FoodCost() int {
	count := a.Count
	if count == 0 {
		count = 1 // Default to 1 if not set
	}
	return a.Definition.FoodCost * count
}

// StartMissionAction represents starting a tavern mission
type StartMissionAction struct {
	Mission *models.Mission
}

func (a *StartMissionAction) Costs() models.Costs {
	return a.Mission.ResourceCosts
}

func (a *StartMissionAction) Duration() int {
	return a.Mission.DurationMinutes * 60
}

func (a *StartMissionAction) Description() string {
	return a.Mission.Name
}
