package models

// Army represents all units with strict typing (no maps)
type Army struct {
	Spearman    int
	Swordsman   int
	Archer      int
	Crossbowman int
	Horseman    int
	Lancer      int
	Handcart    int
	Oxcart      int
}

// Get returns count for a unit type
func (a *Army) Get(ut UnitType) int {
	switch ut {
	case Spearman:
		return a.Spearman
	case Swordsman:
		return a.Swordsman
	case Archer:
		return a.Archer
	case Crossbowman:
		return a.Crossbowman
	case Horseman:
		return a.Horseman
	case Lancer:
		return a.Lancer
	}
	return 0
}

// Set sets count for a unit type
func (a *Army) Set(ut UnitType, count int) {
	switch ut {
	case Spearman:
		a.Spearman = count
	case Swordsman:
		a.Swordsman = count
	case Archer:
		a.Archer = count
	case Crossbowman:
		a.Crossbowman = count
	case Horseman:
		a.Horseman = count
	case Lancer:
		a.Lancer = count
	}
}

// Add adds units of a type
func (a *Army) Add(ut UnitType, count int) {
	switch ut {
	case Spearman:
		a.Spearman += count
	case Swordsman:
		a.Swordsman += count
	case Archer:
		a.Archer += count
	case Crossbowman:
		a.Crossbowman += count
	case Horseman:
		a.Horseman += count
	case Lancer:
		a.Lancer += count
	}
}

// Remove removes units of a type (floors at 0)
func (a *Army) Remove(ut UnitType, count int) {
	current := a.Get(ut)
	newCount := current - count
	if newCount < 0 {
		newCount = 0
	}
	a.Set(ut, newCount)
}

// CanSatisfy checks if army has enough units for requirements
func (a *Army) CanSatisfy(reqs []UnitRequirement) bool {
	for _, req := range reqs {
		if a.Get(req.Type) < req.Count {
			return false
		}
	}
	return true
}

// Subtract removes units according to requirements (for sending on mission)
func (a *Army) Subtract(reqs []UnitRequirement) {
	for _, req := range reqs {
		a.Remove(req.Type, req.Count)
	}
}

// AddFrom adds units from requirements (for returning from mission)
func (a *Army) AddFrom(reqs []UnitRequirement) {
	for _, req := range reqs {
		a.Add(req.Type, req.Count)
	}
}

// TotalUnits returns total count of all units
func (a *Army) TotalUnits() int {
	return a.Spearman + a.Swordsman + a.Archer + a.Crossbowman + a.Horseman + a.Lancer + a.Handcart + a.Oxcart
}

// TotalFood returns total food consumption of army
// Spearman, Swordsman, Archer, Crossbowman, Handcart = 1 food each
// Horseman, Lancer, Oxcart = 2-3 food each
func (a *Army) TotalFood() int {
	return a.Spearman + a.Swordsman + a.Archer + a.Crossbowman + a.Handcart +
		2*a.Horseman + 2*a.Lancer + 3*a.Oxcart
}

// Clone returns a copy of the army
func (a *Army) Clone() Army {
	return Army{
		Spearman:    a.Spearman,
		Swordsman:   a.Swordsman,
		Archer:      a.Archer,
		Crossbowman: a.Crossbowman,
		Horseman:    a.Horseman,
		Lancer:      a.Lancer,
		Handcart:    a.Handcart,
		Oxcart:      a.Oxcart,
	}
}

// IsEmpty returns true if army has no units
func (a *Army) IsEmpty() bool {
	return a.TotalUnits() == 0
}
