package units

// Unit represents a unit type (army or trading)
type Unit struct {
	Name              string
	FoodCost          int
	SpeedMinutesField float64 // minutes per field
	TransportCapacity int
	UnitType          string // Infantry, Cavalry, Artillery, Transport

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
func AllUnits() []*Unit {
	return []*Unit{
		{
			Name:               "spearman",
			FoodCost:           1,
			SpeedMinutesField:  11.666667, // 11 min 40 sec
			TransportCapacity:  12,
			UnitType:           "Infantry",
			DefenseVsCavalry:   59,
			DefenseVsInfantry:  32,
			DefenseVsArtillery: 20,
		},
		{
			Name:               "swordsman",
			FoodCost:           1,
			SpeedMinutesField:  13.333333, // 13 min 20 sec
			TransportCapacity:  10,
			UnitType:           "Infantry",
			DefenseVsCavalry:   38,
			DefenseVsInfantry:  25,
			DefenseVsArtillery: 13,
		},
		{
			Name:               "archer",
			FoodCost:           1,
			SpeedMinutesField:  8.333333, // 8 min 20 sec
			TransportCapacity:  16,
			UnitType:           "Artillery",
			DefenseVsCavalry:   10,
			DefenseVsInfantry:  32,
			DefenseVsArtillery: 15,
		},
		{
			Name:               "crossbowman",
			FoodCost:           1,
			SpeedMinutesField:  10.0,
			TransportCapacity:  13,
			UnitType:           "Artillery",
			DefenseVsCavalry:   33,
			DefenseVsInfantry:  91,
			DefenseVsArtillery: 60,
		},
		{
			Name:               "horseman",
			FoodCost:           2,
			SpeedMinutesField:  5.0,
			TransportCapacity:  22,
			UnitType:           "Cavalry",
			DefenseVsCavalry:   37,
			DefenseVsInfantry:  27,
			DefenseVsArtillery: 60,
		},
		{
			Name:               "lancer",
			FoodCost:           2,
			SpeedMinutesField:  6.666667, // 6 min 40 sec
			TransportCapacity:  20,
			UnitType:           "Cavalry",
			DefenseVsCavalry:   16,
			DefenseVsInfantry:  13,
			DefenseVsArtillery: 25,
		},
		{
			Name:              "handcart",
			FoodCost:          1,
			SpeedMinutesField: 13.333333, // 13 min 20 sec
			TransportCapacity: 500,
			UnitType:          "Transport",
		},
		{
			Name:              "oxcart",
			FoodCost:          3,
			SpeedMinutesField: 16.666667, // 16 min 40 sec
			TransportCapacity: 2500,
			UnitType:          "Transport",
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
