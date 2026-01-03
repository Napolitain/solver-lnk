package loader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/napolitain/solver-lnk/internal/models"
)

// missionJSON represents the JSON structure for a mission
type missionJSON struct {
	Name           string `json:"name"`
	TavernLevel    int    `json:"tavern_level"`
	MaxTavernLevel int    `json:"max_tavern_level"` // 0 means no limit
	DurationMins   int    `json:"duration_minutes"`
	UnitsRequired  []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	} `json:"units_required"`
	ResourceCosts struct {
		Wood  int `json:"wood"`
		Stone int `json:"stone"`
		Iron  int `json:"iron"`
	} `json:"resource_costs"`
	Rewards []struct {
		Type string `json:"type"`
		Min  int    `json:"min"`
		Max  int    `json:"max"`
	} `json:"rewards"`
}

// LoadMissionsFromFile loads missions from JSON file
func LoadMissionsFromFile(dataDir string) ([]*models.Mission, error) {
	path := filepath.Join(dataDir, "missions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var missionsMap map[string]missionJSON
	if err := json.Unmarshal(data, &missionsMap); err != nil {
		return nil, err
	}

	missions := make([]*models.Mission, 0, len(missionsMap))
	for _, mj := range missionsMap {
		m := &models.Mission{
			Name:            mj.Name,
			TavernLevel:     mj.TavernLevel,
			MaxTavernLevel:  mj.MaxTavernLevel,
			DurationMinutes: mj.DurationMins,
			ResourceCosts: models.Costs{
				Wood:  mj.ResourceCosts.Wood,
				Stone: mj.ResourceCosts.Stone,
				Iron:  mj.ResourceCosts.Iron,
			},
		}

		// Convert unit requirements
		for _, ur := range mj.UnitsRequired {
			m.UnitsRequired = append(m.UnitsRequired, models.UnitRequirement{
				Type:  models.UnitType(ur.Type),
				Count: ur.Count,
			})
		}

		// Convert rewards
		for _, r := range mj.Rewards {
			rt := models.Wood
			switch r.Type {
			case "stone":
				rt = models.Stone
			case "iron":
				rt = models.Iron
			}
			m.Rewards = append(m.Rewards, models.ResourceReward{
				Type: rt,
				Min:  r.Min,
				Max:  r.Max,
			})
		}

		missions = append(missions, m)
	}

	// Sort by tavern level for determinism
	sort.Slice(missions, func(i, j int) bool {
		if missions[i].TavernLevel != missions[j].TavernLevel {
			return missions[i].TavernLevel < missions[j].TavernLevel
		}
		return missions[i].Name < missions[j].Name
	})

	return missions, nil
}

// LoadMissions returns all available missions with their requirements (hardcoded fallback)
func LoadMissions() []*models.Mission {
	return []*models.Mission{
		// Tavern Level 1 missions
		{
			Name:            "Overtime Wood",
			DurationMinutes: 5,
			TavernLevel:     1,
			UnitsRequired:   []models.UnitRequirement{{Type: models.Spearman, Count: 5}},
			Rewards:         []models.ResourceReward{{Type: models.Wood, Min: 5, Max: 10}},
		},
		{
			Name:            "Overtime Stone",
			DurationMinutes: 5,
			TavernLevel:     1,
			UnitsRequired:   []models.UnitRequirement{{Type: models.Spearman, Count: 5}},
			Rewards:         []models.ResourceReward{{Type: models.Stone, Min: 5, Max: 10}},
		},
		{
			Name:            "Overtime Ore",
			DurationMinutes: 5,
			TavernLevel:     1,
			UnitsRequired:   []models.UnitRequirement{{Type: models.Spearman, Count: 5}},
			Rewards:         []models.ResourceReward{{Type: models.Iron, Min: 5, Max: 10}},
		},
		{
			Name:            "Mandatory Overtime",
			DurationMinutes: 5,
			TavernLevel:     1,
			UnitsRequired:   []models.UnitRequirement{{Type: models.Spearman, Count: 15}},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 5, Max: 10},
				{Type: models.Stone, Min: 5, Max: 10},
				{Type: models.Iron, Min: 5, Max: 10},
			},
		},
		// Tavern Level 2 missions
		{
			Name:            "Hunting",
			DurationMinutes: 15,
			TavernLevel:     2,
			UnitsRequired:   []models.UnitRequirement{{Type: models.Spearman, Count: 15}},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 10, Max: 20},
				{Type: models.Stone, Min: 10, Max: 20},
				{Type: models.Iron, Min: 10, Max: 20},
			},
		},
		{
			Name:            "Chop Wood",
			DurationMinutes: 30,
			TavernLevel:     2,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 20},
				{Type: models.Horseman, Count: 20},
			},
			Rewards: []models.ResourceReward{{Type: models.Wood, Min: 50, Max: 100}},
		},
		{
			Name:            "Help Stone Cutters",
			DurationMinutes: 120,
			TavernLevel:     2,
			UnitsRequired:   []models.UnitRequirement{{Type: models.Spearman, Count: 40}},
			Rewards:         []models.ResourceReward{{Type: models.Stone, Min: 100, Max: 200}},
		},
		// Tavern Level 3 missions
		{
			Name:            "Forging Tools",
			DurationMinutes: 45,
			TavernLevel:     3,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 20},
				{Type: models.Archer, Count: 40},
				{Type: models.Horseman, Count: 30},
			},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 60, Max: 120},
				{Type: models.Stone, Min: 100, Max: 200},
				{Type: models.Iron, Min: 60, Max: 120},
			},
		},
		// Tavern Level 4 missions
		{
			Name:            "Market Day",
			DurationMinutes: 240,
			TavernLevel:     4,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 50},
				{Type: models.Archer, Count: 50},
				{Type: models.Horseman, Count: 50},
			},
			ResourceCosts: models.Costs{Wood: 400, Stone: 400, Iron: 400},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 400, Max: 800},
				{Type: models.Stone, Min: 400, Max: 800},
				{Type: models.Iron, Min: 400, Max: 800},
			},
		},
		{
			Name:            "Hire Stone Cutters",
			DurationMinutes: 360,
			TavernLevel:     4,
			UnitsRequired:   []models.UnitRequirement{{Type: models.Spearman, Count: 30}},
			ResourceCosts:   models.Costs{Wood: 75, Stone: 75},
			Rewards:         []models.ResourceReward{{Type: models.Stone, Min: 396, Max: 600}},
		},
		// Tavern Level 5 missions
		{
			Name:            "Feed Miners",
			DurationMinutes: 360,
			TavernLevel:     5,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 20},
				{Type: models.Archer, Count: 20},
				{Type: models.Horseman, Count: 20},
				{Type: models.Crossbowman, Count: 20},
			},
			ResourceCosts: models.Costs{Wood: 300, Stone: 300, Iron: 200},
			Rewards:       []models.ResourceReward{{Type: models.Iron, Min: 1000, Max: 2000}},
		},
		// Tavern Level 6 missions
		{
			Name:            "Create Trading Post",
			DurationMinutes: 300,
			TavernLevel:     6,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 70},
				{Type: models.Archer, Count: 70},
				{Type: models.Horseman, Count: 70},
				{Type: models.Crossbowman, Count: 30},
			},
			ResourceCosts: models.Costs{Wood: 700, Stone: 700, Iron: 700},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 750, Max: 1500},
				{Type: models.Stone, Min: 750, Max: 1500},
				{Type: models.Iron, Min: 1250, Max: 2500},
			},
		},
		// Tavern Level 7 missions
		{
			Name:            "Chase Bandits Away",
			DurationMinutes: 360,
			TavernLevel:     7,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 100},
				{Type: models.Archer, Count: 150},
				{Type: models.Horseman, Count: 50},
				{Type: models.Crossbowman, Count: 100},
			},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 300, Max: 1500},
				{Type: models.Stone, Min: 300, Max: 1500},
				{Type: models.Iron, Min: 520, Max: 2600},
			},
		},
		// Tavern Level 8 missions
		{
			Name:            "Collect Taxes",
			DurationMinutes: 1440,
			TavernLevel:     8,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 200},
				{Type: models.Archer, Count: 200},
				{Type: models.Horseman, Count: 100},
				{Type: models.Crossbowman, Count: 50},
			},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 920, Max: 2300},
				{Type: models.Stone, Min: 920, Max: 2300},
				{Type: models.Iron, Min: 1200, Max: 3000},
			},
		},
		// Tavern Level 9 missions
		{
			Name:            "Jousting",
			DurationMinutes: 600,
			TavernLevel:     9,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 100},
				{Type: models.Archer, Count: 100},
				{Type: models.Horseman, Count: 100},
				{Type: models.Lancer, Count: 100},
			},
			Rewards: []models.ResourceReward{
				{Type: models.Wood, Min: 960, Max: 1600},
				{Type: models.Stone, Min: 960, Max: 1600},
				{Type: models.Iron, Min: 1920, Max: 3200},
			},
		},
		// Tavern Level 10 missions
		{
			Name:            "Castle Festival",
			DurationMinutes: 720,
			TavernLevel:     10,
			UnitsRequired: []models.UnitRequirement{
				{Type: models.Spearman, Count: 100},
				{Type: models.Archer, Count: 100},
				{Type: models.Horseman, Count: 100},
				{Type: models.Lancer, Count: 100},
			},
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
