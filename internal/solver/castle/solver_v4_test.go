package castle_test

import (
"sort"
"testing"

"github.com/napolitain/solver-lnk/internal/loader"
"github.com/napolitain/solver-lnk/internal/models"
castle "github.com/napolitain/solver-lnk/internal/solver/castle"
)

const dataDir = "../../../data"

func TestSolverBasic(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

targetLevels := map[models.BuildingType]int{
models.Lumberjack: 10,
models.Quarry:     10,
models.OreMine:    10,
}

initialState := models.NewGameState()
initialState.Resources[models.Wood] = 120
initialState.Resources[models.Stone] = 120
initialState.Resources[models.Iron] = 120
initialState.Resources[models.Food] = 40

for _, bt := range models.AllBuildingTypes() {
initialState.BuildingLevels[bt] = 1
}

solver := castle.NewSolver(buildings, technologies, nil, targetLevels)
solution := solver.Solve(initialState)

if solution == nil {
t.Fatal("Solution should not be nil")
}

if len(solution.BuildingActions) == 0 {
t.Error("Should have building actions")
}

// Verify targets reached
for bt, target := range targetLevels {
if solution.FinalState.BuildingLevels[bt] < target {
t.Errorf("%s should reach level %d, got %d", bt, target, solution.FinalState.BuildingLevels[bt])
}
}

t.Logf("Completed in %.2f days with %d building actions",
float64(solution.TotalTimeSeconds)/86400.0, len(solution.BuildingActions))
}

func TestFullBuildComparison(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

// Full castle targets
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

initialState := models.NewGameState()
initialState.Resources[models.Wood] = 120
initialState.Resources[models.Stone] = 120
initialState.Resources[models.Iron] = 120
initialState.Resources[models.Food] = 40

for _, bt := range models.AllBuildingTypes() {
initialState.BuildingLevels[bt] = 1
}

solver := castle.NewSolver(buildings, technologies, nil, targetLevels)
solution := solver.Solve(initialState)

if solution == nil {
t.Fatal("Solution should not be nil")
}

days := float64(solution.TotalTimeSeconds) / 86400.0
t.Logf("Completion time: %.2f days (%.0f hours)", days, days*24)
t.Logf("Building actions: %d", len(solution.BuildingActions))
t.Logf("Research actions: %d", len(solution.ResearchActions))

// Verify ALL target buildings reached their target levels
for bt, target := range targetLevels {
final := solution.FinalState.BuildingLevels[bt]
if final < target {
t.Errorf("%s: target=%d, final=%d - NOT REACHED", bt, target, final)
} else {
t.Logf("%s: target=%d, final=%d ✓", bt, target, final)
}
}

// Verify ALL loaded technologies are researched
for techName := range technologies {
if !solution.FinalState.ResearchedTechnologies[techName] {
t.Errorf("Technology %s should be researched", techName)
} else {
t.Logf("Technology %s ✓", techName)
}
}
t.Logf("Total technologies researched: %d/%d", len(solution.FinalState.ResearchedTechnologies), len(technologies))

// Should complete in roughly 40-100 days (includes defense unit training)
if days < 40 || days > 100 {
t.Errorf("Completion time %.2f days is outside expected range [40, 100]", days)
}
}

func TestDeterminism(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

targetLevels := map[models.BuildingType]int{
models.Lumberjack: 15,
models.Quarry:     15,
models.OreMine:    15,
}

createInitialState := func() *models.GameState {
s := models.NewGameState()
s.Resources[models.Wood] = 120
s.Resources[models.Stone] = 120
s.Resources[models.Iron] = 120
s.Resources[models.Food] = 40
for _, bt := range models.AllBuildingTypes() {
s.BuildingLevels[bt] = 1
}
return s
}

solver := castle.NewSolver(buildings, technologies, nil, targetLevels)

// Run multiple times
var firstTime int
var firstActionCount int

for i := 0; i < 10; i++ {
solution := solver.Solve(createInitialState())

if i == 0 {
firstTime = solution.TotalTimeSeconds
firstActionCount = len(solution.BuildingActions)
} else {
if solution.TotalTimeSeconds != firstTime {
t.Errorf("Run %d: time %d != first run %d", i, solution.TotalTimeSeconds, firstTime)
}
if len(solution.BuildingActions) != firstActionCount {
t.Errorf("Run %d: action count %d != first run %d", i, len(solution.BuildingActions), firstActionCount)
}
}
}

t.Logf("Determinism verified across 10 runs (time=%d, actions=%d)", firstTime, firstActionCount)
}

