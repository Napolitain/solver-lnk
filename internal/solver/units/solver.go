package units

import (
	"sort"
)

// Constants for maxed castle
const (
	MaxFoodCapacity           = 4265  // Remaining after buildings (5000 - 735)
	ResourceProductionPerHour = 1161  // 387 + 387 + 387 (LJ30 + Q30 + OM30)
	MarketDistanceFields      = 25    // Keep level 10
	RoundTripFields           = 50    // 2 × 25
	SilverPerResource         = 0.02  // 1:50 exchange rate
)

// Solution represents an army composition
type Solution struct {
	UnitCounts        map[string]int
	TotalFood         int
	TotalThroughput   float64 // resources/hour
	DefenseVsCavalry  int
	DefenseVsInfantry int
	DefenseVsArtillery int
	SilverPerHour     float64
}

// MinDefense returns the minimum defense across all types
func (s *Solution) MinDefense() int {
	min := s.DefenseVsCavalry
	if s.DefenseVsInfantry < min {
		min = s.DefenseVsInfantry
	}
	if s.DefenseVsArtillery < min {
		min = s.DefenseVsArtillery
	}
	return min
}

// Solver finds optimal army composition
type Solver struct {
	Units              []*Unit
	FoodCapacity       int
	RequiredThroughput float64
	RoundTripFields    int
}

// NewSolver creates a new units solver with default constants
func NewSolver() *Solver {
	return &Solver{
		Units:              AllUnits(),
		FoodCapacity:       MaxFoodCapacity,
		RequiredThroughput: ResourceProductionPerHour,
		RoundTripFields:    RoundTripFields,
	}
}

// NewSolverWithConfig creates a solver from protobuf config
func NewSolverWithConfig(food, resourceProd, marketDist int32) *Solver {
	s := NewSolver()

	if food > 0 {
		s.FoodCapacity = int(food)
	}
	if resourceProd > 0 {
		s.RequiredThroughput = float64(resourceProd)
	}
	if marketDist > 0 {
		s.RoundTripFields = int(marketDist) * 2
	}

	return s
}

// Solve finds the optimal army composition
// Strategy: 
// 1. Calculate minimum trading capacity needed
// 2. Use remaining food for combat units optimized for balanced defense
func (s *Solver) Solve() *Solution {
	solution := &Solution{
		UnitCounts: make(map[string]int),
	}

	// First, check if combat units alone can handle trading
	combatThroughput := s.calculateCombatOnlyThroughput()
	
	var tradingFoodNeeded int
	if combatThroughput < s.RequiredThroughput {
		// Need dedicated trading units
		tradingFoodNeeded = s.calculateMinTradingFood()
	}

	remainingFood := s.FoodCapacity - tradingFoodNeeded

	// Allocate combat units with remaining food
	s.allocateCombatUnits(solution, remainingFood)

	// Check if combat throughput is enough, otherwise add trading units
	if solution.TotalThroughput < s.RequiredThroughput {
		s.addTradingUnits(solution)
	}

	// Calculate silver per hour
	solution.SilverPerHour = float64(ResourceProductionPerHour) * SilverPerResource

	return solution
}

// calculateCombatOnlyThroughput calculates max throughput if we use all food for combat
func (s *Solver) calculateCombatOnlyThroughput() float64 {
	// Find best combat unit for throughput per food
	var bestThroughputPerFood float64
	for _, u := range CombatUnits() {
		throughput := u.ThroughputPerHour(s.RoundTripFields)
		perFood := throughput / float64(u.FoodCost)
		if perFood > bestThroughputPerFood {
			bestThroughputPerFood = perFood
		}
	}
	return bestThroughputPerFood * float64(s.FoodCapacity)
}

// calculateMinTradingFood finds minimum food needed for trading units
func (s *Solver) calculateMinTradingFood() int {
	// Find most efficient trading unit (throughput per food)
	var bestUnit *Unit
	var bestEfficiency float64

	for _, u := range TransportUnits() {
		throughput := u.ThroughputPerHour(s.RoundTripFields)
		efficiency := throughput / float64(u.FoodCost)
		if efficiency > bestEfficiency {
			bestEfficiency = efficiency
			bestUnit = u
		}
	}

	if bestUnit == nil {
		return 0
	}

	// Calculate how many needed
	throughputPerUnit := bestUnit.ThroughputPerHour(s.RoundTripFields)
	unitsNeeded := int(s.RequiredThroughput/throughputPerUnit) + 1
	return unitsNeeded * bestUnit.FoodCost
}

