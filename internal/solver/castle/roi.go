package castle

import (
	"sort"

	"github.com/napolitain/solver-lnk/internal/models"
)

// ROIMetric represents the components of an ROI calculation
type ROIMetric struct {
	GainPerHour   float64
	TotalCost     float64
	ScarcityBonus float64 // Multiplier adjustment (0.0 = no adjustment, 0.5 = +50% ROI)
}

// Calculate computes the final ROI value
func (m ROIMetric) Calculate() float64 {
	if m.TotalCost <= 0 {
		return m.GainPerHour * 1000 // Very high ROI if free
	}
	baseROI := m.GainPerHour / m.TotalCost
	return baseROI * (1.0 + m.ScarcityBonus)
}

// buildingROI calculates ROI for a building upgrade
func (s *Solver) buildingROI(state *State, action *BuildingAction) float64 {
	metric := s.calculateBuildingMetric(state, action)
	return metric.Calculate()
}

// calculateBuildingMetric computes the ROI metric for a building action
func (s *Solver) calculateBuildingMetric(state *State, action *BuildingAction) ROIMetric {
	costs := action.Costs()
	totalResourceCost := float64(costs.Wood + costs.Stone + costs.Iron)
	if totalResourceCost <= 0 {
		totalResourceCost = 1 // Avoid division by zero
	}

	// Special handling for Tavern - calculate mission ROI
	if action.BuildingType == models.Tavern {
		return s.calculateTavernMetric(state, action.ToLevel, totalResourceCost)
	}

	// Special handling for Arsenal - no direct ROI (deferred)
	if action.BuildingType == models.Arsenal {
		return ROIMetric{GainPerHour: 0, TotalCost: totalResourceCost}
	}

	// For non-production buildings (Keep, Fortifications, etc.), return 0
	if action.LevelData.ProductionRate == nil {
		return ROIMetric{GainPerHour: 0, TotalCost: totalResourceCost}
	}

	// Production building - calculate rate increase
	var currentRate float64
	if action.FromLevel > 0 {
		prevData := action.Building.GetLevelData(action.FromLevel)
		if prevData != nil && prevData.ProductionRate != nil {
			currentRate = *prevData.ProductionRate
		}
	}

	newRate := *action.LevelData.ProductionRate
	gainPerHour := newRate - currentRate

	// Apply dynamic scarcity multiplier based on remaining build costs
	scarcity := s.calculateDynamicScarcity(state, action.BuildingType)

	return ROIMetric{
		GainPerHour:   gainPerHour,
		TotalCost:     totalResourceCost,
		ScarcityBonus: scarcity - 1.0, // Convert multiplier to bonus (1.5x → 0.5 bonus)
	}
}

// calculateTavernMetric computes the ROI metric for Tavern upgrades
func (s *Solver) calculateTavernMetric(state *State, toLevel int, totalResourceCost float64) ROIMetric {
	if len(s.Missions) == 0 {
		return ROIMetric{GainPerHour: 0, TotalCost: totalResourceCost}
	}

	// Find missions unlocked at this level
	var bestNewMissionROI float64
	for _, mission := range s.Missions {
		if mission.TavernLevel == toLevel {
			roi := mission.NetAverageRewardPerHour()
			if roi > bestNewMissionROI {
				bestNewMissionROI = roi
			}
		}
	}

	if bestNewMissionROI == 0 {
		// No new missions at this level, but still needed for progression
		// Check if higher levels have better missions
		for _, mission := range s.Missions {
			if mission.TavernLevel > toLevel {
				roi := mission.NetAverageRewardPerHour()
				if roi > bestNewMissionROI {
					// Discount by levels away
					levelsAway := mission.TavernLevel - toLevel
					bestNewMissionROI = roi / float64(levelsAway+1)
				}
			}
		}
	}

	return ROIMetric{
		GainPerHour: bestNewMissionROI,
		TotalCost:   totalResourceCost,
	}
}