func TestInvariants(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

targetLevels := map[models.BuildingType]int{
models.Lumberjack: 20,
models.Quarry:     20,
models.OreMine:    20,
models.Farm:       15,
}

initialState := models.NewGameState()
initialState.Resources[models.Wood] = 120
initialState.Resources[models.Stone] = 120
initialState.Resources[models.Iron] = 120
initialState.Resources[models.Food] = 40

for _, bt := range models.AllBuildingTypes() {
initialState.BuildingLevels[bt] = 1
}

solver := castle.NewSolver(buildings, technologies, nil, targetLevels)
solution := solver.Solve(initialState)

// Invariant 1: Times are non-negative and ordered
for i, action := range solution.BuildingActions {
if action.StartTime < 0 {
t.Errorf("Action %d has negative start time: %d", i, action.StartTime)
}
if action.EndTime < action.StartTime {
t.Errorf("Action %d end time %d < start time %d", i, action.EndTime, action.StartTime)
}
}

// Invariant 2: Levels increase by 1
for i, action := range solution.BuildingActions {
if action.ToLevel != action.FromLevel+1 {
t.Errorf("Action %d: level change %d->%d (should be +1)", i, action.FromLevel, action.ToLevel)
}
}

// Invariant 3: Food used <= food capacity
for i, action := range solution.BuildingActions {
if action.FoodUsed > action.FoodCapacity {
t.Errorf("Action %d: food used %d > capacity %d", i, action.FoodUsed, action.FoodCapacity)
}
}

// Invariant 4: Building queue is serial (no overlapping building actions)
for i := 1; i < len(solution.BuildingActions); i++ {
prev := solution.BuildingActions[i-1]
curr := solution.BuildingActions[i]
if curr.StartTime < prev.EndTime {
t.Errorf("Building actions overlap: action %d ends at %d, action %d starts at %d",
i-1, prev.EndTime, i, curr.StartTime)
}
}
}

