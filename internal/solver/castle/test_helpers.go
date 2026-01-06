package castle

import (
	"github.com/napolitain/solver-lnk/internal/models"
)

// newTestSolver creates a solver with default empty targets for testing
func NewTestSolver(
	buildings map[models.BuildingType]*models.Building,
	technologies map[string]*models.Technology,
	missions []*models.Mission,
	targetLevels map[models.BuildingType]int,
) *Solver {
	// Default Library target is 10
	libraryTarget := 10
	if target, ok := targetLevels[models.Library]; ok {
		libraryTarget = target
	}

	// Get reachable technologies based on Library target
	targetTechs := models.GetTargetTechnologies(technologies, libraryTarget)

	targetUnits := make(map[models.UnitType]int) // Empty = missions only

	return NewSolver(buildings, technologies, missions, targetLevels, targetTechs, targetUnits)
}
