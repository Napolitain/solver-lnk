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
// Default: all techs, no specific units (missions only)
targetTechs := make([]string, 0, len(technologies))
for name := range technologies {
targetTechs = append(targetTechs, name)
}

targetUnits := make(map[models.UnitType]int) // Empty = missions only

return NewSolver(buildings, technologies, missions, targetLevels, targetTechs, targetUnits)
}