// allocateCombatUnits allocates combat units for balanced defense
func (s *Solver) allocateCombatUnits(solution *Solution, foodBudget int) {
	combat := CombatUnits()

	// Sort by defense efficiency (total defense per food)
	sort.Slice(combat, func(i, j int) bool {
		return combat[i].DefenseEfficiencyPerFood() > combat[j].DefenseEfficiencyPerFood()
	})

	// Greedy allocation aiming for balance
	// Track current defense totals
	defCav, defInf, defArt := 0, 0, 0
	usedFood := 0

	for usedFood < foodBudget {
		// Find which defense type is weakest
		minDef := min(defCav, defInf, defArt)
		
		// Find best unit to improve the weakest defense
		var bestUnit *Unit
		var bestImprovement int

		for _, u := range combat {
			if usedFood+u.FoodCost > foodBudget {
				continue
			}

			// Calculate improvement to minimum defense
			newCav := defCav + u.DefenseVsCavalry
			newInf := defInf + u.DefenseVsInfantry
			newArt := defArt + u.DefenseVsArtillery
			newMin := min(newCav, newInf, newArt)
			improvement := newMin - minDef

			if bestUnit == nil || improvement > bestImprovement {
				bestImprovement = improvement
				bestUnit = u
			}
		}

		if bestUnit == nil {
			break
		}

		// Add unit
		solution.UnitCounts[bestUnit.Name]++
		usedFood += bestUnit.FoodCost
		defCav += bestUnit.DefenseVsCavalry
		defInf += bestUnit.DefenseVsInfantry
		defArt += bestUnit.DefenseVsArtillery
	}

	solution.TotalFood = usedFood
	solution.DefenseVsCavalry = defCav
	solution.DefenseVsInfantry = defInf
	solution.DefenseVsArtillery = defArt

	// Calculate throughput from combat units
	for _, u := range combat {
		count := solution.UnitCounts[u.Name]
		solution.TotalThroughput += float64(count) * u.ThroughputPerHour(s.RoundTripFields)
	}
}

// addTradingUnits adds minimum trading units to meet throughput requirement
func (s *Solver) addTradingUnits(solution *Solution) {
	// Find most efficient trading unit
	var bestUnit *Unit
	var bestEfficiency float64

	for _, u := range TransportUnits() {
		throughput := u.ThroughputPerHour(s.RoundTripFields)
		efficiency := throughput / float64(u.FoodCost)
		if efficiency > bestEfficiency {
			bestEfficiency = efficiency
			bestUnit = u
		}
	}

	if bestUnit == nil {
		return
	}

	// Add units until throughput is sufficient
	for solution.TotalThroughput < s.RequiredThroughput && solution.TotalFood < s.FoodCapacity {
		// Need to remove a combat unit to make room
		if solution.TotalFood+bestUnit.FoodCost > s.FoodCapacity {
			// Remove least efficient combat unit
			s.removeLeastEfficientCombat(solution, bestUnit.FoodCost)
		}

		if solution.TotalFood+bestUnit.FoodCost <= s.FoodCapacity {
			solution.UnitCounts[bestUnit.Name]++
			solution.TotalFood += bestUnit.FoodCost
			solution.TotalThroughput += bestUnit.ThroughputPerHour(s.RoundTripFields)
		} else {
			break
		}
	}
}

// removeLeastEfficientCombat removes combat units to free up food
func (s *Solver) removeLeastEfficientCombat(solution *Solution, foodNeeded int) {
	combat := CombatUnits()
	
	// Sort by efficiency (worst first)
	sort.Slice(combat, func(i, j int) bool {
		return combat[i].DefenseEfficiencyPerFood() < combat[j].DefenseEfficiencyPerFood()
	})

	freedFood := 0
	for _, u := range combat {
		for solution.UnitCounts[u.Name] > 0 && freedFood < foodNeeded {
			solution.UnitCounts[u.Name]--
			solution.TotalFood -= u.FoodCost
			solution.DefenseVsCavalry -= u.DefenseVsCavalry
			solution.DefenseVsInfantry -= u.DefenseVsInfantry
			solution.DefenseVsArtillery -= u.DefenseVsArtillery
			solution.TotalThroughput -= u.ThroughputPerHour(s.RoundTripFields)
			freedFood += u.FoodCost
		}
		if freedFood >= foodNeeded {
			break
		}
	}
}


