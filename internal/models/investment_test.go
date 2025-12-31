package models

import (
	"testing"
)

// InvestmentPath represents an investment strategy
type InvestmentPath struct {
	Name               string
	TotalInvestmentTime int    // seconds to reach full productivity
	TotalResourceCost   Costs
	ProductionRatePerHour float64 // resources/hour once fully invested
}

// TimeToPayoff calculates how long after investment completes to recoup costs
func (p *InvestmentPath) TimeToPayoff() float64 {
	if p.ProductionRatePerHour <= 0 {
		return -1 // Never pays off
	}
	totalCost := 0.0
	for _, cost := range p.TotalResourceCost {
		totalCost += float64(cost)
	}
	hoursToPayoff := totalCost / p.ProductionRatePerHour
	return hoursToPayoff * 3600 // return in seconds
}

// EffectiveProductionAtTime calculates cumulative production at a given time
// Assumes investment starts at t=0
func (p *InvestmentPath) EffectiveProductionAtTime(timeSeconds int) float64 {
	if timeSeconds <= p.TotalInvestmentTime {
		return 0 // Still investing
	}
	productiveTime := float64(timeSeconds - p.TotalInvestmentTime)
	return (productiveTime / 3600.0) * p.ProductionRatePerHour
}

// TestInvestmentComparison_TavernVsLumberjack tests the core algorithm question:
// When is investing in Tavern+Units better than upgrading Lumberjack?
func TestInvestmentComparison_TavernVsLumberjack(t *testing.T) {
	// Scenario: Player has Lumberjack 10, Tavern 0
	// Option A: Upgrade Lumberjack 10→15 (direct production boost)
	// Option B: Build Tavern 1→5 + train 15 archers + run Hunting missions

	// Example costs (simplified, would come from data files)
	lumberjackPath := InvestmentPath{
		Name:               "Lumberjack 10→15",
		TotalInvestmentTime: 5 * 3600, // ~5 hours of build time
		TotalResourceCost:   Costs{Wood: 5000, Stone: 3000, Iron: 2000},
		ProductionRatePerHour: 50, // +50 wood/hour from level 10→15
	}

	// Tavern path: build tavern + train units
	// Hunting: 15 archers, 15 min, ~45 resources avg
	// If running continuously: 45 * 4 = 180 resources/hour
	// But need tavern (build time) + archer training time
	tavernPath := InvestmentPath{
		Name:               "Tavern 5 + 15 Archers (Hunting)",
		TotalInvestmentTime: 3 * 3600, // ~3 hours (tavern build + archer training)
		TotalResourceCost:   Costs{Wood: 2000, Stone: 1500, Iron: 1500}, // tavern + 15 archers
		ProductionRatePerHour: 180, // Hunting continuously
	}

	// Test: Which path produces more resources over various time horizons?
	testTimes := []int{
		6 * 3600,  // 6 hours
		12 * 3600, // 12 hours
		24 * 3600, // 24 hours
		48 * 3600, // 48 hours
	}

	t.Log("Investment Comparison: Lumberjack vs Tavern+Hunting")
	t.Log("------------------------------------------------------")

	for _, totalTime := range testTimes {
		lumberProd := lumberjackPath.EffectiveProductionAtTime(totalTime)
		tavernProd := tavernPath.EffectiveProductionAtTime(totalTime)

		hours := totalTime / 3600
		t.Logf("At %dh: Lumberjack=%.0f, Tavern=%.0f (diff=%.0f)",
			hours, lumberProd, tavernProd, tavernProd-lumberProd)
	}

	// Property: Tavern should be better at shorter horizons (faster investment)
	// Property: Lumberjack is permanent while tavern requires active play
	// (but for now we assume active play)
	
	// At 6 hours: tavern has 3 hours of production, lumberjack has 1 hour
	at6h := 6 * 3600
	if tavernPath.EffectiveProductionAtTime(at6h) <= lumberjackPath.EffectiveProductionAtTime(at6h) {
		t.Log("Note: Tavern path should be faster at 6h horizon")
	}
}

