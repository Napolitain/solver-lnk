package castle

// Game mechanics constants
const (
	// ProductionTechBonus is the production multiplier gained from production technologies
	// (Beer tester and Wheelbarrow each provide +5%)
	ProductionTechBonus = 0.05

	// MinFoodHeadroomForTraining is the minimum spare food capacity required before
	// starting unit training (prevents exhausting food capacity on training)
	MinFoodHeadroomForTraining = 5

	// PrerequisiteTechBuffer is the time buffer (in seconds) for starting prerequisite
	// technology research before it's actually needed (1 day = 86400 seconds)
	PrerequisiteTechBuffer = 86400
)
