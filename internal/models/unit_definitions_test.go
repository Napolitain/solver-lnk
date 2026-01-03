package models

import (
	"testing"
)

func TestUnitTechRequirements(t *testing.T) {
	// These are the tech requirements from data/units/*
	expectedTechs := map[UnitType]string{
		Spearman:    "",             // No tech required
		Swordsman:   "Swordsmith",   // From data/units/swordsman
		Archer:      "Longbow",      // From data/units/archer
		Crossbowman: "Crossbow",     // From data/units/crossbowman
		Horseman:    "Stirrup",      // From data/units/horseman
		Lancer:      "Horse armour", // From data/units/lancer
	}

	for unitType, expectedTech := range expectedTechs {
		def := GetUnitDefinition(unitType)
		if def == nil {
			t.Errorf("No definition found for %s", unitType)
			continue
		}

		if def.RequiredTech != expectedTech {
			t.Errorf("%s: expected tech %q, got %q", unitType, expectedTech, def.RequiredTech)
		}

		// Also check the UnitTechRequirements map
		if UnitTechRequirements[unitType] != expectedTech {
			t.Errorf("UnitTechRequirements[%s]: expected %q, got %q",
				unitType, expectedTech, UnitTechRequirements[unitType])
		}
	}
}

func TestUnitResourceCosts(t *testing.T) {
	// These are the costs from data/units/*
	expectedCosts := map[UnitType]Costs{
		Spearman:    {Wood: 18, Stone: 6, Iron: 30},
		Swordsman:   {Wood: 43, Stone: 20, Iron: 48},
		Archer:      {Wood: 27, Stone: 12, Iron: 39},
		Crossbowman: {Wood: 50, Stone: 28, Iron: 55},
		Horseman:    {Wood: 25, Stone: 15, Iron: 45},
		Lancer:      {Wood: 70, Stone: 60, Iron: 80},
	}

	for unitType, expected := range expectedCosts {
		def := GetUnitDefinition(unitType)
		if def == nil {
			t.Errorf("No definition found for %s", unitType)
			continue
		}

		if def.ResourceCosts != expected {
			t.Errorf("%s costs: expected %+v, got %+v", unitType, expected, def.ResourceCosts)
		}
	}
}

func TestUnitFoodCosts(t *testing.T) {
	// These are the food costs from data/units/*
	expectedFood := map[UnitType]int{
		Spearman:    1,
		Swordsman:   1,
		Archer:      1,
		Crossbowman: 1,
		Horseman:    2,
		Lancer:      2,
	}

	for unitType, expected := range expectedFood {
		def := GetUnitDefinition(unitType)
		if def == nil {
			t.Errorf("No definition found for %s", unitType)
			continue
		}

		if def.FoodCost != expected {
			t.Errorf("%s food: expected %d, got %d", unitType, expected, def.FoodCost)
		}
	}
}

func TestUnitTrainingTimes(t *testing.T) {
	// These are the training times from data/units/*
	expectedTimes := map[UnitType]int{
		Spearman:    750,  // 12:30
		Swordsman:   1200, // 20:00
		Archer:      900,  // 15:00
		Crossbowman: 1350, // 22:30
		Horseman:    1050, // 17:30
		Lancer:      1860, // 31:00
	}

	for unitType, expected := range expectedTimes {
		def := GetUnitDefinition(unitType)
		if def == nil {
			t.Errorf("No definition found for %s", unitType)
			continue
		}

		if def.TrainingTimeSeconds != expected {
			t.Errorf("%s training time: expected %d, got %d", unitType, expected, def.TrainingTimeSeconds)
		}
	}
}

func TestUnitCategories(t *testing.T) {
	// These are the categories from data/units/*
	expectedCategories := map[UnitType]string{
		Spearman:    "Infantry",
		Swordsman:   "Infantry",
		Archer:      "Artillery",
		Crossbowman: "Artillery",
		Horseman:    "Cavalry",
		Lancer:      "Cavalry",
	}

	for unitType, expected := range expectedCategories {
		def := GetUnitDefinition(unitType)
		if def == nil {
			t.Errorf("No definition found for %s", unitType)
			continue
		}

		if def.Category != expected {
			t.Errorf("%s category: expected %q, got %q", unitType, expected, def.Category)
		}
	}
}

func TestAllUnitsHaveDefinitions(t *testing.T) {
	// All combat units should have definitions
	combatUnits := []UnitType{Spearman, Swordsman, Archer, Crossbowman, Horseman, Lancer}

	for _, ut := range combatUnits {
		def := GetUnitDefinition(ut)
		if def == nil {
			t.Errorf("No definition for combat unit %s", ut)
		}
	}
}

func TestUnitDefinitionConsistency(t *testing.T) {
	// Verify UnitTechRequirements matches GetUnitDefinition().RequiredTech
	for unitType, techName := range UnitTechRequirements {
		def := GetUnitDefinition(unitType)
		if def == nil {
			t.Errorf("UnitTechRequirements has %s but no definition exists", unitType)
			continue
		}

		if def.RequiredTech != techName {
			t.Errorf("%s: UnitTechRequirements says %q but definition says %q",
				unitType, techName, def.RequiredTech)
		}
	}
}
