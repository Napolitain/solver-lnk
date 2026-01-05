package castle

import (
	"testing"

	"github.com/napolitain/solver-lnk/internal/models"
)

func TestEventQueuePriority(t *testing.T) {
	q := NewEventQueue()

	// Push events in random order
	q.Push(Event{Time: 100, Type: EventStateChanged})
	q.Push(Event{Time: 100, Type: EventMissionComplete})
	q.Push(Event{Time: 100, Type: EventBuildingComplete})
	q.Push(Event{Time: 100, Type: EventResearchComplete})
	q.Push(Event{Time: 100, Type: EventTrainingComplete})

	// Should come out in priority order
	expected := []EventType{
		EventMissionComplete,  // priority 0
		EventBuildingComplete, // priority 1
		EventResearchComplete, // priority 2
		EventTrainingComplete, // priority 3
		EventStateChanged,     // priority 10
	}

	for i, exp := range expected {
		got := q.Pop()
		if got.Type != exp {
			t.Errorf("Event %d: expected %v, got %v", i, exp, got.Type)
		}
	}
}

func TestEventQueueTimeOrdering(t *testing.T) {
	q := NewEventQueue()

	q.Push(Event{Time: 300, Type: EventStateChanged})
	q.Push(Event{Time: 100, Type: EventStateChanged})
	q.Push(Event{Time: 200, Type: EventStateChanged})

	if e := q.Pop(); e.Time != 100 {
		t.Errorf("Expected time 100, got %d", e.Time)
	}
	if e := q.Pop(); e.Time != 200 {
		t.Errorf("Expected time 200, got %d", e.Time)
	}
	if e := q.Pop(); e.Time != 300 {
		t.Errorf("Expected time 300, got %d", e.Time)
	}
}

func TestEventQueuePushIfNotExists(t *testing.T) {
	q := NewEventQueue()

	q.Push(Event{Time: 100, Type: EventStateChanged})
	q.PushIfNotExists(Event{Time: 100, Type: EventStateChanged}) // Should be ignored
	q.PushIfNotExists(Event{Time: 200, Type: EventStateChanged}) // Should be added

	if q.Len() != 2 {
		t.Errorf("Expected 2 events, got %d", q.Len())
	}
}

func TestArmyOperations(t *testing.T) {
	army := models.Army{
		Spearman: 50,
		Archer:   30,
		Horseman: 20,
	}

	// Test Get
	if army.Get(models.Spearman) != 50 {
		t.Errorf("Expected 50 spearmen, got %d", army.Get(models.Spearman))
	}

	// Test Add
	army.Add(models.Spearman, 10)
	if army.Spearman != 60 {
		t.Errorf("Expected 60 spearmen after add, got %d", army.Spearman)
	}

	// Test Remove
	army.Remove(models.Spearman, 20)
	if army.Spearman != 40 {
		t.Errorf("Expected 40 spearmen after remove, got %d", army.Spearman)
	}

	// Test Remove floors at 0
	army.Remove(models.Spearman, 100)
	if army.Spearman != 0 {
		t.Errorf("Expected 0 spearmen after over-remove, got %d", army.Spearman)
	}
}

func TestArmyCanSatisfy(t *testing.T) {
	army := models.Army{
		Spearman: 50,
		Archer:   30,
		Horseman: 20,
	}

	// Should satisfy
	reqs := []models.UnitRequirement{
		{Type: models.Spearman, Count: 40},
		{Type: models.Archer, Count: 20},
	}
	if !army.CanSatisfy(reqs) {
		t.Error("Army should satisfy requirements")
	}

	// Should not satisfy
	reqs2 := []models.UnitRequirement{
		{Type: models.Spearman, Count: 100}, // Don't have enough
	}
	if army.CanSatisfy(reqs2) {
		t.Error("Army should not satisfy requirements")
	}
}

func TestStateResourceOperations(t *testing.T) {
	gs := models.NewGameState()
	gs.Resources[models.Wood] = 100
	gs.Resources[models.Stone] = 200
	gs.Resources[models.Iron] = 300

	state := NewState(gs)

	// Test GetResource
	if state.GetResource(models.Wood) != 100 {
		t.Errorf("Expected 100 wood, got %f", state.GetResource(models.Wood))
	}

	// Test SetResource
	state.SetResource(models.Wood, 150)
	if state.Resources[0] != 150 {
		t.Errorf("Expected 150 wood after set, got %f", state.Resources[0])
	}

	// Test AddResource
	state.AddResource(models.Stone, 50)
	if state.Resources[1] != 250 {
		t.Errorf("Expected 250 stone after add, got %f", state.Resources[1])
	}

	// Test storage cap
	state.StorageCaps[0] = 120 // Cap wood at 120
	state.AddResource(models.Wood, 100)
	state.CapResources()
	if state.Resources[0] != 120 {
		t.Errorf("Expected 120 wood (capped), got %f", state.Resources[0])
	}
}

