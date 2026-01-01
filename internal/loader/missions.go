package loader

import (
	"github.com/napolitain/solver-lnk/internal/models"
)

// LoadMissions returns all available missions with their requirements
// TODO: Load from data files once format is standardized to JSON
// Current data is based on game observations - unit types are inferred from mission themes
func LoadMissions() []*models.Mission {
	return []*models.Mission{
		// Tavern Level 1 missions
		{
			Name:            "Overtime Wood",
			Description:     "An extra shift in the lumberjack hut",
			DurationMinutes: 5,
			TavernLevel:     1,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 5}, // Infantry for labor
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 5, Max: 10},
			},
		},
		{
			Name:            "Overtime Stone",
			Description:     "An extra shift in the quarry",
			DurationMinutes: 5,
			TavernLevel:     1,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 5},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Stone, Min: 5, Max: 10},
			},
		},
		{
			Name:            "Overtime Ore",
			Description:     "An extra shift in the ore mine",
			DurationMinutes: 5,
			TavernLevel:     1,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 5},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Iron, Min: 5, Max: 10},
			},
		},
		{
			Name:            "Hunting",
			Description:     "Go hunting with your followers through your estates",
			DurationMinutes: 15,
			TavernLevel:     1,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Archer, Count: 15}, // Archers for hunting
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 10, Max: 20},
				{Type: models.Stone, Min: 10, Max: 20},
				{Type: models.Iron, Min: 10, Max: 20},
			},
		},
		// Tavern Level 2 missions
		{
			Name:            "Chop Wood",
			Description:     "Special assignment to chop wood",
			DurationMinutes: 30,
			TavernLevel:     2,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 20},
				{Type: models.Swordsman, Count: 20},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 50, Max: 100},
			},
		},
		{
			Name:            "Help Stone Cutters",
			Description:     "Help stone cutters in the quarry",
			DurationMinutes: 30,
			TavernLevel:     2,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 20},
				{Type: models.Swordsman, Count: 20},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Stone, Min: 50, Max: 100},
			},
		},
		{
			Name:            "Mandatory Overtime",
			Description:     "Force workers to do overtime in the mines",
			DurationMinutes: 30,
			TavernLevel:     2,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 20},
				{Type: models.Swordsman, Count: 20},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Iron, Min: 50, Max: 100},
			},
		},
		// Tavern Level 3 missions
		{
			Name:            "Forging Tools",
			Description:     "Forge new tools for your workers",
			DurationMinutes: 60,
			TavernLevel:     3,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 30},
				{Type: models.Swordsman, Count: 30},
			},
			ResourceCosts: models.Costs{
				Iron: 100,
			},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 100, Max: 200},
				{Type: models.Stone, Min: 100, Max: 200},
			},
		},
		// Tavern Level 4 missions
		{
			Name:            "Hire Stone Cutters",
			Description:     "Hire additional stone cutters",
			DurationMinutes: 120,
			TavernLevel:     4,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 40},
				{Type: models.Swordsman, Count: 40},
			},
			ResourceCosts: models.Costs{
				Wood: 200,
			},
			Rewards: []models.ResourceReward{
				{Type: models.Stone, Min: 300, Max: 500},
			},
		},
		{
			Name:            "Market Day",
			Description:     "Invite traders to a big market day",
			DurationMinutes: 240, // 4 hours
			TavernLevel:     4,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 50},
				{Type: models.Swordsman, Count: 50},
				{Type: models.Archer, Count: 50},
			},
			ResourceCosts: models.Costs{
				Wood:  400,
				Stone: 400,
				Iron:  400,
			},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 400, Max: 800},
				{Type: models.Stone, Min: 400, Max: 800},
				{Type: models.Iron, Min: 400, Max: 800},
			},
		},
		// Tavern Level 5 missions
		{
			Name:            "Create Trading Post",
			Description:     "Create a trading post to improve commercial relationships",
			DurationMinutes: 300, // 5 hours
			TavernLevel:     5,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 70},
				{Type: models.Swordsman, Count: 70},
				{Type: models.Archer, Count: 70},
				{Type: models.Horseman, Count: 30},
			},
			ResourceCosts: models.Costs{
				Wood:  700,
				Stone: 700,
				Iron:  700,
			},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 750, Max: 1500},
				{Type: models.Stone, Min: 750, Max: 1500},
				{Type: models.Iron, Min: 1250, Max: 2500},
			},
		},
		// Tavern Level 6 missions
		{
			Name:            "Collect Taxes",
			Description:     "Send soldiers to collect taxes from your subjects",
			DurationMinutes: 180, // 3 hours
			TavernLevel:     6,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 60},
				{Type: models.Horseman, Count: 40},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 200, Max: 400},
				{Type: models.Stone, Min: 200, Max: 400},
				{Type: models.Iron, Min: 200, Max: 400},
			},
		},
		{
			Name:            "Feed Miners",
			Description:     "Send food to the miners",
			DurationMinutes: 360, // 6 hours
			TavernLevel:     6,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 20},
				{Type: models.Swordsman, Count: 20},
				{Type: models.Archer, Count: 20},
				{Type: models.Horseman, Count: 20},
			},
			ResourceCosts: models.Costs{
				Wood:  300,
				Stone: 300,
				Food:  200,
			},
			Rewards: []models.ResourceReward{
				{Type: models.Iron, Min: 1000, Max: 2000},
			},
		},
		{
			Name:            "Chase Bandits Away",
			Description:     "Bandits have settled in your forests",
			DurationMinutes: 360, // 6 hours
			TavernLevel:     6,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 100},
				{Type: models.Swordsman, Count: 150},
				{Type: models.Archer, Count: 50},
				{Type: models.Horseman, Count: 100},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 300, Max: 1500},
				{Type: models.Stone, Min: 300, Max: 1500},
				{Type: models.Iron, Min: 520, Max: 2600},
			},
		},
		// Tavern Level 8 missions
		{
			Name:            "Jousting",
			Description:     "Organise a jousting tournament in the castle",
			DurationMinutes: 600, // 10 hours
			TavernLevel:     8,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 100},
				{Type: models.Swordsman, Count: 100},
				{Type: models.Archer, Count: 100},
				{Type: models.Horseman, Count: 100},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 960, Max: 1600},
				{Type: models.Stone, Min: 960, Max: 1600},
				{Type: models.Iron, Min: 1920, Max: 3200},
			},
		},
		// Tavern Level 10 missions
		{
			Name:            "Castle Festival",
			Description:     "Organise a festival in the castle",
			DurationMinutes: 720, // 12 hours
			TavernLevel:     10,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 100},
				{Type: models.Swordsman, Count: 100},
				{Type: models.Archer, Count: 100},
				{Type: models.Horseman, Count: 100},
			},
			ResourceCosts: models.Costs{},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 800, Max: 1000},
				{Type: models.Stone, Min: 800, Max: 1000},
				{Type: models.Iron, Min: 1600, Max: 2000},
			},
		},
	}
}

// GetMissionsForTavernLevel returns missions available at given tavern level
func GetMissionsForTavernLevel(tavernLevel int) []*models.Mission {
	all := LoadMissions()
	available := make([]*models.Mission, 0)
	for _, m := range all {
		if m.TavernLevel <= tavernLevel {
			available = append(available, m)
		}
	}
	return available
}

// GetBestMissionForBottleneck returns the mission with highest ROI for a specific resource
func GetBestMissionForBottleneck(tavernLevel int, bottleneck models.ResourceType) *models.Mission {
	available := GetMissionsForTavernLevel(tavernLevel)
	
	var best *models.Mission
	var bestROI float64
	
	for _, m := range available {
		roi := m.NetRewardByType(bottleneck) / float64(m.DurationMinutes)
		if roi > bestROI {
			bestROI = roi
			best = m
		}
	}
	
	return best
}
