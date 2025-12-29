package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/napolitain/solver-lnk/internal/loader"
	"github.com/napolitain/solver-lnk/internal/models"
	"github.com/napolitain/solver-lnk/internal/solver/castle"
	"github.com/napolitain/solver-lnk/internal/solver/units"
)

var (
	dataDir    string
	configFile string
	quiet      bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "castle",
		Short: "Lords and Knights Castle Build Order Optimizer",
		Long: `A greedy simulation solver that optimizes the build order
for Lords and Knights castle development.`,
		Run: runSolver,
	}

	rootCmd.Flags().StringVarP(&dataDir, "data", "d", "data", "Path to data directory")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to JSON config file")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Minimal output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSolver(cmd *cobra.Command, args []string) {
	// Colors
	titleColor := color.New(color.FgCyan, color.Bold)
	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgYellow)

	if !quiet {
		titleColor.Println("\n╭───────────────────────────╮")
		titleColor.Println("│  Lords and Knights        │")
		titleColor.Println("│  Build Order Optimizer    │")
		titleColor.Println("│  (Go Version)             │")
		titleColor.Println("╰───────────────────────────╯")
		fmt.Println()
	}

	// Load buildings
	buildings, err := loader.LoadBuildings(dataDir)
	if err != nil {
		color.Red("Error loading buildings: %v", err)
		os.Exit(1)
	}

	// Load technologies
	technologies, err := loader.LoadTechnologies(dataDir)
	if err != nil {
		color.Yellow("Warning: could not load technologies: %v", err)
		technologies = make(map[string]*models.Technology)
	}

	if !quiet {
		infoColor.Printf("📦 Loaded %d buildings, %d technologies\n\n", len(buildings), len(technologies))
	}

	// Load config or use defaults
	var initialState *models.GameState
	var targetLevels map[models.BuildingType]int

	if configFile != "" {
		config, err := models.LoadCastleConfig(configFile)
		if err != nil {
			color.Red("Error loading config: %v", err)
			os.Exit(1)
		}
		if err := models.ValidateCastleConfig(config); err != nil {
			color.Red("Invalid config: %v", err)
			os.Exit(1)
		}
		initialState = models.CastleConfigToGameState(config)
		targetLevels = models.GetTargetLevels()
		if !quiet {
			infoColor.Printf("📄 Loaded config from %s\n\n", configFile)
		}
	} else {
		// Default initial state
		initialState = models.NewGameState()
		initialState.Resources[models.Wood] = 120
		initialState.Resources[models.Stone] = 120
		initialState.Resources[models.Iron] = 120
		initialState.Resources[models.Food] = 40

		for _, bt := range models.AllBuildingTypes() {
			initialState.BuildingLevels[bt] = 1
		}

		// Default targets
		targetLevels = map[models.BuildingType]int{
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
	}

	if !quiet {
		printInitialState(initialState, targetLevels)
	}

	// Solve using all strategies and pick the best
	if !quiet {
		infoColor.Println("🔄 Solving with multiple strategies...")
	}

	solution, bestStrategy, allResults := castle.SolveAllStrategies(buildings, technologies, initialState, targetLevels)

	// Show all strategy results
	if !quiet {
		fmt.Println("\n📊 Strategy Comparison:")
		for _, r := range allResults {
			hours := float64(r.Solution.TotalTimeSeconds) / 3600
			days := hours / 24
			marker := "  "
			if r.Strategy == bestStrategy {
				marker = "✓ "
			}
			fmt.Printf("   %s%-15s: %.1f days (%.1f hours)\n", marker, r.Strategy, days, hours)
		}
	}

	successColor.Printf("\n✓ Best strategy: %s\n", bestStrategy)
	successColor.Printf("✓ Found solution with %d building upgrades and %d research tasks!\n\n",
		len(solution.BuildingActions), len(solution.ResearchActions))

	// Calculate final food status
	var finalFoodUsed, finalFoodCapacity int
	if len(solution.BuildingActions) > 0 {
		lastBuild := solution.BuildingActions[len(solution.BuildingActions)-1]
		finalFoodUsed = lastBuild.FoodUsed
		finalFoodCapacity = lastBuild.FoodCapacity
	}
	if len(solution.ResearchActions) > 0 {
		lastResearch := solution.ResearchActions[len(solution.ResearchActions)-1]
		if lastResearch.FoodUsed > finalFoodUsed {
			finalFoodUsed = lastResearch.FoodUsed
		}
		if lastResearch.FoodCapacity > finalFoodCapacity {
			finalFoodCapacity = lastResearch.FoodCapacity
		}
	}

	// Print build order table (includes units)
	printBuildOrder(solution, finalFoodUsed, finalFoodCapacity)

	// Print summary
	printSummary(solution, targetLevels, finalFoodUsed, finalFoodCapacity)
}