func TestStateClone(t *testing.T) {
	state := &State{
		Now:             100,
		BuildingLevels:  models.BuildingLevelMap{Lumberjack: 5},
		Resources:       [3]float64{100, 200, 300},
		ResearchedTechs: map[string]bool{"Longbow": true},
		Army:            models.Army{Spearman: 50},
	}

	clone := state.Clone()

	// Modify original
	state.Now = 200
	state.BuildingLevels.Set(models.Lumberjack, 10)
	state.Resources[0] = 999
	state.ResearchedTechs["Beer tester"] = true
	state.Army.Spearman = 100

	// Clone should be unchanged
	if clone.Now != 100 {
		t.Errorf("Clone Now should be 100, got %d", clone.Now)
	}
	if clone.BuildingLevels.Get(models.Lumberjack) != 5 {
		t.Errorf("Clone Lumberjack should be 5, got %d", clone.BuildingLevels.Get(models.Lumberjack))
	}
	if clone.Resources[0] != 100 {
		t.Errorf("Clone Wood should be 100, got %f", clone.Resources[0])
	}
	if clone.ResearchedTechs["Beer tester"] {
		t.Error("Clone should not have Beer tester researched")
	}
	if clone.Army.Spearman != 50 {
		t.Errorf("Clone Spearman should be 50, got %d", clone.Army.Spearman)
	}
}

func TestAdvanceTime(t *testing.T) {
	gs := models.NewGameState()
	gs.Resources[models.Wood] = 100

	state := NewState(gs)
	state.ProductionRates[0] = 10 // 10 wood per hour
	state.ProductionBonus = 1.0
	state.StorageCaps[0] = 1000

	solver := &Solver{}

	// Advance 1 hour (3600 seconds)
	solver.advanceTime(state, 3600)

	if state.Now != 3600 {
		t.Errorf("Expected Now=3600, got %d", state.Now)
	}

	// Should have produced 10 wood
	expectedWood := 100.0 + 10.0
	if state.Resources[0] != expectedWood {
		t.Errorf("Expected %f wood, got %f", expectedWood, state.Resources[0])
	}
}

func TestAdvanceTimeWithProductionBonus(t *testing.T) {
	gs := models.NewGameState()
	gs.Resources[models.Wood] = 100

	state := NewState(gs)
	state.ProductionRates[0] = 10
	state.ProductionBonus = 1.10 // 10% bonus (e.g., from Beer tester)
	state.StorageCaps[0] = 1000

	solver := &Solver{}

	// Advance 1 hour
	solver.advanceTime(state, 3600)

	// Should have produced 10 * 1.10 = 11 wood
	expectedWood := 100.0 + 11.0
	if state.Resources[0] != expectedWood {
		t.Errorf("Expected %f wood, got %f", expectedWood, state.Resources[0])
	}
}

func TestAdvanceTimeStorageCap(t *testing.T) {
	gs := models.NewGameState()
	gs.Resources[models.Wood] = 90

	state := NewState(gs)
	state.ProductionRates[0] = 100 // 100 wood per hour
	state.ProductionBonus = 1.0
	state.StorageCaps[0] = 100 // Cap at 100

	solver := &Solver{}

	// Advance 1 hour - would produce 100 but cap at 100 total
	solver.advanceTime(state, 3600)

	if state.Resources[0] != 100 {
		t.Errorf("Expected 100 wood (capped), got %f", state.Resources[0])
	}
}

func TestCanAfford(t *testing.T) {
	gs := models.NewGameState()
	gs.Resources[models.Wood] = 100
	gs.Resources[models.Stone] = 50
	gs.Resources[models.Iron] = 25

	state := NewState(gs)
	solver := &Solver{}

	// Should afford
	costs1 := models.Costs{Wood: 50, Stone: 30, Iron: 20}
	if !solver.canAfford(state, costs1) {
		t.Error("Should afford costs1")
	}

	// Should not afford (not enough iron)
	costs2 := models.Costs{Wood: 50, Stone: 30, Iron: 30}
	if solver.canAfford(state, costs2) {
		t.Error("Should not afford costs2")
	}
}

func TestWaitTimeForCosts(t *testing.T) {
	gs := models.NewGameState()
	gs.Resources[models.Wood] = 50

	state := NewState(gs)
	state.ProductionRates[0] = 10 // 10 wood per hour
	state.ProductionBonus = 1.0

	solver := &Solver{}

	// Need 100 wood, have 50, produce 10/hour
	// Shortfall = 50, rate = 10/hour, time = 5 hours = 18000 seconds
	costs := models.Costs{Wood: 100}
	waitTime := solver.waitTimeForCosts(state, costs)

	// Should be approximately 5 hours
	expectedSeconds := 5 * 3600
	if waitTime < expectedSeconds-100 || waitTime > expectedSeconds+100 {
		t.Errorf("Expected ~%d seconds, got %d", expectedSeconds, waitTime)
	}
}
