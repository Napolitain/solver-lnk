package loader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/napolitain/solver-lnk/internal/models"
)

// Precompiled regex for better performance
var timeFormatRegex = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)

// BuildingJSON represents the JSON structure for buildings
type BuildingJSON struct {
	BuildingType string                       `json:"building_type"`
	MaxLevel     int                          `json:"max_level"`
	Levels       map[string]BuildingLevelJSON `json:"levels"`
}

// BuildingLevelJSON represents the JSON structure for a building level
type BuildingLevelJSON struct {
	Costs            map[string]int `json:"costs"`
	BuildTimeSeconds int            `json:"build_time_seconds"`
	ProductionRate   *float64       `json:"production_rate,omitempty"`
	StorageCapacity  *float64       `json:"storage_capacity,omitempty"`
}

// LoadBuildings loads buildings from the JSON file
func LoadBuildings(dataDir string) (map[models.BuildingType]*models.Building, error) {
	filePath := filepath.Join(dataDir, "buildings.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read buildings.json: %w", err)
	}

	var rawBuildings map[string]BuildingJSON
	if err := json.Unmarshal(data, &rawBuildings); err != nil {
		return nil, fmt.Errorf("failed to parse buildings.json: %w", err)
	}

	buildings := make(map[models.BuildingType]*models.Building)

	for name, raw := range rawBuildings {
		bType := models.BuildingType(name)
		building := &models.Building{
			Type:                    bType,
			MaxLevel:                raw.MaxLevel,
			Levels:                  make(map[int]*models.BuildingLevel),
			Prerequisites:           make(map[int]map[models.BuildingType]int),
			TechnologyPrerequisites: make(map[int]string),
		}

		for levelStr, levelData := range raw.Levels {
			level, _ := strconv.Atoi(levelStr)
			
			// Build Costs struct from map
			var costs models.Costs
			for res, amount := range levelData.Costs {
				switch models.ResourceType(res) {
				case models.Wood:
					costs.Wood = amount
				case models.Stone:
					costs.Stone = amount
				case models.Iron:
					costs.Iron = amount
				case models.Food:
					costs.Food = amount
				}
			}

			bl := &models.BuildingLevel{
				Costs:            costs,
				BuildTimeSeconds: levelData.BuildTimeSeconds,
				ProductionRate:   levelData.ProductionRate,
			}

			if levelData.StorageCapacity != nil {
				cap := int(*levelData.StorageCapacity)
				bl.StorageCapacity = &cap
			}

			building.Levels[level] = bl
		}

		buildings[bType] = building
	}

	// Load technology prerequisites
	if err := loadTechPrerequisites(dataDir, buildings); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: could not load tech prerequisites: %v\n", err)
	}

	return buildings, nil
}

func loadTechPrerequisites(dataDir string, buildings map[models.BuildingType]*models.Building) error {
	filePath := filepath.Join(dataDir, "technology_prerequisites.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var prereqs map[string]map[string]struct {
		Library    int    `json:"library"`
		Technology string `json:"technology"`
	}

	if err := json.Unmarshal(data, &prereqs); err != nil {
		return err
	}

	for buildingName, levels := range prereqs {
		bType := models.BuildingType(buildingName)
		if building, ok := buildings[bType]; ok {
			for levelStr, req := range levels {
				level, _ := strconv.Atoi(levelStr)
				building.TechnologyPrerequisites[level] = req.Technology
			}
		}
	}

	return nil
}

// LoadTechnologies loads technologies from the techs directory
func LoadTechnologies(dataDir string) (map[string]*models.Technology, error) {
	techsDir := filepath.Join(dataDir, "techs")
	entries, err := os.ReadDir(techsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read techs directory: %w", err)
	}

	technologies := make(map[string]*models.Technology)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(techsDir, entry.Name())
		tech, err := parseTechFile(filePath, entry.Name())
		if err != nil {
			fmt.Printf("Warning: failed to parse tech file %s: %v\n", entry.Name(), err)
			continue
		}

		technologies[tech.Name] = tech
	}

	// Load required library levels from technologies.json
	if err := loadTechLibraryLevels(dataDir, technologies); err != nil {
		fmt.Printf("Warning: could not load tech library levels: %v\n", err)
	}

	return technologies, nil
}

// loadTechLibraryLevels reads required library levels from technologies.json
func loadTechLibraryLevels(dataDir string, technologies map[string]*models.Technology) error {
	filePath := filepath.Join(dataDir, "technologies.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var techJSON map[string]struct {
		Name                 string `json:"name"`
		RequiredLibraryLevel int    `json:"required_library_level"`
	}

	if err := json.Unmarshal(data, &techJSON); err != nil {
		return err
	}

	for name, info := range techJSON {
		if tech, ok := technologies[name]; ok {
			tech.RequiredLibraryLevel = info.RequiredLibraryLevel
		}
	}

	return nil
}

func parseTechFile(filePath, internalName string) (*models.Technology, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tech := &models.Technology{
		InternalName:         internalName,
		RequiredLibraryLevel: 1,
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	costLines := []int{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++

		if lineNum == 1 {
			tech.Name = line
			continue
		}

		// Check for time format HH:MM:SS
		if timeFormatRegex.MatchString(line) {
			parts := strings.Split(line, ":")
			hours, _ := strconv.Atoi(parts[0])
			minutes, _ := strconv.Atoi(parts[1])
			seconds, _ := strconv.Atoi(parts[2])
			tech.ResearchTimeSeconds = hours*3600 + minutes*60 + seconds
			continue
		}

		// Check for "Farm Level X" pattern
		if strings.HasPrefix(line, "Farm Level") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				level, _ := strconv.Atoi(parts[2])
				tech.EnablesBuilding = "farm"
				tech.EnablesLevel = level
			}
			continue
		}

		// Try to parse as number (cost)
		if num, err := strconv.Atoi(line); err == nil {
			costLines = append(costLines, num)
		}
	}

	// Assign costs based on position (wood, stone, iron, food)
	if len(costLines) >= 1 {
		tech.Costs.Wood = costLines[0]
	}
	if len(costLines) >= 2 {
		tech.Costs.Stone = costLines[1]
	}
	if len(costLines) >= 3 {
		tech.Costs.Iron = costLines[2]
	}
	if len(costLines) >= 4 {
		tech.Costs.Food = costLines[3]
	}

	return tech, nil
}