func printInitialState(state *models.GameState, targets map[models.BuildingType]int) {
	infoColor := color.New(color.FgYellow)

	infoColor.Println("📊 Initial State:")
	fmt.Printf("   Resources: Wood=%0.f Stone=%0.f Iron=%0.f Food=%0.f\n",
		state.Resources[models.Wood],
		state.Resources[models.Stone],
		state.Resources[models.Iron],
		state.Resources[models.Food])
	fmt.Println()

	infoColor.Println("🎯 Targets:")

	var sortedTargets []models.BuildingType
	for bt := range targets {
		sortedTargets = append(sortedTargets, bt)
	}
	sort.Slice(sortedTargets, func(i, j int) bool {
		return string(sortedTargets[i]) < string(sortedTargets[j])
	})

	for _, bt := range sortedTargets {
		fmt.Printf("   • %s: Level %d\n", formatBuildingName(string(bt)), targets[bt])
	}
	fmt.Println()
}

func printBuildOrder(solution *models.Solution, finalFoodUsed, finalFoodCapacity int) {
	// Merge and sort all actions
	type actionType int
	const (
		actionBuilding actionType = iota
		actionResearch
		actionUnit
	)

	type action struct {
		actionType   actionType
		startTime    int
		endTime      int
		name         string
		fromLevel    int
		toLevel      int
		count        int // for units
		costs        models.Costs
		foodUsed     int
		foodCapacity int
	}

	var allActions []action
	for _, a := range solution.BuildingActions {
		allActions = append(allActions, action{
			actionType:   actionBuilding,
			startTime:    a.StartTime,
			endTime:      a.EndTime,
			name:         string(a.BuildingType),
			fromLevel:    a.FromLevel,
			toLevel:      a.ToLevel,
			costs:        a.Costs,
			foodUsed:     a.FoodUsed,
			foodCapacity: a.FoodCapacity,
		})
	}
	for _, a := range solution.ResearchActions {
		allActions = append(allActions, action{
			actionType:   actionResearch,
			startTime:    a.StartTime,
			endTime:      a.EndTime,
			name:         a.TechnologyName,
			costs:        a.Costs,
			foodUsed:     a.FoodUsed,
			foodCapacity: a.FoodCapacity,
		})
	}

	// Add unit training actions after all other actions complete
	foodAvailable := finalFoodCapacity - finalFoodUsed
	if foodAvailable > 0 {
		solver := units.NewSolverWithConfig(int32(foodAvailable), units.ResourceProductionPerHour, units.MarketDistanceFields)
		unitSolution := solver.Solve()

		// Find the end time of last action
		var lastEndTime int
		for _, a := range allActions {
			if a.endTime > lastEndTime {
				lastEndTime = a.endTime
			}
		}

		currentTime := lastEndTime
		currentFoodUsed := finalFoodUsed

		// Add each unit type as an action
		for _, u := range units.AllUnits() {
			count := unitSolution.UnitCounts[u.Name]
			if count > 0 {
				trainingTime := count * u.TrainingTimeSeconds
				foodCost := count * u.FoodCost

				// Calculate total resource costs for this unit batch
				unitCosts := models.Costs{
					models.Wood:  u.ResourceCosts[models.Wood] * count,
					models.Stone: u.ResourceCosts[models.Stone] * count,
					models.Iron:  u.ResourceCosts[models.Iron] * count,
					models.Food:  foodCost,
				}

				allActions = append(allActions, action{
					actionType:   actionUnit,
					startTime:    currentTime,
					endTime:      currentTime + trainingTime,
					name:         u.Name,
					count:        count,
					costs:        unitCosts,
					foodUsed:     currentFoodUsed + foodCost,
					foodCapacity: finalFoodCapacity,
				})

				currentTime += trainingTime
				currentFoodUsed += foodCost
			}
		}
	}

	sort.Slice(allActions, func(i, j int) bool {
		return allActions[i].startTime < allActions[j].startTime
	})

	// Create table with new API
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader([]string{"#", "Queue", "Action", "Upgrade", "Start", "End", "Duration", "Costs", "Food"}),
	)

	// Add rows
	for i, a := range allActions {
		var queueType, upgradeStr string
		foodStr := fmt.Sprintf("%d/%d", a.foodUsed, a.foodCapacity)

		switch a.actionType {
		case actionBuilding:
			queueType = "🏗️ Building"
			upgradeStr = fmt.Sprintf("%d → %d", a.fromLevel, a.toLevel)
		case actionResearch:
			queueType = "🔬 Research"
			upgradeStr = ""
		case actionUnit:
			queueType = "⚔️ Train"
			upgradeStr = fmt.Sprintf("×%d", a.count)
		}

		duration := a.endTime - a.startTime
		name := formatBuildingName(a.name)

		row := []string{
			fmt.Sprintf("%d", i+1),
			queueType,
			name,
			upgradeStr,
			formatTime(a.startTime),
			formatTime(a.endTime),
			formatTime(duration),
			formatCosts(a.costs),
			foodStr,
		}
		_ = table.Append(row)
	}

	_ = table.Render()
}

