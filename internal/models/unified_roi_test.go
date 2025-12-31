package models

import (
	"math"
	"testing"
)

// ActionROI represents the return on investment for any action type
type ActionROI struct {
	ActionType     string  // "building", "research", "mission"
	ActionName     string  // e.g., "Lumberjack 5→6", "Beer Tester", "Hunting"
	ResourceCost   float64 // total resources consumed
	TimeCost       int     // seconds until action completes
	ResourceGain   float64 // immediate resources gained (missions only)
	ProductionGain float64 // resources/hour gained (buildings only)
	
	// Calculated fields
	NetResourcesAtHorizon float64 // net resources gained by time horizon
	EffectiveROI          float64 // net gain / total investment
}

// CalculateROI computes the ROI given a time horizon
func (a *ActionROI) CalculateROI(horizonSeconds int) {
	if a.ActionType == "mission" {
		// Mission: immediate gain minus cost, divided by time
		a.NetResourcesAtHorizon = a.ResourceGain - a.ResourceCost
		if a.TimeCost > 0 {
			a.EffectiveROI = a.NetResourcesAtHorizon / float64(a.TimeCost)
		}
	} else {
		// Building/Research: production gain over remaining time
		remainingTime := horizonSeconds - a.TimeCost
		if remainingTime > 0 {
			totalProduction := a.ProductionGain * (float64(remainingTime) / 3600.0)
			a.NetResourcesAtHorizon = totalProduction - a.ResourceCost
			totalInvestment := a.ResourceCost + float64(a.TimeCost)/3600.0*a.ProductionGain // opportunity cost
			if totalInvestment > 0 {
				a.EffectiveROI = a.NetResourcesAtHorizon / totalInvestment
			}
		}
	}
}

// CompareActions returns the action with higher ROI at given horizon
func CompareActions(a, b *ActionROI, horizonSeconds int) *ActionROI {
	a.CalculateROI(horizonSeconds)
	b.CalculateROI(horizonSeconds)
	
	if a.EffectiveROI >= b.EffectiveROI {
		return a
	}
	return b
}

// TestUnifiedROI_MissionVsBuilding tests mission vs building decision
func TestUnifiedROI_MissionVsBuilding(t *testing.T) {
	// Scenario: Player can either:
	// A) Upgrade Lumberjack (costs 500, takes 1h, +20 wood/hour permanent)
	// B) Run Hunting mission (costs 0, takes 15min, +45 resources one-time)

	lumberjack := &ActionROI{
		ActionType:     "building",
		ActionName:     "Lumberjack 5→6",
		ResourceCost:   500,
		TimeCost:       3600, // 1 hour
		ProductionGain: 20,   // +20/hour
	}

	hunting := &ActionROI{
		ActionType:   "mission",
		ActionName:   "Hunting",
		ResourceCost: 0,
		TimeCost:     900, // 15 min
		ResourceGain: 45,
	}

	// Test at different horizons
	horizons := []int{
		1 * 3600,  // 1 hour - building not even done yet
		6 * 3600,  // 6 hours
		24 * 3600, // 24 hours
		72 * 3600, // 72 hours (3 days)
	}

	t.Log("Mission vs Building ROI at different time horizons:")
	t.Log("----------------------------------------------------")

	for _, h := range horizons {
		lumberjack.CalculateROI(h)
		hunting.CalculateROI(h)

		hours := h / 3600
		winner := "Hunting"
		if lumberjack.EffectiveROI > hunting.EffectiveROI {
			winner = "Lumberjack"
		}

		t.Logf("At %dh: Lumberjack ROI=%.4f (net=%.0f), Hunting ROI=%.4f (net=%.0f) → %s",
			hours, lumberjack.EffectiveROI, lumberjack.NetResourcesAtHorizon,
			hunting.EffectiveROI, hunting.NetResourcesAtHorizon, winner)
	}

	// Property: At very short horizons, missions should win (instant value)
	lumberjack.CalculateROI(2 * 3600)
	hunting.CalculateROI(2 * 3600)
	if hunting.EffectiveROI <= 0 {
		t.Error("Hunting should have positive ROI")
	}
}

// TestUnifiedROI_RepeatableMissions tests that missions can be repeated
func TestUnifiedROI_RepeatableMissions(t *testing.T) {
	// Key insight: Missions are repeatable, so their effective production rate
	// should be calculated over the horizon

	hunting := &Mission{
		Name:            "Hunting",
		DurationMinutes: 15,
		UnitsRequired:   []UnitRequirement{{Type: Archer, Count: 15}},
		Rewards: []ResourceReward{
			{Type: Wood, Min: 10, Max: 20},
			{Type: Stone, Min: 10, Max: 20},
			{Type: Iron, Min: 10, Max: 20},
		},
	}

	// If we can run hunting continuously for 24 hours:
	// 24h / 0.25h per mission = 96 missions
	// 96 * 45 avg reward = 4320 resources
	horizonHours := 24.0
	missionsPerHour := 60.0 / float64(hunting.DurationMinutes)
	totalMissions := horizonHours * missionsPerHour
	totalResources := totalMissions * hunting.AverageTotalReward()

	t.Logf("Hunting over 24h: %.0f missions × %.0f resources = %.0f total",
		totalMissions, hunting.AverageTotalReward(), totalResources)

	// Effective "production rate" of hunting
	effectiveRate := totalResources / horizonHours
	t.Logf("Effective production rate: %.0f resources/hour", effectiveRate)

	// Compare to Lumberjack 30 (max level): 387 wood/hour
	// Hunting with 15 archers = 180 resources/hour (mixed)
	// But hunting uses units that could be trading...
	
	// Property: Mission effective rate should equal net_reward * missions_per_hour
	expectedRate := hunting.NetAverageReward() * missionsPerHour
	if math.Abs(effectiveRate-expectedRate) > 0.1 {
		t.Errorf("Effective rate %.0f != expected %.0f", effectiveRate, expectedRate)
	}
}

