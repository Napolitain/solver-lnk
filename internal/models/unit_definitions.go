package models

// UnitDefinition contains static unit data
type UnitDefinition struct {
	Type                UnitType
	Name                string
	FoodCost            int
	ResourceCosts       Costs
	TrainingTimeSeconds int
	RequiredTech        string // Empty if no tech needed
	SpeedMinutesField   float64
	TransportCapacity   int
	Category            string // Infantry, Cavalry, Artillery, Transport

	// Defense stats (only for combat units)
	DefenseVsCavalry   int
	DefenseVsInfantry  int
	DefenseVsArtillery int
}

// UnitTechRequirements maps units to their required technology
var UnitTechRequirements = map[UnitType]string{
	Spearman:    "",             // No tech needed
	Swordsman:   "Swordsmith",   // Library 4
	Archer:      "Longbow",      // Library 1
	Crossbowman: "Crossbow",     // Library 5
	Horseman:    "",             // No tech needed
	Lancer:      "Horse armour", // Library 7
}

// AllUnitDefinitions returns definitions for all trainable units
func AllUnitDefinitions() []*UnitDefinition {
	return []*UnitDefinition{
		{
			Type:                Spearman,
			Name:                "Spearman",
			FoodCost:            1,
			ResourceCosts:       Costs{Wood: 18, Stone: 6, Iron: 30},
			TrainingTimeSeconds: 750, // 12:30
			RequiredTech:        "",
			SpeedMinutesField:   11.666667,
			TransportCapacity:   12,
			Category:            "Infantry",
			DefenseVsCavalry:    59,
			DefenseVsInfantry:   32,
			DefenseVsArtillery:  20,
		},
		{
			Type:                Swordsman,
			Name:                "Swordsman",
			FoodCost:            1,
			ResourceCosts:       Costs{Wood: 43, Stone: 20, Iron: 48},
			TrainingTimeSeconds: 1200, // 20:00
			RequiredTech:        "Swordsmith",
			SpeedMinutesField:   13.333333,
			TransportCapacity:   10,
			Category:            "Infantry",
			DefenseVsCavalry:    38,
			DefenseVsInfantry:   25,
			DefenseVsArtillery:  13,
		},
		{
			Type:                Archer,
			Name:                "Archer",
			FoodCost:            1,
			ResourceCosts:       Costs{Wood: 27, Stone: 12, Iron: 39},
			TrainingTimeSeconds: 900, // 15:00
			RequiredTech:        "Longbow",
			SpeedMinutesField:   8.333333,
			TransportCapacity:   16,
			Category:            "Artillery",
			DefenseVsCavalry:    10,
			DefenseVsInfantry:   32,
			DefenseVsArtillery:  15,
		},
		{
			Type:                Crossbowman,
			Name:                "Crossbowman",
			FoodCost:            1,
			ResourceCosts:       Costs{Wood: 50, Stone: 28, Iron: 55},
			TrainingTimeSeconds: 1350, // 22:30
			RequiredTech:        "Crossbow",
			SpeedMinutesField:   10.0,
			TransportCapacity:   13,
			Category:            "Artillery",
			DefenseVsCavalry:    33,
			DefenseVsInfantry:   91,
			DefenseVsArtillery:  60,
		},
		{
			Type:                Horseman,
			Name:                "Horseman",
			FoodCost:            2,
			ResourceCosts:       Costs{Wood: 25, Stone: 15, Iron: 45},
			TrainingTimeSeconds: 1050, // 17:30
			RequiredTech:        "",
			SpeedMinutesField:   5.0,
			TransportCapacity:   22,
			Category:            "Cavalry",
			DefenseVsCavalry:    37,
			DefenseVsInfantry:   27,
			DefenseVsArtillery:  60,
		},
		{
			Type:                Lancer,
			Name:                "Lancer",
			FoodCost:            2,
			ResourceCosts:       Costs{Wood: 70, Stone: 60, Iron: 80},
			TrainingTimeSeconds: 1860, // 31:00
			RequiredTech:        "Horse armour",
			SpeedMinutesField:   6.666667,
			TransportCapacity:   20,
			Category:            "Cavalry",
			DefenseVsCavalry:    16,
			DefenseVsInfantry:   13,
			DefenseVsArtillery:  25,
		},
	}
}

// GetUnitDefinition returns the definition for a unit type
func GetUnitDefinition(ut UnitType) *UnitDefinition {
	for _, def := range AllUnitDefinitions() {
		if def.Type == ut {
			return def
		}
	}
	return nil
}