// TestInvestmentComparison_BreakEvenAnalysis finds when two paths are equivalent
func TestInvestmentComparison_BreakEvenAnalysis(t *testing.T) {
	// Slower investment, higher long-term rate
	slowPath := InvestmentPath{
		Name:               "Slow (high long-term)",
		TotalInvestmentTime: 8 * 3600,
		TotalResourceCost:   Costs{Wood: 10000},
		ProductionRatePerHour: 200,
	}

	// Faster investment, lower rate
	fastPath := InvestmentPath{
		Name:               "Fast (lower long-term)",
		TotalInvestmentTime: 2 * 3600,
		TotalResourceCost:   Costs{Wood: 2000},
		ProductionRatePerHour: 100,
	}

	// Find break-even point (binary search)
	// Before break-even: fast is better
	// After break-even: slow is better
	low := 0
	high := 100 * 3600 // 100 hours max
	
	for high-low > 60 { // precision: 1 minute
		mid := (low + high) / 2
		slowProd := slowPath.EffectiveProductionAtTime(mid)
		fastProd := fastPath.EffectiveProductionAtTime(mid)
		
		if slowProd > fastProd {
			high = mid
		} else {
			low = mid
		}
	}

	breakEvenHours := float64(low) / 3600.0
	t.Logf("Break-even point: %.1f hours", breakEvenHours)
	t.Logf("Before %.1fh: Fast path is better", breakEvenHours)
	t.Logf("After %.1fh: Slow path is better", breakEvenHours)

	// Verify the break-even makes sense
	// Fast path has 6 hour head start (8-2), produces at 100/h
	// Slow path produces at 200/h (100 more per hour)
	// Head start: 6h * 100 = 600 resources
	// Catch-up rate: 100 resources/hour
	// Time to catch up: 600/100 = 6 hours after slow finishes = 14 hours total
	expectedBreakEven := 14.0
	if absFloat(breakEvenHours - expectedBreakEven) > 1.0 {
		t.Errorf("Break-even at %.1fh, expected ~%.1fh", breakEvenHours, expectedBreakEven)
	}
}

// TestMissionROI_VsProductionBuilding compares mission value to building upgrades
func TestMissionROI_VsProductionBuilding(t *testing.T) {
	// Key insight: Missions are "active" income vs "passive" building production
	// The question: Given X resources and Y time, which investment yields more?

	// Scenario: 1000 resources to invest, 24 hour horizon
	horizonHours := 24.0

	// Option A: Spend on Lumberjack upgrade
	// Assume upgrade costs 1000, takes 2 hours, adds 20 wood/hour
	lumberjackBuildTime := 2.0 // hours
	lumberjackProduction := 20.0 // wood/hour
	lumberjackYield := (horizonHours - lumberjackBuildTime) * lumberjackProduction

	// Option B: Spend on Tavern + Units, run missions
	// Assume: 500 for tavern upgrade (1h), 500 for 20 archers
	// Can run Hunting: 15 archers, 15 min, ~45 resources
	// Effective production: 45 * 4 = 180/hour for hunting group
	tavernBuildTime := 1.0 // hours
	missionProductionRate := 180.0 // resources/hour (active)
	tavernYield := (horizonHours - tavernBuildTime) * missionProductionRate

	t.Logf("24h comparison with 1000 resource budget:")
	t.Logf("  Lumberjack: %.0f resources (passive)", lumberjackYield)
	t.Logf("  Tavern+Hunting: %.0f resources (active)", tavernYield)

	// The huge difference is because missions are much higher throughput
	// BUT: missions require active play, units are busy, etc.
	
	// Real comparison should factor in:
	// 1. Units have alternative uses (trading, defense)
	// 2. Active play requirement
	// 3. Mission availability (can't always run preferred mission?)
	// 4. Building upgrades are permanent, don't need ongoing attention
}