func TestGameRulesValidation(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

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

initialState := models.NewGameState()
initialState.Resources[models.Wood] = 120
initialState.Resources[models.Stone] = 120
initialState.Resources[models.Iron] = 120
initialState.Resources[models.Food] = 40

for _, bt := range models.AllBuildingTypes() {
initialState.BuildingLevels[bt] = 1
}

solver := castle.NewSolver(buildings, technologies, nil, targetLevels)
solution := solver.Solve(initialState)

// === REPLAY SIMULATION ===
type event struct {
time       int
isStart    bool
isBuilding bool
buildIdx   int
resIdx     int
}

var events []event

for i, action := range solution.BuildingActions {
events = append(events, event{time: action.StartTime, isStart: true, isBuilding: true, buildIdx: i})
events = append(events, event{time: action.EndTime, isStart: false, isBuilding: true, buildIdx: i})
}

for i, action := range solution.ResearchActions {
events = append(events, event{time: action.StartTime, isStart: true, isBuilding: false, resIdx: i})
events = append(events, event{time: action.EndTime, isStart: false, isBuilding: false, resIdx: i})
}

sort.Slice(events, func(i, j int) bool {
if events[i].time != events[j].time {
return events[i].time < events[j].time
}
if events[i].isStart != events[j].isStart {
return !events[i].isStart
}
return false
})

simTime := 0
simResources := map[models.ResourceType]float64{
models.Wood:  120,
models.Stone: 120,
models.Iron:  120,
models.Food:  40,
}
simBuildingLevels := make(map[models.BuildingType]int)
for _, bt := range models.AllBuildingTypes() {
simBuildingLevels[bt] = 1
}
simResearchedTechs := make(map[string]bool)
simFoodUsed := 0
simBuildingQueueFreeAt := 0
simResearchQueueFreeAt := 0

getProductionRate := func(bt models.BuildingType, level int) float64 {
building := buildings[bt]
if building == nil {
return 0
}
levelData := building.GetLevelData(level)
if levelData == nil || levelData.ProductionRate == nil {
return 0
}
return *levelData.ProductionRate
}

getStorageCap := func(bt models.BuildingType, level int) int {
building := buildings[bt]
if building == nil {
return 0
}
levelData := building.GetLevelData(level)
if levelData == nil || levelData.StorageCapacity == nil {
return 0
}
return *levelData.StorageCapacity
}

simProductionRates := map[models.ResourceType]float64{
models.Wood:  getProductionRate(models.Lumberjack, 1),
models.Stone: getProductionRate(models.Quarry, 1),
models.Iron:  getProductionRate(models.OreMine, 1),
}

simStorageCaps := map[models.ResourceType]int{
models.Wood:  getStorageCap(models.WoodStore, 1),
models.Stone: getStorageCap(models.StoneStore, 1),
models.Iron:  getStorageCap(models.OreStore, 1),
}

simFoodCapacity := getStorageCap(models.Farm, 1)
simProductionBonus := 1.0

advanceSimTime := func(toTime int) {
if toTime <= simTime {
return
}
deltaSeconds := toTime - simTime
deltaHours := float64(deltaSeconds) / 3600.0

for _, rt := range []models.ResourceType{models.Wood, models.Stone, models.Iron} {
rate := simProductionRates[rt]
produced := rate * deltaHours * simProductionBonus
simResources[rt] += produced

cap := simStorageCaps[rt]
if cap > 0 && simResources[rt] > float64(cap) {
simResources[rt] = float64(cap)
}
}
simTime = toTime
}

for _, ev := range events {
advanceSimTime(ev.time)

if ev.isBuilding {
action := solution.BuildingActions[ev.buildIdx]
building := buildings[action.BuildingType]

if ev.isStart {
levelData := building.GetLevelData(action.ToLevel)

if ev.time < simBuildingQueueFreeAt {
t.Errorf("Building %d (%s %d->%d): starts at %d but queue busy until %d",
ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
ev.time, simBuildingQueueFreeAt)
}

if action.FromLevel != simBuildingLevels[action.BuildingType] {
t.Errorf("Building %d (%s): FromLevel=%d but current level is %d",
ev.buildIdx, action.BuildingType, action.FromLevel, simBuildingLevels[action.BuildingType])
}

costs := levelData.Costs
checkResource := func(rt models.ResourceType, cost int) {
if cost > 0 && simResources[rt] < float64(cost)-0.01 {
t.Errorf("Building %d (%s %d->%d): needs %d %s but only have %.2f",
ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
cost, rt, simResources[rt])
}
}
checkResource(models.Wood, costs.Wood)
checkResource(models.Stone, costs.Stone)
checkResource(models.Iron, costs.Iron)

checkStorage := func(rt models.ResourceType, cost int) {
cap := simStorageCaps[rt]
if cost > cap {
t.Errorf("Building %d (%s %d->%d): cost %d %s exceeds storage cap %d",
ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
cost, rt, cap)
}
}
checkStorage(models.Wood, costs.Wood)
checkStorage(models.Stone, costs.Stone)
checkStorage(models.Iron, costs.Iron)

foodCost := costs.Food
if simFoodUsed+foodCost > simFoodCapacity {
t.Errorf("Building %d (%s %d->%d): needs %d food workers, but %d/%d already used",
ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel,
foodCost, simFoodUsed, simFoodCapacity)
}

if techName, ok := building.TechnologyPrerequisites[action.ToLevel]; ok {
if !simResearchedTechs[techName] {
t.Errorf("Building %d (%s %d->%d): requires tech '%s' which is not researched at start time %d",
ev.buildIdx, action.BuildingType, action.FromLevel, action.ToLevel, techName, ev.time)
}
}

if costs.Wood > 0 {
simResources[models.Wood] -= float64(costs.Wood)
}
if costs.Stone > 0 {
simResources[models.Stone] -= float64(costs.Stone)
}
if costs.Iron > 0 {
simResources[models.Iron] -= float64(costs.Iron)
}
simFoodUsed += foodCost

} else {
simBuildingLevels[action.BuildingType] = action.ToLevel
simBuildingQueueFreeAt = ev.time

switch action.BuildingType {
case models.Lumberjack:
simProductionRates[models.Wood] = getProductionRate(models.Lumberjack, action.ToLevel)
case models.Quarry:
simProductionRates[models.Stone] = getProductionRate(models.Quarry, action.ToLevel)
case models.OreMine:
simProductionRates[models.Iron] = getProductionRate(models.OreMine, action.ToLevel)
}

switch action.BuildingType {
case models.WoodStore:
simStorageCaps[models.Wood] = getStorageCap(models.WoodStore, action.ToLevel)
case models.StoneStore:
simStorageCaps[models.Stone] = getStorageCap(models.StoneStore, action.ToLevel)
case models.OreStore:
simStorageCaps[models.Iron] = getStorageCap(models.OreStore, action.ToLevel)
case models.Farm:
simFoodCapacity = getStorageCap(models.Farm, action.ToLevel)
}
}
} else {
action := solution.ResearchActions[ev.resIdx]
tech := technologies[action.TechnologyName]

if ev.isStart {
if ev.time < simResearchQueueFreeAt {
t.Errorf("Research %d (%s): starts at %d but queue busy until %d",
ev.resIdx, action.TechnologyName, ev.time, simResearchQueueFreeAt)
}

if tech != nil {
libraryLevel := simBuildingLevels[models.Library]
if libraryLevel < tech.RequiredLibraryLevel {
t.Errorf("Research %d (%s): requires Library %d but have %d",
ev.resIdx, action.TechnologyName, tech.RequiredLibraryLevel, libraryLevel)
}
}

if simResearchedTechs[action.TechnologyName] {
t.Errorf("Research %d (%s): already researched", ev.resIdx, action.TechnologyName)
}

if tech != nil {
costs := tech.Costs
checkRes := func(rt models.ResourceType, cost int) {
if cost > 0 && simResources[rt] < float64(cost)-0.01 {
t.Errorf("Research %d (%s): needs %d %s but only have %.2f",
ev.resIdx, action.TechnologyName, cost, rt, simResources[rt])
}
}
checkRes(models.Wood, costs.Wood)
checkRes(models.Stone, costs.Stone)
checkRes(models.Iron, costs.Iron)

if costs.Wood > 0 {
simResources[models.Wood] -= float64(costs.Wood)
}
if costs.Stone > 0 {
simResources[models.Stone] -= float64(costs.Stone)
}
if costs.Iron > 0 {
simResources[models.Iron] -= float64(costs.Iron)
}
}

} else {
simResearchedTechs[action.TechnologyName] = true
simResearchQueueFreeAt = ev.time

if action.TechnologyName == "Beer tester" || action.TechnologyName == "Wheelbarrow" {
simProductionBonus += 0.05
}
}
}
}

if simResources[models.Wood] < -0.01 {
t.Errorf("Final wood is negative: %.2f", simResources[models.Wood])
}
if simResources[models.Stone] < -0.01 {
t.Errorf("Final stone is negative: %.2f", simResources[models.Stone])
}
if simResources[models.Iron] < -0.01 {
t.Errorf("Final iron is negative: %.2f", simResources[models.Iron])
}

t.Logf("Game rules validation completed!")
t.Logf("Final state: %d buildings upgraded, %d techs researched",
len(solution.BuildingActions), len(solution.ResearchActions))
}

