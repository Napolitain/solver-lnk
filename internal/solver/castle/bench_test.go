package castle

import (
"testing"

"github.com/napolitain/solver-lnk/internal/loader"
"github.com/napolitain/solver-lnk/internal/models"
)

func BenchmarkFullSolve(b *testing.B) {
buildings, _ := loader.LoadBuildings("../../../data")
techs, _ := loader.LoadTechnologies("../../../data")
missions, _ := loader.LoadMissions("../../../data")

targetLevels := map[models.BuildingType]int{
models.Lumberjack:     30,
models.Quarry:         30,
models.OreMine:        30,
models.Farm:           30,
models.WoodStore:      20,
models.StoneStore:     20,
models.OreStore:       20,
models.Keep:           20,
models.Arsenal:        30,
models.Library:        10,
models.Tavern:         10,
models.Market:         20,
models.Fortifications: 20,
}

solver := NewSolver(buildings, techs, missions, targetLevels)
initialState := &models.GameState{
BuildingLevels:         map[models.BuildingType]int{},
Resources:              map[models.ResourceType]float64{},
ResearchedTechnologies: map[string]bool{},
StorageCaps:            map[models.ResourceType]int{},
ProductionRates:        map[models.ResourceType]float64{},
}

b.ResetTimer()
for i := 0; i < b.N; i++ {
solver.Solve(initialState)
}
}
