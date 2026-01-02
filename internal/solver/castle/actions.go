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

// TrainUnitAction represents training a single unit
type TrainUnitAction struct {
	UnitType   models.UnitType
	Definition *models.UnitDefinition
}

func (a *TrainUnitAction) Costs() models.Costs {
	return a.Definition.ResourceCosts
}

func (a *TrainUnitAction) Duration() int {
	return a.Definition.TrainingTimeSeconds
}

func (a *TrainUnitAction) Description() string {
	return a.Definition.Name
}

// FoodCost returns the food cost for training this unit
func (a *TrainUnitAction) FoodCost() int {
	return a.Definition.FoodCost
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
