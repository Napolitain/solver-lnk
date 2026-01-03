package loader

import (
"testing"
)

func TestHorseArmourLoaded(t *testing.T) {
techs, err := LoadTechnologies("../../data")
if err != nil {
t.Fatalf("Failed to load technologies: %v", err)
}

horseArmour := techs["Horse armour"]
if horseArmour == nil {
t.Fatal("Horse armour technology not found")
}

if horseArmour.RequiredLibraryLevel != 7 {
t.Errorf("Horse armour should require Library 7, got %d", horseArmour.RequiredLibraryLevel)
}

t.Logf("Horse armour: %+v", horseArmour)
}