// TestMission_OpportunityCost_Trading tests the trade-off between missions and trading
func TestMission_OpportunityCost_Trading(t *testing.T) {
	// When units do missions, they can't trade
	// Trading generates silver (which has resource value)

	// 15 Archers trading:
	// - Transport: 16 resources/trip
	// - Speed: 8.33 min/field, 50 field round trip = ~417 min = ~7 hours per trip
	// - Wait, that's way too slow. Let's recalculate.
	// Actually with 25 field distance (market), round trip = 50 fields
	// 8.33 min/field * 50 = 416 min ≈ 7 hours per round trip
	// 15 archers * 16 capacity / 7 hours = 34 resources/hour throughput
	
	archerThroughputPerHour := 15 * 16.0 / 7.0 // ~34 resources/hour
	silverPerResource := 0.02
	silverIncomePerHour := archerThroughputPerHour * silverPerResource // ~0.68 silver/hour

	// Same 15 archers on Hunting mission:
	// 15 min mission, ~45 resources average
	// 4 missions per hour = 180 resources/hour

	huntingIncomePerHour := 180.0

	t.Logf("15 Archers opportunity cost comparison:")
	t.Logf("  Trading: %.1f resources/hour + %.2f silver/hour", archerThroughputPerHour, silverIncomePerHour)
	t.Logf("  Hunting: %.1f resources/hour", huntingIncomePerHour)
	t.Logf("  Hunting advantage: %.1fx more resources", huntingIncomePerHour/archerThroughputPerHour)

	// Clear winner: Hunting is much better for resources
	// But trading provides silver which has separate value
	// And trading keeps units available for defense
}

// TestMission_InvestmentBreakdown calculates total investment needed
func TestMission_InvestmentBreakdown(t *testing.T) {
	// To enable Hunting missions, player needs:
	// 1. Tavern level 1 (minimum)
	// 2. 15 Archers
	
	// Tavern 1 costs (example - would come from data)
	tavernCost := Costs{Wood: 200, Stone: 150, Iron: 100}
	tavernBuildTime := 30 * 60 // 30 minutes
	
	// 15 Archers: each costs Wood: 27, Stone: 12, Iron: 39, training: 15 min
	archerCost := Costs{Wood: 27, Stone: 12, Iron: 39}
	archerTrainTime := 15 * 60 // 15 minutes each
	numArchers := 15
	
	totalArcherCost := Costs{
		Wood:  archerCost[Wood] * numArchers,
		Stone: archerCost[Stone] * numArchers,
		Iron:  archerCost[Iron] * numArchers,
	}
	
	// Training can be parallel in arsenal, so time = 15 min * 15 = 225 min (sequential)
	// Or if arsenal has queue, could be longer
	totalTrainTime := archerTrainTime * numArchers
	
	// Total investment
	totalCost := Costs{
		Wood:  tavernCost[Wood] + totalArcherCost[Wood],
		Stone: tavernCost[Stone] + totalArcherCost[Stone],
		Iron:  tavernCost[Iron] + totalArcherCost[Iron],
	}
	
	// Time: tavern build + archer training (sequential or parallel?)
	// Assume sequential for now
	totalTime := tavernBuildTime + totalTrainTime
	
	t.Log("Investment to enable Hunting missions:")
	t.Logf("  Tavern 1: Wood=%d, Stone=%d, Iron=%d (%.0f min build)",
		tavernCost[Wood], tavernCost[Stone], tavernCost[Iron], float64(tavernBuildTime)/60)
	t.Logf("  15 Archers: Wood=%d, Stone=%d, Iron=%d (%.0f min train)",
		totalArcherCost[Wood], totalArcherCost[Stone], totalArcherCost[Iron], float64(totalTrainTime)/60)
	t.Logf("  TOTAL: Wood=%d, Stone=%d, Iron=%d (%.1f hours)",
		totalCost[Wood], totalCost[Stone], totalCost[Iron], float64(totalTime)/3600)
	
	// Payoff calculation
	// Hunting: ~180 resources/hour
	huntingRate := 180.0
	totalInvestmentResources := float64(totalCost[Wood] + totalCost[Stone] + totalCost[Iron])
	payoffHours := totalInvestmentResources / huntingRate
	
	t.Logf("  Payoff time: %.1f hours of continuous hunting", payoffHours)
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
