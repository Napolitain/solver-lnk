package units

import "github.com/napolitain/solver-lnk/internal/models"

// Unit represents a unit type (army or trading)
type Unit struct {
	Name                string
	FoodCost            int
	ResourceCosts       models.Costs // Wood, Stone, Iron costs per unit
	SpeedMinutesField   float64      // minutes per field
	TransportCapacity   int
	UnitType            string // Infantry, Cavalry, Artillery, Transport
	TrainingTimeSeconds int    // seconds to train one unit

	// Defense stats (only for combat units)
	DefenseVsCavalry   int
	DefenseVsInfantry  int
	DefenseVsArtillery int
}

// ThroughputPerHour calculates resources this unit can transport per hour
// given a round trip distance in fields
func (u *Unit) ThroughputPerHour(roundTripFields int) float64 {
	if u.TransportCapacity == 0 || u.SpeedMinutesField == 0 {
		return 0
	}
	tripTimeMinutes := float64(roundTripFields) * u.SpeedMinutesField
	tripsPerHour := 60.0 / tripTimeMinutes
	return float64(u.TransportCapacity) * tripsPerHour
}

// TotalDefense returns sum of all defense stats
func (u *Unit) TotalDefense() int {
	return u.DefenseVsCavalry + u.DefenseVsInfantry + u.DefenseVsArtillery
}

// MinDefense returns the minimum defense stat (for balanced optimization)
func (u *Unit) MinDefense() int {
	min := u.DefenseVsCavalry
	if u.DefenseVsInfantry < min {
		min = u.DefenseVsInfantry
	}
	if u.DefenseVsArtillery < min {
		min = u.DefenseVsArtillery
	}
	return min
}

// DefenseEfficiencyPerFood returns defense per food cost
func (u *Unit) DefenseEfficiencyPerFood() float64 {
	if u.FoodCost == 0 {
		return 0
	}
	return float64(u.TotalDefense()) / float64(u.FoodCost)
}

// AllUnits returns all available units (hardcoded from data files)
// Costs from data files: Wood, Stone, Iron, Food
func AllUnits() []*Unit {
	return []*Unit{
		{
			// From data/units/spearman: Costs 18, 6, 30, 1
			Name:                "Spearman",
			FoodCost:            1,
			ResourceCosts:       models.Costs{models.Wood: 18, models.Stone: 6, models.Iron: 30},
			SpeedMinutesField:   11.666667, // 11 min 40 sec
			TransportCapacity:   12,
			UnitType:            "Infantry",
			TrainingTimeSeconds: 750, // 12:30
			DefenseVsCavalry:    59,
			DefenseVsInfantry:   32,
			DefenseVsArtillery:  20,
		},
		{
			// From data/units/swordsman: Costs 43, 20, 48, 1
			Name:                "Swordsman",
			FoodCost:            1,
			ResourceCosts:       models.Costs{models.Wood: 43, models.Stone: 20, models.Iron: 48},
			SpeedMinutesField:   13.333333, // 13 min 20 sec
			TransportCapacity:   10,
			UnitType:            "Infantry",
			TrainingTimeSeconds: 1200, // 20:00
			DefenseVsCavalry:    38,
			DefenseVsInfantry:   25,
			DefenseVsArtillery:  13,
		},
		{
			// From data/units/archer: Costs 27, 12, 39, 1
			Name:                "Archer",
			FoodCost:            1,
			ResourceCosts:       models.Costs{models.Wood: 27, models.Stone: 12, models.Iron: 39},
			SpeedMinutesField:   8.333333, // 8 min 20 sec
			TransportCapacity:   16,
			UnitType:            "Artillery",
			TrainingTimeSeconds: 900, // 15:00
			DefenseVsCavalry:    10,
			DefenseVsInfantry:   32,
			DefenseVsArtillery:  15,
		},
		{
			// From data/units/crossbowman: Costs 50, 28, 55, 1
			Name:                "Crossbowman",
			FoodCost:            1,
			ResourceCosts:       models.Costs{models.Wood: 50, models.Stone: 28, models.Iron: 55},
			SpeedMinutesField:   10.0,
			TransportCapacity:   13,
			UnitType:            "Artillery",
			TrainingTimeSeconds: 1350, // 22:30
			DefenseVsCavalry:    33,
			DefenseVsInfantry:   91,
			DefenseVsArtillery:  60,
		},
		{
			// From data/units/horseman: Costs 25, 15, 45, 2
			Name:                "Horseman",
			FoodCost:            2,
			ResourceCosts:       models.Costs{models.Wood: 25, models.Stone: 15, models.Iron: 45},
			SpeedMinutesField:   5.0,
			TransportCapacity:   22,
			UnitType:            "Cavalry",
			TrainingTimeSeconds: 1050, // 17:30
			DefenseVsCavalry:    37,
			DefenseVsInfantry:   27,
			DefenseVsArtillery:  60,
		},
		{
			// From data/units/lancer: Costs 70, 60, 80, 2
			Name:                "Lancer",
			FoodCost:            2,
			ResourceCosts:       models.Costs{models.Wood: 70, models.Stone: 60, models.Iron: 80},
			SpeedMinutesField:   6.666667, // 6 min 40 sec
			TransportCapacity:   20,
			UnitType:            "Cavalry",
			TrainingTimeSeconds: 1860, // 31:00
			DefenseVsCavalry:    16,
			DefenseVsInfantry:   13,
			DefenseVsArtillery:  25,
		},
		{
			// From data/units/handcart: Costs 45, 25, 30, 1
			Name:                "Handcart",
			FoodCost:            1,
			ResourceCosts:       models.Costs{models.Wood: 45, models.Stone: 25, models.Iron: 30},
			SpeedMinutesField:   13.333333, // 13 min 20 sec
			TransportCapacity:   500,
			UnitType:            "Transport",
			TrainingTimeSeconds: 600, // 10:00
		},
		{
			// From data/units/oxcart: Costs 95, 40, 65, 3
			Name:                "Oxcart",
			FoodCost:            3,
			ResourceCosts:       models.Costs{models.Wood: 95, models.Stone: 40, models.Iron: 65},
			SpeedMinutesField:   16.666667, // 16 min 40 sec
			TransportCapacity:   2500,
			UnitType:            "Transport",
			TrainingTimeSeconds: 1200, // 20:00
		},
	}
}

// CombatUnits returns only units that can fight
func CombatUnits() []*Unit {
	var combat []*Unit
	for _, u := range AllUnits() {
		if u.UnitType != "Transport" {
			combat = append(combat, u)
		}
	}
	return combat
}

// TransportUnits returns only transport units
func TransportUnits() []*Unit {
	var transport []*Unit
	for _, u := range AllUnits() {
		if u.UnitType == "Transport" {
			transport = append(transport, u)
		}
	}
	return transport
}