// calculateProductionTechMetric computes the ROI metric for production technologies
func (s *Solver) calculateProductionTechMetric(state *State, action *ProductionTechAction) ROIMetric {
	bonusMultiplier := ProductionTechBonus

	// Current total production rate
	totalRate := state.GetProductionRate(models.Wood) +
		state.GetProductionRate(models.Stone) +
		state.GetProductionRate(models.Iron)

	// Gain in production rate (resources per hour)
	gainPerHour := totalRate * bonusMultiplier

	// Calculate total resource cost (tech cost + Library upgrade costs if needed)
	techCosts := action.Technology.Costs
	totalCost := float64(techCosts.Wood + techCosts.Stone + techCosts.Iron)

	libraryLevel := state.GetBuildingLevel(models.Library)
	if libraryLevel < action.RequiredLibraryLevel {
		libraryBuilding := s.Buildings[models.Library]
		if libraryBuilding != nil {
			for level := libraryLevel + 1; level <= action.RequiredLibraryLevel; level++ {
				levelData := libraryBuilding.GetLevelData(level)
				if levelData != nil {
					totalCost += float64(levelData.Costs.Wood + levelData.Costs.Stone + levelData.Costs.Iron)
				}
			}
		}
	}

	return ROIMetric{
		GainPerHour: gainPerHour,
		TotalCost:   totalCost,
	}
}

// productionTechROI calculates the ROI for a production tech using ROIMetric
func (s *Solver) productionTechROI(state *State, action *ProductionTechAction) float64 {
	metric := s.calculateProductionTechMetric(state, action)
	return metric.Calculate()
}

// getBestProductionTechAction returns the best production tech to research
func (s *Solver) getBestProductionTechAction(state *State) *ProductionTechAction {
	libraryBuilding := s.Buildings[models.Library]
	if libraryBuilding == nil {
		return nil
	}

	var best *ProductionTechAction
	var bestROI float64

	for _, techName := range []string{"Beer tester", "Wheelbarrow"} {
		if state.ResearchedTechs[techName] {
			continue
		}
		tech := s.Technologies[techName]
		if tech == nil {
			continue
		}

		action := &ProductionTechAction{
			Technology:           tech,
			RequiredLibraryLevel: tech.RequiredLibraryLevel,
		}

		roi := s.productionTechROI(state, action)
		if roi > bestROI {
			bestROI = roi
			best = action
		}
	}

	return best
}

// getAllBuildingActionsSortedByROI returns all building actions sorted by ROI (best first)
func (s *Solver) getAllBuildingActionsSortedByROI(state *State) []*BuildingAction {
	var candidates []*BuildingAction
	var zeroROICandidates []*BuildingAction

	s.TargetLevels.Each(func(bt models.BuildingType, target int) {
		if target == 0 {
			return
		}
		current := state.GetBuildingLevel(bt)
		if current >= target {
			return
		}

		building := s.Buildings[bt]
		if building == nil {
			return
		}

		toLevel := current + 1
		levelData := building.GetLevelData(toLevel)
		if levelData == nil {
			return
		}

		action := &BuildingAction{
			BuildingType: bt,
			FromLevel:    current,
			ToLevel:      toLevel,
			Building:     building,
			LevelData:    levelData,
		}

		// Separate zero-ROI buildings (Fortifications, Keep, etc.) from productive buildings
		// These should ONLY be built after all production targets are reached
		if s.buildingROI(state, action) == 0 {
			zeroROICandidates = append(zeroROICandidates, action)
		} else {
			candidates = append(candidates, action)
		}
	})

	// If there are still production buildings to build, return only those
	// Zero-ROI buildings are deferred until production is complete
	if len(candidates) > 0 {
		// Sort by ROI (descending)
		sort.Slice(candidates, func(i, j int) bool {
			roiI := s.buildingROI(state, candidates[i])
			roiJ := s.buildingROI(state, candidates[j])
			if roiI != roiJ {
				return roiI > roiJ
			}
			return candidates[i].BuildingType < candidates[j].BuildingType
		})
		return candidates
	}

	// All production buildings are at target - now build zero-ROI buildings
	// Sort by build time (shorter first to finish faster)
	sort.Slice(zeroROICandidates, func(i, j int) bool {
		timeI := zeroROICandidates[i].LevelData.BuildTimeSeconds
		timeJ := zeroROICandidates[j].LevelData.BuildTimeSeconds
		if timeI != timeJ {
			return timeI < timeJ
		}
		return zeroROICandidates[i].BuildingType < zeroROICandidates[j].BuildingType
	})

	return zeroROICandidates
}