func TestMissionsIntegration(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

missions := loader.LoadMissions()
if len(missions) == 0 {
t.Fatal("Should have missions loaded")
}

// Test with small targets to verify missions work during building
targetLevels := map[models.BuildingType]int{
models.Lumberjack: 5,
models.Quarry:     5,
models.OreMine:    5,
models.Farm:       5,
models.Arsenal:    5,
models.Tavern:     3,
}

initialState := models.NewGameState()
initialState.Resources[models.Wood] = 500
initialState.Resources[models.Stone] = 500
initialState.Resources[models.Iron] = 500
initialState.Resources[models.Food] = 40

for _, bt := range models.AllBuildingTypes() {
initialState.BuildingLevels[bt] = 1
}

solver := castle.NewSolver(buildings, technologies, missions, targetLevels)
solution := solver.Solve(initialState)

if solution == nil {
t.Fatal("Solution should not be nil")
}

// Verify all targets reached
for bt, target := range targetLevels {
if solution.FinalState.BuildingLevels[bt] < target {
t.Errorf("%s: expected level %d, got %d", bt, target, solution.FinalState.BuildingLevels[bt])
}
}

// With Tavern 3, we should be able to run some missions
// The food headroom check should allow training during building
t.Logf("Buildings: %d, Research: %d", len(solution.BuildingActions), len(solution.ResearchActions))
t.Logf("Completion time: %.2f days", float64(solution.TotalTimeSeconds)/3600/24)
}

