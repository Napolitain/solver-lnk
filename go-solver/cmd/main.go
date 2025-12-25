package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/napolitain/solver-lnk/pkg/loader"
	"github.com/napolitain/solver-lnk/pkg/models"
	"github.com/napolitain/solver-lnk/pkg/solver"
)

func formatTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func formatCosts(costs models.Costs) string {
	return fmt.Sprintf("W:%5d S:%5d I:%4d F:%3d",
		costs[models.Wood],
		costs[models.Stone],
		costs[models.Iron],
		costs[models.Food])
}

func main() {
	// Find data directory
	dataDir := "../data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		// Try from go-solver directory
		dataDir = filepath.Join("..", "data")
	}

	fmt.Println("╭───────────────────────╮")
	fmt.Println("│ Lords and Knights     │")
	fmt.Println("│ Build Order Optimizer │")
	fmt.Println("│ (Go Version)          │")
	fmt.Println("╰───────────────────────╯")
	fmt.Println()

	// Load buildings
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		fmt.Printf("Error loading buildings: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d buildings\n", len(buildings))

	// Load technologies
	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		fmt.Printf("Warning: could not load technologies: %v\n", err)
		technologies = make(map[string]*models.Technology)
	}
	fmt.Printf("Loaded %d technologies\n", len(technologies))

	// Define initial state
	initialState := models.NewGameState()
	initialState.Resources[models.Wood] = 120
	initialState.Resources[models.Stone] = 120
	initialState.Resources[models.Iron] = 120
	initialState.Resources[models.Food] = 40

	// All buildings start at level 1
	for _, bt := range models.AllBuildingTypes() {
		initialState.BuildingLevels[bt] = 1
	}

	// Define targets
	targetLevels := map[models.BuildingType]int{
		models.Lumberjack:     30,
		models.Quarry:         30,
		models.OreMine:        30,
		models.Farm:           30,
		models.WoodStore:      20,
		models.StoneStore:     20,
		models.OreStore:       20,
		models.Keep:           10,
		models.Arsenal:        30,
		models.Library:        10,
		models.Tavern:         10,
		models.Market:         8,
		models.Fortifications: 20,
	}

	fmt.Println("\nInitial State:")
	for _, bt := range models.AllBuildingTypes() {
		fmt.Printf("  • %s: Level %d\n", bt, initialState.BuildingLevels[bt])
	}

	fmt.Println("\nResources:")
	fmt.Printf("  Wood: %.0f  Stone: %.0f  Iron: %.0f  Food: %.0f\n",
		initialState.Resources[models.Wood],
		initialState.Resources[models.Stone],
		initialState.Resources[models.Iron],
		initialState.Resources[models.Food])

	fmt.Println("\nTargets:")
	// Sort target levels for consistent output
	var sortedTargets []models.BuildingType
	for bt := range targetLevels {
		sortedTargets = append(sortedTargets, bt)
	}
	sort.Slice(sortedTargets, func(i, j int) bool {
		return string(sortedTargets[i]) < string(sortedTargets[j])
	})
	for _, bt := range sortedTargets {
		fmt.Printf("  • %s: Level %d\n", bt, targetLevels[bt])
	}

	fmt.Println("\nSolving...")

	// Create and run solver
	s := solver.NewGreedySolver(buildings, technologies, initialState, targetLevels)
	solution := s.Solve()

	fmt.Printf("\n✓ Found solution with %d building upgrades and %d research tasks!\n\n",
		len(solution.BuildingActions), len(solution.ResearchActions))

	// Print table header
	fmt.Println(strings.Repeat("─", 120))
	fmt.Printf("%-4s │ %-10s │ %-20s │ %-8s │ %-12s │ %-12s │ %-10s │ %-25s\n",
		"#", "Queue", "Action", "Upgrade", "Start", "End", "Duration", "Costs")
	fmt.Println(strings.Repeat("─", 120))

	// Merge and sort all actions by start time
	type action struct {
		isBuilding bool
		startTime  int
		endTime    int
		name       string
		fromLevel  int
		toLevel    int
		costs      models.Costs
	}

	var allActions []action
	for _, a := range solution.BuildingActions {
		allActions = append(allActions, action{
			isBuilding: true,
			startTime:  a.StartTime,
			endTime:    a.EndTime,
			name:       string(a.BuildingType),
			fromLevel:  a.FromLevel,
			toLevel:    a.ToLevel,
			costs:      a.Costs,
		})
	}
	for _, a := range solution.ResearchActions {
		allActions = append(allActions, action{
			isBuilding: false,
			startTime:  a.StartTime,
			endTime:    a.EndTime,
			name:       a.TechnologyName,
			costs:      a.Costs,
		})
	}

	sort.Slice(allActions, func(i, j int) bool {
		return allActions[i].startTime < allActions[j].startTime
	})

	// Print actions
	for i, a := range allActions {
		queueType := "🏗️ Building"
		upgradeStr := fmt.Sprintf("%d → %d", a.fromLevel, a.toLevel)
		if !a.isBuilding {
			queueType = "📚 Research"
			upgradeStr = ""
		}

		duration := a.endTime - a.startTime
		name := strings.Replace(a.name, "_", " ", -1)
		name = strings.Title(name)

		fmt.Printf("%4d │ %-10s │ %-20s │ %-8s │ %12s │ %12s │ %10s │ %s\n",
			i+1,
			queueType,
			name,
			upgradeStr,
			formatTime(a.startTime),
			formatTime(a.endTime),
			formatTime(duration),
			formatCosts(a.costs))
	}

	fmt.Println(strings.Repeat("─", 120))

	// Summary
	totalHours := float64(solution.TotalTimeSeconds) / 3600
	totalDays := totalHours / 24
	fmt.Printf("\nTotal completion time: %s (%.1f hours = %.1f days)\n",
		formatTime(solution.TotalTimeSeconds), totalHours, totalDays)

	// Verify all targets reached
	fmt.Println("\nTarget verification:")
	allOk := true
	for bt, target := range targetLevels {
		final := solution.FinalState.BuildingLevels[bt]
		status := "✅"
		if final < target {
			status = "❌"
			allOk = false
		}
		fmt.Printf("  %s %s: target=%d, final=%d\n", status, bt, target, final)
	}

	if allOk {
		fmt.Println("\n✅ All buildings reached target levels!")
	} else {
		fmt.Println("\n❌ Some buildings did not reach target levels!")
	}

	// Print researched technologies
	if len(solution.FinalState.ResearchedTechnologies) > 0 {
		fmt.Println("\nResearched technologies:")
		for tech := range solution.FinalState.ResearchedTechnologies {
			fmt.Printf("  • %s\n", tech)
		}
	}
}