func printSummary(solution *models.Solution, targets map[models.BuildingType]int, finalFoodUsed, finalFoodCapacity int) {
	successColor := color.New(color.FgGreen)
	errorColor := color.New(color.FgRed)

	totalHours := float64(solution.TotalTimeSeconds) / 3600
	totalDays := totalHours / 24

	fmt.Printf("\n⏱️  Total completion time: %s (%.1f hours = %.1f days)\n",
		formatTime(solution.TotalTimeSeconds), totalHours, totalDays)

	// Show final food status
	if finalFoodCapacity > 0 {
		remaining := finalFoodCapacity - finalFoodUsed
		fmt.Printf("\n🍞 Food: %d/%d used (%d remaining for units)\n",
			finalFoodUsed, finalFoodCapacity, remaining)
	}

	// Verify targets
	fmt.Println("\n📋 Target verification:")
	allOk := true
	for bt, target := range targets {
		final := solution.FinalState.BuildingLevels[bt]
		if final >= target {
			successColor.Printf("   ✅ %s: target=%d, final=%d\n", formatBuildingName(string(bt)), target, final)
		} else {
			errorColor.Printf("   ❌ %s: target=%d, final=%d\n", formatBuildingName(string(bt)), target, final)
			allOk = false
		}
	}

	if allOk {
		successColor.Println("\n✅ All buildings reached target levels!")
	} else {
		errorColor.Println("\n❌ Some buildings did not reach target levels!")
	}

	// Print researched technologies
	if len(solution.FinalState.ResearchedTechnologies) > 0 {
		fmt.Println("\n🔬 Researched technologies:")
		for tech := range solution.FinalState.ResearchedTechnologies {
			fmt.Printf("   • %s\n", tech)
		}
	}

	// Print units stats (units are already in the table above)
	if allOk && finalFoodCapacity > 0 {
		foodAvailable := finalFoodCapacity - finalFoodUsed
		printUnitsStats(foodAvailable)
	}
}

func printUnitsStats(foodAvailable int) {
	infoColor := color.New(color.FgCyan)
	
	// Create units solver with available food
	solver := units.NewSolverWithConfig(int32(foodAvailable), units.ResourceProductionPerHour, units.MarketDistanceFields)
	solution := solver.Solve()

	// Calculate total training time
	totalTrainingSeconds := 0
	for _, u := range units.AllUnits() {
		count := solution.UnitCounts[u.Name]
		if count > 0 {
			totalTrainingSeconds += count * u.TrainingTimeSeconds
		}
	}

	trainingDays := float64(totalTrainingSeconds) / 3600 / 24

	infoColor.Println("\n⚔️  Army Stats:")
	fmt.Printf("   • Total training time: %s (%.1f days)\n", formatTime(totalTrainingSeconds), trainingDays)
	fmt.Printf("   • Total food used: %d / %d\n", solution.TotalFood, foodAvailable)
	fmt.Printf("   • Trading throughput: %.0f resources/hour\n", solution.TotalThroughput)
	fmt.Printf("   • Defense vs Cavalry: %d\n", solution.DefenseVsCavalry)
	fmt.Printf("   • Defense vs Infantry: %d\n", solution.DefenseVsInfantry)
	fmt.Printf("   • Defense vs Artillery: %d\n", solution.DefenseVsArtillery)
	fmt.Printf("   • Min defense (balanced): %d\n", solution.MinDefense())
}

func formatTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func formatCosts(costs models.Costs) string {
	return fmt.Sprintf("W:%5d S:%5d I:%4d F:%2d",
		costs[models.Wood],
		costs[models.Stone],
		costs[models.Iron],
		costs[models.Food])
}

func formatBuildingName(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
