package services

import (
	"encoding/json"
	"eventexplorer/models"
	"os"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

const sampleCitiesPath = "data/sample_cities.json"

var (
	execSampleOnce sync.Once
	sampleCities   []models.SampleCity
)

func loadSampleCities(path string) []models.SampleCity {
	raw, err := os.ReadFile(path)
	if err != nil {
		logs.Warn("sample cities unavailable (%s): %v", path, err)
		return []models.SampleCity{}
	}

	var set models.SampleCitySet
	if err := json.Unmarshal(raw, &set); err != nil {
		logs.Warn("sample cities malformed (%s): %v", path, err)
		return []models.SampleCity{}
	}

	cities := set.ValidCities()
	logs.Info("loaded %d sample cities", len(cities))
	return cities
}

func SampleCities() []models.SampleCity {
	execSampleOnce.Do(func() {
		sampleCities = loadSampleCities(sampleCitiesPath)
	})
	return sampleCities
}