func TestMissionNoIdleTime(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

missions := loader.LoadMissions()

// Full build to get enough training and missions
targetLevels := map[models.BuildingType]int{
models.Lumberjack: 15,
models.Quarry:     15,
models.OreMine:    15,
models.Farm:       15,
models.Arsenal:    10,
models.Tavern:     5,
models.WoodStore:  10,
models.StoneStore: 10,
models.OreStore:   10,
}

initialState := models.NewGameState()
initialState.Resources[models.Wood] = 500
initialState.Resources[models.Stone] = 500
initialState.Resources[models.Iron] = 500
initialState.Resources[models.Food] = 40

for _, bt := range models.AllBuildingTypes() {
initialState.BuildingLevels[bt] = 1
}

solver := castle.NewSolver(buildings, technologies, missions, targetLevels)

// Use SolveWithMissionTracking to get mission events
solution, missionEvents := solver.SolveWithMissionTracking(initialState)

if solution == nil {
t.Fatal("Solution should not be nil")
}

t.Logf("Mission events: %d", len(missionEvents))

if len(missionEvents) < 2 {
t.Logf("Not enough mission events to check idle time (got %d)", len(missionEvents))
return
}

// Sort events by start time
sort.Slice(missionEvents, func(i, j int) bool {
return missionEvents[i].StartTime < missionEvents[j].StartTime
})

// Check that after the first mission, there's always at least one mission running
// This means we need to find any gaps where NO missions are running
// Build timeline of mission coverage

// Find the time range
firstStart := missionEvents[0].StartTime
lastEnd := missionEvents[0].EndTime
for _, me := range missionEvents {
if me.EndTime > lastEnd {
lastEnd = me.EndTime
}
}

// For efficiency, we'll check key points (start/end times of missions)
// At each point, count running missions
type timePoint struct {
time  float64
delta int // +1 for start, -1 for end
}

var points []timePoint
for _, me := range missionEvents {
points = append(points, timePoint{me.StartTime, 1})
points = append(points, timePoint{me.EndTime, -1})
}

sort.Slice(points, func(i, j int) bool {
if points[i].time != points[j].time {
return points[i].time < points[j].time
}
// End before start at same time (to detect momentary gaps)
return points[i].delta < points[j].delta
})

running := 0
gapTime := 0.0
lastTime := firstStart

for _, p := range points {
if running == 0 && p.time > lastTime {
gap := p.time - lastTime
if gap > 1 { // More than 1 second gap
gapTime += gap
}
}
running += p.delta
lastTime = p.time
}

t.Logf("Mission coverage: first=%.1f, last=%.1f, total_gap=%.1f seconds (%.2f hours)",
firstStart, lastEnd, gapTime, gapTime/3600)

// Allow some gap during initial training phase (first 10% of timeline)
totalDuration := lastEnd - firstStart
allowedGap := totalDuration * 0.1 // Allow 10% gap

if gapTime > allowedGap {
t.Errorf("Too much mission idle time: %.1f seconds (%.2f%% of total), allowed %.1f seconds",
gapTime, gapTime/totalDuration*100, allowedGap)
} else {
t.Logf("Mission idle time within acceptable range: %.1f seconds (%.2f%% of total)",
gapTime, gapTime/totalDuration*100)
}
}

// TestFoodCapacityFullyUsed verifies that at the end of the build order,
// all food capacity is used (5000/5000 for Farm level 30)
func TestFoodCapacityFullyUsed(t *testing.T) {
buildings, err := loader.LoadBuildings(dataDir)
if err != nil {
t.Fatalf("Failed to load buildings: %v", err)
}

technologies, err := loader.LoadTechnologies(dataDir)
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

missions := loader.LoadMissions()

// Full castle build to level 30
targetLevels := map[models.BuildingType]int{
models.Lumberjack:     30,
models.Quarry:         30,
models.OreMine:        30,
models.Farm:           30,
models.WoodStore:      20,
models.StoneStore:     20,
models.OreStore:       20,
models.Tavern:         10,
models.Fortifications: 20,
models.Arsenal:        30,
models.Keep:           10,
models.Market:         8,
models.Library:        10,
}

initialState := models.NewGameState()
initialState.Resources[models.Wood] = 120
initialState.Resources[models.Stone] = 120
initialState.Resources[models.Iron] = 120
initialState.Resources[models.Food] = 40

for _, bt := range models.AllBuildingTypes() {
initialState.BuildingLevels[bt] = 1
}

solver := castle.NewSolver(buildings, technologies, missions, targetLevels)
solution := solver.Solve(initialState)

if solution == nil {
t.Fatal("Solution should not be nil")
}

if len(solution.TrainingActions) == 0 {
t.Error("Should have training actions")
return
}

// Get the last training action's food used
lastTraining := solution.TrainingActions[len(solution.TrainingActions)-1]

// Farm level 30 gives 5000 food capacity
expectedFoodCapacity := 5000

if lastTraining.FoodCapacity != expectedFoodCapacity {
t.Errorf("Expected food capacity %d, got %d", expectedFoodCapacity, lastTraining.FoodCapacity)
}

if lastTraining.FoodUsed != expectedFoodCapacity {
t.Errorf("Food not fully used: %d / %d (expected %d / %d)",
lastTraining.FoodUsed, lastTraining.FoodCapacity,
expectedFoodCapacity, expectedFoodCapacity)
}

t.Logf("Food fully used: %d / %d", lastTraining.FoodUsed, lastTraining.FoodCapacity)
}