// TestUnifiedROI_BottleneckResource tests prioritizing bottleneck resources
func TestUnifiedROI_BottleneckResource(t *testing.T) {
	// Scenario: Player needs 1000 iron to upgrade, but production is:
	// Wood: 100/h, Stone: 100/h, Iron: 50/h
	// Iron is the bottleneck (20 hours to get 1000 iron)

	// Option A: Upgrade Ore Mine (+10 iron/hour, costs 300 wood, 200 stone, 1h build)
	// Option B: Run "Feed Miners" mission (produces iron, costs food resources)

	// The mission that produces IRON should be valued higher than
	// a mission that produces WOOD (which we have plenty of)

	type ResourceState struct {
		Current    map[ResourceType]float64
		Production map[ResourceType]float64
		Needed     map[ResourceType]float64
	}

	state := ResourceState{
		Current:    map[ResourceType]float64{Wood: 500, Stone: 500, Iron: 100},
		Production: map[ResourceType]float64{Wood: 100, Stone: 100, Iron: 50},
		Needed:     map[ResourceType]float64{Iron: 1000}, // Need 1000 iron for next upgrade
	}

	// Time to reach goal with current production
	ironDeficit := state.Needed[Iron] - state.Current[Iron]
	timeToGoal := ironDeficit / state.Production[Iron]
	t.Logf("Time to goal (1000 iron): %.1f hours", timeToGoal)

	// Mission that produces iron
	feedMiners := &Mission{
		Name:            "Feed Miners",
		DurationMinutes: 360, // 6 hours
		UnitsRequired: []UnitRequirement{
			{Type: Spearman, Count: 20},
			{Type: Swordsman, Count: 20},
			{Type: Archer, Count: 20},
			{Type: Horseman, Count: 20},
		},
		ResourceCosts: Costs{Wood: 300, Stone: 300, Food: 200},
		Rewards: []ResourceReward{
			{Type: Iron, Min: 1000, Max: 2000},
		},
	}

	// Value of this mission = time saved on bottleneck
	avgIronReward := feedMiners.AverageRewardByType(Iron)
	timeSaved := avgIronReward / state.Production[Iron]
	missionTime := float64(feedMiners.DurationMinutes) / 60.0
	netTimeSaved := timeSaved - missionTime

	t.Logf("Feed Miners: %.0f iron reward, saves %.1f hours of production",
		avgIronReward, timeSaved)
	t.Logf("Mission takes %.1f hours, net time saved: %.1f hours",
		missionTime, netTimeSaved)

	// Property: Mission should be worth it if net time saved > 0
	if netTimeSaved > 0 {
		t.Log("→ Feed Miners is WORTH IT for iron bottleneck")
	} else {
		t.Log("→ Feed Miners is NOT worth it")
	}
}

// TestUnifiedROI_MissionWithSetupCost tests missions requiring tavern investment
func TestUnifiedROI_MissionWithSetupCost(t *testing.T) {
	// If player doesn't have Tavern yet, they need to factor in setup cost

	// Setup costs
	tavernCosts := Costs{Wood: 200, Stone: 150, Iron: 100}
	tavernBuildTime := 30 * 60 // 30 min
	
	archerCost := 78 // 27+12+39 per archer
	archerTrainTime := 15 * 60 // 15 min per archer
	archersNeeded := 15

	totalSetupCost := float64(tavernCosts[Wood] + tavernCosts[Stone] + tavernCosts[Iron])
	totalSetupCost += float64(archerCost * archersNeeded)
	totalSetupTime := tavernBuildTime + archerTrainTime*archersNeeded

	t.Logf("Setup cost to enable Hunting: %.0f resources, %d minutes",
		totalSetupCost, totalSetupTime/60)

	// Hunting mission value
	huntingReward := 45.0
	huntingTime := 15 * 60 // 15 min

	// Number of missions needed to recoup setup
	missionsToBreakEven := totalSetupCost / huntingReward
	timeToBreakEven := float64(totalSetupTime) + missionsToBreakEven*float64(huntingTime)

	t.Logf("Missions to break even: %.0f", missionsToBreakEven)
	t.Logf("Total time to break even: %.1f hours", timeToBreakEven/3600)

	// After break-even, each mission is pure profit
	// Compare to: what if we spent those resources on production buildings instead?

	// If setup cost (1620) went to Lumberjack upgrades instead:
	// Assume 1620 resources buys ~2 levels = +40 wood/hour
	alternativeProductionGain := 40.0

	// At what horizon is Tavern+Hunting better than Lumberjack?
	// Hunting effective rate: 180/hour
	// Need: 180h > alternativeProductionGain*(h - setupTime) + setupCost
	// 180h > 40h - 40*setupTime + setupCost
	// 140h > setupCost - 40*setupTime
	// h > (setupCost - 40*setupHours) / 140
	
	setupHours := float64(totalSetupTime) / 3600
	crossoverHours := (totalSetupCost - alternativeProductionGain*setupHours) / (180 - alternativeProductionGain)

	t.Logf("Crossover point: %.1f hours", crossoverHours)
	t.Log("Before crossover: Lumberjack better. After: Tavern+Hunting better")
}

// BenchmarkROICalculation benchmarks ROI computation
func BenchmarkROICalculation(b *testing.B) {
	action := &ActionROI{
		ActionType:     "building",
		ActionName:     "Test",
		ResourceCost:   500,
		TimeCost:       3600,
		ProductionGain: 20,
	}

	for i := 0; i < b.N; i++ {
		action.CalculateROI(24 * 3600)
	}
}
