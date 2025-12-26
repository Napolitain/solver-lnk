package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/napolitain/solver-lnk/internal/units"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "units",
		Short: "Lords and Knights Army & Trading Optimizer",
		Long: `Optimizes army composition for balanced defense while ensuring
sufficient trading capacity to convert all resource production to silver.`,
		Run: runSolver,
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSolver(cmd *cobra.Command, args []string) {
	titleColor := color.New(color.FgCyan, color.Bold)
	successColor := color.New(color.FgGreen, color.Bold)

	titleColor.Println("\n╭───────────────────────────╮")
	titleColor.Println("│  Lords and Knights        │")
	titleColor.Println("│  Army & Trading Optimizer │")
	titleColor.Println("╰───────────────────────────╯")
	fmt.Println()

	// Print constants
	fmt.Println("📊 Castle Status (Maxed):")
	fmt.Printf("   Food capacity: %d (Farm 30)\n", units.MaxFoodCapacity)
	fmt.Printf("   Resource production: %d/hour (LJ30 + Q30 + OM30)\n", units.ResourceProductionPerHour)
	fmt.Printf("   Market distance: %d fields (Keep 10)\n", units.MarketDistanceFields)
	fmt.Printf("   Exchange rate: 50 resources = 1 silver\n")
	fmt.Println()

	// Show unit stats
	fmt.Println("📋 Available Units:")
	printUnitStats()

	// Solve
	fmt.Println("\n🔄 Optimizing army composition...")
	solver := units.NewSolver()
	solution := solver.Solve()

	// Print results
	successColor.Println("\n✓ Optimal composition found!")
	printSolution(solution)
}

func printUnitStats() {
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader([]string{"Unit", "Food", "Speed", "Capacity", "Throughput/h", "Def Cav", "Def Inf", "Def Art", "Total Def"}),
	)

	for _, u := range units.AllUnits() {
		throughput := u.ThroughputPerHour(units.RoundTripFields)
		row := []string{
			u.Name,
			fmt.Sprintf("%d", u.FoodCost),
			fmt.Sprintf("%.1f min/field", u.SpeedMinutesField),
			fmt.Sprintf("%d", u.TransportCapacity),
			fmt.Sprintf("%.0f", throughput),
			fmt.Sprintf("%d", u.DefenseVsCavalry),
			fmt.Sprintf("%d", u.DefenseVsInfantry),
			fmt.Sprintf("%d", u.DefenseVsArtillery),
			fmt.Sprintf("%d", u.TotalDefense()),
		}
		table.Append(row)
	}
	table.Render()
}

func printSolution(solution *units.Solution) {
	fmt.Println("\n📦 Army Composition:")
	
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader([]string{"Unit", "Count", "Food Used", "Throughput/h", "Def Cav", "Def Inf", "Def Art"}),
	)

	for _, u := range units.AllUnits() {
		count := solution.UnitCounts[u.Name]
		if count > 0 {
			throughput := float64(count) * u.ThroughputPerHour(units.RoundTripFields)
			row := []string{
				u.Name,
				fmt.Sprintf("%d", count),
				fmt.Sprintf("%d", count*u.FoodCost),
				fmt.Sprintf("%.0f", throughput),
				fmt.Sprintf("%d", count*u.DefenseVsCavalry),
				fmt.Sprintf("%d", count*u.DefenseVsInfantry),
				fmt.Sprintf("%d", count*u.DefenseVsArtillery),
			}
			table.Append(row)
		}
	}
	table.Render()

	fmt.Println("\n📊 Summary:")
	fmt.Printf("   Total food used: %d / %d\n", solution.TotalFood, units.MaxFoodCapacity)
	fmt.Printf("   Trading throughput: %.0f / %d resources/hour\n", 
		solution.TotalThroughput, units.ResourceProductionPerHour)
	fmt.Printf("   Silver income: %.2f/hour (%.1f/day)\n", 
		solution.SilverPerHour, solution.SilverPerHour*24)

	fmt.Println("\n🛡️  Defense Totals:")
	fmt.Printf("   vs Cavalry:   %d\n", solution.DefenseVsCavalry)
	fmt.Printf("   vs Infantry:  %d\n", solution.DefenseVsInfantry)
	fmt.Printf("   vs Artillery: %d\n", solution.DefenseVsArtillery)
	fmt.Printf("   Minimum (balanced): %d\n", solution.MinDefense())
}
